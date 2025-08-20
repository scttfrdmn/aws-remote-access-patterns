# Advanced CLI Tool Example

A comprehensive command-line application demonstrating advanced external tool authentication patterns using AWS Remote Access Patterns with rich interactive UI, configuration management, and production-ready features.

## 🎯 Features

- **Rich Interactive Setup**: Guided authentication configuration with auto-detection of existing AWS setups
- **Multiple Authentication Methods**: AWS SSO, profiles, and interactive authentication
- **Advanced UI Components**: Progress indicators, colored output, tables, and confirmation dialogs
- **Configuration Management**: Comprehensive config system with validation and persistence
- **Authentication Management**: Status checking, testing, refreshing, and troubleshooting
- **Shell Completion**: Bash, Zsh, Fish, and PowerShell completion support
- **Production Ready**: Structured logging, error handling, and graceful shutdown

## 🚀 Quick Start

### Installation

```bash
# Build the CLI tool
cd examples/cli-tool
go build -o datatool main.go

# Optional: Install globally
sudo mv datatool /usr/local/bin/

# Or add to PATH
export PATH="$PATH:$(pwd)"
```

### First-Time Setup

```bash
# Interactive setup wizard
datatool setup

# The wizard will:
# 1. Detect existing AWS configurations
# 2. Guide you through authentication method selection
# 3. Test the authentication
# 4. Save the configuration
```

### Basic Usage

```bash
# Check authentication status
datatool auth status

# Test authentication
datatool auth test

# Show configuration
datatool config show

# List S3 buckets (example AWS operation)
datatool s3 list

# Get help for any command
datatool --help
datatool setup --help
```

## 📋 Commands

### Setup and Authentication

```bash
# Interactive setup wizard
datatool setup                    # Full interactive setup
datatool setup --method sso       # Use AWS SSO
datatool setup --force           # Force reconfiguration

# Authentication management
datatool auth status              # Show auth status
datatool auth status --detailed   # Detailed status
datatool auth test                # Test authentication
datatool auth refresh             # Refresh credentials
datatool auth clear               # Clear configuration
```

### Configuration Management

```bash
# View configuration
datatool config show             # Show all config (YAML)
datatool config show --format json # JSON format
datatool config show auth        # Show only auth section

# Modify configuration  
datatool config set cli.output_format table
datatool config set auth.session_duration 7200
datatool config set data.max_concurrency 20

# Validate and reset
datatool config validate         # Validate configuration
datatool config reset           # Reset to defaults
datatool config reset auth      # Reset auth section only
```

### AWS Operations (Examples)

```bash
# S3 operations
datatool s3 list                 # List buckets
datatool s3 list bucket-name     # List objects in bucket
datatool s3 sync source dest     # Sync data

# EC2 operations  
datatool ec2 instances           # List instances
datatool ec2 instances --running # Filter by state

# Data operations
datatool data sync --env prod    # Sync to production environment
datatool data backup --bucket my-backup
```

### Utility Commands

```bash
# Shell completion
datatool completion bash > /usr/local/etc/bash_completion.d/datatool
datatool completion zsh > "${fpath[1]}/_datatool"

# Version information
datatool version

# Global options
datatool --debug <command>       # Enable debug logging
datatool --quiet <command>       # Suppress output
datatool --no-color <command>    # Disable colors
```

## ⚙️ Configuration

### Configuration File

DataTool stores configuration in `~/.datatool/config.yaml`:

```yaml
# Authentication settings
auth:
  method: "sso"                   # sso, profile, interactive
  region: "us-east-1"
  session_duration: 3600
  cache_enabled: true
  sso:
    start_url: "https://company.awsapps.com/start"
    region: "us-east-1"

# CLI behavior settings
cli:
  output_format: "table"          # table, json, yaml, csv
  table_style: "default"
  page_size: 50
  confirm_actions: true
  show_progress: true
  auto_pagination: true

# Data processing settings
data:
  default_bucket: "my-data-bucket"
  temporary_directory: "/tmp/datatool"
  max_concurrency: 10
  chunk_size: 10485760            # 10MB
  environments:
    dev: "my-dev-bucket"
    staging: "my-staging-bucket"
    prod: "my-prod-bucket"
```

### Environment Variables

Override configuration with environment variables:

```bash
export DATATOOL_DEBUG=true
export DATATOOL_AWS_REGION=us-west-2
export DATATOOL_CLI_OUTPUT_FORMAT=json
export DATATOOL_AUTH_METHOD=sso
```

## 🔧 Authentication Methods

### 1. AWS SSO (Recommended)

```bash
# Setup with SSO
datatool setup --method sso

# The tool will prompt for:
# - SSO Start URL
# - SSO Region
# - Browser authentication
```

Interactive browser-based authentication with automatic token refresh.

### 2. AWS Profile

```bash
# Use existing AWS profile
datatool setup --method profile

# The tool will:
# - Detect available profiles
# - Let you select one
# - Use existing credentials
```

Leverages existing `~/.aws/credentials` and `~/.aws/config` files.

### 3. Interactive Authentication

```bash
# Interactive authentication
datatool setup --method interactive

# Provides guided setup for first-time users
```

## 🎨 User Interface Features

### Rich Interactive Components

- **Setup Wizard**: Step-by-step configuration with progress indication
- **Colored Output**: Success (green), error (red), warning (yellow), info (blue)
- **Tables**: Formatted data display with proper alignment
- **Progress Indicators**: Progress bars and spinners for long operations
- **Confirmations**: Safe confirmation dialogs for destructive operations

### Output Formats

```bash
# Table format (default)
datatool s3 list
┌─────────────────────────┬─────────────┬──────────────┐
│ Bucket Name             │ Region      │ Created      │
├─────────────────────────┼─────────────┼──────────────┤
│ my-app-data             │ us-east-1   │ 2024-01-15   │
│ my-app-logs             │ us-east-1   │ 2024-01-10   │
└─────────────────────────┴─────────────┴──────────────┘

# JSON format
datatool s3 list --format json
{
  "buckets": [
    {
      "name": "my-app-data",
      "region": "us-east-1", 
      "created": "2024-01-15T10:30:00Z"
    }
  ]
}

# YAML format
datatool s3 list --format yaml
buckets:
  - name: my-app-data
    region: us-east-1
    created: 2024-01-15T10:30:00Z
```

## 🔍 Troubleshooting

### Common Issues

#### 1. Authentication Failures

```bash
# Check authentication status
datatool auth status

# Test authentication
datatool auth test

# Refresh credentials
datatool auth refresh

# Reconfigure if needed
datatool setup --force
```

#### 2. Configuration Issues

```bash
# Validate configuration
datatool config validate

# Show current configuration
datatool config show

# Reset problematic sections
datatool config reset auth
```

#### 3. Permission Errors

```bash
# Check current AWS identity
datatool auth status --detailed

# Verify required permissions are granted
datatool auth test
```

### Debug Mode

```bash
# Enable debug logging
datatool --debug <command>

# Check log file
cat ~/.datatool/datatool.log

# Show version and build info
datatool version
```

### Configuration Recovery

```bash
# Reset all configuration
datatool config reset --force

# Reset specific sections
datatool config reset auth --force
datatool config reset cli --force

# Re-run setup
datatool setup
```

## 🏗️ Architecture

### Project Structure

```
examples/cli-tool/
├── main.go                      # Main entry point
├── cmd/                         # Command implementations
│   ├── root.go                  # Root command and global flags
│   ├── setup.go                 # Setup wizard command
│   ├── auth.go                  # Authentication commands
│   ├── config.go                # Configuration commands
│   ├── s3.go                    # S3 operation commands
│   └── ec2.go                   # EC2 operation commands
├── internal/
│   ├── auth/                    # Authentication management
│   │   ├── manager.go           # Auth manager
│   │   └── detector.go          # Config detection
│   ├── config/                  # Configuration management
│   │   └── config.go            # Config structure and I/O
│   ├── ui/                      # User interface components
│   │   └── handler.go           # Rich UI components
│   └── commands/                # Business logic for commands
└── pkg/
    └── client/                  # AWS service clients
```

### Key Components

1. **Command Layer** (`cmd/`): Cobra-based CLI commands with flag handling
2. **Business Logic** (`internal/`): Core functionality and state management
3. **UI Layer** (`internal/ui/`): Rich terminal interface components
4. **Configuration** (`internal/config/`): YAML-based configuration with validation
5. **Authentication** (`internal/auth/`): AWS authentication management

### Authentication Flow

```
┌─────────────────┐
│   User Command  │
└─────────┬───────┘
          │
┌─────────▼───────┐
│  Auth Manager   │ ──── Check if configured
└─────────┬───────┘
          │
┌─────────▼───────┐
│  awsauth.Client │ ──── Get AWS credentials
└─────────┬───────┘
          │
┌─────────▼───────┐
│  AWS SDK Call   │ ──── Execute AWS operation
└─────────────────┘
```

## 🧪 Development

### Building

```bash
# Build for development
go build -o datatool main.go

# Build with version information
go build -ldflags="-X main.Version=v1.0.0 -X main.GitCommit=$(git rev-parse HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o datatool main.go

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o datatool-linux main.go
GOOS=darwin GOARCH=amd64 go build -o datatool-darwin main.go
GOOS=windows GOARCH=amd64 go build -o datatool-windows.exe main.go
```

### Testing

```bash
# Run tests
go test ./...

# Test with coverage
go test -cover ./...

# Test specific package
go test ./internal/auth/

# Integration testing
datatool setup --force
datatool auth test
datatool s3 list
```

### Adding New Commands

1. Create command file in `cmd/`
2. Implement command logic
3. Add to root command in `cmd/root.go`
4. Add business logic in `internal/`
5. Update documentation

Example new command:

```go
// cmd/example.go
func newExampleCommand(ctx context.Context, cfg *config.Config) *cobra.Command {
    return &cobra.Command{
        Use:   "example",
        Short: "Example command",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runExample(ctx, cfg)
        },
    }
}
```

## 📚 Examples

### Complete Workflow Example

```bash
# 1. Initial setup
datatool setup

# 2. Check status
datatool auth status

# 3. Configure preferences
datatool config set cli.output_format json
datatool config set data.default_bucket my-data-bucket

# 4. Perform operations
datatool s3 list
datatool ec2 instances --running

# 5. Sync data between environments
datatool data sync --from dev --to staging

# 6. Troubleshoot if needed
datatool auth test
datatool auth refresh
```

### Batch Operations Example

```bash
# List all S3 buckets and save to file
datatool s3 list --format json > buckets.json

# Get all running EC2 instances
datatool ec2 instances --running --format csv > running-instances.csv

# Sync multiple data sources
for env in dev staging prod; do
  datatool data sync --env $env --backup
done
```

### Automation Example

```bash
#!/bin/bash
# automation-script.sh

set -e

echo "🔍 Checking authentication..."
datatool auth test

echo "📊 Getting resource inventory..."
datatool s3 list --format json > inventory/s3-buckets.json
datatool ec2 instances --format json > inventory/ec2-instances.json

echo "🔄 Syncing critical data..."
datatool data sync --env prod --verify

echo "✅ Automation complete!"
```

## 🔗 Integration

### Shell Integration

```bash
# Add to ~/.bashrc or ~/.zshrc
alias dt='datatool'
alias dts='datatool auth status'
alias dtl='datatool s3 list'

# Enable completion
source <(datatool completion bash)  # or zsh, fish, powershell
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
- name: Setup DataTool
  run: |
    curl -L https://github.com/example/datatool/releases/latest/download/datatool-linux -o datatool
    chmod +x datatool
    
- name: Configure Authentication
  run: |
    ./datatool setup --method profile --force
    
- name: Deploy Data
  run: |
    ./datatool data sync --env production --verify
```

---

This advanced CLI tool demonstrates production-ready external tool authentication with a rich user experience, comprehensive configuration management, and robust error handling. Use it as a foundation for building your own AWS-integrated command-line tools.