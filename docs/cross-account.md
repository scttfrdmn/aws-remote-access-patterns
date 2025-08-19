# Cross-Account AWS Integration

Complete implementation guide for SaaS services that need secure access to customer AWS accounts.

## Overview

This pattern is for **services running in AWS** that need access to **customer AWS accounts**. Think Coiled, Datadog, New Relic, or any SaaS tool that manages AWS resources on behalf of customers.

### When to Use This Pattern

- ✅ Your service runs in AWS (has its own AWS account)
- ✅ Customers have separate AWS accounts  
- ✅ You need to create/manage resources in customer accounts
- ✅ You want to avoid customers sharing long-lived access keys
- ✅ You need programmatic access (API calls, not human users)

### Architecture Overview

```
┌─────────────────────┐    ┌─────────────────────┐
│   Your AWS Account  │    │ Customer AWS Account│
│                     │    │                     │
│  ┌───────────────┐  │    │  ┌───────────────┐  │
│  │ Your Service  │  │    │  │ IAM Role      │  │
│  │ (EC2/Lambda)  │◄─┼────┼─►│ (Trust Policy)│  │
│  └───────────────┘  │    │  └───────────────┘  │
│                     │    │                     │
│  Account ID:        │    │  External ID:       │
│  123456789012       │    │  unique-customer-id │
└─────────────────────┘    └─────────────────────┘
```

## Complete Implementation

### Project Structure

```
aws-cross-account-integration/
├── README.md
├── go.mod
├── cmd/
│   └── example/
│       └── main.go
├── pkg/
│   └── crossaccount/
│       ├── client.go           # Main client
│       ├── config.go           # Configuration
│       ├── templates.go        # CloudFormation generation
│       ├── validation.go       # Role validation
│       ├── storage.go          # Credential storage
│       └── handlers.go         # HTTP handlers
├── internal/
│   ├── crypto/
│   │   └── encrypt.go          # Credential encryption
│   └── templates/
│       ├── cloudformation.yaml # CF template
│       └── setup-ui.html       # Setup UI
├── web/
│   ├── static/
│   │   ├── css/
│   │   └── js/
│   └── templates/
│       └── setup.html
└── examples/
    ├── gin-service/
    ├── lambda-function/
    └── kubernetes-controller/
```

### Core Implementation

#### pkg/crossaccount/client.go

```go
package crossaccount

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/sts"
)

type Client struct {
    config  *Config
    storage CredentialStorage
    crypto  Encryptor
}

func New(cfg *Config, opts ...Option) (*Client, error) {
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    c := &Client{
        config:  cfg,
        storage: NewMemoryStorage(),
        crypto:  NewAESEncryptor(),
    }

    for _, opt := range opts {
        opt(c)
    }

    return c, nil
}

type Option func(*Client)

func WithStorage(s CredentialStorage) Option {
    return func(c *Client) { c.storage = s }
}

func WithEncryption(e Encryptor) Option {
    return func(c *Client) { c.crypto = e }
}

// GenerateIntegrationURL creates a CloudFormation launch URL
func (c *Client) GenerateIntegrationURL(ctx context.Context, req *IntegrationRequest) (*IntegrationResponse, error) {
    externalID := c.generateExternalID(req.CustomerID)
    
    // Upload CloudFormation template to S3
    templateURL, err := c.uploadTemplate(ctx, req.CustomerID)
    if err != nil {
        return nil, fmt.Errorf("failed to upload template: %w", err)
    }

    // Build CloudFormation launch URL
    params := map[string]string{
        "ExternalId":       externalID,
        "ServiceAccountId": c.config.ServiceAccountID,
        "RoleName":        fmt.Sprintf("%s-CrossAccount-%s", c.config.ServiceName, req.CustomerID),
        "SetupPhase":      "true",
    }

    launchURL := c.buildLaunchURL(templateURL, params, req.Region)

    return &IntegrationResponse{
        LaunchURL:  launchURL,
        ExternalID: externalID,
        StackName:  params["RoleName"],
    }, nil
}

// AssumeCustomerRole assumes the customer's cross-account role
func (c *Client) AssumeCustomerRole(ctx context.Context, customerID string) (aws.Config, error) {
    creds, err := c.getCustomerCredentials(ctx, customerID)
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to get credentials: %w", err)
    }

    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
    }

    stsClient := sts.NewFromConfig(cfg)

    result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
        RoleArn:         aws.String(creds.RoleARN),
        RoleSessionName: aws.String(fmt.Sprintf("%s-%s", c.config.ServiceName, customerID)),
        ExternalId:      aws.String(creds.ExternalID),
        DurationSeconds: aws.Int32(int32(c.config.SessionDuration.Seconds())),
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to assume role: %w", err)
    }

    // Create config with temporary credentials
    return config.LoadDefaultConfig(ctx,
        config.WithCredentialsProvider(aws.NewCredentialsCache(
            &staticCredentialsProvider{
                accessKey:    *result.Credentials.AccessKeyId,
                secretKey:    *result.Credentials.SecretAccessKey,
                sessionToken: *result.Credentials.SessionToken,
            },
        )),
    )
}

// VerifyIntegration tests the cross-account role
func (c *Client) VerifyIntegration(ctx context.Context, req *VerificationRequest) error {
    tempCreds := CustomerCredentials{
        RoleARN:    req.RoleARN,
        ExternalID: req.ExternalID,
    }

    if err := c.storage.Store(ctx, fmt.Sprintf("temp-%s", req.CustomerID), tempCreds); err != nil {
        return fmt.Errorf("failed to store temp credentials: %w", err)
    }

    defer c.storage.Delete(ctx, fmt.Sprintf("temp-%s", req.CustomerID))

    // Try to assume the role
    cfg, err := c.AssumeCustomerRole(ctx, fmt.Sprintf("temp-%s", req.CustomerID))
    if err != nil {
        return fmt.Errorf("role assumption failed: %w", err)
    }

    // Test basic permissions
    return c.validatePermissions(ctx, cfg)
}

// StoreCustomerCredentials securely stores customer role information
func (c *Client) StoreCustomerCredentials(ctx context.Context, req *StoreCredentialsRequest) error {
    creds := CustomerCredentials{
        RoleARN:    req.RoleARN,
        ExternalID: req.ExternalID,
        CreatedAt:  time.Now(),
    }

    return c.storage.Store(ctx, req.CustomerID, creds)
}

func (c *Client) generateExternalID(customerID string) string {
    // Generate cryptographically secure external ID
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        // Fallback to timestamp-based ID
        return fmt.Sprintf("%s-%s-%d", c.config.ServiceName, customerID, time.Now().Unix())
    }
    return fmt.Sprintf("%s-%s-%s", c.config.ServiceName, customerID, hex.EncodeToString(bytes))
}

func (c *Client) validatePermissions(ctx context.Context, cfg aws.Config) error {
    // Implement permission validation based on your service requirements
    // Example: Try to list EC2 instances
    // ec2Client := ec2.NewFromConfig(cfg)
    // _, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{MaxResults: aws.Int32(5)})
    // return err
    return nil
}

type staticCredentialsProvider struct {
    accessKey, secretKey, sessionToken string
}

func (s *staticCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
    return aws.Credentials{
        AccessKeyID:     s.accessKey,
        SecretAccessKey: s.secretKey,
        SessionToken:    s.sessionToken,
    }, nil
}
```

#### pkg/crossaccount/config.go

```go
package crossaccount

import (
    "errors"
    "time"
    "github.com/aws/aws-sdk-go-v2/aws"
)

type Config struct {
    // Service identification
    ServiceName      string `json:"service_name"`
    ServiceAccountID string `json:"service_account_id"`
    
    // AWS configuration
    AWSConfig        aws.Config     `json:"-"`
    DefaultRegion    string         `json:"default_region"`
    TemplateS3Bucket string         `json:"template_s3_bucket"`
    
    // Security settings
    SessionDuration    time.Duration `json:"session_duration"`
    ExternalIDPrefix   string        `json:"external_id_prefix"`
    
    // Permissions
    OngoingPermissions []Permission `json:"ongoing_permissions"`
    SetupPermissions   []Permission `json:"setup_permissions"`
    
    // UI customization
    BrandingOptions map[string]string `json:"branding_options"`
}

type Permission struct {
    Sid       string                 `json:"sid"`
    Effect    string                 `json:"effect"`
    Actions   []string              `json:"actions"`
    Resources []string              `json:"resources"`
    Condition map[string]interface{} `json:"condition,omitempty"`
}

func (c *Config) Validate() error {
    if c.ServiceName == "" {
        return errors.New("service_name is required")
    }
    if c.ServiceAccountID == "" {
        return errors.New("service_account_id is required")
    }
    if c.TemplateS3Bucket == "" {
        return errors.New("template_s3_bucket is required")
    }
    
    // Set defaults
    if c.DefaultRegion == "" {
        c.DefaultRegion = "us-east-1"
    }
    if c.SessionDuration == 0 {
        c.SessionDuration = time.Hour
    }
    
    return nil
}

// Request/Response types
type IntegrationRequest struct {
    CustomerID   string `json:"customer_id"`
    CustomerName string `json:"customer_name"`
    Region       string `json:"region"`
}

type IntegrationResponse struct {
    LaunchURL  string `json:"launch_url"`
    ExternalID string `json:"external_id"`
    StackName  string `json:"stack_name"`
}

type VerificationRequest struct {
    CustomerID string `json:"customer_id"`
    RoleARN    string `json:"role_arn"`
    ExternalID string `json:"external_id"`
}

type StoreCredentialsRequest struct {
    CustomerID string `json:"customer_id"`
    RoleARN    string `json:"role_arn"`
    ExternalID string `json:"external_id"`
}

type CustomerCredentials struct {
    RoleARN    string    `json:"role_arn"`
    ExternalID string    `json:"external_id"`
    CreatedAt  time.Time `json:"created_at"`
}
```

#### pkg/crossaccount/templates.go

```go
package crossaccount

import (
    "bytes"
    "fmt"
    "text/template"
)

const cloudFormationTemplate = `
AWSTemplateFormatVersion: '2010-09-09'
Description: 'Cross-account IAM role for {{.ServiceName}}'

Parameters:
  ExternalId:
    Type: String
    Description: 'Unique identifier for additional security'
    MinLength: 8
    MaxLength: 64
    
  ServiceAccountId:
    Type: String
    Description: 'AWS Account ID for {{.ServiceName}}'
    Default: '{{.ServiceAccountID}}'
    AllowedPattern: '[0-9]{12}'
    
  RoleName:
    Type: String
    Description: 'Name for the IAM role'
    Default: '{{.ServiceName}}-CrossAccountRole'
    
  SetupPhase:
    Type: String
    Description: 'Include setup permissions?'
    Default: 'true'
    AllowedValues: ['true', 'false']

Conditions:
  IncludeSetupPermissions: !Equals [!Ref SetupPhase, 'true']

Resources:
  CrossAccountRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Ref RoleName
      Path: '/{{.ServiceName}}/'
      MaxSessionDuration: {{.SessionDurationSeconds}}
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              AWS: !Sub 'arn:aws:iam::${ServiceAccountId}:root'
            Action: 'sts:AssumeRole'
            Condition:
              StringEquals:
                'sts:ExternalId': !Ref ExternalId
      ManagedPolicyArns:
        - !Ref OngoingPolicy
        - !If [IncludeSetupPermissions, !Ref SetupPolicy, !Ref 'AWS::NoValue']

  OngoingPolicy:
    Type: AWS::IAM::ManagedPolicy
    Properties:
      ManagedPolicyName: !Sub '${RoleName}-Ongoing'
      Path: '/{{.ServiceName}}/'
      PolicyDocument:
        Version: '2012-10-17'
        Statement:
{{range .OngoingPermissions}}          - Sid: '{{.Sid}}'
            Effect: {{.Effect}}
            Action:
{{range .Actions}}              - '{{.}}'
{{end}}            Resource:
{{range .Resources}}              - '{{.}}'
{{end}}{{if .Condition}}
            Condition:
{{range $key, $value := .Condition}}              {{$key}}: {{$value}}
{{end}}{{end}}
{{end}}

  SetupPolicy:
    Type: AWS::IAM::ManagedPolicy
    Condition: IncludeSetupPermissions
    Properties:
      ManagedPolicyName: !Sub '${RoleName}-Setup'
      Path: '/{{.ServiceName}}/'
      PolicyDocument:
        Version: '2012-10-17'
        Statement:
{{range .SetupPermissions}}          - Sid: '{{.Sid}}'
            Effect: {{.Effect}}
            Action:
{{range .Actions}}              - '{{.}}'
{{end}}            Resource:
{{range .Resources}}              - '{{.}}'
{{end}}{{if .Condition}}
            Condition:
{{range $key, $value := .Condition}}              {{$key}}: {{$value}}
{{end}}{{end}}
{{end}}

Outputs:
  RoleArn:
    Description: 'ARN of the cross-account role'
    Value: !GetAtt CrossAccountRole.Arn
    
  ExternalId:
    Description: 'External ID for additional security'
    Value: !Ref ExternalId
    
  NextSteps:
    Description: 'What to do next'
    Value: !If
      - IncludeSetupPermissions
      - 'Setup complete! Remember to update stack with SetupPhase=false after initial setup.'
      - 'Role configured for ongoing operations.'
`

func (c *Client) GenerateCloudFormationTemplate() (string, error) {
    tmpl, err := template.New("cloudformation").Parse(cloudFormationTemplate)
    if err != nil {
        return "", fmt.Errorf("failed to parse template: %w", err)
    }

    data := struct {
        ServiceName             string
        ServiceAccountID        string
        SessionDurationSeconds  int
        OngoingPermissions      []Permission
        SetupPermissions        []Permission
    }{
        ServiceName:             c.config.ServiceName,
        ServiceAccountID:        c.config.ServiceAccountID,
        SessionDurationSeconds:  int(c.config.SessionDuration.Seconds()),
        OngoingPermissions:      c.config.OngoingPermissions,
        SetupPermissions:        c.config.SetupPermissions,
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute template: %w", err)
    }

    return buf.String(), nil
}
```

### Usage Examples

#### Simple SaaS Service

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/your-org/aws-cross-account-integration/pkg/crossaccount"
)

func main() {
    // Configure your service
    config := &crossaccount.Config{
        ServiceName:      "MyDataPlatform",
        ServiceAccountID: "123456789012",
        TemplateS3Bucket: "my-platform-templates",
        DefaultRegion:    "us-west-2",
        OngoingPermissions: []crossaccount.Permission{
            {
                Sid:    "S3DataAccess",
                Effect: "Allow", 
                Actions: []string{
                    "s3:GetObject",
                    "s3:PutObject",
                    "s3:ListBucket",
                },
                Resources: []string{"arn:aws:s3:::customer-data-*/*"},
            },
        },
        SetupPermissions: []crossaccount.Permission{
            {
                Sid:    "S3BucketSetup",
                Effect: "Allow",
                Actions: []string{
                    "s3:CreateBucket",
                    "s3:PutBucketPolicy",
                },
                Resources: []string{"*"},
            },
        },
    }

    client, err := crossaccount.New(config)
    if err != nil {
        log.Fatal(err)
    }

    r := gin.Default()

    // Integration endpoints
    r.GET("/integrate", func(c *gin.Context) {
        // Serve integration UI
        client.ServeSetupUI(c.Writer, c.Request)
    })

    r.POST("/api/integration/start", func(c *gin.Context) {
        var req crossaccount.IntegrationRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        resp, err := client.GenerateIntegrationURL(c.Request.Context(), &req)
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        c.JSON(200, resp)
    })

    r.POST("/api/integration/complete", func(c *gin.Context) {
        var req crossaccount.StoreCredentialsRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }

        // Verify the integration works
        verifyReq := &crossaccount.VerificationRequest{
            CustomerID: req.CustomerID,
            RoleARN:    req.RoleARN,
            ExternalID: req.ExternalID,
        }

        if err := client.VerifyIntegration(c.Request.Context(), verifyReq); err != nil {
            c.JSON(400, gin.H{"error": "Integration verification failed: " + err.Error()})
            return
        }

        // Store credentials
        if err := client.StoreCustomerCredentials(c.Request.Context(), &req); err != nil {
            c.JSON(500, gin.H{"error": "Failed to store credentials: " + err.Error()})
            return
        }

        c.JSON(200, gin.H{"status": "success"})
    })

    // Business logic endpoints
    r.GET("/api/customer/:id/data", func(c *gin.Context) {
        customerID := c.Param("id")

        // Get AWS session for this customer
        awsConfig, err := client.AssumeCustomerRole(c.Request.Context(), customerID)
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to access customer AWS: " + err.Error()})
            return
        }

        // Use AWS services with customer's permissions
        // s3Client := s3.NewFromConfig(awsConfig)
        // ... business logic

        c.JSON(200, gin.H{"message": "Data processed successfully"})
    })

    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}
```

#### Lambda Function Handler

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/your-org/aws-cross-account-integration/pkg/crossaccount"
)

type LambdaHandler struct {
    client *crossaccount.Client
}

func (h *LambdaHandler) HandleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    switch event.HTTPMethod {
    case "POST":
        return h.handleIntegrationSetup(ctx, event)
    case "GET":
        return h.handleCustomerOperation(ctx, event)
    default:
        return events.APIGatewayProxyResponse{StatusCode: 405}, nil
    }
}

func (h *LambdaHandler) handleIntegrationSetup(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    var req crossaccount.IntegrationRequest
    if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
        return events.APIGatewayProxyResponse{
            StatusCode: 400,
            Body:       fmt.Sprintf(`{"error": "%s"}`, err.Error()),
        }, nil
    }

    resp, err := h.client.GenerateIntegrationURL(ctx, &req)
    if err != nil {
        return events.APIGatewayProxyResponse{
            StatusCode: 500,
            Body:       fmt.Sprintf(`{"error": "%s"}`, err.Error()),
        }, nil
    }

    body, _ := json.Marshal(resp)
    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Headers:    map[string]string{"Content-Type": "application/json"},
        Body:       string(body),
    }, nil
}

func (h *LambdaHandler) handleCustomerOperation(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    customerID := event.PathParameters["customer_id"]

    // Assume customer role
    awsConfig, err := h.client.AssumeCustomerRole(ctx, customerID)
    if err != nil {
        return events.APIGatewayProxyResponse{
            StatusCode: 500,
            Body:       fmt.Sprintf(`{"error": "Failed to access customer AWS: %s"}`, err.Error()),
        }, nil
    }

    // Use AWS services with customer permissions
    // s3Client := s3.NewFromConfig(awsConfig)
    // ... perform operations

    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body:       `{"status": "success"}`,
    }, nil
}

func main() {
    config := &crossaccount.Config{
        ServiceName:      "MyLambdaService",
        ServiceAccountID: "123456789012",
        TemplateS3Bucket: "my-service-cf-templates",
        // ... other config
    }

    client, err := crossaccount.New(config)
    if err != nil {
        panic(err)
    }

    handler := &LambdaHandler{client: client}
    lambda.Start(handler.HandleRequest)
}
```

## Security Best Practices

### 1. External ID Management
- Generate cryptographically secure external IDs
- Store external IDs securely (encrypt at rest)
- Never log external IDs

### 2. Permission Boundaries
- Use least-privilege permissions
- Separate setup vs ongoing permissions
- Regular permission audits

### 3. Monitoring & Alerting
- Log all role assumptions
- Alert on unusual access patterns
- Monitor failed assumption attempts

### 4. Customer Communication
- Clearly document required permissions
- Provide guidance on removing setup permissions
- Offer role usage monitoring

## Deployment Considerations

### CloudFormation Template Storage
- Use versioned S3 buckets for templates
- Enable server-side encryption
- Implement proper bucket policies

### Credential Storage
- Encrypt credentials at rest
- Use secure key management (AWS KMS)
- Implement credential rotation

### High Availability
- Deploy across multiple regions
- Implement circuit breakers
- Handle AWS service outages gracefully

This cross-account pattern provides enterprise-grade security while maintaining excellent user experience, following Coiled's proven model.