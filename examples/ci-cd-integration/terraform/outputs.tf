# Terraform outputs for CI/CD integration infrastructure

# GitHub Actions Outputs
output "github_actions_role_arn" {
  description = "ARN of the GitHub Actions IAM role"
  value       = var.enable_github_actions ? aws_iam_role.github_actions[0].arn : null
}

output "github_actions_role_name" {
  description = "Name of the GitHub Actions IAM role"
  value       = var.enable_github_actions ? aws_iam_role.github_actions[0].name : null
}

output "github_oidc_provider_arn" {
  description = "ARN of the GitHub OIDC provider"
  value       = var.enable_github_actions ? aws_iam_openid_connect_provider.github_actions[0].arn : null
}

# Cross-Account Role Outputs
output "cross_account_role_arns" {
  description = "ARNs of cross-account deployment roles"
  value       = { for env, role in aws_iam_role.cross_account_roles : env => role.arn }
}

output "cross_account_role_names" {
  description = "Names of cross-account deployment roles"
  value       = { for env, role in aws_iam_role.cross_account_roles : env => role.name }
}

# External IDs (sensitive)
output "external_ids" {
  description = "External IDs for cross-account access (sensitive)"
  value       = { for env, password in random_password.external_ids : env => password.result }
  sensitive   = true
}

output "external_id_parameter_names" {
  description = "SSM parameter names containing external IDs"
  value       = { for env, param in aws_ssm_parameter.external_ids : env => param.name }
}

# S3 Bucket Outputs
output "deployment_artifacts_bucket_name" {
  description = "Name of the deployment artifacts S3 bucket"
  value       = var.create_artifact_bucket ? aws_s3_bucket.deployment_artifacts[0].id : var.deployment_bucket_name
}

output "deployment_artifacts_bucket_arn" {
  description = "ARN of the deployment artifacts S3 bucket"
  value       = var.create_artifact_bucket ? aws_s3_bucket.deployment_artifacts[0].arn : null
}

# KMS Key Outputs
output "kms_key_id" {
  description = "ID of the KMS key for CI/CD secrets"
  value       = var.enable_kms_encryption ? aws_kms_key.ci_cd_secrets[0].key_id : null
}

output "kms_key_arn" {
  description = "ARN of the KMS key for CI/CD secrets"
  value       = var.enable_kms_encryption ? aws_kms_key.ci_cd_secrets[0].arn : null
}

output "kms_key_alias" {
  description = "Alias of the KMS key for CI/CD secrets"
  value       = var.enable_kms_encryption ? aws_kms_alias.ci_cd_secrets[0].name : null
}

# Monitoring Outputs
output "cloudwatch_log_group_name" {
  description = "Name of the CloudWatch log group for CI/CD"
  value       = aws_cloudwatch_log_group.ci_cd_logs.name
}

output "cloudwatch_log_group_arn" {
  description = "ARN of the CloudWatch log group for CI/CD"
  value       = aws_cloudwatch_log_group.ci_cd_logs.arn
}

output "sns_topic_arn" {
  description = "ARN of the SNS topic for CI/CD notifications"
  value       = var.create_sns_topic ? aws_sns_topic.ci_cd_notifications[0].arn : var.sns_topic_arn
}

# Configuration Examples
output "github_actions_workflow_config" {
  description = "Example GitHub Actions workflow configuration"
  value = var.enable_github_actions ? {
    permissions = {
      id-token = "write"
      contents = "read"
    }
    env = {
      AWS_REGION = var.aws_region
    }
    steps = [
      {
        name = "Configure AWS credentials"
        uses = "aws-actions/configure-aws-credentials@v4"
        with = {
          role-to-assume    = aws_iam_role.github_actions[0].arn
          role-session-name = "GitHubActions"
          aws-region        = var.aws_region
        }
      }
    ]
  } : null
}

output "cross_account_assume_role_examples" {
  description = "Examples of cross-account role assumption commands"
  value = {
    for env, role in aws_iam_role.cross_account_roles : env => {
      aws_cli_command = "aws sts assume-role --role-arn ${role.arn} --role-session-name 'ci-cd-session' --external-id '$(aws ssm get-parameter --name ${aws_ssm_parameter.external_ids[env].name} --with-decryption --query Parameter.Value --output text)'"
      script_command  = "./scripts/assume-role.sh ${role.arn} $(aws ssm get-parameter --name ${aws_ssm_parameter.external_ids[env].name} --with-decryption --query Parameter.Value --output text)"
    }
  }
}

# Security Information
output "security_recommendations" {
  description = "Security recommendations for CI/CD setup"
  value = {
    external_id_retrieval = "Retrieve external IDs from SSM Parameter Store: aws ssm get-parameter --name /{parameter_name} --with-decryption"
    role_session_naming  = "Use consistent session naming patterns: GitHubActions-{repository}-{run_id}"
    ip_restrictions      = length(var.allowed_ip_ranges) > 0 ? "IP restrictions are enabled for: ${join(", ", var.allowed_ip_ranges)}" : "No IP restrictions configured"
    encryption_status    = var.enable_kms_encryption ? "KMS encryption is enabled for secrets" : "KMS encryption is disabled"
  }
}

# Quick Setup Guide
output "setup_instructions" {
  description = "Quick setup instructions for CI/CD integration"
  value = {
    step_1 = "Store external IDs in your CI/CD system's secret management"
    step_2 = var.enable_github_actions ? "Configure GitHub repository secrets with the role ARN: ${aws_iam_role.github_actions[0].arn}" : "Configure your CI/CD system with appropriate role ARNs"
    step_3 = "Use the assume-role.sh script or configure OIDC for authentication"
    step_4 = "Test role assumption with: aws sts get-caller-identity"
    step_5 = "Deploy your applications using the assumed roles"
  }
}

# Resource Summary
output "resource_summary" {
  description = "Summary of created resources"
  value = {
    github_actions_enabled    = var.enable_github_actions
    cross_account_roles_count = length(aws_iam_role.cross_account_roles)
    environments             = keys(aws_iam_role.cross_account_roles)
    artifact_bucket_created  = var.create_artifact_bucket
    kms_encryption_enabled   = var.enable_kms_encryption
    monitoring_enabled       = var.enable_monitoring
    sns_notifications        = var.create_sns_topic
  }
}

# Terraform State Information
output "terraform_state_info" {
  description = "Information about Terraform state management"
  value = {
    backend_bucket     = "aws-remote-access-patterns-terraform-state"
    state_key          = "ci-cd-integration/terraform.tfstate"
    dynamodb_table     = "terraform-state-locks"
    environment_suffix = var.environment
  }
}

# Cost Estimation
output "estimated_monthly_costs" {
  description = "Estimated monthly costs (approximate, in USD)"
  value = {
    iam_roles             = "Free (within AWS free tier limits)"
    s3_storage           = var.create_artifact_bucket ? "$0.023 per GB/month (Standard storage)" : "N/A"
    kms_key              = var.enable_kms_encryption ? "$1.00 per key/month" : "N/A"
    cloudwatch_logs      = "$0.50 per GB ingested, $0.03 per GB stored"
    ssm_parameters       = "Free (within standard parameter limits)"
    sns_notifications    = "$0.50 per 1M requests"
    total_estimated      = "Approximately $2-10/month depending on usage"
  }
}

# Integration Test Commands
output "test_commands" {
  description = "Commands to test the CI/CD integration"
  value = {
    test_github_actions = var.enable_github_actions ? "Use GitHub Actions workflow with the provided role ARN" : "GitHub Actions not enabled"
    test_cross_account = {
      for env, role in aws_iam_role.cross_account_roles : env => "aws sts assume-role --role-arn ${role.arn} --role-session-name test-session --external-id $(aws ssm get-parameter --name ${aws_ssm_parameter.external_ids[env].name} --with-decryption --query Parameter.Value --output text)"
    }
    verify_permissions = "aws sts get-caller-identity"
    test_s3_access    = var.create_artifact_bucket ? "aws s3 ls s3://${aws_s3_bucket.deployment_artifacts[0].id}" : "Artifact bucket not created"
  }
}

# Environment-specific URLs and Endpoints  
output "environment_endpoints" {
  description = "Environment-specific endpoints and URLs"
  value = {
    for env in var.target_environments : env => {
      parameter_store_path = "/${var.project_name}/ci-cd/${env}/"
      role_arn            = aws_iam_role.cross_account_roles[env].arn
      external_id_param   = aws_ssm_parameter.external_ids[env].name
      session_name_prefix = "ci-cd-${env}"
    }
  }
}