# Terraform variables for CI/CD integration infrastructure

variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "aws-remote-access-patterns"
}

variable "environment" {
  description = "Environment name (e.g., dev, staging, prod)"
  type        = string
  default     = "dev"
  
  validation {
    condition     = can(regex("^(dev|staging|prod)$", var.environment))
    error_message = "Environment must be one of: dev, staging, prod."
  }
}

variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

# CI/CD Platform Configuration
variable "enable_github_actions" {
  description = "Enable GitHub Actions OIDC integration"
  type        = bool
  default     = true
}

variable "github_repository" {
  description = "GitHub repository in the format 'owner/repo'"
  type        = string
  default     = ""
}

variable "enable_gitlab_ci" {
  description = "Enable GitLab CI integration"
  type        = bool
  default     = false
}

variable "enable_jenkins" {
  description = "Enable Jenkins integration"
  type        = bool
  default     = false
}

variable "enable_azure_devops" {
  description = "Enable Azure DevOps integration"
  type        = bool
  default     = false
}

# Cross-Account Configuration
variable "cross_account_roles" {
  description = "Configuration for cross-account deployment roles"
  type = map(object({
    trusted_principals   = list(string)
    additional_policies = list(object({
      Effect   = string
      Action   = list(string)
      Resource = string
    }))
  }))
  default = {
    dev = {
      trusted_principals = []
      additional_policies = []
    }
    staging = {
      trusted_principals = []
      additional_policies = []
    }
    production = {
      trusted_principals = []
      additional_policies = []
    }
  }
}

variable "target_environments" {
  description = "List of target environments for deployment"
  type        = list(string)
  default     = ["dev", "staging", "production"]
}

# Security Configuration
variable "allowed_ip_ranges" {
  description = "List of IP ranges allowed to assume CI/CD roles"
  type        = list(string)
  default     = []
}

variable "session_duration" {
  description = "Maximum session duration for assumed roles (in seconds)"
  type        = number
  default     = 3600
  
  validation {
    condition     = var.session_duration >= 900 && var.session_duration <= 43200
    error_message = "Session duration must be between 900 (15 minutes) and 43200 (12 hours) seconds."
  }
}

variable "external_id_length" {
  description = "Length of generated external IDs"
  type        = number
  default     = 32
  
  validation {
    condition     = var.external_id_length >= 16 && var.external_id_length <= 128
    error_message = "External ID length must be between 16 and 128 characters."
  }
}

# Storage Configuration
variable "create_artifact_bucket" {
  description = "Create S3 bucket for storing deployment artifacts"
  type        = bool
  default     = true
}

variable "deployment_bucket_name" {
  description = "Name of the S3 bucket for deployment artifacts"
  type        = string
  default     = ""
}

variable "artifact_retention_days" {
  description = "Number of days to retain deployment artifacts"
  type        = number
  default     = 90
}

# Encryption Configuration
variable "enable_kms_encryption" {
  description = "Enable KMS encryption for secrets and artifacts"
  type        = bool
  default     = true
}

variable "kms_deletion_window" {
  description = "KMS key deletion window in days"
  type        = number
  default     = 7
  
  validation {
    condition     = var.kms_deletion_window >= 7 && var.kms_deletion_window <= 30
    error_message = "KMS deletion window must be between 7 and 30 days."
  }
}

# Monitoring Configuration
variable "enable_monitoring" {
  description = "Enable CloudWatch monitoring and alarms"
  type        = bool
  default     = true
}

variable "log_retention_days" {
  description = "CloudWatch log retention period in days"
  type        = number
  default     = 14
  
  validation {
    condition = contains([1, 3, 5, 7, 14, 30, 60, 90, 120, 150, 180, 365, 400, 545, 731, 1827, 3653], var.log_retention_days)
    error_message = "Log retention days must be one of the valid CloudWatch log retention periods."
  }
}

variable "create_sns_topic" {
  description = "Create SNS topic for CI/CD notifications"
  type        = bool
  default     = true
}

variable "sns_topic_arn" {
  description = "ARN of existing SNS topic for notifications (if not creating new one)"
  type        = string
  default     = ""
}

variable "notification_endpoints" {
  description = "List of notification endpoints (email, SMS, etc.)"
  type = list(object({
    protocol = string
    endpoint = string
  }))
  default = []
}

# Resource Tagging
variable "additional_tags" {
  description = "Additional tags to apply to resources"
  type        = map(string)
  default     = {}
}

# Cost Management
variable "enable_cost_allocation_tags" {
  description = "Enable cost allocation tags for billing"
  type        = bool
  default     = true
}

variable "budget_amount" {
  description = "Monthly budget amount in USD for cost monitoring"
  type        = number
  default     = 100
}

variable "create_budget_alarm" {
  description = "Create AWS Budget alarm for cost monitoring"
  type        = bool
  default     = false
}

# Advanced Configuration
variable "enable_cloudtrail_integration" {
  description = "Enable CloudTrail integration for auditing"
  type        = bool
  default     = true
}

variable "cloudtrail_bucket_name" {
  description = "S3 bucket name for CloudTrail logs"
  type        = string
  default     = ""
}

variable "enable_config_rules" {
  description = "Enable AWS Config rules for compliance monitoring"
  type        = bool
  default     = false
}

variable "allowed_regions" {
  description = "List of AWS regions where resources can be created"
  type        = list(string)
  default     = ["us-east-1", "us-west-2", "eu-west-1"]
}

# Development/Testing Configuration
variable "enable_debug_logging" {
  description = "Enable debug logging for troubleshooting"
  type        = bool
  default     = false
}

variable "create_test_resources" {
  description = "Create additional resources for testing purposes"
  type        = bool
  default     = false
}

variable "test_role_name" {
  description = "Name of test role for CI/CD testing"
  type        = string
  default     = "ci-cd-test-role"
}

# Integration Configuration
variable "webhook_endpoints" {
  description = "List of webhook endpoints for deployment notifications"
  type = list(object({
    name = string
    url  = string
    secret = optional(string)
  }))
  default = []
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for notifications"
  type        = string
  default     = ""
  sensitive   = true
}

variable "teams_webhook_url" {
  description = "Microsoft Teams webhook URL for notifications"
  type        = string
  default     = ""
  sensitive   = true
}

# Performance Configuration
variable "lambda_timeout" {
  description = "Timeout for Lambda functions in seconds"
  type        = number
  default     = 300
}

variable "lambda_memory_size" {
  description = "Memory size for Lambda functions in MB"
  type        = number
  default     = 512
}

# Compliance and Security
variable "enable_encryption_in_transit" {
  description = "Enforce encryption in transit for all communications"
  type        = bool
  default     = true
}

variable "enable_vpc_endpoints" {
  description = "Create VPC endpoints for AWS services"
  type        = bool
  default     = false
}

variable "vpc_id" {
  description = "VPC ID for VPC endpoints (if enabled)"
  type        = string
  default     = ""
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for VPC endpoints"
  type        = list(string)
  default     = []
}

variable "security_group_ids" {
  description = "Security group IDs for VPC endpoints"
  type        = list(string)
  default     = []
}