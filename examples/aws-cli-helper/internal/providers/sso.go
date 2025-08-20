// Package providers implements SSO credential provider
package providers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/cache"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/awsauth"
)

// SSOProvider implements SSO-based credential provider
type SSOProvider struct {
	*BaseProvider
}

// NewSSOProvider creates a new SSO provider
func NewSSOProvider(logger *slog.Logger) *SSOProvider {
	return &SSOProvider{
		BaseProvider: NewBaseProvider(logger),
	}
}

// Type returns the provider type
func (p *SSOProvider) Type() string {
	return "sso"
}

// GetCredentials retrieves credentials using AWS SSO
func (p *SSOProvider) GetCredentials(ctx context.Context, profile *config.Profile, ciMode bool) (*cache.Credentials, error) {
	if profile.SSOConfig == nil {
		return nil, fmt.Errorf("SSO configuration missing")
	}

	p.logger.Debug("Getting SSO credentials", 
		slog.String("start_url", profile.SSOConfig.StartURL),
		slog.String("region", profile.SSOConfig.Region))

	// Create awsauth client configuration
	authConfig := &awsauth.Config{
		ToolName:        profile.ToolName,
		DefaultRegion:   profile.Region,
		SessionDuration: time.Duration(profile.SessionDuration) * time.Second,
		PreferSSO:       true,
		CIMode:          ciMode,
	}

	// Create awsauth client
	client, err := awsauth.New(authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	// Get AWS config with credentials
	awsConfig, err := client.GetAWSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Extract credentials from AWS config
	credentials, err := awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	// Convert to cache credentials format
	cacheCredentials := &cache.Credentials{
		AccessKeyID:     credentials.AccessKeyID,
		SecretAccessKey: credentials.SecretAccessKey,
		SessionToken:    credentials.SessionToken,
		ExpiresAt:       credentials.Expires,
		Region:          profile.Region,
	}

	p.logger.Debug("SSO credentials retrieved successfully",
		slog.Time("expires_at", cacheCredentials.ExpiresAt))

	return cacheCredentials, nil
}