package awsauth

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

// SSOAuthenticator handles AWS SSO authentication using device flow
type SSOAuthenticator struct {
	config   *Config
	startURL string
	region   string
}

// SSOConfig holds AWS SSO configuration
type SSOConfig struct {
	StartURL  string `yaml:"start_url"`
	Region    string `yaml:"region"`
	AccountID string `yaml:"account_id"`
	RoleName  string `yaml:"role_name"`
}

// NewSSOAuthenticator creates a new SSO authenticator
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

// getSSOConfig gets SSO configuration from user or existing setup
func (s *SSOAuthenticator) getSSOConfig(ctx context.Context) (*SSOConfig, error) {
	// Try to detect existing SSO configuration
	if cfg := s.detectExistingSSO(); cfg != nil {
		return cfg, nil
	}

	// Interactive setup
	return s.interactiveSSOSetup(ctx)
}

// detectExistingSSO tries to find existing SSO configuration
func (s *SSOAuthenticator) detectExistingSSO() *SSOConfig {
	// Try to read from ~/.aws/config
	// This is a simplified implementation - real version would parse AWS config
	return nil
}

// interactiveSSOSetup guides user through SSO setup
func (s *SSOAuthenticator) interactiveSSOSetup(ctx context.Context) (*SSOConfig, error) {
	fmt.Println("\n📋 AWS SSO Configuration")
	fmt.Println("We'll help you set up AWS Single Sign-On authentication.")

	config := &SSOConfig{
		Region: s.region,
	}

	// Get SSO start URL from user
	fmt.Print("\nEnter your organization's AWS SSO start URL: ")
	var startURL string
	fmt.Scanln(&startURL)

	if startURL == "" {
		return nil, fmt.Errorf("SSO start URL is required")
	}

	// Validate URL format
	if _, err := url.Parse(startURL); err != nil {
		return nil, fmt.Errorf("invalid SSO start URL: %w", err)
	}

	config.StartURL = startURL
	return config, nil
}

// performDeviceFlow executes the AWS SSO device authorization flow
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
	fmt.Printf("Opening: %s\n", aws.ToString(deviceAuth.VerificationUriComplete))
	fmt.Printf("Verification code: %s\n", aws.ToString(deviceAuth.UserCode))
	fmt.Printf("\nWaiting for authentication...\n")

	// Open browser
	if err := s.openBrowser(aws.ToString(deviceAuth.VerificationUriComplete)); err != nil {
		fmt.Printf("\nCould not open browser automatically. Please visit the URL above.\n")
	}

	// Poll for token
	return s.pollForToken(ctx, oidcClient, clientCreds, deviceAuth, ssoConfig)
}

// pollForToken polls for the authentication token
func (s *SSOAuthenticator) pollForToken(ctx context.Context, oidcClient *ssooidc.Client, clientCreds *ssooidc.RegisterClientOutput, deviceAuth *ssooidc.StartDeviceAuthorizationOutput, ssoConfig *SSOConfig) (aws.Config, error) {
	interval := time.Duration(deviceAuth.Interval) * time.Second
	timeout := time.Now().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second)

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
			return s.completeSSOSetup(ctx, tokenResp, ssoConfig)
		}
	}

	return aws.Config{}, fmt.Errorf("authentication timed out")
}

// shouldContinuePolling determines if we should continue polling for the token
func (s *SSOAuthenticator) shouldContinuePolling(err error) bool {
	// In a real implementation, we'd check for specific error types
	// indicating authorization is still pending vs actual failures
	return true
}

// completeSSOSetup finishes SSO setup by getting role credentials
func (s *SSOAuthenticator) completeSSOSetup(ctx context.Context, token *ssooidc.CreateTokenOutput, ssoConfig *SSOConfig) (aws.Config, error) {
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
	// In a real implementation, you'd let the user choose
	account := accounts.AccountList[0]
	ssoConfig.AccountID = aws.ToString(account.AccountId)

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
	ssoConfig.RoleName = aws.ToString(role.RoleName)

	fmt.Printf("Using account: %s (%s)\n", aws.ToString(account.AccountName), aws.ToString(account.AccountId))
	fmt.Printf("Using role: %s\n", aws.ToString(role.RoleName))

	// Save SSO configuration to AWS config file
	if err := s.saveSSOConfig(ssoConfig); err != nil {
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
			accessKeyID:     aws.ToString(roleCreds.RoleCredentials.AccessKeyId),
			secretAccessKey: aws.ToString(roleCreds.RoleCredentials.SecretAccessKey),
			sessionToken:    aws.ToString(roleCreds.RoleCredentials.SessionToken),
		})),
	)
}

// saveSSOConfig saves SSO configuration to AWS config file
func (s *SSOAuthenticator) saveSSOConfig(cfg *SSOConfig) error {
	// Implementation would save SSO config to ~/.aws/config
	// This is a placeholder for now
	return nil
}

// openBrowser opens the default browser to the verification URL
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

// ssoCredentialsProvider implements aws.CredentialsProvider for SSO credentials
type ssoCredentialsProvider struct {
	accessKeyID, secretAccessKey, sessionToken string
}

// Retrieve implements the aws.CredentialsProvider interface
func (p *ssoCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     p.accessKeyID,
		SecretAccessKey: p.secretAccessKey,
		SessionToken:    p.sessionToken,
	}, nil
}