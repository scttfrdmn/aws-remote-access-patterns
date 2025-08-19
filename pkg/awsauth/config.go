package awsauth

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// Config defines the tool's AWS authentication requirements
type Config struct {
	// Tool identification
	ToolName    string `json:"tool_name" yaml:"tool_name"`
	ToolVersion string `json:"tool_version" yaml:"tool_version"`

	// AWS settings
	DefaultRegion   string        `json:"default_region" yaml:"default_region"`
	ProfileName     string        `json:"profile_name" yaml:"profile_name"`
	SessionDuration time.Duration `json:"session_duration" yaml:"session_duration"`

	// Required permissions
	RequiredActions   []string     `json:"required_actions" yaml:"required_actions"`
	CustomPermissions []Permission `json:"custom_permissions" yaml:"custom_permissions"`

	// Authentication preferences
	PreferSSO    bool `json:"prefer_sso" yaml:"prefer_sso"`
	AllowIAMUser bool `json:"allow_iam_user" yaml:"allow_iam_user"`
	AllowEnvVars bool `json:"allow_env_vars" yaml:"allow_env_vars"`

	// Setup options
	SetupUI         bool              `json:"setup_ui" yaml:"setup_ui"`
	BrandingOptions map[string]string `json:"branding_options" yaml:"branding_options"`

	// CI/CD settings
	CIMode bool `json:"ci_mode" yaml:"ci_mode"`
}

// Permission represents an IAM policy statement
type Permission struct {
	Sid       string                 `json:"sid" yaml:"sid"`
	Effect    string                 `json:"effect" yaml:"effect"`
	Actions   []string               `json:"actions" yaml:"actions"`
	Resources []string               `json:"resources" yaml:"resources"`
	Condition map[string]interface{} `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// Validate ensures the config has minimum required fields and sets defaults
func (c *Config) Validate() error {
	if c.ToolName == "" {
		return errors.New("tool_name is required")
	}

	// Set defaults
	if c.DefaultRegion == "" {
		c.DefaultRegion = "us-east-1"
	}
	if c.SessionDuration == 0 {
		c.SessionDuration = 12 * time.Hour
	}
	if c.RequiredActions == nil {
		c.RequiredActions = []string{"sts:GetCallerIdentity"}
	}

	// Enable reasonable defaults if nothing specified
	if !c.PreferSSO && !c.AllowIAMUser && !c.AllowEnvVars {
		c.PreferSSO = true
		c.AllowIAMUser = true
		c.AllowEnvVars = true
	}

	return nil
}

// DefaultConfig returns a config with sensible defaults for most CLI tools
func DefaultConfig(toolName string) *Config {
	return &Config{
		ToolName:        toolName,
		DefaultRegion:   "us-east-1",
		SessionDuration: 12 * time.Hour,
		RequiredActions: []string{
			"sts:GetCallerIdentity",
		},
		PreferSSO:    true,
		AllowIAMUser: true,
		AllowEnvVars: true,
		SetupUI:      true,
	}
}

// CICDConfig returns a config optimized for CI/CD environments
func CICDConfig(toolName string) *Config {
	return &Config{
		ToolName:        toolName,
		DefaultRegion:   "us-east-1",
		SessionDuration: 1 * time.Hour,
		RequiredActions: []string{
			"sts:GetCallerIdentity",
		},
		PreferSSO:    false, // SSO doesn't work in CI
		AllowIAMUser: true,
		AllowEnvVars: true,
		SetupUI:      false, // No UI in CI
		CIMode:       true,
	}
}

// CachedCredentials stores AWS config with expiration time
type CachedCredentials struct {
	AWSConfig aws.Config
	ExpiresAt time.Time
}

// IsValid checks if cached credentials are still valid
func (c *CachedCredentials) IsValid() bool {
	return time.Now().Before(c.ExpiresAt.Add(-5 * time.Minute)) // 5min buffer
}

// CredentialCache manages cached AWS credentials
type CredentialCache struct {
	cache map[string]*CachedCredentials
}

// NewCredentialCache creates a new credential cache
func NewCredentialCache() *CredentialCache {
	return &CredentialCache{
		cache: make(map[string]*CachedCredentials),
	}
}

// Get retrieves cached credentials if they're still valid
func (c *CredentialCache) Get(key string) *CachedCredentials {
	if creds, ok := c.cache[key]; ok && creds.IsValid() {
		return creds
	}
	return nil
}

// Set stores credentials in the cache
func (c *CredentialCache) Set(key string, creds *CachedCredentials) {
	c.cache[key] = creds
}

// Clear removes cached credentials
func (c *CredentialCache) Clear(key string) {
	delete(c.cache, key)
}

// SetupUI handles web-based setup interface
type SetupUI struct {
	config *Config
}

// NewSetupUI creates a new setup UI handler
func NewSetupUI(cfg *Config) *SetupUI {
	return &SetupUI{config: cfg}
}

// Launch starts the web-based setup interface
func (s *SetupUI) Launch(ctx context.Context) error {
	// This would launch a local web server with setup UI
	// For now, return not implemented
	return errors.New("web UI setup not yet implemented")
}