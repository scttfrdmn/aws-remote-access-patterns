# Production Deployment Guide

This guide covers deploying AWS Remote Access Patterns in production environments with security, scalability, and reliability best practices.

## 🏗️ Architecture Overview

### Production Architecture Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Production Environment                        │
├─────────────────────────────────────────────────────────────────┤
│  Load Balancer (ALB)                                           │
│      │                                                          │
│      ├─── ECS/Fargate Service ──── DynamoDB                    │
│      │    (Multiple instances)     (Encrypted)                 │
│      │                                                          │
│      ├─── Lambda Functions ────── S3 Bucket                    │
│      │    (Auto-scaling)          (Templates)                  │
│      │                                                          │
│      └─── CloudFront ──────────── WAF                         │
│           (Static assets)         (Security)                   │
│                                                                 │
│  Monitoring: CloudWatch + X-Ray + AWS Config                  │
│  Security: KMS + Secrets Manager + IAM                        │
│  Networking: VPC + Private Subnets + NACLs                    │
└─────────────────────────────────────────────────────────────────┘
```

## 🚀 Deployment Options

### Option 1: Serverless Deployment (Recommended)

**Best for**: Small to medium services, cost optimization, automatic scaling

```yaml
# serverless.yml
service: aws-remote-access-patterns

provider:
  name: aws
  runtime: provided.al2
  architecture: arm64
  stage: ${opt:stage, 'dev'}
  region: ${opt:region, 'us-east-1'}
  
functions:
  api:
    handler: main
    events:
      - httpApi: '*'
    environment:
      DYNAMODB_TABLE: !Ref CredentialsTable
      S3_TEMPLATE_BUCKET: !Ref TemplateBucket
    
resources:
  Resources:
    CredentialsTable:
      Type: AWS::DynamoDB::Table
      Properties:
        BillingMode: PAY_PER_REQUEST
        PointInTimeRecoverySpecification:
          PointInTimeRecoveryEnabled: true
        SSESpecification:
          SSEEnabled: true
        StreamSpecification:
          StreamViewType: NEW_AND_OLD_IMAGES
```

### Option 2: Container Deployment (ECS/Fargate)

**Best for**: Medium to large services, consistent workloads, hybrid environments

```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  app:
    build: .
    environment:
      - AWS_REGION=${AWS_REGION}
      - DYNAMODB_TABLE=${DYNAMODB_TABLE}
      - S3_TEMPLATE_BUCKET=${S3_TEMPLATE_BUCKET}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Option 3: Kubernetes Deployment

**Best for**: Large scale, multi-cloud, complex orchestration needs

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aws-remote-access-patterns
spec:
  replicas: 3
  selector:
    matchLabels:
      app: aws-remote-access-patterns
  template:
    metadata:
      labels:
        app: aws-remote-access-patterns
    spec:
      containers:
      - name: app
        image: your-registry/aws-remote-access-patterns:latest
        ports:
        - containerPort: 8080
        env:
        - name: AWS_REGION
          value: "us-east-1"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## 🔐 Security Configuration

### 1. Infrastructure Security

#### VPC Configuration
```yaml
# CloudFormation template for VPC
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
      EnableDnsHostnames: true
      EnableDnsSupport: true
      
  PrivateSubnet1:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: 10.0.1.0/24
      AvailabilityZone: !Select [0, !GetAZs '']
      
  PrivateSubnet2:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      CidrBlock: 10.0.2.0/24
      AvailabilityZone: !Select [1, !GetAZs '']
```

#### WAF Configuration
```yaml
WebACL:
  Type: AWS::WAFv2::WebACL
  Properties:
    Scope: CLOUDFRONT
    DefaultAction:
      Allow: {}
    Rules:
      - Name: AWSManagedRulesCommonRuleSet
        Priority: 1
        OverrideAction:
          None: {}
        Statement:
          ManagedRuleGroupStatement:
            VendorName: AWS
            Name: AWSManagedRulesCommonRuleSet
```

### 2. Application Security

#### Environment Variables
```bash
# Use AWS Systems Manager Parameter Store
AWS_REGION=us-east-1
DYNAMODB_TABLE=/prod/aws-patterns/dynamodb-table
S3_TEMPLATE_BUCKET=/prod/aws-patterns/template-bucket
KMS_KEY_ID=/prod/aws-patterns/kms-key
JWT_SECRET=/prod/aws-patterns/jwt-secret
```

#### Encryption Configuration
```go
// Production storage with encryption
storage, err := NewDynamoDBStorage(ctx, &DynamoDBConfig{
    TableName: os.Getenv("DYNAMODB_TABLE"),
    KMSKeyID:  os.Getenv("KMS_KEY_ID"),
    Encrypted: true,
})
```

## 📊 Monitoring and Observability

### 1. Metrics Collection

#### Custom Metrics
```go
// CloudWatch custom metrics
import (
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
    "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func recordSetupCompletion(customerID string, success bool) {
    metric := &types.MetricDatum{
        MetricName: aws.String("SetupCompletion"),
        Dimensions: []types.Dimension{
            {
                Name:  aws.String("CustomerID"),
                Value: aws.String(customerID),
            },
            {
                Name:  aws.String("Success"),
                Value: aws.String(fmt.Sprintf("%t", success)),
            },
        },
        Value:     aws.Float64(1),
        Unit:      types.StandardUnitCount,
        Timestamp: aws.Time(time.Now()),
    }
    
    cloudwatchClient.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
        Namespace:  aws.String("AWSRemoteAccessPatterns"),
        MetricData: []types.MetricDatum{*metric},
    })
}
```

#### Dashboard Configuration
```json
{
  "widgets": [
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["AWSRemoteAccessPatterns", "SetupCompletion", "Success", "true"],
          ["AWSRemoteAccessPatterns", "SetupCompletion", "Success", "false"]
        ],
        "period": 300,
        "stat": "Sum",
        "region": "us-east-1",
        "title": "Setup Success Rate"
      }
    }
  ]
}
```

### 2. Logging Strategy

#### Structured Logging
```go
import (
    "log/slog"
    "os"
)

// Production logger configuration
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
    AddSource: true,
}))

// Usage in application
logger.Info("Customer setup initiated",
    slog.String("customer_id", customerID),
    slog.String("external_id", externalID),
    slog.String("role_arn", roleARN),
)
```

#### Log Aggregation
```yaml
# CloudWatch Logs configuration
LogGroup:
  Type: AWS::Logs::LogGroup
  Properties:
    LogGroupName: /aws/lambda/aws-remote-access-patterns
    RetentionInDays: 30
    
LogStream:
  Type: AWS::Logs::LogStream
  Properties:
    LogGroupName: !Ref LogGroup
```

## 🎚️ Configuration Management

### 1. Environment-Specific Configurations

#### Development Environment
```yaml
# config/development.yaml
database:
  type: memory
  
aws:
  region: us-east-1
  
logging:
  level: debug
  
security:
  cors_origins: ["http://localhost:3000"]
```

#### Production Environment
```yaml
# config/production.yaml
database:
  type: dynamodb
  table_name: ${DYNAMODB_TABLE}
  encryption_enabled: true
  
aws:
  region: ${AWS_REGION}
  
logging:
  level: info
  format: json
  
security:
  cors_origins: ["https://yourdomain.com"]
  rate_limiting: true
  
monitoring:
  metrics_enabled: true
  tracing_enabled: true
```

### 2. Feature Flags

```go
// Feature flag configuration
type FeatureFlags struct {
    EnableWebUI          bool `json:"enable_web_ui"`
    EnableMetrics        bool `json:"enable_metrics"`
    EnableRateLimiting   bool `json:"enable_rate_limiting"`
    MaxCustomersPerHour  int  `json:"max_customers_per_hour"`
}

func loadFeatureFlags(ctx context.Context) (*FeatureFlags, error) {
    // Load from AWS App Config or Parameter Store
    value, err := parameterStore.GetParameter(ctx, "/prod/aws-patterns/feature-flags")
    if err != nil {
        return defaultFeatureFlags(), nil
    }
    
    var flags FeatureFlags
    if err := json.Unmarshal([]byte(value), &flags); err != nil {
        return defaultFeatureFlags(), nil
    }
    
    return &flags, nil
}
```

## 🧪 Testing Strategy

### 1. Integration Testing

```go
// Integration test setup
func TestProductionIntegration(t *testing.T) {
    ctx := context.Background()
    
    // Use test AWS account and resources
    config := crossaccount.SimpleConfig(
        "TestService",
        "123456789012",
        "test-templates-bucket",
    )
    
    client, err := crossaccount.New(config)
    require.NoError(t, err)
    
    // Test complete flow
    setupResp, err := client.GenerateSetupLink("test-customer", "Test Customer")
    require.NoError(t, err)
    require.NotEmpty(t, setupResp.LaunchURL)
    
    // Test template generation
    template, err := client.GenerateCloudFormationTemplate()
    require.NoError(t, err)
    require.Contains(t, template, "AWS::IAM::Role")
}
```

### 2. Load Testing

```go
// Load testing with proper AWS SDK rate limiting
func TestLoadPerformance(t *testing.T) {
    const (
        concurrentUsers = 100
        requestsPerUser = 50
        testDuration    = 5 * time.Minute
    )
    
    var wg sync.WaitGroup
    results := make(chan time.Duration, concurrentUsers*requestsPerUser)
    
    for i := 0; i < concurrentUsers; i++ {
        wg.Add(1)
        go func(userID int) {
            defer wg.Done()
            
            for j := 0; j < requestsPerUser; j++ {
                start := time.Now()
                
                // Simulate customer setup
                customerID := fmt.Sprintf("load-test-%d-%d", userID, j)
                _, err := client.GenerateSetupLink(customerID, "Load Test Customer")
                
                duration := time.Since(start)
                results <- duration
                
                if err != nil {
                    t.Errorf("Request failed: %v", err)
                }
                
                // Rate limiting
                time.Sleep(10 * time.Millisecond)
            }
        }(i)
    }
    
    wg.Wait()
    close(results)
    
    // Analyze results
    var totalDuration time.Duration
    count := 0
    for duration := range results {
        totalDuration += duration
        count++
    }
    
    avgDuration := totalDuration / time.Duration(count)
    t.Logf("Average response time: %v", avgDuration)
    
    if avgDuration > 500*time.Millisecond {
        t.Errorf("Average response time too high: %v", avgDuration)
    }
}
```

## 📈 Scaling Considerations

### 1. Auto Scaling Configuration

#### Lambda Auto Scaling
```yaml
Functions:
  api:
    reservedConcurrency: 100
    provisionedConcurrency: 10
    timeout: 30
    memorySize: 1024
```

#### ECS Auto Scaling
```yaml
AutoScalingGroup:
  Type: AWS::AutoScaling::AutoScalingGroup
  Properties:
    MinSize: 2
    MaxSize: 20
    DesiredCapacity: 4
    TargetGroupARNs:
      - !Ref LoadBalancerTargetGroup
    HealthCheckType: ELB
    HealthCheckGracePeriod: 300
```

### 2. Database Scaling

#### DynamoDB Auto Scaling
```yaml
ReadCapacityScalableTarget:
  Type: AWS::ApplicationAutoScaling::ScalableTarget
  Properties:
    MaxCapacity: 4000
    MinCapacity: 5
    ResourceId: !Sub table/${CredentialsTable}
    RoleARN: !Sub arn:aws:iam::${AWS::AccountId}:role/aws-dynamodb-auto-scaling-role
    ScalableDimension: dynamodb:table:ReadCapacityUnits
    ServiceNamespace: dynamodb
```

## 🛡️ Disaster Recovery

### 1. Backup Strategy

```yaml
# Automated backups
BackupPlan:
  Type: AWS::Backup::BackupPlan
  Properties:
    BackupPlan:
      BackupPlanName: aws-remote-access-patterns-backup
      BackupPlanRule:
        - RuleName: DailyBackups
          TargetBackupVault: default
          ScheduleExpression: cron(0 2 * * ? *)
          Lifecycle:
            DeleteAfterDays: 30
```

### 2. Multi-Region Setup

```go
// Multi-region client configuration
type MultiRegionClient struct {
    primaryRegion   string
    secondaryRegion string
    clients         map[string]*crossaccount.Client
}

func (m *MultiRegionClient) GenerateSetupLink(customerID, customerName string) (*crossaccount.SetupResponse, error) {
    // Try primary region first
    if client, exists := m.clients[m.primaryRegion]; exists {
        if resp, err := client.GenerateSetupLink(customerID, customerName); err == nil {
            return resp, nil
        }
    }
    
    // Fallback to secondary region
    if client, exists := m.clients[m.secondaryRegion]; exists {
        return client.GenerateSetupLink(customerID, customerName)
    }
    
    return nil, fmt.Errorf("all regions unavailable")
}
```

## 🚀 Deployment Scripts

### 1. Automated Deployment

```bash
#!/bin/bash
# scripts/deploy-production.sh

set -euo pipefail

STAGE=${1:-production}
REGION=${2:-us-east-1}

echo "🚀 Deploying AWS Remote Access Patterns to ${STAGE}..."

# Build application
echo "📦 Building application..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main ./cmd/server

# Run tests
echo "🧪 Running tests..."
go test ./... -race -coverprofile=coverage.out

# Deploy infrastructure
echo "☁️ Deploying infrastructure..."
aws cloudformation deploy \
    --template-file infrastructure/production.yaml \
    --stack-name aws-remote-access-patterns-${STAGE} \
    --parameter-overrides Stage=${STAGE} \
    --capabilities CAPABILITY_IAM \
    --region ${REGION}

# Deploy application
echo "📱 Deploying application..."
serverless deploy --stage ${STAGE} --region ${REGION}

echo "✅ Deployment complete!"
```

### 2. Rollback Script

```bash
#!/bin/bash
# scripts/rollback.sh

set -euo pipefail

STAGE=${1:-production}
VERSION=${2}

if [ -z "${VERSION}" ]; then
    echo "Usage: $0 <stage> <version>"
    exit 1
fi

echo "🔄 Rolling back to version ${VERSION}..."

# Rollback application
serverless rollback --stage ${STAGE} --timestamp ${VERSION}

echo "✅ Rollback complete!"
```

## 📋 Production Checklist

### Pre-Deployment
- [ ] Security review completed
- [ ] Load testing passed
- [ ] Integration tests passing
- [ ] Infrastructure code reviewed
- [ ] Secrets configured in AWS Secrets Manager
- [ ] Monitoring dashboards created
- [ ] Rollback procedure tested

### Post-Deployment  
- [ ] Health checks passing
- [ ] Metrics being collected
- [ ] Logs being aggregated
- [ ] SSL certificates valid
- [ ] DNS routing working
- [ ] Error rates within SLA
- [ ] Documentation updated

### Ongoing Operations
- [ ] Weekly security scans
- [ ] Monthly cost reviews
- [ ] Quarterly disaster recovery tests
- [ ] Annual security audits
- [ ] Regular dependency updates
- [ ] Performance optimization reviews

---

This deployment guide provides a comprehensive foundation for running AWS Remote Access Patterns in production with enterprise-grade security, monitoring, and reliability.