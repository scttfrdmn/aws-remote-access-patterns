// Package providers implements cross-account credential provider
package providers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/cache"
	configPkg "github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/crossaccount"
)

// CrossAccountProvider implements cross-account role credential provider
type CrossAccountProvider struct {
	*BaseProvider
}

// NewCrossAccountProvider creates a new cross-account provider
func NewCrossAccountProvider(logger *slog.Logger) *CrossAccountProvider {
	return &CrossAccountProvider{
		BaseProvider: NewBaseProvider(logger),
	}
}

// Type returns the provider type
func (p *CrossAccountProvider) Type() string {
	return "cross_account"
}

// GetCredentials retrieves credentials by assuming a cross-account role
func (p *CrossAccountProvider) GetCredentials(ctx context.Context, profile *configPkg.Profile, ciMode bool) (*cache.Credentials, error) {
	if profile.CrossAccount == nil {
		return nil, fmt.Errorf("cross account configuration missing")
	}

	p.logger.Debug("Getting cross-account credentials",
		slog.String("customer_id", profile.CrossAccount.CustomerID),
		slog.String("role_arn", profile.CrossAccount.RoleARN))

	// Create crossaccount client configuration
	// Note: This is a simplified example. In practice, you would need to
	// configure the crossaccount client with your service credentials
	crossAccountConfig := &crossaccount.Config{
		ServiceName:      profile.ToolName,
		ServiceAccountID: "123456789012", // This should come from configuration
		DefaultRegion:    profile.Region,
		SessionDuration:  time.Duration(profile.SessionDuration) * time.Second,
	}

	// Create crossaccount client
	client, err := crossaccount.New(crossAccountConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cross-account client: %w", err)
	}

	// Assume the customer's role
	awsConfig, err := client.AssumeRole(ctx, profile.CrossAccount.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
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

	p.logger.Debug("Cross-account credentials retrieved successfully",
		slog.String("customer_id", profile.CrossAccount.CustomerID),
		slog.Time("expires_at", cacheCredentials.ExpiresAt))

	return cacheCredentials, nil
}