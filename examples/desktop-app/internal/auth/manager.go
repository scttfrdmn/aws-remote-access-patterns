// Package auth provides authentication management for the desktop application
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/desktop-app/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/awsauth"
)

// Manager handles authentication operations for the desktop app
type Manager struct {
	config    *config.Config
	awsClient *awsauth.Client
	status    *AuthStatus
}

// AuthStatus represents the current authentication status
type AuthStatus struct {
	Configured    bool      `json:"configured"`
	Active        bool      `json:"active"`
	Method        string    `json:"method"`
	Region        string    `json:"region"`
	Identity      *Identity `json:"identity,omitempty"`
	LastRefresh   time.Time `json:"last_refresh"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	Error         string    `json:"error,omitempty"`
	RefreshNeeded bool      `json:"refresh_needed"`
}

// Identity represents AWS identity information
type Identity struct {
	UserID  string `json:"user_id"`
	Account string `json:"account"`
	ARN     string `json:"arn"`
	Type    string `json:"type"` // IAMUser, AssumedRole, etc.
}

// SetupRequest represents a setup request from the UI
type SetupRequest struct {
	Method     string `json:"method"`
	Region     string `json:"region,omitempty"`
	StartURL   string `json:"start_url,omitempty"`   // For SSO
	ProfileName string `json:"profile_name,omitempty"` // For profile method
}

// NewManager creates a new authentication manager
func NewManager(cfg *config.Config) (*Manager, error) {
	manager := &Manager{
		config: cfg,
		status: &AuthStatus{
			Configured: cfg.IsAuthConfigured(),
			Method:     cfg.Auth.Method,
			Region:     cfg.GetAWSRegion(),
		},
	}

	// Initialize AWS client if configured
	if cfg.IsAuthConfigured() {
		if err := manager.initializeAWSClient(); err != nil {
			slog.Warn("Failed to initialize AWS client", slog.String("error", err.Error()))
			manager.status.Error = err.Error()
		}
	}

	return manager, nil
}

// GetStatus returns the current authentication status
func (m *Manager) GetStatus(ctx context.Context) *AuthStatus {
	// Update status with fresh information
	if m.awsClient != nil {
		m.updateStatus(ctx)
	}
	
	return m.status
}

// Setup configures authentication using the provided setup request
func (m *Manager) Setup(ctx context.Context, req *SetupRequest) error {
	slog.Info("Setting up authentication",
		slog.String("method", req.Method),
		slog.String("region", req.Region))

	// Update configuration
	m.config.Auth.Method = req.Method
	if req.Region != "" {
		m.config.Auth.Region = req.Region
		m.config.AWSRegion = req.Region
	}

	// Method-specific configuration
	switch req.Method {
	case "sso":
		if req.StartURL == "" {
			return fmt.Errorf("SSO start URL is required")
		}
		m.config.Auth.SSO.StartURL = req.StartURL
		m.config.Auth.SSO.Region = req.Region
	case "profile":
		if req.ProfileName == "" {
			return fmt.Errorf("profile name is required")
		}
		m.config.Auth.Profile.Name = req.ProfileName
	case "interactive":
		// No additional configuration needed
	default:
		return fmt.Errorf("unsupported authentication method: %s", req.Method)
	}

	// Save configuration
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Initialize AWS client
	if err := m.initializeAWSClient(); err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Test authentication
	if err := m.TestAuthentication(ctx); err != nil {
		return fmt.Errorf("authentication test failed: %w", err)
	}

	// Update status
	m.status.Configured = true
	m.status.Method = req.Method
	m.status.Region = m.config.GetAWSRegion()
	m.status.Error = ""

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

	// Update identity information
	m.status.Identity = &Identity{
		UserID:  aws.ToString(result.UserId),
		Account: aws.ToString(result.Account),
		ARN:     aws.ToString(result.Arn),
		Type:    getIdentityType(aws.ToString(result.Arn)),
	}

	m.status.Active = true
	m.status.LastRefresh = time.Now()
	m.status.Error = ""

	slog.Info("Authentication test successful",
		slog.String("user_id", m.status.Identity.UserID),
		slog.String("account", m.status.Identity.Account),
		slog.String("arn", m.status.Identity.ARN))

	return nil
}

// GetAWSConfig returns the AWS configuration for making AWS API calls
func (m *Manager) GetAWSConfig(ctx context.Context) (aws.Config, error) {
	if m.awsClient == nil {
		return aws.Config{}, fmt.Errorf("authentication not configured")
	}

	return m.awsClient.GetAWSConfig(ctx)
}

// Refresh forces a refresh of cached credentials
func (m *Manager) Refresh(ctx context.Context) error {
	if m.awsClient == nil {
		return fmt.Errorf("authentication not configured")
	}

	// Force refresh by running setup again
	if err := m.awsClient.RunSetup(ctx); err != nil {
		m.status.Error = err.Error()
		return err
	}

	// Update status
	m.status.LastRefresh = time.Now()
	m.status.RefreshNeeded = false
	m.status.Error = ""

	return nil
}

// Clear clears the authentication configuration
func (m *Manager) Clear() error {
	// Reset auth configuration
	m.config.Auth = config.AuthConfig{
		SessionDuration: 3600,
		CacheEnabled:    true,
		AutoRefresh:     true,
	}

	// Save configuration
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Clear AWS client
	m.awsClient = nil

	// Reset status
	m.status = &AuthStatus{
		Configured: false,
		Active:     false,
		Method:     "",
		Region:     m.config.GetAWSRegion(),
	}

	return nil
}

// IsConfigured returns true if authentication is configured
func (m *Manager) IsConfigured() bool {
	return m.config.IsAuthConfigured()
}

// initializeAWSClient initializes the AWS authentication client
func (m *Manager) initializeAWSClient() error {
	// Create awsauth configuration
	authConfig := &awsauth.Config{
		ToolName:        "AWS Desktop App",
		ToolVersion:     "1.0.0",
		DefaultRegion:   m.config.GetAWSRegion(),
		SessionDuration: time.Duration(m.config.Auth.SessionDuration) * time.Second,
		PreferSSO:       m.config.Auth.Method == "sso",
		SetupUI:         true, // Enable web UI for desktop app
	}

	// Create AWS auth client
	client, err := awsauth.New(authConfig)
	if err != nil {
		return fmt.Errorf("failed to create AWS auth client: %w", err)
	}

	m.awsClient = client
	return nil
}

// updateStatus updates the authentication status with current information
func (m *Manager) updateStatus(ctx context.Context) {
	// Check if refresh is needed
	if m.status.LastRefresh.IsZero() || time.Since(m.status.LastRefresh) > 5*time.Minute {
		// Try to get current identity without forcing authentication
		if awsConfig, err := m.awsClient.GetAWSConfig(ctx); err == nil {
			stsClient := sts.NewFromConfig(awsConfig)
			if result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}); err == nil {
				m.status.Identity = &Identity{
					UserID:  aws.ToString(result.UserId),
					Account: aws.ToString(result.Account),
					ARN:     aws.ToString(result.Arn),
					Type:    getIdentityType(aws.ToString(result.Arn)),
				}
				m.status.Active = true
				m.status.LastRefresh = time.Now()
				m.status.Error = ""
			} else {
				m.status.Active = false
				m.status.RefreshNeeded = true
				m.status.Error = err.Error()
			}
		}
	}
}

// getIdentityType determines the type of AWS identity from the ARN
func getIdentityType(arn string) string {
	if arn == "" {
		return "Unknown"
	}

	if contains(arn, ":assumed-role/") {
		return "AssumedRole"
	} else if contains(arn, ":user/") {
		return "IAMUser"
	} else if contains(arn, ":root") {
		return "Root"
	}

	return "Unknown"
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (len(substr) == 0 || s[len(s)-len(substr):] == substr || 
			s[:len(substr)] == substr || 
			findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}