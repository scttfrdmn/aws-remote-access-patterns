#!/bin/bash

# AWS Remote Access Patterns - Deployment Script
# Handles deployment to different environments (dev, staging, production)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT=""
REGION="us-east-1"
SKIP_TESTS=false
SKIP_BUILD=false
DRY_RUN=false
VERBOSE=false

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
usage() {
    cat << EOF
AWS Remote Access Patterns Deployment Script

Usage: $0 [OPTIONS] <environment>

Arguments:
  environment    Target environment (dev, staging, production)

Options:
  -r, --region REGION      AWS region (default: us-east-1)
  -s, --skip-tests        Skip running tests
  -b, --skip-build        Skip building binaries
  -d, --dry-run           Show what would be done without executing
  -v, --verbose           Enable verbose output
  -h, --help              Show this help message

Examples:
  $0 dev                          # Deploy to development
  $0 production --region us-west-2 # Deploy to production in us-west-2
  $0 staging --dry-run            # Show staging deployment plan

EOF
}

# Function to parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -r|--region)
                REGION="$2"
                shift 2
                ;;
            -s|--skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            -b|--skip-build)
                SKIP_BUILD=true
                shift
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            -*)
                print_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                if [ -z "$ENVIRONMENT" ]; then
                    ENVIRONMENT="$1"
                else
                    print_error "Multiple environments specified"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    if [ -z "$ENVIRONMENT" ]; then
        print_error "Environment is required"
        usage
        exit 1
    fi
    
    if [[ ! "$ENVIRONMENT" =~ ^(dev|development|staging|stage|production|prod)$ ]]; then
        print_error "Invalid environment: $ENVIRONMENT"
        print_error "Valid environments: dev, staging, production"
        exit 1
    fi
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if AWS CLI is installed
    if ! command -v aws >/dev/null 2>&1; then
        print_error "AWS CLI is not installed"
        exit 1
    fi
    
    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed"
        exit 1
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity >/dev/null 2>&1; then
        print_error "AWS credentials not configured or invalid"
        print_error "Run 'aws configure' to set up credentials"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to load environment configuration
load_config() {
    local config_file="config/${ENVIRONMENT}.yaml"
    
    if [ ! -f "$config_file" ]; then
        print_warning "Configuration file not found: $config_file"
        print_status "Creating default configuration..."
        
        mkdir -p config
        cat > "$config_file" << EOF
# ${ENVIRONMENT} environment configuration
aws:
  region: ${REGION}
  
deployment:
  type: serverless  # or container, lambda
  
application:
  name: aws-remote-access-patterns
  version: latest
  
resources:
  dynamodb_table: aws-remote-access-patterns-${ENVIRONMENT}
  s3_template_bucket: aws-remote-access-patterns-templates-${ENVIRONMENT}
  
monitoring:
  enable_cloudwatch: true
  enable_xray: true
  log_retention_days: 30
EOF
        print_status "Created default configuration at $config_file"
        print_status "Please review and customize the configuration before proceeding"
    fi
    
    print_status "Using configuration: $config_file"
}

# Function to run tests
run_tests() {
    if [ "$SKIP_TESTS" = true ]; then
        print_warning "Skipping tests as requested"
        return
    fi
    
    print_status "Running tests..."
    
    if [ "$DRY_RUN" = true ]; then
        print_status "[DRY RUN] Would run: go test ./... -race -cover"
        return
    fi
    
    if [ "$VERBOSE" = true ]; then
        go test ./... -race -cover -v
    else
        go test ./... -race -cover
    fi
    
    print_success "All tests passed"
}

# Function to build application
build_application() {
    if [ "$SKIP_BUILD" = true ]; then
        print_warning "Skipping build as requested"
        return
    fi
    
    print_status "Building application..."
    
    if [ "$DRY_RUN" = true ]; then
        print_status "[DRY RUN] Would build binaries for $ENVIRONMENT"
        return
    fi
    
    # Create build directory
    mkdir -p build
    
    # Build for Linux (common deployment target)
    GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/simple-saas ./examples/simple-saas
    GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/simple-cli ./examples/simple-cli
    
    # Also build for current OS for testing
    go build -o build/simple-saas-local ./examples/simple-saas
    go build -o build/simple-cli-local ./examples/simple-cli
    
    print_success "Build completed"
}

# Function to deploy infrastructure
deploy_infrastructure() {
    print_status "Deploying infrastructure for $ENVIRONMENT..."
    
    local stack_name="aws-remote-access-patterns-${ENVIRONMENT}"
    local template_file="infrastructure/${ENVIRONMENT}.yaml"
    
    if [ "$DRY_RUN" = true ]; then
        print_status "[DRY RUN] Would deploy CloudFormation stack: $stack_name"
        return
    fi
    
    # Create infrastructure template if it doesn't exist
    if [ ! -f "$template_file" ]; then
        print_status "Creating infrastructure template..."
        mkdir -p infrastructure
        
        cat > "$template_file" << EOF
AWSTemplateFormatVersion: '2010-09-09'
Description: 'AWS Remote Access Patterns - ${ENVIRONMENT} environment'

Parameters:
  Environment:
    Type: String
    Default: ${ENVIRONMENT}
    
Resources:
  # DynamoDB table for storing customer credentials
  CredentialsTable:
    Type: AWS::DynamoDB::Table
    Properties:
      TableName: !Sub 'aws-remote-access-patterns-\${Environment}'
      BillingMode: PAY_PER_REQUEST
      PointInTimeRecoverySpecification:
        PointInTimeRecoveryEnabled: true
      SSESpecification:
        SSEEnabled: true
      StreamSpecification:
        StreamViewType: NEW_AND_OLD_IMAGES
      AttributeDefinitions:
        - AttributeName: customer_id
          AttributeType: S
      KeySchema:
        - AttributeName: customer_id
          KeyType: HASH
      Tags:
        - Key: Environment
          Value: !Ref Environment
        - Key: Project
          Value: aws-remote-access-patterns

  # S3 bucket for CloudFormation templates
  TemplateBucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Sub 'aws-remote-access-patterns-templates-\${Environment}'
      BucketEncryption:
        ServerSideEncryptionConfiguration:
          - ServerSideEncryptionByDefault:
              SSEAlgorithm: AES256
      PublicAccessBlockConfiguration:
        BlockPublicAcls: true
        BlockPublicPolicy: true
        IgnorePublicAcls: true
        RestrictPublicBuckets: true
      Tags:
        - Key: Environment
          Value: !Ref Environment
        - Key: Project
          Value: aws-remote-access-patterns

  # IAM role for the application
  ApplicationRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Sub 'aws-remote-access-patterns-\${Environment}-role'
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              Service: lambda.amazonaws.com
            Action: 'sts:AssumeRole'
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
      Policies:
        - PolicyName: DynamoDBAccess
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Action:
                  - 'dynamodb:GetItem'
                  - 'dynamodb:PutItem'
                  - 'dynamodb:DeleteItem'
                  - 'dynamodb:Scan'
                Resource: !GetAtt CredentialsTable.Arn
        - PolicyName: S3Access
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Action:
                  - 's3:GetObject'
                  - 's3:PutObject'
                Resource: !Sub '\${TemplateBucket}/*'

Outputs:
  DynamoDBTable:
    Description: 'DynamoDB table for credentials'
    Value: !Ref CredentialsTable
    Export:
      Name: !Sub '\${AWS::StackName}-DynamoDBTable'
      
  S3TemplateBucket:
    Description: 'S3 bucket for templates'
    Value: !Ref TemplateBucket
    Export:
      Name: !Sub '\${AWS::StackName}-S3TemplateBucket'
      
  ApplicationRole:
    Description: 'IAM role for the application'
    Value: !GetAtt ApplicationRole.Arn
    Export:
      Name: !Sub '\${AWS::StackName}-ApplicationRole'
EOF
    fi
    
    # Deploy the stack
    aws cloudformation deploy \
        --template-file "$template_file" \
        --stack-name "$stack_name" \
        --parameter-overrides Environment="$ENVIRONMENT" \
        --capabilities CAPABILITY_NAMED_IAM \
        --region "$REGION"
    
    print_success "Infrastructure deployment completed"
}

# Function to upload templates
upload_templates() {
    print_status "Uploading CloudFormation templates..."
    
    local bucket_name="aws-remote-access-patterns-templates-${ENVIRONMENT}"
    
    if [ "$DRY_RUN" = true ]; then
        print_status "[DRY RUN] Would upload templates to s3://$bucket_name"
        return
    fi
    
    # Upload templates to S3
    if [ -d "templates" ]; then
        aws s3 sync templates/ "s3://$bucket_name/templates/" \
            --region "$REGION" \
            --delete
        print_success "Templates uploaded to S3"
    else
        print_warning "No templates directory found"
    fi
}

# Function to deploy application
deploy_application() {
    print_status "Deploying application..."
    
    if [ "$DRY_RUN" = true ]; then
        print_status "[DRY RUN] Would deploy application to $ENVIRONMENT"
        return
    fi
    
    # For this example, we'll create a simple deployment
    # In a real scenario, you might use AWS Lambda, ECS, or other services
    
    case "$ENVIRONMENT" in
        dev|development)
            print_status "Development deployment - running locally"
            print_status "Application built and ready for local testing"
            ;;
        staging|stage)
            print_status "Staging deployment - would deploy to staging environment"
            # Here you would deploy to your staging infrastructure
            ;;
        production|prod)
            print_status "Production deployment - would deploy to production environment"
            # Here you would deploy to your production infrastructure
            print_warning "Production deployment requires additional confirmation"
            ;;
    esac
    
    print_success "Application deployment completed"
}

# Function to verify deployment
verify_deployment() {
    print_status "Verifying deployment..."
    
    if [ "$DRY_RUN" = true ]; then
        print_status "[DRY RUN] Would verify deployment"
        return
    fi
    
    # Check CloudFormation stack status
    local stack_name="aws-remote-access-patterns-${ENVIRONMENT}"
    local stack_status
    
    stack_status=$(aws cloudformation describe-stacks \
        --stack-name "$stack_name" \
        --region "$REGION" \
        --query 'Stacks[0].StackStatus' \
        --output text 2>/dev/null || echo "NOT_FOUND")
    
    if [ "$stack_status" = "CREATE_COMPLETE" ] || [ "$stack_status" = "UPDATE_COMPLETE" ]; then
        print_success "CloudFormation stack is healthy"
    else
        print_error "CloudFormation stack status: $stack_status"
    fi
    
    # Test basic functionality
    if [ -f "build/simple-cli-local" ]; then
        print_status "Testing CLI functionality..."
        if ./build/simple-cli-local --help >/dev/null 2>&1; then
            print_success "CLI test passed"
        else
            print_warning "CLI test failed"
        fi
    fi
    
    print_success "Deployment verification completed"
}

# Function to show deployment summary
show_summary() {
    print_success "Deployment Summary"
    echo "===================="
    echo "Environment: $ENVIRONMENT"
    echo "Region: $REGION"
    echo "Stack: aws-remote-access-patterns-${ENVIRONMENT}"
    echo
    
    if [ "$DRY_RUN" = false ]; then
        # Get outputs from CloudFormation
        local outputs
        outputs=$(aws cloudformation describe-stacks \
            --stack-name "aws-remote-access-patterns-${ENVIRONMENT}" \
            --region "$REGION" \
            --query 'Stacks[0].Outputs' \
            --output table 2>/dev/null || echo "No outputs available")
        
        echo "CloudFormation Outputs:"
        echo "$outputs"
    else
        echo "[DRY RUN] No actual deployment performed"
    fi
}

# Main deployment function
main() {
    parse_args "$@"
    
    print_status "Starting deployment to $ENVIRONMENT environment..."
    
    if [ "$VERBOSE" = true ]; then
        set -x
    fi
    
    check_prerequisites
    load_config
    run_tests
    build_application
    deploy_infrastructure
    upload_templates
    deploy_application
    verify_deployment
    show_summary
    
    print_success "Deployment completed successfully! 🚀"
    
    if [ "$ENVIRONMENT" = "dev" ] || [ "$ENVIRONMENT" = "development" ]; then
        echo
        print_status "Development environment ready!"
        echo "Try running: ./build/simple-cli-local --help"
    fi
}

# Run main function
main "$@"