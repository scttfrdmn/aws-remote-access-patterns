// Package cmd implements CLI commands for the advanced CLI tool example
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/config"
)

// NewRootCommand creates the root command for the CLI tool
func NewRootCommand(ctx context.Context, cfg *config.Config, version, gitCommit, buildTime string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "datatool",
		Short: "Advanced Data Platform CLI - AWS Remote Access Patterns Example",
		Long: `
DataTool is a comprehensive CLI application demonstrating advanced external tool
authentication patterns using AWS Remote Access Patterns.

This tool showcases:
- Multiple authentication methods (SSO, profiles, interactive setup)
- Secure credential caching and management
- Rich interactive UI with progress indicators
- Comprehensive AWS service integration
- Production-ready error handling and logging

Examples:
  datatool setup                 # Interactive AWS authentication setup
  datatool auth status          # Check authentication status  
  datatool s3 list              # List S3 buckets with rich output
  datatool ec2 instances        # List EC2 instances with filtering
  datatool data sync            # Sync data between environments
  datatool config show         # Show current configuration`,
		
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Apply global flags to config
			if viper.GetBool("debug") {
				cfg.Debug = true
			}
			if viper.GetBool("quiet") {
				cfg.Quiet = true
			}
			if region := viper.GetString("region"); region != "" {
				cfg.AWSRegion = region
			}
			if profile := viper.GetString("profile"); profile != "" {
				cfg.AWSProfile = profile
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().Bool("quiet", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().StringP("region", "r", "", "AWS region to use")
	rootCmd.PersistentFlags().StringP("profile", "p", "", "AWS profile to use")
	rootCmd.PersistentFlags().String("config", "", "Config file (default is $HOME/.datatool/config.yaml)")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output")

	// Bind flags to viper
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))

	// Add subcommands
	rootCmd.AddCommand(newVersionCommand(version, gitCommit, buildTime))
	rootCmd.AddCommand(newSetupCommand(ctx, cfg))
	rootCmd.AddCommand(newAuthCommand(ctx, cfg))
	rootCmd.AddCommand(newConfigCommand(ctx, cfg))
	rootCmd.AddCommand(newS3Command(ctx, cfg))
	rootCmd.AddCommand(newEC2Command(ctx, cfg))
	rootCmd.AddCommand(newDataCommand(ctx, cfg))
	rootCmd.AddCommand(newCompletionCommand())

	return rootCmd
}

// newVersionCommand creates the version command
func newVersionCommand(version, gitCommit, buildTime string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("DataTool CLI\n")
			fmt.Printf("Version:    %s\n", version)
			fmt.Printf("Git Commit: %s\n", gitCommit)
			fmt.Printf("Built:      %s\n", buildTime)
			fmt.Printf("Platform:   %s\n", fmt.Sprintf("%s/%s", 
				os.Getenv("GOOS"), os.Getenv("GOARCH")))
		},
	}
}

// newCompletionCommand creates the completion command
func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:
  $ source <(datatool completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ datatool completion bash > /etc/bash_completion.d/datatool
  # macOS:
  $ datatool completion bash > /usr/local/etc/bash_completion.d/datatool

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ datatool completion zsh > "${fpath[1]}/_datatool"

  # You will need to start a new shell for this setup to take effect.

fish:
  $ datatool completion fish | source

  # To load completions for each session, execute once:
  $ datatool completion fish > ~/.config/fish/completions/datatool.fish

PowerShell:
  PS> datatool completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> datatool completion powershell > datatool.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
}