// Package ui provides HTTP handlers for the desktop application web interface
package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/desktop-app/internal/auth"
	"github.com/scttfrdmn/aws-remote-access-patterns/examples/desktop-app/internal/config"
)

// Handler handles HTTP requests for the desktop app UI
type Handler struct {
	config   *config.Config
	authMgr  *auth.Manager
	logger   *slog.Logger
	template *template.Template
}

// NewHandler creates a new UI handler
func NewHandler(cfg *config.Config, authMgr *auth.Manager, tmpl *template.Template) *Handler {
	return &Handler{
		config:   cfg,
		authMgr:  authMgr,
		logger:   slog.Default(),
		template: tmpl,
	}
}

// RegisterRoutes registers all HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// Main page
	mux.HandleFunc("/", h.handleIndex)

	// API routes
	mux.HandleFunc("/api/status", h.handleStatus)
	mux.HandleFunc("/api/auth/status", h.handleAuthStatus)
	mux.HandleFunc("/api/auth/setup", h.handleAuthSetup)
	mux.HandleFunc("/api/auth/test", h.handleAuthTest)
	mux.HandleFunc("/api/auth/refresh", h.handleAuthRefresh)
	mux.HandleFunc("/api/auth/clear", h.handleAuthClear)
	mux.HandleFunc("/api/s3/buckets", h.handleS3Buckets)
	mux.HandleFunc("/api/ec2/instances", h.handleEC2Instances)
	mux.HandleFunc("/api/config", h.handleConfig)
}

// handleIndex serves the main application page
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := struct {
		Title   string
		Version string
	}{
		Title:   "AWS Desktop App",
		Version: "1.0.0",
	}

	w.Header().Set("Content-Type", "text/html")
	if err := h.template.Execute(w, data); err != nil {
		h.logger.Error("Failed to render template", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// handleStatus returns application status
func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"status":    "running",
		"version":   "1.0.0",
		"timestamp": time.Now(),
		"auth":      h.authMgr.IsConfigured(),
	}

	h.writeJSON(w, status)
}

// handleAuthStatus returns authentication status
func (h *Handler) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status := h.authMgr.GetStatus(ctx)
	h.writeJSON(w, status)
}

// handleAuthSetup configures authentication
func (h *Handler) handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req auth.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := h.authMgr.Setup(ctx, &req); err != nil {
		h.logger.Error("Auth setup failed", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.writeJSON(w, map[string]string{"status": "success"})
}

// handleAuthTest tests the current authentication
func (h *Handler) handleAuthTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := h.authMgr.TestAuthentication(ctx); err != nil {
		h.logger.Error("Auth test failed", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	h.writeJSON(w, map[string]string{"status": "success"})
}

// handleAuthRefresh refreshes authentication credentials
func (h *Handler) handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := h.authMgr.Refresh(ctx); err != nil {
		h.logger.Error("Auth refresh failed", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]string{"status": "success"})
}

// handleAuthClear clears authentication configuration
func (h *Handler) handleAuthClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.authMgr.Clear(); err != nil {
		h.logger.Error("Auth clear failed", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]string{"status": "success"})
}

// handleS3Buckets lists S3 buckets
func (h *Handler) handleS3Buckets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Get AWS configuration
	awsConfig, err := h.authMgr.GetAWSConfig(ctx)
	if err != nil {
		h.logger.Error("Failed to get AWS config", slog.String("error", err.Error()))
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig)

	// List buckets
	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		h.logger.Error("Failed to list S3 buckets", slog.String("error", err.Error()))
		http.Error(w, "Failed to list buckets", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	buckets := make([]map[string]interface{}, len(result.Buckets))
	for i, bucket := range result.Buckets {
		buckets[i] = map[string]interface{}{
			"Name":         *bucket.Name,
			"CreationDate": bucket.CreationDate,
		}
	}

	h.writeJSON(w, buckets)
}

// handleEC2Instances lists EC2 instances
func (h *Handler) handleEC2Instances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Get AWS configuration
	awsConfig, err := h.authMgr.GetAWSConfig(ctx)
	if err != nil {
		h.logger.Error("Failed to get AWS config", slog.String("error", err.Error()))
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Create EC2 client
	ec2Client := ec2.NewFromConfig(awsConfig)

	// Describe instances
	result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		h.logger.Error("Failed to describe EC2 instances", slog.String("error", err.Error()))
		http.Error(w, "Failed to list instances", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var instances []map[string]interface{}
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			instanceData := map[string]interface{}{
				"InstanceId":     *instance.InstanceId,
				"InstanceType":   instance.InstanceType,
				"State":          map[string]interface{}{
					"Name": instance.State.Name,
					"Code": *instance.State.Code,
				},
				"Placement": map[string]interface{}{
					"AvailabilityZone": *instance.Placement.AvailabilityZone,
				},
				"LaunchTime": instance.LaunchTime,
			}

			// Add optional fields
			if instance.PublicIpAddress != nil {
				instanceData["PublicIpAddress"] = *instance.PublicIpAddress
			}
			if instance.PrivateIpAddress != nil {
				instanceData["PrivateIpAddress"] = *instance.PrivateIpAddress
			}
			if instance.PublicDnsName != nil {
				instanceData["PublicDnsName"] = *instance.PublicDnsName
			}

			// Add tags
			tags := make(map[string]string)
			for _, tag := range instance.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}
			if len(tags) > 0 {
				instanceData["Tags"] = tags
			}

			instances = append(instances, instanceData)
		}
	}

	h.writeJSON(w, instances)
}

// handleConfig handles configuration get/set operations
func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGetConfig(w, r)
	case http.MethodPost:
		h.handleUpdateConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetConfig returns the current configuration
func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Return safe configuration (no sensitive data)
	safeConfig := map[string]interface{}{
		"theme":       h.config.UI.Theme,
		"aws_region":  h.config.AWSRegion,
		"debug":       h.config.Debug,
		"ui": map[string]interface{}{
			"auto_refresh":     h.config.UI.AutoRefresh,
			"refresh_interval": h.config.UI.RefreshInterval,
			"notifications":    h.config.UI.Notifications,
		},
		"features": h.config.Features,
	}

	h.writeJSON(w, safeConfig)
}

// handleUpdateConfig updates the configuration
func (h *Handler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update allowed fields
	if theme, ok := updates["theme"].(string); ok {
		h.config.UI.Theme = theme
	}
	if awsRegion, ok := updates["aws_region"].(string); ok {
		h.config.AWSRegion = awsRegion
	}
	if debug, ok := updates["debug"].(bool); ok {
		h.config.Debug = debug
	}
	if ui, ok := updates["ui"].(map[string]interface{}); ok {
		if autoRefresh, ok := ui["auto_refresh"].(bool); ok {
			h.config.UI.AutoRefresh = autoRefresh
		}
		if refreshInterval, ok := ui["refresh_interval"].(float64); ok {
			h.config.UI.RefreshInterval = int(refreshInterval)
		}
		if notifications, ok := ui["notifications"].(bool); ok {
			h.config.UI.Notifications = notifications
		}
	}

	// Save configuration
	if err := h.config.Save(); err != nil {
		h.logger.Error("Failed to save config", slog.String("error", err.Error()))
		http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]string{"status": "success"})
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// corsMiddleware adds CORS headers for local development
func (h *Handler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (h *Handler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapper, r)
		
		duration := time.Since(start)
		
		h.logger.Info("HTTP request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapper.statusCode),
			slog.Duration("duration", duration),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// CreateHandler creates a fully configured HTTP handler with middleware
func CreateHandler(cfg *config.Config, authMgr *auth.Manager, tmpl *template.Template) http.Handler {
	handler := NewHandler(cfg, authMgr, tmpl)
	
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	
	// Apply middleware
	var h http.Handler = mux
	h = handler.loggingMiddleware(h)
	h = handler.corsMiddleware(h)
	
	return h
}