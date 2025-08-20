// Package config handles configuration management for the AWS CLI plugin
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the plugin configuration
type Config struct {
	ProfileName     string           `json:"profile_name"`
	AuthMethod      string           `json:"auth_method"` // sso, cross-account, interactive
	AWSRegion       string           `json:"aws_region"`
	SessionDuration int              `json:"session_duration"`
	SSOStartURL     string           `json:"sso_start_url,omitempty"`
	CrossAccount    CrossAccountConfig `json:"cross_account,omitempty"`
	CacheEnabled    bool             `json:"cache_enabled"`
	Debug           bool             `json:"debug"`
}

// CrossAccountConfig contains cross-account role assumption settings
type CrossAccountConfig struct {
	RoleARN     string `json:"role_arn"`
	ExternalID  string `json:"external_id,omitempty"`
	SessionName string `json:"session_name,omitempty"`
}

// NewDefault creates a new configuration with default values
func NewDefault() *Config {
	return &Config{
		ProfileName:     "remote-access",
		AuthMethod:      "interactive",
		AWSRegion:       "us-east-1", 
		SessionDuration: 3600,
		CacheEnabled:    true,
		Debug:           false,
	}
}

// Load loads configuration from the default location
func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewDefault(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	if debug := os.Getenv("AWS_REMOTE_ACCESS_DEBUG"); debug == "true" {
		config.Debug = true
	}

	return &config, nil
}

// Save saves the configuration to the default location
func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Clear removes the configuration file
func Clear() error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	return nil
}

// IsConfigured returns true if a configuration file exists
func IsConfigured() bool {
	configPath, err := getConfigPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(configPath)
	return err == nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ProfileName == "" {
		return fmt.Errorf("profile_name is required")
	}

	if c.AuthMethod == "" {
		return fmt.Errorf("auth_method is required")
	}

	switch c.AuthMethod {
	case "sso":
		if c.SSOStartURL == "" {
			return fmt.Errorf("sso_start_url is required for SSO authentication")
		}
	case "cross-account":
		if c.CrossAccount.RoleARN == "" {
			return fmt.Errorf("cross_account.role_arn is required for cross-account authentication")
		}
	case "interactive":
		// No additional validation needed
	default:
		return fmt.Errorf("invalid auth_method: %s", c.AuthMethod)
	}

	if c.AWSRegion == "" {
		return fmt.Errorf("aws_region is required")
	}

	if c.SessionDuration < 900 || c.SessionDuration > 43200 {
		return fmt.Errorf("session_duration must be between 900 and 43200 seconds")
	}

	return nil
}

// getConfigPath returns the path to the configuration file
func getConfigPath() (string, error) {
	// Check for environment override
	if configPath := os.Getenv("AWS_REMOTE_ACCESS_CONFIG"); configPath != "" {
		return configPath, nil
	}

	// Use default location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".aws-remote-access-patterns", "plugin-config.json"), nil
}

// GetCacheDir returns the directory for caching credentials and data
func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".aws-remote-access-patterns", "cache")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return cacheDir, nil
}