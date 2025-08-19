package awsauth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// setupSSO performs AWS SSO setup
func (c *Client) setupSSO(ctx context.Context) error {
	fmt.Println("\n🔐 Setting up AWS SSO")

	ssoAuth := NewSSOAuthenticator(c.config)
	cfg, err := ssoAuth.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("SSO setup failed: %w", err)
	}

	// Test the configuration
	if !c.validateCredentials(ctx, cfg) {
		return fmt.Errorf("SSO credentials don't have required permissions")
	}

	fmt.Println("✅ AWS SSO setup completed successfully!")
	return nil
}

// setupIAMUser guides through IAM user setup with CloudFormation
func (c *Client) setupIAMUser(ctx context.Context) error {
	fmt.Println("\n🔑 Setting up IAM User")
	fmt.Printf("We'll create an IAM user with minimal permissions for %s\n", c.config.ToolName)

	// Generate CloudFormation template
	template, err := c.generateIAMTemplate()
	if err != nil {
		return fmt.Errorf("failed to generate CloudFormation template: %w", err)
	}

	// Save template
	tempDir := os.TempDir()
	templatePath := filepath.Join(tempDir, fmt.Sprintf("%s-iam-setup.yaml", c.config.ToolName))

	if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	fmt.Printf("\n📄 CloudFormation template saved to:\n%s\n", templatePath)

	fmt.Println("\nNext steps:")
	fmt.Println("1. Open the AWS CloudFormation console in your browser")
	fmt.Println("2. Create a new stack using the template file above")
	fmt.Println("3. After the stack is created, find the Outputs tab")
	fmt.Println("4. Copy the AccessKeyId and SecretAccessKey values")
	fmt.Println("5. Return here to complete the setup")

	// Open CloudFormation console
	cfURL := "https://console.aws.amazon.com/cloudformation/home"
	fmt.Printf("\n🌐 Open CloudFormation console? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" || input == "y" || input == "yes" {
		if err := c.openBrowser(cfURL); err != nil {
			fmt.Printf("Could not open browser. Please visit: %s\n", cfURL)
		}
	}

	// Wait for user to complete CloudFormation setup
	fmt.Print("\nPress Enter when you have the access keys ready...")
	reader.ReadString('\n')

	return c.promptForCredentials()
}

// setupExistingProfile helps user select and validate existing AWS profile
func (c *Client) setupExistingProfile(ctx context.Context) error {
	fmt.Println("\n📋 Using existing AWS profile")

	profiles := c.listAWSProfiles()
	if len(profiles) == 0 {
		fmt.Println("No existing AWS profiles found.")
		fmt.Println("Please run 'aws configure' first or choose a different authentication method.")
		return fmt.Errorf("no AWS profiles found")
	}

	fmt.Println("Available AWS profiles:")
	for i, profile := range profiles {
		fmt.Printf("%d. %s\n", i+1, profile)
	}

	fmt.Print("Select profile to use [1]: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice := 1
	if input != "" {
		if c, err := strconv.Atoi(input); err == nil {
			choice = c
		}
	}

	if choice < 1 || choice > len(profiles) {
		return fmt.Errorf("invalid profile selection")
	}

	selectedProfile := profiles[choice-1]

	// Test the profile
	cfg, err := c.loadProfile(ctx, selectedProfile)
	if err != nil {
		return fmt.Errorf("failed to load profile %s: %w", selectedProfile, err)
	}

	if !c.validateCredentials(ctx, cfg) {
		return fmt.Errorf("profile %s doesn't have required permissions", selectedProfile)
	}

	// Save as our tool's profile
	if selectedProfile != c.profileName {
		if err := c.copyProfile(selectedProfile, c.profileName); err != nil {
			return fmt.Errorf("failed to copy profile: %w", err)
		}
	}

	fmt.Printf("✅ Successfully configured to use profile: %s\n", selectedProfile)
	return nil
}

// generateIAMTemplate creates CloudFormation template for IAM user
func (c *Client) generateIAMTemplate() (string, error) {
	permissions := c.buildPermissionStatements()

	templateStr := `AWSTemplateFormatVersion: '2010-09-09'
Description: 'IAM User for {{.ToolName}}'

Resources:
  {{.ToolName}}User:
    Type: AWS::IAM::User
    Properties:
      UserName: !Sub '{{.ToolName}}-user-${AWS::AccountId}'
      Path: '/external-tools/'
      
  {{.ToolName}}AccessKey:
    Type: AWS::IAM::AccessKey
    Properties:
      UserName: !Ref {{.ToolName}}User
      
  {{.ToolName}}Policy:
    Type: AWS::IAM::UserPolicy
    Properties:
      UserName: !Ref {{.ToolName}}User
      PolicyName: '{{.ToolName}}Permissions'
      PolicyDocument:
        Version: '2012-10-17'
        Statement:
{{.Permissions}}

Outputs:
  AccessKeyId:
    Description: 'Access Key ID for {{.ToolName}}'
    Value: !Ref {{.ToolName}}AccessKey
    
  SecretAccessKey:
    Description: 'Secret Access Key'
    Value: !GetAtt {{.ToolName}}AccessKey.SecretAccessKey
    
  SetupInstructions:
    Description: 'Next steps'
    Value: 'Copy the AccessKeyId and SecretAccessKey values and return to your tool setup'`

	tmpl, err := template.New("iam").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		ToolName    string
		Permissions string
	}{
		ToolName:    c.config.ToolName,
		Permissions: permissions,
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
}

// buildPermissionStatements creates IAM policy statements
func (c *Client) buildPermissionStatements() string {
	if len(c.config.CustomPermissions) > 0 {
		return c.formatCustomPermissions()
	}

	// Build from required actions
	actions := c.config.RequiredActions
	if len(actions) == 0 {
		actions = []string{"sts:GetCallerIdentity"}
	}

	var statements []string

	// Group actions by service for better organization
	serviceActions := make(map[string][]string)
	for _, action := range actions {
		parts := strings.SplitN(action, ":", 2)
		if len(parts) == 2 {
			service := parts[0]
			serviceActions[service] = append(serviceActions[service], action)
		}
	}

	for service, serviceActionsSlice := range serviceActions {
		statement := fmt.Sprintf(`          - Sid: '%s%sPermissions'
            Effect: Allow
            Action:
%s
            Resource: '*'`,
			c.config.ToolName,
			strings.Title(service),
			c.formatActions(serviceActionsSlice),
		)
		statements = append(statements, statement)
	}

	return strings.Join(statements, "\n")
}

// formatActions formats action list for YAML
func (c *Client) formatActions(actions []string) string {
	var formatted []string
	for _, action := range actions {
		formatted = append(formatted, fmt.Sprintf("              - '%s'", action))
	}
	return strings.Join(formatted, "\n")
}

// formatCustomPermissions formats custom permissions for CloudFormation
func (c *Client) formatCustomPermissions() string {
	var statements []string
	
	for _, perm := range c.config.CustomPermissions {
		statement := fmt.Sprintf(`          - Sid: '%s'
            Effect: %s
            Action:
%s
            Resource:
%s`,
			perm.Sid,
			perm.Effect,
			c.formatActions(perm.Actions),
			c.formatResources(perm.Resources),
		)
		statements = append(statements, statement)
	}
	
	return strings.Join(statements, "\n")
}

// formatResources formats resource list for YAML
func (c *Client) formatResources(resources []string) string {
	var formatted []string
	for _, resource := range resources {
		formatted = append(formatted, fmt.Sprintf("              - '%s'", resource))
	}
	return strings.Join(formatted, "\n")
}

// promptForCredentials prompts user to enter AWS credentials
func (c *Client) promptForCredentials() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Access Key ID: ")
	accessKey, _ := reader.ReadString('\n')
	accessKey = strings.TrimSpace(accessKey)

	fmt.Print("Enter Secret Access Key: ")
	secretKey, _ := reader.ReadString('\n')
	secretKey = strings.TrimSpace(secretKey)

	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("access key and secret key are required")
	}

	return c.saveCredentials(accessKey, secretKey)
}

// saveCredentials saves credentials to AWS credentials file
func (c *Client) saveCredentials(accessKey, secretKey string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	awsDir := filepath.Join(homeDir, ".aws")
	if err := os.MkdirAll(awsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	credFile := filepath.Join(awsDir, "credentials")

	// Read existing credentials file
	content := ""
	if data, err := os.ReadFile(credFile); err == nil {
		content = string(data)
	}

	// Add/update our profile
	profileSection := fmt.Sprintf("\n[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\nregion = %s\n",
		c.profileName, accessKey, secretKey, c.config.DefaultRegion)

	// Remove existing profile if it exists
	lines := strings.Split(content, "\n")
	var newLines []string
	inOurProfile := false

	for _, line := range lines {
		if strings.TrimSpace(line) == fmt.Sprintf("[%s]", c.profileName) {
			inOurProfile = true
			continue
		}
		if strings.HasPrefix(line, "[") && line != fmt.Sprintf("[%s]", c.profileName) {
			inOurProfile = false
		}
		if !inOurProfile {
			newLines = append(newLines, line)
		}
	}

	content = strings.Join(newLines, "\n") + profileSection

	// Write back with secure permissions
	if err := os.WriteFile(credFile, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("✅ Credentials saved to profile: %s\n", c.profileName)
	return nil
}

// listAWSProfiles lists available AWS profiles
func (c *Client) listAWSProfiles() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	credFile := filepath.Join(homeDir, ".aws", "credentials")
	configFile := filepath.Join(homeDir, ".aws", "config")

	profiles := make(map[string]bool)

	// Read credentials file
	if data, err := os.ReadFile(credFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				profile := strings.Trim(line, "[]")
				if profile != "" {
					profiles[profile] = true
				}
			}
		}
	}

	// Read config file
	if data, err := os.ReadFile(configFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
				profile := strings.TrimPrefix(strings.Trim(line, "[]"), "profile ")
				if profile != "" {
					profiles[profile] = true
				}
			}
		}
	}

	var result []string
	for profile := range profiles {
		result = append(result, profile)
	}

	return result
}

// copyProfile copies AWS profile configuration
func (c *Client) copyProfile(source, dest string) error {
	// This would copy profile settings from source to dest
	// For now, just a placeholder
	return nil
}

// openBrowser opens URL in default browser
func (c *Client) openBrowser(url string) error {
	// Reuse the implementation from sso.go
	ssoAuth := NewSSOAuthenticator(c.config)
	return ssoAuth.openBrowser(url)
}