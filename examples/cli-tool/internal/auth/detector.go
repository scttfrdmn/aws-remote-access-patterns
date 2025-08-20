// Package auth provides configuration detection capabilities
package auth

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// ConfigDetector detects existing AWS configurations
type ConfigDetector struct{}

// DetectedConfig represents a detected AWS configuration
type DetectedConfig struct {
	Name        string
	Type        string
	Description string
	Path        string
}

// NewConfigDetector creates a new configuration detector
func NewConfigDetector() *ConfigDetector {
	return &ConfigDetector{}
}

// DetectConfigurations detects all available AWS configurations
func (d *ConfigDetector) DetectConfigurations(ctx context.Context) ([]DetectedConfig, error) {
	var configs []DetectedConfig

	// Detect AWS profiles
	profiles, err := d.DetectProfiles(ctx)
	if err == nil && len(profiles) > 0 {
		for _, profile := range profiles {
			configs = append(configs, DetectedConfig{
				Name:        profile,
				Type:        "profile",
				Description: "AWS profile from ~/.aws/credentials",
				Path:        d.getCredentialsPath(),
			})
		}
	}

	// Detect SSO configurations
	ssoConfigs, err := d.DetectSSOConfigurations(ctx)
	if err == nil && len(ssoConfigs) > 0 {
		configs = append(configs, ssoConfigs...)
	}

	// Detect environment variables
	if d.hasEnvironmentCredentials() {
		configs = append(configs, DetectedConfig{
			Name:        "environment",
			Type:        "environment",
			Description: "AWS credentials from environment variables",
			Path:        "environment",
		})
	}

	return configs, nil
}

// DetectProfiles detects AWS profiles from ~/.aws/credentials
func (d *ConfigDetector) DetectProfiles(ctx context.Context) ([]string, error) {
	credentialsPath := d.getCredentialsPath()

	// Check if credentials file exists
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return nil, nil
	}

	// Parse credentials file
	cfg, err := ini.Load(credentialsPath)
	if err != nil {
		return nil, err
	}

	var profiles []string
	for _, section := range cfg.Sections() {
		if section.Name() != "DEFAULT" {
			profiles = append(profiles, section.Name())
		}
	}

	return profiles, nil
}

// DetectSSOConfigurations detects AWS SSO configurations
func (d *ConfigDetector) DetectSSOConfigurations(ctx context.Context) ([]DetectedConfig, error) {
	var configs []DetectedConfig

	configPath := d.getConfigPath()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return configs, nil
	}

	// Parse config file
	cfg, err := ini.Load(configPath)
	if err != nil {
		return configs, err
	}

	for _, section := range cfg.Sections() {
		if section.HasKey("sso_start_url") {
			profileName := strings.TrimPrefix(section.Name(), "profile ")
			if profileName == section.Name() {
				profileName = "default"
			}

			startURL := section.Key("sso_start_url").String()
			configs = append(configs, DetectedConfig{
				Name:        profileName,
				Type:        "sso",
				Description: fmt.Sprintf("AWS SSO profile (%s)", startURL),
				Path:        configPath,
			})
		}
	}

	return configs, nil
}

// hasEnvironmentCredentials checks if AWS credentials are available in environment
func (d *ConfigDetector) hasEnvironmentCredentials() bool {
	accessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	
	return accessKeyID != "" && secretAccessKey != ""
}

// getCredentialsPath returns the path to the AWS credentials file
func (d *ConfigDetector) getCredentialsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".aws", "credentials")
}

// getConfigPath returns the path to the AWS config file
func (d *ConfigDetector) getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".aws", "config")
}

// GetSSOCacheDir returns the SSO cache directory
func (d *ConfigDetector) GetSSOCacheDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".aws", "sso", "cache")
}

// DetectSSOSessions detects active SSO sessions
func (d *ConfigDetector) DetectSSOSessions(ctx context.Context) ([]string, error) {
	cacheDir := d.GetSSOCacheDir()
	
	// Check if cache directory exists
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return nil, nil
	}

	// List cache files
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			sessionName := strings.TrimSuffix(entry.Name(), ".json")
			sessions = append(sessions, sessionName)
		}
	}

	return sessions, nil
}