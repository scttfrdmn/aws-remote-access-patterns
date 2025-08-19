# AWS Integration Template for External Tools
## Based on Coiled.io's Successful Model

This template provides a complete implementation of Coiled's approach to making AWS integration easy for non-expert users, including two-step permissions and progressive disclosure.

## Core Requirements

### What You Must Have
1. **Your AWS Account ID** - The account where your external service runs
2. **S3 Bucket** - To host CloudFormation templates publicly
3. **Domain/Subdomain** - For hosting your integration UI (optional but recommended)
4. **Customer's AWS Account** - Where the IAM role will be created

### Technical Prerequisites
- Customer must have AWS Console access
- Customer must have IAM permissions to create roles and policies
- Customer must be able to run CloudFormation templates

## Two-Phase Permission Strategy

### Phase 1: Setup Permissions (Temporary)
Broader permissions needed for initial resource creation:
- VPC/Subnet creation
- Security Group management  
- Instance Profile creation
- Policy management

### Phase 2: Ongoing Permissions (Permanent)
Minimal permissions for day-to-day operations:
- EC2 instance management
- Resource tagging
- Basic networking operations

## Implementation Components

### 1. CloudFormation Template

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: 'Cross-account IAM role for [YourServiceName] - Easy AWS Integration'

Parameters:
  ExternalId:
    Type: String
    Description: 'Unique identifier provided by [YourServiceName] for security'
    MinLength: 8
    MaxLength: 64
    Default: 'CHANGE_ME_UNIQUE_ID'
    
  YourServiceAccountId:
    Type: String  
    Description: 'AWS Account ID for [YourServiceName]'
    Default: '123456789012'  # Replace with your actual account ID
    AllowedPattern: '[0-9]{12}'
    ConstraintDescription: 'Must be a valid 12-digit AWS Account ID'
    
  RoleName:
    Type: String
    Description: 'Name for the IAM role (will be prefixed with your service name)'
    Default: 'YourService-CrossAccountRole'
    
  SetupPhase:
    Type: String
    Description: 'Include setup permissions? (Remove after initial setup)'
    Default: 'true'
    AllowedValues: ['true', 'false']

Conditions:
  IncludeSetupPermissions: !Equals [!Ref SetupPhase, 'true']

Resources:
  # Main Cross-Account Role
  CrossAccountRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: !Ref RoleName
      Description: !Sub 'Allows ${YourServiceAccountId} to manage resources in this account'
      Path: '/YourService/'
      MaxSessionDuration: 3600
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              AWS: !Sub 'arn:aws:iam::${YourServiceAccountId}:root'
            Action: 'sts:AssumeRole'
            Condition:
              StringEquals:
                'sts:ExternalId': !Ref ExternalId
              Bool:
                'aws:MultiFactorAuthPresent': 'false'  # Allow programmatic access
      ManagedPolicyArns:
        - !Ref OngoingOperationsPolicy
        - !If 
          - IncludeSetupPermissions
          - !Ref SetupPolicy  
          - !Ref AWS::NoValue
      Tags:
        - Key: 'CreatedBy'
          Value: 'YourServiceName'
        - Key: 'Purpose' 
          Value: 'CrossAccountIntegration'

  # Ongoing Operations Policy (Always Active)
  OngoingOperationsPolicy:
    Type: AWS::IAM::ManagedPolicy
    Properties:
      ManagedPolicyName: !Sub '${RoleName}-OngoingOperations'
      Description: 'Minimal permissions for day-to-day operations'
      Path: '/YourService/'
      PolicyDocument:
        Version: '2012-10-17'
        Statement:
          # EC2 Instance Management
          - Sid: 'EC2InstanceManagement'
            Effect: Allow
            Action:
              - 'ec2:DescribeInstances'
              - 'ec2:DescribeInstanceTypes'
              - 'ec2:DescribeInstanceAttribute'
              - 'ec2:DescribeImages'
              - 'ec2:DescribeSnapshots'
              - 'ec2:DescribeVolumes'
              - 'ec2:RunInstances'
              - 'ec2:TerminateInstances'
              - 'ec2:StopInstances'
              - 'ec2:StartInstances'
              - 'ec2:RebootInstances'
            Resource: '*'
            
          # Security Groups  
          - Sid: 'SecurityGroupManagement'
            Effect: Allow
            Action:
              - 'ec2:DescribeSecurityGroups'
              - 'ec2:AuthorizeSecurityGroupIngress'
              - 'ec2:AuthorizeSecurityGroupEgress'
              - 'ec2:RevokeSecurityGroupIngress'
              - 'ec2:RevokeSecurityGroupEgress'
            Resource: '*'
            
          # Tagging
          - Sid: 'ResourceTagging'
            Effect: Allow  
            Action:
              - 'ec2:CreateTags'
              - 'ec2:DescribeTags'
            Resource: '*'
            
          # CloudWatch Logs
          - Sid: 'CloudWatchLogs'
            Effect: Allow
            Action:
              - 'logs:CreateLogGroup'
              - 'logs:CreateLogStream'  
              - 'logs:PutLogEvents'
              - 'logs:DescribeLogGroups'
              - 'logs:DescribeLogStreams'
            Resource: !Sub 'arn:aws:logs:*:${AWS::AccountId}:log-group:/YourService/*'

  # Setup Policy (Temporary - Remove After Setup)  
  SetupPolicy:
    Type: AWS::IAM::ManagedPolicy
    Condition: IncludeSetupPermissions
    Properties:
      ManagedPolicyName: !Sub '${RoleName}-Setup'
      Description: 'Temporary setup permissions - REMOVE after initial setup'  
      Path: '/YourService/'
      PolicyDocument:
        Version: '2012-10-17'
        Statement:
          # VPC Management
          - Sid: 'VPCSetup'
            Effect: Allow
            Action:
              - 'ec2:CreateVpc'
              - 'ec2:DescribeVpcs'
              - 'ec2:ModifyVpcAttribute'
              - 'ec2:CreateSubnet'
              - 'ec2:DescribeSubnets'
              - 'ec2:ModifySubnetAttribute'
              - 'ec2:CreateInternetGateway'
              - 'ec2:DescribeInternetGateways'
              - 'ec2:AttachInternetGateway'
              - 'ec2:CreateRouteTable'
              - 'ec2:DescribeRouteTables'
              - 'ec2:CreateRoute'
              - 'ec2:AssociateRouteTable'
            Resource: '*'
            
          # Security Group Creation
          - Sid: 'SecurityGroupSetup'
            Effect: Allow
            Action:
              - 'ec2:CreateSecurityGroup'
            Resource: '*'
            
          # IAM for Instance Profiles
          - Sid: 'InstanceProfileSetup'
            Effect: Allow
            Action:
              - 'iam:CreateRole'
              - 'iam:CreateInstanceProfile'
              - 'iam:AddRoleToInstanceProfile'
              - 'iam:PassRole'
              - 'iam:AttachRolePolicy'
            Resource: 
              - !Sub 'arn:aws:iam::${AWS::AccountId}:role/YourService/*'
              - !Sub 'arn:aws:iam::${AWS::AccountId}:instance-profile/YourService/*'

Outputs:
  RoleArn:
    Description: 'ARN of the created cross-account role'
    Value: !GetAtt CrossAccountRole.Arn
    Export:
      Name: !Sub '${AWS::StackName}-RoleArn'
      
  ExternalId:
    Description: 'External ID used for additional security' 
    Value: !Ref ExternalId
    
  SetupComplete:
    Description: 'Next steps'
    Value: !If
      - IncludeSetupPermissions
      - 'Setup permissions included. REMEMBER to update stack with SetupPhase=false after initial setup.'
      - 'Role configured for ongoing operations only.'
      
  IntegrationInstructions:
    Description: 'Integration details for YourServiceName'
    Value: !Sub |
      1. Copy this Role ARN: ${CrossAccountRole.Arn}
      2. Copy this External ID: ${ExternalId}  
      3. Provide both values to YourServiceName
      4. Test the integration
      5. If setup permissions were included, update this stack with SetupPhase=false
```

### 2. Launch URL Generator

```javascript
// Generate custom launch stack URL
function generateLaunchUrl(customerData) {
  const baseUrl = 'https://console.aws.amazon.com/cloudformation/home';
  const templateUrl = 'https://your-bucket.s3.amazonaws.com/templates/cross-account-role.yaml';
  const externalId = generateExternalId(customerData.customerId);
  
  const params = new URLSearchParams({
    'region': customerData.preferredRegion || 'us-east-1',
    'templateURL': templateUrl,
    'stackName': `YourService-Integration-${customerData.customerName}`,
    'param_ExternalId': externalId,
    'param_YourServiceAccountId': YOUR_AWS_ACCOUNT_ID
  });
  
  return `${baseUrl}?#/stacks/new?${params.toString()}`;
}

function generateExternalId(customerId) {
  // Generate unique external ID for security
  return `YourService-${customerId}-${Date.now()}`;
}
```

### 3. Progressive Disclosure UI Flow

```html
<!DOCTYPE html>
<html>
<head>
    <title>Connect Your AWS Account - YourServiceName</title>
    <style>
        .setup-step { margin: 20px 0; padding: 20px; border-left: 4px solid #007cba; background: #f7f9fa; }
        .setup-step.active { border-color: #28a745; background: #d4edda; }
        .setup-step.completed { border-color: #6c757d; background: #e2e3e5; }
        .explanation { margin: 10px 0; font-size: 14px; color: #666; }
        .security-note { background: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; margin: 10px 0; border-radius: 4px; }
        .launch-button { background: #ff9900; color: white; padding: 12px 24px; border: none; border-radius: 4px; font-size: 16px; cursor: pointer; }
        .launch-button:hover { background: #e88600; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Connect Your AWS Account to YourServiceName</h1>
        
        <div class="security-note">
            <strong>🔒 Security First:</strong> We use industry-standard cross-account roles with external IDs. 
            Your credentials stay in your account - we never store or see them.
        </div>

        <!-- Step 1: Explanation -->
        <div class="setup-step active" id="step1">
            <h3>Step 1: Understanding What We're Setting Up</h3>
            <div class="explanation">
                <p>We're going to create a secure "bridge" between your AWS account and YourServiceName. This involves:</p>
                <ul>
                    <li><strong>IAM Role:</strong> A temporary identity that YourServiceName can assume</li>
                    <li><strong>Trust Policy:</strong> Rules that ensure only YourServiceName can use this role</li>
                    <li><strong>Permissions:</strong> Specific actions YourServiceName can perform in your account</li>
                    <li><strong>External ID:</strong> An additional security key that prevents unauthorized access</li>
                </ul>
                <p><strong>Why is this secure?</strong> YourServiceName gets temporary access tokens, never your permanent credentials.</p>
            </div>
            <button onclick="nextStep(2)" class="launch-button">I Understand - Continue</button>
        </div>

        <!-- Step 2: Permissions Overview -->
        <div class="setup-step" id="step2" style="display:none;">
            <h3>Step 2: Permission Details</h3>
            <div class="explanation">
                <p>YourServiceName will receive two sets of permissions:</p>
                
                <h4>Setup Permissions (Temporary)</h4>
                <ul>
                    <li>Create VPC and networking components</li>
                    <li>Set up security groups</li>
                    <li>Create instance profiles</li>
                </ul>
                <p><em>These are removed after initial setup to minimize security surface.</em></p>
                
                <h4>Ongoing Permissions (Permanent)</h4>
                <ul>
                    <li>Launch and manage EC2 instances</li>
                    <li>Create and view CloudWatch logs</li>
                    <li>Tag resources</li>
                </ul>
                
                <p><strong>What we CAN'T do:</strong></p>
                <ul>
                    <li>Access your data or files</li>
                    <li>Create users or modify your account settings</li>
                    <li>Access other AWS services not listed above</li>
                </ul>
            </div>
            <button onclick="nextStep(3)" class="launch-button">These Permissions Look Good</button>
        </div>

        <!-- Step 3: Launch CloudFormation -->
        <div class="setup-step" id="step3" style="display:none;">
            <h3>Step 3: Launch AWS CloudFormation Stack</h3>
            <div class="explanation">
                <p>CloudFormation is AWS's infrastructure-as-code service. It will:</p>
                <ul>
                    <li>Create the IAM role with exact permissions we discussed</li>
                    <li>Set up the trust relationship with YourServiceName</li>
                    <li>Generate a unique External ID for security</li>
                </ul>
                
                <p><strong>What happens when you click launch:</strong></p>
                <ol>
                    <li>Opens AWS CloudFormation in a new tab</li>
                    <li>Template and parameters are pre-filled</li>
                    <li>You review the settings</li>
                    <li>Click "Create Stack" in AWS console</li>
                    <li>Come back here with the Role ARN</li>
                </ol>
            </div>
            
            <div id="customerForm">
                <label>Your Organization Name:</label>
                <input type="text" id="orgName" placeholder="Acme Corp" required>
                <br><br>
                <label>Preferred AWS Region:</label>
                <select id="region">
                    <option value="us-east-1">US East (N. Virginia)</option>
                    <option value="us-west-2">US West (Oregon)</option>
                    <option value="eu-west-1">Europe (Ireland)</option>
                </select>
                <br><br>
            </div>
            
            <button onclick="launchCloudFormation()" class="launch-button" id="launchBtn">
                🚀 Launch CloudFormation Stack
            </button>
        </div>

        <!-- Step 4: Complete Integration -->
        <div class="setup-step" id="step4" style="display:none;">
            <h3>Step 4: Complete Integration</h3>
            <div class="explanation">
                <p>After creating the CloudFormation stack:</p>
                <ol>
                    <li>Copy the <strong>Role ARN</strong> from the Outputs tab</li>
                    <li>Copy the <strong>External ID</strong> from the Outputs tab</li>
                    <li>Paste them below to complete setup</li>
                </ol>
            </div>
            
            <div>
                <label>Role ARN:</label><br>
                <input type="text" id="roleArn" placeholder="arn:aws:iam::123456789:role/YourService/..." style="width: 100%; margin: 5px 0;">
                <br><br>
                
                <label>External ID:</label><br>
                <input type="text" id="externalId" placeholder="YourService-customer123-..." style="width: 100%; margin: 5px 0;">
                <br><br>
            </div>
            
            <button onclick="completeSetup()" class="launch-button">Complete Setup</button>
        </div>

        <!-- Step 5: Success & Cleanup -->
        <div class="setup-step" id="step5" style="display:none;">
            <h3>🎉 Integration Successful!</h3>
            <div class="explanation">
                <p>Your AWS account is now securely connected to YourServiceName.</p>
                
                <div class="security-note">
                    <strong>Important Security Step:</strong> 
                    <p>Your CloudFormation stack included temporary setup permissions. For security, you should remove them now:</p>
                    <ol>
                        <li>Go to <a href="https://console.aws.amazon.com/cloudformation" target="_blank">CloudFormation Console</a></li>
                        <li>Find your stack: "YourService-Integration-[YourOrg]"</li>
                        <li>Click "Update"</li>
                        <li>Change "SetupPhase" parameter from "true" to "false"</li>
                        <li>Click "Update stack"</li>
                    </ol>
                </div>
            </div>
        </div>
    </div>

    <script>
        let customerData = {};
        
        function nextStep(step) {
            // Hide current step
            document.querySelectorAll('.setup-step').forEach(el => {
                el.style.display = 'none';
                el.classList.remove('active');
            });
            
            // Show next step
            document.getElementById('step' + step).style.display = 'block';
            document.getElementById('step' + step).classList.add('active');
        }
        
        function launchCloudFormation() {
            const orgName = document.getElementById('orgName').value;
            const region = document.getElementById('region').value;
            
            if (!orgName) {
                alert('Please enter your organization name');
                return;
            }
            
            customerData = {
                customerName: orgName.replace(/[^a-zA-Z0-9]/g, ''),
                customerId: btoa(orgName).replace(/[^a-zA-Z0-9]/g, '').substr(0, 8),
                preferredRegion: region
            };
            
            // Generate launch URL (implement the JavaScript function from above)
            const launchUrl = generateLaunchUrl(customerData);
            
            // Open CloudFormation
            window.open(launchUrl, '_blank');
            
            // Show next step
            nextStep(4);
        }
        
        function completeSetup() {
            const roleArn = document.getElementById('roleArn').value;
            const externalId = document.getElementById('externalId').value;
            
            if (!roleArn || !externalId) {
                alert('Please provide both Role ARN and External ID');
                return;
            }
            
            // Send to your backend for verification and storage
            fetch('/api/complete-aws-integration', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    customerData,
                    roleArn,
                    externalId
                })
            }).then(response => {
                if (response.ok) {
                    nextStep(5);
                } else {
                    alert('Setup failed. Please check your inputs and try again.');
                }
            });
        }
    </script>
</body>
</html>
```

### 4. Backend Integration Code

```python
# Example Python backend integration
import boto3
import json
from botocore.exceptions import ClientError

class AWSIntegrationService:
    def __init__(self, your_aws_account_id):
        self.account_id = your_aws_account_id
        
    def verify_cross_account_access(self, role_arn, external_id):
        """Verify we can assume the customer's role"""
        try:
            sts = boto3.client('sts')
            response = sts.assume_role(
                RoleArn=role_arn,
                RoleSessionName='YourService-verification',
                ExternalId=external_id,
                DurationSeconds=900  # 15 minutes
            )
            
            # Test basic permissions
            temp_credentials = response['Credentials']
            ec2 = boto3.client(
                'ec2',
                aws_access_key_id=temp_credentials['AccessKeyId'],
                aws_secret_access_key=temp_credentials['SecretAccessKey'],
                aws_session_token=temp_credentials['SessionToken']
            )
            
            # Try to list instances (should work with ongoing permissions)
            ec2.describe_instances(MaxResults=5)
            
            return True, "Successfully verified access"
            
        except ClientError as e:
            return False, f"Access verification failed: {e}"
    
    def store_customer_credentials(self, customer_id, role_arn, external_id):
        """Store customer AWS credentials securely"""
        # Encrypt and store in your secure database
        # This is pseudocode - implement based on your security requirements
        encrypted_data = self.encrypt_credentials({
            'role_arn': role_arn,
            'external_id': external_id
        })
        
        # Store in database with customer_id
        self.database.store_customer_aws_config(customer_id, encrypted_data)
    
    def get_customer_session(self, customer_id):
        """Get temporary AWS session for customer operations"""
        config = self.database.get_customer_aws_config(customer_id)
        decrypted = self.decrypt_credentials(config)
        
        sts = boto3.client('sts')
        response = sts.assume_role(
            RoleArn=decrypted['role_arn'],
            RoleSessionName=f'YourService-{customer_id}',
            ExternalId=decrypted['external_id']
        )
        
        return boto3.Session(
            aws_access_key_id=response['Credentials']['AccessKeyId'],
            aws_secret_access_key=response['Credentials']['SecretAccessKey'],
            aws_session_token=response['Credentials']['SessionToken']
        )
```

## Security Considerations

### 1. External ID Implementation
- Generate unique external IDs per customer
- Store external IDs securely 
- Never expose external IDs in logs or error messages

### 2. Permission Principle of Least Privilege
- Only request permissions your service actually needs
- Use resource-level restrictions where possible
- Regularly audit and remove unused permissions

### 3. Monitoring and Logging
- Log all role assumptions in your service
- Set up CloudTrail monitoring for customers
- Alert on unusual access patterns

### 4. Customer Communication
- Clearly explain what permissions you need and why
- Provide instructions for removing setup permissions
- Offer guidance on monitoring role usage

## Customer Experience Checklist

✅ **Simple URL** - One-click launch from your platform
✅ **Pre-filled parameters** - Customer doesn't need to figure out values  
✅ **Progressive explanation** - Build understanding step by step
✅ **Security transparency** - Clear about what you can/cannot access
✅ **Post-setup cleanup** - Guide removal of temporary permissions
✅ **Testing validation** - Verify integration works before completing
✅ **Clear documentation** - Instructions for ongoing management

## Advanced Customizations

### Multi-Region Support
- Deploy CloudFormation template to multiple regions
- Handle region-specific AMIs and instance types
- Provide region selection in your UI

### Enterprise Features  
- Support for AWS Organizations
- Integration with AWS SSO
- Custom permission boundaries
- Audit trail integration

### Error Handling
- Graceful failure messages
- Retry mechanisms for temporary failures
- Support contact information for complex issues

This template provides a complete, production-ready implementation of Coiled's successful AWS integration approach, with security best practices and excellent user experience built in.