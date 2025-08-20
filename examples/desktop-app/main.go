// Package main implements a desktop application example demonstrating
// external tool authentication patterns with a web UI interface
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/pkg/browser"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/desktop-app/internal/auth"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/desktop-app/internal/config"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/desktop-app/internal/ui"
)

// Version information - should be set during build
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

//go:embed web/static/* web/templates/*
var webFiles embed.FS

func main() {
	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Received shutdown signal")
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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

	slog.Info("Starting AWS Desktop App",
		slog.String("version", Version),
		slog.String("platform", runtime.GOOS),
		slog.String("arch", runtime.GOARCH))

	// Create app instance
	app := &DesktopApp{
		config:   cfg,
		webFiles: webFiles,
		logger:   logger,
	}

	// Start the application
	if err := app.Start(ctx); err != nil {
		slog.Error("Application failed to start", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Application shutting down...")
}

// DesktopApp represents the main desktop application
type DesktopApp struct {
	config   *config.Config
	webFiles embed.FS
	logger   *slog.Logger
	server   *http.Server
	authMgr  *auth.Manager
}

// Start starts the desktop application
func (app *DesktopApp) Start(ctx context.Context) error {
	// Initialize authentication manager
	authMgr, err := auth.NewManager(app.config)
	if err != nil {
		return fmt.Errorf("failed to create auth manager: %w", err)
	}
	app.authMgr = authMgr

	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	// Create web UI handler
	uiHandler := ui.NewHandler(app.config, app.authMgr, app.webFiles)

	// Setup HTTP server
	mux := http.NewServeMux()
	
	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(app.webFiles)))
	
	// API endpoints
	mux.HandleFunc("/api/status", uiHandler.HandleStatus)
	mux.HandleFunc("/api/auth/status", uiHandler.HandleAuthStatus)
	mux.HandleFunc("/api/auth/setup", uiHandler.HandleAuthSetup)
	mux.HandleFunc("/api/auth/test", uiHandler.HandleAuthTest)
	mux.HandleFunc("/api/auth/clear", uiHandler.HandleAuthClear)
	mux.HandleFunc("/api/s3/buckets", uiHandler.HandleS3Buckets)
	mux.HandleFunc("/api/ec2/instances", uiHandler.HandleEC2Instances)
	mux.HandleFunc("/api/config", uiHandler.HandleConfig)
	
	// Main UI
	mux.HandleFunc("/", uiHandler.HandleHome)
	mux.HandleFunc("/setup", uiHandler.HandleSetupPage)
	mux.HandleFunc("/dashboard", uiHandler.HandleDashboard)

	app.server = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		app.logger.Info("Starting web server", slog.String("addr", app.server.Addr))
		serverErrors <- app.server.ListenAndServe()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Open browser
	url := fmt.Sprintf("http://%s", app.server.Addr)
	app.logger.Info("Opening browser", slog.String("url", url))
	
	if err := browser.OpenURL(url); err != nil {
		app.logger.Warn("Failed to open browser", slog.String("error", err.Error()))
		fmt.Printf("\nAWS Desktop App is running!\n")
		fmt.Printf("Open your browser to: %s\n\n", url)
	}

	// Handle server errors or context cancellation
	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		app.logger.Info("Shutting down web server")
		
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := app.server.Shutdown(shutdownCtx); err != nil {
			app.logger.Error("Server shutdown error", slog.String("error", err.Error()))
			return err
		}
	}

	return nil
}

// findAvailablePort finds an available port starting from 8080
func findAvailablePort() (int, error) {
	for port := 8080; port < 8100; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found")
}