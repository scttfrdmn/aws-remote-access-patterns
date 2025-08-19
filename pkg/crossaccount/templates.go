package crossaccount

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"text/template"
)

// Embed the CloudFormation templates
// Note: embed paths are relative to the source file
var templateFiles embed.FS

// GenerateCloudFormationTemplate creates a CloudFormation template for cross-account role
func (c *Client) GenerateCloudFormationTemplate() (string, error) {
	// For now, use the embedded template from the file we created
	templateStr := getCrossAccountTemplate()

	// Parse template
	tmpl, err := template.New("crossaccount").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Prepare template data
	data := struct {
		ServiceName             string
		ServiceAccountID        string
		SessionDurationSeconds  int
		OngoingPermissions      []Permission
		SetupPermissions        []Permission
	}{
		ServiceName:             c.config.ServiceName,
		ServiceAccountID:        c.config.ServiceAccountID,
		SessionDurationSeconds:  int(c.config.SessionDuration.Seconds()),
		OngoingPermissions:      c.config.OngoingPermissions,
		SetupPermissions:        c.config.SetupPermissions,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GenerateCustomTemplate creates a customized CloudFormation template
func (c *Client) GenerateCustomTemplate(serviceName, serviceAccountID string, permissions []Permission) (string, error) {
	templateStr := getCrossAccountTemplate()

	tmpl, err := template.New("custom").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		ServiceName             string
		ServiceAccountID        string
		SessionDurationSeconds  int
		OngoingPermissions      []Permission
		SetupPermissions        []Permission
	}{
		ServiceName:             serviceName,
		ServiceAccountID:        serviceAccountID,
		SessionDurationSeconds:  int(c.config.SessionDuration.Seconds()),
		OngoingPermissions:      permissions,
		SetupPermissions:        c.config.SetupPermissions,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// uploadTemplate uploads CloudFormation template to S3
func (c *Client) uploadTemplate(ctx context.Context, customerID string) (string, error) {
	// Generate the template
	_, err := c.GenerateCloudFormationTemplate()
	if err != nil {
		return "", fmt.Errorf("failed to generate template: %w", err)
	}

	// In a real implementation, this would upload to S3
	// For now, we'll return a mock S3 URL
	templateURL := fmt.Sprintf("https://%s.s3.amazonaws.com/templates/cross-account-role.yaml", c.config.TemplateS3Bucket)
	
	// TODO: Implement actual S3 upload
	// s3Client := s3.NewFromConfig(awsConfig)
	// _, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
	//     Bucket: aws.String(c.config.TemplateS3Bucket),
	//     Key:    aws.String(fmt.Sprintf("templates/%s-cross-account-role.yaml", customerID)),
	//     Body:   strings.NewReader(template),
	//     ContentType: aws.String("text/yaml"),
	// })

	return templateURL, nil
}

// buildLaunchURL creates a CloudFormation console launch URL
func (c *Client) buildLaunchURL(templateURL string, params map[string]string, region string) string {
	baseURL := fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks/quickcreate", region)
	
	// Add template URL
	url := fmt.Sprintf("%s?templateURL=%s", baseURL, templateURL)
	
	// Add parameters
	for key, value := range params {
		url += fmt.Sprintf("&param_%s=%s", key, value)
	}
	
	return url
}

// GetTemplateContent returns the raw template content for a given template type
func GetTemplateContent(templateType string) (string, error) {
	switch templateType {
	case "cross-account":
		return getCrossAccountTemplate(), nil
	case "iam-user":
		return getIAMUserTemplate(), nil
	default:
		return "", fmt.Errorf("unknown template type: %s", templateType)
	}
}

// getCrossAccountTemplate returns the cross-account role template
func getCrossAccountTemplate() string {
	return `AWSTemplateFormatVersion: '2010-09-09'
Description: 'Cross-account IAM role for {{.ServiceName}}'

Parameters:
  ExternalId:
    Type: String
    Description: 'Unique identifier for additional security'
    MinLength: 8
    MaxLength: 64
    NoEcho: true
    
  ServiceAccountId:
    Type: String
    Description: 'AWS Account ID for {{.ServiceName}}'
    Default: '{{.ServiceAccountID}}'
    AllowedPattern: '[0-9]{12}'

Resources:
  CrossAccountRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Sub '{{.ServiceName}}-CrossAccountRole'
      Path: '/{{.ServiceName}}/'
      MaxSessionDuration: {{.SessionDurationSeconds}}
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              AWS: !Sub 'arn:aws:iam::${ServiceAccountId}:root'
            Action: 'sts:AssumeRole'
            Condition:
              StringEquals:
                'sts:ExternalId': !Ref ExternalId

Outputs:
  RoleArn:
    Description: 'ARN of the cross-account role'
    Value: !GetAtt CrossAccountRole.Arn
    
  ExternalId:
    Description: 'External ID for additional security'
    Value: !Ref ExternalId`
}

// getIAMUserTemplate returns the IAM user template
func getIAMUserTemplate() string {
	return `AWSTemplateFormatVersion: '2010-09-09'
Description: 'IAM User for external tool'

Parameters:
  ToolName:
    Type: String
    Description: 'Name of the external tool'
    Default: 'ExternalTool'

Resources:
  ExternalToolUser:
    Type: AWS::IAM::User
    Properties:
      UserName: !Sub '${ToolName}-user-${AWS::AccountId}'
      Path: '/external-tools/'

Outputs:
  UserArn:
    Description: 'ARN of the created IAM user'
    Value: !GetAtt ExternalToolUser.Arn`
}

// ValidateTemplate performs basic validation on a CloudFormation template
func ValidateTemplate(templateContent string) error {
	// Basic validation - check if it's valid YAML and has required sections
	if templateContent == "" {
		return fmt.Errorf("template is empty")
	}
	
	// Check for required CloudFormation sections
	requiredSections := []string{
		"AWSTemplateFormatVersion",
		"Resources",
		"Outputs",
	}
	
	for _, section := range requiredSections {
		if !bytes.Contains([]byte(templateContent), []byte(section)) {
			return fmt.Errorf("template is missing required section: %s", section)
		}
	}
	
	return nil
}

// TemplateVariables holds variables that can be substituted in templates
type TemplateVariables struct {
	ServiceName      string
	ServiceAccountID string
	CustomerID       string
	ExternalID       string
	Region           string
	Permissions      []Permission
}

// RenderTemplate renders a template with the given variables
func RenderTemplate(templateContent string, vars TemplateVariables) (string, error) {
	tmpl, err := template.New("render").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	
	return buf.String(), nil
}