# CI/CD Integration Example

Comprehensive examples demonstrating how to implement secure AWS authentication patterns in CI/CD pipelines using temporary credentials, cross-account roles, and modern DevOps best practices.

## 🎯 Overview

This example provides production-ready CI/CD configurations for:

- **GitHub Actions** - Complete workflows with OIDC and cross-account access
- **GitLab CI** - Pipeline configurations with temporary credentials
- **Jenkins** - Jenkinsfile examples with AWS authentication
- **Azure DevOps** - Pipeline templates with cross-account roles
- **Terraform** - Infrastructure as Code for CI/CD resources
- **Scripts** - Reusable authentication and deployment utilities

## 🚀 Quick Start

### 1. Choose Your CI/CD Platform

Each platform has specific setup requirements:

- [GitHub Actions](#github-actions) - OIDC-based authentication
- [GitLab CI](#gitlab-ci) - Service accounts and temporary credentials
- [Jenkins](#jenkins) - IAM roles and credential plugins
- [Azure DevOps](#azure-devops) - Service connections and cross-account roles

### 2. Deploy Infrastructure

```bash
# Deploy the CI/CD infrastructure using Terraform
cd terraform/
terraform init
terraform plan -var="environment=dev"
terraform apply
```

### 3. Configure Your Pipeline

Copy the relevant workflow/pipeline file to your repository and customize the variables.

## 🔧 Platform Configurations

### GitHub Actions

#### OIDC Configuration

GitHub Actions supports OpenID Connect (OIDC) for secure, temporary credential access without storing long-lived secrets.

**Setup:**
1. Create an OIDC identity provider in AWS
2. Create a role that trusts the OIDC provider  
3. Configure the workflow to assume the role

**Example Workflow:**
```yaml
# .github/workflows/deploy.yml
name: Deploy to AWS
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  id-token: write
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/GitHubActionsRole
          role-session-name: GitHubActions
          aws-region: us-east-1
      
      - name: Deploy application
        run: |
          aws sts get-caller-identity
          # Your deployment commands here
```

#### Cross-Account Deployment

```yaml
# .github/workflows/cross-account-deploy.yml
name: Cross-Account Deployment
on:
  push:
    branches: [main]

permissions:
  id-token: write
  contents: read

jobs:
  deploy-dev:
    runs-on: ubuntu-latest
    environment: development
    steps:
      - uses: actions/checkout@v4
      - name: Configure AWS (Dev)
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::111111111111:role/DevDeploymentRole
          aws-region: us-east-1
          role-session-name: GitHubActions-Dev
      - name: Deploy to Dev
        run: ./scripts/deploy.sh dev

  deploy-staging:
    needs: deploy-dev
    runs-on: ubuntu-latest
    environment: staging
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Configure AWS (Staging)  
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::222222222222:role/StagingDeploymentRole
          aws-region: us-east-1
          role-session-name: GitHubActions-Staging
      - name: Deploy to Staging
        run: ./scripts/deploy.sh staging

  deploy-prod:
    needs: deploy-staging
    runs-on: ubuntu-latest
    environment: production
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Configure AWS (Production)
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::333333333333:role/ProductionDeploymentRole
          aws-region: us-east-1
          role-session-name: GitHubActions-Production
      - name: Deploy to Production
        run: ./scripts/deploy.sh production
```

### GitLab CI

#### Basic Configuration

```yaml
# .gitlab-ci.yml
stages:
  - test
  - build
  - deploy

variables:
  AWS_DEFAULT_REGION: us-east-1
  
before_script:
  - ./scripts/assume-role.sh

test:
  stage: test
  image: golang:1.21
  script:
    - go test ./...

build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA

deploy:
  stage: deploy
  image: amazon/aws-cli:latest
  variables:
    ROLE_ARN: arn:aws:iam::123456789012:role/GitLabDeploymentRole
    EXTERNAL_ID: $GITLAB_EXTERNAL_ID
  script:
    - ./scripts/cross-account-deploy.sh
  only:
    - main
```

#### Multi-Environment Pipeline

```yaml
# .gitlab-ci.yml (Multi-environment)
stages:
  - test
  - deploy-dev
  - deploy-staging  
  - deploy-prod

.deploy_template: &deploy_template
  image: amazon/aws-cli:latest
  before_script:
    - ./scripts/assume-role.sh $ROLE_ARN $EXTERNAL_ID
  script:
    - ./scripts/deploy.sh $ENVIRONMENT

test:
  stage: test
  image: golang:1.21
  script:
    - go test ./...

deploy:dev:
  <<: *deploy_template
  stage: deploy-dev
  variables:
    ENVIRONMENT: dev
    ROLE_ARN: arn:aws:iam::111111111111:role/DevRole
    EXTERNAL_ID: $DEV_EXTERNAL_ID
  environment:
    name: development
    url: https://dev.example.com

deploy:staging:
  <<: *deploy_template
  stage: deploy-staging
  variables:
    ENVIRONMENT: staging
    ROLE_ARN: arn:aws:iam::222222222222:role/StagingRole
    EXTERNAL_ID: $STAGING_EXTERNAL_ID
  environment:
    name: staging
    url: https://staging.example.com
  only:
    - main

deploy:production:
  <<: *deploy_template
  stage: deploy-prod
  variables:
    ENVIRONMENT: production
    ROLE_ARN: arn:aws:iam::333333333333:role/ProductionRole
    EXTERNAL_ID: $PRODUCTION_EXTERNAL_ID
  environment:
    name: production
    url: https://example.com
  only:
    - main
  when: manual
```

### Jenkins

#### Declarative Pipeline

```groovy
// Jenkinsfile
pipeline {
    agent any
    
    parameters {
        choice(
            name: 'ENVIRONMENT',
            choices: ['dev', 'staging', 'production'],
            description: 'Environment to deploy to'
        )
    }
    
    environment {
        AWS_DEFAULT_REGION = 'us-east-1'
        ROLE_SESSION_NAME = "Jenkins-${env.BUILD_NUMBER}"
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Test') {
            steps {
                script {
                    sh 'go test ./...'
                }
            }
        }
        
        stage('Assume Role') {
            steps {
                script {
                    def roleMapping = [
                        'dev': 'arn:aws:iam::111111111111:role/DevRole',
                        'staging': 'arn:aws:iam::222222222222:role/StagingRole', 
                        'production': 'arn:aws:iam::333333333333:role/ProductionRole'
                    ]
                    
                    env.TARGET_ROLE = roleMapping[params.ENVIRONMENT]
                    
                    withCredentials([
                        string(credentialsId: "${params.ENVIRONMENT}-external-id", variable: 'EXTERNAL_ID')
                    ]) {
                        sh './scripts/jenkins-assume-role.sh'
                    }
                }
            }
        }
        
        stage('Deploy') {
            steps {
                script {
                    sh "./scripts/deploy.sh ${params.ENVIRONMENT}"
                }
            }
        }
    }
    
    post {
        always {
            script {
                // Clean up temporary credentials
                sh 'rm -f ~/.aws/credentials.tmp'
            }
        }
        success {
            echo "Deployment to ${params.ENVIRONMENT} completed successfully!"
        }
        failure {
            echo "Deployment to ${params.ENVIRONMENT} failed!"
        }
    }
}
```

### Azure DevOps

#### Pipeline Template

```yaml
# azure-pipelines.yml
trigger:
  branches:
    include:
      - main
      - develop

pool:
  vmImage: 'ubuntu-latest'

variables:
  - group: aws-credentials
  - name: AWS_DEFAULT_REGION
    value: 'us-east-1'

stages:
- stage: Test
  jobs:
  - job: UnitTests
    steps:
    - task: GoTool@0
      inputs:
        version: '1.21'
    - script: go test ./...
      displayName: 'Run tests'

- stage: Deploy
  condition: eq(variables['Build.SourceBranch'], 'refs/heads/main')
  jobs:
  - deployment: DeployToDev
    environment: 'development'
    strategy:
      runOnce:
        deploy:
          steps:
          - template: templates/deploy-template.yml
            parameters:
              environment: 'dev'
              roleArn: 'arn:aws:iam::111111111111:role/DevRole'
              externalId: $(DEV_EXTERNAL_ID)

  - deployment: DeployToStaging
    dependsOn: DeployToDev
    environment: 'staging'
    strategy:
      runOnce:
        deploy:
          steps:
          - template: templates/deploy-template.yml
            parameters:
              environment: 'staging'
              roleArn: 'arn:aws:iam::222222222222:role/StagingRole'
              externalId: $(STAGING_EXTERNAL_ID)
```

## 📜 Scripts

### Cross-Account Role Assumption Script

```bash
#!/bin/bash
# scripts/assume-role.sh

set -euo pipefail

ROLE_ARN="${1:-$ROLE_ARN}"
EXTERNAL_ID="${2:-$EXTERNAL_ID}"
SESSION_NAME="${3:-ci-cd-session-$(date +%s)}"
DURATION="${4:-3600}"

if [[ -z "$ROLE_ARN" ]]; then
    echo "Error: ROLE_ARN is required"
    exit 1
fi

echo "Assuming role: $ROLE_ARN"
echo "External ID: ${EXTERNAL_ID:0:10}..." # Only show first 10 chars for security

# Assume the role
TEMP_CREDS=$(aws sts assume-role \
    --role-arn "$ROLE_ARN" \
    --role-session-name "$SESSION_NAME" \
    --duration-seconds "$DURATION" \
    ${EXTERNAL_ID:+--external-id "$EXTERNAL_ID"} \
    --output json)

# Extract credentials
ACCESS_KEY=$(echo "$TEMP_CREDS" | jq -r '.Credentials.AccessKeyId')
SECRET_KEY=$(echo "$TEMP_CREDS" | jq -r '.Credentials.SecretAccessKey')
SESSION_TOKEN=$(echo "$TEMP_CREDS" | jq -r '.Credentials.SessionToken')
EXPIRATION=$(echo "$TEMP_CREDS" | jq -r '.Credentials.Expiration')

# Export credentials for subsequent commands
export AWS_ACCESS_KEY_ID="$ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$SECRET_KEY"
export AWS_SESSION_TOKEN="$SESSION_TOKEN"

# Also write to credentials file for tools that prefer file-based config
mkdir -p ~/.aws
cat > ~/.aws/credentials << EOF
[default]
aws_access_key_id = $ACCESS_KEY
aws_secret_access_key = $SECRET_KEY
aws_session_token = $SESSION_TOKEN
EOF

echo "Role assumed successfully. Credentials expire at: $EXPIRATION"

# Verify the assumed role
echo "Current identity:"
aws sts get-caller-identity

# Export for CI/CD environment
echo "AWS_ACCESS_KEY_ID=$ACCESS_KEY" >> $GITHUB_ENV || true
echo "AWS_SECRET_ACCESS_KEY=$SECRET_KEY" >> $GITHUB_ENV || true  
echo "AWS_SESSION_TOKEN=$SESSION_TOKEN" >> $GITHUB_ENV || true
```

### Deployment Script

```bash
#!/bin/bash
# scripts/deploy.sh

set -euo pipefail

ENVIRONMENT="${1:-dev}"
APPLICATION_NAME="${APPLICATION_NAME:-remote-access-app}"
AWS_REGION="${AWS_REGION:-us-east-1}"

echo "Deploying $APPLICATION_NAME to $ENVIRONMENT environment in $AWS_REGION"

# Verify AWS credentials
echo "Current AWS identity:"
aws sts get-caller-identity

# Set environment-specific variables
case "$ENVIRONMENT" in
    "dev")
        STACK_NAME="$APPLICATION_NAME-dev"
        PARAMETER_OVERRIDES="Environment=dev InstanceType=t3.micro"
        ;;
    "staging")
        STACK_NAME="$APPLICATION_NAME-staging"
        PARAMETER_OVERRIDES="Environment=staging InstanceType=t3.small"
        ;;
    "production")
        STACK_NAME="$APPLICATION_NAME-production"
        PARAMETER_OVERRIDES="Environment=production InstanceType=t3.medium"
        ;;
    *)
        echo "Error: Unknown environment $ENVIRONMENT"
        exit 1
        ;;
esac

# Build application (if needed)
if [[ -f "go.mod" ]]; then
    echo "Building Go application..."
    GOOS=linux GOARCH=amd64 go build -o main .
fi

# Deploy using SAM/CloudFormation
if [[ -f "template.yaml" || -f "template.yml" ]]; then
    echo "Deploying SAM application..."
    sam build
    sam deploy \
        --stack-name "$STACK_NAME" \
        --parameter-overrides $PARAMETER_OVERRIDES \
        --capabilities CAPABILITY_IAM \
        --region "$AWS_REGION" \
        --no-confirm-changeset \
        --no-fail-on-empty-changeset
elif [[ -f "cloudformation.yaml" ]]; then
    echo "Deploying CloudFormation stack..."
    aws cloudformation deploy \
        --template-file cloudformation.yaml \
        --stack-name "$STACK_NAME" \
        --parameter-overrides $PARAMETER_OVERRIDES \
        --capabilities CAPABILITY_IAM \
        --region "$AWS_REGION"
else
    echo "No deployment template found"
    exit 1
fi

# Get stack outputs
echo "Deployment completed. Stack outputs:"
aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --region "$AWS_REGION" \
    --query 'Stacks[0].Outputs[*].[OutputKey,OutputValue]' \
    --output table

echo "Deployment to $ENVIRONMENT completed successfully!"
```

### Jenkins Role Assumption Script

```bash
#!/bin/bash  
# scripts/jenkins-assume-role.sh

set -euo pipefail

# Jenkins-specific role assumption with credential file management
ROLE_ARN="${TARGET_ROLE}"
EXTERNAL_ID="${EXTERNAL_ID}"
SESSION_NAME="${ROLE_SESSION_NAME:-jenkins-${BUILD_NUMBER}}"

echo "Jenkins: Assuming role $ROLE_ARN"

# Create temporary credentials file
TEMP_CREDS_FILE="$HOME/.aws/credentials.tmp"

# Assume role and save credentials
aws sts assume-role \
    --role-arn "$ROLE_ARN" \
    --role-session-name "$SESSION_NAME" \
    --external-id "$EXTERNAL_ID" \
    --query 'Credentials.[AccessKeyId,SecretAccessKey,SessionToken]' \
    --output text | {
    read ACCESS_KEY SECRET_KEY SESSION_TOKEN
    
    # Create AWS credentials file
    mkdir -p ~/.aws
    cat > "$TEMP_CREDS_FILE" << EOF
[default]
aws_access_key_id = $ACCESS_KEY
aws_secret_access_key = $SECRET_KEY
aws_session_token = $SESSION_TOKEN
EOF
    
    # Move to active credentials
    mv "$TEMP_CREDS_FILE" ~/.aws/credentials
    
    echo "Jenkins: Credentials configured successfully"
}

# Verify credentials
aws sts get-caller-identity
```

## 🏗️ Terraform Infrastructure

### OIDC Provider for GitHub Actions

```hcl
# terraform/github-oidc.tf
resource "aws_iam_openid_connect_provider" "github_actions" {
  url = "https://token.actions.githubusercontent.com"
  
  client_id_list = ["sts.amazonaws.com"]
  
  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1",
    "1c58a3a8518e8759bf075b76b750d4f2df264fcd"
  ]
  
  tags = {
    Name        = "${var.environment}-github-actions-oidc"
    Environment = var.environment
    Project     = "aws-remote-access-patterns"
  }
}

resource "aws_iam_role" "github_actions_role" {
  name = "${var.environment}-github-actions-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRoleWithWebIdentity"
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.github_actions.arn
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
  
  tags = {
    Name        = "${var.environment}-github-actions-role"
    Environment = var.environment
    Project     = "aws-remote-access-patterns"
  }
}

resource "aws_iam_role_policy_attachment" "github_actions_policy" {
  policy_arn = aws_iam_policy.ci_cd_policy.arn
  role       = aws_iam_role.github_actions_role.name
}
```

### Cross-Account Roles

```hcl
# terraform/cross-account-roles.tf
resource "aws_iam_role" "cross_account_deployment_role" {
  for_each = toset(var.environments)
  
  name = "${each.value}-deployment-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          AWS = var.ci_cd_account_roles
        }
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.external_ids[each.value]
          }
          StringLike = {
            "sts:RoleSessionName" = "ci-cd-*"
          }
        }
      }
    ]
  })
  
  tags = {
    Name        = "${each.value}-deployment-role" 
    Environment = each.value
    Project     = "aws-remote-access-patterns"
    Purpose     = "ci-cd-deployment"
  }
}

resource "aws_iam_role_policy" "deployment_policy" {
  for_each = toset(var.environments)
  
  name = "${each.value}-deployment-policy"
  role = aws_iam_role.cross_account_deployment_role[each.value].id
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cloudformation:*",
          "s3:*",
          "lambda:*",
          "apigateway:*",
          "iam:PassRole",
          "logs:*"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:RequestedRegion" = var.allowed_regions
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "iam:CreateRole",
          "iam:DeleteRole", 
          "iam:AttachRolePolicy",
          "iam:DetachRolePolicy",
          "iam:PutRolePolicy",
          "iam:DeleteRolePolicy"
        ]
        Resource = "arn:aws:iam::*:role/${each.value}-*"
      }
    ]
  })
}
```

## 🔒 Security Best Practices

### 1. Least Privilege Access

Each CI/CD role has only the minimum permissions required:

```hcl
# Example: Limited S3 deployment permissions
resource "aws_iam_policy" "limited_s3_policy" {
  name = "ci-cd-s3-deployment"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject", 
          "s3:DeleteObject"
        ]
        Resource = "arn:aws:s3:::${var.deployment_bucket}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = "arn:aws:s3:::${var.deployment_bucket}"
      }
    ]
  })
}
```

### 2. External ID Management

```bash
# Generate secure external IDs for each environment
for env in dev staging production; do
  EXTERNAL_ID=$(openssl rand -hex 32)
  
  # Store in parameter store
  aws ssm put-parameter \
    --name "/ci-cd/${env}/external-id" \
    --value "$EXTERNAL_ID" \
    --type "SecureString" \
    --description "External ID for ${env} environment CI/CD"
done
```

### 3. Session Name Patterns

Use consistent session naming patterns:

```bash
# GitHub Actions
ROLE_SESSION_NAME="GitHubActions-${GITHUB_REPOSITORY}-${GITHUB_RUN_ID}"

# GitLab CI
ROLE_SESSION_NAME="GitLabCI-${CI_PROJECT_NAME}-${CI_PIPELINE_ID}"

# Jenkins  
ROLE_SESSION_NAME="Jenkins-${JOB_NAME}-${BUILD_NUMBER}"
```

## 📊 Monitoring and Alerting

### CloudTrail Monitoring

```hcl
# terraform/monitoring.tf
resource "aws_cloudwatch_log_group" "ci_cd_logs" {
  name              = "/aws/ci-cd/deployment-logs"
  retention_in_days = 30
  
  tags = {
    Environment = var.environment
    Project     = "aws-remote-access-patterns"
  }
}

resource "aws_cloudwatch_metric_alarm" "failed_deployments" {
  alarm_name          = "ci-cd-failed-deployments"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ErrorCount"
  namespace           = "AWS/Lambda"
  period              = "300"
  statistic           = "Sum"
  threshold           = "5"
  alarm_description   = "This metric monitors failed CI/CD deployments"
  
  dimensions = {
    FunctionName = "ci-cd-deployment-function"
  }
  
  alarm_actions = [aws_sns_topic.alerts.arn]
}
```

## 🚨 Troubleshooting

### Common Issues

#### 1. Role Assumption Failed
```bash
# Debug role assumption
aws sts assume-role \
  --role-arn "$ROLE_ARN" \
  --role-session-name "debug-session" \
  --external-id "$EXTERNAL_ID" \
  --debug
```

#### 2. Permission Denied
```bash
# Check current identity
aws sts get-caller-identity

# Test specific permissions
aws iam simulate-principal-policy \
  --policy-source-arn "$ROLE_ARN" \
  --action-names "s3:ListBucket" \
  --resource-arns "arn:aws:s3:::my-bucket"
```

#### 3. External ID Mismatch
```bash
# Verify external ID configuration
aws iam get-role --role-name MyDeploymentRole \
  --query 'Role.AssumeRolePolicyDocument' \
  --output json | jq '.Statement[].Condition'
```

## 📚 Additional Resources

### Documentation
- [GitHub Actions OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [AWS Cross-Account Access](https://docs.aws.amazon.com/IAM/latest/UserGuide/tutorial_cross-account-with-roles.html)
- [GitLab CI AWS Integration](https://docs.gitlab.com/ee/ci/cloud_services/aws/)

### Related Examples
- [Simple SaaS](../simple-saas/) - Basic cross-account patterns
- [Lambda Function](../lambda-function/) - Serverless authentication
- [CLI Tool](../cli-tool/) - Interactive authentication

## 📄 License

This example is part of the AWS Remote Access Patterns project and follows the same MIT license.

---

These CI/CD integration examples demonstrate how to implement secure, temporary credential-based authentication across different CI/CD platforms while maintaining security best practices and operational excellence.