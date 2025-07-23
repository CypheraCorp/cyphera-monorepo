# Deployment Guide

> **Navigation:** [← API Reference](api-reference.md) | [↑ README](../README.md) | [Troubleshooting →](troubleshooting.md)

Complete guide for deploying the Cyphera platform to production environments.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Environment Configuration](#environment-configuration)
- [Database Deployment](#database-deployment)
- [Service Deployment](#service-deployment)
- [Monitoring & Observability](#monitoring--observability)
- [Security Considerations](#security-considerations)
- [Scaling & Performance](#scaling--performance)

## Overview

The Cyphera platform supports multiple deployment strategies:

- **AWS Lambda + ECS** (Recommended for production)
- **Docker Compose** (Development and small deployments)
- **Kubernetes** (Large scale deployments)
- **Hybrid** (Mix of serverless and containerized services)

### Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        AWS Cloud                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │     Web     │  │    Main     │  │   Subscription      │ │
│  │   App CDN   │  │ API Lambda  │  │ Processor Lambda    │ │
│  │ (CloudFront)│  │    (Go)     │  │      (Go)           │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│         │                │                     │            │
│         │                └─────────┬───────────┘            │
│  ┌─────────────┐                  │                        │
│  │    ALB      │                  │                        │
│  │(Load Balancer)                 │                        │
│  └─────────────┘                  │                        │
│         │                         │                        │
│  ┌─────────────┐          ┌───────▼────────┐              │
│  │    ECS      │          │   PostgreSQL   │              │
│  │ Delegation  │          │      RDS       │              │
│  │   Server    │          │   (Database)   │              │
│  └─────────────┘          └────────────────┘              │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

### AWS Account Setup
- AWS CLI configured with appropriate permissions
- IAM roles for Lambda execution and ECS tasks
- VPC with public and private subnets
- Security groups configured for service communication

### Domain & SSL
- Domain name registered and DNS configured
- SSL certificates via AWS Certificate Manager
- CloudFront distribution for web app

### External Services
- **Web3Auth** application configured
- **Circle API** keys and configuration
- **RPC Providers** (Alchemy, Infura, etc.)
- **Monitoring** (CloudWatch, Datadog, etc.)

## Environment Configuration

### Secrets Management

#### AWS Secrets Manager
Store sensitive configuration in AWS Secrets Manager:

```bash
# Database credentials
aws secretsmanager create-secret \
  --name "/cyphera/prod/database" \
  --description "Production database credentials" \
  --secret-string '{
    "host": "cyphera-prod.cluster-xyz.us-east-1.rds.amazonaws.com",
    "port": "5432",
    "username": "cyphera_api",
    "password": "secure_password_here",
    "database": "cyphera_prod"
  }'

# API keys and tokens
aws secretsmanager create-secret \
  --name "/cyphera/prod/api-keys" \
  --secret-string '{
    "web3auth_client_id": "your_web3auth_client_id",
    "web3auth_client_secret": "your_web3auth_client_secret",
    "circle_api_key": "your_circle_api_key",
    "encryption_secret": "your_encryption_secret"
  }'

# Blockchain RPC URLs
aws secretsmanager create-secret \
  --name "/cyphera/prod/rpc-endpoints" \
  --secret-string '{
    "ethereum_rpc_url": "https://eth-mainnet.g.alchemy.com/v2/your_key",
    "polygon_rpc_url": "https://polygon-mainnet.g.alchemy.com/v2/your_key",
    "arbitrum_rpc_url": "https://arb-mainnet.g.alchemy.com/v2/your_key"
  }'
```

#### Environment Variables
Non-sensitive configuration via environment variables:

```bash
# Application configuration
NODE_ENV=production
LOG_LEVEL=info
CORS_ALLOWED_ORIGINS=https://app.cyphera.com,https://api.cyphera.com

# Service configuration
DELEGATION_GRPC_ADDR=delegation-server.internal:50051
SUBSCRIPTION_PROCESSOR_INTERVAL=300s
MAX_RETRY_ATTEMPTS=3
```

## Database Deployment

### RDS PostgreSQL Setup

#### Create RDS Instance
```bash
# Create DB subnet group
aws rds create-db-subnet-group \
  --db-subnet-group-name cyphera-prod-subnet-group \
  --db-subnet-group-description "Cyphera production subnet group" \
  --subnet-ids subnet-12345 subnet-67890

# Create RDS instance
aws rds create-db-instance \
  --db-instance-identifier cyphera-prod \
  --db-instance-class db.r5.large \
  --engine postgres \
  --engine-version 14.9 \
  --master-username cyphera_admin \
  --master-user-password "$(aws secretsmanager get-random-password --password-length 32 --exclude-characters "\"'@/\\" --output text --query Password)" \
  --allocated-storage 100 \
  --storage-type gp2 \
  --storage-encrypted \
  --vpc-security-group-ids sg-12345678 \
  --db-subnet-group-name cyphera-prod-subnet-group \
  --backup-retention-period 30 \
  --multi-az \
  --publicly-accessible false
```

#### Database Migration
```bash
# Create migration Lambda function
cat > migrate.js << 'EOF'
const { Client } = require('pg');

exports.handler = async (event) => {
  const client = new Client({
    connectionString: process.env.DATABASE_URL
  });
  
  await client.connect();
  
  // Read schema from S3 or include in deployment package
  const schema = require('./schema.sql');
  await client.query(schema);
  
  await client.end();
  
  return { statusCode: 200, body: 'Migration completed' };
};
EOF

# Deploy and run migration
aws lambda create-function \
  --function-name cyphera-db-migrate \
  --runtime nodejs18.x \
  --role arn:aws:iam::account:role/lambda-execution-role \
  --handler migrate.handler \
  --zip-file fileb://migrate.zip

aws lambda invoke \
  --function-name cyphera-db-migrate \
  --payload '{}' \
  response.json
```

### Database Monitoring
```bash
# Enable Performance Insights
aws rds modify-db-instance \
  --db-instance-identifier cyphera-prod \
  --enable-performance-insights \
  --performance-insights-retention-period 7

# Create CloudWatch alarms
aws cloudwatch put-metric-alarm \
  --alarm-name "RDS-CPU-High" \
  --alarm-description "RDS CPU usage is high" \
  --metric-name CPUUtilization \
  --namespace AWS/RDS \
  --statistic Average \
  --period 300 \
  --evaluation-periods 2 \
  --threshold 80 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=DBInstanceIdentifier,Value=cyphera-prod
```

## Service Deployment

### Main API (AWS Lambda)

#### Build and Package
```bash
# Build for Lambda
cd apps/api
GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main/main.go
zip lambda-deployment.zip bootstrap

# Upload to S3
aws s3 cp lambda-deployment.zip s3://cyphera-deployments/api/
```

#### Deploy Lambda Function
```bash
# Create Lambda function
aws lambda create-function \
  --function-name cyphera-api-prod \
  --runtime provided.al2 \
  --role arn:aws:iam::account:role/cyphera-lambda-role \
  --handler bootstrap \
  --code S3Bucket=cyphera-deployments,S3Key=api/lambda-deployment.zip \
  --timeout 30 \
  --memory-size 512 \
  --environment Variables='{
    "NODE_ENV":"production",
    "LOG_LEVEL":"info",
    "SECRETS_MANAGER_REGION":"us-east-1"
  }'

# Create API Gateway
aws apigatewayv2 create-api \
  --name cyphera-api-prod \
  --protocol-type HTTP \
  --target arn:aws:lambda:region:account:function:cyphera-api-prod
```

#### Auto-scaling Configuration
```bash
# Configure reserved concurrency
aws lambda put-reserved-concurrency \
  --function-name cyphera-api-prod \
  --reserved-concurrent-executions 100

# Enable provisioned concurrency for consistent performance
aws lambda put-provisioned-concurrency-config \
  --function-name cyphera-api-prod \
  --qualifier '$LATEST' \
  --provisioned-concurrency-config '{"ProvisionedConcurrencyConfig": {"TotalProvisionedConcurrency": 10}}'
```

### Delegation Server (ECS)

#### Container Build
```bash
# Build Docker image
cd apps/delegation-server
docker build -t cyphera/delegation-server:latest .

# Push to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin account.dkr.ecr.us-east-1.amazonaws.com
docker tag cyphera/delegation-server:latest account.dkr.ecr.us-east-1.amazonaws.com/cyphera/delegation-server:latest
docker push account.dkr.ecr.us-east-1.amazonaws.com/cyphera/delegation-server:latest
```

#### ECS Service Definition
```json
{
  "family": "cyphera-delegation-server",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/cyphera-task-role",
  "containerDefinitions": [
    {
      "name": "delegation-server",
      "image": "account.dkr.ecr.us-east-1.amazonaws.com/cyphera/delegation-server:latest",
      "portMappings": [
        {
          "containerPort": 50051,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "NODE_ENV",
          "value": "production"
        },
        {
          "name": "GRPC_PORT",
          "value": "50051"
        }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:account:secret:/cyphera/prod/database"
        },
        {
          "name": "DELEGATION_PRIVATE_KEY",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:account:secret:/cyphera/prod/delegation-key"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/cyphera-delegation-server",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

#### Deploy ECS Service
```bash
# Register task definition
aws ecs register-task-definition --cli-input-json file://task-definition.json

# Create ECS service
aws ecs create-service \
  --cluster cyphera-prod \
  --service-name delegation-server \
  --task-definition cyphera-delegation-server:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration '{
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345", "subnet-67890"],
      "securityGroups": ["sg-12345678"],
      "assignPublicIp": "DISABLED"
    }
  }' \
  --load-balancers '[
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:account:targetgroup/delegation-server/1234567890123456",
      "containerName": "delegation-server",
      "containerPort": 50051
    }
  ]'
```

### Subscription Processor (Lambda)

#### Scheduled Lambda Deployment
```bash
# Build and deploy
cd apps/subscription-processor
GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go
zip subscription-processor.zip bootstrap

# Create Lambda function
aws lambda create-function \
  --function-name cyphera-subscription-processor \
  --runtime provided.al2 \
  --role arn:aws:iam::account:role/cyphera-lambda-role \
  --handler bootstrap \
  --code S3Bucket=cyphera-deployments,S3Key=subscription-processor.zip \
  --timeout 300 \
  --memory-size 512

# Schedule with EventBridge
aws events put-rule \
  --name cyphera-subscription-processor-schedule \
  --schedule-expression "rate(5 minutes)"

aws lambda add-permission \
  --function-name cyphera-subscription-processor \
  --statement-id allow-eventbridge \
  --action lambda:InvokeFunction \
  --principal events.amazonaws.com \
  --source-arn arn:aws:events:region:account:rule/cyphera-subscription-processor-schedule

aws events put-targets \
  --rule cyphera-subscription-processor-schedule \
  --targets "Id"="1","Arn"="arn:aws:lambda:region:account:function:cyphera-subscription-processor"
```

### Web Application (CloudFront + S3)

#### Build and Deploy
```bash
# Build Next.js application
cd apps/web-app
npm run build

# Deploy to S3
aws s3 sync .next/static s3://cyphera-web-assets/static/
aws s3 sync public s3://cyphera-web-assets/public/

# Update CloudFront distribution
aws cloudfront create-invalidation \
  --distribution-id E1234567890123 \
  --paths "/*"
```

#### CloudFront Configuration
```json
{
  "DistributionConfig": {
    "CallerReference": "cyphera-web-prod",
    "Origins": [
      {
        "Id": "cyphera-web-origin",
        "DomainName": "cyphera-web-assets.s3.amazonaws.com",
        "S3OriginConfig": {
          "OriginAccessIdentity": "origin-access-identity/cloudfront/E1234567890123"
        }
      }
    ],
    "DefaultCacheBehavior": {
      "TargetOriginId": "cyphera-web-origin",
      "ViewerProtocolPolicy": "redirect-to-https",
      "Compress": true,
      "CachePolicyId": "caching-optimized"
    },
    "Aliases": ["app.cyphera.com"],
    "ViewerCertificate": {
      "AcmCertificateArn": "arn:aws:acm:us-east-1:account:certificate/12345678-1234-1234-1234-123456789012",
      "SslSupportMethod": "sni-only"
    }
  }
}
```

## Monitoring & Observability

### CloudWatch Setup

#### Log Groups
```bash
# Create log groups
aws logs create-log-group --log-group-name /aws/lambda/cyphera-api-prod
aws logs create-log-group --log-group-name /ecs/cyphera-delegation-server
aws logs create-log-group --log-group-name /aws/lambda/cyphera-subscription-processor

# Set retention policies
aws logs put-retention-policy \
  --log-group-name /aws/lambda/cyphera-api-prod \
  --retention-in-days 30
```

#### Custom Metrics
```bash
# Create custom metric filters
aws logs put-metric-filter \
  --log-group-name /aws/lambda/cyphera-api-prod \
  --filter-name ErrorCount \
  --filter-pattern "ERROR" \
  --metric-transformations \
    metricName=ApiErrors,metricNamespace=Cyphera/API,metricValue=1

# Create alarms
aws cloudwatch put-metric-alarm \
  --alarm-name "API-Error-Rate-High" \
  --alarm-description "API error rate is high" \
  --metric-name ApiErrors \
  --namespace Cyphera/API \
  --statistic Sum \
  --period 300 \
  --evaluation-periods 2 \
  --threshold 10 \
  --comparison-operator GreaterThanThreshold
```

### Application Performance Monitoring

#### X-Ray Tracing
```bash
# Enable X-Ray for Lambda functions
aws lambda update-function-configuration \
  --function-name cyphera-api-prod \
  --tracing-config Mode=Active

# Enable X-Ray for ECS
# Add to task definition:
{
  "name": "xray-daemon",
  "image": "amazon/aws-xray-daemon:latest",
  "cpu": 32,
  "memory": 256,
  "portMappings": [
    {
      "containerPort": 2000,
      "protocol": "udp"
    }
  ]
}
```

### Health Checks

#### API Health Monitoring
```bash
# Create health check alarm
aws cloudwatch put-metric-alarm \
  --alarm-name "API-Health-Check-Failed" \
  --alarm-description "API health check is failing" \
  --metric-name HealthCheck \
  --namespace AWS/ApplicationELB \
  --statistic Average \
  --period 60 \
  --evaluation-periods 3 \
  --threshold 1 \
  --comparison-operator LessThanThreshold \
  --dimensions Name=LoadBalancer,Value=app/cyphera-api-alb/1234567890123456
```

## Security Considerations

### Network Security

#### VPC Configuration
```bash
# Create VPC with public and private subnets
aws ec2 create-vpc --cidr-block 10.0.0.0/16

# Create security groups
aws ec2 create-security-group \
  --group-name cyphera-api-sg \
  --description "Security group for Cyphera API" \
  --vpc-id vpc-12345678

# Configure security group rules
aws ec2 authorize-security-group-ingress \
  --group-id sg-12345678 \
  --protocol tcp \
  --port 443 \
  --cidr 0.0.0.0/0
```

#### WAF Configuration
```bash
# Create WAF Web ACL
aws wafv2 create-web-acl \
  --name cyphera-api-waf \
  --scope REGIONAL \
  --default-action Allow={} \
  --rules '[
    {
      "Name": "AWSManagedRulesCommonRuleSet",
      "Priority": 1,
      "OverrideAction": {"None": {}},
      "Statement": {
        "ManagedRuleGroupStatement": {
          "VendorName": "AWS",
          "Name": "AWSManagedRulesCommonRuleSet"
        }
      },
      "VisibilityConfig": {
        "SampledRequestsEnabled": true,
        "CloudWatchMetricsEnabled": true,
        "MetricName": "CommonRuleSetMetric"
      }
    }
  ]'
```

### IAM Policies

#### Lambda Execution Role
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": "arn:aws:secretsmanager:*:*:secret:/cyphera/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBInstances"
      ],
      "Resource": "*"
    }
  ]
}
```

### Secret Rotation
```bash
# Enable automatic secret rotation
aws secretsmanager update-secret \
  --secret-id /cyphera/prod/database \
  --rotation-lambda-arn arn:aws:lambda:region:account:function:SecretsManagerRDSPostgreSQLRotationSingleUser \
  --rotation-rules AutomaticallyAfterDays=30
```

## Scaling & Performance

### Auto Scaling Configuration

#### Lambda Concurrency
```bash
# Set reserved concurrency
aws lambda put-reserved-concurrency \
  --function-name cyphera-api-prod \
  --reserved-concurrent-executions 100

# Configure provisioned concurrency for predictable performance
aws application-autoscaling register-scalable-target \
  --service-namespace lambda \
  --resource-id function:cyphera-api-prod:provisioned \
  --scalable-dimension lambda:function:ProvisionedConcurrency \
  --min-capacity 10 \
  --max-capacity 100
```

#### ECS Auto Scaling
```bash
# Register ECS service as scalable target
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --resource-id service/cyphera-prod/delegation-server \
  --scalable-dimension ecs:service:DesiredCount \
  --min-capacity 2 \
  --max-capacity 10

# Create scaling policy
aws application-autoscaling put-scaling-policy \
  --policy-name cpu-scaling \
  --service-namespace ecs \
  --resource-id service/cyphera-prod/delegation-server \
  --scalable-dimension ecs:service:DesiredCount \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration '{
    "TargetValue": 70.0,
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ECSServiceAverageCPUUtilization"
    }
  }'
```

### Database Scaling

#### Read Replicas
```bash
# Create read replica
aws rds create-db-instance-read-replica \
  --db-instance-identifier cyphera-prod-replica \
  --source-db-instance-identifier cyphera-prod \
  --db-instance-class db.r5.large
```

#### Connection Pooling
```bash
# Create RDS Proxy for connection pooling
aws rds create-db-proxy \
  --db-proxy-name cyphera-prod-proxy \
  --engine-family POSTGRESQL \
  --auth '[
    {
      "AuthScheme": "SECRETS",
      "SecretArn": "arn:aws:secretsmanager:region:account:secret:/cyphera/prod/database",
      "IAMAuth": "DISABLED"
    }
  ]' \
  --role-arn arn:aws:iam::account:role/rds-proxy-role \
  --vpc-subnet-ids subnet-12345 subnet-67890 \
  --vpc-security-group-ids sg-12345678
```

## Disaster Recovery

### Backup Strategy

#### Database Backups
```bash
# Automated backups are enabled by default
# Create manual snapshot
aws rds create-db-snapshot \
  --db-instance-identifier cyphera-prod \
  --db-snapshot-identifier cyphera-prod-manual-snapshot-$(date +%Y%m%d)

# Cross-region backup
aws rds copy-db-snapshot \
  --source-db-snapshot-identifier arn:aws:rds:us-east-1:account:snapshot:cyphera-prod-snapshot \
  --target-db-snapshot-identifier cyphera-prod-snapshot-backup \
  --source-region us-east-1 \
  --target-region us-west-2
```

#### Application Code Backup
```bash
# S3 versioning for deployment artifacts
aws s3api put-bucket-versioning \
  --bucket cyphera-deployments \
  --versioning-configuration Status=Enabled

# Cross-region replication
aws s3api put-bucket-replication \
  --bucket cyphera-deployments \
  --replication-configuration file://replication-config.json
```

### Recovery Procedures

#### Database Recovery
```bash
# Restore from snapshot
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier cyphera-prod-restored \
  --db-snapshot-identifier cyphera-prod-snapshot

# Point-in-time recovery
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier cyphera-prod \
  --target-db-instance-identifier cyphera-prod-pitr \
  --restore-time 2024-01-01T12:00:00Z
```

---

## Related Documentation

- **[Architecture Guide](architecture.md)** - System design overview
- **[API Reference](api-reference.md)** - API endpoints and usage
- **[Troubleshooting](troubleshooting.md)** - Common deployment issues
- **[Security Guide](security.md)** - Security best practices

## Support

- **AWS Documentation** - Service-specific deployment guides
- **[GitHub Issues](https://github.com/your-org/cyphera-api/issues)** - Deployment problems
- **Infrastructure as Code** - Terraform/CloudFormation templates available

---

*Last updated: $(date '+%Y-%m-%d')*
*For automated deployment scripts, see `/infrastructure` directory*