// Package cmd implements configuration management commands
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/ui"
)

// newConfigCommand creates the config command with subcommands
func newConfigCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
		Long: `Configuration management for DataTool CLI.

These commands help you view, modify, and validate your DataTool configuration.

Examples:
  datatool config show             # Show current configuration
  datatool config show --format json # Show config as JSON
  datatool config set cli.output_format table # Set a config value
  datatool config validate         # Validate configuration`,
	}

	cmd.AddCommand(newConfigShowCommand(ctx, cfg))
	cmd.AddCommand(newConfigSetCommand(ctx, cfg))
	cmd.AddCommand(newConfigValidateCommand(ctx, cfg))
	cmd.AddCommand(newConfigResetCommand(ctx, cfg))

	return cmd
}

// newConfigShowCommand creates the config show command
func newConfigShowCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	var (
		outputFormat string
		section      string
	)

	cmd := &cobra.Command{
		Use:   "show [section]",
		Short: "Show current configuration",
		Long: `Show the current DataTool configuration.

You can specify a section to show only that part of the configuration:
- auth: Authentication settings  
- cli: CLI behavior settings
- data: Data processing settings

Examples:
  datatool config show             # Show all configuration
  datatool config show auth       # Show only auth configuration
  datatool config show --format json # Show as JSON
  datatool config show --format yaml # Show as YAML`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				section = args[0]
			}
			return runConfigShow(cfg, outputFormat, section)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "yaml", "Output format (yaml, json, table)")
	
	return cmd
}

// newConfigSetCommand creates the config set command  
func newConfigSetCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value using dot notation.

Available configuration keys:
  auth.method                    # Authentication method
  auth.region                    # AWS region for auth
  auth.session_duration          # Session duration in seconds
  auth.cache_enabled             # Enable credential caching
  cli.output_format              # Default output format
  cli.page_size                  # Default page size
  cli.confirm_actions            # Confirm destructive actions
  data.default_bucket            # Default S3 bucket
  data.max_concurrency          # Max concurrent operations

Examples:
  datatool config set cli.output_format table
  datatool config set auth.session_duration 7200
  datatool config set data.max_concurrency 20`,

		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(cfg, args[0], args[1])
		},
	}
}

// newConfigValidateCommand creates the config validate command
func newConfigValidateCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate current configuration",
		Long: `Validate the current configuration for correctness.

This command will check:
- Configuration file syntax
- Value ranges and constraints
- Authentication configuration
- Required settings

Examples:
  datatool config validate        # Validate configuration`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigValidate(cfg)
		},
	}
}

// newConfigResetCommand creates the config reset command
func newConfigResetCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
	var (
		force   bool
		section string
	)

	cmd := &cobra.Command{
		Use:   "reset [section]",
		Short: "Reset configuration to defaults",
		Long: `Reset configuration to default values.

You can reset the entire configuration or just a specific section:
- auth: Authentication settings
- cli: CLI behavior settings  
- data: Data processing settings

WARNING: This will permanently remove your current configuration!

Examples:
  datatool config reset           # Reset all configuration
  datatool config reset auth     # Reset only auth configuration
  datatool config reset --force  # Reset without confirmation`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				section = args[0]
			}
			return runConfigReset(cfg, section, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Reset without confirmation")

	return cmd
}

func runConfigShow(cfg *config.Config, outputFormat, section string) error {
	var data interface{} = cfg

	// Filter to specific section if requested
	switch section {
	case "auth":
		data = cfg.Auth
	case "cli":
		data = cfg.CLI
	case "data":  
		data = cfg.Data
	case "":
		// Show all
	default:
		return fmt.Errorf("unknown config section: %s", section)
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)

	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		encoder.SetIndent(2)
		return encoder.Encode(data)

	case "table":
		return showConfigTable(cfg, section)

	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func showConfigTable(cfg *config.Config, section string) error {
	uiHandler := ui.NewHandler(true, !cfg.NoColor)

	if section == "" || section == "auth" {
		fmt.Println("\n🔐 Authentication Configuration:")
		headers := []string{"Setting", "Value"}
		rows := [][]string{
			{"Method", cfg.Auth.Method},
			{"Region", cfg.Auth.Region},
			{"Session Duration", fmt.Sprintf("%ds", cfg.Auth.SessionDuration)},
			{"Cache Enabled", fmt.Sprintf("%t", cfg.Auth.CacheEnabled)},
		}

		if cfg.Auth.Method == "sso" {
			rows = append(rows, []string{"SSO Start URL", cfg.Auth.SSO.StartURL})
			rows = append(rows, []string{"SSO Region", cfg.Auth.SSO.Region})
		} else if cfg.Auth.Method == "profile" {
			rows = append(rows, []string{"Profile Name", cfg.Auth.Profile.Name})
		}

		uiHandler.ShowTable(headers, rows)
	}

	if section == "" || section == "cli" {
		fmt.Println("\n⚙️  CLI Configuration:")
		headers := []string{"Setting", "Value"}
		rows := [][]string{
			{"Output Format", cfg.CLI.OutputFormat},
			{"Table Style", cfg.CLI.TableStyle},
			{"Page Size", fmt.Sprintf("%d", cfg.CLI.PageSize)},
			{"Confirm Actions", fmt.Sprintf("%t", cfg.CLI.ConfirmActions)},
			{"Show Progress", fmt.Sprintf("%t", cfg.CLI.ShowProgress)},
			{"Auto Pagination", fmt.Sprintf("%t", cfg.CLI.AutoPagination)},
		}
		uiHandler.ShowTable(headers, rows)
	}

	if section == "" || section == "data" {
		fmt.Println("\n📊 Data Configuration:")
		headers := []string{"Setting", "Value"}
		rows := [][]string{
			{"Default Bucket", cfg.Data.DefaultBucket},
			{"Temp Directory", cfg.Data.TemporaryDirectory},
			{"Max Concurrency", fmt.Sprintf("%d", cfg.Data.MaxConcurrency)},
			{"Chunk Size", fmt.Sprintf("%d bytes", cfg.Data.ChunkSize)},
		}
		uiHandler.ShowTable(headers, rows)

		if len(cfg.Data.Environments) > 0 {
			fmt.Println("\n🌍 Environment Mappings:")
			headers = []string{"Environment", "Bucket"}
			rows = [][]string{}
			for env, bucket := range cfg.Data.Environments {
				rows = append(rows, []string{env, bucket})
			}
			uiHandler.ShowTable(headers, rows)
		}
	}

	return nil
}

func runConfigSet(cfg *config.Config, key, value string) error {
	uiHandler := ui.NewHandler(true, !cfg.NoColor)

	// Parse the key and set the value
	switch key {
	case "auth.method":
		cfg.Auth.Method = value
	case "auth.region":
		cfg.Auth.Region = value
	case "auth.session_duration":
		var duration int
		if _, err := fmt.Sscanf(value, "%d", &duration); err != nil {
			return fmt.Errorf("invalid session duration: %s", value)
		}
		cfg.Auth.SessionDuration = duration
	case "auth.cache_enabled":
		var enabled bool
		if _, err := fmt.Sscanf(value, "%t", &enabled); err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		cfg.Auth.CacheEnabled = enabled
	case "cli.output_format":
		cfg.CLI.OutputFormat = value
	case "cli.page_size":
		var size int
		if _, err := fmt.Sscanf(value, "%d", &size); err != nil {
			return fmt.Errorf("invalid page size: %s", value)
		}
		cfg.CLI.PageSize = size
	case "cli.confirm_actions":
		var confirm bool
		if _, err := fmt.Sscanf(value, "%t", &confirm); err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		cfg.CLI.ConfirmActions = confirm
	case "data.default_bucket":
		cfg.Data.DefaultBucket = value
	case "data.max_concurrency":
		var concurrency int
		if _, err := fmt.Sscanf(value, "%d", &concurrency); err != nil {
			return fmt.Errorf("invalid concurrency value: %s", value)
		}
		cfg.Data.MaxConcurrency = concurrency
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Save the configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	uiHandler.Success(fmt.Sprintf("Set %s = %s", key, value))
	return nil
}

func runConfigValidate(cfg *config.Config) error {
	uiHandler := ui.NewHandler(true, !cfg.NoColor)

	uiHandler.ShowStep("Validating configuration...")

	if err := cfg.Validate(); err != nil {
		uiHandler.Error(fmt.Sprintf("Configuration validation failed: %v", err))
		return err
	}

	uiHandler.Success("Configuration is valid!")

	// Show any warnings or recommendations
	if cfg.Auth.Method == "" {
		uiHandler.Warning("No authentication method configured. Run 'datatool setup' to configure authentication.")
	}

	if cfg.Data.DefaultBucket == "" {
		uiHandler.ShowInfo("No default S3 bucket configured. Some data commands may require explicit bucket specification.")
	}

	return nil
}

func runConfigReset(cfg *config.Config, section string, force bool) error {
	uiHandler := ui.NewHandler(true, !cfg.NoColor)

	// Confirmation
	if !force {
		message := "This will reset configuration to defaults"
		if section != "" {
			message = fmt.Sprintf("This will reset the '%s' configuration section to defaults", section)
		}
		
		if !uiHandler.Confirm(message + ". Continue?") {
			uiHandler.ShowInfo("Operation cancelled")
			return nil
		}
	}

	// Reset configuration
	defaultCfg := config.DefaultConfig()
	
	switch section {
	case "auth":
		cfg.Auth = defaultCfg.Auth
	case "cli":
		cfg.CLI = defaultCfg.CLI
	case "data":
		cfg.Data = defaultCfg.Data
	case "":
		// Reset everything except paths
		configDir := cfg.ConfigDir
		*cfg = *defaultCfg
		cfg.ConfigDir = configDir
	default:
		return fmt.Errorf("unknown config section: %s", section)
	}

	// Save the configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if section != "" {
		uiHandler.Success(fmt.Sprintf("Reset '%s' configuration to defaults", section))
	} else {
		uiHandler.Success("Reset all configuration to defaults")
	}

	return nil
}