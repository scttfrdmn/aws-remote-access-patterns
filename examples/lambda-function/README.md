# Lambda Function Example

A serverless AWS Lambda function demonstrating secure cross-account role assumption patterns for SaaS applications. This example shows how to implement external tool authentication in a serverless environment using AWS Lambda, API Gateway, and proper IAM configurations.

## 🎯 Features

- **Serverless Architecture**: Built with AWS Lambda and API Gateway
- **Cross-Account Access**: Secure role assumption with external ID validation
- **RESTful API**: Clean API endpoints for various operations
- **Comprehensive Logging**: Structured logging with CloudWatch integration
- **Error Handling**: Proper error responses and dead letter queue configuration
- **Infrastructure as Code**: Complete CloudFormation/SAM template
- **Security Best Practices**: Least privilege IAM policies and secure credential handling
- **Monitoring & Alerting**: CloudWatch alarms and X-Ray tracing
- **Production Ready**: Includes throttling, CORS, and environment configurations

## 🚀 Quick Start

### Prerequisites

- AWS CLI configured with appropriate permissions
- AWS SAM CLI installed
- Go 1.21 or later
- Docker (for local testing)

### 1. Build and Deploy

```bash
# Navigate to the lambda function directory
cd examples/lambda-function

# Build the function
sam build

# Deploy with guided configuration (first time)
sam deploy --guided

# Or deploy with parameters
sam deploy \
  --parameter-overrides \
    Environment=dev \
    CrossAccountRoleArn=arn:aws:iam::123456789012:role/MyRole \
    ExternalId=my-external-id
```

### 2. Test the Function

```bash
# Get the API Gateway URL from outputs
API_URL=$(aws cloudformation describe-stacks \
  --stack-name lambda-function \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiGatewayUrl`].OutputValue' \
  --output text)

# Health check
curl -X POST "$API_URL/" \
  -H "Content-Type: application/json" \
  -d '{"action": "health_check"}'

# Get caller identity
curl -X POST "$API_URL/" \
  -H "Content-Type: application/json" \
  -d '{"action": "get_caller_identity"}'

# List S3 buckets
curl -X POST "$API_URL/" \
  -H "Content-Type: application/json" \
  -d '{"action": "list_s3_buckets"}'
```

### 3. Cross-Account Role Assumption

```bash
# Assume cross-account role
curl -X POST "$API_URL/" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "assume_role",
    "target_role": "arn:aws:iam::123456789012:role/CrossAccountRole",
    "external_id": "your-external-id"
  }'

# List S3 buckets with assumed role
curl -X POST "$API_URL/" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "list_s3_buckets",
    "target_role": "arn:aws:iam::123456789012:role/CrossAccountRole",
    "external_id": "your-external-id"
  }'
```

## 📋 API Reference

### Endpoints

All requests are sent to the API Gateway endpoint as POST requests with JSON payloads.

#### Health Check

```json
{
  "action": "health_check"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Lambda function is healthy",
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

#### Get Caller Identity

```json
{
  "action": "get_caller_identity",
  "target_role": "arn:aws:iam::123456789012:role/Role" // optional
}
```

**Response:**
```json
{
  "success": true,
  "message": "Caller identity retrieved successfully",
  "data": {
    "user_id": "AIDACKCEVSQ6C2EXAMPLE",
    "account": "123456789012",
    "arn": "arn:aws:sts::123456789012:assumed-role/lambda-role/lambda-function"
  }
}
```

#### Assume Role

```json
{
  "action": "assume_role",
  "target_role": "arn:aws:iam::123456789012:role/CrossAccountRole",
  "external_id": "your-external-id"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Role assumed successfully",
  "data": {
    "user_id": "AIDACKCEVSQ6C2EXAMPLE",
    "account": "123456789012",
    "arn": "arn:aws:sts::123456789012:assumed-role/CrossAccountRole/lambda-function-session",
    "expires_at": "2024-01-15T11:30:00Z",
    "session_name": "lambda-function-session"
  }
}
```

#### List S3 Buckets

```json
{
  "action": "list_s3_buckets",
  "target_role": "arn:aws:iam::123456789012:role/Role", // optional
  "external_id": "your-external-id" // optional
}
```

**Response:**
```json
{
  "success": true,
  "message": "Found 3 S3 buckets",
  "data": {
    "buckets": [
      {
        "name": "my-bucket-1",
        "creation_date": "2024-01-01T00:00:00Z"
      },
      {
        "name": "my-bucket-2", 
        "creation_date": "2024-01-02T00:00:00Z"
      }
    ],
    "count": 2
  }
}
```

### Error Responses

All errors follow this format:

```json
{
  "success": false,
  "error": "Error description here"
}
```

Common HTTP status codes:
- `200` - Success
- `400` - Bad Request (invalid action, missing parameters)
- `401` - Unauthorized (authentication failed)
- `500` - Internal Server Error (AWS API errors, Lambda errors)

## 🏗️ Architecture

### Components

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   API Gateway   │───▶│    Lambda    │───▶│  Cross-Account  │
│                 │    │   Function   │    │      Role       │
└─────────────────┘    └──────────────┘    └─────────────────┘
         │                       │                    │
         ▼                       ▼                    ▼
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   CloudWatch    │    │     SQS      │    │   Target AWS    │
│     Logs        │    │     DLQ      │    │    Services     │
└─────────────────┘    └──────────────┘    └─────────────────┘
```

### Security Model

1. **Lambda Execution Role**: Minimal permissions for basic Lambda operation
2. **Cross-Account Role Assumption**: Uses STS AssumeRole with external ID
3. **Temporary Credentials**: All AWS operations use temporary, time-limited credentials
4. **External ID Validation**: Prevents confused deputy attacks
5. **Least Privilege**: Each role has only the permissions needed for its function

### Data Flow

1. Client sends API request to API Gateway
2. API Gateway invokes Lambda function
3. Lambda function validates request and assumes target role (if specified)
4. Lambda function performs AWS operations with assumed credentials
5. Response is returned through API Gateway to client
6. All operations are logged to CloudWatch

## ⚙️ Configuration

### Environment Variables

The Lambda function uses these environment variables:

```bash
ENVIRONMENT=dev                    # Deployment environment
LOG_LEVEL=INFO                    # Logging level
CROSS_ACCOUNT_ROLE_ARN=...        # Default cross-account role ARN
EXTERNAL_ID=...                   # External ID for role assumption
FUNCTION_NAME=...                 # Lambda function name
API_GATEWAY_URL=...               # API Gateway endpoint URL
```

### SAM Parameters

Customize deployment with these parameters:

```yaml
# template.yaml parameters
Environment: dev                   # Environment (dev/staging/prod)
LogLevel: INFO                    # Log level
CrossAccountRoleArn: ""           # Cross-account role ARN
ExternalId: ""                    # External ID for cross-account access
```

### CloudFormation Outputs

The stack provides these outputs:

- `ApiGatewayUrl` - API Gateway endpoint URL
- `LambdaFunctionName` - Lambda function name  
- `LambdaFunctionArn` - Lambda function ARN
- `LambdaExecutionRoleArn` - Lambda execution role ARN

## 🔧 Development

### Local Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build for deployment
GOOS=linux GOARCH=amd64 go build -o main main.go

# Local testing with SAM
sam local start-api

# Invoke function directly
sam local invoke RemoteAccessFunction -e events/test-event.json
```

### Project Structure

```
examples/lambda-function/
├── main.go                      # Lambda function entry point
├── template.yaml                # SAM template
├── go.mod                       # Go module definition
├── README.md                    # This file
├── events/                      # Test events
│   ├── health-check.json
│   ├── assume-role.json
│   └── list-s3.json
└── tests/                       # Test files
    └── main_test.go
```

### Testing Events

Create test events in the `events/` directory:

```json
// events/health-check.json
{
  "httpMethod": "POST",
  "path": "/",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": "{\"action\": \"health_check\"}",
  "requestContext": {
    "identity": {
      "sourceIp": "127.0.0.1"
    }
  }
}
```

### Adding New Actions

1. Add a new case to the switch statement in `HandleRequest`
2. Implement the handler function
3. Add appropriate IAM permissions to the execution role
4. Update API documentation
5. Create test events

Example:

```go
case "my_new_action":
    responseBody, statusCode = f.handleMyNewAction(ctx, requestBody)

func (f *LambdaFunction) handleMyNewAction(ctx context.Context, req RequestBody) (ResponseBody, int) {
    // Implementation here
    return ResponseBody{
        Success: true,
        Message: "Action completed successfully",
        Data:    result,
    }, 200
}
```

## 🚀 Deployment

### Development Environment

```bash
sam deploy --parameter-overrides Environment=dev LogLevel=DEBUG
```

### Staging Environment

```bash
sam deploy \
  --parameter-overrides \
    Environment=staging \
    CrossAccountRoleArn=arn:aws:iam::123456789012:role/StagingRole \
    ExternalId=staging-external-id
```

### Production Environment

```bash
sam deploy \
  --parameter-overrides \
    Environment=prod \
    LogLevel=WARN \
    CrossAccountRoleArn=arn:aws:iam::123456789012:role/ProductionRole \
    ExternalId=production-external-id
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
name: Deploy Lambda Function
on:
  push:
    branches: [main]
    paths: ['examples/lambda-function/**']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - uses: aws-actions/setup-sam@v2
      - name: SAM build
        run: |
          cd examples/lambda-function
          sam build
      - name: SAM deploy
        run: |
          cd examples/lambda-function
          sam deploy --no-confirm-changeset --no-fail-on-empty-changeset
```

## 📊 Monitoring

### CloudWatch Metrics

The function automatically generates these metrics:

- `AWS/Lambda/Duration` - Function execution time
- `AWS/Lambda/Errors` - Function errors
- `AWS/Lambda/Invocations` - Function invocations
- `AWS/Lambda/Throttles` - Function throttles

### CloudWatch Alarms

Pre-configured alarms:

- **Error Alarm**: Triggers when errors exceed 5 in 10 minutes
- **Duration Alarm**: Triggers when average duration exceeds 25 seconds

### X-Ray Tracing

Enable X-Ray tracing for detailed performance analysis:

```bash
aws lambda update-function-configuration \
  --function-name lambda-function-RemoteAccessFunction \
  --tracing-config Mode=Active
```

### Custom Metrics

Add custom metrics to your function:

```go
import "github.com/aws/aws-sdk-go-v2/service/cloudwatch"

// Emit custom metric
func (f *LambdaFunction) emitMetric(metricName string, value float64) {
    // CloudWatch metrics implementation
}
```

## 🔒 Security Considerations

### IAM Best Practices

1. **Least Privilege**: Grant only necessary permissions
2. **Resource-Based Policies**: Use specific resource ARNs when possible
3. **Condition Keys**: Use condition keys to restrict access
4. **External ID**: Always use external ID for cross-account access
5. **Regular Audits**: Review and rotate access keys regularly

### External ID Management

```bash
# Generate secure external ID
EXTERNAL_ID=$(openssl rand -hex 32)

# Store securely (e.g., AWS Systems Manager Parameter Store)
aws ssm put-parameter \
  --name "/lambda-function/external-id" \
  --value "$EXTERNAL_ID" \
  --type "SecureString" \
  --description "External ID for cross-account role assumption"
```

### Network Security

For enhanced security, deploy in a VPC:

```yaml
# Add to template.yaml
VpcConfig:
  SecurityGroupIds:
    - !Ref LambdaSecurityGroup
  SubnetIds:
    - !Ref PrivateSubnet1
    - !Ref PrivateSubnet2
```

## 🚨 Troubleshooting

### Common Issues

#### 1. Role Assumption Failed

```
Error: Failed to assume role: Access Denied
```

**Solutions:**
- Verify the target role exists and is correctly configured
- Check that the external ID matches exactly
- Ensure the Lambda execution role has `sts:AssumeRole` permissions
- Verify the trust policy on the target role allows assumption from Lambda

#### 2. API Gateway Timeout

```
Error: Task timed out after 30.00 seconds
```

**Solutions:**
- Increase Lambda timeout in template.yaml
- Optimize function code for better performance
- Check for network connectivity issues
- Review CloudWatch logs for bottlenecks

#### 3. Permission Denied on S3

```
Error: Access Denied when listing S3 buckets
```

**Solutions:**
- Verify S3 permissions on the assumed role
- Check bucket policies and ACLs
- Ensure the assumed role has the required S3 permissions

### Debug Mode

Enable debug logging:

```bash
sam deploy --parameter-overrides LogLevel=DEBUG
```

### View Logs

```bash
# View Lambda logs
aws logs tail /aws/lambda/dev-remote-access-function --follow

# View API Gateway logs  
aws logs tail /aws/apigateway/dev-remote-access-api --follow
```

## 📚 Additional Resources

### Related Examples

- [Simple SaaS Example](../simple-saas/) - Basic cross-account access
- [CLI Tool Example](../cli-tool/) - Interactive command-line tool
- [Desktop App Example](../desktop-app/) - GUI application

### Documentation

- [AWS Lambda Developer Guide](https://docs.aws.amazon.com/lambda/)
- [AWS SAM Developer Guide](https://docs.aws.amazon.com/serverless-application-model/)
- [Cross-Account Role Assumption](../../docs/cross-account.md)
- [Security Best Practices](../../docs/security.md)

### AWS Services Used

- **AWS Lambda** - Serverless compute
- **API Gateway** - REST API endpoints
- **CloudFormation/SAM** - Infrastructure as Code
- **IAM** - Identity and Access Management
- **STS** - Security Token Service
- **CloudWatch** - Logging and monitoring
- **X-Ray** - Distributed tracing
- **SQS** - Dead letter queue

## 📄 License

This example is part of the AWS Remote Access Patterns project and follows the same MIT license.

---

This Lambda function example demonstrates how to implement secure cross-account access patterns in a serverless environment, providing a foundation for building scalable SaaS applications with proper security controls.