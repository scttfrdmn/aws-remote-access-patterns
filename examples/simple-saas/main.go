package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/scttfrdmn/aws-remote-access-patterns/pkg/crossaccount"
)

// This example shows how much simpler and more secure cross-account roles are
// compared to traditional access keys

func main() {
	// ✅ SECURE APPROACH: Cross-account roles
	// No access keys stored anywhere - everything is temporary and scoped
	
	client, err := crossaccount.New(crossaccount.SimpleConfig(
		"DataAnalyzer",                    // Your service name
		os.Getenv("AWS_ACCOUNT_ID"),       // Your AWS account ID  
		os.Getenv("TEMPLATE_S3_BUCKET"),   // S3 bucket for hosting templates
	))
	if err != nil {
		log.Fatal("Failed to initialize:", err)
	}

	r := gin.Default()

	// Customer onboarding - generates a single click setup link
	r.POST("/customers/:id/aws-setup", func(c *gin.Context) {
		customerID := c.Param("id")
		customerName := c.PostForm("name")

		// Generate one-click setup link - customer just clicks and follows wizard
		setupResp, err := client.GenerateSetupLink(customerID, customerName)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "🎉 Setup link generated! Customer clicks this link and follows the guided setup.",
			"setup_link": setupResp.LaunchURL,
			"external_id": setupResp.ExternalID, // They'll need this from CloudFormation outputs
			"next_steps": []string{
				"1. Send the setup_link to your customer",
				"2. Customer clicks link and follows the CloudFormation wizard",  
				"3. Customer copies Role ARN and External ID from stack outputs",
				"4. Customer calls your /complete-setup endpoint with those values",
				"5. Done! You can now securely access their AWS resources",
			},
		})
	})

	// Customer completes setup by providing the role ARN from CloudFormation
	r.POST("/customers/:id/complete-setup", func(c *gin.Context) {
		customerID := c.Param("id")
		
		var req struct {
			RoleARN    string `json:"role_arn"`
			ExternalID string `json:"external_id"`
		}
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		// Verify and store the role - this tests that everything works
		err := client.CompleteSetup(c.Request.Context(), &crossaccount.SetupCompleteRequest{
			CustomerID: customerID,
			RoleARN:    req.RoleARN,
			ExternalID: req.ExternalID,
		})
		if err != nil {
			c.JSON(400, gin.H{"error": fmt.Sprintf("Setup verification failed: %v", err)})
			return
		}

		c.JSON(200, gin.H{
			"message": "✅ Setup complete! Your service can now securely access the customer's AWS resources.",
			"security_benefits": []string{
				"🔒 No access keys stored anywhere",
				"⏰ Temporary credentials only (expire automatically)",
				"🎯 Least privilege permissions (only what you need)",
				"🔍 Full audit trail of all access", 
				"🚫 Customer can revoke access instantly by deleting CloudFormation stack",
			},
		})
	})

	// Your business logic - accessing customer AWS resources
	r.GET("/customers/:id/data-analysis", func(c *gin.Context) {
		customerID := c.Param("id")

		// ✅ Get temporary, scoped credentials for this specific customer
		awsConfig, err := client.AssumeRole(c.Request.Context(), customerID)
		if err != nil {
			c.JSON(500, gin.H{
				"error": "Failed to access customer AWS account",
				"details": err.Error(),
				"common_causes": []string{
					"Customer hasn't completed setup yet",
					"Customer deleted the CloudFormation stack", 
					"Role permissions were modified",
				},
			})
			return
		}

		// Use AWS services with customer's permissions - just like normal AWS SDK usage
		s3Client := s3.NewFromConfig(awsConfig)
		
		// List their S3 buckets (this will only work if they granted you permission)
		buckets, err := s3Client.ListBuckets(c.Request.Context(), &s3.ListBucketsInput{})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to list buckets: " + err.Error()})
			return
		}

		bucketNames := make([]string, len(buckets.Buckets))
		for i, bucket := range buckets.Buckets {
			bucketNames[i] = *bucket.Name
		}

		c.JSON(200, gin.H{
			"customer_id": customerID,
			"buckets":     bucketNames,
			"analysis_complete": true,
			"security_info": "✅ Used temporary credentials that expire in 1 hour",
		})
	})

	// Security cleanup - remove setup permissions after initial setup
	r.POST("/customers/:id/remove-setup-permissions", func(c *gin.Context) {
		customerID := c.Param("id")

		instructions, err := client.RemoveSetupPermissions(customerID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message": "🔒 Ready to remove setup permissions for enhanced security",
			"instructions": instructions.Instructions,
			"automation_script": instructions.AutomationScript,
			"why_important": "Setup permissions are broader than needed for daily operations. Removing them follows security best practices.",
		})
	})

	// Compare this to the old way with access keys
	r.GET("/compare-approaches", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"cross_account_roles_approach": gin.H{
				"security": "✅ Temporary credentials only, automatic expiration",
				"permissions": "✅ Least privilege, scoped to specific resources",
				"revocation": "✅ Instant - customer deletes CloudFormation stack",
				"audit_trail": "✅ Full CloudTrail logging of all activities",
				"setup_complexity": "✅ One-click CloudFormation deployment",
				"credential_storage": "✅ No long-lived secrets stored anywhere",
				"rotation": "✅ Automatic - no action needed",
			},
			"access_keys_approach": gin.H{
				"security": "❌ Long-lived credentials, never expire unless manually rotated",
				"permissions": "❌ Often over-privileged (admin access common)",
				"revocation": "❌ Manual process, customers often forget",
				"audit_trail": "❌ Limited, hard to correlate activities",
				"setup_complexity": "❌ Manual IAM user creation, policy attachment",
				"credential_storage": "❌ Long-lived secrets stored in databases",
				"rotation": "❌ Manual process, rarely done",
			},
			"why_cross_account_is_better": []string{
				"🔒 No secrets to leak - everything is temporary",
				"🎯 Precise permissions - only what your service actually needs",
				"⚡ Easy revocation - customer deletes one CloudFormation stack",
				"📊 Complete audit trail - every action logged in CloudTrail",
				"🚀 Better UX - one-click setup vs manual IAM configuration",
				"💼 Enterprise friendly - works with AWS Organizations and SCPs",
			},
		})
	})

	fmt.Println("🚀 Starting DataAnalyzer service...")
	fmt.Println("📖 Try these endpoints to see the secure cross-account approach:")
	fmt.Println("   POST /customers/acme/aws-setup")
	fmt.Println("   GET  /compare-approaches")
	fmt.Println("   👉 Visit http://localhost:8080/compare-approaches to see why this is better than access keys")
	
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Example of what you would do with the OLD access key approach
// (Don't do this - it's insecure!)
func badAccessKeyExample() {
	/*
	❌ INSECURE APPROACH: Access keys (don't do this!)
	
	type CustomerCredentials struct {
		AccessKey    string // Long-lived, never expires
		SecretKey    string // Stored in your database forever
		Permissions  string // Probably "AdminAccess" because IAM is hard
	}
	
	Problems with this approach:
	1. Long-lived credentials that never expire
	2. Stored permanently in your database (security risk)
	3. Often over-privileged because precise permissions are hard
	4. Difficult for customers to revoke access
	5. No audit trail of what you're actually doing
	6. Credentials get leaked in logs, repos, etc.
	7. Rotation is manual and rarely done
	8. Setup is complex (customer needs to create IAM user, policies, etc.)
	*/
	
	fmt.Println("❌ Don't use access keys! Use cross-account roles instead.")
}

// Demonstration of security benefits
func demonstrateSecurityBenefits() {
	fmt.Println(`
🔒 Security Benefits of Cross-Account Roles vs Access Keys:

┌─────────────────────┬──────────────────────┬─────────────────────┐
│ Aspect              │ Cross-Account Roles  │ Access Keys         │
├─────────────────────┼──────────────────────┼─────────────────────┤
│ Credential Lifetime │ ✅ 1 hour (temp)     │ ❌ Forever          │
│ Permission Scope    │ ✅ Least privilege   │ ❌ Often admin      │
│ Storage Security    │ ✅ No secrets stored │ ❌ Long-lived keys  │
│ Revocation Speed    │ ✅ Instant           │ ❌ Manual process   │
│ Audit Trail         │ ✅ Complete logging  │ ❌ Limited visibility│
│ Setup Complexity    │ ✅ One-click         │ ❌ Manual IAM work  │
│ Rotation Required   │ ✅ Automatic         │ ❌ Manual (rarely)  │
│ Leak Impact         │ ✅ Minimal (expires) │ ❌ Full compromise  │
└─────────────────────┴──────────────────────┴─────────────────────┘

This is why companies like Datadog, Coiled, and others have moved to cross-account roles.
	`)
}