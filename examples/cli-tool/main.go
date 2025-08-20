// Package main implements a comprehensive CLI tool demonstrating advanced
// external tool authentication patterns with AWS Remote Access Patterns
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/cmd"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/cli-tool/internal/config"
)

// Version information - should be set during build
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup structured logging
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Create root command
	rootCmd := cmd.NewRootCommand(ctx, cfg, Version, GitCommit, BuildTime)

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		if cfg.Debug {
			logger.Error("Command execution failed", slog.String("error", err.Error()))
		}
		os.Exit(1)
	}
}