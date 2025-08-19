// Package handlers implements HTTP handlers for the SaaS service example
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/crossaccount"
)

// Config holds configuration for handlers
type Config struct {
	CrossAccountClient *crossaccount.Client
	ServiceName        string
	Environment        string
}

// Handler contains all HTTP handlers and dependencies
type Handler struct {
	crossAccountClient *crossaccount.Client
	serviceName        string
	environment        string
	customers          map[string]*Customer // In-memory store for demo
	templates          *template.Template
}

// Customer represents a customer in our system
type Customer struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	AWSAccountID string    `json:"aws_account_id,omitempty"`
	RoleARN      string    `json:"role_arn,omitempty"`
	ExternalID   string    `json:"external_id,omitempty"`
	SetupURL     string    `json:"setup_url,omitempty"`
	Status       string    `json:"status"` // pending, setup_required, active, error
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// New creates a new handler instance
func New(config *Config) (*Handler, error) {
	// Load HTML templates
	tmpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		slog.Warn("Failed to load templates, using default responses", slog.String("error", err.Error()))
	}

	return &Handler{
		crossAccountClient: config.CrossAccountClient,
		serviceName:        config.ServiceName,
		environment:        config.Environment,
		customers:          make(map[string]*Customer),
		templates:          tmpl,
	}, nil
}

// HomePage serves the main application page
func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"ServiceName": h.serviceName,
		"Environment": h.environment,
		"Customers":   h.customers,
	}

	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "home.html", data); err != nil {
			slog.Error("Failed to execute template", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// HealthCheck returns the health status of the service
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":      "healthy",
		"service":     h.serviceName,
		"environment": h.environment,
		"timestamp":   time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ReadinessCheck returns the readiness status of the service
func (h *Handler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// In a real service, you might check database connections, etc.
	status := map[string]interface{}{
		"status":    "ready",
		"service":   h.serviceName,
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// CreateCustomer creates a new customer
func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	customer := &Customer{
		ID:        generateCustomerID(req.Name),
		Name:      req.Name,
		Email:     req.Email,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	h.customers[customer.ID] = customer

	slog.Info("Customer created",
		slog.String("customer_id", customer.ID),
		slog.String("customer_name", customer.Name))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customer)
}

// ListCustomers returns all customers
func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	customers := make([]*Customer, 0, len(h.customers))
	for _, customer := range h.customers {
		customers = append(customers, customer)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"customers": customers,
		"total":     len(customers),
	})
}

// GetCustomer returns a specific customer
func (h *Handler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	if customerID == "" {
		http.Error(w, "Customer ID is required", http.StatusBadRequest)
		return
	}

	customer, exists := h.customers[customerID]
	if !exists {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}

// GenerateSetupLink generates an AWS integration setup link for a customer
func (h *Handler) GenerateSetupLink(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	if customerID == "" {
		http.Error(w, "Customer ID is required", http.StatusBadRequest)
		return
	}

	customer, exists := h.customers[customerID]
	if !exists {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	// Generate setup link using cross-account client
	setupResp, err := h.crossAccountClient.GenerateSetupLink(customerID, customer.Name)
	if err != nil {
		slog.Error("Failed to generate setup link",
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()))
		http.Error(w, "Failed to generate setup link", http.StatusInternalServerError)
		return
	}

	// Update customer record
	customer.SetupURL = setupResp.LaunchURL
	customer.ExternalID = setupResp.ExternalID
	customer.Status = "setup_required"
	customer.UpdatedAt = time.Now()

	slog.Info("Setup link generated",
		slog.String("customer_id", customerID),
		slog.String("external_id", setupResp.ExternalID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(setupResp)
}

// CompleteSetup completes the AWS integration setup for a customer
func (h *Handler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	if customerID == "" {
		http.Error(w, "Customer ID is required", http.StatusBadRequest)
		return
	}

	var req struct {
		RoleARN      string `json:"role_arn"`
		ExternalID   string `json:"external_id"`
		AWSAccountID string `json:"aws_account_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	customer, exists := h.customers[customerID]
	if !exists {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	// Complete setup using cross-account client
	setupReq := &crossaccount.SetupCompleteRequest{
		CustomerID: customerID,
		RoleARN:    req.RoleARN,
		ExternalID: req.ExternalID,
	}

	if err := h.crossAccountClient.CompleteSetup(context.Background(), setupReq); err != nil {
		slog.Error("Failed to complete setup",
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()))

		customer.Status = "error"
		customer.UpdatedAt = time.Now()

		http.Error(w, "Failed to complete setup: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Update customer record
	customer.RoleARN = req.RoleARN
	customer.ExternalID = req.ExternalID
	customer.AWSAccountID = req.AWSAccountID
	customer.Status = "active"
	customer.UpdatedAt = time.Now()

	slog.Info("Setup completed successfully",
		slog.String("customer_id", customerID),
		slog.String("role_arn", req.RoleARN))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "completed",
		"message": "AWS integration setup completed successfully",
	})
}

// DeleteCustomer removes a customer
func (h *Handler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	if customerID == "" {
		http.Error(w, "Customer ID is required", http.StatusBadRequest)
		return
	}

	if _, exists := h.customers[customerID]; !exists {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	delete(h.customers, customerID)

	slog.Info("Customer deleted", slog.String("customer_id", customerID))

	w.WriteHeader(http.StatusNoContent)
}

// IntegrationPage serves the customer integration page
func (h *Handler) IntegrationPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"ServiceName": h.serviceName,
		"Environment": h.environment,
	}

	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "integration.html", data); err != nil {
			slog.Error("Failed to execute template", slog.String("error", err.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}
}

// HandleIntegration processes customer integration requests
func (h *Handler) HandleIntegration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompanyName  string `json:"company_name"`
		Email        string `json:"email"`
		AWSAccountID string `json:"aws_account_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.CompanyName == "" || req.Email == "" {
		http.Error(w, "Company name and email are required", http.StatusBadRequest)
		return
	}

	// Create customer
	customerID := generateCustomerID(req.CompanyName)
	customer := &Customer{
		ID:           customerID,
		Name:         req.CompanyName,
		Email:        req.Email,
		AWSAccountID: req.AWSAccountID,
		Status:       "setup_required",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Generate setup link
	setupResp, err := h.crossAccountClient.GenerateSetupLink(customerID, customer.Name)
	if err != nil {
		slog.Error("Failed to generate setup link",
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()))
		http.Error(w, "Failed to generate setup link", http.StatusInternalServerError)
		return
	}

	customer.SetupURL = setupResp.LaunchURL
	customer.ExternalID = setupResp.ExternalID
	h.customers[customerID] = customer

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"customer_id": customerID,
		"setup_url":   setupResp.LaunchURL,
		"external_id": setupResp.ExternalID,
		"status":      "setup_required",
		"message":     "Integration initiated. Please click the setup URL to complete.",
	})
}

// IntegrationStatus returns the status of a customer integration
func (h *Handler) IntegrationStatus(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("id")
	if customerID == "" {
		http.Error(w, "Customer ID is required", http.StatusBadRequest)
		return
	}

	customer, exists := h.customers[customerID]
	if !exists {
		http.Error(w, "Integration not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"customer_id": customerID,
		"status":      customer.Status,
		"created_at":  customer.CreatedAt,
		"updated_at":  customer.UpdatedAt,
	})
}

// generateCustomerID creates a unique customer ID from the company name
func generateCustomerID(companyName string) string {
	// Simple ID generation - in production, use proper UUID generation
	return fmt.Sprintf("customer-%d", time.Now().Unix())
}