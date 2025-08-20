# AWS CLI Integration Guide

This guide shows how to use AWS CLI with the temporary credentials from AWS Remote Access Patterns, providing secure, time-limited access without long-lived access keys.

## 🎯 Why Use Temporary Credentials with AWS CLI?

Traditional AWS CLI setup relies on permanent access keys stored in `~/.aws/credentials`. Our approach provides:

- **Enhanced Security**: Credentials expire automatically (default: 1 hour)
- **No Long-Lived Secrets**: No permanent keys in configuration files
- **Centralized Management**: Single authentication flow for all AWS tools
- **Audit Trail**: Complete logging of all credential usage
- **Easy Revocation**: Instant access removal when needed

## 🔧 Integration Methods

### Method 1: Credential Process (Recommended)

AWS CLI supports external credential providers through the `credential_process` configuration. This is the most secure and transparent method.

#### Setup

1. **Build the credential helper:**
```bash
cd examples/aws-cli-helper
go build -o aws-cli-helper main.go
sudo mv aws-cli-helper /usr/local/bin/  # Make it globally available
```

2. **Configure AWS CLI profile:**
```bash
# Create or edit ~/.aws/config
aws configure set credential_process "/usr/local/bin/aws-cli-helper --profile myservice" --profile myservice
aws configure set region us-east-1 --profile myservice
```

3. **Use with any AWS CLI command:**
```bash
aws s3 ls --profile myservice
aws ec2 describe-instances --profile myservice
```

#### How It Works

```bash
# When you run: aws s3 ls --profile myservice
# AWS CLI calls: /usr/local/bin/aws-cli-helper --profile myservice
# Helper returns JSON credentials:
{
  "Version": 1,
  "AccessKeyId": "ASIAIOSFODNN7EXAMPLE",
  "SecretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",  
  "SessionToken": "AgoGb3JpZ2luX...",
  "Expiration": "2024-01-15T12:00:00Z"
}
```

### Method 2: Environment Variables

For CI/CD environments or when you need temporary environment setup:

```bash
# Get credentials and export as environment variables
eval $(aws-cli-helper --export --profile myservice)

# Now all AWS CLI commands use temporary credentials
aws s3 ls
aws ec2 describe-instances
export | grep AWS  # See the credentials (don't do this in production!)
```

### Method 3: Assume Role Profile

Use AWS CLI's native assume role functionality with our external tool pattern:

```bash
# ~/.aws/config
[profile myservice-base]
region = us-east-1
# Use your existing SSO or IAM user credentials

[profile myservice]
role_arn = arn:aws:iam::CUSTOMER-ACCOUNT:role/MyService-CrossAccount
source_profile = myservice-base
external_id = MyService-customer-123-abc123def456
duration_seconds = 3600
```

## 🛠️ AWS CLI Helper Implementation

### Command-Line Interface

The helper tool provides a clean interface for credential management:

```bash
# Basic usage
aws-cli-helper --profile myservice                    # Get JSON credentials
aws-cli-helper --export --profile myservice          # Export as env vars
aws-cli-helper --check --profile myservice           # Check credential status

# Configuration
aws-cli-helper --setup --profile myservice           # Interactive setup
aws-cli-helper --list-profiles                       # Show available profiles
aws-cli-helper --refresh --profile myservice         # Force credential refresh

# Troubleshooting
aws-cli-helper --debug --profile myservice           # Debug output
aws-cli-helper --validate --profile myservice        # Validate configuration
```

### Configuration File

The helper uses a configuration file at `~/.aws-remote-access/config.yaml`:

```yaml
# ~/.aws-remote-access/config.yaml
profiles:
  myservice:
    tool_name: "myservice-cli"
    auth_method: "sso"  # sso, profile, iam_user
    sso_config:
      start_url: "https://myorg.awsapps.com/start"
      region: "us-east-1"
    required_actions:
      - "s3:GetObject"
      - "s3:PutObject"
      - "ec2:DescribeInstances"
    session_duration: 3600
    
  dataplatform:
    tool_name: "data-platform-cli"  
    auth_method: "profile"
    profile_name: "data-platform"
    required_actions:
      - "s3:*"
      - "glue:*"
    session_duration: 7200

cache:
  directory: "~/.aws-remote-access/cache"
  max_age: 3300  # 55 minutes (5 min buffer before expiry)

logging:
  level: "info"
  file: "~/.aws-remote-access/aws-cli-helper.log"
```

### Interactive Setup

The helper provides guided setup for first-time users:

```bash
$ aws-cli-helper --setup --profile myservice

🔧 AWS CLI Helper Setup
=======================

Profile Name: myservice
Service: MyService Data Platform

✅ Step 1: Authentication Method
Choose your authentication method:
1) AWS SSO (Recommended for organizations)
2) AWS Profile (Use existing ~/.aws/credentials)
3) IAM User (Not recommended)

Selection [1]: 1

✅ Step 2: SSO Configuration  
SSO Start URL: https://myorg.awsapps.com/start
SSO Region [us-east-1]: us-east-1

✅ Step 3: Required Permissions
The following permissions will be requested:
- s3:GetObject, s3:PutObject (Data access)
- ec2:DescribeInstances (Infrastructure monitoring)

Continue? [Y/n]: Y

✅ Step 4: Session Settings
Session Duration [3600 seconds]: 3600
Cache credentials? [Y/n]: Y

✅ Step 5: Test Authentication
Testing AWS SSO authentication...
🌐 Opening browser for AWS SSO login...
✅ Authentication successful!

✅ Setup Complete!
Configuration saved to ~/.aws-remote-access/config.yaml

Next steps:
1. Configure AWS CLI: aws configure set credential_process "/usr/local/bin/aws-cli-helper --profile myservice" --profile myservice
2. Test access: aws sts get-caller-identity --profile myservice
```

## 📋 Configuration Examples

### Example 1: Development Environment

For developers working on multiple projects:

```bash
# ~/.aws/config
[profile project-a-dev]
credential_process = aws-cli-helper --profile project-a-dev
region = us-east-1

[profile project-a-prod]
credential_process = aws-cli-helper --profile project-a-prod  
region = us-east-1

[profile project-b]
credential_process = aws-cli-helper --profile project-b
region = eu-west-1
```

```yaml
# ~/.aws-remote-access/config.yaml
profiles:
  project-a-dev:
    tool_name: "project-a-dev-cli"
    auth_method: "sso"
    sso_config:
      start_url: "https://company.awsapps.com/start"
      region: "us-east-1"
    session_duration: 3600
    
  project-a-prod:
    tool_name: "project-a-prod-cli"
    auth_method: "sso"
    sso_config:
      start_url: "https://company.awsapps.com/start" 
      region: "us-east-1"
    session_duration: 1800  # Shorter for production
    
  project-b:
    tool_name: "project-b-cli"
    auth_method: "profile"
    profile_name: "project-b-base"
    session_duration: 3600
```

### Example 2: CI/CD Environment  

For automated deployments:

```bash
# In CI/CD pipeline
export AWS_PROFILE=deployment
eval $(aws-cli-helper --export --profile deployment --ci-mode)

# All subsequent AWS commands use temporary credentials
aws s3 sync ./dist/ s3://deployment-bucket/
aws cloudformation deploy --template-file infra.yaml --stack-name app-stack
```

```yaml
# ~/.aws-remote-access/config.yaml (on CI/CD runner)
profiles:
  deployment:
    tool_name: "deployment-pipeline"
    auth_method: "iam_user"  # Use IAM user for CI/CD
    session_duration: 3600
    ci_mode: true
    required_actions:
      - "s3:*"
      - "cloudformation:*" 
      - "iam:PassRole"
```

### Example 3: Customer Data Access

For SaaS services accessing customer accounts:

```bash
# Configure customer-specific profiles
aws-cli-helper --setup-customer --customer-id acme-corp

# Use customer-specific credentials
aws s3 ls s3://acme-corp-data/ --profile customer-acme-corp
aws ec2 describe-instances --profile customer-acme-corp
```

```yaml
profiles:
  customer-acme-corp:
    tool_name: "myservice-customer-access"
    auth_method: "cross_account"
    cross_account:
      customer_id: "acme-corp"
      role_arn: "arn:aws:iam::123456789012:role/MyService-CrossAccount"
      external_id: "MyService-acme-corp-abc123def456"
    session_duration: 3600
```

## 🔐 Security Best Practices

### Credential Storage

```bash
# Credentials are cached securely
~/.aws-remote-access/
├── cache/
│   ├── myservice.json              # Encrypted credential cache
│   └── customer-acme-corp.json     # Customer-specific cache
├── config.yaml                     # Configuration (no secrets)
└── aws-cli-helper.log             # Audit log
```

### Encryption at Rest

Cached credentials are encrypted using system keyring or AES-256:

```go
// Credential caching with encryption
type EncryptedCache struct {
    keyring keyring.Keyring
    baseDir string
}

func (c *EncryptedCache) Store(profile string, creds *Credentials) error {
    // Encrypt credentials before writing to disk
    encrypted, err := c.encrypt(creds)
    if err != nil {
        return err
    }
    return os.WriteFile(filepath.Join(c.baseDir, profile+".enc"), encrypted, 0600)
}
```

### Audit Logging

All credential operations are logged:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "INFO", 
  "event": "credential_request",
  "profile": "myservice",
  "user": "john.doe",
  "source_ip": "192.168.1.100",
  "success": true,
  "duration_ms": 245
}
```

### Permission Validation

Before returning credentials, validate required permissions:

```go
func (h *Helper) validatePermissions(ctx context.Context, creds aws.Config, profile string) error {
    client := iam.NewFromConfig(creds)
    
    for _, action := range profile.RequiredActions {
        // Test that the action is allowed
        if err := h.testPermission(ctx, client, action); err != nil {
            return fmt.Errorf("missing permission %s: %w", action, err)
        }
    }
    return nil
}
```

## 🔄 Integration Patterns

### Pattern 1: Multi-Account Development

Developers working across multiple AWS accounts:

```bash
# Switch between environments seamlessly
aws s3 ls --profile dev-account
aws s3 ls --profile staging-account  
aws s3 ls --profile prod-account

# Each profile uses different authentication methods as appropriate
```

### Pattern 2: Customer Support Tools

Support engineers accessing customer environments:

```bash
# Generate time-limited access for support case
aws-cli-helper --generate-support-access --customer acme-corp --duration 1800 --case CS-12345

# Support engineer uses temporary profile
aws logs describe-log-groups --profile support-acme-corp-CS-12345
```

### Pattern 3: Batch Processing

Automated data processing across multiple customer accounts:

```bash
# Process all customers in parallel
for customer in $(aws-cli-helper --list-customers); do
  echo "Processing $customer..."
  aws s3 sync s3://$customer-data/ ./local-processing/ --profile customer-$customer &
done
wait  # Wait for all parallel processes
```

## 🚨 Troubleshooting

### Common Issues

#### 1. Credentials Not Found
```bash
Error: NoCredentialProviders: no valid providers in chain
```

**Solution:**
```bash
# Check helper configuration
aws-cli-helper --debug --profile myservice

# Verify helper is callable
/usr/local/bin/aws-cli-helper --profile myservice

# Check AWS CLI configuration
aws configure list --profile myservice
```

#### 2. Expired Credentials
```bash
Error: ExpiredToken: The security token included in the request is expired
```

**Solution:**
```bash
# Force credential refresh
aws-cli-helper --refresh --profile myservice

# Check cache status
aws-cli-helper --check --profile myservice
```

#### 3. Permission Denied
```bash
Error: AccessDenied: User is not authorized to perform: s3:ListBucket
```

**Solution:**
```bash
# Validate required permissions
aws-cli-helper --validate --profile myservice

# Check what permissions are actually granted
aws sts get-caller-identity --profile myservice
aws iam simulate-principal-policy --policy-source-arn $(aws sts get-caller-identity --query Arn --output text --profile myservice) --action-names s3:ListBucket --resource-arns arn:aws:s3:::test-bucket --profile myservice
```

#### 4. SSO Authentication Fails
```bash
Error: Unable to locate credentials
```

**Solution:**
```bash
# Re-authenticate with SSO
aws sso login --profile myservice-base

# Or use the helper's SSO flow
aws-cli-helper --setup --profile myservice --force-sso-reauth
```

### Debug Mode

Enable detailed logging for troubleshooting:

```bash
# Enable debug logging
export AWS_CLI_HELPER_DEBUG=1
aws-cli-helper --debug --profile myservice

# Check logs
tail -f ~/.aws-remote-access/aws-cli-helper.log
```

### Health Check

Validate your entire setup:

```bash
# Comprehensive health check
aws-cli-helper --health-check --profile myservice

✅ Configuration valid
✅ Authentication working  
✅ Credentials cached
✅ Permissions verified
✅ AWS CLI integration working

# Test with actual AWS CLI
aws sts get-caller-identity --profile myservice
```

## 📊 Monitoring and Metrics

### Credential Usage Metrics

Track credential usage across your organization:

```bash
# Generate usage report
aws-cli-helper --usage-report --days 30

Profile: myservice
├─ Total requests: 1,247
├─ Success rate: 99.2%
├─ Average duration: 234ms
├─ Cache hit rate: 87.3%
└─ Users: alice, bob, charlie (12 total)

Profile: customer-acme-corp  
├─ Total requests: 89
├─ Success rate: 100%
├─ Average duration: 456ms
├─ Cache hit rate: 91.0%
└─ Users: support-team (3 total)
```

### CloudWatch Integration

Send metrics to CloudWatch for monitoring:

```go
// Send custom metrics
func (h *Helper) recordMetrics(profile string, success bool, duration time.Duration) {
    cloudWatch := cloudwatch.NewFromConfig(h.adminConfig)
    
    cloudWatch.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
        Namespace: aws.String("AWSRemoteAccessPatterns/CLIHelper"),
        MetricData: []types.MetricDatum{
            {
                MetricName: aws.String("CredentialRequest"),
                Value:      aws.Float64(1),
                Unit:       types.StandardUnitCount,
                Dimensions: []types.Dimension{
                    {Name: aws.String("Profile"), Value: aws.String(profile)},
                    {Name: aws.String("Success"), Value: aws.String(fmt.Sprintf("%t", success))},
                },
            },
        },
    })
}
```

## 🔧 Advanced Configuration

### Custom Credential Providers

Extend the helper with custom authentication methods:

```go
// Custom credential provider interface
type CredentialProvider interface {
    GetCredentials(ctx context.Context, profile *Profile) (*aws.Credentials, error)
    Refresh(ctx context.Context, profile *Profile) (*aws.Credentials, error)
    Type() string
}

// Example: HashiCorp Vault integration
type VaultProvider struct {
    client *vault.Client
}

func (v *VaultProvider) GetCredentials(ctx context.Context, profile *Profile) (*aws.Credentials, error) {
    secret, err := v.client.Logical().Read("aws/creds/" + profile.VaultRole)
    if err != nil {
        return nil, err
    }
    
    return &aws.Credentials{
        AccessKeyID:     secret.Data["access_key"].(string),
        SecretAccessKey: secret.Data["secret_key"].(string),
        SessionToken:    secret.Data["security_token"].(string),
    }, nil
}
```

### Plugin Architecture

Support for third-party extensions:

```yaml
# ~/.aws-remote-access/config.yaml
plugins:
  - name: "vault-provider"
    path: "/usr/local/bin/aws-cli-helper-vault"
    config:
      vault_addr: "https://vault.company.com"
      
  - name: "okta-sso"  
    path: "/usr/local/bin/aws-cli-helper-okta"
    config:
      okta_domain: "company.okta.com"
```

## 🎓 Migration Guide

### From Static Access Keys

Replace static credentials with temporary ones:

```bash
# Before: ~/.aws/credentials
[myservice]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# After: ~/.aws/config  
[profile myservice]
credential_process = aws-cli-helper --profile myservice
region = us-east-1
```

### From AWS SSO CLI

Enhance existing SSO setup:

```bash
# Before: aws sso login --profile myservice
# After: Automatic SSO with credential helper
aws s3 ls --profile myservice  # Automatically handles SSO if needed
```

### Gradual Migration

Migrate profiles one at a time:

```bash
# Week 1: Migrate development profiles
aws-cli-helper --migrate --profile dev-profile --from-credentials

# Week 2: Migrate staging profiles  
aws-cli-helper --migrate --profile staging-profile --from-sso

# Week 3: Migrate production profiles (with extra care)
aws-cli-helper --migrate --profile prod-profile --from-credentials --dry-run
aws-cli-helper --migrate --profile prod-profile --from-credentials
```

---

## 📚 Additional Resources

- 🔗 [Cross-Account Access Pattern](cross-account.md) - For SaaS services
- 🖥️ [External Tool Access Pattern](external-tool.md) - For CLI/desktop authentication
- 🛡️ [Security Analysis](security.md) - Security best practices
- 📖 [API Reference](api-reference.md) - Complete API documentation

## 💡 Next Steps

1. **Install the CLI helper**: Follow the setup instructions above
2. **Configure your first profile**: Use `aws-cli-helper --setup`
3. **Test with AWS CLI**: Run `aws sts get-caller-identity --profile myservice`
4. **Migrate existing profiles**: Use `aws-cli-helper --migrate`
5. **Set up monitoring**: Configure CloudWatch metrics and logging

This integration makes AWS CLI usage more secure and manageable while maintaining the familiar AWS CLI experience your teams already know.