// Package auth provides authentication management for the CLI tool
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/ui"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/awsauth"
)

// Manager handles authentication operations
type Manager struct {
	config    *config.Config
	awsClient *awsauth.Client
}

// SetupConfig represents setup configuration options
type SetupConfig struct {
	Method      string
	Region      string
	Interactive bool
}

// NewManager creates a new authentication manager
func NewManager(cfg *config.Config) (*Manager, error) {
	return &Manager{
		config: cfg,
	}, nil
}

// IsConfigured returns true if authentication is configured
func (m *Manager) IsConfigured() bool {
	return m.config.IsAuthConfigured()
}

// Setup configures authentication using the specified method
func (m *Manager) Setup(ctx context.Context, setupConfig *SetupConfig, uiHandler *ui.Handler) error {
	logger := slog.Default()
	
	logger.Debug("Setting up authentication",
		slog.String("method", setupConfig.Method),
		slog.String("region", setupConfig.Region))

	// Create awsauth configuration
	authConfig := &awsauth.Config{
		ToolName:        "DataTool CLI",
		ToolVersion:     "1.0.0",
		DefaultRegion:   m.getRegion(setupConfig.Region),
		SessionDuration: time.Duration(m.config.Auth.SessionDuration) * time.Second,
		CIMode:          !setupConfig.Interactive,
	}

	// Configure based on authentication method
	switch setupConfig.Method {
	case "sso":
		if err := m.setupSSO(ctx, setupConfig, authConfig, uiHandler); err != nil {
			return fmt.Errorf("SSO setup failed: %w", err)
		}
	case "profile":
		if err := m.setupProfile(ctx, setupConfig, authConfig, uiHandler); err != nil {
			return fmt.Errorf("profile setup failed: %w", err)
		}
	case "interactive":
		if err := m.setupInteractive(ctx, setupConfig, authConfig, uiHandler); err != nil {
			return fmt.Errorf("interactive setup failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported authentication method: %s", setupConfig.Method)
	}

	// Create AWS auth client
	client, err := awsauth.New(authConfig)
	if err != nil {
		return fmt.Errorf("failed to create AWS auth client: %w", err)
	}

	m.awsClient = client

	// Update configuration
	m.config.Auth.Method = setupConfig.Method
	m.config.Auth.Region = authConfig.DefaultRegion
	m.config.Auth.SessionDuration = int(authConfig.SessionDuration.Seconds())

	return nil
}

// TestAuthentication tests the configured authentication
func (m *Manager) TestAuthentication(ctx context.Context) error {
	if m.awsClient == nil {
		return fmt.Errorf("authentication not configured")
	}

	// Get AWS configuration
	awsConfig, err := m.awsClient.GetAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS configuration: %w", err)
	}

	// Test with STS GetCallerIdentity
	stsClient := sts.NewFromConfig(awsConfig)
	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	slog.Default().Info("Authentication test successful",
		slog.String("user_id", aws.ToString(result.UserId)),
		slog.String("account", aws.ToString(result.Account)),
		slog.String("arn", aws.ToString(result.Arn)))

	return nil
}

// GetAWSConfig returns the AWS configuration for making AWS API calls
func (m *Manager) GetAWSConfig(ctx context.Context) (aws.Config, error) {
	if m.awsClient == nil {
		return aws.Config{}, fmt.Errorf("authentication not configured")
	}

	return m.awsClient.GetAWSConfig(ctx)
}

// GetStatus returns the current authentication status
func (m *Manager) GetStatus(ctx context.Context) (*AuthStatus, error) {
	status := &AuthStatus{
		Configured: m.IsConfigured(),
		Method:     m.config.Auth.Method,
		Region:     m.config.Auth.Region,
	}

	if !status.Configured {
		return status, nil
	}

	// Try to get current identity
	if awsConfig, err := m.GetAWSConfig(ctx); err == nil {
		stsClient := sts.NewFromConfig(awsConfig)
		if result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err == nil {
			status.Identity = &Identity{
				UserID:  aws.ToString(result.UserId),
				Account: aws.ToString(result.Account),
				ARN:     aws.ToString(result.Arn),
			}
			status.Active = true
		}
	}

	return status, nil
}

// Refresh forces a refresh of cached credentials
func (m *Manager) Refresh(ctx context.Context) error {
	if m.awsClient == nil {
		return fmt.Errorf("authentication not configured")
	}

	// Force refresh by running setup again
	return m.awsClient.RunSetup(ctx)
}

// setupSSO configures AWS SSO authentication
func (m *Manager) setupSSO(ctx context.Context, setupConfig *SetupConfig, authConfig *awsauth.Config, uiHandler *ui.Handler) error {
	authConfig.PreferSSO = true

	if setupConfig.Interactive {
		// Get SSO configuration from user
		startURL, err := uiHandler.Prompt("SSO Start URL", "")
		if err != nil {
			return err
		}

		region, err := uiHandler.Prompt("SSO Region", setupConfig.Region)
		if err != nil {
			return err
		}

		// Update config
		m.config.Auth.SSO.StartURL = startURL
		m.config.Auth.SSO.Region = region
	}

	return nil
}

// setupProfile configures AWS profile authentication
func (m *Manager) setupProfile(ctx context.Context, setupConfig *SetupConfig, authConfig *awsauth.Config, uiHandler *ui.Handler) error {
	// Detect available profiles
	detector := NewConfigDetector()
	profiles, err := detector.DetectProfiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS profiles: %w", err)
	}

	if len(profiles) == 0 {
		return fmt.Errorf("no AWS profiles found. Please run 'aws configure' first")
	}

	var selectedProfile string
	if setupConfig.Interactive && len(profiles) > 1 {
		// Let user select from available profiles
		options := make([]ui.SelectOption, len(profiles))
		for i, profile := range profiles {
			options[i] = ui.SelectOption{
				Value:       profile,
				Label:       profile,
				Description: "AWS profile from ~/.aws/credentials",
			}
		}

		selectedProfile, err = uiHandler.Select("Select AWS profile:", options)
		if err != nil {
			return err
		}
	} else {
		selectedProfile = profiles[0]
	}

	// Update configuration
	m.config.Auth.Profile.Name = selectedProfile
	m.config.AWSProfile = selectedProfile

	return nil
}

// setupInteractive configures interactive authentication
func (m *Manager) setupInteractive(ctx context.Context, setupConfig *SetupConfig, authConfig *awsauth.Config, uiHandler *ui.Handler) error {
	// Interactive setup will guide the user through the authentication process
	// This typically involves AWS SSO device flow or other interactive methods
	authConfig.PreferSSO = true
	
	if setupConfig.Interactive {
		uiHandler.ShowInfo("Interactive authentication will open your web browser for AWS login.")
		if !uiHandler.Confirm("Continue with interactive authentication?") {
			return fmt.Errorf("interactive authentication cancelled")
		}
	}

	return nil
}

// getRegion returns the region to use, with fallback logic
func (m *Manager) getRegion(region string) string {
	if region != "" {
		return region
	}
	return m.config.GetAWSRegion()
}

// AuthStatus represents the current authentication status
type AuthStatus struct {
	Configured bool      `json:"configured"`
	Active     bool      `json:"active"`
	Method     string    `json:"method"`
	Region     string    `json:"region"`
	Identity   *Identity `json:"identity,omitempty"`
}

// Identity represents AWS identity information
type Identity struct {
	UserID  string `json:"user_id"`
	Account string `json:"account"`
	ARN     string `json:"arn"`
}