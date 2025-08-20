// Package auth provides authentication management for the AWS CLI plugin
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-plugin/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/awsauth"
)

// Manager handles authentication for the AWS CLI plugin
type Manager struct {
	config    *config.Config
	awsClient *awsauth.Client
	logger    *slog.Logger
}

// NewManager creates a new authentication manager
func NewManager(cfg *config.Config) (*Manager, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	logger := slog.Default()

	// Create awsauth configuration
	authConfig := &awsauth.Config{
		ToolName:        "AWS CLI Plugin",
		ToolVersion:     "1.0.0", 
		DefaultRegion:   cfg.AWSRegion,
		SessionDuration: time.Duration(cfg.SessionDuration) * time.Second,
		PreferSSO:       cfg.AuthMethod == "sso",
		SetupUI:         false, // CLI plugin doesn't use UI
	}

	// Configure based on authentication method
	switch cfg.AuthMethod {
	case "sso":
		authConfig.SSOURL = cfg.SSOStartURL
		authConfig.SSORegion = cfg.AWSRegion
	case "cross-account":
		// Cross-account configuration will be handled in GetAWSConfig
	case "interactive":
		// Interactive authentication will be handled by awsauth
	}

	// Create AWS auth client
	awsClient, err := awsauth.New(authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS auth client: %w", err)
	}

	return &Manager{
		config:    cfg,
		awsClient: awsClient,
		logger:    logger,
	}, nil
}

// GetAWSConfig returns AWS configuration with credentials
func (m *Manager) GetAWSConfig(ctx context.Context) (aws.Config, error) {
	m.logger.Debug("Getting AWS configuration",
		slog.String("auth_method", m.config.AuthMethod),
		slog.String("region", m.config.AWSRegion))

	switch m.config.AuthMethod {
	case "cross-account":
		return m.getCrossAccountConfig(ctx)
	default:
		// Use the standard awsauth client
		return m.awsClient.GetAWSConfig(ctx)
	}
}

// getCrossAccountConfig handles cross-account role assumption
func (m *Manager) getCrossAccountConfig(ctx context.Context) (aws.Config, error) {
	if m.config.CrossAccount.RoleARN == "" {
		return aws.Config{}, fmt.Errorf("cross-account role ARN not configured")
	}

	m.logger.Debug("Assuming cross-account role",
		slog.String("role_arn", m.config.CrossAccount.RoleARN),
		slog.String("external_id", m.config.CrossAccount.ExternalID[:min(10, len(m.config.CrossAccount.ExternalID))]))

	// First, get base AWS configuration
	baseConfig, err := m.awsClient.GetAWSConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to get base AWS config: %w", err)
	}

	// Create STS client with base configuration
	stsClient := sts.NewFromConfig(baseConfig)

	// Prepare assume role input
	assumeRoleInput := &sts.AssumeRoleInput{
		RoleArn:         aws.String(m.config.CrossAccount.RoleARN),
		RoleSessionName: aws.String(m.getSessionName()),
		DurationSeconds: aws.Int32(int32(m.config.SessionDuration)),
	}

	// Add external ID if configured
	if m.config.CrossAccount.ExternalID != "" {
		assumeRoleInput.ExternalId = aws.String(m.config.CrossAccount.ExternalID)
	}

	// Assume the role
	result, err := stsClient.AssumeRole(ctx, assumeRoleInput)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to assume role: %w", err)
	}

	// Create new AWS config with assumed role credentials
	newConfig := baseConfig.Copy()
	newConfig.Credentials = aws.NewCredentialsCache(&staticCredentialsProvider{
		accessKey:    *result.Credentials.AccessKeyId,
		secretKey:    *result.Credentials.SecretAccessKey,
		sessionToken: *result.Credentials.SessionToken,
		expires:      *result.Credentials.Expiration,
	})

	m.logger.Info("Successfully assumed cross-account role",
		slog.String("role_arn", m.config.CrossAccount.RoleARN),
		slog.Time("expires", *result.Credentials.Expiration))

	return newConfig, nil
}

// TestConnection tests the authentication configuration
func (m *Manager) TestConnection(ctx context.Context) error {
	m.logger.Debug("Testing authentication connection")

	// Get AWS configuration
	awsConfig, err := m.GetAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS configuration: %w", err)
	}

	// Test with STS GetCallerIdentity
	stsClient := sts.NewFromConfig(awsConfig)
	result, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to get caller identity: %w", err)
	}

	m.logger.Info("Authentication test successful",
		slog.String("user_id", aws.ToString(result.UserId)),
		slog.String("account", aws.ToString(result.Account)),
		slog.String("arn", aws.ToString(result.Arn)))

	return nil
}

// getSessionName generates a session name for role assumption
func (m *Manager) getSessionName() string {
	if m.config.CrossAccount.SessionName != "" {
		return m.config.CrossAccount.SessionName
	}

	// Generate default session name
	return fmt.Sprintf("aws-cli-plugin-%d", time.Now().Unix())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// staticCredentialsProvider provides static AWS credentials
type staticCredentialsProvider struct {
	accessKey    string
	secretKey    string
	sessionToken string
	expires      time.Time
}

// Retrieve returns the static credentials
func (p *staticCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     p.accessKey,
		SecretAccessKey: p.secretKey,
		SessionToken:    p.sessionToken,
		Source:          "aws-cli-plugin-static",
		CanExpire:       !p.expires.IsZero(),
		Expires:         p.expires,
	}, nil
}