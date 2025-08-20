# AWS CLI Helper

A secure credential helper that integrates AWS Remote Access Patterns with the standard AWS CLI, providing temporary credentials without long-lived access keys.

## 🎯 Purpose

This helper acts as a bridge between AWS CLI and our secure authentication patterns, enabling:

- **Temporary Credentials**: All credentials expire automatically (configurable duration)
- **Multiple Auth Methods**: SSO, profiles, cross-account roles, and IAM users
- **Secure Caching**: Encrypted credential cache with automatic cleanup
- **Seamless Integration**: Drop-in replacement for static AWS CLI credentials
- **CI/CD Support**: Special modes for automated environments

## 🚀 Quick Start

### Installation

```bash
# Build the helper
cd examples/aws-cli-helper
go build -o aws-cli-helper main.go

# Install globally (optional)
sudo mv aws-cli-helper /usr/local/bin/

# Or add to your PATH
export PATH="$PATH:$(pwd)"
```

### Setup Your First Profile

```bash
# Interactive setup
aws-cli-helper --setup --profile myservice

# Follow the prompts to configure authentication
```

### Configure AWS CLI

```bash
# Configure AWS CLI to use the helper
aws configure set credential_process "aws-cli-helper --profile myservice" --profile myservice
aws configure set region us-east-1 --profile myservice

# Test the integration
aws sts get-caller-identity --profile myservice
```

## 🔧 Configuration

### Configuration File

The helper stores configuration in `~/.aws-remote-access/config.yaml`:

```yaml
profiles:
  myservice:
    tool_name: "myservice-cli"
    auth_method: "sso"
    region: "us-east-1"
    session_duration: 3600
    sso_config:
      start_url: "https://myorg.awsapps.com/start"
      region: "us-east-1"

  production-access:
    tool_name: "prod-cli"
    auth_method: "cross_account"
    region: "us-east-1" 
    session_duration: 1800
    cross_account:
      customer_id: "acme-corp"
      role_arn: "arn:aws:iam::999999999999:role/MyService-CrossAccount"
      external_id: "MyService-acme-corp-abc123def456"

cache:
  directory: "~/.aws-remote-access/cache"
  max_age: 3300  # 55 minutes (5 min buffer)

logging:
  level: "info"
  file: "~/.aws-remote-access/aws-cli-helper.log"
```

### Authentication Methods

#### 1. AWS SSO (Recommended)

```yaml
profiles:
  myservice-sso:
    auth_method: "sso"
    sso_config:
      start_url: "https://company.awsapps.com/start"
      region: "us-east-1"
```

#### 2. AWS Profile

```yaml
profiles:
  myservice-profile:
    auth_method: "profile"
    profile_name: "my-base-profile"
```

#### 3. Cross-Account Role

```yaml
profiles:
  customer-access:
    auth_method: "cross_account"
    cross_account:
      customer_id: "customer-123"
      role_arn: "arn:aws:iam::CUSTOMER-ACCOUNT:role/MyService-Role"
      external_id: "unique-external-id"
```

#### 4. IAM User (Not Recommended)

```yaml
profiles:
  legacy-access:
    auth_method: "iam_user"
    iam_user:
      access_key_id: "AKIA..."
      secret_access_key: "secret..."
```

## 📋 Usage Examples

### Basic Usage

```bash
# Get JSON credentials (AWS CLI format)
aws-cli-helper --profile myservice

# Export as environment variables
eval $(aws-cli-helper --export --profile myservice)

# Check credential status
aws-cli-helper --check --profile myservice

# Force refresh
aws-cli-helper --refresh --profile myservice
```

### Setup and Management

```bash
# Interactive setup
aws-cli-helper --setup --profile newservice

# List all profiles
aws-cli-helper --list-profiles

# Validate configuration
aws-cli-helper --validate --profile myservice

# Health check
aws-cli-helper --health-check --profile myservice
```

### CI/CD Mode

```bash
# Enable CI/CD optimizations
aws-cli-helper --profile deployment --ci-mode

# Export for CI/CD pipelines
eval $(aws-cli-helper --export --profile deployment --ci-mode)
```

### Debug and Troubleshooting

```bash
# Enable debug logging
aws-cli-helper --debug --profile myservice

# Generate usage report
aws-cli-helper --usage-report

# Check version
aws-cli-helper --version
```

## 🔒 Security Features

### Encrypted Credential Cache

- **AES-GCM Encryption**: All cached credentials are encrypted at rest
- **Automatic Expiration**: Credentials are automatically cleaned up when expired
- **Secure Key Storage**: Encryption keys are generated per installation
- **File Permissions**: Cache files use restrictive permissions (0600)

### Audit Logging

All operations are logged with structured JSON:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "event": "credential_request", 
  "profile": "myservice",
  "success": true,
  "duration_ms": 245
}
```

### Credential Validation

- **Expiration Checking**: Automatic validation of credential expiration
- **Permission Testing**: Optional validation of required permissions
- **Error Recovery**: Automatic retry and fallback mechanisms

## 🏗️ Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                    AWS CLI Helper                           │
├─────────────────────────────────────────────────────────────┤
│  CLI Interface                                              │
│      │                                                      │
│      ├─── Configuration ─────────── YAML Config File       │
│      ├─── Caching ───────────────── Encrypted Cache        │  
│      └─── Providers ─────────────── Authentication         │
│                                                             │
│  Provider Types                                             │
│      │                                                      │
│      ├─── SSO Provider ─────────── AWS SSO Integration     │
│      ├─── Profile Provider ──────── ~/.aws/credentials     │
│      ├─── Cross-Account Provider ── Role Assumption        │
│      └─── IAM User Provider ─────── Static Keys → STS      │
│                                                             │
│  Output Formats                                             │
│      │                                                      │
│      ├─── AWS CLI JSON ─────────── credential_process      │
│      └─── Environment Variables ── export statements       │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **AWS CLI Request**: AWS CLI invokes the helper via `credential_process`
2. **Cache Check**: Helper checks for valid cached credentials
3. **Provider Selection**: If no cache, select appropriate provider
4. **Authentication**: Provider authenticates using configured method
5. **Credential Retrieval**: Get temporary AWS credentials
6. **Caching**: Encrypt and cache credentials for future use
7. **Response**: Return credentials in AWS CLI JSON format

## 🛠️ Development

### Building

```bash
# Build for current platform
go build -o aws-cli-helper main.go

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o aws-cli-helper-linux main.go
GOOS=darwin GOARCH=amd64 go build -o aws-cli-helper-darwin main.go
GOOS=windows GOARCH=amd64 go build -o aws-cli-helper-windows.exe main.go
```

### Testing

```bash
# Run unit tests
go test ./...

# Test integration with AWS CLI
aws sts get-caller-identity --profile test-profile

# Test credential caching
aws-cli-helper --debug --profile test-profile
aws-cli-helper --check --profile test-profile
```

### Configuration for Development

```yaml
# ~/.aws-remote-access/config.yaml
profiles:
  dev-test:
    tool_name: "dev-test-cli"
    auth_method: "profile"
    profile_name: "default"
    region: "us-east-1"
    session_duration: 3600

cache:
  directory: "~/.aws-remote-access/cache"
  max_age: 300  # 5 minutes for development

logging:
  level: "debug"
  file: "~/.aws-remote-access/aws-cli-helper.log"
```

## 🔧 Integration Patterns

### Pattern 1: Developer Workstation

Multiple profiles for different environments:

```bash
# ~/.aws/config
[profile dev]
credential_process = aws-cli-helper --profile dev
region = us-east-1

[profile staging] 
credential_process = aws-cli-helper --profile staging
region = us-east-1

[profile prod]
credential_process = aws-cli-helper --profile prod
region = us-east-1
```

### Pattern 2: CI/CD Pipeline

```bash
# In CI/CD script
export AWS_PROFILE=deployment
eval $(aws-cli-helper --export --profile deployment --ci-mode)

# All AWS commands now use temporary credentials
aws s3 sync ./build/ s3://deployment-bucket/
aws cloudformation deploy --template-file infra.yaml
```

### Pattern 3: Customer Support

Time-limited access to customer accounts:

```bash
# Generate temporary customer access
aws-cli-helper --setup --profile support-customer-123
aws ec2 describe-instances --profile support-customer-123

# Access automatically expires after configured duration
```

## 🚨 Troubleshooting

### Common Issues

#### 1. No Credentials Error
```
Error: NoCredentialProviders: no valid providers in chain
```

**Solution:**
```bash
# Check if helper is properly configured
aws configure list --profile myservice

# Test helper directly
aws-cli-helper --profile myservice

# Check helper path
which aws-cli-helper
```

#### 2. Permission Denied
```
Error: AccessDenied: User is not authorized to perform: sts:AssumeRole
```

**Solution:**
```bash
# Check what identity is being used
aws sts get-caller-identity --profile myservice

# Validate profile configuration
aws-cli-helper --validate --profile myservice
```

#### 3. Expired Token
```
Error: ExpiredToken: The security token included in the request is expired
```

**Solution:**
```bash
# Force credential refresh
aws-cli-helper --refresh --profile myservice

# Check cache status
aws-cli-helper --check --profile myservice
```

#### 4. Cache Issues
```
Error: Failed to decrypt cached credentials
```

**Solution:**
```bash
# Clear corrupted cache
rm -rf ~/.aws-remote-access/cache/

# Regenerate credentials
aws-cli-helper --profile myservice
```

### Debug Mode

```bash
# Enable comprehensive debugging
export AWS_CLI_HELPER_DEBUG=1
aws-cli-helper --debug --profile myservice

# Check logs
tail -f ~/.aws-remote-access/aws-cli-helper.log
```

### Health Check

```bash
# Run full health check
aws-cli-helper --health-check --profile myservice

✅ Configuration valid
✅ Authentication working
✅ Credentials cached  
✅ AWS CLI integration working
```

## 🔄 Migration

### From Static Credentials

```bash
# Before: ~/.aws/credentials
[myservice]
aws_access_key_id = AKIA...
aws_secret_access_key = secret...

# After: ~/.aws/config
[profile myservice]
credential_process = aws-cli-helper --profile myservice
region = us-east-1
```

### From AWS SSO

```bash
# Enhance existing SSO with caching and additional features
aws-cli-helper --setup --profile myservice-sso
```

## 📊 Monitoring

### Usage Metrics

```bash
# Generate usage report
aws-cli-helper --usage-report

Profile: myservice
├─ Total requests: 1,247
├─ Success rate: 99.2%
├─ Cache hit rate: 87.3%
└─ Average duration: 234ms
```

### Log Analysis

```bash
# Analyze logs
grep "credential_request" ~/.aws-remote-access/aws-cli-helper.log | jq
```

## 📚 References

- 🔗 [Cross-Account Access Pattern](../../docs/cross-account.md)
- 🖥️ [External Tool Access Pattern](../../docs/external-tool.md)
- 📖 [API Reference](../../docs/api-reference.md)
- 🛡️ [Security Best Practices](../../docs/security.md)
- 🚀 [AWS CLI Integration Guide](../../docs/aws-cli-integration.md)

---

This helper provides a secure, convenient way to use AWS CLI with temporary credentials while maintaining the familiar AWS CLI experience your team already knows.