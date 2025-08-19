# External Tool AWS Integration

Complete implementation guide for tools that run outside AWS (laptops, workstations, CI/CD) and need secure AWS access.

## Overview

This pattern is for **tools running outside AWS** that need access to **user AWS accounts**. Think AWS CLI, Terraform, kubectl, or any desktop/CLI tool that manages AWS resources.

### When to Use This Pattern

- ✅ Your tool runs on user workstations/laptops
- ✅ Users have their own AWS accounts
- ✅ You need AWS API access from the tool
- ✅ You want to avoid long-lived access keys
- ✅ You support both individual users and enterprises
- ✅ You need to work in CI/CD environments

### Architecture Overview

```
┌─────────────────────┐    ┌─────────────────────┐
│   User Workstation  │    │    User AWS Account │
│                     │    │                     │
│  ┌───────────────┐  │    │  ┌───────────────┐  │
│  │ Your Tool     │  │    │  │ IAM User/Role │  │
│  │ (CLI/Desktop) │◄─┼────┼─►│ + Permissions │  │
│  └───────────────┘  │    │  └───────────────┘  │
│                     │    │                     │
│  AWS Credentials:   │    │  AWS SSO:           │
│  ~/.aws/credentials │    │  Temporary tokens   │
└─────────────────────┘    └─────────────────────┘
```

## Complete Implementation

### Project Structure

```
aws-external-tool-integration/
├── README.md
├── go.mod
├── cmd/
│   ├── setup/
│   │   └── main.go         # Standalone setup CLI
│   └── example/
│       └── main.go
├── pkg/
│   └── awsauth/
│       ├── client.go       # Main client
│       ├── config.go       # Configuration
│       ├── sso.go          # AWS SSO integration
│       ├── profiles.go     # AWS profile management
│       ├── setup.go        # Interactive setup
│       ├── credentials.go  # Credential management
│       └── validation.go   # Permission validation
├── internal/
│   ├── browser/
│   │   └── opener.go       # Browser launcher
│   ├── server/
│   │   └── callback.go     # Local callback server
│   └── templates/
│       ├── cf-template.yaml # CloudFormation templates
│       └── setup-ui.html   # Setup web interface
├── web/
│   ├── static/
│   └── templates/
└── examples/
    ├── cli-tool/
    ├── desktop-app/
    ├── ci-cd-runner/
    └── terraform-provider/
```

### Core Implementation

#### pkg/awsauth/client.go

```go
package awsauth

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/sts"
)

type Client struct {
    config      *Config
    profileName string
    credCache   *CredentialCache
    setupUI     *SetupUI
}

func New(cfg *Config, opts ...Option) (*Client, error) {
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    profileName := cfg.ProfileName
    if profileName == "" {
        profileName = fmt.Sprintf("%s-profile", cfg.ToolName)
    }

    c := &Client{
        config:      cfg,
        profileName: profileName,
        credCache:   NewCredentialCache(),
        setupUI:     NewSetupUI(cfg),
    }

    for _, opt := range opts {
        opt(c)
    }

    return c, nil
}

type Option func(*Client)

func WithProfileName(name string) Option {
    return func(c *Client) { c.profileName = name }
}

func WithCredentialCache(cache *CredentialCache) Option {
    return func(c *Client) { c.credCache = cache }
}

// GetAWSConfig returns AWS config, handling all authentication complexity
func (c *Client) GetAWSConfig(ctx context.Context) (aws.Config, error) {
    // Try cached credentials first
    if creds := c.credCache.Get(c.profileName); creds != nil && creds.IsValid() {
        return creds.AWSConfig, nil
    }

    // Try existing AWS configuration
    if cfg, err := c.tryExistingCredentials(ctx); err == nil {
        c.cacheCredentials(cfg)
        return cfg, nil
    }

    // Need setup
    fmt.Printf("🔐 AWS authentication required for %s\n", c.config.ToolName)
    return c.runSetup(ctx)
}

// tryExistingCredentials attempts to use existing AWS credentials
func (c *Client) tryExistingCredentials(ctx context.Context) (aws.Config, error) {
    // Try tool-specific profile
    if cfg, err := c.loadProfile(ctx, c.profileName); err == nil {
        if c.validateCredentials(ctx, cfg) {
            return cfg, nil
        }
    }

    // Try default profile
    if cfg, err := c.loadProfile(ctx, "default"); err == nil {
        if c.validateCredentials(ctx, cfg) {
            return cfg, nil
        }
    }

    // Try environment variables
    if cfg, err := config.LoadDefaultConfig(ctx); err == nil {
        if c.validateCredentials(ctx, cfg) {
            return cfg, nil
        }
    }

    return aws.Config{}, fmt.Errorf("no valid AWS credentials found")
}

func (c *Client) loadProfile(ctx context.Context, profileName string) (aws.Config, error) {
    return config.LoadDefaultConfig(ctx,
        config.WithSharedConfigProfile(profileName),
        config.WithRegion(c.config.DefaultRegion),
    )
}

func (c *Client) validateCredentials(ctx context.Context, cfg aws.Config) bool {
    stsClient := sts.NewFromConfig(cfg)
    
    // Test basic access
    if _, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err != nil {
        return false
    }

    // Validate required permissions
    return c.validatePermissions(ctx, cfg)
}

func (c *Client) validatePermissions(ctx context.Context, cfg aws.Config) bool {
    // Implement permission validation based on RequiredActions
    // This is a simplified check - in practice, you'd want more comprehensive testing
    for _, action := range c.config.RequiredActions {
        if !c.testAction(ctx, cfg, action) {
            return false
        }
    }
    return true
}

func (c *Client) testAction(ctx context.Context, cfg aws.Config, action string) bool {
    // Implement action-specific testing
    // This is tool-specific and would need to be customized
    switch action {
    case "sts:GetCallerIdentity":
        stsClient := sts.NewFromConfig(cfg)
        _, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
        return err == nil
    // Add more action tests as needed
    default:
        // For unknown actions, assume they're valid
        return true
    }
}

// RunSetup initiates the interactive setup process
func (c *Client) RunSetup(ctx context.Context) error {
    if c.config.SetupUI {
        return c.setupUI.Launch(ctx)
    }
    return c.runCLISetup(ctx)
}

func (c *Client) runSetup(ctx context.Context) (aws.Config, error) {
    if err := c.RunSetup(ctx); err != nil {
        return aws.Config{}, fmt.Errorf("setup failed: %w", err)
    }

    // After setup, try to load credentials again
    return c.tryExistingCredentials(ctx)
}

func (c *Client) cacheCredentials(cfg aws.Config) {
    c.credCache.Set(c.profileName, &CachedCredentials{
        AWSConfig: cfg,
        ExpiresAt: time.Now().Add(c.config.SessionDuration),
    })
}
```

#### pkg/awsauth/config.go

```go
package awsauth

import (
    "errors"
    "time"
)

type Config struct {
    // Tool identification
    ToolName        string `json:"tool_name" yaml:"tool_name"`
    ToolVersion     string `json:"tool_version" yaml:"tool_version"`
    
    // AWS settings
    DefaultRegion     string        `json:"default_region" yaml:"default_region"`
    ProfileName       string        `json:"profile_name" yaml:"profile_name"`
    SessionDuration   time.Duration `json:"session_duration" yaml:"session_duration"`
    
    // Required permissions
    RequiredActions   []string     `json:"required_actions" yaml:"required_actions"`
    CustomPermissions []Permission `json:"custom_permissions" yaml:"custom_permissions"`
    
    // Authentication preferences
    PreferSSO         bool `json:"prefer_sso" yaml:"prefer_sso"`
    AllowIAMUser      bool `json:"allow_iam_user" yaml:"allow_iam_user"`
    AllowEnvVars      bool `json:"allow_env_vars" yaml:"allow_env_vars"`
    
    // Setup options
    SetupUI           bool              `json:"setup_ui" yaml:"setup_ui"`
    BrandingOptions   map[string]string `json:"branding_options" yaml:"branding_options"`
    
    // CI/CD settings
    CIMode            bool `json:"ci_mode" yaml:"ci_mode"`
}

type Permission struct {
    Sid       string                 `json:"sid" yaml:"sid"`
    Effect    string                 `json:"effect" yaml:"effect"`
    Actions   []string              `json:"actions" yaml:"actions"`
    Resources []string              `json:"resources" yaml:"resources"`
    Condition map[string]interface{} `json:"condition,omitempty" yaml:"condition,omitempty"`
}

func (c *Config) Validate() error {
    if c.ToolName == "" {
        return errors.New("tool_name is required")
    }
    
    // Set defaults
    if c.DefaultRegion == "" {
        c.DefaultRegion = "us-east-1"
    }
    if c.SessionDuration == 0 {
        c.SessionDuration = 12 * time.Hour
    }
    if c.RequiredActions == nil {
        c.RequiredActions = []string{"sts:GetCallerIdentity"}
    }
    
    // Enable reasonable defaults
    if !c.PreferSSO && !c.AllowIAMUser && !c.AllowEnvVars {
        c.PreferSSO = true
        c.AllowIAMUser = true
        c.AllowEnvVars = true
    }
    
    return nil
}

// DefaultConfig returns a config with sensible defaults for most CLI tools
func DefaultConfig(toolName string) *Config {
    return &Config{
        ToolName:        toolName,
        DefaultRegion:   "us-east-1",
        SessionDuration: 12 * time.Hour,
        RequiredActions: []string{
            "sts:GetCallerIdentity",
        },
        PreferSSO:    true,
        AllowIAMUser: true,
        AllowEnvVars: true,
        SetupUI:      true,
    }
}

// CICDConfig returns a config optimized for CI/CD environments
func CICDConfig(toolName string) *Config {
    return &Config{
        ToolName:        toolName,
        DefaultRegion:   "us-east-1",
        SessionDuration: 1 * time.Hour,
        RequiredActions: []string{
            "sts:GetCallerIdentity",
        },
        PreferSSO:    false, // SSO doesn't work in CI
        AllowIAMUser: true,
        AllowEnvVars: true,
        SetupUI:      false, // No UI in CI
        CIMode:       true,
    }
}

type CachedCredentials struct {
    AWSConfig aws.Config
    ExpiresAt time.Time
}

func (c *CachedCredentials) IsValid() bool {
    return time.Now().Before(c.ExpiresAt.Add(-5 * time.Minute)) // 5min buffer
}

type CredentialCache struct {
    cache map[string]*CachedCredentials
}

func NewCredentialCache() *CredentialCache {
    return &CredentialCache{
        cache: make(map[string]*CachedCredentials),
    }
}

func (c *CredentialCache) Get(key string) *CachedCredentials {
    if creds, ok := c.cache[key]; ok && creds.IsValid() {
        return creds
    }
    return nil
}

func (c *CredentialCache) Set(key string, creds *CachedCredentials) {
    c.cache[key] = creds
}
```

#### pkg/awsauth/sso.go

```go
package awsauth

import (
    "context"
    "fmt"
    "net/http"
    "net/url"
    "os/exec"
    "runtime"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/sso"
    "github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

// SSOAuthenticator handles AWS SSO authentication
type SSOAuthenticator struct {
    config   *Config
    startURL string
    region   string
}

func NewSSOAuthenticator(cfg *Config) *SSOAuthenticator {
    return &SSOAuthenticator{
        config: cfg,
        region: cfg.DefaultRegion,
    }
}

// Authenticate performs AWS SSO device flow authentication
func (s *SSOAuthenticator) Authenticate(ctx context.Context) (aws.Config, error) {
    fmt.Printf("🚀 Starting AWS SSO authentication for %s\n", s.config.ToolName)

    // Get SSO configuration
    ssoConfig, err := s.getSSOConfig(ctx)
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to get SSO config: %w", err)
    }

    // Perform device authorization flow
    return s.performDeviceFlow(ctx, ssoConfig)
}

func (s *SSOAuthenticator) getSSOConfig(ctx context.Context) (*SSOConfig, error) {
    // Try to detect existing SSO configuration
    if cfg := s.detectExistingSSO(); cfg != nil {
        return cfg, nil
    }

    // Interactive setup
    return s.interactiveSSOMSetup(ctx)
}

type SSOConfig struct {
    StartURL  string
    Region    string
    AccountID string
    RoleName  string
}

func (s *SSOAuthenticator) detectExistingSSO() *SSOConfig {
    // Try to read from ~/.aws/config
    // Implementation would parse AWS config file for existing SSO settings
    // This is a simplified version
    return nil
}

func (s *SSOAuthenticator) interactiveSSOMSetup(ctx context.Context) (*SSOConfig, error) {
    fmt.Println("\n📋 AWS SSO Configuration")
    fmt.Println("We'll help you set up AWS Single Sign-On authentication.")
    
    config := &SSOConfig{
        Region: s.region,
    }
    
    // Get SSO start URL
    fmt.Print("\nEnter your organization's AWS SSO start URL: ")
    fmt.Scanln(&config.StartURL)
    
    if config.StartURL == "" {
        return nil, fmt.Errorf("SSO start URL is required")
    }
    
    return config, nil
}

func (s *SSOAuthenticator) performDeviceFlow(ctx context.Context, ssoConfig *SSOConfig) (aws.Config, error) {
    // Load AWS config for the region
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(ssoConfig.Region))
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
    }

    // Create SSOOIDC client for device authorization
    oidcClient := ssooidc.NewFromConfig(cfg)

    // Register the client
    clientCreds, err := oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
        ClientName: aws.String(s.config.ToolName),
        ClientType: aws.String("public"),
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to register SSO client: %w", err)
    }

    // Start device authorization
    deviceAuth, err := oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
        ClientId:     clientCreds.ClientId,
        ClientSecret: clientCreds.ClientSecret,
        StartUrl:     aws.String(ssoConfig.StartURL),
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to start device authorization: %w", err)
    }

    // Display instructions to user
    fmt.Printf("\n🌐 Please complete authentication in your browser\n")
    fmt.Printf("Opening: %s\n", *deviceAuth.VerificationUriComplete)
    fmt.Printf("Verification code: %s\n", *deviceAuth.UserCode)
    fmt.Printf("\nWaiting for authentication...")

    // Open browser
    if err := s.openBrowser(*deviceAuth.VerificationUriComplete); err != nil {
        fmt.Printf("\nCould not open browser automatically. Please visit the URL above.\n")
    }

    // Poll for token
    return s.pollForToken(ctx, oidcClient, clientCreds, deviceAuth, ssoConfig)
}

func (s *SSOAuthenticator) pollForToken(ctx context.Context, oidcClient *ssooidc.Client, clientCreds *ssooidc.RegisterClientOutput, deviceAuth *ssooidc.StartDeviceAuthorizationOutput, ssoConfig *SSOConfig) (aws.Config, error) {
    interval := time.Duration(*deviceAuth.Interval) * time.Second
    timeout := time.Now().Add(time.Duration(*deviceAuth.ExpiresIn) * time.Second)

    for time.Now().Before(timeout) {
        select {
        case <-ctx.Done():
            return aws.Config{}, ctx.Err()
        case <-time.After(interval):
            // Try to get the token
            tokenResp, err := oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
                ClientId:     clientCreds.ClientId,
                ClientSecret: clientCreds.ClientSecret,
                GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
                DeviceCode:   deviceAuth.DeviceCode,
            })
            
            if err != nil {
                // Check if we should continue polling
                if s.shouldContinuePolling(err) {
                    continue
                }
                return aws.Config{}, fmt.Errorf("failed to get token: %w", err)
            }

            fmt.Printf("\n✅ Authentication successful!\n")

            // Get account and role information
            return s.completeSSOMSetup(ctx, tokenResp, ssoConfig)
        }
    }

    return aws.Config{}, fmt.Errorf("authentication timed out")
}

func (s *SSOAuthenticator) shouldContinuePolling(err error) bool {
    // Check for specific errors that indicate we should keep polling
    // This would need to inspect the actual AWS error types
    return true
}

func (s *SSOAuthenticator) completeSSOMSetup(ctx context.Context, token *ssooidc.CreateTokenOutput, ssoConfig *SSOConfig) (aws.Config, error) {
    // Create SSO client with the access token
    cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(ssoConfig.Region))
    if err != nil {
        return aws.Config{}, err
    }

    ssoClient := sso.NewFromConfig(cfg)

    // List available accounts
    accounts, err := ssoClient.ListAccounts(ctx, &sso.ListAccountsInput{
        AccessToken: token.AccessToken,
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to list accounts: %w", err)
    }

    if len(accounts.AccountList) == 0 {
        return aws.Config{}, fmt.Errorf("no AWS accounts available")
    }

    // For simplicity, use the first account
    // In a real implementation, you'd want to let the user choose
    account := accounts.AccountList[0]
    ssoConfig.AccountID = *account.AccountId

    // List roles for the account
    roles, err := ssoClient.ListAccountRoles(ctx, &sso.ListAccountRolesInput{
        AccessToken: token.AccessToken,
        AccountId:   account.AccountId,
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to list roles: %w", err)
    }

    if len(roles.RoleList) == 0 {
        return aws.Config{}, fmt.Errorf("no roles available in account")
    }

    // Use the first available role
    role := roles.RoleList[0]
    ssoConfig.RoleName = *role.RoleName

    // Save SSO configuration to AWS config file
    if err := s.saveSSOMConfig(ssoConfig); err != nil {
        fmt.Printf("Warning: Could not save SSO config: %v\n", err)
    }

    // Get role credentials
    roleCreds, err := ssoClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
        AccessToken: token.AccessToken,
        AccountId:   account.AccountId,
        RoleName:    role.RoleName,
    })
    if err != nil {
        return aws.Config{}, fmt.Errorf("failed to get role credentials: %w", err)
    }

    // Create AWS config with the SSO credentials
    return config.LoadDefaultConfig(ctx,
        config.WithRegion(ssoConfig.Region),
        config.WithCredentialsProvider(aws.NewCredentialsCache(&ssoCredentialsProvider{
            accessKeyID:     *roleCreds.RoleCredentials.AccessKeyId,
            secretAccessKey: *roleCreds.RoleCredentials.SecretAccessKey,
            sessionToken:    *roleCreds.RoleCredentials.SessionToken,
        })),
    )
}

func (s *SSOAuthenticator) saveSSOMConfig(cfg *SSOConfig) error {
    // Implementation would save SSO config to ~/.aws/config
    // This is a placeholder
    return nil
}

func (s *SSOAuthenticator) openBrowser(url string) error {
    var cmd string
    var args []string

    switch runtime.GOOS {
    case "windows":
        cmd = "rundll32"
        args = []string{"url.dll,FileProtocolHandler", url}
    case "darwin":
        cmd = "open"
        args = []string{url}
    default: // "linux", "freebsd", "openbsd", "netbsd"
        cmd = "xdg-open"
        args = []string{url}
    }

    return exec.Command(cmd, args...).Start()
}

type ssoCredentialsProvider struct {
    accessKeyID, secretAccessKey, sessionToken string
}

func (p *ssoCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
    return aws.Credentials{
        AccessKeyID:     p.accessKeyID,
        SecretAccessKey: p.secretAccessKey,
        SessionToken:    p.sessionToken,
    }, nil
}
```

#### pkg/awsauth/setup.go

```go
package awsauth

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
)

// runCLISetup runs command-line interactive setup
func (c *Client) runCLISetup(ctx context.Context) error {
    fmt.Printf("\n⚙️  Setting up AWS authentication for %s\n", c.config.ToolName)
    fmt.Println("This tool needs AWS access to work properly.")
    
    if len(c.config.RequiredActions) > 0 {
        fmt.Printf("\nRequired AWS permissions:\n")
        for _, action := range c.config.RequiredActions {
            fmt.Printf("  • %s\n", action)
        }
    }
    
    fmt.Println("\nAvailable authentication methods:")
    options := []string{}
    
    if c.config.PreferSSO {
        options = append(options, "AWS SSO (recommended for organizations)")
    }
    if c.config.AllowIAMUser {
        options = append(options, "IAM User with access keys")
    }
    if c.config.AllowEnvVars {
        options = append(options, "Use existing AWS profile")
    }
    
    for i, option := range options {
        fmt.Printf("%d. %s\n", i+1, option)
    }
    
    fmt.Print("\nSelect authentication method [1]: ")
    reader := bufio.NewReader(os.Stdin)
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(input)
    
    choice := 1
    if input != "" {
        if c, err := strconv.Atoi(input); err == nil {
            choice = c
        }
    }
    
    switch choice {
    case 1:
        if c.config.PreferSSO {
            return c.setupSSO(ctx)
        } else if c.config.AllowIAMUser {
            return c.setupIAMUser(ctx)
        }
    case 2:
        if c.config.PreferSSO && c.config.AllowIAMUser {
            return c.setupIAMUser(ctx)
        } else if c.config.PreferSSO && c.config.AllowEnvVars {
            return c.setupExistingProfile(ctx)
        }
    case 3:
        return c.setupExistingProfile(ctx)
    }
    
    return fmt.Errorf("invalid choice")
}

func (c *Client) setupSSO(ctx context.Context) error {
    fmt.Println("\n🔐 Setting up AWS SSO")
    
    ssoAuth := NewSSOAuthenticator(c.config)
    cfg, err := ssoAuth.Authenticate(ctx)
    if err != nil {
        return fmt.Errorf("SSO setup failed: %w", err)
    }
    
    // Test the configuration
    if !c.validateCredentials(ctx, cfg) {
        return fmt.Errorf("SSO credentials don't have required permissions")
    }
    
    fmt.Println("✅ AWS SSO setup completed successfully!")
    return nil
}

func (c *Client) setupIAMUser(ctx context.Context) error {
    fmt.Println("\n🔑 Setting up IAM User")
    fmt.Printf("We'll create an IAM user with minimal permissions for %s\n", c.config.ToolName)
    
    // Generate CloudFormation template
    template, err := c.generateIAMTemplate()
    if err != nil {
        return fmt.Errorf("failed to generate CloudFormation template: %w", err)
    }
    
    // Save template
    tempDir := os.TempDir()
    templatePath := filepath.Join(tempDir, fmt.Sprintf("%s-iam-setup.yaml", c.config.ToolName))
    
    if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
        return fmt.Errorf("failed to save template: %w", err)
    }
    
    fmt.Printf("\n📄 CloudFormation template saved to:\n%s\n", templatePath)
    
    fmt.Println("\nNext steps:")
    fmt.Println("1. Open the AWS CloudFormation console in your browser")
    fmt.Println("2. Create a new stack using the template file above")
    fmt.Println("3. After the stack is created, find the Outputs tab")
    fmt.Println("4. Copy the AccessKeyId and SecretAccessKey values")
    fmt.Println("5. Return here to complete the setup")
    
    // Open CloudFormation console
    cfURL := "https://console.aws.amazon.com/cloudformation/home"
    fmt.Printf("\n🌐 Open CloudFormation console? [Y/n]: ")
    
    reader := bufio.NewReader(os.Stdin)
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(strings.ToLower(input))
    
    if input == "" || input == "y" || input == "yes" {
        if err := c.openBrowser(cfURL); err != nil {
            fmt.Printf("Could not open browser. Please visit: %s\n", cfURL)
        }
    }
    
    // Wait for user to complete CloudFormation setup
    fmt.Print("\nPress Enter when you have the access keys ready...")
    reader.ReadString('\n')
    
    return c.promptForCredentials()
}

func (c *Client) setupExistingProfile(ctx context.Context) error {
    fmt.Println("\n📋 Using existing AWS profile")
    
    profiles := c.listAWSProfiles()
    if len(profiles) == 0 {
        fmt.Println("No existing AWS profiles found.")
        fmt.Println("Please run 'aws configure' first or choose a different authentication method.")
        return fmt.Errorf("no AWS profiles found")
    }
    
    fmt.Println("Available AWS profiles:")
    for i, profile := range profiles {
        fmt.Printf("%d. %s\n", i+1, profile)
    }
    
    fmt.Print("Select profile to use [1]: ")
    reader := bufio.NewReader(os.Stdin)
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(input)
    
    choice := 1
    if input != "" {
        if c, err := strconv.Atoi(input); err == nil {
            choice = c
        }
    }
    
    if choice < 1 || choice > len(profiles) {
        return fmt.Errorf("invalid profile selection")
    }
    
    selectedProfile := profiles[choice-1]
    
    // Test the profile
    cfg, err := c.loadProfile(ctx, selectedProfile)
    if err != nil {
        return fmt.Errorf("failed to load profile %s: %w", selectedProfile, err)
    }
    
    if !c.validateCredentials(ctx, cfg) {
        return fmt.Errorf("profile %s doesn't have required permissions", selectedProfile)
    }
    
    // Save as our tool's profile
    if selectedProfile != c.profileName {
        if err := c.copyProfile(selectedProfile, c.profileName); err != nil {
            return fmt.Errorf("failed to copy profile: %w", err)
        }
    }
    
    fmt.Printf("✅ Successfully configured to use profile: %s\n", selectedProfile)
    return nil
}

func (c *Client) generateIAMTemplate() (string, error) {
    permissions := c.buildPermissionStatements()
    
    template := fmt.Sprintf(`AWSTemplateFormatVersion: '2010-09-09'
Description: 'IAM User for %s'

Resources:
  %sUser:
    Type: AWS::IAM::User
    Properties:
      UserName: !Sub '%s-user-${AWS::AccountId}'
      Path: '/external-tools/'
      
  %sAccessKey:
    Type: AWS::IAM::AccessKey
    Properties:
      UserName: !Ref %sUser
      
  %sPolicy:
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
    Value: !Ref %sAccessKey
    
  SecretAccessKey:
    Description: 'Secret Access Key'
    Value: !GetAtt %sAccessKey.SecretAccessKey
    
  SetupInstructions:
    Description: 'Next steps'
    Value: 'Copy the AccessKeyId and SecretAccessKey values and return to your tool setup'
`,
        c.config.ToolName,                    // Description
        c.config.ToolName,                    // User resource name
        c.config.ToolName,                    // UserName
        c.config.ToolName,                    // AccessKey resource name
        c.config.ToolName,                    // User reference
        c.config.ToolName,                    // Policy resource name
        c.config.ToolName,                    // User reference
        c.config.ToolName,                    // Policy name
        permissions,                          // Permission statements
        c.config.ToolName,                    // Output description
        c.config.ToolName,                    // AccessKey reference
        c.config.ToolName,                    // AccessKey reference
    )
    
    return template, nil
}

func (c *Client) buildPermissionStatements() string {
    if len(c.config.CustomPermissions) > 0 {
        return c.formatCustomPermissions()
    }
    
    // Build from required actions
    actions := c.config.RequiredActions
    if len(actions) == 0 {
        actions = []string{"sts:GetCallerIdentity"}
    }
    
    var statements []string
    
    // Group actions by service for better organization
    serviceActions := make(map[string][]string)
    for _, action := range actions {
        parts := strings.SplitN(action, ":", 2)
        if len(parts) == 2 {
            service := parts[0]
            serviceActions[service] = append(serviceActions[service], action)
        }
    }
    
    for service, serviceActionsSlice := range serviceActions {
        statement := fmt.Sprintf(`          - Sid: '%s%sPermissions'
            Effect: Allow
            Action:
%s
            Resource: '*'`,
            c.config.ToolName,
            strings.Title(service),
            c.formatActions(serviceActionsSlice),
        )
        statements = append(statements, statement)
    }
    
    return strings.Join(statements, "\n")
}

func (c *Client) formatActions(actions []string) string {
    var formatted []string
    for _, action := range actions {
        formatted = append(formatted, fmt.Sprintf("              - '%s'", action))
    }
    return strings.Join(formatted, "\n")
}

func (c *Client) formatCustomPermissions() string {
    // Implementation for custom permissions formatting
    return ""
}

func (c *Client) promptForCredentials() error {
    reader := bufio.NewReader(os.Stdin)
    
    fmt.Print("Enter Access Key ID: ")
    accessKey, _ := reader.ReadString('\n')
    accessKey = strings.TrimSpace(accessKey)
    
    fmt.Print("Enter Secret Access Key: ")
    secretKey, _ := reader.ReadString('\n')
    secretKey = strings.TrimSpace(secretKey)
    
    if accessKey == "" || secretKey == "" {
        return fmt.Errorf("access key and secret key are required")
    }
    
    return c.saveCredentials(accessKey, secretKey)
}

func (c *Client) saveCredentials(accessKey, secretKey string) error {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return fmt.Errorf("failed to get home directory: %w", err)
    }
    
    awsDir := filepath.Join(homeDir, ".aws")
    if err := os.MkdirAll(awsDir, 0755); err != nil {
        return fmt.Errorf("failed to create .aws directory: %w", err)
    }
    
    credFile := filepath.Join(awsDir, "credentials")
    
    // Read existing credentials file
    content := ""
    if data, err := os.ReadFile(credFile); err == nil {
        content = string(data)
    }
    
    // Add/update our profile
    profileSection := fmt.Sprintf("\n[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\nregion = %s\n",
        c.profileName, accessKey, secretKey, c.config.DefaultRegion)
    
    // Remove existing profile if it exists
    lines := strings.Split(content, "\n")
    var newLines []string
    inOurProfile := false
    
    for _, line := range lines {
        if strings.TrimSpace(line) == fmt.Sprintf("[%s]", c.profileName) {
            inOurProfile = true
            continue
        }
        if strings.HasPrefix(line, "[") && line != fmt.Sprintf("[%s]", c.profileName) {
            inOurProfile = false
        }
        if !inOurProfile {
            newLines = append(newLines, line)
        }
    }
    
    content = strings.Join(newLines, "\n") + profileSection
    
    // Write back
    if err := os.WriteFile(credFile, []byte(content), 0600); err != nil {
        return fmt.Errorf("failed to save credentials: %w", err)
    }
    
    fmt.Printf("✅ Credentials saved to profile: %s\n", c.profileName)
    return nil
}

func (c *Client) listAWSProfiles() []string {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil
    }
    
    credFile := filepath.Join(homeDir, ".aws", "credentials")
    configFile := filepath.Join(homeDir, ".aws", "config")
    
    profiles := make(map[string]bool)
    
    // Read credentials file
    if data, err := os.ReadFile(credFile); err == nil {
        lines := strings.Split(string(data), "\n")
        for _, line := range lines {
            line = strings.TrimSpace(line)
            if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
                profile := strings.Trim(line, "[]")
                if profile != "" {
                    profiles[profile] = true
                }
            }
        }
    }
    
    // Read config file
    if data, err := os.ReadFile(configFile); err == nil {
        lines := strings.Split(string(data), "\n")
        for _, line := range lines {
            line = strings.TrimSpace(line)
            if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
                profile := strings.TrimPrefix(strings.Trim(line, "[]"), "profile ")
                if profile != "" {
                    profiles[profile] = true
                }
            }
        }
    }
    
    var result []string
    for profile := range profiles {
        result = append(result, profile)
    }
    
    return result
}

func (c *Client) copyProfile(source, dest string) error {
    // Implementation to copy AWS profile configuration
    // This would copy from ~/.aws/credentials and ~/.aws/config
    return nil
}

func (c *Client) openBrowser(url string) error {
    // Implementation depends on OS - same as in SSO code
    return nil
}
```

### Usage Examples

#### Command Line Tool

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    
    "github.com/aws/aws-sdk-go-v2/service/ec2"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    awsauth "github.com/your-org/aws-external-tool-integration/pkg/awsauth"
)

func main() {
    var (
        setupFlag   = flag.Bool("setup", false, "Run AWS authentication setup")
        profileFlag = flag.String("profile", "", "AWS profile to use")
        regionFlag  = flag.String("region", "us-east-1", "AWS region")
    )
    flag.Parse()
    
    // Create config
    config := awsauth.DefaultConfig("my-cloud-tool")
    config.RequiredActions = []string{
        "ec2:DescribeInstances",
        "ec2:StartInstances", 
        "ec2:StopInstances",
        "s3:ListBuckets",
        "s3:GetObject",
    }
    config.DefaultRegion = *regionFlag
    
    if *profileFlag != "" {
        config.ProfileName = *profileFlag
    }
    
    // Initialize auth client
    client, err := awsauth.New(config)
    if err != nil {
        log.Fatal("Failed to initialize AWS auth:", err)
    }
    
    // Handle setup command
    if *setupFlag {
        if err := client.RunSetup(context.Background()); err != nil {
            log.Fatal("Setup failed:", err)
        }
        fmt.Println("✅ AWS authentication setup completed!")
        return
    }
    
    // Get AWS config
    ctx := context.Background()
    awsConfig, err := client.GetAWSConfig(ctx)
    if err != nil {
        fmt.Printf("❌ AWS authentication required: %v\n", err)
        fmt.Println("Run with --setup to configure authentication")
        os.Exit(1)
    }
    
    // Parse command
    if len(flag.Args()) == 0 {
        fmt.Println("Usage: my-cloud-tool <command> [args]")
        fmt.Println("Commands: list-instances, start-instance, list-buckets")
        os.Exit(1)
    }
    
    command := flag.Arg(0)
    
    switch command {
    case "list-instances":
        listInstances(ctx, awsConfig)
    case "start-instance":
        if len(flag.Args()) < 2 {
            fmt.Println("Usage: my-cloud-tool start-instance <instance-id>")
            os.Exit(1)
        }
        startInstance(ctx, awsConfig, flag.Arg(1))
    case "list-buckets":
        listBuckets(ctx, awsConfig)
    default:
        fmt.Printf("Unknown command: %s\n", command)
        os.Exit(1)
    }
}

func listInstances(ctx context.Context, cfg aws.Config) {
    ec2Client := ec2.NewFromConfig(cfg)
    
    result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
    if err != nil {
        log.Fatal("Failed to list instances:", err)
    }
    
    fmt.Println("EC2 Instances:")
    for _, reservation := range result.Reservations {
        for _, instance := range reservation.Instances {
            name := getInstanceName(instance.Tags)
            fmt.Printf("  %s (%s) - %s\n", 
                *instance.InstanceId, 
                name, 
                instance.State.Name)
        }
    }
}

func startInstance(ctx context.Context, cfg aws.Config, instanceID string) {
    ec2Client := ec2.NewFromConfig(cfg)
    
    _, err := ec2Client.StartInstances(ctx, &ec2.StartInstancesInput{
        InstanceIds: []string{instanceID},
    })
    if err != nil {
        log.Fatal("Failed to start instance:", err)
    }
    
    fmt.Printf("✅ Started instance %s\n", instanceID)
}

func listBuckets(ctx context.Context, cfg aws.Config) {
    s3Client := s3.NewFromConfig(cfg)
    
    result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
    if err != nil {
        log.Fatal("Failed to list buckets:", err)
    }
    
    fmt.Println("S3 Buckets:")
    for _, bucket := range result.Buckets {
        fmt.Printf("  %s (created: %s)\n", 
            *bucket.Name, 
            bucket.CreationDate.Format("2006-01-02"))
    }
}

func getInstanceName(tags []types.Tag) string {
    for _, tag := range tags {
        if *tag.Key == "Name" {
            return *tag.Value
        }
    }
    return "unnamed"
}
```

#### Desktop Application (using Fyne)

```go
package main

import (
    "context"
    "log"
    
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    awsauth "github.com/your-org/aws-external-tool-integration/pkg/awsauth"
)

type App struct {
    awsClient *awsauth.Client
    window    fyne.Window
    content   *container.VBox
}

func main() {
    myApp := app.New()
    myApp.SetIcon(resourceIconPng) // Your app icon
    
    window := myApp.NewWindow("AWS Desktop Manager")
    window.Resize(fyne.NewSize(800, 600))
    
    app := &App{
        window:  window,
        content: container.NewVBox(),
    }
    
    // Initialize AWS auth
    config := awsauth.DefaultConfig("aws-desktop-manager")
    config.RequiredActions = []string{
        "ec2:DescribeInstances",
        "s3:ListBuckets",
        "cloudwatch:GetMetricStatistics",
    }
    config.SetupUI = true // Enable web UI for setup
    config.BrandingOptions = map[string]string{
        "primary_color": "#ff6b35",
        "company_name":  "Your Company",
    }
    
    client, err := awsauth.New(config)
    if err != nil {
        log.Fatal("Failed to initialize AWS auth:", err)
    }
    
    app.awsClient = client
    
    // Check AWS configuration
    app.checkAWSConfig()
    
    window.SetContent(app.content)
    window.ShowAndRun()
}

func (a *App) checkAWSConfig() {
    ctx := context.Background()
    _, err := a.awsClient.GetAWSConfig(ctx)
    
    if err != nil {
        a.showSetupRequired()
    } else {
        a.showMainInterface()
    }
}

func (a *App) showSetupRequired() {
    title := widget.NewLabel("AWS Configuration Required")
    title.TextStyle.Bold = true
    
    description := widget.NewLabel(
        "This application needs access to your AWS account to function.\n" +
        "Click the button below to set up secure authentication.")
    
    setupBtn := widget.NewButton("Configure AWS Access", func() {
        go func() {
            if err := a.awsClient.RunSetup(context.Background()); err != nil {
                // Show error dialog
                return
            }
            // Refresh the interface
            a.checkAWSConfig()
        }()
    })
    
    a.content.Objects = []fyne.CanvasObject{
        container.NewVBox(
            title,
            widget.NewSeparator(),
            description,
            setupBtn,
        ),
    }
    a.content.Refresh()
}

func (a *App) showMainInterface() {
    // Main application interface
    title := widget.NewLabel("AWS Desktop Manager")
    title.TextStyle.Bold = true
    
    // Create tabs for different AWS services
    tabs := container.NewAppTabs(
        container.NewTabItem("EC2 Instances", a.createEC2Tab()),
        container.NewTabItem("S3 Buckets", a.createS3Tab()),
        container.NewTabItem("CloudWatch", a.createCloudWatchTab()),
    )
    
    // Add refresh button
    refreshBtn := widget.NewButton("Refresh", func() {
        a.refreshData()
    })
    
    a.content.Objects = []fyne.CanvasObject{
        container.NewVBox(
            title,
            refreshBtn,
            widget.NewSeparator(),
            tabs,
        ),
    }
    a.content.Refresh()
}

func (a *App) createEC2Tab() fyne.CanvasObject {
    // Implementation for EC2 instance management
    return widget.NewLabel("EC2 Instances will be listed here")
}

func (a *App) createS3Tab() fyne.CanvasObject {
    // Implementation for S3 bucket management
    return widget.NewLabel("S3 Buckets will be listed here")
}

func (a *App) createCloudWatchTab() fyne.CanvasObject {
    // Implementation for CloudWatch metrics
    return widget.NewLabel("CloudWatch metrics will be shown here")
}

func (a *App) refreshData() {
    // Refresh all data from AWS
}
```

#### CI/CD Runner

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    awsauth "github.com/your-org/aws-external-tool-integration/pkg/awsauth"
    "github.com/aws/aws-sdk-go-v2/service/ecs"
)

func main() {
    // CI/CD optimized configuration
    config := awsauth.CICDConfig("deployment-runner")
    config.RequiredActions = []string{
        "ecs:UpdateService",
        "ecs:DescribeServices",
        "ecs:DescribeTaskDefinition",
        "ecs:RegisterTaskDefinition",
    }
    
    client, err := awsauth.New(config)
    if err != nil {
        fmt.Printf("❌ Failed to initialize AWS auth: %v\n", err)
        os.Exit(1)
    }
    
    ctx := context.Background()
    
    // In CI/CD, fail fast if credentials aren't available
    awsConfig, err := client.GetAWSConfig(ctx)
    if err != nil {
        fmt.Printf("❌ AWS credentials not configured\n")
        fmt.Println("For CI/CD environments, ensure one of:")
        fmt.Println("- AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables")
        fmt.Println("- IAM instance role (if running on EC2)")
        fmt.Println("- AWS_PROFILE environment variable")
        os.Exit(1)
    }
    
    // Deployment logic
    serviceName := os.Getenv("ECS_SERVICE_NAME")
    clusterName := os.Getenv("ECS_CLUSTER_NAME")
    
    if serviceName == "" || clusterName == "" {
        fmt.Println("❌ ECS_SERVICE_NAME and ECS_CLUSTER_NAME required")
        os.Exit(1)
    }
    
    fmt.Printf("🚀 Deploying service %s in cluster %s\n", serviceName, clusterName)
    
    ecsClient := ecs.NewFromConfig(awsConfig)
    
    // Update ECS service
    _, err = ecsClient.UpdateService(ctx, &ecs.UpdateServiceInput{
        Cluster:            &clusterName,
        Service:            &serviceName,
        ForceNewDeployment: aws.Bool(true),
    })
    if err != nil {
        fmt.Printf("❌ Deployment failed: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println("✅ Deployment initiated successfully")
}
```

### Configuration Examples

#### config.yaml

```yaml
# Basic CLI tool configuration
tool_name: "my-awesome-tool"
tool_version: "1.0.0"
default_region: "us-west-2"
profile_name: "my-tool-profile"
session_duration: "12h"

# Required AWS permissions
required_actions:
  - "ec2:DescribeInstances"
  - "ec2:StartInstances"
  - "ec2:StopInstances"
  - "s3:ListBuckets"
  - "s3:GetObject"

# Authentication preferences
prefer_sso: true
allow_iam_user: true
allow_env_vars: true

# Setup customization
setup_ui: true
branding_options:
  primary_color: "#2196F3"
  company_name: "Awesome Corp"
  support_email: "support@awesome.com"
  logo_url: "https://awesome.com/logo.png"

# Custom permissions (alternative to required_actions)
custom_permissions:
  - sid: "EC2Management"
    effect: "Allow"
    actions:
      - "ec2:DescribeInstances"
      - "ec2:StartInstances"
      - "ec2:StopInstances"
    resources: ["*"]
    condition:
      StringEquals:
        "ec2:Region": ["us-west-2", "us-east-1"]
```

#### CI/CD Configuration

```yaml
# Optimized for CI/CD environments
tool_name: "ci-deployment-tool"
default_region: "us-east-1"
session_duration: "1h"

required_actions:
  - "ecs:UpdateService"
  - "ecs:DescribeServices"
  - "ecr:GetAuthorizationToken"
  - "ecr:BatchCheckLayerAvailability"

# CI/CD specific settings
prefer_sso: false      # SSO doesn't work in CI
allow_iam_user: true   # Allow access keys
allow_env_vars: true   # Prefer environment variables
setup_ui: false        # No interactive setup
ci_mode: true          # Optimize for CI/CD
```

## Security Best Practices

### 1. Credential Storage
- Store credentials in `~/.aws/credentials` with 0600 permissions
- Use temporary credentials when possible (SSO)
- Implement credential caching with expiration

### 2. Permission Management
- Request only required permissions
- Generate CloudFormation templates for minimal IAM policies
- Provide clear explanations of why each permission is needed

### 3. Multi-Environment Support
- Different configs for development, staging, production
- Support for multiple AWS profiles
- Environment variable overrides for CI/CD

### 4. User Experience
- Progressive disclosure of complexity
- Clear error messages with actionable guidance
- Fallback options when primary method fails

## Integration Patterns

### Framework Integration
```go
// Gin middleware
func AWSAuthMiddleware(client *awsauth.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        awsConfig, err := client.GetAWSConfig(ctx)
        if err != nil {
            c.JSON(401, gin.H{"error": "AWS authentication required"})
            c.Abort()
            return
        }
        
        c.Set("aws_config", awsConfig)
        c.Next()
    }
}
```

### Plugin Architecture
```go
// Plugin interface for different authentication methods
type AuthPlugin interface {
    Name() string
    Available(ctx context.Context) bool
    Authenticate(ctx context.Context) (aws.Config, error)
}

// Register custom authentication plugins
client.RegisterPlugin(&CustomSAMLPlugin{})
client.RegisterPlugin(&CustomOIDCPlugin{})
```

This external tool pattern provides the same excellent UX as Coiled's approach while being perfectly suited for tools that run outside AWS on user workstations and in CI/CD environments.
    