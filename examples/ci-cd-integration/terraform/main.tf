# Main Terraform configuration for CI/CD integration infrastructure
terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.4"
    }
  }
  
  backend "s3" {
    bucket = "aws-remote-access-patterns-terraform-state"
    key    = "ci-cd-integration/terraform.tfstate"
    region = "us-east-1"
    
    # Enable state locking
    dynamodb_table = "terraform-state-locks"
    encrypt        = true
  }
}

# Configure AWS Provider
provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Project     = "aws-remote-access-patterns"
      Component   = "ci-cd-integration"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# Local values for common configurations
locals {
  name_prefix = "${var.project_name}-${var.environment}"
  
  common_tags = {
    Project     = var.project_name
    Component   = "ci-cd-integration"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
  
  # CI/CD platforms configuration
  platforms = {
    github = {
      name = "GitHub Actions"
      oidc_url = "https://token.actions.githubusercontent.com"
      oidc_client_ids = ["sts.amazonaws.com"]
      oidc_thumbprints = [
        "6938fd4d98bab03faadb97b34396831e3780aea1",
        "1c58a3a8518e8759bf075b76b750d4f2df264fcd"
      ]
    }
    gitlab = {
      name = "GitLab CI"
      # GitLab doesn't use OIDC for AWS, uses traditional IAM users/roles
    }
    jenkins = {
      name = "Jenkins"
      # Jenkins uses IAM roles with instance profiles or IAM users
    }
    azure = {
      name = "Azure DevOps"
      # Azure DevOps typically uses service principals or IAM users
    }
  }
}

# Data sources
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# Generate external IDs for cross-account access
resource "random_password" "external_ids" {
  for_each = toset(var.target_environments)
  
  length  = 32
  special = false
  upper   = true
  lower   = true
  numeric = true
}

# Store external IDs in AWS Systems Manager Parameter Store
resource "aws_ssm_parameter" "external_ids" {
  for_each = toset(var.target_environments)
  
  name        = "/${var.project_name}/ci-cd/${each.value}/external-id"
  description = "External ID for ${each.value} environment CI/CD access"
  type        = "SecureString"
  value       = random_password.external_ids[each.value].result
  
  tags = merge(local.common_tags, {
    Environment = each.value
    Purpose     = "ci-cd-external-id"
  })
}

# GitHub Actions OIDC Provider
resource "aws_iam_openid_connect_provider" "github_actions" {
  count = var.enable_github_actions ? 1 : 0
  
  url = local.platforms.github.oidc_url
  
  client_id_list = local.platforms.github.oidc_client_ids
  thumbprint_list = local.platforms.github.oidc_thumbprints
  
  tags = merge(local.common_tags, {
    Platform = "github-actions"
    Purpose  = "oidc-provider"
  })
}

# GitHub Actions IAM Role
resource "aws_iam_role" "github_actions" {
  count = var.enable_github_actions ? 1 : 0
  
  name = "${local.name_prefix}-github-actions-role"
  description = "Role for GitHub Actions CI/CD pipeline"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRoleWithWebIdentity"
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.github_actions[0].arn
        }
        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          StringLike = {
            "token.actions.githubusercontent.com:sub" = "repo:${var.github_repository}:*"
          }
        }
      }
    ]
  })
  
  tags = merge(local.common_tags, {
    Platform = "github-actions"
    Purpose  = "ci-cd-role"
  })
}

# Cross-account deployment roles for each target environment
resource "aws_iam_role" "cross_account_roles" {
  for_each = var.cross_account_roles
  
  name = "${local.name_prefix}-${each.key}-deployment-role"
  description = "Cross-account role for ${each.key} environment deployment"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = each.value.trusted_principals
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = random_password.external_ids[each.key].result
          }
          StringLike = {
            "sts:RoleSessionName" = [
              "GitHubActions-*",
              "GitLabCI-*", 
              "Jenkins-*",
              "AzureDevOps-*"
            ]
          }
          IpAddress = var.allowed_ip_ranges != [] ? {
            "aws:SourceIp" = var.allowed_ip_ranges
          } : null
        }
      }
    ]
  })
  
  tags = merge(local.common_tags, {
    Environment = each.key
    Purpose     = "cross-account-deployment"
  })
}

# Deployment policies for cross-account roles
resource "aws_iam_role_policy" "cross_account_deployment" {
  for_each = var.cross_account_roles
  
  name = "${local.name_prefix}-${each.key}-deployment-policy"
  role = aws_iam_role.cross_account_roles[each.key].id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat(
      [
        # CloudFormation permissions
        {
          Effect = "Allow"
          Action = [
            "cloudformation:CreateStack",
            "cloudformation:UpdateStack",
            "cloudformation:DeleteStack",
            "cloudformation:DescribeStacks",
            "cloudformation:DescribeStackEvents",
            "cloudformation:DescribeStackResources",
            "cloudformation:ListStackResources",
            "cloudformation:GetTemplate",
            "cloudformation:ValidateTemplate"
          ]
          Resource = "arn:aws:cloudformation:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:stack/${var.project_name}-*/*"
        },
        # S3 permissions for deployment artifacts
        {
          Effect = "Allow"
          Action = [
            "s3:GetObject",
            "s3:PutObject",
            "s3:DeleteObject"
          ]
          Resource = "arn:aws:s3:::${var.deployment_bucket_name}/*"
        },
        {
          Effect = "Allow"
          Action = [
            "s3:ListBucket"
          ]
          Resource = "arn:aws:s3:::${var.deployment_bucket_name}"
        },
        # Lambda permissions
        {
          Effect = "Allow"
          Action = [
            "lambda:CreateFunction",
            "lambda:UpdateFunctionCode", 
            "lambda:UpdateFunctionConfiguration",
            "lambda:DeleteFunction",
            "lambda:GetFunction",
            "lambda:ListFunctions",
            "lambda:InvokeFunction",
            "lambda:CreateAlias",
            "lambda:UpdateAlias",
            "lambda:DeleteAlias",
            "lambda:GetAlias"
          ]
          Resource = "arn:aws:lambda:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:function:${var.project_name}-*"
        },
        # API Gateway permissions
        {
          Effect = "Allow"
          Action = [
            "apigateway:GET",
            "apigateway:POST",
            "apigateway:PUT",
            "apigateway:DELETE",
            "apigateway:PATCH"
          ]
          Resource = "arn:aws:apigateway:${data.aws_region.current.name}::/*"
        },
        # IAM permissions for deployment
        {
          Effect = "Allow"
          Action = [
            "iam:PassRole"
          ]
          Resource = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.project_name}-*"
        },
        # CloudWatch Logs permissions
        {
          Effect = "Allow"
          Action = [
            "logs:CreateLogGroup",
            "logs:CreateLogStream",
            "logs:PutLogEvents",
            "logs:DescribeLogGroups",
            "logs:DescribeLogStreams"
          ]
          Resource = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:*"
        }
      ],
      # Environment-specific permissions
      each.value.additional_policies
    )
  })
}

# CI/CD monitoring and logging
resource "aws_cloudwatch_log_group" "ci_cd_logs" {
  name              = "/aws/ci-cd/${local.name_prefix}"
  retention_in_days = var.log_retention_days
  
  tags = merge(local.common_tags, {
    Purpose = "ci-cd-logging"
  })
}

# CloudWatch alarms for monitoring CI/CD activities
resource "aws_cloudwatch_metric_alarm" "failed_deployments" {
  count = var.enable_monitoring ? 1 : 0
  
  alarm_name          = "${local.name_prefix}-failed-deployments"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ErrorCount"
  namespace           = "AWS/CloudFormation"
  period              = "300"
  statistic           = "Sum"
  threshold           = "3"
  alarm_description   = "This metric monitors failed CI/CD deployments"
  
  dimensions = {
    StackName = "${var.project_name}-*"
  }
  
  alarm_actions = var.sns_topic_arn != "" ? [var.sns_topic_arn] : []
  
  tags = merge(local.common_tags, {
    Purpose = "deployment-monitoring"
  })
}

# SNS topic for CI/CD notifications (optional)
resource "aws_sns_topic" "ci_cd_notifications" {
  count = var.create_sns_topic ? 1 : 0
  
  name = "${local.name_prefix}-notifications"
  
  tags = merge(local.common_tags, {
    Purpose = "ci-cd-notifications"
  })
}

# KMS key for encrypting CI/CD secrets
resource "aws_kms_key" "ci_cd_secrets" {
  count = var.enable_kms_encryption ? 1 : 0
  
  description             = "KMS key for encrypting CI/CD secrets and parameters"
  deletion_window_in_days = var.kms_deletion_window
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow CI/CD roles to use the key"
        Effect = "Allow"
        Principal = {
          AWS = concat(
            var.enable_github_actions ? [aws_iam_role.github_actions[0].arn] : [],
            [for role in aws_iam_role.cross_account_roles : role.arn]
          )
        }
        Action = [
          "kms:Decrypt",
          "kms:DescribeKey"
        ]
        Resource = "*"
      }
    ]
  })
  
  tags = merge(local.common_tags, {
    Purpose = "ci-cd-encryption"
  })
}

resource "aws_kms_alias" "ci_cd_secrets" {
  count = var.enable_kms_encryption ? 1 : 0
  
  name          = "alias/${local.name_prefix}-ci-cd-secrets"
  target_key_id = aws_kms_key.ci_cd_secrets[0].key_id
}

# S3 bucket for storing deployment artifacts (optional)
resource "aws_s3_bucket" "deployment_artifacts" {
  count = var.create_artifact_bucket ? 1 : 0
  
  bucket = "${local.name_prefix}-deployment-artifacts"
  
  tags = merge(local.common_tags, {
    Purpose = "deployment-artifacts"
  })
}

resource "aws_s3_bucket_versioning" "deployment_artifacts" {
  count = var.create_artifact_bucket ? 1 : 0
  
  bucket = aws_s3_bucket.deployment_artifacts[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "deployment_artifacts" {
  count = var.create_artifact_bucket ? 1 : 0
  
  bucket = aws_s3_bucket.deployment_artifacts[0].id
  
  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = var.enable_kms_encryption ? aws_kms_key.ci_cd_secrets[0].arn : null
      sse_algorithm     = var.enable_kms_encryption ? "aws:kms" : "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "deployment_artifacts" {
  count = var.create_artifact_bucket ? 1 : 0
  
  bucket = aws_s3_bucket.deployment_artifacts[0].id
  
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}