// Package providers implements IAM user credential provider
package providers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/cache"
	configPkg "github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/config"
)

// IAMUserProvider implements IAM user-based credential provider
type IAMUserProvider struct {
	*BaseProvider
}

// NewIAMUserProvider creates a new IAM user provider
func NewIAMUserProvider(logger *slog.Logger) *IAMUserProvider {
	return &IAMUserProvider{
		BaseProvider: NewBaseProvider(logger),
	}
}

// Type returns the provider type
func (p *IAMUserProvider) Type() string {
	return "iam_user"
}

// GetCredentials retrieves credentials using IAM user access keys
func (p *IAMUserProvider) GetCredentials(ctx context.Context, profile *configPkg.Profile, ciMode bool) (*cache.Credentials, error) {
	if profile.IAMUser == nil {
		return nil, fmt.Errorf("IAM user configuration missing")
	}

	p.logger.Debug("Getting IAM user credentials")
	
	// Create static credentials provider
	staticCredentials := credentials.NewStaticCredentialsProvider(
		profile.IAMUser.AccessKeyID,
		profile.IAMUser.SecretAccessKey,
		"", // No session token for IAM user
	)

	// Create AWS config with static credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(staticCredentials),
		config.WithRegion(profile.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Convert to temporary credentials using STS GetSessionToken
	temporaryCredentials, err := p.getTemporaryCredentials(ctx, cfg, profile.SessionDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get temporary credentials: %w", err)
	}

	// Convert to cache credentials format
	cacheCredentials := &cache.Credentials{
		AccessKeyID:     temporaryCredentials.AccessKeyID,
		SecretAccessKey: temporaryCredentials.SecretAccessKey,
		SessionToken:    temporaryCredentials.SessionToken,
		ExpiresAt:       temporaryCredentials.Expires,
		Region:          profile.Region,
	}

	p.logger.Debug("IAM user credentials retrieved successfully",
		slog.Time("expires_at", cacheCredentials.ExpiresAt))

	return cacheCredentials, nil
}

// getTemporaryCredentials uses STS GetSessionToken to get temporary credentials
func (p *IAMUserProvider) getTemporaryCredentials(ctx context.Context, cfg aws.Config, durationSeconds int) (aws.Credentials, error) {
	stsClient := sts.NewFromConfig(cfg)
	
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(int32(durationSeconds)),
	}
	
	result, err := stsClient.GetSessionToken(ctx, input)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to get session token: %w", err)
	}
	
	if result.Credentials == nil {
		return aws.Credentials{}, fmt.Errorf("no credentials returned from STS")
	}
	
	creds := result.Credentials
	return aws.Credentials{
		AccessKeyID:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		Expires:         *creds.Expiration,
	}, nil
}