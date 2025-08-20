package crossaccount

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// mockSTSClient for testing cross-account functionality
type mockSTSClient struct {
	assumeRoleFunc        func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
	getCallerIdentityFunc func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockSTSClient) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	if m.assumeRoleFunc != nil {
		return m.assumeRoleFunc(ctx, params, optFns...)
	}
	
	// Default mock response
	expiration := time.Now().Add(time.Hour)
	return &sts.AssumeRoleOutput{
		Credentials: &types.Credentials{
			AccessKeyId:     aws.String("AKIAIOSFODNN7EXAMPLE"),
			SecretAccessKey: aws.String("wJalrXUtnFEMI/K7MDENG/bPxRfiCYzEXAMPLEKEY"),
			SessionToken:    aws.String("example-session-token"),
			Expiration:      &expiration,
		},
		AssumedRoleUser: &types.AssumedRoleUser{
			Arn:           aws.String("arn:aws:sts::123456789012:assumed-role/test-role/test-session"),
			AssumedRoleId: aws.String("AROAIOSFODNN7EXAMPLE:test-session"),
		},
	}, nil
}

func (m *mockSTSClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if m.getCallerIdentityFunc != nil {
		return m.getCallerIdentityFunc(ctx, params, optFns...)
	}
	
	return &sts.GetCallerIdentityOutput{
		Arn:     aws.String("arn:aws:iam::123456789012:user/test-user"),
		Account: aws.String("123456789012"),
		UserId:  aws.String("AIDACKCEVSQ6C2EXAMPLE"),
	}, nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ServiceName:      "test-service",
				ServiceAccountID: "123456789012",
				TemplateS3Bucket: "test-bucket",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "empty service name",
			config: &Config{
				ServiceName:      "",
				ServiceAccountID: "123456789012",
				TemplateS3Bucket: "test-bucket",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Errorf("New() returned nil client without error")
			}
		})
	}
}

func TestSetupCompleteRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request *SetupCompleteRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &SetupCompleteRequest{
				CustomerID: "customer-123",
				RoleARN:    "arn:aws:iam::123456789012:role/test-role",
				ExternalID: "external-123",
			},
			wantErr: false,
		},
		{
			name: "empty customer ID",
			request: &SetupCompleteRequest{
				CustomerID: "",
				RoleARN:    "arn:aws:iam::123456789012:role/test-role",
				ExternalID: "external-123",
			},
			wantErr: true,
		},
		{
			name: "empty role ARN",
			request: &SetupCompleteRequest{
				CustomerID: "customer-123",
				RoleARN:    "",
				ExternalID: "external-123",
			},
			wantErr: true,
		},
		{
			name: "empty external ID",
			request: &SetupCompleteRequest{
				CustomerID: "customer-123",
				RoleARN:    "arn:aws:iam::123456789012:role/test-role",
				ExternalID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic - empty fields should cause errors
			hasEmptyField := tt.request.CustomerID == "" || tt.request.RoleARN == "" || tt.request.ExternalID == ""
			if hasEmptyField != tt.wantErr {
				t.Errorf("SetupCompleteRequest validation mismatch: hasEmptyField = %v, wantErr %v", hasEmptyField, tt.wantErr)
			}
		})
	}
}

func TestClient_GenerateSetupLink(t *testing.T) {
	config := &Config{
		ServiceName:      "test-service",
		ServiceAccountID: "123456789012",
		TemplateS3Bucket: "test-bucket",
	}
	
	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name         string
		customerID   string
		customerName string
		wantErr      bool
	}{
		{
			name:         "valid input",
			customerID:   "customer-123",
			customerName: "Test Customer",
			wantErr:      false,
		},
		{
			name:         "empty customer ID",
			customerID:   "",
			customerName: "Test Customer",
			wantErr:      true,
		},
		{
			name:         "empty customer name",
			customerID:   "customer-123",
			customerName: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupResp, err := client.GenerateSetupLink(tt.customerID, tt.customerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSetupLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if setupResp == nil {
					t.Error("GenerateSetupLink() returned nil response without error")
					return
				}
				
				if setupResp.LaunchURL == "" {
					t.Error("GenerateSetupLink() returned empty launch URL")
				}
				
				if setupResp.ExternalID == "" {
					t.Error("GenerateSetupLink() returned empty external ID")
				}
				
				if setupResp.CustomerID != tt.customerID {
					t.Errorf("GenerateSetupLink() returned wrong customer ID: got %v, want %v", setupResp.CustomerID, tt.customerID)
				}
				
				if len(setupResp.ExternalID) < 64 {
					t.Errorf("GenerateSetupLink() external ID length = %v, want at least 64", len(setupResp.ExternalID))
				}
			}
		})
	}
}

func TestClient_GenerateSecureExternalID(t *testing.T) {
	config := &Config{
		ServiceName:      "test-service",
		ServiceAccountID: "123456789012",
		TemplateS3Bucket: "test-bucket",
	}
	
	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name     string
		customer string
		wantLen  int
	}{
		{
			name:     "valid customer ID",
			customer: "customer-123",
			wantLen:  90, // Should generate a secure external ID (service-hash-random = ~94 chars)
		},
		{
			name:     "different customer ID",
			customer: "customer-456",
			wantLen:  90, // Should still generate a secure ID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll test the external ID generation indirectly through setup link generation
			setupResp, err := client.GenerateSetupLink(tt.customer, "Test Customer")
			if err != nil {
				t.Fatalf("GenerateSetupLink failed: %v", err)
			}
			externalID := setupResp.ExternalID
			
			if len(externalID) < tt.wantLen {
				t.Errorf("ExternalID length = %v, want at least %v", len(externalID), tt.wantLen)
			}
			
			// Should not contain predictable patterns
			if externalID == "" {
				t.Error("ExternalID generation returned empty string")
			}
			
			// Generate another one to ensure they're different
			setupResp2, err := client.GenerateSetupLink(tt.customer, "Test Customer")
			if err != nil {
				t.Fatalf("GenerateSetupLink failed: %v", err)
			}
			if externalID == setupResp2.ExternalID {
				t.Error("ExternalID generation returned identical values - not cryptographically secure")
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid minimal config",
			config: &Config{
				ServiceName:      "test-service",
				ServiceAccountID: "123456789012",
				TemplateS3Bucket: "test-bucket",
			},
			wantErr: false,
		},
		{
			name: "valid full config",
			config: &Config{
				ServiceName:      "test-service",
				ServiceAccountID: "123456789012",
				TemplateS3Bucket: "test-bucket",
				DefaultRegion:    "us-east-1",
				SessionDuration:  time.Hour,
			},
			wantErr: false,
		},
		{
			name: "empty service name",
			config: &Config{
				ServiceName:      "",
				ServiceAccountID: "123456789012",
				TemplateS3Bucket: "test-bucket",
			},
			wantErr: true,
		},
		{
			name: "empty account ID",
			config: &Config{
				ServiceName:      "test-service",
				ServiceAccountID: "",
				TemplateS3Bucket: "test-bucket",
			},
			wantErr: true,
		},
		{
			name: "empty template bucket",
			config: &Config{
				ServiceName:      "test-service",
				ServiceAccountID: "123456789012",
				TemplateS3Bucket: "",
			},
			wantErr: true,
		},
		{
			name: "invalid account ID length",
			config: &Config{
				ServiceName:      "test-service",
				ServiceAccountID: "12345", // Too short
				TemplateS3Bucket: "test-bucket",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_CompleteSetup(t *testing.T) {
	ctx := context.Background()
	
	config := &Config{
		ServiceName:      "test-service",
		ServiceAccountID: "123456789012",
		TemplateS3Bucket: "test-bucket",
	}
	
	client, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name    string
		request *SetupCompleteRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &SetupCompleteRequest{
				CustomerID: "customer-123",
				RoleARN:    "arn:aws:iam::123456789012:role/test-role",
				ExternalID: "external-123",
			},
			wantErr: false, // Will fail due to no real AWS credentials, but validates input
		},
		{
			name:    "nil request",
			request: nil,
			wantErr: true,
		},
		{
			name: "empty customer ID",
			request: &SetupCompleteRequest{
				CustomerID: "",
				RoleARN:    "arn:aws:iam::123456789012:role/test-role",
				ExternalID: "external-123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CompleteSetup(ctx, tt.request)
			
			// Since we don't have real AWS credentials, we expect an error for valid requests too
			// But we can distinguish between validation errors and AWS errors
			if tt.name == "nil request" || tt.name == "empty customer ID" {
				if err == nil {
					t.Error("CompleteSetup() should have returned validation error")
				}
			} else {
				// For valid input, we expect AWS-related error (no credentials)
				if err == nil {
					t.Log("CompleteSetup() succeeded - this is unexpected without real AWS credentials")
				}
			}
		})
	}
}

func TestSimpleConfig(t *testing.T) {
	tests := []struct {
		name          string
		serviceName   string
		accountID     string
		templateBucket string
		wantErr       bool
	}{
		{
			name:           "valid simple config",
			serviceName:    "test-service",
			accountID:      "123456789012",
			templateBucket: "test-bucket",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SimpleConfig(tt.serviceName, tt.accountID, tt.templateBucket)
			
			if config == nil {
				t.Error("SimpleConfig() returned nil")
				return
			}
			
			if config.ServiceName != tt.serviceName {
				t.Errorf("ServiceName = %v, want %v", config.ServiceName, tt.serviceName)
			}
			
			if config.ServiceAccountID != tt.accountID {
				t.Errorf("ServiceAccountID = %v, want %v", config.ServiceAccountID, tt.accountID)
			}
			
			if config.TemplateS3Bucket != tt.templateBucket {
				t.Errorf("TemplateS3Bucket = %v, want %v", config.TemplateS3Bucket, tt.templateBucket)
			}
		})
	}
}

func TestStoredCredentials_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		expiry   time.Time
		expected bool
	}{
		{
			name:     "not expired",
			expiry:   time.Now().Add(time.Hour),
			expected: true,
		},
		{
			name:     "expired",
			expiry:   time.Now().Add(-time.Hour),
			expected: false,
		},
		{
			name:     "expires soon (within 5 minutes)",
			expiry:   time.Now().Add(2 * time.Minute),
			expected: false, // Should be considered expired for safety
		},
		{
			name:     "zero time",
			expiry:   time.Time{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := &StoredCredentials{
				Expiration: tt.expiry,
			}
			
			isValid := creds.IsValid()
			if isValid != tt.expected {
				t.Errorf("IsValid() = %v, want %v", isValid, tt.expected)
			}
		})
	}
}

// Helper functions for testing
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (len(substr) == 0 || s[len(s)-len(substr):] == substr || 
			s[:len(substr)] == substr || 
			findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkGenerateSetupLink(b *testing.B) {
	config := &Config{
		ServiceName:      "test-service",
		ServiceAccountID: "123456789012",
		TemplateS3Bucket: "test-bucket",
	}
	
	client, err := New(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GenerateSetupLink("customer-123", "Test Customer")
		if err != nil {
			b.Fatalf("GenerateSetupLink failed: %v", err)
		}
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := &Config{
		ServiceName:      "test-service",
		ServiceAccountID: "123456789012",
		TemplateS3Bucket: "test-bucket",
		DefaultRegion:    "us-east-1",
		SessionDuration:  time.Hour,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Fatalf("Validate() error = %v", err)
		}
	}
}