#!/bin/bash
set -eo pipefail

# ===============================================
# Test Script for Webhook Deployment
# ===============================================

echo "🧪 Testing webhook deployment script..."

# Check if the deployment script exists
if [ ! -f "scripts/deploy_webhook_lambdas.sh" ]; then
    echo "❌ Deployment script not found: scripts/deploy_webhook_lambdas.sh"
    exit 1
fi

# Check if the script is executable
if [ ! -x "scripts/deploy_webhook_lambdas.sh" ]; then
    echo "❌ Deployment script is not executable"
    exit 1
fi

# Test environment variable validation
echo "🔍 Testing environment variable validation..."

# Test missing STAGE
unset STAGE AWS_REGION SAM_DEPLOYMENT_BUCKET LAMBDA_SG_ID PRIVATE_SUBNET_1_ID PRIVATE_SUBNET_2_ID
if ./scripts/deploy_webhook_lambdas.sh 2>/dev/null; then
    echo "❌ Script should fail with missing environment variables"
    exit 1
else
    echo "✅ Script correctly fails with missing environment variables"
fi

# Test with valid environment variables (dry run)
export STAGE="test"
export AWS_REGION="us-east-1"
export SAM_DEPLOYMENT_BUCKET="test-bucket"
export LAMBDA_SG_ID="sg-12345"
export PRIVATE_SUBNET_1_ID="subnet-12345"
export PRIVATE_SUBNET_2_ID="subnet-67890"

echo "🔍 Testing parameter name construction..."

# Check if the script constructs the correct parameter names
EXPECTED_PARAMS=(
    "/cyphera/rds-secret-arn-test"
    "/cyphera/rds-endpoint-test"
    "/cyphera/webhook-sqs-queue-url-test"
    "/cyphera/webhook-dlq-queue-url-test"
    "/cyphera/payment-sync-encryption-key-arn-test"
    "/cyphera/webhook-secrets-policy-arn-test"
)

# Extract parameter names from the script
SCRIPT_PARAMS=$(grep -o "/cyphera/[^\"]*-\${STAGE}" scripts/deploy_webhook_lambdas.sh | sed "s/\${STAGE}/${STAGE}/g" | sort | uniq)

echo "Expected parameters:"
printf '%s\n' "${EXPECTED_PARAMS[@]}"
echo ""
echo "Script parameters:"
echo "$SCRIPT_PARAMS"

# Verify all expected parameters are present
for param in "${EXPECTED_PARAMS[@]}"; do
    if echo "$SCRIPT_PARAMS" | grep -q "$param"; then
        echo "✅ Found parameter: $param"
    else
        echo "❌ Missing parameter: $param"
        exit 1
    fi
done

echo ""
echo "🔍 Testing SAM template compatibility..."

# Check if the SAM template exists
if [ ! -f "deployment/template-webhook.yaml" ]; then
    echo "❌ SAM template not found: deployment/template-webhook.yaml"
    exit 1
fi

# Check if the template has the required parameters
REQUIRED_SAM_PARAMS=(
    "Stage"
    "RdsSecretArnValue"
    "DbHostValue"
    "WebhookSqsQueueUrl"
    "WebhookDlqQueueUrl"
    "PaymentSyncEncryptionKeySecretArn"
    "LambdaSecurityGroupId"
    "PrivateSubnet1Id"
    "PrivateSubnet2Id"
    "WebhookSecretsManagerPolicyArn"
)

for param in "${REQUIRED_SAM_PARAMS[@]}"; do
    if grep -q "$param:" deployment/template-webhook.yaml; then
        echo "✅ SAM template has parameter: $param"
    else
        echo "❌ SAM template missing parameter: $param"
        exit 1
    fi
done

echo ""
echo "🔍 Testing Makefile targets..."

# Check if the required Makefile targets exist
REQUIRED_TARGETS=(
    "build-WebhookReceiverFunction"
    "build-WebhookProcessorFunction"
    "build-DLQProcessorFunction"
)

for target in "${REQUIRED_TARGETS[@]}"; do
    if grep -q "^$target:" Makefile; then
        echo "✅ Makefile has target: $target"
    else
        echo "❌ Makefile missing target: $target"
        exit 1
    fi
done

echo ""
echo "🔍 Testing Lambda function directories..."

# Check if the Lambda function directories exist
LAMBDA_DIRS=(
    "cmd/webhook-receiver"
    "cmd/webhook-processor"
    "cmd/dlq-processor"
)

for dir in "${LAMBDA_DIRS[@]}"; do
    if [ -d "$dir" ] && [ -f "$dir/main.go" ]; then
        echo "✅ Lambda directory exists: $dir"
    else
        echo "❌ Lambda directory missing or no main.go: $dir"
        exit 1
    fi
done

echo ""
echo "✅ All webhook deployment tests passed!"
echo ""
echo "📋 Summary:"
echo "  - Deployment script exists and is executable"
echo "  - Environment variable validation works"
echo "  - Parameter names are correctly constructed"
echo "  - SAM template has all required parameters"
echo "  - Makefile has all required build targets"
echo "  - Lambda function directories exist"
echo ""
echo "🚀 The webhook deployment infrastructure is ready!" 