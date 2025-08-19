package awsauth

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Client provides AWS authentication for external tools
// Handles the complexity of credential discovery and setup
type Client struct {
	config      *Config
	profileName string
	credCache   *CredentialCache
	setupUI     *SetupUI
}

// New creates a new AWS auth client for external tools
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

// Option allows customization of the client
type Option func(*Client)

// WithProfileName sets a custom AWS profile name
func WithProfileName(name string) Option {
	return func(c *Client) { c.profileName = name }
}

// WithCredentialCache sets a custom credential cache
func WithCredentialCache(cache *CredentialCache) Option {
	return func(c *Client) { c.credCache = cache }
}

// GetAWSConfig returns AWS config, handling all authentication complexity
// This is the main entry point - it tries cached credentials first,
// then existing AWS profiles, then guides user through setup if needed
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

	// Need setup - guide user through authentication
	fmt.Printf("🔐 AWS authentication required for %s\n", c.config.ToolName)
	fmt.Println("Let's get you set up securely!")
	
	if c.config.CIMode {
		return aws.Config{}, fmt.Errorf("no AWS credentials found and running in CI mode (no interactive setup)")
	}

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

// loadProfile loads a specific AWS profile
func (c *Client) loadProfile(ctx context.Context, profileName string) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profileName),
		config.WithRegion(c.config.DefaultRegion),
	)
}

// validateCredentials tests if credentials work and have required permissions
func (c *Client) validateCredentials(ctx context.Context, cfg aws.Config) bool {
	stsClient := sts.NewFromConfig(cfg)

	// Test basic access
	if _, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err != nil {
		return false
	}

	// Validate required permissions (simplified check)
	return c.validatePermissions(ctx, cfg)
}

// validatePermissions checks if credentials have required permissions
func (c *Client) validatePermissions(ctx context.Context, cfg aws.Config) bool {
	// For now, just check if we can call GetCallerIdentity
	// In a real implementation, you'd test specific required actions
	for _, action := range c.config.RequiredActions {
		if !c.testAction(ctx, cfg, action) {
			return false
		}
	}
	return true
}

// testAction tests if a specific AWS action is allowed
func (c *Client) testAction(ctx context.Context, cfg aws.Config, action string) bool {
	// This is a simplified implementation
	// Real implementation would use AWS IAM simulator or try actual calls
	switch action {
	case "sts:GetCallerIdentity":
		stsClient := sts.NewFromConfig(cfg)
		_, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		return err == nil
	default:
		// For now, assume other actions are valid if STS works
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

// runSetup handles the setup process
func (c *Client) runSetup(ctx context.Context) (aws.Config, error) {
	if err := c.RunSetup(ctx); err != nil {
		return aws.Config{}, fmt.Errorf("setup failed: %w", err)
	}

	// After setup, try to load credentials again
	return c.tryExistingCredentials(ctx)
}

// cacheCredentials stores credentials in cache
func (c *Client) cacheCredentials(cfg aws.Config) {
	c.credCache.Set(c.profileName, &CachedCredentials{
		AWSConfig: cfg,
		ExpiresAt: time.Now().Add(c.config.SessionDuration),
	})
}

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

	// For now, just return an error indicating setup is needed
	// Full implementation would handle the interactive flow
	return fmt.Errorf("interactive setup not yet implemented - please configure AWS credentials manually")
}