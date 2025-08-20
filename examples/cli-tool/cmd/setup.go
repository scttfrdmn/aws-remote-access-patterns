// Package cmd implements the setup command for interactive AWS authentication setup
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/auth"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/ui"
)

// newSetupCommand creates the setup command
func newSetupCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	var (
		force       bool
		authMethod  string
		region      string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive AWS authentication setup",
		Long: `Setup configures AWS authentication for the DataTool CLI.

This command guides you through configuring secure AWS access using one of several methods:
- AWS SSO (recommended for organizations)
- AWS profiles (use existing ~/.aws/credentials)
- Interactive authentication (for first-time users)

The setup process will:
1. Detect existing AWS configurations
2. Guide you through authentication method selection
3. Test the chosen authentication method
4. Cache credentials securely for future use
5. Provide usage instructions

Examples:
  datatool setup                    # Interactive setup wizard
  datatool setup --force           # Force reconfiguration
  datatool setup --method sso      # Use AWS SSO authentication
  datatool setup --region us-west-2 # Set specific AWS region`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(ctx, cfg, setupOptions{
				force:       force,
				authMethod:  authMethod,
				region:      region,
				interactive: interactive,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force reconfiguration even if already configured")
	cmd.Flags().StringVar(&authMethod, "method", "", "Authentication method (sso, profile, interactive)")
	cmd.Flags().StringVar(&region, "region", "", "AWS region to use")
	cmd.Flags().BoolVar(&interactive, "interactive", true, "Use interactive mode")

	return cmd
}

type setupOptions struct {
	force       bool
	authMethod  string
	region      string
	interactive bool
}

func runSetup(ctx context.Context, cfg *config.Config, opts setupOptions) error {
	logger := slog.Default()
	
	// Create UI handler
	uiHandler := ui.NewHandler(!cfg.Quiet, !cfg.NoColor)
	
	// Show welcome message
	if opts.interactive {
		uiHandler.ShowWelcome("DataTool CLI Setup", `
Welcome to DataTool CLI! This setup wizard will help you configure secure AWS authentication.

DataTool uses AWS Remote Access Patterns to provide:
• Temporary credentials (no long-lived access keys)
• Multiple authentication methods
• Secure credential caching
• Automatic token refresh

Let's get started!`)
	}

	// Check if already configured
	if !opts.force {
		if authManager, err := auth.NewManager(cfg); err == nil {
			if authManager.IsConfigured() {
				if !opts.interactive {
					logger.Info("Authentication already configured. Use --force to reconfigure.")
					return nil
				}

				if !uiHandler.Confirm("AWS authentication is already configured. Do you want to reconfigure?") {
					uiHandler.Success("Setup cancelled. Current configuration preserved.")
					return nil
				}
			}
		}
	}

	// Detect existing AWS configurations
	uiHandler.ShowStep("Detecting existing AWS configurations...")
	
	detector := auth.NewConfigDetector()
	existingConfigs, err := detector.DetectConfigurations(ctx)
	if err != nil {
		logger.Warn("Failed to detect existing configurations", slog.String("error", err.Error()))
	}

	// Show detected configurations
	if len(existingConfigs) > 0 && opts.interactive {
		uiHandler.ShowInfo("Found existing AWS configurations:")
		for _, config := range existingConfigs {
			uiHandler.ShowListItem(fmt.Sprintf("%s (%s)", config.Name, config.Type))
		}
		fmt.Println()
	}

	// Determine authentication method
	var selectedMethod string
	if opts.authMethod != "" {
		selectedMethod = opts.authMethod
	} else if opts.interactive {
		selectedMethod = selectAuthenticationMethod(uiHandler, existingConfigs)
	} else {
		// Default to most appropriate method
		if len(existingConfigs) > 0 {
			selectedMethod = existingConfigs[0].Type
		} else {
			selectedMethod = "sso"
		}
	}

	// Configure authentication
	uiHandler.ShowStep("Configuring authentication...")
	
	authManager, err := auth.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}

	setupConfig := &auth.SetupConfig{
		Method:      selectedMethod,
		Region:      opts.region,
		Interactive: opts.interactive,
	}

	if err := authManager.Setup(ctx, setupConfig, uiHandler); err != nil {
		return fmt.Errorf("authentication setup failed: %w", err)
	}

	// Test authentication
	uiHandler.ShowStep("Testing authentication...")
	
	if err := authManager.TestAuthentication(ctx); err != nil {
		uiHandler.Error(fmt.Sprintf("Authentication test failed: %v", err))
		return fmt.Errorf("authentication test failed: %w", err)
	}

	// Save configuration
	uiHandler.ShowStep("Saving configuration...")
	
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Show success message and usage instructions
	uiHandler.Success("Setup completed successfully! 🎉")
	
	if opts.interactive {
		uiHandler.ShowUsageInstructions()
	}

	return nil
}

func selectAuthenticationMethod(uiHandler *ui.Handler, existingConfigs []auth.DetectedConfig) string {
	methods := []ui.SelectOption{
		{
			Value: "sso",
			Label: "AWS SSO",
			Description: "Recommended for organizations using AWS Single Sign-On",
		},
		{
			Value: "profile", 
			Label: "AWS Profile",
			Description: "Use existing AWS profiles from ~/.aws/credentials",
		},
		{
			Value: "interactive",
			Label: "Interactive Setup",
			Description: "Guided setup for first-time users",
		},
	}

	// If existing configurations found, prioritize those methods
	if len(existingConfigs) > 0 {
		for i, method := range methods {
			for _, config := range existingConfigs {
				if method.Value == config.Type {
					methods[i].Label += " ✓"
					methods[i].Description += " (detected)"
					break
				}
			}
		}
	}

	selected, err := uiHandler.Select("Choose authentication method:", methods)
	if err != nil {
		slog.Default().Error("Failed to get user selection", slog.String("error", err.Error()))
		return "sso" // fallback
	}

	return selected
}