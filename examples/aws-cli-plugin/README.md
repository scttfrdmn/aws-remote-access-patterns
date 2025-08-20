# AWS CLI Credential Provider Plugin

A credential provider plugin for AWS CLI that integrates with the AWS Remote Access Patterns authentication system. This plugin enables seamless authentication using SSO, cross-account roles, and interactive methods directly from the AWS CLI.

## 🎯 Features

- **AWS CLI Integration**: Direct integration with AWS CLI credential process
- **Multiple Auth Methods**: SSO, cross-account roles, and interactive authentication
- **Secure Credential Caching**: Encrypted credential storage and automatic refresh
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Easy Setup**: Interactive configuration wizard
- **Debug Support**: Comprehensive logging for troubleshooting
- **Zero Dependencies**: No external credential helpers required

## 🚀 Quick Start

### 1. Build and Install

```bash
# Navigate to the plugin directory
cd examples/aws-cli-plugin

# Build the plugin
go build -o aws-remote-access-cli-plugin main.go

# Make executable (Unix/Linux/macOS)
chmod +x aws-remote-access-cli-plugin

# Move to PATH (optional)
sudo mv aws-remote-access-cli-plugin /usr/local/bin/
```

### 2. Setup Configuration

```bash
# Run interactive setup
./aws-remote-access-cli-plugin setup
```

The setup wizard will guide you through:
- Profile name selection
- Authentication method configuration
- AWS region selection
- Session duration settings

### 3. Configure AWS CLI

Add the following to your AWS CLI config file (`~/.aws/config`):

```ini
[profile remote-access]
credential_process = aws-remote-access-cli-plugin get-credentials
region = us-east-1
```

### 4. Use with AWS CLI

```bash
# Test the configuration
aws --profile remote-access sts get-caller-identity

# Use with any AWS service
aws --profile remote-access s3 ls
aws --profile remote-access ec2 describe-instances
```

## ⚙️ Configuration

### Authentication Methods

#### 1. AWS SSO

```bash
# During setup, choose SSO and provide:
# - SSO Start URL: https://my-company.awsapps.com/start
# - Region: us-east-1
```

#### 2. Cross-Account Role

```bash
# During setup, choose cross-account and provide:
# - Role ARN: arn:aws:iam::123456789012:role/MyRole
# - External ID: my-external-id (optional)
```

#### 3. Interactive Authentication

```bash
# During setup, choose interactive
# The plugin will guide you through authentication on first use
```

### Configuration File

The plugin stores configuration in `~/.aws-remote-access-patterns/plugin-config.json`:

```json
{
  "profile_name": "remote-access",
  "auth_method": "sso",
  "aws_region": "us-east-1",
  "session_duration": 3600,
  "sso_start_url": "https://my-company.awsapps.com/start",
  "cache_enabled": true,
  "debug": false
}
```

### Environment Variables

```bash
export AWS_REMOTE_ACCESS_DEBUG=true          # Enable debug logging
export AWS_REMOTE_ACCESS_CONFIG=/path/config # Override config location
```

## 🔧 Usage

### Command Reference

#### get-credentials
Used by AWS CLI to retrieve credentials:
```bash
aws-remote-access-cli-plugin get-credentials
```

#### setup
Interactive configuration wizard:
```bash
aws-remote-access-cli-plugin setup
```

#### test
Test current configuration:
```bash
aws-remote-access-cli-plugin test
```

#### info
Display plugin information:
```bash
aws-remote-access-cli-plugin info
```

#### clear
Clear plugin configuration:
```bash
aws-remote-access-cli-plugin clear
```

#### version
Display version information:
```bash
aws-remote-access-cli-plugin version
```

### Advanced Usage

#### Multiple Profiles

Configure multiple profiles for different environments:

```ini
# ~/.aws/config
[profile dev]
credential_process = aws-remote-access-cli-plugin get-credentials
region = us-east-1

[profile staging] 
credential_process = aws-remote-access-cli-plugin get-credentials
region = us-west-2

[profile production]
credential_process = aws-remote-access-cli-plugin get-credentials
region = eu-west-1
```

#### Custom Session Names

For cross-account roles, customize session names:

```json
{
  "cross_account": {
    "role_arn": "arn:aws:iam::123456789012:role/MyRole",
    "external_id": "my-external-id",
    "session_name": "my-custom-session"
  }
}
```

#### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Environment variable
export AWS_REMOTE_ACCESS_DEBUG=true
aws --profile remote-access sts get-caller-identity

# Or in config file
{
  "debug": true
}
```

## 🔒 Security Features

### Credential Protection

- **No Long-lived Credentials**: Uses temporary credentials with automatic expiration
- **Encrypted Storage**: Configuration and cached credentials are encrypted
- **Minimal Permissions**: Requests only necessary permissions
- **Session Isolation**: Each session is isolated with unique session names

### External ID Support

For cross-account access, external IDs provide additional security:

```json
{
  "cross_account": {
    "role_arn": "arn:aws:iam::123456789012:role/MyRole",
    "external_id": "unique-external-id-for-security"
  }
}
```

### Audit Trail

All authentication events are logged:

```bash
# View logs (when debug is enabled)
tail -f ~/.aws-remote-access-patterns/plugin.log
```

## 🚨 Troubleshooting

### Common Issues

#### 1. Plugin Not Found

```
Error: credential process returned non-zero exit status
```

**Solutions:**
- Ensure plugin is in PATH or use full path in credential_process
- Check file permissions: `chmod +x aws-remote-access-cli-plugin`
- Verify plugin builds correctly: `./aws-remote-access-cli-plugin version`

#### 2. Configuration Not Found

```
Error: failed to load configuration
```

**Solutions:**
- Run setup: `./aws-remote-access-cli-plugin setup`
- Check config file exists: `~/.aws-remote-access-patterns/plugin-config.json`
- Verify permissions: `chmod 600 ~/.aws-remote-access-patterns/plugin-config.json`

#### 3. Authentication Failed

```
Error: failed to get AWS credentials
```

**Solutions:**
- Test configuration: `./aws-remote-access-cli-plugin test`
- Check authentication method configuration
- Verify network connectivity
- Enable debug mode: `AWS_REMOTE_ACCESS_DEBUG=true`

#### 4. Role Assumption Failed

```
Error: failed to assume role
```

**Solutions:**
- Verify role ARN is correct
- Check external ID matches
- Ensure base credentials have `sts:AssumeRole` permission
- Verify role trust policy allows assumption

#### 5. Session Expired

```
Error: The security token included in the request is expired
```

**Solutions:**
- Authentication will automatically refresh
- Force refresh by clearing cache: `./aws-remote-access-cli-plugin clear`
- Check session duration configuration

### Debug Mode

Enable comprehensive logging:

```bash
# Enable debug logging
export AWS_REMOTE_ACCESS_DEBUG=true

# Test with verbose output
./aws-remote-access-cli-plugin test

# Use with AWS CLI
aws --profile remote-access sts get-caller-identity
```

### Log Locations

- Configuration: `~/.aws-remote-access-patterns/plugin-config.json`
- Cache: `~/.aws-remote-access-patterns/cache/`
- Logs: `~/.aws-remote-access-patterns/plugin.log` (when debug enabled)

## 🏗️ Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/example/aws-remote-access-patterns.git
cd aws-remote-access-patterns/examples/aws-cli-plugin

# Download dependencies
go mod download

# Build
go build -o aws-remote-access-cli-plugin main.go

# Test build
./aws-remote-access-cli-plugin version
```

### Cross-Platform Builds

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o aws-remote-access-cli-plugin.exe main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o aws-remote-access-cli-plugin-darwin main.go

# Linux
GOOS=linux GOARCH=amd64 go build -o aws-remote-access-cli-plugin-linux main.go
```

### Testing

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# Test with different configurations
./aws-remote-access-cli-plugin setup
./aws-remote-access-cli-plugin test
```

### Code Structure

```
examples/aws-cli-plugin/
├── main.go                     # Plugin entry point
├── internal/
│   ├── auth/
│   │   └── manager.go          # Authentication management
│   └── config/
│       └── config.go           # Configuration handling
├── go.mod                      # Go module definition
└── README.md                   # This file
```

## 📚 Integration Examples

### With Terraform

```hcl
# Use with Terraform AWS provider
provider "aws" {
  profile = "remote-access"
  region  = "us-east-1"
}
```

### With Boto3 (Python)

```python
# Use with boto3
import boto3

session = boto3.Session(profile_name='remote-access')
s3 = session.client('s3')
```

### With AWS SDK for Go

```go
// Use with AWS SDK for Go
import "github.com/aws/aws-sdk-go-v2/config"

cfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithSharedConfigProfile("remote-access"),
)
```

### With Docker

```bash
# Mount AWS config into container
docker run --rm -v ~/.aws:/root/.aws \
  amazon/aws-cli:latest \
  --profile remote-access \
  sts get-caller-identity
```

## 🔄 Credential Process Protocol

The plugin implements the AWS CLI credential process protocol:

### Input
The plugin receives no command-line arguments when called by AWS CLI.

### Output
JSON response with credentials:

```json
{
  "Version": 1,
  "AccessKeyId": "ASIA...",
  "SecretAccessKey": "...",
  "SessionToken": "...",
  "Expiration": "2024-01-15T12:00:00Z"
}
```

### Error Handling
Errors are returned as JSON:

```json
{
  "error": "Failed to authenticate: invalid configuration"
}
```

## 📄 License

This plugin is part of the AWS Remote Access Patterns project and follows the same MIT license.

## 🤝 Contributing

Contributions are welcome! Please see the main project README for contribution guidelines.

## 📚 Related Documentation

- [AWS CLI Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)
- [Credential Process](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html)
- [Cross-Account Access](../../docs/cross-account.md)
- [Security Best Practices](../../docs/security.md)

---

This AWS CLI plugin provides a seamless way to integrate the AWS Remote Access Patterns authentication system directly into your AWS CLI workflows, enabling secure and convenient access to AWS resources across multiple authentication methods.