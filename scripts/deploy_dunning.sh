#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    if [ "$2" = "error" ]; then
        echo -e "${RED}✗ $1${NC}"
    elif [ "$2" = "success" ]; then
        echo -e "${GREEN}✓ $1${NC}"
    elif [ "$2" = "warning" ]; then
        echo -e "${YELLOW}⚠ $1${NC}"
    else
        echo -e "→ $1"
    fi
}

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    print_status "AWS CLI is not installed. Please install it first." "error"
    exit 1
fi

# Check if SAM CLI is installed
if ! command -v sam &> /dev/null; then
    print_status "SAM CLI is not installed. Please install it first." "error"
    exit 1
fi

# Get the directory of the script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Parse command line arguments
STAGE=${1:-dev}
REGION=${2:-us-east-1}

if [[ "$STAGE" != "dev" && "$STAGE" != "prod" ]]; then
    print_status "Invalid stage: $STAGE. Must be 'dev' or 'prod'" "error"
    exit 1
fi

print_status "Deploying Dunning Processor Lambda"
print_status "Stage: $STAGE"
print_status "Region: $REGION"
echo ""

# Change to dunning processor directory
cd "$PROJECT_ROOT/apps/dunning-processor"

# Clean previous build
print_status "Cleaning previous build..."
make clean

# Build the Lambda function
print_status "Building Lambda function..."
make build
if [ $? -eq 0 ]; then
    print_status "Build successful" "success"
else
    print_status "Build failed" "error"
    exit 1
fi

# Get parameters based on stage
if [ "$STAGE" = "prod" ]; then
    STACK_NAME="dunning-processor-prod"
    SCHEDULE="rate(1 minute)"
else
    STACK_NAME="dunning-processor-dev"
    SCHEDULE="rate(5 minutes)"
fi

# Get required parameters from environment or AWS SSM
print_status "Fetching deployment parameters..."

# Try to get parameters from SSM
DB_HOST=$(aws ssm get-parameter --name "/${STAGE}/cyphera/db_host" --query 'Parameter.Value' --output text 2>/dev/null || echo "")
RDS_SECRET_ARN=$(aws ssm get-parameter --name "/${STAGE}/cyphera/rds_secret_arn" --query 'Parameter.Value' --output text 2>/dev/null || echo "")
RESEND_API_KEY_ARN=$(aws ssm get-parameter --name "/${STAGE}/cyphera/resend_api_key_arn" --query 'Parameter.Value' --output text 2>/dev/null || echo "")
VPC_SUBNET_IDS=$(aws ssm get-parameter --name "/${STAGE}/cyphera/vpc_subnet_ids" --query 'Parameter.Value' --output text 2>/dev/null || echo "")
VPC_SECURITY_GROUP_IDS=$(aws ssm get-parameter --name "/${STAGE}/cyphera/vpc_security_group_ids" --query 'Parameter.Value' --output text 2>/dev/null || echo "")

# Check if we have all required parameters
if [ -z "$DB_HOST" ] || [ -z "$RDS_SECRET_ARN" ] || [ -z "$RESEND_API_KEY_ARN" ] || [ -z "$VPC_SUBNET_IDS" ] || [ -z "$VPC_SECURITY_GROUP_IDS" ]; then
    print_status "Missing required parameters. Please ensure all SSM parameters are set:" "error"
    echo "  - /${STAGE}/cyphera/db_host"
    echo "  - /${STAGE}/cyphera/rds_secret_arn"
    echo "  - /${STAGE}/cyphera/resend_api_key_arn"
    echo "  - /${STAGE}/cyphera/vpc_subnet_ids"
    echo "  - /${STAGE}/cyphera/vpc_security_group_ids"
    exit 1
fi

print_status "Parameters loaded successfully" "success"

# Deploy using SAM
print_status "Deploying to AWS..."
sam deploy \
    --template-file "$PROJECT_ROOT/infrastructure/aws-sam/template-dunning.yaml" \
    --stack-name "$STACK_NAME" \
    --capabilities CAPABILITY_IAM \
    --region "$REGION" \
    --parameter-overrides \
        Stage="$STAGE" \
        DBHost="$DB_HOST" \
        RDSSecretArn="$RDS_SECRET_ARN" \
        ResendAPIKeyArn="$RESEND_API_KEY_ARN" \
        VpcSubnetIds="$VPC_SUBNET_IDS" \
        VpcSecurityGroupIds="$VPC_SECURITY_GROUP_IDS" \
        ScheduleExpression="$SCHEDULE" \
    --no-confirm-changeset \
    --no-fail-on-empty-changeset

if [ $? -eq 0 ]; then
    print_status "Deployment successful!" "success"
    
    # Get the function name
    FUNCTION_NAME=$(aws cloudformation describe-stacks \
        --stack-name "$STACK_NAME" \
        --query 'Stacks[0].Outputs[?OutputKey==`DunningProcessorFunctionName`].OutputValue' \
        --output text \
        --region "$REGION")
    
    print_status "Function deployed: $FUNCTION_NAME"
    echo ""
    print_status "To test the function manually, run:"
    echo "aws lambda invoke --function-name $FUNCTION_NAME --region $REGION output.json"
    echo ""
    print_status "To view logs, run:"
    echo "aws logs tail /aws/lambda/$FUNCTION_NAME --follow --region $REGION"
else
    print_status "Deployment failed" "error"
    exit 1
fi