// Package cmd implements authentication-related commands
package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/auth"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/ui"
)

// newAuthCommand creates the auth command with subcommands
func newAuthCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication management commands",
		Long: `Authentication management for DataTool CLI.

These commands help you manage AWS authentication, check status,
and troubleshoot authentication issues.

Examples:
  datatool auth status              # Show current authentication status
  datatool auth test                # Test current authentication
  datatool auth refresh             # Refresh cached credentials
  datatool auth clear               # Clear cached credentials`,
	}

	cmd.AddCommand(newAuthStatusCommand(ctx, cfg))
	cmd.AddCommand(newAuthTestCommand(ctx, cfg))
	cmd.AddCommand(newAuthRefreshCommand(ctx, cfg))
	cmd.AddCommand(newAuthClearCommand(ctx, cfg))

	return cmd
}

// newAuthStatusCommand creates the auth status command
func newAuthStatusCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	var (
		outputFormat string
		detailed     bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long: `Show the current authentication status including:
- Whether authentication is configured
- Authentication method in use
- AWS region
- Current AWS identity (if active)

Examples:
  datatool auth status              # Show basic status
  datatool auth status --detailed   # Show detailed information
  datatool auth status --format json # JSON output`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthStatus(ctx, cfg, outputFormat, detailed)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table, json, yaml)")
	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "Show detailed information")

	return cmd
}

// newAuthTestCommand creates the auth test command
func newAuthTestCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test current authentication",
		Long: `Test the current authentication configuration by attempting
to authenticate with AWS and retrieve the caller identity.

This command will:
- Verify authentication configuration
- Attempt to get AWS credentials
- Call AWS STS GetCallerIdentity
- Display the results

Examples:
  datatool auth test                # Test authentication`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthTest(ctx, cfg)
		},
	}
}

// newAuthRefreshCommand creates the auth refresh command
func newAuthRefreshCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh cached credentials",
		Long: `Force a refresh of cached AWS credentials.

This command will:
- Clear any cached credentials
- Re-authenticate with the configured method
- Cache fresh credentials

Use this command if you're getting authentication errors
or if your credentials have been updated externally.

Examples:
  datatool auth refresh             # Refresh credentials`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthRefresh(ctx, cfg)
		},
	}
}

// newAuthClearCommand creates the auth clear command
func newAuthClearCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear cached credentials and configuration",
		Long: `Clear all cached credentials and optionally reset authentication configuration.

This command will:
- Remove all cached credentials
- Optionally reset authentication configuration
- Require re-authentication on next use

WARNING: This will require you to re-authenticate the next time you use the CLI.

Examples:
  datatool auth clear               # Clear with confirmation
  datatool auth clear --force      # Clear without confirmation`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthClear(ctx, cfg, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Clear without confirmation")

	return cmd
}

func runAuthStatus(ctx context.Context, cfg *config.Config, outputFormat string, detailed bool) error {
	authManager, err := auth.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	status, err := authManager.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth status: %w", err)
	}

	// Format output
	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(status)

	case "yaml":
		// Simple YAML output (could use yaml library for better formatting)
		fmt.Printf("configured: %t\n", status.Configured)
		fmt.Printf("active: %t\n", status.Active)
		fmt.Printf("method: %s\n", status.Method)
		fmt.Printf("region: %s\n", status.Region)
		if status.Identity != nil {
			fmt.Printf("identity:\n")
			fmt.Printf("  user_id: %s\n", status.Identity.UserID)
			fmt.Printf("  account: %s\n", status.Identity.Account)
			fmt.Printf("  arn: %s\n", status.Identity.ARN)
		}

	default: // table format
		uiHandler := ui.NewHandler(true, !cfg.NoColor)
		
		if !status.Configured {
			uiHandler.Warning("Authentication is not configured")
			fmt.Println("\nRun 'datatool setup' to configure authentication.")
			return nil
		}

		if status.Active {
			uiHandler.Success("Authentication is active")
		} else {
			uiHandler.Warning("Authentication is configured but not active")
		}

		fmt.Printf("\nAuthentication Details:\n")
		fmt.Printf("  Method: %s\n", status.Method)
		fmt.Printf("  Region: %s\n", status.Region)

		if status.Identity != nil {
			fmt.Printf("\nAWS Identity:\n")
			fmt.Printf("  User ID: %s\n", status.Identity.UserID)
			fmt.Printf("  Account: %s\n", status.Identity.Account)
			fmt.Printf("  ARN: %s\n", status.Identity.ARN)
		}

		if detailed && status.Configured {
			fmt.Printf("\nConfiguration:\n")
			fmt.Printf("  Session Duration: %ds\n", cfg.Auth.SessionDuration)
			fmt.Printf("  Cache Enabled: %t\n", cfg.Auth.CacheEnabled)
			
			if cfg.Auth.Method == "sso" {
				fmt.Printf("  SSO Start URL: %s\n", cfg.Auth.SSO.StartURL)
				fmt.Printf("  SSO Region: %s\n", cfg.Auth.SSO.Region)
			} else if cfg.Auth.Method == "profile" {
				fmt.Printf("  Profile Name: %s\n", cfg.Auth.Profile.Name)
			}
		}
	}

	return nil
}

func runAuthTest(ctx context.Context, cfg *config.Config) error {
	uiHandler := ui.NewHandler(true, !cfg.NoColor)

	authManager, err := auth.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	if !authManager.IsConfigured() {
		uiHandler.Error("Authentication is not configured")
		fmt.Println("\nRun 'datatool setup' to configure authentication.")
		return fmt.Errorf("authentication not configured")
	}

	uiHandler.ShowStep("Testing authentication...")

	err = authManager.TestAuthentication(ctx)
	if err != nil {
		uiHandler.Error(fmt.Sprintf("Authentication test failed: %v", err))
		fmt.Println("\nTroubleshooting tips:")
		fmt.Println("  • Run 'datatool auth refresh' to refresh credentials")
		fmt.Println("  • Run 'datatool setup --force' to reconfigure authentication")
		fmt.Println("  • Check your AWS permissions and network connectivity")
		return err
	}

	uiHandler.Success("Authentication test successful!")
	
	// Show current identity
	status, err := authManager.GetStatus(ctx)
	if err == nil && status.Identity != nil {
		fmt.Printf("\nAuthenticated as:\n")
		fmt.Printf("  Account: %s\n", status.Identity.Account)
		fmt.Printf("  ARN: %s\n", status.Identity.ARN)
	}

	return nil
}

func runAuthRefresh(ctx context.Context, cfg *config.Config) error {
	uiHandler := ui.NewHandler(true, !cfg.NoColor)

	authManager, err := auth.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	if !authManager.IsConfigured() {
		uiHandler.Error("Authentication is not configured")
		fmt.Println("\nRun 'datatool setup' to configure authentication.")
		return fmt.Errorf("authentication not configured")
	}

	uiHandler.ShowStep("Refreshing credentials...")

	err = authManager.Refresh(ctx)
	if err != nil {
		uiHandler.Error(fmt.Sprintf("Failed to refresh credentials: %v", err))
		return err
	}

	uiHandler.Success("Credentials refreshed successfully!")
	return nil
}

func runAuthClear(ctx context.Context, cfg *config.Config, force bool) error {
	uiHandler := ui.NewHandler(true, !cfg.NoColor)

	if !force {
		if !uiHandler.Confirm("This will clear all cached credentials and require re-authentication. Continue?") {
			uiHandler.ShowInfo("Operation cancelled")
			return nil
		}
	}

	// Clear auth configuration
	cfg.Auth = config.AuthConfig{
		SessionDuration: 3600,
		CacheEnabled:    true,
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	uiHandler.Success("Authentication configuration cleared")
	fmt.Println("\nRun 'datatool setup' to configure authentication again.")

	return nil
}