// Package main provides a Lambda function demonstrating cross-account role assumption
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/crossaccount"
)

// Response represents the Lambda function response
type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// RequestBody represents the incoming request body
type RequestBody struct {
	Action      string            `json:"action"`
	TargetRole  string            `json:"target_role,omitempty"`
	ExternalID  string            `json:"external_id,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	BucketName  string            `json:"bucket_name,omitempty"`
	Region      string            `json:"region,omitempty"`
}

// ResponseBody represents the response payload
type ResponseBody struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// LambdaFunction handles AWS Lambda requests
type LambdaFunction struct {
	logger       *slog.Logger
	crossAccount *crossaccount.Manager
}

// NewLambdaFunction creates a new Lambda function instance
func NewLambdaFunction() (*LambdaFunction, error) {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Initialize cross-account manager
	crossAccountMgr, err := crossaccount.NewManager(&crossaccount.Config{
		ToolName:    "Lambda Function",
		ToolVersion: "1.0.0",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cross-account manager: %w", err)
	}

	return &LambdaFunction{
		logger:       logger,
		crossAccount: crossAccountMgr,
	}, nil
}

// HandleRequest processes incoming Lambda requests
func (f *LambdaFunction) HandleRequest(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {
	f.logger.Info("Processing request",
		slog.String("method", event.HTTPMethod),
		slog.String("path", event.Path),
		slog.String("source_ip", event.RequestContext.Identity.SourceIP))

	// Parse request body
	var requestBody RequestBody
	if event.Body != "" {
		if err := json.Unmarshal([]byte(event.Body), &requestBody); err != nil {
			f.logger.Error("Failed to parse request body", slog.String("error", err.Error()))
			return f.errorResponse("Invalid request body", 400), nil
		}
	}

	// Route based on action
	var responseBody ResponseBody
	var statusCode int

	switch requestBody.Action {
	case "assume_role":
		responseBody, statusCode = f.handleAssumeRole(ctx, requestBody)
	case "list_s3_buckets":
		responseBody, statusCode = f.handleListS3Buckets(ctx, requestBody)
	case "get_caller_identity":
		responseBody, statusCode = f.handleGetCallerIdentity(ctx, requestBody)
	case "health_check":
		responseBody, statusCode = f.handleHealthCheck(ctx)
	default:
		responseBody = ResponseBody{
			Success: false,
			Error:   "Unknown action",
		}
		statusCode = 400
	}

	// Convert response to JSON
	bodyBytes, err := json.Marshal(responseBody)
	if err != nil {
		f.logger.Error("Failed to marshal response", slog.String("error", err.Error()))
		return f.errorResponse("Internal server error", 500), nil
	}

	return Response{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(bodyBytes),
	}, nil
}

// handleAssumeRole demonstrates cross-account role assumption
func (f *LambdaFunction) handleAssumeRole(ctx context.Context, req RequestBody) (ResponseBody, int) {
	if req.TargetRole == "" {
		return ResponseBody{
			Success: false,
			Error:   "target_role is required",
		}, 400
	}

	f.logger.Info("Assuming role",
		slog.String("target_role", req.TargetRole),
		slog.String("external_id", req.ExternalID))

	// Assume the target role
	credentials, err := f.crossAccount.AssumeRole(ctx, &crossaccount.AssumeRoleInput{
		RoleARN:    req.TargetRole,
		ExternalID: req.ExternalID,
		SessionName: "lambda-function-session",
	})
	if err != nil {
		f.logger.Error("Failed to assume role", slog.String("error", err.Error()))
		return ResponseBody{
			Success: false,
			Error:   fmt.Sprintf("Failed to assume role: %v", err),
		}, 500
	}

	// Get caller identity with assumed role
	stsClient := sts.NewFromConfig(credentials.AWSConfig)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		f.logger.Error("Failed to get caller identity", slog.String("error", err.Error()))
		return ResponseBody{
			Success: false,
			Error:   "Failed to verify assumed role",
		}, 500
	}

	return ResponseBody{
		Success: true,
		Message: "Role assumed successfully",
		Data: map[string]interface{}{
			"user_id":     *identity.UserId,
			"account":     *identity.Account,
			"arn":         *identity.Arn,
			"expires_at":  credentials.Expiration,
			"session_name": "lambda-function-session",
		},
	}, 200
}

// handleListS3Buckets lists S3 buckets using cross-account credentials
func (f *LambdaFunction) handleListS3Buckets(ctx context.Context, req RequestBody) (ResponseBody, int) {
	var awsConfig config.Config
	var err error

	if req.TargetRole != "" {
		// Use cross-account role
		f.logger.Info("Using cross-account role for S3 access",
			slog.String("target_role", req.TargetRole))

		credentials, err := f.crossAccount.AssumeRole(ctx, &crossaccount.AssumeRoleInput{
			RoleARN:     req.TargetRole,
			ExternalID:  req.ExternalID,
			SessionName: "lambda-s3-access",
		})
		if err != nil {
			return ResponseBody{
				Success: false,
				Error:   fmt.Sprintf("Failed to assume role: %v", err),
			}, 500
		}
		awsConfig = credentials.AWSConfig
	} else {
		// Use Lambda execution role
		f.logger.Info("Using Lambda execution role for S3 access")
		awsConfig, err = config.LoadDefaultConfig(ctx)
		if err != nil {
			return ResponseBody{
				Success: false,
				Error:   "Failed to load AWS configuration",
			}, 500
		}
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig)

	// List buckets
	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		f.logger.Error("Failed to list S3 buckets", slog.String("error", err.Error()))
		return ResponseBody{
			Success: false,
			Error:   fmt.Sprintf("Failed to list buckets: %v", err),
		}, 500
	}

	// Convert to response format
	buckets := make([]map[string]interface{}, len(result.Buckets))
	for i, bucket := range result.Buckets {
		buckets[i] = map[string]interface{}{
			"name":          *bucket.Name,
			"creation_date": bucket.CreationDate,
		}
	}

	return ResponseBody{
		Success: true,
		Message: fmt.Sprintf("Found %d S3 buckets", len(buckets)),
		Data: map[string]interface{}{
			"buckets": buckets,
			"count":   len(buckets),
		},
	}, 200
}

// handleGetCallerIdentity returns the current caller identity
func (f *LambdaFunction) handleGetCallerIdentity(ctx context.Context, req RequestBody) (ResponseBody, int) {
	var awsConfig config.Config
	var err error

	if req.TargetRole != "" {
		// Use cross-account role
		credentials, err := f.crossAccount.AssumeRole(ctx, &crossaccount.AssumeRoleInput{
			RoleARN:     req.TargetRole,
			ExternalID:  req.ExternalID,
			SessionName: "lambda-identity-check",
		})
		if err != nil {
			return ResponseBody{
				Success: false,
				Error:   fmt.Sprintf("Failed to assume role: %v", err),
			}, 500
		}
		awsConfig = credentials.AWSConfig
	} else {
		// Use Lambda execution role
		awsConfig, err = config.LoadDefaultConfig(ctx)
		if err != nil {
			return ResponseBody{
				Success: false,
				Error:   "Failed to load AWS configuration",
			}, 500
		}
	}

	// Get caller identity
	stsClient := sts.NewFromConfig(awsConfig)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		f.logger.Error("Failed to get caller identity", slog.String("error", err.Error()))
		return ResponseBody{
			Success: false,
			Error:   fmt.Sprintf("Failed to get identity: %v", err),
		}, 500
	}

	return ResponseBody{
		Success: true,
		Message: "Caller identity retrieved successfully",
		Data: map[string]interface{}{
			"user_id": *identity.UserId,
			"account": *identity.Account,
			"arn":     *identity.Arn,
		},
	}, 200
}

// handleHealthCheck performs a health check
func (f *LambdaFunction) handleHealthCheck(ctx context.Context) (ResponseBody, int) {
	return ResponseBody{
		Success: true,
		Message: "Lambda function is healthy",
		Data: map[string]interface{}{
			"status":    "healthy",
			"timestamp": ctx.Value("timestamp"),
		},
	}, 200
}

// errorResponse creates an error response
func (f *LambdaFunction) errorResponse(message string, statusCode int) Response {
	body := ResponseBody{
		Success: false,
		Error:   message,
	}

	bodyBytes, _ := json.Marshal(body)

	return Response{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(bodyBytes),
	}
}

// main is the Lambda function entry point
func main() {
	lambdaFunc, err := NewLambdaFunction()
	if err != nil {
		fmt.Printf("Failed to initialize Lambda function: %v\n", err)
		os.Exit(1)
	}

	lambda.Start(lambdaFunc.HandleRequest)
}