#!/bin/bash

# AWS Remote Access Patterns - Setup Script
# This script sets up the development environment and dependencies

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check Go version
check_go_version() {
    local required_version="1.21"
    local current_version
    
    if ! command_exists go; then
        return 1
    fi
    
    current_version=$(go version | cut -d' ' -f3 | sed 's/go//')
    
    # Simple version comparison (works for semantic versions)
    if [[ "$(printf '%s\n' "$required_version" "$current_version" | sort -V | head -n1)" = "$required_version" ]]; then
        return 0
    else
        return 1
    fi
}

# Function to install Go (if needed)
install_go() {
    print_status "Go not found or version too old. Installing Go 1.21..."
    
    case "$(uname -s)" in
        Darwin)
            if command_exists brew; then
                brew install go
            else
                print_error "Homebrew not found. Please install Go manually from https://golang.org/dl/"
                exit 1
            fi
            ;;
        Linux)
            # Download and install Go
            local go_version="1.21.6"
            local go_archive="go${go_version}.linux-amd64.tar.gz"
            local go_url="https://golang.org/dl/${go_archive}"
            
            print_status "Downloading Go ${go_version}..."
            wget -q "$go_url" -O "/tmp/${go_archive}"
            
            print_status "Installing Go..."
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf "/tmp/${go_archive}"
            
            # Add to PATH if not already there
            if ! echo "$PATH" | grep -q "/usr/local/go/bin"; then
                echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
                echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc 2>/dev/null || true
                export PATH=$PATH:/usr/local/go/bin
            fi
            
            rm "/tmp/${go_archive}"
            ;;
        *)
            print_error "Unsupported OS. Please install Go manually from https://golang.org/dl/"
            exit 1
            ;;
    esac
}

# Function to setup AWS CLI (if needed)
setup_aws_cli() {
    if command_exists aws; then
        print_success "AWS CLI already installed"
        return
    fi
    
    print_status "Installing AWS CLI v2..."
    
    case "$(uname -s)" in
        Darwin)
            if command_exists brew; then
                brew install awscli
            else
                # Download and install manually
                curl "https://awscli.amazonaws.com/AWSCLIV2.pkg" -o "AWSCLIV2.pkg"
                sudo installer -pkg AWSCLIV2.pkg -target /
                rm AWSCLIV2.pkg
            fi
            ;;
        Linux)
            curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
            unzip -q awscliv2.zip
            sudo ./aws/install
            rm -rf aws awscliv2.zip
            ;;
        *)
            print_warning "Please install AWS CLI v2 manually from https://aws.amazon.com/cli/"
            ;;
    esac
    
    if command_exists aws; then
        print_success "AWS CLI installed successfully"
    else
        print_error "Failed to install AWS CLI"
    fi
}

# Function to setup development tools
setup_dev_tools() {
    print_status "Setting up development tools..."
    
    # Install golangci-lint for linting
    if ! command_exists golangci-lint; then
        print_status "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
    fi
    
    # Install air for live reloading (optional)
    if ! command_exists air; then
        print_status "Installing air for live reloading..."
        go install github.com/cosmtrek/air@latest
    fi
    
    # Install staticcheck
    if ! command_exists staticcheck; then
        print_status "Installing staticcheck..."
        go install honnef.co/go/tools/cmd/staticcheck@latest
    fi
}

# Function to initialize project
init_project() {
    print_status "Initializing project dependencies..."
    
    # Download dependencies
    go mod download
    
    # Verify dependencies
    go mod verify
    
    # Build the project to ensure everything works
    print_status "Building project..."
    go build ./...
    
    # Run tests
    print_status "Running tests..."
    go test ./... -v
}

# Function to create development configuration
create_dev_config() {
    local config_dir="config"
    
    if [ ! -d "$config_dir" ]; then
        print_status "Creating configuration directory..."
        mkdir -p "$config_dir"
    fi
    
    # Create development configuration
    if [ ! -f "$config_dir/development.yaml" ]; then
        print_status "Creating development configuration..."
        cat > "$config_dir/development.yaml" << EOF
# Development configuration
database:
  type: memory
  
aws:
  region: us-east-1
  
logging:
  level: debug
  format: text
  
security:
  cors_origins: 
    - "http://localhost:3000"
    - "http://localhost:8080"
    
features:
  enable_web_ui: true
  enable_metrics: true
  enable_rate_limiting: false
EOF
    fi
    
    # Create example environment file
    if [ ! -f ".env.example" ]; then
        print_status "Creating example environment file..."
        cat > ".env.example" << EOF
# AWS Configuration
AWS_REGION=us-east-1
AWS_PROFILE=default

# Service Configuration  
SERVICE_NAME=YourService
SERVICE_ACCOUNT_ID=123456789012
TEMPLATE_S3_BUCKET=your-templates-bucket

# Database (for production)
# DYNAMODB_TABLE=aws-remote-access-patterns-prod
# KMS_KEY_ID=alias/aws-remote-access-patterns

# Optional: For web UI customization
# COMPANY_NAME=Your Company
# PRIMARY_COLOR=#2196F3
# SUPPORT_EMAIL=support@yourcompany.com
EOF
    fi
}

# Function to setup git hooks
setup_git_hooks() {
    if [ ! -d ".git" ]; then
        print_warning "Not a git repository, skipping git hooks setup"
        return
    fi
    
    print_status "Setting up git hooks..."
    
    # Create pre-commit hook
    cat > ".git/hooks/pre-commit" << 'EOF'
#!/bin/bash
# Pre-commit hook for AWS Remote Access Patterns

set -euo pipefail

echo "Running pre-commit checks..."

# Format code
echo "Formatting Go code..."
go fmt ./...

# Run linter
if command -v golangci-lint >/dev/null 2>&1; then
    echo "Running linter..."
    golangci-lint run
else
    echo "golangci-lint not found, skipping linting"
fi

# Run tests
echo "Running tests..."
go test ./... -race -short

# Check for TODO/FIXME comments in staged files
if git diff --cached --name-only | xargs grep -l "TODO\|FIXME" >/dev/null 2>&1; then
    echo "Warning: TODO/FIXME comments found in staged files"
fi

echo "Pre-commit checks passed!"
EOF
    
    chmod +x ".git/hooks/pre-commit"
}

# Function to create sample configurations
create_samples() {
    local examples_dir="examples"
    
    # Create .gitignore if it doesn't exist
    if [ ! -f ".gitignore" ]; then
        print_status "Creating .gitignore..."
        cat > ".gitignore" << EOF
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
main
simple-cli
simple-saas

# Test binary, built with \`go test -c\`
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Environment variables
.env
.env.local

# Temporary files
tmp/
temp/

# AWS
.aws/credentials
*.pem

# Build artifacts
dist/
build/
EOF
    fi
    
    # Create Makefile for common tasks
    if [ ! -f "Makefile" ]; then
        print_status "Creating Makefile..."
        cat > "Makefile" << 'EOF'
.PHONY: build test lint clean install-tools setup dev

# Build all binaries
build:
	go build -o bin/simple-cli ./examples/simple-cli
	go build -o bin/simple-saas ./examples/simple-saas

# Run tests
test:
	go test ./... -race -cover

# Run tests with verbose output
test-verbose:
	go test ./... -race -cover -v

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

# Setup development environment
setup: install-tools
	go mod download
	go mod verify

# Run development server with live reload
dev:
	air -c .air.toml

# Format code
fmt:
	go fmt ./...

# Update dependencies
deps:
	go mod tidy
	go mod download

# Generate documentation
docs:
	go doc -all ./pkg/crossaccount > docs/crossaccount-godoc.md
	go doc -all ./pkg/awsauth > docs/awsauth-godoc.md

# Run security checks
security:
	gosec ./...

# Benchmark tests
benchmark:
	go test -bench=. -benchmem ./...
EOF
    fi
}

# Main setup function
main() {
    print_status "Starting AWS Remote Access Patterns setup..."
    
    # Check and install Go
    if ! check_go_version; then
        install_go
    else
        print_success "Go $(go version | cut -d' ' -f3) is installed"
    fi
    
    # Setup AWS CLI
    setup_aws_cli
    
    # Setup development tools
    setup_dev_tools
    
    # Create development configuration
    create_dev_config
    
    # Initialize project
    init_project
    
    # Setup git hooks
    setup_git_hooks
    
    # Create sample files
    create_samples
    
    print_success "Setup completed successfully!"
    echo
    print_status "Next steps:"
    echo "1. Copy .env.example to .env and configure your settings"
    echo "2. Configure AWS credentials: aws configure"
    echo "3. Run examples: make build && ./bin/simple-cli --help"
    echo "4. Start development: make dev"
    echo
    print_status "Available commands:"
    echo "- make build      # Build all examples"
    echo "- make test       # Run tests"  
    echo "- make lint       # Run linter"
    echo "- make dev        # Start development server"
    echo
    print_success "Happy coding! 🚀"
}

# Run main function
main "$@"