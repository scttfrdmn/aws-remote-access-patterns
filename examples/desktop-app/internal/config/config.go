// Package config provides configuration management for the desktop application
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the desktop application configuration
type Config struct {
	// Application settings
	Debug         bool   `json:"debug"`
	LogLevel      string `json:"log_level"`
	ConfigDir     string `json:"config_dir"`
	Theme         string `json:"theme"`
	Language      string `json:"language"`
	AutoStart     bool   `json:"auto_start"`
	MinimizeToTray bool  `json:"minimize_to_tray"`

	// AWS settings
	AWSRegion     string `json:"aws_region"`
	AWSProfile    string `json:"aws_profile,omitempty"`
	SessionTimeout int   `json:"session_timeout"` // minutes

	// Authentication settings
	Auth AuthConfig `json:"auth"`

	// UI settings
	UI UIConfig `json:"ui"`

	// Features
	Features FeatureConfig `json:"features"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Method          string        `json:"method"`           // sso, profile, interactive
	Region          string        `json:"region"`
	SessionDuration int           `json:"session_duration"` // seconds
	CacheEnabled    bool          `json:"cache_enabled"`
	AutoRefresh     bool          `json:"auto_refresh"`
	SSO             SSOConfig     `json:"sso"`
	Profile         ProfileConfig `json:"profile"`
}

// SSOConfig represents AWS SSO configuration
type SSOConfig struct {
	StartURL  string `json:"start_url"`
	Region    string `json:"region"`
	RoleName  string `json:"role_name,omitempty"`
	AccountID string `json:"account_id,omitempty"`
}

// ProfileConfig represents AWS profile configuration
type ProfileConfig struct {
	Name string `json:"name"`
}

// UIConfig represents UI configuration
type UIConfig struct {
	Theme                string `json:"theme"`                  // light, dark, auto
	CompactMode          bool   `json:"compact_mode"`
	ShowAdvancedFeatures bool   `json:"show_advanced_features"`
	AutoRefresh          bool   `json:"auto_refresh"`
	RefreshInterval      int    `json:"refresh_interval"` // seconds
	Notifications        bool   `json:"notifications"`
	SoundEnabled         bool   `json:"sound_enabled"`
}

// FeatureConfig represents feature toggles
type FeatureConfig struct {
	S3Browser       bool `json:"s3_browser"`
	EC2Management   bool `json:"ec2_management"`
	LogsViewer      bool `json:"logs_viewer"`
	CostExplorer    bool `json:"cost_explorer"`
	IAMHelper       bool `json:"iam_helper"`
	QuickActions    bool `json:"quick_actions"`
	SystemTray      bool `json:"system_tray"`
	ShortcutKeys    bool `json:"shortcut_keys"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".aws-desktop-app")

	return &Config{
		Debug:          false,
		LogLevel:       "info",
		ConfigDir:      configDir,
		Theme:          "auto",
		Language:       "en",
		AutoStart:      false,
		MinimizeToTray: true,
		AWSRegion:      "us-east-1",
		SessionTimeout: 60, // 1 hour

		Auth: AuthConfig{
			Method:          "",
			Region:          "us-east-1",
			SessionDuration: 3600, // 1 hour
			CacheEnabled:    true,
			AutoRefresh:     true,
		},

		UI: UIConfig{
			Theme:                "auto",
			CompactMode:          false,
			ShowAdvancedFeatures: false,
			AutoRefresh:          true,
			RefreshInterval:      30, // 30 seconds
			Notifications:        true,
			SoundEnabled:         true,
		},

		Features: FeatureConfig{
			S3Browser:       true,
			EC2Management:   true,
			LogsViewer:      true,
			CostExplorer:    false, // Advanced feature
			IAMHelper:       false, // Advanced feature
			QuickActions:    true,
			SystemTray:      true,
			ShortcutKeys:    true,
		},
	}
}

// Load loads configuration from file
func Load() (*Config, error) {
	cfg := DefaultConfig()
	configFile := filepath.Join(cfg.ConfigDir, "config.json")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(cfg.ConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return cfg, nil
	}

	// Read and parse config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configFile := filepath.Join(c.ConfigDir, "config.json")

	// Ensure config directory exists
	if err := os.MkdirAll(c.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate auth method
	if c.Auth.Method != "" {
		validMethods := []string{"sso", "profile", "interactive"}
		isValid := false
		for _, method := range validMethods {
			if c.Auth.Method == method {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid auth method: %s", c.Auth.Method)
		}
	}

	// Validate session duration
	if c.Auth.SessionDuration < 900 || c.Auth.SessionDuration > 43200 {
		return fmt.Errorf("session duration must be between 900 and 43200 seconds")
	}

	// Validate session timeout
	if c.SessionTimeout < 5 || c.SessionTimeout > 1440 {
		return fmt.Errorf("session timeout must be between 5 and 1440 minutes")
	}

	// Validate theme
	validThemes := []string{"light", "dark", "auto"}
	isValidTheme := false
	for _, theme := range validThemes {
		if c.UI.Theme == theme {
			isValidTheme = true
			break
		}
	}
	if !isValidTheme {
		return fmt.Errorf("invalid theme: %s", c.UI.Theme)
	}

	return nil
}

// IsAuthConfigured returns true if authentication is configured
func (c *Config) IsAuthConfigured() bool {
	return c.Auth.Method != ""
}

// GetAWSRegion returns the AWS region to use
func (c *Config) GetAWSRegion() string {
	if c.AWSRegion != "" {
		return c.AWSRegion
	}
	if c.Auth.Region != "" {
		return c.Auth.Region
	}
	return "us-east-1"
}

// GetCacheDir returns the cache directory path
func (c *Config) GetCacheDir() string {
	return filepath.Join(c.ConfigDir, "cache")
}

// GetLogFile returns the log file path
func (c *Config) GetLogFile() string {
	return filepath.Join(c.ConfigDir, "app.log")
}

// SetAuthConfig updates the authentication configuration
func (c *Config) SetAuthConfig(authConfig AuthConfig) {
	c.Auth = authConfig
}

// EnableFeature enables a specific feature
func (c *Config) EnableFeature(feature string) {
	switch feature {
	case "s3_browser":
		c.Features.S3Browser = true
	case "ec2_management":
		c.Features.EC2Management = true
	case "logs_viewer":
		c.Features.LogsViewer = true
	case "cost_explorer":
		c.Features.CostExplorer = true
	case "iam_helper":
		c.Features.IAMHelper = true
	case "quick_actions":
		c.Features.QuickActions = true
	case "system_tray":
		c.Features.SystemTray = true
	case "shortcut_keys":
		c.Features.ShortcutKeys = true
	}
}

// DisableFeature disables a specific feature
func (c *Config) DisableFeature(feature string) {
	switch feature {
	case "s3_browser":
		c.Features.S3Browser = false
	case "ec2_management":
		c.Features.EC2Management = false
	case "logs_viewer":
		c.Features.LogsViewer = false
	case "cost_explorer":
		c.Features.CostExplorer = false
	case "iam_helper":
		c.Features.IAMHelper = false
	case "quick_actions":
		c.Features.QuickActions = false
	case "system_tray":
		c.Features.SystemTray = false
	case "shortcut_keys":
		c.Features.ShortcutKeys = false
	}
}