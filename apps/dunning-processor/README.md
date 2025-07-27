# Dunning Processor Lambda

This AWS Lambda function processes dunning campaigns for failed subscription payments. It runs on a scheduled basis (every 1 minute in production, every 5 minutes in development) to check for due campaigns and send recovery emails.

## Architecture

The dunning processor:
1. Queries the database for due dunning campaigns
2. Sends email notifications to customers based on the retry schedule
3. Updates campaign status and attempt records
4. Optionally retries payments (when integrated with delegation server)

## Local Development

### Prerequisites
- Go 1.21+
- AWS CLI configured
- Docker (for PostgreSQL)
- Environment variables set in `.env` file

### Running Locally

```bash
# From the dunning-processor directory
make run-local

# Or from the project root
cd apps/dunning-processor
go run cmd/main.go
```

### Testing

```bash
make test
```

## Building for AWS Lambda

```bash
# Build the Lambda binary
make build

# This creates a Linux binary at bin/bootstrap
```

## Deployment

### Using the deployment script:
```bash
# Deploy to dev environment
./scripts/deploy_dunning.sh dev

# Deploy to production
./scripts/deploy_dunning.sh prod
```

### Using GitHub Actions:
The dunning processor is automatically deployed when changes are pushed to:
- `develop` branch → deploys to dev environment
- `main` branch → deploys to production environment

### Manual SAM deployment:
```bash
# Build
sam build --template-file infrastructure/aws-sam/template-dunning.yaml

# Deploy
sam deploy \
  --template-file infrastructure/aws-sam/template-dunning.yaml \
  --stack-name dunning-processor-dev \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides Stage=dev ...
```

## Configuration

### Environment Variables
- `STAGE`: Deployment stage (dev/prod/local)
- `DB_HOST`: Database endpoint
- `DB_NAME`: Database name
- `RDS_SECRET_ARN`: ARN of RDS credentials in Secrets Manager
- `RESEND_API_KEY_ARN`: ARN of Resend API key in Secrets Manager
- `DUNNING_FROM_EMAIL`: From email for dunning notifications
- `DUNNING_FROM_NAME`: From name for dunning notifications

### Schedule
- Production: Every 1 minute
- Development: Every 5 minutes

## Monitoring

### CloudWatch Logs
Logs are available in CloudWatch under:
- `/aws/lambda/dunning-processor-dev` (development)
- `/aws/lambda/dunning-processor-prod` (production)

### CloudWatch Alarms
- **Error Alarm**: Triggers when Lambda execution errors occur
- **Throttle Alarm**: Triggers when Lambda is throttled

### Metrics
The processor logs the following metrics:
- Total campaigns processed
- Successful email sends
- Failed attempts
- Payment retries (when implemented)

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Verify VPC configuration and security groups
   - Check RDS credentials in Secrets Manager
   - Ensure Lambda has VPC access

2. **Email Sending Failures**
   - Verify Resend API key is valid
   - Check email templates exist in database
   - Verify from email domain is verified in Resend

3. **No Campaigns Processing**
   - Check if campaigns exist with status='active'
   - Verify next_attempt_at is in the past
   - Check dunning configurations are active

### Debug Commands

```bash
# View recent logs
aws logs tail /aws/lambda/dunning-processor-dev --follow

# Invoke manually
aws lambda invoke \
  --function-name dunning-processor-dev \
  --payload '{}' \
  response.json

# Check function configuration
aws lambda get-function --function-name dunning-processor-dev
```

## Integration with Other Services

- **Database**: Reads/writes dunning campaigns and attempts
- **Email Service**: Sends notifications via Resend API
- **Dunning Service**: Core business logic for campaign management
- **Payment Service**: Will integrate for payment retries (TODO)