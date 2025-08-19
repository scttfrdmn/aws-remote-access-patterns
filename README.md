# AWS Remote Account Access Patterns

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/scttfrdmn/aws-remote-access-patterns)](https://goreportcard.com/report/github.com/scttfrdmn/aws-remote-access-patterns)

A comprehensive demonstration of secure AWS remote account access patterns, showcasing industry best practices for both **cross-account access** (SaaS services) and **external tool access** (CLI tools, desktop applications).

## 🎯 Problem This Solves

Modern applications need secure access to customer AWS accounts, but traditional approaches have significant issues:

- **Long-lived access keys** - Security risk, hard to rotate, often leaked
- **Overly broad permissions** - Tools ask for admin access "to be safe"
- **Poor user experience** - "Here's some JSON policy, figure it out yourself"
- **No clear boundaries** - Setup vs ongoing permissions mixed together

## 🏗️ Our Approach

This project demonstrates **two proven patterns** used by successful companies like Coiled, Datadog, and others:

### Pattern 1: Cross-Account Access (SaaS Services)
- **Use case**: Your service runs in AWS and needs access to customer AWS accounts
- **Method**: IAM cross-account roles with external IDs
- **Security**: Temporary credentials, least-privilege permissions
- **UX**: One-click CloudFormation deployment with progressive disclosure

### Pattern 2: External Tool Access (CLI/Desktop Tools)  
- **Use case**: Your tool runs on workstations/laptops and needs AWS access
- **Method**: AWS SSO + IAM roles with intelligent fallbacks
- **Security**: Temporary SSO credentials with automatic refresh
- **UX**: Guided setup with multiple authentication options

## 🚀 Quick Start

### For Cross-Account Access (SaaS Services)

```go
package main

import (
    "context"
    "log"
    
    "github.com/scttfrdmn/aws-remote-access-patterns/pkg/crossaccount"
)

func main() {
    client, err := crossaccount.New(&crossaccount.Config{
        ServiceName:      "MyDataPlatform",
        ServiceAccountID: "123456789012", 
        TemplateS3Bucket: "my-platform-templates",
        OngoingPermissions: []crossaccount.Permission{
            {
                Sid:    "S3DataAccess",
                Effect: "Allow",
                Actions: []string{"s3:GetObject", "s3:PutObject"},
                Resources: []string{"arn:aws:s3:::customer-data-*/*"},
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Generate integration URL for customer
    resp, err := client.GenerateIntegrationURL(context.Background(), &crossaccount.IntegrationRequest{
        CustomerID:   "acme-corp",
        CustomerName: "Acme Corp",
        Region:       "us-west-2",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Customer clicks this URL to set up the integration
    fmt.Printf("Integration URL: %s\n", resp.LaunchURL)
    
    // Later, assume the customer's role to perform operations
    awsConfig, err := client.AssumeCustomerRole(context.Background(), "acme-corp")
    if err != nil {
        log.Fatal(err)
    }
    
    // Use AWS services with customer's permissions
    s3Client := s3.NewFromConfig(awsConfig)
    // ... perform operations
}
```

### For External Tool Access (CLI Tools)

```go
package main

import (
    "context"
    "log"
    
    "github.com/scttfrdmn/aws-remote-access-patterns/pkg/awsauth"
)

func main() {
    client, err := awsauth.New(&awsauth.Config{
        ToolName: "my-awesome-cli",
        RequiredActions: []string{
            "ec2:DescribeInstances",
            "s3:ListBuckets",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Get AWS credentials (handles all authentication complexity)
    awsConfig, err := client.GetAWSConfig(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Use AWS services normally
    ec2Client := ec2.NewFromConfig(awsConfig)
    result, err := ec2Client.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{})
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d reservations\n", len(result.Reservations))
}
```

## 📁 Project Structure

```
aws-remote-access-patterns/
├── README.md                  # This file
├── LICENSE                    # MIT License
├── go.mod                     # Go module definition
├── go.sum                     # Go module checksums
│
├── pkg/                       # Core library packages
│   ├── crossaccount/          # Cross-account access pattern (SaaS services)
│   │   ├── client.go          # Main client implementation
│   │   ├── config.go          # Configuration structures
│   │   ├── templates.go       # CloudFormation generation
│   │   ├── validation.go      # Role validation
│   │   └── storage.go         # Credential storage interfaces
│   └── awsauth/               # External tool access pattern (CLI/desktop)
│       ├── client.go          # Main authentication client
│       ├── config.go          # Configuration structures
│       ├── sso.go             # AWS SSO integration
│       ├── setup.go           # Interactive setup process
│       └── credentials.go     # Credential management
│
├── examples/                  # Complete working examples
│   ├── simple-cli/            # Basic CLI tool (current)
│   ├── simple-saas/           # Basic SaaS service (current)
│   ├── saas-service/          # Complete SaaS service example (planned)
│   ├── cli-tool/              # Advanced CLI tool example (planned)
│   ├── desktop-app/           # Desktop application example (planned)
│   └── lambda-function/       # AWS Lambda example (planned)
│
├── templates/                 # CloudFormation templates
│   ├── cross-account-role.yaml # Production cross-account role template
│   └── iam-user-policy.yaml   # IAM user policy template
│
├── docs/                      # Comprehensive documentation
│   ├── cross-account.md       # Cross-account pattern complete guide
│   ├── external-tool.md       # External tool pattern complete guide
│   ├── security.md            # Security analysis and best practices
│   ├── deployment.md          # Production deployment guide
│   └── api-reference.md       # Complete API documentation
│
└── scripts/                   # Automation and setup scripts
    ├── setup.sh               # Development environment setup
    └── deploy.sh              # Production deployment script
```

## 🔐 Security Features

### Built-in Security Best Practices

- **Least Privilege**: Generate minimal IAM policies based on actual requirements
- **External IDs**: Cryptographically secure external IDs for cross-account roles
- **Temporary Credentials**: No long-lived access keys stored or transmitted
- **Two-Phase Permissions**: Separate setup vs ongoing permissions
- **Credential Rotation**: Automatic handling of credential refresh and rotation
- **Audit Logging**: Comprehensive logging of all authentication events

### Security Validations

- Permission boundary validation
- Cross-account role testing
- Credential expiration handling
- Failed authentication alerting

## 📚 Documentation

### Core Concepts & Patterns
- 🔗 [Cross-Account Access Pattern](docs/cross-account.md) - Complete guide for SaaS services accessing customer accounts
- 🖥️ [External Tool Access Pattern](docs/external-tool.md) - Complete guide for CLI tools and desktop applications  
- 🛡️ [Security Analysis](docs/security.md) - Security comparison: cross-account roles vs access keys
- 🚀 [Production Deployment](docs/deployment.md) - Enterprise deployment with scaling, monitoring, and DR
- 📖 [API Reference](docs/api-reference.md) - Complete API documentation with examples

### Working Examples
- **Basic CLI**: [examples/simple-cli/](examples/simple-cli/) - Simple command-line tool (current)
- **Basic SaaS**: [examples/simple-saas/](examples/simple-saas/) - Simple web service (current)
- **Advanced SaaS**: [examples/saas-service/](examples/saas-service/) - Complete web service with UI (planned)
- **Advanced CLI**: [examples/cli-tool/](examples/cli-tool/) - Feature-rich CLI application (planned)
- **Desktop App**: [examples/desktop-app/](examples/desktop-app/) - GUI application with visual setup (planned)
- **Lambda Function**: [examples/lambda-function/](examples/lambda-function/) - Serverless cross-account access (planned)

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or later
- AWS CLI v2 configured with appropriate permissions
- Basic understanding of AWS IAM roles and policies

### Installation

1. Clone the repository:
```bash
git clone https://github.com/scttfrdmn/aws-remote-access-patterns.git
cd aws-remote-access-patterns
```

2. Install dependencies:
```bash
go mod download
```

3. Run the setup script:
```bash
./scripts/setup.sh
```

4. Try the examples:
```bash
# Run the CLI tool example
cd examples/simple-cli
go run main.go --setup

# Run the SaaS service example  
cd examples/simple-saas
go run main.go
```

## ✨ Features

### Cross-Account Pattern (SaaS Services)
- 🚀 One-click CloudFormation deployment for customers
- 🔒 Cryptographically secure external ID generation
- 📋 Two-phase permission strategy (setup vs ongoing)
- ⚡ Automatic credential refresh and caching
- 🎯 Least-privilege permission templates
- 🌍 Multi-region support
- 📊 Comprehensive audit logging

### External Tool Pattern (CLI/Desktop)
- 🔐 AWS SSO device flow integration
- 🔄 Multiple authentication fallbacks (SSO → Profile → IAM User)
- 🧙‍♂️ Interactive setup wizard with clear guidance
- 💾 Secure credential caching with expiration
- 🛠️ CI/CD environment optimization
- 🖥️ Desktop application support with web UI
- 📱 Cross-platform compatibility

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Add tests for new functionality
5. Run tests: `go test ./...`
6. Commit changes: `git commit -m 'Add amazing feature'`
7. Push to branch: `git push origin feature/amazing-feature`
8. Open a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

This project is inspired by the excellent work of:
- **Coiled** - for pioneering user-friendly AWS integration UX
- **Datadog** - for cross-account role best practices
- **AWS** - for providing the underlying security primitives

## ⭐ Show Your Support

If this project helps you build secure AWS integrations, please give it a star! It helps others discover these patterns.

## 📞 Support

- 📖 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/scttfrdmn/aws-remote-access-patterns/issues)
- 💬 [Discussions](https://github.com/scttfrdmn/aws-remote-access-patterns/discussions)
- 🔒 [Security Issues](SECURITY.md)