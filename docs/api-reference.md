# API Reference

This document provides comprehensive API documentation for the AWS Remote Access Patterns Go library.

## 📚 Package Overview

### Core Packages

- **`pkg/crossaccount`**: Cross-account AWS role management for SaaS services
- **`pkg/awsauth`**: External tool AWS authentication for CLI/desktop applications

---

## 📦 pkg/crossaccount

For SaaS services that need secure access to customer AWS accounts.

### Client

The main client for cross-account operations.

#### type Client

```go
type Client struct {
    // contains filtered or unexported fields
}
```

#### func New

```go
func New(cfg *Config) (*Client, error)
```

Creates a new cross-account client.

**Parameters:**
- `cfg`: Configuration object defining service requirements

**Returns:**
- `*Client`: Configured client instance
- `error`: Configuration validation error

**Example:**
```go
config := crossaccount.SimpleConfig(
    "MyService",
    "123456789012", // Your AWS account ID
    "my-templates-bucket",
)

client, err := crossaccount.New(config)
if err != nil {
    log.Fatal(err)
}
```

#### func (*Client) GenerateSetupLink

```go
func (c *Client) GenerateSetupLink(customerID, customerName string) (*SetupResponse, error)
```

Generates a one-click CloudFormation setup link for customers.

**Parameters:**
- `customerID`: Unique identifier for the customer
- `customerName`: Human-readable customer name

**Returns:**
- `*SetupResponse`: Setup link and metadata
- `error`: Generation error

**Example:**
```go
setup, err := client.GenerateSetupLink("customer-123", "Acme Corp")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Setup URL: %s\n", setup.LaunchURL)
fmt.Printf("External ID: %s\n", setup.ExternalID)
```

#### func (*Client) CompleteSetup

```go
func (c *Client) CompleteSetup(ctx context.Context, req *SetupCompleteRequest) error
```

Completes customer setup after CloudFormation stack creation.

**Parameters:**
- `ctx`: Context for cancellation and deadlines
- `req`: Setup completion request with role details

**Returns:**
- `error`: Setup validation error

**Example:**
```go
err := client.CompleteSetup(ctx, &crossaccount.SetupCompleteRequest{
    CustomerID: "customer-123",
    RoleARN:    "arn:aws:iam::999999999999:role/MyService-CrossAccount",
    ExternalID: "MyService-customer-123-abc123def456",
})
if err != nil {
    log.Fatal(err)
}
```

#### func (*Client) AssumeRole

```go
func (c *Client) AssumeRole(ctx context.Context, customerID string) (aws.Config, error)
```

Assumes the customer's role and returns AWS config for API calls.

**Parameters:**
- `ctx`: Context for cancellation and deadlines  
- `customerID`: Customer identifier

**Returns:**
- `aws.Config`: AWS configuration with temporary credentials
- `error`: Assume role error

**Example:**
```go
awsConfig, err := client.AssumeRole(ctx, "customer-123")
if err != nil {
    log.Fatal(err)
}

// Use the config with any AWS service
s3Client := s3.NewFromConfig(awsConfig)
buckets, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
```

#### func (*Client) GenerateCloudFormationTemplate

```go
func (c *Client) GenerateCloudFormationTemplate() (string, error)
```

Generates a CloudFormation template for the cross-account role.

**Returns:**
- `string`: YAML CloudFormation template
- `error`: Template generation error

### Configuration

#### type Config

```go
type Config struct {
    ServiceName          string        `json:"service_name" yaml:"service_name"`
    ServiceAccountID     string        `json:"service_account_id" yaml:"service_account_id"`
    TemplateS3Bucket     string        `json:"template_s3_bucket" yaml:"template_s3_bucket"`
    DefaultRegion        string        `json:"default_region" yaml:"default_region"`
    SessionDuration      time.Duration `json:"session_duration" yaml:"session_duration"`
    OngoingPermissions   []Permission  `json:"ongoing_permissions" yaml:"ongoing_permissions"`
    SetupPermissions     []Permission  `json:"setup_permissions" yaml:"setup_permissions"`
    BrandingOptions      map[string]string `json:"branding_options" yaml:"branding_options"`
}
```

Configuration for cross-account integration.

#### func SimpleConfig

```go
func SimpleConfig(serviceName, serviceAccountID, templateBucket string) *Config
```

Creates a configuration with minimal required fields.

**Parameters:**
- `serviceName`: Name of your service
- `serviceAccountID`: Your AWS account ID (12 digits)
- `templateBucket`: S3 bucket for hosting CloudFormation templates

**Example:**
```go
config := crossaccount.SimpleConfig(
    "DataPlatform", 
    "123456789012",
    "dataplatform-cf-templates",
)
```

#### func QuickConfig

```go
func QuickConfig(serviceType, serviceName, serviceAccountID, templateBucket string) *Config
```

Creates configuration with common permissions for different service types.

**Parameters:**
- `serviceType`: Type of service ("data-platform", "compute-platform", "monitoring-platform")
- `serviceName`: Name of your service
- `serviceAccountID`: Your AWS account ID
- `templateBucket`: S3 bucket for templates

**Example:**
```go
config := crossaccount.QuickConfig(
    "data-platform",
    "DataPlatform",
    "123456789012", 
    "dataplatform-templates",
)
```

### Data Types

#### type SetupResponse

```go
type SetupResponse struct {
    LaunchURL     string `json:"launch_url"`
    ExternalID    string `json:"external_id"`
    CustomerID    string `json:"customer_id"`
    StackName     string `json:"stack_name"`
    SetupComplete bool   `json:"setup_complete"`
}
```

Response from setup link generation.

#### type SetupCompleteRequest

```go
type SetupCompleteRequest struct {
    CustomerID string `json:"customer_id"`
    RoleARN    string `json:"role_arn"`
    ExternalID string `json:"external_id"`
}
```

Request for completing customer setup.

#### type Permission

```go
type Permission struct {
    Sid       string                 `json:"sid" yaml:"sid"`
    Effect    string                 `json:"effect" yaml:"effect"`
    Actions   []string               `json:"actions" yaml:"actions"`
    Resources []string               `json:"resources" yaml:"resources"`
    Condition map[string]interface{} `json:"condition,omitempty" yaml:"condition,omitempty"`
}
```

IAM policy statement for role permissions.

### Storage Interface

#### type CredentialStorage

```go
type CredentialStorage interface {
    Store(ctx context.Context, customerID string, creds CustomerCredentials) error
    Get(ctx context.Context, customerID string) (*CustomerCredentials, error)
    Delete(ctx context.Context, customerID string) error
    List(ctx context.Context) ([]string, error)
}
```

Interface for storing customer credentials. Implement this for production storage backends.

---

## 📦 pkg/awsauth

For external tools (CLI, desktop apps) that need AWS authentication.

### Client

The main client for external tool authentication.

#### type Client

```go
type Client struct {
    // contains filtered or unexported fields
}
```

#### func New

```go
func New(cfg *Config, opts ...Option) (*Client, error)
```

Creates a new AWS authentication client for external tools.

**Parameters:**
- `cfg`: Configuration object
- `opts`: Optional configuration functions

**Returns:**
- `*Client`: Configured client instance
- `error`: Configuration validation error

**Example:**
```go
config := awsauth.DefaultConfig("my-cli-tool")
config.RequiredActions = []string{
    "ec2:DescribeInstances",
    "s3:ListBuckets",
}

client, err := awsauth.New(config,
    awsauth.WithProfileName("my-tool-profile"),
)
if err != nil {
    log.Fatal(err)
}
```

#### func (*Client) GetAWSConfig

```go
func (c *Client) GetAWSConfig(ctx context.Context) (aws.Config, error)
```

Gets AWS configuration, handling credential discovery and setup automatically.

**Parameters:**
- `ctx`: Context for cancellation and deadlines

**Returns:**
- `aws.Config`: AWS configuration ready for use
- `error`: Authentication error

**Example:**
```go
awsConfig, err := client.GetAWSConfig(ctx)
if err != nil {
    // This means authentication is required
    fmt.Println("Run with --setup to configure AWS access")
    os.Exit(1)
}

// Use with any AWS service
ec2Client := ec2.NewFromConfig(awsConfig)
instances, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
```

#### func (*Client) RunSetup

```go
func (c *Client) RunSetup(ctx context.Context) error
```

Runs interactive setup process to configure AWS authentication.

**Parameters:**
- `ctx`: Context for cancellation and deadlines

**Returns:**
- `error`: Setup error

**Example:**
```go
if *setupFlag {
    if err := client.RunSetup(ctx); err != nil {
        log.Fatal("Setup failed:", err)
    }
    fmt.Println("✅ Setup completed!")
    return
}
```

### Configuration

#### type Config

```go
type Config struct {
    ToolName         string        `json:"tool_name" yaml:"tool_name"`
    ToolVersion      string        `json:"tool_version" yaml:"tool_version"`
    DefaultRegion    string        `json:"default_region" yaml:"default_region"`
    ProfileName      string        `json:"profile_name" yaml:"profile_name"`
    SessionDuration  time.Duration `json:"session_duration" yaml:"session_duration"`
    RequiredActions  []string      `json:"required_actions" yaml:"required_actions"`
    PreferSSO        bool          `json:"prefer_sso" yaml:"prefer_sso"`
    AllowIAMUser     bool          `json:"allow_iam_user" yaml:"allow_iam_user"`
    AllowEnvVars     bool          `json:"allow_env_vars" yaml:"allow_env_vars"`
    SetupUI          bool          `json:"setup_ui" yaml:"setup_ui"`
    CIMode           bool          `json:"ci_mode" yaml:"ci_mode"`
}
```

Configuration for external tool authentication.

#### func DefaultConfig

```go
func DefaultConfig(toolName string) *Config
```

Creates configuration with sensible defaults for CLI tools.

**Parameters:**
- `toolName`: Name of your tool

**Example:**
```go
config := awsauth.DefaultConfig("my-awesome-cli")
```

#### func CICDConfig

```go
func CICDConfig(toolName string) *Config
```

Creates configuration optimized for CI/CD environments.

**Parameters:**
- `toolName`: Name of your tool

**Example:**
```go
config := awsauth.CICDConfig("deployment-tool")
```

### Options

#### type Option

```go
type Option func(*Client)
```

Functional option for configuring the client.

#### func WithProfileName

```go
func WithProfileName(name string) Option
```

Sets a custom AWS profile name.

#### func WithCredentialCache

```go
func WithCredentialCache(cache *CredentialCache) Option
```

Sets a custom credential cache implementation.

### SSO Authentication

#### type SSOAuthenticator

```go
type SSOAuthenticator struct {
    // contains filtered or unexported fields
}
```

Handles AWS SSO device flow authentication.

#### func NewSSOAuthenticator

```go
func NewSSOAuthenticator(cfg *Config) *SSOAuthenticator
```

Creates a new SSO authenticator.

#### func (*SSOAuthenticator) Authenticate

```go
func (s *SSOAuthenticator) Authenticate(ctx context.Context) (aws.Config, error)
```

Performs AWS SSO device flow authentication.

### Credential Management

#### type CredentialManager

```go
type CredentialManager struct {
    // contains filtered or unexported fields
}
```

Manages AWS credential storage and retrieval.

#### func NewCredentialManager

```go
func NewCredentialManager(profileName, region string) *CredentialManager
```

Creates a new credential manager.

#### func (*CredentialManager) SaveProfile

```go
func (cm *CredentialManager) SaveProfile(accessKey, secretKey, sessionToken string) error
```

Saves AWS credentials to a specific profile.

#### func (*CredentialManager) LoadProfile

```go
func (cm *CredentialManager) LoadProfile(ctx context.Context) (aws.Config, error)
```

Loads AWS credentials from a profile.

---

## 🔧 Utility Functions

### Template Functions

#### func GetTemplateContent

```go
func GetTemplateContent(templateType string) (string, error)
```

Returns raw CloudFormation template content.

**Parameters:**
- `templateType`: Template type ("cross-account" or "iam-user")

#### func ValidateTemplate

```go
func ValidateTemplate(templateContent string) error
```

Performs basic validation on CloudFormation template.

#### func RenderTemplate

```go
func RenderTemplate(templateContent string, vars TemplateVariables) (string, error)
```

Renders template with variable substitution.

---

## 🌟 Common Usage Patterns

### SaaS Service Integration

```go
// Initialize client
config := crossaccount.QuickConfig(
    "data-platform",
    "MyDataPlatform", 
    "123456789012",
    "mydataplatform-templates",
)

client, err := crossaccount.New(config)
if err != nil {
    log.Fatal(err)
}

// Generate setup link for customer
setup, err := client.GenerateSetupLink("customer-123", "Acme Corp")
if err != nil {
    log.Fatal(err)
}

// Send setup.LaunchURL to customer
fmt.Printf("Customer setup link: %s\n", setup.LaunchURL)

// After customer completes CloudFormation:
err = client.CompleteSetup(ctx, &crossaccount.SetupCompleteRequest{
    CustomerID: "customer-123",
    RoleARN:    roleArnFromCustomer,
    ExternalID: setup.ExternalID,
})

// Access customer resources
awsConfig, err := client.AssumeRole(ctx, "customer-123") 
if err != nil {
    log.Fatal(err)
}

s3Client := s3.NewFromConfig(awsConfig)
buckets, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
```

### CLI Tool Authentication

```go
// Initialize auth client
config := awsauth.DefaultConfig("my-cli")
config.RequiredActions = []string{
    "ec2:DescribeInstances",
    "ec2:StartInstances",
    "ec2:StopInstances",
}

client, err := awsauth.New(config)
if err != nil {
    log.Fatal(err)
}

// Handle setup command
if *setupFlag {
    if err := client.RunSetup(ctx); err != nil {
        log.Fatal(err)
    }
    fmt.Println("✅ Setup complete!")
    return
}

// Get AWS config (prompts for setup if needed)
awsConfig, err := client.GetAWSConfig(ctx)
if err != nil {
    fmt.Printf("❌ AWS authentication required: %v\n", err)
    fmt.Println("Run with --setup to configure")
    os.Exit(1)
}

// Use AWS services
ec2Client := ec2.NewFromConfig(awsConfig)
instances, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
```

### Desktop Application with UI

```go
config := awsauth.DefaultConfig("my-desktop-app")
config.SetupUI = true // Enable web UI
config.BrandingOptions = map[string]string{
    "primary_color": "#2196F3",
    "company_name":  "My Company",
}

client, err := awsauth.New(config)
if err != nil {
    log.Fatal(err)
}

// Check if authentication is configured
awsConfig, err := client.GetAWSConfig(ctx)
if err != nil {
    // Show setup button in UI that calls:
    // client.RunSetup(ctx)  // This opens web browser for setup
    showSetupDialog()
} else {
    showMainApplicationUI()
}
```

---

## 🚨 Error Handling

### Common Errors

#### CrossAccount Package

- `invalid config: *`: Configuration validation failed
- `customer * not found`: Customer hasn't completed setup
- `failed to assume role: *`: Role assumption failed (permissions/trust policy issue)
- `setup failed: *`: Setup process encountered error

#### AWSAuth Package  

- `no valid AWS credentials found`: No existing credentials available
- `setup failed: *`: Interactive setup failed
- `SSO authentication failed: *`: SSO device flow failed
- `profile * doesn't have required permissions`: Credentials lack required permissions

### Error Handling Patterns

```go
// Cross-account role assumption
awsConfig, err := client.AssumeRole(ctx, customerID)
if err != nil {
    var notFoundErr *CustomerNotFoundError
    if errors.As(err, &notFoundErr) {
        // Customer needs to complete setup
        return setupRequired(customerID)
    }
    
    // Other errors (permissions, network, etc.)
    return fmt.Errorf("failed to access customer account: %w", err)
}

// External tool authentication
awsConfig, err := client.GetAWSConfig(ctx)
if err != nil {
    if strings.Contains(err.Error(), "no valid AWS credentials") {
        // Need to run setup
        fmt.Println("Run with --setup to configure AWS access")
        os.Exit(1)
    }
    
    // Other errors
    return fmt.Errorf("AWS authentication failed: %w", err)
}
```

---

## 🔒 Security Considerations

- **External IDs**: Always use the generated external IDs for security
- **Permissions**: Request only the minimum required permissions  
- **Session Duration**: Use appropriate session durations (shorter for higher security)
- **Credential Storage**: Never log or expose temporary credentials
- **Validation**: Always validate role access before storing credentials

---

## 📈 Performance Tips

- **Caching**: Use credential caching to avoid repeated role assumptions
- **Batching**: Batch AWS API calls when possible
- **Regions**: Choose regions close to your users
- **Connection Pooling**: Reuse HTTP connections for AWS API calls
- **Rate Limiting**: Implement backoff for AWS API rate limits

---

This API reference provides the complete interface for both cross-account and external tool authentication patterns. For additional examples and guides, see the other documentation files.