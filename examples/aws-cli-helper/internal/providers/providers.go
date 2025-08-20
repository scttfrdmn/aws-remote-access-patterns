// Package providers implements different credential providers for the AWS CLI helper
package providers

import (
	"context"
	"log/slog"

	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/cache"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/config"
)

// Provider interface defines credential provider methods
type Provider interface {
	GetCredentials(ctx context.Context, profile *config.Profile, ciMode bool) (*cache.Credentials, error)
	Type() string
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	logger *slog.Logger
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(logger *slog.Logger) *BaseProvider {
	return &BaseProvider{
		logger: logger,
	}
}