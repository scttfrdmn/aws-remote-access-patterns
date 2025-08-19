# Contributing to AWS Remote Account Access Patterns

Thank you for your interest in contributing! This project demonstrates secure AWS remote account access patterns and welcomes contributions that improve security, usability, and documentation.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Security Considerations](#security-considerations)

## 🤝 Code of Conduct

This project follows the [Contributor Covenant](https://www.contributor-covenant.org/) code of conduct. By participating, you are expected to uphold this code.

## 🚀 Getting Started

### Types of Contributions We Welcome

- **Security improvements**: Better authentication flows, permission models, or vulnerability fixes
- **User experience enhancements**: Simpler setup processes, better error messages, clearer documentation
- **New examples**: Additional use cases, integration patterns, or deployment scenarios  
- **Documentation**: Tutorials, security guides, troubleshooting help, or API documentation
- **Bug fixes**: Issues with existing functionality
- **Testing**: Improved test coverage, integration tests, or security validation

### What We're Looking For

- Contributions that make AWS integration **simpler** for end users
- Security-first approaches that follow AWS best practices
- Clear, well-documented code with examples
- Comprehensive testing of new functionality

## 🛠️ Development Setup

### Prerequisites

- Go 1.21 or later
- AWS CLI v2 configured with appropriate permissions
- Docker (for running examples locally)
- Git

### Local Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/yourusername/aws-remote-access-patterns.git
   cd aws-remote-access-patterns
   ```

2. **Install dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Set up AWS credentials for testing**
   ```bash
   # For cross-account testing, you'll need:
   export AWS_ACCOUNT_ID="your-aws-account-id"
   export TEMPLATE_S3_BUCKET="your-test-bucket"
   
   # For external tool testing:
   aws configure  # or use AWS SSO
   ```

4. **Run tests**
   ```bash
   go test ./...
   ```

5. **Try the examples**
   ```bash
   # Test the CLI tool
   cd examples/simple-cli
   go run main.go --compare
   
   # Test the SaaS service  
   cd examples/simple-saas
   go run main.go
   ```

## 📝 How to Contribute

### Reporting Issues

Before creating an issue, please:

1. **Check existing issues** to avoid duplicates
2. **Use the issue templates** when available
3. **Provide detailed information**:
   - Go version and operating system
   - AWS SDK versions
   - Steps to reproduce the issue
   - Expected vs actual behavior
   - Error messages or logs

### Suggesting Enhancements

Enhancement suggestions should include:

- **Clear description** of the proposed feature
- **Use case** - why would this be useful?
- **User experience impact** - how does this improve simplicity?
- **Security considerations** - any security implications?
- **Implementation ideas** (optional)

## 🔧 Pull Request Process

### Before You Start

1. **Create an issue** describing what you plan to work on
2. **Wait for feedback** from maintainers before investing significant time
3. **Fork the repository** and create a feature branch

### Development Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-new-feature
   ```

2. **Make your changes**
   - Follow the [coding standards](#coding-standards)
   - Add tests for new functionality
   - Update documentation as needed

3. **Test thoroughly**
   ```bash
   # Run all tests
   go test ./...
   
   # Test examples
   cd examples/simple-cli && go run main.go --setup
   cd examples/simple-saas && go run main.go
   
   # Run security checks
   go vet ./...
   ```

4. **Update documentation**
   - Update relevant README sections
   - Add/update code comments
   - Update CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com/)

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add amazing new feature
   
   - Implements X to improve Y
   - Adds tests for Z scenario  
   - Updates documentation
   
   Closes #123"
   ```

6. **Push and create PR**
   ```bash
   git push origin feature/amazing-new-feature
   ```

### Pull Request Requirements

Your PR must:

- ✅ **Pass all tests** (`go test ./...`)
- ✅ **Follow coding standards** (see below)
- ✅ **Include tests** for new functionality
- ✅ **Update documentation** as needed
- ✅ **Have a clear description** of changes
- ✅ **Reference related issues** with "Closes #123"
- ✅ **Follow security best practices**

### PR Review Process

1. **Automated checks** run (tests, linting, security scans)
2. **Maintainer review** focuses on:
   - Code quality and security
   - User experience impact
   - Documentation completeness
   - Test coverage
3. **Feedback incorporation** and iteration
4. **Final approval** and merge

## 📏 Coding Standards

### Go Code Style

- **Use `gofmt`** and `goimports` for formatting
- **Follow standard Go conventions** from [Effective Go](https://golang.org/doc/effective_go.html)
- **Use meaningful variable names** - clarity over brevity
- **Add package and function comments** for public APIs

### Code Organization

```go
// Good: Clear package documentation
// Package crossaccount provides secure cross-account AWS integration
// for SaaS services that need access to customer AWS accounts.
package crossaccount

// Good: Clear function documentation with examples
// GenerateSetupLink creates a one-click setup link for customer AWS integration.
// 
// Example:
//   setupResp, err := client.GenerateSetupLink("acme-corp", "Acme Corporation")
//   if err != nil {
//       return err
//   }
//   // Send setupResp.LaunchURL to customer
func (c *Client) GenerateSetupLink(customerID, customerName string) (*SetupResponse, error) {
    // Validate inputs early
    if customerID == "" {
        return nil, fmt.Errorf("customer ID is required")
    }
    
    // Use clear, descriptive variable names
    externalID := c.generateSecureExternalID(customerID)
    
    // Return structured response with helpful information
    return &SetupResponse{
        LaunchURL:  launchURL,
        ExternalID: externalID,
        // ... other fields
    }, nil
}
```

### Error Handling

```go
// Good: Descriptive error messages that help users
if err := client.CompleteSetup(ctx, req); err != nil {
    return fmt.Errorf("failed to complete AWS setup for customer %s: %w", customerID, err)
}

// Good: User-friendly error responses
c.JSON(400, gin.H{
    "error": "Setup verification failed",
    "details": err.Error(),
    "common_solutions": []string{
        "Verify the Role ARN was copied correctly from CloudFormation outputs",
        "Ensure the CloudFormation stack creation completed successfully",
        "Check that the External ID matches the one provided during setup",
    },
})
```

### Configuration and Defaults

```go
// Good: Provide sensible defaults to minimize configuration
func New(cfg *Config) (*Client, error) {
    // Set helpful defaults
    if cfg.SessionDuration == 0 {
        cfg.SessionDuration = time.Hour // 1 hour is reasonable
    }
    if cfg.DefaultRegion == "" {
        cfg.DefaultRegion = "us-east-1" // Most common region
    }
    
    return &Client{config: cfg}, nil
}
```

### Testing

```go
// Good: Test both happy path and error conditions
func TestGenerateSetupLink(t *testing.T) {
    tests := []struct {
        name         string
        customerID   string  
        customerName string
        wantErr      bool
    }{
        {
            name:         "valid input",
            customerID:   "acme-corp", 
            customerName: "Acme Corporation",
            wantErr:      false,
        },
        {
            name:         "empty customer ID",
            customerID:   "",
            customerName: "Acme Corporation", 
            wantErr:      true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := &Client{config: testConfig()}
            resp, err := client.GenerateSetupLink(tt.customerID, tt.customerName)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, resp)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, resp)
                assert.NotEmpty(t, resp.LaunchURL)
                assert.NotEmpty(t, resp.ExternalID)
            }
        })
    }
}
```

## 🔒 Security Considerations

### Security Review Requirements

All contributions are reviewed for security implications:

- **Credential handling**: No long-lived secrets, proper credential lifecycle
- **Permission scope**: Least privilege principles, clear permission boundaries
- **Input validation**: Proper validation and sanitization of user inputs
- **Error information**: Avoid leaking sensitive data in error messages
- **Audit trails**: Ensure actions are properly logged for security monitoring

### Common Security Patterns

```go
// Good: Validate inputs early and clearly
func (c *Client) AssumeRole(ctx context.Context, customerID string) (aws.Config, error) {
    if customerID == "" {
        return aws.Config{}, fmt.Errorf("customer ID is required")
    }
    
    // Good: Use cryptographically secure random generation
    randomBytes := make([]byte, 16)
    if _, err := rand.Read(randomBytes); err != nil {
        return aws.Config{}, fmt.Errorf("failed to generate secure external ID: %w", err)
    }
    
    // Good: Don't log sensitive information
    log.Info("assuming role for customer", "customer_id", customerID)
    // Never log: external_id, role_arn, or temporary credentials
}
```

### Prohibited Patterns

```go
// ❌ DON'T: Store long-lived credentials
type BadCredentials struct {
    AccessKey string // Long-lived, insecure
    SecretKey string // Permanent secret
}

// ❌ DON'T: Use predictable external IDs  
func badExternalID(customerID string) string {
    return fmt.Sprintf("external-%s", customerID) // Predictable!
}

// ❌ DON'T: Over-privilege by default
var badPermissions = Permission{
    Effect:    "Allow",
    Actions:   []string{"*"}, // Too broad!
    Resources: []string{"*"}, // Everything!
}

// ❌ DON'T: Log sensitive information
log.Info("role assumed", "external_id", externalID) // Sensitive!
```

## 🏷️ Versioning and Releases

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible functionality additions  
- **PATCH**: Backward-compatible bug fixes

### Changelog Updates

Update `CHANGELOG.md` with your changes:

```markdown
## [Unreleased]

### Added
- New QuickConfig helper for common service types
- Additional permission templates for monitoring platforms

### Changed  
- Improved error messages in setup validation

### Fixed
- Fixed race condition in credential caching

### Security
- Enhanced external ID generation for better entropy
```

## 🙋‍♂️ Questions or Help?

- **Open an issue** for questions about the codebase
- **Check existing documentation** in the `docs/` directory
- **Review examples** in the `examples/` directory
- **Read the security guide** in `SECURITY.md`

## 🎉 Recognition

Contributors will be recognized in:
- `CONTRIBUTORS.md` file
- Release notes for significant contributions
- GitHub contributor graphs

Thank you for helping make AWS integration more secure and user-friendly! 🚀