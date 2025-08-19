# SaaS Service Example

A complete SaaS service implementation demonstrating secure AWS cross-account access patterns with a modern web UI for customer onboarding.

## Features

- **Modern Web UI**: Complete customer onboarding experience with responsive design
- **Secure Cross-Account Access**: Uses AWS IAM roles with external IDs for maximum security
- **One-Click Integration**: Customers can set up AWS integration with a single CloudFormation link
- **Real-Time Status**: Track integration status and health in real-time
- **Production Ready**: Includes structured logging, middleware, graceful shutdown, and error handling
- **Enterprise Security**: Follows AWS security best practices with audit logging

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      SaaS Service                           │
├─────────────────────────────────────────────────────────────┤
│  Web UI (HTML/CSS/JS)                                      │
│      │                                                      │
│      ├─── Customer Dashboard ──── Integration Status       │
│      ├─── Integration Wizard ──── Setup Progress          │
│      └─── Admin Panel ──────────── Customer Management     │
│                                                             │
│  HTTP Handlers                                              │
│      │                                                      │
│      ├─── Customer CRUD ──────────── In-Memory Store       │
│      ├─── Setup Link Generation ──── CrossAccount Client   │
│      └─── Integration Completion ──── Role Validation      │
│                                                             │
│  Middleware                                                 │
│      │                                                      │
│      ├─── Request Logging ──────────── Structured JSON     │
│      ├─── Error Recovery ──────────── Panic Handling       │
│      ├─── CORS Support ────────────── Cross-Origin         │
│      └─── Request ID ──────────────── Tracing              │
└─────────────────────────────────────────────────────────────┘
```

## Running the Example

### Prerequisites

- Go 1.21 or later
- AWS CLI configured with appropriate permissions
- Your AWS account ID and an S3 bucket for CloudFormation templates

### Quick Start

1. **Set up environment variables:**
```bash
export SERVICE_NAME="MyDataPlatform"
export SERVICE_ACCOUNT_ID="123456789012"  # Your AWS account ID
export TEMPLATE_S3_BUCKET="mydataplatform-templates"  # Your template bucket
export AWS_REGION="us-east-1"
export ENVIRONMENT="development"
```

2. **Run the service:**
```bash
cd examples/saas-service
go run main.go
```

3. **Open your browser:**
```
http://localhost:8080
```

### Configuration Options

The service can be configured via environment variables or command-line flags:

```bash
# Environment Variables
PORT=8080                                    # HTTP port (default: 8080)
SERVICE_NAME="MyDataPlatform"               # Your service name
SERVICE_ACCOUNT_ID="123456789012"           # Your AWS account ID
TEMPLATE_S3_BUCKET="mydataplatform-templates" # S3 bucket for templates
AWS_REGION="us-east-1"                      # AWS region
ENVIRONMENT="development"                    # Environment (dev/staging/prod)
LOG_LEVEL="info"                            # Log level (debug/info/warn/error)

# Command Line Options
go run main.go -config config.json         # Load from config file
go run main.go -show-config                # Show current configuration
```

## Using the Web Interface

### 1. Customer Dashboard

The main dashboard shows:
- **Customer Statistics**: Total customers, active integrations, pending setups
- **Customer Management**: View, add, edit, and delete customers
- **Integration Status**: Real-time status of AWS integrations
- **Quick Actions**: Generate setup links, test integrations

### 2. Integration Wizard

Customer-facing integration page includes:
- **Step-by-Step Guide**: Clear instructions for AWS setup
- **Security Information**: Transparent permissions and security details
- **One-Click Setup**: Direct CloudFormation deployment link
- **Status Tracking**: Real-time integration progress

### 3. API Endpoints

The service exposes REST API endpoints:

```bash
# Health and Status
GET  /health                    # Health check
GET  /ready                     # Readiness check

# Customer Management
GET  /api/customers             # List all customers
POST /api/customers             # Create new customer
GET  /api/customers/{id}        # Get customer details
DELETE /api/customers/{id}      # Delete customer

# Integration Management
POST /api/customers/{id}/setup  # Generate setup link
POST /api/customers/{id}/complete # Complete setup
GET  /integrate/status/{id}     # Check integration status

# Customer-Facing
GET  /integrate                 # Integration wizard page
POST /integrate                 # Start integration process
```

## Customer Integration Flow

### For Your Customers

1. **Visit Integration Page**: Customer goes to `/integrate`
2. **Enter Company Details**: Company name, email, AWS account ID (optional)
3. **Review Permissions**: See exactly what permissions will be granted
4. **Deploy CloudFormation**: One-click link opens AWS Console
5. **Automatic Verification**: Service validates the integration

### For Your Service

1. **Generate Setup Link**: Create unique CloudFormation deployment URL
2. **Customer Deploys Stack**: Customer runs CloudFormation in their account
3. **Validate Integration**: Test that the cross-account role works
4. **Start Using Service**: Begin accessing customer AWS resources securely

## Security Features

### Built-In Security

- **Temporary Credentials**: All access uses short-lived STS tokens
- **External IDs**: Unique external IDs prevent confused deputy attacks
- **Least Privilege**: Minimal required permissions, separated by setup vs ongoing
- **Audit Logging**: All requests logged with structured JSON
- **CORS Protection**: Configurable cross-origin access controls
- **Security Headers**: XSS protection, content type sniffing prevention

### Customer Security

- **No Long-Lived Keys**: No access keys stored anywhere
- **Instant Revocation**: Customer can revoke access by deleting CloudFormation stack
- **Full Visibility**: All actions logged in customer's CloudTrail
- **Permission Transparency**: Clear documentation of required permissions

## Production Deployment

### Environment Configuration

For production deployment, set these additional variables:

```bash
ENVIRONMENT="production"
LOG_LEVEL="info"
PORT="8080"

# Database (replace in-memory store)
DYNAMODB_TABLE="customers-prod"
KMS_KEY_ID="alias/saas-service-prod"

# Monitoring
ENABLE_METRICS="true"
ENABLE_TRACING="true"

# Security
ENABLE_RATE_LIMITING="true"
CORS_ORIGINS="https://yourdomain.com"
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./main"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: saas-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: saas-service
  template:
    metadata:
      labels:
        app: saas-service
    spec:
      containers:
      - name: saas-service
        image: your-registry/saas-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: SERVICE_NAME
          value: "MyDataPlatform"
        - name: SERVICE_ACCOUNT_ID
          value: "123456789012"
        - name: TEMPLATE_S3_BUCKET
          value: "mydataplatform-templates"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Customization

### Branding

Modify the web templates in `web/templates/` to match your brand:
- Update colors in `base.html`
- Replace service name and logos
- Customize integration messaging

### Permissions

Adjust required AWS permissions in `main.go`:

```go
OngoingPermissions: []crossaccount.Permission{
    {
        Sid:    "YourCustomPermission",
        Effect: "Allow",
        Actions: []string{
            "service:Action1",
            "service:Action2",
        },
        Resources: []string{
            "arn:aws:service:::resource/*",
        },
    },
},
```

### Storage Backend

Replace the in-memory customer store with a persistent backend:

```go
// Replace in handlers.go
type Handler struct {
    customerDB CustomerDatabase // Instead of map[string]*Customer
    // ... other fields
}
```

## Monitoring and Observability

### Structured Logging

All logs are JSON formatted with consistent fields:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "Request completed",
  "request_id": "req-abc123",
  "method": "POST",
  "path": "/api/customers",
  "status": 201,
  "duration_ms": 45
}
```

### Health Endpoints

- `/health`: Basic health check
- `/ready`: Readiness check (includes dependency validation)

### Metrics Collection

The service includes hooks for custom metrics:

```go
// Example: Record setup completion
recordSetupCompletion(customerID, success)
```

## Troubleshooting

### Common Issues

1. **Setup Link Generation Fails**
   - Verify `SERVICE_ACCOUNT_ID` and `TEMPLATE_S3_BUCKET` are correct
   - Ensure AWS credentials have S3 access

2. **Customer Integration Fails**
   - Check CloudFormation stack deployment status
   - Verify external ID matches in both systems
   - Confirm customer has admin permissions

3. **Template Not Found**
   - Ensure templates are uploaded to S3 bucket
   - Check bucket permissions and CloudFormation access

### Debug Mode

Enable debug logging for detailed information:

```bash
LOG_LEVEL=debug go run main.go
```

## Support

- 📖 [Main Documentation](../../docs/)
- 🔐 [Security Guide](../../docs/security.md)
- 🚀 [Deployment Guide](../../docs/deployment.md)
- 📞 [Issue Tracker](https://github.com/scttfrdmn/aws-remote-access-patterns/issues)

---

This example demonstrates a production-ready SaaS service with secure AWS integration. Use it as a foundation for building your own customer-facing AWS services.