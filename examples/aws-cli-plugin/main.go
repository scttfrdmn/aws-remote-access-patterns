// Package main provides a credential provider plugin for AWS CLI
// This plugin integrates with the aws-remote-access-patterns authentication system
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-plugin/internal/auth"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/aws-cli-plugin/internal/config"
)

// AWSCredentialResponse represents the credential response format expected by AWS CLI
type AWSCredentialResponse struct {
	Version         int    `json:"Version"`
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken,omitempty"`
	Expiration      string `json:"Expiration,omitempty"`
}

// PluginMetadata provides information about the plugin
type PluginMetadata struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Homepage    string `json:"homepage"`
}

const (
	PluginVersion = "1.0.0"
	PluginName    = "aws-remote-access-patterns-plugin"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))
	slog.SetDefault(logger)

	// Check command line arguments
	if len(os.Args) < 2 {
		logger.Error("Usage: aws-cli-plugin <command> [args...]")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "get-credentials":
		handleGetCredentials()
	case "info":
		handleInfo()
	case "setup":
		handleSetup()
	case "test":
		handleTest()
	case "clear":
		handleClear()
	case "version":
		handleVersion()
	case "help":
		handleHelp()
	default:
		logger.Error("Unknown command", slog.String("command", command))
		handleHelp()
		os.Exit(1)
	}
}

// handleGetCredentials implements the AWS credential provider interface
func handleGetCredentials() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := slog.Default()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		outputError("Failed to load configuration: " + err.Error())
		return
	}

	// Create authentication manager
	authMgr, err := auth.NewManager(cfg)
	if err != nil {
		logger.Error("Failed to create auth manager", slog.String("error", err.Error()))
		outputError("Failed to create authentication manager: " + err.Error())
		return
	}

	// Get AWS credentials
	awsConfig, err := authMgr.GetAWSConfig(ctx)
	if err != nil {
		logger.Error("Failed to get AWS config", slog.String("error", err.Error()))
		outputError("Failed to get AWS credentials: " + err.Error())
		return
	}

	// Retrieve credentials from the config
	creds, err := awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		logger.Error("Failed to retrieve credentials", slog.String("error", err.Error()))
		outputError("Failed to retrieve credentials: " + err.Error())
		return
	}

	// Format response for AWS CLI
	response := AWSCredentialResponse{
		Version:         1,
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
	}

	// Add expiration if available
	if !creds.Expires.IsZero() {
		response.Expiration = creds.Expires.Format(time.RFC3339)
	}

	// Output JSON response
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(response); err != nil {
		logger.Error("Failed to encode response", slog.String("error", err.Error()))
		outputError("Failed to encode credential response: " + err.Error())
		return
	}

	logger.Info("Credentials provided successfully",
		slog.String("access_key", creds.AccessKeyID[:10]+"..."),
		slog.Bool("has_session_token", creds.SessionToken != ""),
		slog.Time("expires", creds.Expires))
}

// handleInfo provides information about the plugin
func handleInfo() {
	metadata := PluginMetadata{
		Name:        PluginName,
		Version:     PluginVersion,
		Description: "AWS CLI credential provider plugin for remote access patterns",
		Author:      "AWS Remote Access Patterns Project",
		Homepage:    "https://github.com/example/aws-remote-access-patterns",
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		slog.Default().Error("Failed to encode info response", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// handleSetup guides the user through plugin setup
func handleSetup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger := slog.Default()
	logger.Info("Starting plugin setup")

	// Create configuration manager
	cfg := config.NewDefault()

	// Interactive setup
	if err := runInteractiveSetup(cfg); err != nil {
		logger.Error("Setup failed", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		os.Exit(1)
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		logger.Error("Failed to save configuration", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to save configuration: %v\n", err)
		os.Exit(1)
	}

	// Test the configuration
	authMgr, err := auth.NewManager(cfg)
	if err != nil {
		logger.Error("Failed to create auth manager", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to create authentication manager: %v\n", err)
		os.Exit(1)
	}

	if err := authMgr.TestConnection(ctx); err != nil {
		logger.Error("Authentication test failed", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Authentication test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Plugin setup completed successfully!")
	fmt.Println()
	fmt.Println("To use with AWS CLI, add this to your AWS config file (~/.aws/config):")
	fmt.Println()
	fmt.Printf("[profile %s]\n", cfg.ProfileName)
	fmt.Printf("credential_process = %s get-credentials\n", os.Args[0])
	fmt.Println()
	fmt.Println("Then use: aws --profile " + cfg.ProfileName + " sts get-caller-identity")
}

// handleTest tests the current configuration
func handleTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := slog.Default()
	logger.Info("Testing plugin configuration")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create authentication manager
	authMgr, err := auth.NewManager(cfg)
	if err != nil {
		logger.Error("Failed to create auth manager", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to create authentication manager: %v\n", err)
		os.Exit(1)
	}

	// Test authentication
	fmt.Println("🔄 Testing authentication...")
	if err := authMgr.TestConnection(ctx); err != nil {
		logger.Error("Authentication test failed", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "❌ Authentication test failed: %v\n", err)
		os.Exit(1)
	}

	// Get credentials to verify they work
	fmt.Println("🔄 Testing credential retrieval...")
	awsConfig, err := authMgr.GetAWSConfig(ctx)
	if err != nil {
		logger.Error("Failed to get AWS config", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "❌ Failed to get AWS credentials: %v\n", err)
		os.Exit(1)
	}

	creds, err := awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		logger.Error("Failed to retrieve credentials", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "❌ Failed to retrieve credentials: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Plugin test completed successfully!")
	fmt.Printf("   Access Key: %s...\n", creds.AccessKeyID[:10])
	fmt.Printf("   Has Session Token: %v\n", creds.SessionToken != "")
	if !creds.Expires.IsZero() {
		fmt.Printf("   Expires: %v\n", creds.Expires.Format(time.RFC3339))
	}
}

// handleClear clears the plugin configuration
func handleClear() {
	logger := slog.Default()
	logger.Info("Clearing plugin configuration")

	if err := config.Clear(); err != nil {
		logger.Error("Failed to clear configuration", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to clear configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Plugin configuration cleared successfully!")
}

// handleVersion displays version information
func handleVersion() {
	fmt.Printf("%s version %s\n", PluginName, PluginVersion)
}

// handleHelp displays help information
func handleHelp() {
	fmt.Printf(`%s - AWS CLI credential provider plugin

Usage: %s <command> [options]

Commands:
  get-credentials    Retrieve AWS credentials (used by AWS CLI)
  setup             Interactive setup of the plugin
  test              Test the current configuration
  clear             Clear the plugin configuration
  info              Display plugin information (JSON format)
  version           Display version information
  help              Display this help message

Environment Variables:
  AWS_REMOTE_ACCESS_DEBUG    Enable debug logging (true/false)
  AWS_REMOTE_ACCESS_CONFIG   Override config file location

Examples:
  # Setup the plugin
  %s setup

  # Test configuration
  %s test

  # Use with AWS CLI (after setup)
  aws --profile myprofile sts get-caller-identity

  # Direct credential retrieval (for testing)
  %s get-credentials

Configuration:
  The plugin stores configuration in ~/.aws-remote-access-patterns/plugin-config.json
  
  AWS CLI integration requires adding this to ~/.aws/config:
  [profile myprofile]
  credential_process = %s get-credentials

For more information, visit:
https://github.com/example/aws-remote-access-patterns
`, PluginName, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

// runInteractiveSetup guides the user through configuration
func runInteractiveSetup(cfg *config.Config) error {
	fmt.Println("🔧 AWS Remote Access Patterns - Plugin Setup")
	fmt.Println("=" + string(make([]byte, 45, 45)))
	fmt.Println()

	// Profile name
	fmt.Print("Enter AWS profile name [remote-access]: ")
	var profileName string
	fmt.Scanln(&profileName)
	if profileName == "" {
		profileName = "remote-access"
	}
	cfg.ProfileName = profileName

	// Authentication method
	fmt.Println()
	fmt.Println("Select authentication method:")
	fmt.Println("1. AWS SSO")
	fmt.Println("2. Cross-account role assumption")
	fmt.Println("3. Interactive authentication")
	fmt.Print("Choose [1-3]: ")
	
	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		cfg.AuthMethod = "sso"
		fmt.Print("Enter SSO start URL: ")
		fmt.Scanln(&cfg.SSOStartURL)
	case "2":
		cfg.AuthMethod = "cross-account"
		fmt.Print("Enter target role ARN: ")
		fmt.Scanln(&cfg.CrossAccount.RoleARN)
		fmt.Print("Enter external ID (optional): ")
		fmt.Scanln(&cfg.CrossAccount.ExternalID)
	case "3":
		cfg.AuthMethod = "interactive"
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}

	// AWS Region
	fmt.Print("Enter AWS region [us-east-1]: ")
	var region string
	fmt.Scanln(&region)
	if region == "" {
		region = "us-east-1"
	}
	cfg.AWSRegion = region

	// Session duration
	fmt.Print("Enter session duration in seconds [3600]: ")
	var duration string
	fmt.Scanln(&duration)
	if duration == "" {
		cfg.SessionDuration = 3600
	} else {
		fmt.Sscanf(duration, "%d", &cfg.SessionDuration)
	}

	fmt.Println()
	fmt.Println("Configuration summary:")
	fmt.Printf("  Profile: %s\n", cfg.ProfileName)
	fmt.Printf("  Method: %s\n", cfg.AuthMethod)
	fmt.Printf("  Region: %s\n", cfg.AWSRegion)
	fmt.Printf("  Duration: %d seconds\n", cfg.SessionDuration)

	return nil
}

// outputError outputs an error in the format expected by AWS CLI
func outputError(message string) {
	errorResponse := map[string]interface{}{
		"error": message,
	}
	
	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(errorResponse)
}

// getLogLevel returns the appropriate log level based on environment
func getLogLevel() slog.Level {
	if os.Getenv("AWS_REMOTE_ACCESS_DEBUG") == "true" {
		return slog.LevelDebug
	}
	return slog.LevelWarn
}