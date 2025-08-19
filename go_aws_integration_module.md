# aws-external-tool-integration

A Go module for secure AWS integration from external tools (CLI tools, desktop apps, workstation software). Inspired by Coiled's proven UX model but designed for tools that run outside AWS.

## The Problem This Solves

External tools (running on laptops, workstations, CI/CD runners) need AWS access but:
- **Access keys are insecure** - Long-lived, hard to rotate, often committed to git
- **IAM setup is complex** - Users struggle with policies and permissions  
- **No clear permission boundaries** - Tools often ask for overly broad permissions
- **Poor user experience** - "Here's some JSON, figure it out yourself"

## Our Approach: AWS SSO + IAM Roles + Great UX

Instead of access keys, we use:
1. **AWS SSO/SAML integration** for authentication  
2. **IAM roles** for authorization with minimal permissions
3. **Progressive disclosure UI** to guide users through setup
4. **Two-phase permissions** (setup + ongoing)

## Project Structure

```
aws-external-tool-integration/
├── README.md
├── go.mod
├── go.sum
├── LICENSE
├── cmd/
│   └── aws-setup/
│       └── main.go                 # Standalone setup CLI
├── pkg/
│   └── awsauth/
│       ├── client.go               # Main client for external tools
│       ├── sso.go                  # AWS SSO integration
│       ├── credentials.go          # Credential management
│       ├── setup.go                # Interactive setup
│       ├── profiles.go             # AWS profile management
│       └── permissions.go          # Permission templates
├── internal/
│   ├── browser/
│   │   └── launcher.go             # Open browser for SSO flow
│   ├── server/
│   │   └── callback.go             # Local callback server
│   └── templates/
│       ├── setup-ui.html           # Setup web UI
│       └── permissions.yaml        # Permission templates
├── examples/
│   ├── cli-tool/
│   │   └── main.go
│   ├── desktop-app/
│   │   └── main.go
│   └── ci-cd/
│       └── main.go
└── docs/
    ├── getting-started.md
    ├── sso-setup.md
    └── security.md
```

## Installation

```bash
go get github.com/yourusername/aws-external-tool-integration
```

## Quick Start - CLI Tool

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    awsauth "github.com/yourusername/aws-external-tool-integration/pkg/awsauth"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
)

func main() {
    // Initialize auth client
    client, err := awsauth.New(&awsauth.Config{
        ToolName:        "my-awesome-cli",
        RequiredActions: []string{"ec2:DescribeInstances", "s3:ListBuckets"},
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Get AWS credentials (handles all the complexity)
    awsConfig, err := client.GetAWSConfig(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    // Use AWS services normally
    ec2Client := ec2.NewFromConfig(awsConfig)
    result, err := ec2Client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{})
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d reservations\n", len(result.Reservations))
}
```

## Core API Design

### pkg/awsauth/client.go

```go
package awsauth

import (
    "context"
    "fmt"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
)

// Client provides AWS authentication for external tools
type Client struct {
    config      *Config
    profileName string
    credCache   *CredentialCache
}

// Config defines the tool's AWS requirements
type Config struct {
    // Tool identification
    ToolName        string `yaml:"tool_name"`
    ToolVersion     string `yaml:"tool_version"`
    
    // Required permissions (used for setup guidance)
    RequiredActions []string `yaml:"required_actions"`
    
    // Auth preferences
    PreferSSO          bool          `yaml:"prefer_sso"`
    SessionDuration    time.Duration `yaml:"session_duration"`
    ProfileName        string        `yaml:"profile_name"`
    
    // Setup customization
    SetupUI            bool              `yaml:"setup_ui"`
    BrandingOptions    map[string]string `yaml:"branding_options"`
    CustomPermissions  []Permission      `yaml:"custom_permissions"`
}

// New creates a new AWS auth client
func New(config *Config) (*Client, error) {
    if config.ToolName == "" {
        return nil, fmt.Errorf("tool_name is required")
    }
    
    if config.ProfileName == "" {
        config.ProfileName = fmt.Sprintf("%s-profile", config.ToolName)
    }
    
    if config.SessionDuration == 0 {
        config.SessionDuration = 12 * time.Hour
    }
    
    return &Client{
        config:      config,
        profileName: config.ProfileName,
        credCache:   NewCredentialCache(),
    }, nil
}

// GetAWSConfig returns AWS config, handling all authentication complexity
func (c *Client) GetAWSConfig(ctx context.Context) (aws.Config, error) {
    // Try cached credentials first
    if creds := c.credCache.Get(c.profileName); creds != nil {
        if creds.Expires.After(time.Now().Add(10 * time.Minute)) {
            return creds.AWSConfig, nil
        }
    }
    
    // Try existing AWS profile
    if cfg, err := c.tryExistingProfile(ctx); err == nil {
        return cfg, nil
    }
    
    // Need to set up authentication
    return c.setupAuthentication(ctx)
}

// tryExistingProfile attempts to use existing AWS credentials
func (c *Client) tryExistingProfile(ctx context.Context) (aws.Config, error) {
    // Try the tool-specific profile first
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithSharedConfigProfile(c.profileName),
    )
    if err == nil {
        // Verify permissions
        if c.validatePermissions(ctx, cfg) {
            c.credCache.Set(c.profileName, &CachedCredentials{
                AWSConfig: cfg,
                Expires:   time.Now().Add(c.config.SessionDuration),
            })
            return cfg, nil
        }
    }
    
    // Try default profile
    cfg, err = config.LoadDefaultConfig(ctx)
    if err == nil && c.validatePermissions(ctx, cfg) {
        return cfg, nil
    }
    
    return aws.Config{}, fmt.Errorf("no valid AWS credentials found")
}

// setupAuthentication guides user through authentication setup
func (c *Client) setupAuthentication(ctx context.Context) (aws.Config, error) {
    fmt.Printf("🔐 AWS authentication required for %s\n", c.config.ToolName)
    fmt.Printf("Let's get you set up securely!\n\n")
    
    // Check for AWS SSO availability
    if c.config.PreferSSO || c.hasSSO() {
        return c.setupSSO(ctx)
    }
    
    // Fallback to guided profile setup
    return c.setupProfile(ctx)
}

// SetupInteractive runs interactive setup process
func (c *Client) SetupInteractive() error {
    if c.config.SetupUI {
        return c.setupWithUI()
    }
    return c.setupCLI()
}
```

### pkg/awsauth/sso.go

```go
package awsauth

import (
    "context"
    "fmt"
    "net/http"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/sso"
)

// SSOConfig holds AWS SSO configuration
type SSOConfig struct {
    StartURL    string `yaml:"start_url"`
    Region      string `yaml:"region"`
    AccountID   string `yaml:"account_id"`
    RoleName    string `yaml:"role_name"`
}

// setupSSO guides user through AWS SSO setup
func (c *Client) setupSSO(ctx context.Context) (aws.Config, error) {
    fmt.Println("🚀 Setting up AWS SSO authentication...")
    
    // Detect existing SSO configuration
    ssoConfig, err := c.detectSSOConfig()
    if err != nil {
        return c.guidedSSOSetup(ctx)
    }
    
    return c.authenticateSSO(ctx, ssoConfig)
}

// guidedSSOSetup helps user set up SSO from scratch
func (c *Client) guidedSSOSetup(ctx context.Context) (aws.Config, error) {
    fmt.Println("\n📋 AWS SSO Setup Required")
    fmt.Println("We'll help you connect to your organization's AWS SSO.")
    
    var ssoConfig SSOConfig
    
    // Get SSO start URL
    fmt.Print("Enter your AWS SSO start URL: ")
    fmt.Scanln(&ssoConfig.StartURL)
    
    // Get region
    fmt.Print("Enter your AWS region [us-east-1]: ")
    fmt.Scanln(&ssoConfig.Region)
    if ssoConfig.Region == "" {
        ssoConfig.Region = "us-east-1"
    }
    
    // Start device flow
    return c.startSSODeviceFlow(ctx, &ssoConfig)
}

// startSSODeviceFlow initiates AWS SSO device authorization flow
func (c *Client) startSSODeviceFlow(ctx context.Context, ssoConfig *SSOConfig) (aws.Config, error) {
    fmt.Println("\n🔄 Starting SSO authentication...")
    
    // Initialize SSO client
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(ssoConfig.Region))
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to create AWS config: %w", err)
    }
    
    ssoClient := sso.NewFromConfig(cfg)
    
    // Start device authorization
    deviceAuth, err := ssoClient.RegisterClient(ctx, &sso.RegisterClientInput{
        ClientName: aws.String(c.config.ToolName),
        ClientType: aws.String("public"),
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to register SSO client: %w", err)
    }
    
    // Get device code
    deviceCode, err := ssoClient.StartDeviceAuthorization(ctx, &sso.StartDeviceAuthorizationInput{
        ClientId:     deviceAuth.ClientId,
        ClientSecret: deviceAuth.ClientSecret,
        StartUrl:     aws.String(ssoConfig.StartURL),
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to start device authorization: %w", err)
    }
    
    // Display user instructions
    fmt.Printf("\n🌐 Opening browser for authentication...\n")
    fmt.Printf("If browser doesn't open, visit: %s\n", *deviceCode.VerificationUriComplete)
    fmt.Printf("Enter code: %s\n", *deviceCode.UserCode)
    
    // Open browser
    c.openBrowser(*deviceCode.VerificationUriComplete)
    
    // Poll for completion
    return c.pollForSSOToken(ctx, ssoClient, deviceAuth, deviceCode)
}

// openBrowser opens the default browser to the verification URL
func (c *Client) openBrowser(url string) error {
    // Implementation depends on OS
    // Use internal/browser package
    return nil
}
```

### pkg/awsauth/setup.go

```go
package awsauth

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/aws/aws-sdk-go-v2/aws"
)

// setupCLI runs command-line interactive setup
func (c *Client) setupCLI() error {
    fmt.Printf("⚙️  Setting up AWS access for %s\n", c.config.ToolName)
    fmt.Println("Choose your authentication method:")
    fmt.Println("1. AWS SSO (recommended)")
    fmt.Println("2. IAM User credentials")
    fmt.Println("3. Existing AWS profile")
    
    var choice int
    fmt.Print("Enter choice [1]: ")
    fmt.Scanln(&choice)
    
    switch choice {
    case 2:
        return c.setupIAMUser()
    case 3:
        return c.setupExistingProfile()
    default:
        return c.setupSSO(context.Background())
    }
}

// setupIAMUser guides through IAM user setup with CloudFormation
func (c *Client) setupIAMUser() error {
    fmt.Println("\n🔑 Setting up IAM User")
    fmt.Printf("We'll create an IAM user with minimal permissions for %s\n", c.config.ToolName)
    
    // Generate CloudFormation template
    template, err := c.generateIAMTemplate()
    if err != nil {
        return fmt.Errorf("failed to generate template: %w", err)
    }
    
    // Save template to temp file
    tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-iam-setup.yaml", c.config.ToolName))
    if err := os.WriteFile(tempFile, []byte(template), 0644); err != nil {
        return fmt.Errorf("failed to save template: %w", err)
    }
    
    fmt.Printf("\n📄 CloudFormation template saved to: %s\n", tempFile)
    fmt.Println("\nNext steps:")
    fmt.Println("1. Open AWS CloudFormation console")
    fmt.Println("2. Create stack using the template file")
    fmt.Println("3. Copy the Access Key ID and Secret from stack outputs")
    fmt.Println("4. Run this tool again to complete setup")
    
    // Wait for user input
    fmt.Print("\nPress Enter when you have the credentials ready...")
    fmt.Scanln()
    
    return c.promptForCredentials()
}

// generateIAMTemplate creates CloudFormation template for IAM user
func (c *Client) generateIAMTemplate() (string, error) {
    permissions := c.buildPermissionStatements()
    
    template := fmt.Sprintf(`AWSTemplateFormatVersion: '2010-09-09'
Description: 'IAM User for %s external tool'

Resources:
  %sUser:
    Type: AWS::IAM::User
    Properties:
      UserName: !Sub '%s-user-${AWS::AccountId}'
      Path: '/external-tools/'
      
  %sUserAccessKey:
    Type: AWS::IAM::AccessKey
    Properties:
      UserName: !Ref %sUser
      
  %sUserPolicy:
    Type: AWS::IAM::UserPolicy
    Properties:
      UserName: !Ref %sUser
      PolicyName: '%sPermissions'
      PolicyDocument:
        Version: '2012-10-17'
        Statement:
%s

Outputs:
  AccessKeyId:
    Description: 'Access Key ID for %s'
    Value: !Ref %sUserAccessKey
    
  SecretAccessKey:
    Description: 'Secret Access Key (store securely!)'
    Value: !GetAtt %sUserAccessKey.SecretAccessKey
    
  Instructions:
    Description: 'Next steps'
    Value: 'Copy these credentials and run your tool setup again'
`, 
        c.config.ToolName, // Description
        c.config.ToolName, // User resource name
        c.config.ToolName, // UserName prefix
        c.config.ToolName, // AccessKey resource name  
        c.config.ToolName, // User reference
        c.config.ToolName, // Policy resource name
        c.config.ToolName, // User reference
        c.config.ToolName, // Policy name
        permissions,       // Permission statements
        c.config.ToolName, // Output description
        c.config.ToolName, // AccessKey reference
        c.config.ToolName, // AccessKey reference
    )
    
    return template, nil
}

// buildPermissionStatements creates IAM policy statements
func (c *Client) buildPermissionStatements() string {
    if len(c.config.CustomPermissions) > 0 {
        return c.formatCustomPermissions()
    }
    
    // Build from required actions
    actions := c.config.RequiredActions
    if len(actions) == 0 {
        // Minimal default permissions
        actions = []string{
            "sts:GetCallerIdentity",
            "iam:GetUser",
        }
    }
    
    return fmt.Sprintf(`          - Sid: '%sPermissions'
            Effect: Allow
            Action:
%s
            Resource: '*'`,
        c.config.ToolName,
        c.formatActions(actions),
    )
}

// formatActions formats action list for YAML
func (c *Client) formatActions(actions []string) string {
    result := ""
    for _, action := range actions {
        result += fmt.Sprintf("              - '%s'\n", action)
    }
    return result
}

// promptForCredentials prompts user to enter AWS credentials
func (c *Client) promptForCredentials() error {
    var accessKey, secretKey string
    
    fmt.Print("Enter Access Key ID: ")
    fmt.Scanln(&accessKey)
    
    fmt.Print("Enter Secret Access Key: ")
    fmt.Scanln(&secretKey)
    
    // Save to AWS credentials file
    return c.saveCredentialsProfile(accessKey, secretKey)
}

// saveCredentialsProfile saves credentials to AWS profile
func (c *Client) saveCredentialsProfile(accessKey, secretKey string) error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return fmt.Errorf("failed to get home directory: %w", err)
    }
    
    credFile := filepath.Join(homeDir, ".aws", "credentials")
    
    // Ensure .aws directory exists
    if err := os.MkdirAll(filepath.Dir(credFile), 0755); err != nil {
        return fmt.Errorf("failed to create .aws directory: %w", err)
    }
    
    // Read existing credentials
    content := ""
    if data, err := os.ReadFile(credFile); err == nil {
        content = string(data)
    }
    
    // Add our profile
    profile := fmt.Sprintf("\n[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\n",
        c.profileName, accessKey, secretKey)
    
    content += profile
    
    // Write back
    if err := os.WriteFile(credFile, []byte(content), 0600); err != nil {
        return fmt.Errorf("failed to save credentials: %w", err)
    }
    
    fmt.Printf("✅ Credentials saved to profile: %s\n", c.profileName)
    return nil
}
```

### Examples for Different Tool Types

### examples/cli-tool/main.go

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    
    awsauth "github.com/yourusername/aws-external-tool-integration/pkg/awsauth"
    "github.com/aws/aws-sdk-go-v2/service/ec2"
)

func main() {
    var setupFlag = flag.Bool("setup", false, "Run AWS authentication setup")
    flag.Parse()
    
    // Initialize auth client
    client, err := awsauth.New(&awsauth.Config{
        ToolName:        "my-ec2-tool",
        ToolVersion:     "1.0.0",
        RequiredActions: []string{
            "ec2:DescribeInstances",
            "ec2:DescribeImages",
            "ec2:StartInstances",
            "ec2:StopInstances",
        },
        PreferSSO:       true,
        SessionDuration: 12 * time.Hour,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Handle setup command
    if *setupFlag {
        if err := client.SetupInteractive(); err != nil {
            log.Fatal("Setup failed:", err)
        }
        fmt.Println("✅ Setup completed successfully!")
        return
    }
    
    // Get AWS config
    awsConfig, err := client.GetAWSConfig(context.Background())
    if err != nil {
        fmt.Printf("❌ AWS authentication failed: %v\n", err)
        fmt.Println("Run with --setup to configure authentication")
        os.Exit(1)
    }
    
    // Use AWS services
    ec2Client := ec2.NewFromConfig(awsConfig)
    
    switch flag.Arg(0) {
    case "list":
        listInstances(context.Background(), ec2Client)
    case "start":
        if flag.NArg() < 2 {
            fmt.Println("Usage: my-ec2-tool start <instance-id>")
            os.Exit(1)
        }
        startInstance(context.Background(), ec2Client, flag.Arg(1))
    default:
        fmt.Println("Available commands: list, start")
    }
}

func listInstances(ctx context.Context, client *ec2.Client) {
    result, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
    if err != nil {
        log.Fatal("Failed to list instances:", err)
    }
    
    fmt.Println("EC2 Instances:")
    for _, reservation := range result.Reservations {
        for _, instance := range reservation.Instances {
            name := "unnamed"
            for _, tag := range instance.Tags {
                if *tag.Key == "Name" {
                    name = *tag.Value
                    break
                }
            }
            fmt.Printf("  %s (%s) - %s\n", *instance.InstanceId, name, instance.State.Name)
        }
    }
}

func startInstance(ctx context.Context, client *ec2.Client, instanceID string) {
    _, err := client.StartInstances(ctx, &ec2.StartInstancesInput{
        InstanceIds: []string{instanceID},
    })
    if err != nil {
        log.Fatal("Failed to start instance:", err)
    }
    fmt.Printf("✅ Started instance %s\n", instanceID)
}
```

### examples/desktop-app/main.go

```go
package main

import (
    "context"
    "log"
    
    awsauth "github.com/yourusername/aws-external-tool-integration/pkg/awsauth"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/widget"
)

func main() {
    myApp := app.New()
    myWindow := myApp.NewWindow("AWS Desktop Tool")
    
    // Initialize AWS auth with UI enabled
    client, err := awsauth.New(&awsauth.Config{
        ToolName:        "aws-desktop-tool",
        RequiredActions: []string{"s3:ListBuckets", "s3:GetObject"},
        SetupUI:         true, // Enable web UI for setup
        BrandingOptions: map[string]string{
            "primary_color": "#ff6b35",
            "company_name":  "My Company",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Check if AWS is configured
    awsConfig, err := client.GetAWSConfig(context.Background())
    if err != nil {
        // Show setup button
        setupBtn := widget.NewButton("Configure AWS Access", func() {
            client.SetupInteractive() // This will open browser UI
        })
        
        myWindow.SetContent(widget.NewVBox(
            widget.NewLabel("AWS configuration required"),
            setupBtn,
        ))
    } else {
        // Show main app UI
        myWindow.SetContent(widget.NewLabel("AWS configured successfully!"))
        // ... rest of app logic
    }
    
    myWindow.ShowAndRun()
}
```

### examples/ci-cd/main.go

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    awsauth "github.com/yourusername/aws-external-tool-integration/pkg/awsauth"
)

func main() {
    // For CI/CD, prefer environment variables or instance roles
    client, err := awsauth.New(&awsauth.Config{
        ToolName:        "ci-deployment-tool",
        RequiredActions: []string{"ecs:UpdateService", "ecs:DescribeServices"},
        PreferSSO:       false, // SSO doesn't work in CI
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // In CI/CD, we want to fail fast if no creds available
    awsConfig, err := client.GetAWSConfig(context.Background())
    if err != nil {
        fmt.Printf("❌ AWS credentials not found\n")
        fmt.Println("For CI/CD, ensure one of:")
        fmt.Println("- AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables")
        fmt.Println("- IAM instance role (if running on EC2)")
        fmt.Println("- AWS_PROFILE environment variable pointing to configured profile")
        os.Exit(1)
    }
    
    // Use awsConfig for deployment operations
    fmt.Println("✅ AWS credentials found, proceeding with deployment")
}
```

## Key Differences from Cross-Account Model

### 1. **No Service Account** 
- External tools don't have their own AWS account
- Users provide credentials to the tool directly

### 2. **Local Credential Storage**
- Credentials stored in `~/.aws/credentials` or similar
- Uses AWS SDK's standard credential chain

### 3. **SSO-First Approach**
- AWS SSO provides better security for workstation tools
- Temporary credentials with automatic refresh

### 4. **Progressive Setup UX**
- Guides users through CloudFormation template deployment
- Explains exactly what permissions are needed and why
- Provides fallback options for different environments

This approach gives you all the UX benefits of Coiled's model while being appropriate for external tools that run on user workstations.