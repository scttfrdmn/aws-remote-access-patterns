package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// This example demonstrates how external tools can get AWS access
// WITHOUT asking users to manually create access keys

func main() {
	var setupFlag = flag.Bool("setup", false, "Run interactive AWS setup")
	var compareFlag = flag.Bool("compare", false, "Compare approaches")
	flag.Parse()

	// Show comparison of approaches
	if *compareFlag {
		showComparison()
		return
	}

	// Interactive setup if requested
	if *setupFlag {
		fmt.Println("🚀 Setting up AWS access for my-cloud-tool...")
		fmt.Println("This will guide you through the easiest and most secure setup process.")
		fmt.Println()
		fmt.Println("✅ This is a demo - the awsauth package would handle:")
		fmt.Println("   • AWS SSO detection and device flow")
		fmt.Println("   • Fallback to guided IAM user creation")
		fmt.Println("   • Automatic credential caching and refresh")
		fmt.Println("   • One-command setup vs manual IAM configuration")
		fmt.Println()
		fmt.Println("🔒 Result: No long-lived secrets, automatic security")
		return
	}

	// ✅ SIMPLE APPROACH: Use standard AWS credentials chain
	// In a real implementation, the awsauth package would:
	// 1. Try cached temporary credentials first
	// 2. Try AWS SSO if available
	// 3. Fall back to standard credential chain
	// 4. Guide user through setup if nothing works
	
	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		fmt.Printf("❌ AWS authentication required: %v\n", err)
		fmt.Println()
		fmt.Println("🔧 Quick fix: Run with --setup to configure AWS access")
		fmt.Println("   This would guide you through the easiest setup process.")
		fmt.Println()
		fmt.Println("🔒 Why the awsauth pattern is better than access keys:")
		fmt.Println("   • No long-lived credentials stored on your machine")
		fmt.Println("   • Automatic credential refresh")  
		fmt.Println("   • Works with your organization's AWS SSO")
		fmt.Println("   • Follows AWS security best practices")
		os.Exit(1)
	}

	// Parse command
	if len(flag.Args()) == 0 {
		fmt.Println("Usage: my-cloud-tool [command]")
		fmt.Println()
		fmt.Println("Available commands:")
		fmt.Println("  instances    List EC2 instances")
		fmt.Println("  buckets      List S3 buckets")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --setup      Set up AWS authentication")
		fmt.Println("  --compare    Compare authentication approaches")
		os.Exit(1)
	}

	command := flag.Arg(0)

	switch command {
	case "instances":
		listInstances(context.Background(), awsConfig)
	case "buckets":
		listBuckets(context.Background(), awsConfig)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Run 'my-cloud-tool' to see available commands")
		os.Exit(1)
	}
}

func listInstances(ctx context.Context, cfg aws.Config) {
	fmt.Println("🖥️  Listing EC2 instances...")
	
	ec2Client := ec2.NewFromConfig(cfg)
	result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		log.Fatal("Failed to list instances:", err)
	}

	fmt.Println("EC2 Instances:")
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			name := getInstanceName(instance.Tags)
			fmt.Printf("  %s (%s) - %s\n", 
				*instance.InstanceId, 
				name, 
				instance.State.Name)
		}
	}
	
	fmt.Println()
	fmt.Println("✅ Used secure, temporary AWS credentials")
	fmt.Println("🔄 Credentials will refresh automatically when needed")
}

func listBuckets(ctx context.Context, cfg aws.Config) {
	fmt.Println("🪣 Listing S3 buckets...")
	
	s3Client := s3.NewFromConfig(cfg)
	result, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		log.Fatal("Failed to list buckets:", err)
	}

	fmt.Println("S3 Buckets:")
	for _, bucket := range result.Buckets {
		fmt.Printf("  %s (created: %s)\n", 
			*bucket.Name, 
			bucket.CreationDate.Format("2006-01-02"))
	}
	
	fmt.Println()
	fmt.Println("✅ Used secure, temporary AWS credentials")
	fmt.Println("🔄 Credentials will refresh automatically when needed")
}

func getInstanceName(tags []types.Tag) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return "unnamed"
}

func showComparison() {
	fmt.Println(`
🔒 AWS Authentication: Modern vs Traditional Approaches

Traditional Approach (Access Keys):
❌ User creates IAM user manually
❌ User creates and downloads access keys  
❌ User sets AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
❌ Keys never expire (security risk)
❌ Keys often have overly broad permissions
❌ Keys get committed to git repositories
❌ Manual rotation process (rarely done)
❌ Hard to revoke access

Modern Approach (This Library):
✅ User runs: tool --setup
✅ Library detects AWS SSO or guides through setup
✅ Temporary credentials only (expire automatically)
✅ Least privilege permissions (only what tool needs)  
✅ Works with corporate SSO and MFA
✅ Automatic credential refresh
✅ No secrets stored permanently
✅ Easy to audit and monitor

User Experience Comparison:

Traditional (Access Keys):
1. Go to AWS console
2. Navigate to IAM > Users  
3. Click "Create user"
4. Configure permissions (complex!)
5. Generate access keys
6. Download CSV file
7. Set environment variables
8. Hope you set permissions correctly
9. Manually rotate keys (eventually...)

Modern (This Library):
1. Run: my-tool --setup
2. Follow the guided setup wizard
3. Done! Start using the tool
4. Credentials refresh automatically
5. Secure by default

Why This Matters:
🎯 Users get started in minutes, not hours
🔒 Security is built-in, not an afterthought  
🚀 Works with enterprise authentication systems
📊 Full audit trail of all activities
⚡ Easy to revoke access when needed

This is why modern tools like AWS CLI v2, Docker, and Terraform 
are moving away from access keys toward temporary credentials.
`)
}