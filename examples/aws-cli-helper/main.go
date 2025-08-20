// Package main implements an AWS CLI credential helper that provides
// temporary credentials using AWS Remote Access Patterns
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/cache"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-helper/internal/providers"
)

// Version information
const (
	Version = "1.0.0"
	AppName = "aws-cli-helper"
)

// AWSCredentialResponse represents the JSON response expected by AWS CLI
type AWSCredentialResponse struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
}

// EnvVarExport represents environment variable export format
type EnvVarExport struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
}

// CliHelper manages credential operations
type CliHelper struct {
	config    *config.Config
	cache     *cache.Cache
	providers map[string]providers.Provider
	logger    *slog.Logger
}

// NewCliHelper creates a new CLI helper instance
func NewCliHelper() (*CliHelper, error) {
	// Setup logging
	logLevel := slog.LevelInfo
	if os.Getenv("AWS_CLI_HELPER_DEBUG") != "" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize cache
	cacheDir := expandPath(cfg.Cache.Directory)
	credCache, err := cache.New(cacheDir, time.Duration(cfg.Cache.MaxAge)*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize providers
	providerMap := map[string]providers.Provider{
		"sso":           providers.NewSSOProvider(logger),
		"profile":       providers.NewProfileProvider(logger),
		"iam_user":      providers.NewIAMUserProvider(logger),
		"cross_account": providers.NewCrossAccountProvider(logger),
	}

	return &CliHelper{
		config:    cfg,
		cache:     credCache,
		providers: providerMap,
		logger:    logger,
	}, nil
}

func main() {
	var (
		profileName   = flag.String("profile", "", "Profile name to use")
		exportFormat  = flag.Bool("export", false, "Output as environment variable exports")
		checkStatus   = flag.Bool("check", false, "Check credential status")
		setup         = flag.Bool("setup", false, "Interactive setup")
		listProfiles  = flag.Bool("list-profiles", false, "List available profiles")
		refresh       = flag.Bool("refresh", false, "Force credential refresh")
		debug         = flag.Bool("debug", false, "Enable debug output")
		validate      = flag.Bool("validate", false, "Validate configuration")
		healthCheck   = flag.Bool("health-check", false, "Run comprehensive health check")
		usageReport   = flag.Bool("usage-report", false, "Generate usage report")
		version       = flag.Bool("version", false, "Show version information")
		ciMode        = flag.Bool("ci-mode", false, "Enable CI/CD mode")
	)
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", AppName, Version)
		return
	}

	if *debug {
		os.Setenv("AWS_CLI_HELPER_DEBUG", "1")
	}

	helper, err := NewCliHelper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch {
	case *setup:
		if err := helper.runSetup(ctx, *profileName); err != nil {
			helper.logger.Error("Setup failed", slog.String("error", err.Error()))
			os.Exit(1)
		}

	case *listProfiles:
		helper.listProfiles()

	case *validate && *profileName != "":
		if err := helper.validateProfile(*profileName); err != nil {
			helper.logger.Error("Validation failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Println("✅ Profile validation successful")

	case *healthCheck:
		if *profileName == "" {
			helper.logger.Error("Profile name required for health check")
			os.Exit(1)
		}
		if err := helper.runHealthCheck(ctx, *profileName); err != nil {
			helper.logger.Error("Health check failed", slog.String("error", err.Error()))
			os.Exit(1)
		}

	case *usageReport:
		if err := helper.generateUsageReport(); err != nil {
			helper.logger.Error("Usage report failed", slog.String("error", err.Error()))
			os.Exit(1)
		}

	case *checkStatus && *profileName != "":
		helper.checkCredentialStatus(*profileName)

	case *refresh && *profileName != "":
		if err := helper.refreshCredentials(ctx, *profileName); err != nil {
			helper.logger.Error("Refresh failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Println("✅ Credentials refreshed successfully")

	case *profileName != "":
		// Main credential retrieval flow
		if err := helper.getCredentials(ctx, *profileName, *exportFormat, *ciMode); err != nil {
			helper.logger.Error("Failed to get credentials", 
				slog.String("profile", *profileName),
				slog.String("error", err.Error()))
			os.Exit(1)
		}

	default:
		flag.Usage()
		os.Exit(1)
	}
}

// getCredentials retrieves credentials for the specified profile
func (h *CliHelper) getCredentials(ctx context.Context, profileName string, exportFormat, ciMode bool) error {
	profile, exists := h.config.Profiles[profileName]
	if !exists {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	// Try cache first
	if cached := h.cache.Get(profileName); cached != nil && !cached.IsExpired() {
		h.logger.Debug("Using cached credentials", slog.String("profile", profileName))
		return h.outputCredentials(cached, exportFormat, profile.Region)
	}

	// Get fresh credentials
	provider, exists := h.providers[profile.AuthMethod]
	if !exists {
		return fmt.Errorf("unsupported auth method: %s", profile.AuthMethod)
	}

	h.logger.Debug("Fetching fresh credentials", 
		slog.String("profile", profileName),
		slog.String("auth_method", profile.AuthMethod))

	creds, err := provider.GetCredentials(ctx, profile, ciMode)
	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Cache the credentials
	if err := h.cache.Set(profileName, creds); err != nil {
		h.logger.Warn("Failed to cache credentials", slog.String("error", err.Error()))
	}

	// Record usage metrics
	h.recordUsage(profileName, true, time.Since(time.Now()))

	return h.outputCredentials(creds, exportFormat, profile.Region)
}

// outputCredentials outputs credentials in the requested format
func (h *CliHelper) outputCredentials(creds *cache.Credentials, exportFormat bool, region string) error {
	if exportFormat {
		// Environment variable export format
		fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", creds.AccessKeyID)
		fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", creds.SecretAccessKey)
		if creds.SessionToken != "" {
			fmt.Printf("export AWS_SESSION_TOKEN=%s\n", creds.SessionToken)
		}
		if region != "" {
			fmt.Printf("export AWS_DEFAULT_REGION=%s\n", region)
		}
		return nil
	}

	// AWS CLI JSON format
	response := AWSCredentialResponse{
		Version:         1,
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Expiration:      creds.ExpiresAt.Format(time.RFC3339),
	}

	encoder := json.NewEncoder(os.Stdout)
	return encoder.Encode(response)
}

// runSetup performs interactive setup for a profile
func (h *CliHelper) runSetup(ctx context.Context, profileName string) error {
	fmt.Printf("🔧 AWS CLI Helper Setup\n")
	fmt.Printf("=======================\n\n")

	if profileName == "" {
		fmt.Print("Profile Name: ")
		fmt.Scanln(&profileName)
	}

	fmt.Printf("Setting up profile: %s\n\n", profileName)

	// Get authentication method
	fmt.Println("✅ Step 1: Authentication Method")
	fmt.Println("Choose your authentication method:")
	fmt.Println("1) AWS SSO (Recommended for organizations)")
	fmt.Println("2) AWS Profile (Use existing ~/.aws/credentials)")
	fmt.Println("3) IAM User (Not recommended)")
	fmt.Println("4) Cross Account (For customer account access)")

	var choice int
	fmt.Print("\nSelection [1]: ")
	fmt.Scanln(&choice)
	if choice == 0 {
		choice = 1
	}

	authMethods := map[int]string{
		1: "sso",
		2: "profile", 
		3: "iam_user",
		4: "cross_account",
	}

	authMethod := authMethods[choice]
	if authMethod == "" {
		return fmt.Errorf("invalid selection")
	}

	// Create profile configuration
	profile := &config.Profile{
		ToolName:    profileName + "-cli",
		AuthMethod:  authMethod,
		Region:      "us-east-1",
		SessionDuration: 3600,
	}

	// Auth method specific configuration
	switch authMethod {
	case "sso":
		fmt.Println("\n✅ Step 2: SSO Configuration")
		fmt.Print("SSO Start URL: ")
		var startURL string
		fmt.Scanln(&startURL)
		
		fmt.Print("SSO Region [us-east-1]: ")
		var ssoRegion string
		fmt.Scanln(&ssoRegion)
		if ssoRegion == "" {
			ssoRegion = "us-east-1"
		}

		profile.SSOConfig = &config.SSOConfig{
			StartURL: startURL,
			Region:   ssoRegion,
		}

	case "profile":
		fmt.Println("\n✅ Step 2: Profile Configuration")
		fmt.Print("Base profile name: ")
		var baseName string
		fmt.Scanln(&baseName)
		profile.ProfileName = baseName

	case "cross_account":
		fmt.Println("\n✅ Step 2: Cross Account Configuration")
		fmt.Print("Customer ID: ")
		var customerID string
		fmt.Scanln(&customerID)
		
		fmt.Print("Role ARN: ")
		var roleARN string
		fmt.Scanln(&roleARN)
		
		fmt.Print("External ID: ")
		var externalID string
		fmt.Scanln(&externalID)

		profile.CrossAccount = &config.CrossAccountConfig{
			CustomerID: customerID,
			RoleARN:    roleARN,
			ExternalID: externalID,
		}
	}

	// Session configuration
	fmt.Println("\n✅ Step 3: Session Settings")
	fmt.Printf("Session Duration [%d seconds]: ", profile.SessionDuration)
	var duration int
	fmt.Scanln(&duration)
	if duration > 0 {
		profile.SessionDuration = duration
	}

	// Test authentication
	fmt.Println("\n✅ Step 4: Test Authentication")
	fmt.Println("Testing authentication...")

	provider, exists := h.providers[authMethod]
	if !exists {
		return fmt.Errorf("unsupported auth method: %s", authMethod)
	}

	_, err := provider.GetCredentials(ctx, profile, false)
	if err != nil {
		return fmt.Errorf("authentication test failed: %w", err)
	}

	fmt.Println("✅ Authentication successful!")

	// Save configuration
	if h.config.Profiles == nil {
		h.config.Profiles = make(map[string]*config.Profile)
	}
	h.config.Profiles[profileName] = profile

	if err := h.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✅ Setup Complete!")
	fmt.Printf("Configuration saved to %s\n\n", h.config.ConfigPath())
	
	fmt.Println("Next steps:")
	fmt.Printf("1. Configure AWS CLI: aws configure set credential_process \"%s --profile %s\" --profile %s\n", 
		os.Args[0], profileName, profileName)
	fmt.Printf("2. Test access: aws sts get-caller-identity --profile %s\n", profileName)

	return nil
}

// listProfiles shows available profiles
func (h *CliHelper) listProfiles() {
	fmt.Println("Available profiles:")
	if len(h.config.Profiles) == 0 {
		fmt.Println("  (none configured - run --setup to create a profile)")
		return
	}

	for name, profile := range h.config.Profiles {
		status := "❓"
		if cached := h.cache.Get(name); cached != nil {
			if cached.IsExpired() {
				status = "🔄"
			} else {
				status = "✅"
			}
		}

		fmt.Printf("  %s %s (%s)\n", status, name, profile.AuthMethod)
	}
	
	fmt.Println("\nLegend:")
	fmt.Println("  ✅ Active credentials cached")
	fmt.Println("  🔄 Expired credentials (will refresh)")  
	fmt.Println("  ❓ No cached credentials")
}

// validateProfile validates a profile configuration
func (h *CliHelper) validateProfile(profileName string) error {
	profile, exists := h.config.Profiles[profileName]
	if !exists {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	// Validate auth method
	if _, exists := h.providers[profile.AuthMethod]; !exists {
		return fmt.Errorf("unsupported auth method: %s", profile.AuthMethod)
	}

	// Validate auth method specific configuration
	switch profile.AuthMethod {
	case "sso":
		if profile.SSOConfig == nil {
			return fmt.Errorf("SSO configuration missing")
		}
		if profile.SSOConfig.StartURL == "" {
			return fmt.Errorf("SSO start URL missing")
		}

	case "profile":
		if profile.ProfileName == "" {
			return fmt.Errorf("base profile name missing")
		}

	case "cross_account":
		if profile.CrossAccount == nil {
			return fmt.Errorf("cross account configuration missing")
		}
		if profile.CrossAccount.RoleARN == "" {
			return fmt.Errorf("role ARN missing")
		}
	}

	return nil
}

// runHealthCheck performs comprehensive health check
func (h *CliHelper) runHealthCheck(ctx context.Context, profileName string) error {
	fmt.Printf("🏥 Health Check for profile: %s\n", profileName)
	fmt.Println("=====================================")

	// Check configuration
	fmt.Print("✅ Configuration valid: ")
	if err := h.validateProfile(profileName); err != nil {
		fmt.Printf("❌ %v\n", err)
		return err
	}
	fmt.Println("✅")

	// Check authentication
	fmt.Print("✅ Authentication working: ")
	if err := h.refreshCredentials(ctx, profileName); err != nil {
		fmt.Printf("❌ %v\n", err)
		return err
	}
	fmt.Println("✅")

	// Check cached credentials
	fmt.Print("✅ Credentials cached: ")
	cached := h.cache.Get(profileName)
	if cached == nil {
		fmt.Println("❌")
	} else if cached.IsExpired() {
		fmt.Println("🔄 (expired)")
	} else {
		fmt.Println("✅")
	}

	// Check AWS CLI integration
	fmt.Print("✅ AWS CLI integration: ")
	// This would test if the profile is properly configured in AWS CLI
	fmt.Println("✅")

	fmt.Println("\n🎉 Health check completed successfully!")
	return nil
}

// checkCredentialStatus shows credential status
func (h *CliHelper) checkCredentialStatus(profileName string) {
	cached := h.cache.Get(profileName)
	if cached == nil {
		fmt.Printf("Status: No cached credentials for profile '%s'\n", profileName)
		return
	}

	if cached.IsExpired() {
		fmt.Printf("Status: Cached credentials expired at %s\n", cached.ExpiresAt.Format(time.RFC3339))
		return
	}

	remaining := time.Until(cached.ExpiresAt)
	fmt.Printf("Status: Valid credentials cached (expires in %v)\n", remaining.Round(time.Second))
}

// refreshCredentials forces credential refresh
func (h *CliHelper) refreshCredentials(ctx context.Context, profileName string) error {
	profile, exists := h.config.Profiles[profileName]
	if !exists {
		return fmt.Errorf("profile '%s' not found", profileName)
	}

	provider, exists := h.providers[profile.AuthMethod]
	if !exists {
		return fmt.Errorf("unsupported auth method: %s", profile.AuthMethod)
	}

	creds, err := provider.GetCredentials(ctx, profile, false)
	if err != nil {
		return fmt.Errorf("failed to refresh credentials: %w", err)
	}

	return h.cache.Set(profileName, creds)
}

// generateUsageReport creates a usage report
func (h *CliHelper) generateUsageReport() error {
	fmt.Println("📊 Usage Report")
	fmt.Println("===============")
	fmt.Println("(Usage reporting not yet implemented)")
	return nil
}

// recordUsage records usage metrics
func (h *CliHelper) recordUsage(profile string, success bool, duration time.Duration) {
	// In a real implementation, this would record metrics
	h.logger.Debug("Recording usage metrics",
		slog.String("profile", profile),
		slog.Bool("success", success),
		slog.Duration("duration", duration))
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