#!/bin/bash
# Cross-platform role assumption script for CI/CD pipelines
# Supports GitHub Actions, GitLab CI, Jenkins, and Azure DevOps

set -euo pipefail

# Script configuration
SCRIPT_NAME="assume-role.sh"
SCRIPT_VERSION="1.0.0"

# Color output for better readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# Help function
show_help() {
    cat << EOF
$SCRIPT_NAME v$SCRIPT_VERSION - Cross-platform AWS role assumption for CI/CD

Usage: $SCRIPT_NAME [OPTIONS] ROLE_ARN [EXTERNAL_ID]

Arguments:
  ROLE_ARN      The ARN of the role to assume (required)
  EXTERNAL_ID   The external ID for role assumption (optional)

Options:
  -s, --session-name NAME    Role session name (default: auto-generated)
  -d, --duration SECONDS     Session duration in seconds (default: 3600)
  -r, --region REGION        AWS region (default: us-east-1)
  -p, --profile PROFILE      AWS profile to use for assumption
  -f, --format FORMAT        Output format: env|file|both (default: both)
  -o, --output-file FILE     Credentials output file (default: ~/.aws/credentials)
  -v, --verify               Verify assumed role with GetCallerIdentity
  -q, --quiet                Suppress informational output
  -h, --help                 Show this help message
  --debug                    Enable debug output

Environment Variables:
  ROLE_ARN              Role ARN (overridden by argument)
  EXTERNAL_ID           External ID (overridden by argument)
  AWS_REGION            AWS region
  ROLE_SESSION_NAME     Session name
  SESSION_DURATION      Duration in seconds
  DEBUG                 Enable debug mode (true/false)

CI/CD Platform Detection:
  - GitHub Actions: Automatically detected via GITHUB_ACTIONS
  - GitLab CI: Automatically detected via GITLAB_CI
  - Jenkins: Automatically detected via JENKINS_URL
  - Azure DevOps: Automatically detected via TF_BUILD

Examples:
  # Basic role assumption
  $SCRIPT_NAME arn:aws:iam::123456789012:role/MyRole

  # With external ID
  $SCRIPT_NAME arn:aws:iam::123456789012:role/MyRole my-external-id

  # Custom session name and duration
  $SCRIPT_NAME --session-name "custom-session" --duration 7200 \\
    arn:aws:iam::123456789012:role/MyRole

  # Environment variables only
  $SCRIPT_NAME --format env arn:aws:iam::123456789012:role/MyRole

  # Quiet mode with verification
  $SCRIPT_NAME --quiet --verify arn:aws:iam::123456789012:role/MyRole
EOF
}

# Default values
SESSION_DURATION="${SESSION_DURATION:-3600}"
AWS_REGION="${AWS_REGION:-us-east-1}"
OUTPUT_FORMAT="both"
OUTPUT_FILE="$HOME/.aws/credentials"
VERIFY_ROLE="false"
QUIET="false"
DEBUG="${DEBUG:-false}"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--session-name)
            ROLE_SESSION_NAME="$2"
            shift 2
            ;;
        -d|--duration)
            SESSION_DURATION="$2"
            shift 2
            ;;
        -r|--region)
            AWS_REGION="$2"
            shift 2
            ;;
        -p|--profile)
            AWS_PROFILE="$2"
            shift 2
            ;;
        -f|--format)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        -o|--output-file)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        -v|--verify)
            VERIFY_ROLE="true"
            shift
            ;;
        -q|--quiet)
            QUIET="true"
            shift
            ;;
        --debug)
            DEBUG="true"
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
        *)
            if [[ -z "${ROLE_ARN:-}" ]]; then
                ROLE_ARN="$1"
            elif [[ -z "${EXTERNAL_ID:-}" ]]; then
                EXTERNAL_ID="$1"
            else
                log_error "Too many arguments: $1"
                show_help
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate required parameters
if [[ -z "${ROLE_ARN:-}" ]]; then
    log_error "ROLE_ARN is required"
    show_help
    exit 1
fi

# Detect CI/CD platform and set defaults
detect_platform() {
    local platform="unknown"
    
    if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
        platform="github-actions"
        ROLE_SESSION_NAME="${ROLE_SESSION_NAME:-GitHubActions-${GITHUB_REPOSITORY:-repo}-${GITHUB_RUN_ID:-0}}"
    elif [[ -n "${GITLAB_CI:-}" ]]; then
        platform="gitlab-ci"
        ROLE_SESSION_NAME="${ROLE_SESSION_NAME:-GitLabCI-${CI_PROJECT_NAME:-project}-${CI_PIPELINE_ID:-0}}"
    elif [[ -n "${JENKINS_URL:-}" ]]; then
        platform="jenkins"
        ROLE_SESSION_NAME="${ROLE_SESSION_NAME:-Jenkins-${JOB_NAME:-job}-${BUILD_NUMBER:-0}}"
    elif [[ -n "${TF_BUILD:-}" ]]; then
        platform="azure-devops"
        ROLE_SESSION_NAME="${ROLE_SESSION_NAME:-AzureDevOps-${BUILD_DEFINITIONNAME:-build}-${BUILD_BUILDID:-0}}"
    else
        platform="local"
        ROLE_SESSION_NAME="${ROLE_SESSION_NAME:-local-session-$(date +%s)}"
    fi
    
    echo "$platform"
}

# Platform-specific environment variable export
export_credentials_for_platform() {
    local platform="$1"
    local access_key="$2"
    local secret_key="$3"
    local session_token="$4"
    
    case "$platform" in
        "github-actions")
            if [[ -n "${GITHUB_ENV:-}" ]] && [[ "$OUTPUT_FORMAT" == "env" || "$OUTPUT_FORMAT" == "both" ]]; then
                echo "AWS_ACCESS_KEY_ID=$access_key" >> "$GITHUB_ENV"
                echo "AWS_SECRET_ACCESS_KEY=$secret_key" >> "$GITHUB_ENV"
                echo "AWS_SESSION_TOKEN=$session_token" >> "$GITHUB_ENV"
                log_debug "Exported credentials to GITHUB_ENV"
            fi
            ;;
        "gitlab-ci")
            # GitLab CI doesn't have persistent environment file, use exports
            export AWS_ACCESS_KEY_ID="$access_key"
            export AWS_SECRET_ACCESS_KEY="$secret_key"
            export AWS_SESSION_TOKEN="$session_token"
            log_debug "Exported credentials as environment variables"
            ;;
        "jenkins")
            # Jenkins: Create temporary credentials file that can be sourced
            local jenkins_creds_file="/tmp/jenkins-aws-credentials-$$"
            cat > "$jenkins_creds_file" << EOF
export AWS_ACCESS_KEY_ID='$access_key'
export AWS_SECRET_ACCESS_KEY='$secret_key'
export AWS_SESSION_TOKEN='$session_token'
EOF
            echo "JENKINS_AWS_CREDS_FILE=$jenkins_creds_file"
            log_debug "Created Jenkins credentials file: $jenkins_creds_file"
            ;;
        "azure-devops")
            # Azure DevOps: Use logging commands to set variables
            echo "##vso[task.setvariable variable=AWS_ACCESS_KEY_ID;issecret=true]$access_key"
            echo "##vso[task.setvariable variable=AWS_SECRET_ACCESS_KEY;issecret=true]$secret_key"
            echo "##vso[task.setvariable variable=AWS_SESSION_TOKEN;issecret=true]$session_token"
            log_debug "Set credentials as Azure DevOps variables"
            ;;
        *)
            # Local/unknown platform: Export to current shell
            export AWS_ACCESS_KEY_ID="$access_key"
            export AWS_SECRET_ACCESS_KEY="$secret_key"
            export AWS_SESSION_TOKEN="$session_token"
            log_debug "Exported credentials to current shell"
            ;;
    esac
}

# Main execution
main() {
    local platform
    platform=$(detect_platform)
    
    if [[ "$QUIET" != "true" ]]; then
        log_info "Starting role assumption"
        log_info "Platform: $platform"
        log_debug "Role ARN: $ROLE_ARN"
        log_debug "External ID: ${EXTERNAL_ID:0:10}${EXTERNAL_ID:+...}" # Show only first 10 chars
        log_debug "Session Name: $ROLE_SESSION_NAME"
        log_debug "Duration: ${SESSION_DURATION}s"
        log_debug "Region: $AWS_REGION"
    fi
    
    # Build assume role command
    local assume_role_cmd=(
        "aws" "sts" "assume-role"
        "--role-arn" "$ROLE_ARN"
        "--role-session-name" "$ROLE_SESSION_NAME"
        "--duration-seconds" "$SESSION_DURATION"
        "--region" "$AWS_REGION"
        "--output" "json"
    )
    
    # Add external ID if provided
    if [[ -n "${EXTERNAL_ID:-}" ]]; then
        assume_role_cmd+=("--external-id" "$EXTERNAL_ID")
    fi
    
    # Add profile if specified
    if [[ -n "${AWS_PROFILE:-}" ]]; then
        assume_role_cmd+=("--profile" "$AWS_PROFILE")
    fi
    
    log_debug "Executing: ${assume_role_cmd[*]}"
    
    # Assume the role
    local temp_creds
    if ! temp_creds=$("${assume_role_cmd[@]}" 2>/dev/null); then
        log_error "Failed to assume role $ROLE_ARN"
        log_error "Please check:"
        log_error "  1. Role ARN is correct and exists"
        log_error "  2. External ID matches (if required)"
        log_error "  3. Current AWS credentials have sts:AssumeRole permission"
        log_error "  4. Trust policy allows assumption from current identity"
        exit 1
    fi
    
    # Parse credentials from JSON response
    local access_key secret_key session_token expiration
    access_key=$(echo "$temp_creds" | jq -r '.Credentials.AccessKeyId')
    secret_key=$(echo "$temp_creds" | jq -r '.Credentials.SecretAccessKey')
    session_token=$(echo "$temp_creds" | jq -r '.Credentials.SessionToken')
    expiration=$(echo "$temp_creds" | jq -r '.Credentials.Expiration')
    
    if [[ "$access_key" == "null" || -z "$access_key" ]]; then
        log_error "Failed to parse credentials from assume role response"
        exit 1
    fi
    
    # Export credentials based on output format and platform
    if [[ "$OUTPUT_FORMAT" == "env" || "$OUTPUT_FORMAT" == "both" ]]; then
        export_credentials_for_platform "$platform" "$access_key" "$secret_key" "$session_token"
    fi
    
    # Write credentials file
    if [[ "$OUTPUT_FORMAT" == "file" || "$OUTPUT_FORMAT" == "both" ]]; then
        mkdir -p "$(dirname "$OUTPUT_FILE")"
        cat > "$OUTPUT_FILE" << EOF
[default]
aws_access_key_id = $access_key
aws_secret_access_key = $secret_key  
aws_session_token = $session_token
region = $AWS_REGION
EOF
        chmod 600 "$OUTPUT_FILE"
        log_debug "Wrote credentials to $OUTPUT_FILE"
    fi
    
    if [[ "$QUIET" != "true" ]]; then
        log_info "Role assumed successfully"
        log_info "Session expires at: $expiration"
    fi
    
    # Verify the assumed role if requested
    if [[ "$VERIFY_ROLE" == "true" ]]; then
        if [[ "$QUIET" != "true" ]]; then
            log_info "Verifying assumed role identity..."
        fi
        
        local identity
        if identity=$(AWS_ACCESS_KEY_ID="$access_key" AWS_SECRET_ACCESS_KEY="$secret_key" AWS_SESSION_TOKEN="$session_token" aws sts get-caller-identity --region "$AWS_REGION" --output json 2>/dev/null); then
            local user_id account arn
            user_id=$(echo "$identity" | jq -r '.UserId')
            account=$(echo "$identity" | jq -r '.Account')
            arn=$(echo "$identity" | jq -r '.Arn')
            
            if [[ "$QUIET" != "true" ]]; then
                log_info "Identity verified successfully:"
                log_info "  User ID: $user_id"
                log_info "  Account: $account"
                log_info "  ARN: $arn"
            fi
        else
            log_warn "Failed to verify assumed role identity"
            log_warn "Credentials may still be valid - check AWS service permissions"
        fi
    fi
    
    if [[ "$QUIET" != "true" ]]; then
        log_info "Role assumption completed successfully!"
        
        # Platform-specific usage instructions
        case "$platform" in
            "jenkins")
                if [[ -n "${JENKINS_AWS_CREDS_FILE:-}" ]]; then
                    log_info "In Jenkins, source the credentials file: source \$JENKINS_AWS_CREDS_FILE"
                fi
                ;;
            *)
                log_info "AWS credentials are now configured for subsequent commands"
                ;;
        esac
    fi
}

# Cleanup function
cleanup() {
    if [[ -n "${JENKINS_AWS_CREDS_FILE:-}" ]] && [[ -f "$JENKINS_AWS_CREDS_FILE" ]]; then
        rm -f "$JENKINS_AWS_CREDS_FILE"
        log_debug "Cleaned up temporary credentials file"
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Run main function
main "$@"