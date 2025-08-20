package crossaccount

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Client provides simple cross-account AWS integration
// This handles all the complexity so your customers just click one link
type Client struct {
	config  *Config
	storage CredentialStorage
}

// New creates a new cross-account client with sane defaults
// Only requires your service name and account ID to get started
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Set helpful defaults to minimize configuration
	if cfg.SessionDuration == 0 {
		cfg.SessionDuration = time.Hour // 1 hour default
	}
	if cfg.DefaultRegion == "" {
		cfg.DefaultRegion = "us-east-1"
	}
	if cfg.TemplateS3Bucket == "" {
		return nil, fmt.Errorf("template S3 bucket is required for hosting CloudFormation templates")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Client{
		config:  cfg,
		storage: NewMemoryStorage(), // Simple in-memory storage by default
	}, nil
}

// GenerateSetupLink creates a one-click setup link for your customer
// This is the main entry point - customer clicks this link and follows the wizard
func (c *Client) GenerateSetupLink(customerID, customerName string) (*SetupResponse, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}
	if customerName == "" {
		return nil, fmt.Errorf("customer name is required")
	}

	// Generate a unique, secure external ID for this customer
	externalID := c.generateSecureExternalID(customerID)

	// Create CloudFormation launch URL with all parameters pre-filled
	templateURL := fmt.Sprintf("https://%s.s3.amazonaws.com/cross-account-role.yaml", c.config.TemplateS3Bucket)
	
	params := url.Values{}
	params.Set("templateURL", templateURL)
	params.Set("stackName", fmt.Sprintf("%s-Integration-%s", c.config.ServiceName, customerName))
	params.Set("param_ExternalId", externalID)
	params.Set("param_ServiceAccountId", c.config.ServiceAccountID)
	params.Set("param_RoleName", fmt.Sprintf("%s-CrossAccount-%s", c.config.ServiceName, customerID))
	params.Set("param_SetupPhase", "true") // Include setup permissions initially

	launchURL := fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks/quickcreate?%s", 
		c.config.DefaultRegion, params.Encode())

	return &SetupResponse{
		LaunchURL:      launchURL,
		ExternalID:     externalID,
		CustomerID:     customerID,
		StackName:      params.Get("stackName"),
		SetupComplete:  false,
	}, nil
}

// CompleteSetup verifies the customer's role and stores credentials
// Call this after the customer has created the CloudFormation stack
func (c *Client) CompleteSetup(ctx context.Context, req *SetupCompleteRequest) error {
	if req == nil {
		return fmt.Errorf("setup request is required")
	}
	if req.CustomerID == "" || req.RoleARN == "" || req.ExternalID == "" {
		return fmt.Errorf("customer ID, role ARN, and external ID are all required")
	}

	// Test that we can actually assume the role
	if err := c.validateRoleAccess(ctx, req.RoleARN, req.ExternalID); err != nil {
		return fmt.Errorf("role validation failed: %w", err)
	}

	// Store the credentials securely
	creds := &StoredCredentials{
		RoleARN:     req.RoleARN,
		ExternalID:  req.ExternalID,
		SessionName: fmt.Sprintf("%s-%s", c.config.ServiceName, req.CustomerID),
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		Expiration:  time.Now().Add(24 * time.Hour), // Set expiration
	}

	if err := c.storage.Store(ctx, req.CustomerID, creds); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	return nil
}

// AssumeRole gets temporary credentials for a customer's AWS account
// This is what you use in your application code to access customer resources
func (c *Client) AssumeRole(ctx context.Context, customerID string) (aws.Config, error) {
	if customerID == "" {
		return aws.Config{}, fmt.Errorf("customer ID is required")
	}

	// Get stored credentials
	creds, err := c.storage.Retrieve(ctx, customerID)
	if err != nil {
		return aws.Config{}, fmt.Errorf("customer not found: %w", err)
	}

	// Load our service's AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	// Assume the customer's role
	sessionName := fmt.Sprintf("%s-%s-%d", c.config.ServiceName, customerID, time.Now().Unix())
	result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(creds.RoleARN),
		RoleSessionName: aws.String(sessionName),
		ExternalId:      aws.String(creds.ExternalID),
		DurationSeconds: aws.Int32(int32(c.config.SessionDuration.Seconds())),
	})
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to assume role: %w", err)
	}

	// Create new AWS config with the temporary credentials
	return config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(&staticCredentialsProvider{
			accessKey:    aws.ToString(result.Credentials.AccessKeyId),
			secretKey:    aws.ToString(result.Credentials.SecretAccessKey),
			sessionToken: aws.ToString(result.Credentials.SessionToken),
		}),
		config.WithRegion(c.config.DefaultRegion),
	)
}

// RemoveSetupPermissions removes temporary setup permissions from customer role
// Call this after initial setup is complete to improve security
func (c *Client) RemoveSetupPermissions(customerID string) (*CleanupInstructions, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customer ID is required")
	}

	creds, err := c.storage.Retrieve(context.Background(), customerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// Update credentials to mark setup phase as complete
	creds.LastUsed = time.Now()
	if err := c.storage.Store(context.Background(), customerID, creds); err != nil {
		return nil, fmt.Errorf("failed to update credentials: %w", err)
	}

	// Return instructions for customer
	return &CleanupInstructions{
		CustomerID:    customerID,
		Instructions: []string{
			"1. Go to AWS CloudFormation console",
			"2. Find your stack: " + fmt.Sprintf("%s-Integration-*", c.config.ServiceName),
			"3. Click 'Update'",
			"4. Change 'SetupPhase' parameter from 'true' to 'false'",
			"5. Click 'Update stack'",
		},
		AutomationScript: c.generateCleanupScript(customerID),
	}, nil
}

// validateRoleAccess tests that we can assume the customer's role
func (c *Client) validateRoleAccess(ctx context.Context, roleARN, externalID string) error {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	// Try to assume the role
	_, err = stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(fmt.Sprintf("%s-validation", c.config.ServiceName)),
		ExternalId:      aws.String(externalID),
		DurationSeconds: aws.Int32(900), // 15 minutes for validation
	})

	return err
}

// generateSecureExternalID creates a cryptographically secure external ID
func (c *Client) generateSecureExternalID(customerID string) string {
	// Use crypto/rand for security - 32 bytes for extra entropy
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		// If crypto/rand fails, this is a critical security issue
		// Do not fallback to predictable timestamp-based IDs
		panic(fmt.Sprintf("Critical security error: unable to generate secure random bytes: %v", err))
	}
	
	// Create a secure external ID with hex encoding
	hexString := hex.EncodeToString(randomBytes)
	
	// Include customer ID hash for traceability without exposing customer info
	hasher := sha256.New()
	hasher.Write([]byte(customerID))
	customerHash := hex.EncodeToString(hasher.Sum(nil)[:8]) // First 8 bytes of SHA256
	
	return fmt.Sprintf("%s-%s-%s", c.config.ServiceName, customerHash, hexString)
}

// GenerateExternalID creates a cryptographically secure external ID for cross-account access
func GenerateExternalID(customerID string) string {
	// Use crypto/rand for security - 32 bytes for extra entropy
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		// If crypto/rand fails, this is a critical security issue
		// Do not fallback to predictable timestamp-based IDs
		panic(fmt.Sprintf("Critical security error: unable to generate secure random bytes: %v", err))
	}
	
	// Create a secure external ID with hex encoding
	hexString := hex.EncodeToString(randomBytes)
	
	if customerID == "" {
		// If no customer ID provided, just use random hex
		return hexString
	}
	
	// Include customer ID hash for traceability without exposing customer info
	hasher := sha256.New()
	hasher.Write([]byte(customerID))
	customerHash := hex.EncodeToString(hasher.Sum(nil)[:8]) // First 8 bytes of SHA256
	
	return fmt.Sprintf("%s-%s", customerHash, hexString)
}

// generateCleanupScript creates an AWS CLI script for removing setup permissions
func (c *Client) generateCleanupScript(customerID string) string {
	stackName := fmt.Sprintf("%s-Integration-%s", c.config.ServiceName, customerID)
	return fmt.Sprintf(`#!/bin/bash
# Remove setup permissions from %s integration
aws cloudformation update-stack \
  --stack-name "%s" \
  --use-previous-template \
  --parameters ParameterKey=SetupPhase,ParameterValue=false \
  --capabilities CAPABILITY_IAM

echo "Setup permissions removed. Integration is now secure for ongoing operations."`, 
		c.config.ServiceName, stackName)
}

// staticCredentialsProvider implements aws.CredentialsProvider for temporary credentials
type staticCredentialsProvider struct {
	accessKey, secretKey, sessionToken string
}

func (s *staticCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     s.accessKey,
		SecretAccessKey: s.secretKey,
		SessionToken:    s.sessionToken,
	}, nil
}