// Package config handles configuration management for the AWS CLI helper
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Profiles map[string]*Profile `yaml:"profiles"`
	Cache    CacheConfig         `yaml:"cache"`
	Logging  LoggingConfig       `yaml:"logging"`
	
	// Internal fields
	configPath string `yaml:"-"`
}

// Profile represents a credential profile configuration
type Profile struct {
	ToolName         string               `yaml:"tool_name"`
	AuthMethod       string               `yaml:"auth_method"` // sso, profile, iam_user, cross_account
	Region           string               `yaml:"region"`
	SessionDuration  int                  `yaml:"session_duration"`
	RequiredActions  []string             `yaml:"required_actions,omitempty"`
	SSOConfig        *SSOConfig           `yaml:"sso_config,omitempty"`
	ProfileName      string               `yaml:"profile_name,omitempty"`
	CrossAccount     *CrossAccountConfig  `yaml:"cross_account,omitempty"`
	IAMUser          *IAMUserConfig       `yaml:"iam_user,omitempty"`
}

// SSOConfig represents AWS SSO configuration
type SSOConfig struct {
	StartURL string `yaml:"start_url"`
	Region   string `yaml:"region"`
	RoleName string `yaml:"role_name,omitempty"`
	AccountID string `yaml:"account_id,omitempty"`
}

// CrossAccountConfig represents cross-account role configuration
type CrossAccountConfig struct {
	CustomerID string `yaml:"customer_id"`
	RoleARN    string `yaml:"role_arn"`
	ExternalID string `yaml:"external_id"`
}

// IAMUserConfig represents IAM user configuration
type IAMUserConfig struct {
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Directory string `yaml:"directory"`
	MaxAge    int    `yaml:"max_age"` // seconds
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	
	return &Config{
		Profiles: make(map[string]*Profile),
		Cache: CacheConfig{
			Directory: "~/.aws-remote-access/cache",
			MaxAge:    3300, // 55 minutes (5 min buffer)
		},
		Logging: LoggingConfig{
			Level: "info",
			File:  "~/.aws-remote-access/aws-cli-helper.log",
		},
		configPath: filepath.Join(homeDir, ".aws-remote-access", "config.yaml"),
	}
}

// Load loads configuration from the default location
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configPath := filepath.Join(homeDir, ".aws-remote-access", "config.yaml")
	return LoadFromPath(configPath)
}

// LoadFromPath loads configuration from a specific path
func LoadFromPath(configPath string) (*Config, error) {
	config := DefaultConfig()
	config.configPath = configPath
	
	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}
	
	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return config, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	// Create directory if it doesn't exist
	configDir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(c.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// ConfigPath returns the configuration file path
func (c *Config) ConfigPath() string {
	return c.configPath
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate profiles
	for name, profile := range c.Profiles {
		if err := profile.Validate(); err != nil {
			return fmt.Errorf("profile '%s': %w", name, err)
		}
	}
	
	// Validate cache configuration
	if c.Cache.MaxAge <= 0 {
		return fmt.Errorf("cache max_age must be positive")
	}
	
	return nil
}

// Validate validates a profile configuration
func (p *Profile) Validate() error {
	if p.ToolName == "" {
		return fmt.Errorf("tool_name is required")
	}
	
	if p.AuthMethod == "" {
		return fmt.Errorf("auth_method is required")
	}
	
	validAuthMethods := map[string]bool{
		"sso":           true,
		"profile":       true,
		"iam_user":      true,
		"cross_account": true,
	}
	
	if !validAuthMethods[p.AuthMethod] {
		return fmt.Errorf("invalid auth_method: %s", p.AuthMethod)
	}
	
	if p.SessionDuration <= 0 {
		return fmt.Errorf("session_duration must be positive")
	}
	
	// Validate auth method specific configuration
	switch p.AuthMethod {
	case "sso":
		if p.SSOConfig == nil {
			return fmt.Errorf("sso_config is required for SSO auth method")
		}
		if err := p.SSOConfig.Validate(); err != nil {
			return fmt.Errorf("sso_config: %w", err)
		}
		
	case "profile":
		if p.ProfileName == "" {
			return fmt.Errorf("profile_name is required for profile auth method")
		}
		
	case "cross_account":
		if p.CrossAccount == nil {
			return fmt.Errorf("cross_account config is required for cross_account auth method")
		}
		if err := p.CrossAccount.Validate(); err != nil {
			return fmt.Errorf("cross_account: %w", err)
		}
		
	case "iam_user":
		if p.IAMUser == nil {
			return fmt.Errorf("iam_user config is required for iam_user auth method")
		}
		if err := p.IAMUser.Validate(); err != nil {
			return fmt.Errorf("iam_user: %w", err)
		}
	}
	
	return nil
}

// Validate validates SSO configuration
func (s *SSOConfig) Validate() error {
	if s.StartURL == "" {
		return fmt.Errorf("start_url is required")
	}
	
	if s.Region == "" {
		return fmt.Errorf("region is required")
	}
	
	return nil
}

// Validate validates cross-account configuration
func (c *CrossAccountConfig) Validate() error {
	if c.CustomerID == "" {
		return fmt.Errorf("customer_id is required")
	}
	
	if c.RoleARN == "" {
		return fmt.Errorf("role_arn is required")
	}
	
	if c.ExternalID == "" {
		return fmt.Errorf("external_id is required")
	}
	
	// Validate role ARN format
	if !strings.HasPrefix(c.RoleARN, "arn:aws:iam::") {
		return fmt.Errorf("invalid role_arn format")
	}
	
	return nil
}

// Validate validates IAM user configuration
func (i *IAMUserConfig) Validate() error {
	if i.AccessKeyID == "" {
		return fmt.Errorf("access_key_id is required")
	}
	
	if i.SecretAccessKey == "" {
		return fmt.Errorf("secret_access_key is required")
	}
	
	// Validate access key format
	if !strings.HasPrefix(i.AccessKeyID, "AKIA") {
		return fmt.Errorf("invalid access_key_id format")
	}
	
	return nil
}

// GetExpandedCacheDirectory returns the cache directory with ~ expanded
func (c *Config) GetExpandedCacheDirectory() string {
	return expandPath(c.Cache.Directory)
}

// GetExpandedLogFile returns the log file path with ~ expanded
func (c *Config) GetExpandedLogFile() string {
	return expandPath(c.Logging.File)
}

// expandPath expands ~ in file paths
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}