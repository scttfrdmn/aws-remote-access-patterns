// Package main implements a complete SaaS service example demonstrating
// cross-account AWS access patterns with a web UI for customer onboarding.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/scttfrdmn/aws-remote-access-patterns/examples/saas-service/internal/handlers"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/saas-service/internal/middleware"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/crossaccount"
)

// Config represents the application configuration
type Config struct {
	Port             string `json:"port" env:"PORT"`
	ServiceName      string `json:"service_name" env:"SERVICE_NAME"`
	ServiceAccountID string `json:"service_account_id" env:"SERVICE_ACCOUNT_ID"`
	TemplateS3Bucket string `json:"template_s3_bucket" env:"TEMPLATE_S3_BUCKET"`
	AWSRegion        string `json:"aws_region" env:"AWS_REGION"`
	Environment      string `json:"environment" env:"ENVIRONMENT"`
	LogLevel         string `json:"log_level" env:"LOG_LEVEL"`
}

// loadConfig loads configuration from environment variables or uses defaults
func loadConfig() *Config {
	return &Config{
		Port:             getEnvOrDefault("PORT", "8080"),
		ServiceName:      getEnvOrDefault("SERVICE_NAME", "MyDataPlatform"),
		ServiceAccountID: getEnvOrDefault("SERVICE_ACCOUNT_ID", "123456789012"),
		TemplateS3Bucket: getEnvOrDefault("TEMPLATE_S3_BUCKET", "mydataplatform-templates"),
		AWSRegion:        getEnvOrDefault("AWS_REGION", "us-east-1"),
		Environment:      getEnvOrDefault("ENVIRONMENT", "development"),
		LogLevel:         getEnvOrDefault("LOG_LEVEL", "info"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Parse command line flags
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		showConfig = flag.Bool("show-config", false, "Show current configuration and exit")
	)
	flag.Parse()

	// Load configuration
	config := loadConfig()
	if *configFile != "" {
		if err := loadConfigFromFile(*configFile, config); err != nil {
			log.Fatalf("Failed to load config file: %v", err)
		}
	}

	if *showConfig {
		configJSON, _ := json.MarshalIndent(config, "", "  ")
		fmt.Printf("Current configuration:\n%s\n", configJSON)
		return
	}

	// Setup structured logging
	logLevel := slog.LevelInfo
	switch config.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting SaaS service",
		slog.String("service", config.ServiceName),
		slog.String("version", "1.0.0"),
		slog.String("port", config.Port),
		slog.String("environment", config.Environment))

	// Initialize cross-account client
	crossAccountConfig := &crossaccount.Config{
		ServiceName:      config.ServiceName,
		ServiceAccountID: config.ServiceAccountID,
		TemplateS3Bucket: config.TemplateS3Bucket,
		DefaultRegion:    config.AWSRegion,
		OngoingPermissions: []crossaccount.Permission{
			{
				Sid:    "S3DataAccess",
				Effect: "Allow",
				Actions: []string{
					"s3:GetObject",
					"s3:PutObject",
					"s3:DeleteObject",
					"s3:ListBucket",
				},
				Resources: []string{
					"arn:aws:s3:::customer-data-*",
					"arn:aws:s3:::customer-data-*/*",
				},
			},
			{
				Sid:    "CloudWatchMetrics",
				Effect: "Allow",
				Actions: []string{
					"cloudwatch:PutMetricData",
				},
				Resources: []string{"*"},
			},
		},
		SetupPermissions: []crossaccount.Permission{
			{
				Sid:    "S3SetupAccess",
				Effect: "Allow",
				Actions: []string{
					"s3:CreateBucket",
					"s3:PutBucketPolicy",
					"s3:PutBucketEncryption",
					"s3:PutBucketPublicAccessBlock",
				},
				Resources: []string{
					"arn:aws:s3:::customer-data-*",
				},
			},
		},
	}

	client, err := crossaccount.New(crossAccountConfig)
	if err != nil {
		slog.Error("Failed to initialize cross-account client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize HTTP handlers
	handlerConfig := &handlers.Config{
		CrossAccountClient: client,
		ServiceName:        config.ServiceName,
		Environment:        config.Environment,
	}

	handler, err := handlers.New(handlerConfig)
	if err != nil {
		slog.Error("Failed to initialize handlers", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Setup HTTP server with middleware
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", handler.HealthCheck)
	mux.HandleFunc("/ready", handler.ReadinessCheck)

	// API endpoints
	mux.HandleFunc("POST /api/customers", handler.CreateCustomer)
	mux.HandleFunc("GET /api/customers", handler.ListCustomers)
	mux.HandleFunc("GET /api/customers/{id}", handler.GetCustomer)
	mux.HandleFunc("POST /api/customers/{id}/setup", handler.GenerateSetupLink)
	mux.HandleFunc("POST /api/customers/{id}/complete", handler.CompleteSetup)
	mux.HandleFunc("DELETE /api/customers/{id}", handler.DeleteCustomer)

	// Customer-facing integration endpoints
	mux.HandleFunc("GET /integrate", handler.IntegrationPage)
	mux.HandleFunc("POST /integrate", handler.HandleIntegration)
	mux.HandleFunc("GET /integrate/status/{id}", handler.IntegrationStatus)

	// Static files and templates
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/", handler.HomePage)

	// Apply middleware
	wrappedMux := middleware.Chain(mux,
		middleware.Logging(logger),
		middleware.Recovery(logger),
		middleware.CORS(),
		middleware.RequestID(),
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      wrappedMux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("HTTP server starting", slog.String("addr", server.Addr))
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for interrupt signal to gracefully shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		slog.Error("Server failed to start", slog.String("error", err.Error()))
		os.Exit(1)

	case sig := <-shutdown:
		slog.Info("Shutdown signal received", slog.String("signal", sig.String()))

		// Give outstanding requests 30 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Graceful shutdown failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	slog.Info("Server stopped")
}

func loadConfigFromFile(filename string, config *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(config)
}