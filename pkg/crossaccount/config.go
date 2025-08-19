package crossaccount

import (
	"errors"
	"fmt"
	"time"
)

// Config defines your service's AWS integration requirements
// Only a few fields are required - the rest have sensible defaults
type Config struct {
	// Required: Your service identification
	ServiceName      string `json:"service_name" yaml:"service_name"`
	ServiceAccountID string `json:"service_account_id" yaml:"service_account_id"`
	
	// Required: Where to host CloudFormation templates
	TemplateS3Bucket string `json:"template_s3_bucket" yaml:"template_s3_bucket"`
	
	// Optional: Will use sensible defaults if not specified
	DefaultRegion     string        `json:"default_region" yaml:"default_region"`
	SessionDuration   time.Duration `json:"session_duration" yaml:"session_duration"`
	
	// Optional: Define specific permissions your service needs
	OngoingPermissions []Permission `json:"ongoing_permissions" yaml:"ongoing_permissions"`
	SetupPermissions   []Permission `json:"setup_permissions" yaml:"setup_permissions"`
	
	// Optional: Customize the setup experience for your customers
	BrandingOptions map[string]string `json:"branding_options" yaml:"branding_options"`
}

// SimpleConfig creates a config with just the essentials
// Perfect for getting started quickly - you can add more later
func SimpleConfig(serviceName, serviceAccountID, templateBucket string) *Config {
	return &Config{
		ServiceName:      serviceName,
		ServiceAccountID: serviceAccountID,
		TemplateS3Bucket: templateBucket,
		// Everything else uses defaults
	}
}

// Permission represents an IAM policy statement
// Uses familiar AWS IAM syntax but simplified
type Permission struct {
	Sid       string                 `json:"sid" yaml:"sid"`
	Effect    string                 `json:"effect" yaml:"effect"` // "Allow" or "Deny"
	Actions   []string               `json:"actions" yaml:"actions"`
	Resources []string               `json:"resources" yaml:"resources"`
	Condition map[string]interface{} `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// Validate ensures the config has minimum required fields
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return errors.New("service_name is required - this identifies your service to customers")
	}
	
	if c.ServiceAccountID == "" {
		return errors.New("service_account_id is required - this is your AWS account ID")
	}
	
	if c.TemplateS3Bucket == "" {
		return errors.New("template_s3_bucket is required - this hosts your CloudFormation templates")
	}

	// Set helpful defaults
	if c.DefaultRegion == "" {
		c.DefaultRegion = "us-east-1"
	}
	
	if c.SessionDuration == 0 {
		c.SessionDuration = time.Hour // 1 hour is reasonable for most use cases
	}

	// Validate AWS account ID format
	if len(c.ServiceAccountID) != 12 {
		return errors.New("service_account_id must be a 12-digit AWS account ID")
	}

	return nil
}

// SetupResponse contains everything needed for customer setup
type SetupResponse struct {
	LaunchURL     string `json:"launch_url"`     // One-click CloudFormation link
	ExternalID    string `json:"external_id"`    // Security token for the role
	CustomerID    string `json:"customer_id"`    // Your customer identifier  
	StackName     string `json:"stack_name"`     // CloudFormation stack name
	SetupComplete bool   `json:"setup_complete"` // Whether setup is finished
}

// SetupCompleteRequest is sent after customer creates the CloudFormation stack
type SetupCompleteRequest struct {
	CustomerID string `json:"customer_id"`
	RoleARN    string `json:"role_arn"`    // From CloudFormation outputs
	ExternalID string `json:"external_id"` // From CloudFormation outputs
}

// CustomerCredentials stores what we need to access customer's AWS account
type CustomerCredentials struct {
	CustomerID string    `json:"customer_id"`
	RoleARN    string    `json:"role_arn"`
	ExternalID string    `json:"external_id"`
	SetupPhase bool      `json:"setup_phase"` // True if setup permissions are still active
	CreatedAt  time.Time `json:"created_at"`
}

// CleanupInstructions helps customers remove setup permissions
type CleanupInstructions struct {
	CustomerID       string   `json:"customer_id"`
	Instructions     []string `json:"instructions"`      // Human-readable steps
	AutomationScript string   `json:"automation_script"` // AWS CLI script
}

// Common permission templates that most services need
var (
	// EC2InstanceManagement - Basic EC2 operations
	EC2InstanceManagement = Permission{
		Sid:    "EC2InstanceManagement",
		Effect: "Allow",
		Actions: []string{
			"ec2:DescribeInstances",
			"ec2:DescribeInstanceTypes",
			"ec2:RunInstances",
			"ec2:TerminateInstances",
			"ec2:StartInstances",
			"ec2:StopInstances",
		},
		Resources: []string{"*"},
	}

	// S3DataAccess - Read/write specific S3 buckets
	S3DataAccess = Permission{
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
	}

	// CloudWatchLogs - Create and write logs
	CloudWatchLogs = Permission{
		Sid:    "CloudWatchLogs",
		Effect: "Allow",
		Actions: []string{
			"logs:CreateLogGroup",
			"logs:CreateLogStream",
			"logs:PutLogEvents",
			"logs:DescribeLogGroups",
			"logs:DescribeLogStreams",
		},
		Resources: []string{"*"},
	}
)

// QuickConfig creates a config with common permissions for different service types
func QuickConfig(serviceType, serviceName, serviceAccountID, templateBucket string) *Config {
	config := SimpleConfig(serviceName, serviceAccountID, templateBucket)
	
	switch serviceType {
	case "data-platform":
		config.OngoingPermissions = []Permission{S3DataAccess, CloudWatchLogs}
		config.SetupPermissions = []Permission{
			{
				Sid:    "S3BucketSetup",
				Effect: "Allow",
				Actions: []string{"s3:CreateBucket", "s3:PutBucketPolicy"},
				Resources: []string{"*"},
			},
		}
		
	case "compute-platform":
		config.OngoingPermissions = []Permission{EC2InstanceManagement, CloudWatchLogs}
		config.SetupPermissions = []Permission{
			{
				Sid:    "VPCSetup",
				Effect: "Allow",
				Actions: []string{
					"ec2:CreateVpc", "ec2:CreateSubnet", "ec2:CreateSecurityGroup",
					"ec2:CreateInternetGateway", "ec2:CreateRouteTable",
				},
				Resources: []string{"*"},
			},
		}
		
	case "monitoring-platform":
		config.OngoingPermissions = []Permission{
			{
				Sid:    "CloudWatchMetrics",
				Effect: "Allow",
				Actions: []string{
					"cloudwatch:GetMetricStatistics",
					"cloudwatch:ListMetrics",
					"cloudwatch:GetMetricData",
				},
				Resources: []string{"*"},
			},
			CloudWatchLogs,
		}
	}
	
	return config
}