// Package providers implements profile-based credential provider
package providers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/cache"
	configPkg "github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/config"
)

// ProfileProvider implements profile-based credential provider
type ProfileProvider struct {
	*BaseProvider
}

// NewProfileProvider creates a new profile provider
func NewProfileProvider(logger *slog.Logger) *ProfileProvider {
	return &ProfileProvider{
		BaseProvider: NewBaseProvider(logger),
	}
}

// Type returns the provider type
func (p *ProfileProvider) Type() string {
	return "profile"
}

// GetCredentials retrieves credentials from an AWS profile
func (p *ProfileProvider) GetCredentials(ctx context.Context, profile *configPkg.Profile, ciMode bool) (*cache.Credentials, error) {
	if profile.ProfileName == "" {
		return nil, fmt.Errorf("profile name missing")
	}

	p.logger.Debug("Getting profile credentials", slog.String("profile", profile.ProfileName))

	// Load AWS config from the specified profile
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(profile.ProfileName),
		config.WithRegion(profile.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Get credentials from the config
	credentials, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	// If these are long-lived credentials, we might want to assume a role
	// to get temporary credentials with a defined expiration
	if credentials.Expires.IsZero() {
		p.logger.Debug("Converting long-lived credentials to temporary credentials")
		credentials, err = p.getTemporaryCredentials(ctx, cfg, profile.SessionDuration)
		if err != nil {
			return nil, fmt.Errorf("failed to get temporary credentials: %w", err)
		}
	}

	// Convert to cache credentials format
	cacheCredentials := &cache.Credentials{
		AccessKeyID:     credentials.AccessKeyID,
		SecretAccessKey: credentials.SecretAccessKey,
		SessionToken:    credentials.SessionToken,
		ExpiresAt:       credentials.Expires,
		Region:          profile.Region,
	}

	p.logger.Debug("Profile credentials retrieved successfully",
		slog.Time("expires_at", cacheCredentials.ExpiresAt))

	return cacheCredentials, nil
}

// getTemporaryCredentials uses STS GetSessionToken to get temporary credentials
func (p *ProfileProvider) getTemporaryCredentials(ctx context.Context, cfg aws.Config, durationSeconds int) (aws.Credentials, error) {
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