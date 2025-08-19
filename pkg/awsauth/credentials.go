package awsauth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// CredentialManager handles AWS credential storage and retrieval
type CredentialManager struct {
	profileName string
	region      string
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager(profileName, region string) *CredentialManager {
	return &CredentialManager{
		profileName: profileName,
		region:      region,
	}
}

// SaveProfile saves AWS credentials to a specific profile
func (cm *CredentialManager) SaveProfile(accessKey, secretKey, sessionToken string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	awsDir := filepath.Join(homeDir, ".aws")
	if err := os.MkdirAll(awsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	// Update credentials file
	if err := cm.updateCredentialsFile(accessKey, secretKey, sessionToken); err != nil {
		return fmt.Errorf("failed to update credentials file: %w", err)
	}

	// Update config file
	if err := cm.updateConfigFile(); err != nil {
		return fmt.Errorf("failed to update config file: %w", err)
	}

	return nil
}

// LoadProfile loads AWS credentials from a profile
func (cm *CredentialManager) LoadProfile(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile(cm.profileName),
		config.WithRegion(cm.region),
	)
}

// ProfileExists checks if a profile exists
func (cm *CredentialManager) ProfileExists() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	credFile := filepath.Join(homeDir, ".aws", "credentials")
	data, err := os.ReadFile(credFile)
	if err != nil {
		return false
	}

	profileHeader := fmt.Sprintf("[%s]", cm.profileName)
	return strings.Contains(string(data), profileHeader)
}

// DeleteProfile removes a profile from AWS credentials
func (cm *CredentialManager) DeleteProfile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Remove from credentials file
	credFile := filepath.Join(homeDir, ".aws", "credentials")
	if err := cm.removeFromFile(credFile); err != nil {
		return fmt.Errorf("failed to remove from credentials file: %w", err)
	}

	// Remove from config file
	configFile := filepath.Join(homeDir, ".aws", "config")
	if err := cm.removeFromFile(configFile); err != nil {
		return fmt.Errorf("failed to remove from config file: %w", err)
	}

	return nil
}

// ListProfiles lists all available AWS profiles
func (cm *CredentialManager) ListProfiles() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	profiles := make(map[string]bool)

	// Read credentials file
	credFile := filepath.Join(homeDir, ".aws", "credentials")
	if data, err := os.ReadFile(credFile); err == nil {
		profiles = cm.extractProfiles(string(data), profiles, false)
	}

	// Read config file
	configFile := filepath.Join(homeDir, ".aws", "config")
	if data, err := os.ReadFile(configFile); err == nil {
		profiles = cm.extractProfiles(string(data), profiles, true)
	}

	var result []string
	for profile := range profiles {
		result = append(result, profile)
	}

	return result, nil
}

// updateCredentialsFile updates the AWS credentials file
func (cm *CredentialManager) updateCredentialsFile(accessKey, secretKey, sessionToken string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	credFile := filepath.Join(homeDir, ".aws", "credentials")

	// Read existing content
	content := ""
	if data, err := os.ReadFile(credFile); err == nil {
		content = string(data)
	}

	// Remove existing profile section
	content = cm.removeProfileSection(content, cm.profileName)

	// Add new profile section
	profileSection := fmt.Sprintf("\n[%s]\n", cm.profileName)
	profileSection += fmt.Sprintf("aws_access_key_id = %s\n", accessKey)
	profileSection += fmt.Sprintf("aws_secret_access_key = %s\n", secretKey)
	if sessionToken != "" {
		profileSection += fmt.Sprintf("aws_session_token = %s\n", sessionToken)
	}

	content += profileSection

	// Write back with secure permissions
	return os.WriteFile(credFile, []byte(content), 0600)
}

// updateConfigFile updates the AWS config file
func (cm *CredentialManager) updateConfigFile() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configFile := filepath.Join(homeDir, ".aws", "config")

	// Read existing content
	content := ""
	if data, err := os.ReadFile(configFile); err == nil {
		content = string(data)
	}

	// Check if profile already exists in config
	profileHeader := fmt.Sprintf("[profile %s]", cm.profileName)
	if strings.Contains(content, profileHeader) {
		return nil // Profile already exists
	}

	// Add profile section
	profileSection := fmt.Sprintf("\n[profile %s]\n", cm.profileName)
	profileSection += fmt.Sprintf("region = %s\n", cm.region)
	profileSection += fmt.Sprintf("output = json\n")

	content += profileSection

	// Write with secure permissions
	return os.WriteFile(configFile, []byte(content), 0600)
}

// removeFromFile removes a profile section from a file
func (cm *CredentialManager) removeFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil // File doesn't exist, nothing to remove
	}

	content := string(data)
	
	// Determine profile header format based on file type
	profileHeader := fmt.Sprintf("[%s]", cm.profileName)
	if strings.Contains(filepath, "config") {
		profileHeader = fmt.Sprintf("[profile %s]", cm.profileName)
	}

	content = cm.removeProfileSection(content, profileHeader)

	return os.WriteFile(filepath, []byte(content), 0600)
}

// removeProfileSection removes a profile section from content
func (cm *CredentialManager) removeProfileSection(content, profileIdentifier string) string {
	lines := strings.Split(content, "\n")
	var newLines []string
	inTargetProfile := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're entering the target profile
		if trimmed == profileIdentifier || trimmed == fmt.Sprintf("[%s]", cm.profileName) || 
		   trimmed == fmt.Sprintf("[profile %s]", cm.profileName) {
			inTargetProfile = true
			continue
		}
		
		// Check if we're entering a different profile
		if strings.HasPrefix(trimmed, "[") && trimmed != profileIdentifier {
			inTargetProfile = false
		}
		
		// Only keep lines that are not in the target profile
		if !inTargetProfile {
			newLines = append(newLines, line)
		}
	}

	return strings.Join(newLines, "\n")
}

// extractProfiles extracts profile names from AWS config content
func (cm *CredentialManager) extractProfiles(content string, profiles map[string]bool, isConfigFile bool) map[string]bool {
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			profileName := strings.Trim(trimmed, "[]")
			
			if isConfigFile && strings.HasPrefix(profileName, "profile ") {
				profileName = strings.TrimPrefix(profileName, "profile ")
			}
			
			if profileName != "" && profileName != "default" {
				profiles[profileName] = true
			} else if profileName == "default" {
				profiles["default"] = true
			}
		}
	}
	
	return profiles
}

// TemporaryCredentials represents temporary AWS credentials
type TemporaryCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// IsExpired checks if the temporary credentials are expired
func (tc *TemporaryCredentials) IsExpired() bool {
	return time.Now().After(tc.Expiration.Add(-5 * time.Minute)) // 5 minute buffer
}

// ToAWSCredentials converts to AWS SDK credentials
func (tc *TemporaryCredentials) ToAWSCredentials() aws.Credentials {
	return aws.Credentials{
		AccessKeyID:     tc.AccessKeyID,
		SecretAccessKey: tc.SecretAccessKey,
		SessionToken:    tc.SessionToken,
	}
}

// CredentialRefresher handles automatic credential refresh
type CredentialRefresher struct {
	refreshFunc func(ctx context.Context) (*TemporaryCredentials, error)
	credentials *TemporaryCredentials
}

// NewCredentialRefresher creates a new credential refresher
func NewCredentialRefresher(refreshFunc func(ctx context.Context) (*TemporaryCredentials, error)) *CredentialRefresher {
	return &CredentialRefresher{
		refreshFunc: refreshFunc,
	}
}

// GetCredentials gets credentials, refreshing if necessary
func (cr *CredentialRefresher) GetCredentials(ctx context.Context) (*TemporaryCredentials, error) {
	if cr.credentials != nil && !cr.credentials.IsExpired() {
		return cr.credentials, nil
	}

	// Need to refresh
	newCreds, err := cr.refreshFunc(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh credentials: %w", err)
	}

	cr.credentials = newCreds
	return newCreds, nil
}

// ClearCredentials clears cached credentials
func (cr *CredentialRefresher) ClearCredentials() {
	cr.credentials = nil
}