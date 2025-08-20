// Package config provides configuration management for the CLI tool
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	// Global settings
	Debug     bool   `yaml:"debug" mapstructure:"debug"`
	Quiet     bool   `yaml:"quiet" mapstructure:"quiet"`
	NoColor   bool   `yaml:"no_color" mapstructure:"no_color"`
	ConfigDir string `yaml:"config_dir" mapstructure:"config_dir"`

	// AWS settings
	AWSRegion  string `yaml:"aws_region" mapstructure:"aws_region"`
	AWSProfile string `yaml:"aws_profile" mapstructure:"aws_profile"`

	// Authentication settings
	Auth AuthConfig `yaml:"auth" mapstructure:"auth"`

	// CLI-specific settings
	CLI CLIConfig `yaml:"cli" mapstructure:"cli"`

	// Data processing settings
	Data DataConfig `yaml:"data" mapstructure:"data"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Method          string        `yaml:"method" mapstructure:"method"`
	Region          string        `yaml:"region" mapstructure:"region"`
	SessionDuration int           `yaml:"session_duration" mapstructure:"session_duration"`
	CacheEnabled    bool          `yaml:"cache_enabled" mapstructure:"cache_enabled"`
	SSO             SSOConfig     `yaml:"sso" mapstructure:"sso"`
	Profile         ProfileConfig `yaml:"profile" mapstructure:"profile"`
}

// SSOConfig represents AWS SSO configuration
type SSOConfig struct {
	StartURL  string `yaml:"start_url" mapstructure:"start_url"`
	Region    string `yaml:"region" mapstructure:"region"`
	RoleName  string `yaml:"role_name" mapstructure:"role_name"`
	AccountID string `yaml:"account_id" mapstructure:"account_id"`
}

// ProfileConfig represents AWS profile configuration
type ProfileConfig struct {
	Name string `yaml:"name" mapstructure:"name"`
}

// CLIConfig represents CLI-specific configuration
type CLIConfig struct {
	OutputFormat    string `yaml:"output_format" mapstructure:"output_format"`
	TableStyle      string `yaml:"table_style" mapstructure:"table_style"`
	PageSize        int    `yaml:"page_size" mapstructure:"page_size"`
	ConfirmActions  bool   `yaml:"confirm_actions" mapstructure:"confirm_actions"`
	ShowProgress    bool   `yaml:"show_progress" mapstructure:"show_progress"`
	AutoPagination  bool   `yaml:"auto_pagination" mapstructure:"auto_pagination"`
}

// DataConfig represents data processing configuration
type DataConfig struct {
	DefaultBucket      string            `yaml:"default_bucket" mapstructure:"default_bucket"`
	TemporaryDirectory string            `yaml:"temporary_directory" mapstructure:"temporary_directory"`
	MaxConcurrency     int               `yaml:"max_concurrency" mapstructure:"max_concurrency"`
	ChunkSize          int64             `yaml:"chunk_size" mapstructure:"chunk_size"`
	Environments       map[string]string `yaml:"environments" mapstructure:"environments"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".datatool")

	return &Config{
		Debug:     false,
		Quiet:     false,
		NoColor:   false,
		ConfigDir: configDir,
		AWSRegion: "us-east-1",

		Auth: AuthConfig{
			Method:          "",
			Region:          "us-east-1",
			SessionDuration: 3600,
			CacheEnabled:    true,
		},

		CLI: CLIConfig{
			OutputFormat:    "table",
			TableStyle:      "default",
			PageSize:        50,
			ConfirmActions:  true,
			ShowProgress:    true,
			AutoPagination:  true,
		},

		Data: DataConfig{
			DefaultBucket:      "",
			TemporaryDirectory: filepath.Join(os.TempDir(), "datatool"),
			MaxConcurrency:     10,
			ChunkSize:          10 * 1024 * 1024, // 10MB
			Environments:       make(map[string]string),
		},
	}
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Set config file search paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(cfg.ConfigDir)
	viper.AddConfigPath(".")

	// Set environment variable prefix
	viper.SetEnvPrefix("DATATOOL")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}

	// Unmarshal configuration
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(cfg.ConfigDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configFile := filepath.Join(c.ConfigDir, "config.yaml")

	// Ensure config directory exists
	if err := os.MkdirAll(c.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set values in viper
	viper.Set("debug", c.Debug)
	viper.Set("quiet", c.Quiet)
	viper.Set("no_color", c.NoColor)
	viper.Set("aws_region", c.AWSRegion)
	viper.Set("aws_profile", c.AWSProfile)
	viper.Set("auth", c.Auth)
	viper.Set("cli", c.CLI)
	viper.Set("data", c.Data)

	// Write config file
	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate auth configuration
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

	// Validate CLI configuration
	if c.CLI.OutputFormat != "" {
		validFormats := []string{"table", "json", "yaml", "csv"}
		isValid := false
		for _, format := range validFormats {
			if c.CLI.OutputFormat == format {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid output format: %s", c.CLI.OutputFormat)
		}
	}

	// Validate data configuration
	if c.Data.MaxConcurrency < 1 {
		return fmt.Errorf("max_concurrency must be at least 1")
	}

	if c.Data.ChunkSize < 1024 {
		return fmt.Errorf("chunk_size must be at least 1024 bytes")
	}

	return nil
}

// GetCacheDir returns the cache directory path
func (c *Config) GetCacheDir() string {
	return filepath.Join(c.ConfigDir, "cache")
}

// GetLogFile returns the log file path
func (c *Config) GetLogFile() string {
	return filepath.Join(c.ConfigDir, "datatool.log")
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
	return "us-east-1" // fallback default
}

// SetAuthConfig updates the authentication configuration
func (c *Config) SetAuthConfig(authConfig AuthConfig) {
	c.Auth = authConfig
}

// GetEnvironmentBucket returns the S3 bucket for a specific environment
func (c *Config) GetEnvironmentBucket(env string) (string, bool) {
	bucket, exists := c.Data.Environments[env]
	return bucket, exists
}