#!/bin/bash
set -eo pipefail # Exit on error, treat unset variables as error, propagate pipeline errors

# ===============================================
# Webhook Lambda Functions SAM Deployment Script
# ===============================================

# 1. Input Validation
# Check required environment variables passed from GitHub Actions
: "${STAGE:?Environment variable STAGE is required. Should be 'dev' or 'prod'.}"
: "${AWS_REGION:?Environment variable AWS_REGION is required.}"
: "${SAM_DEPLOYMENT_BUCKET:?Environment variable SAM_DEPLOYMENT_BUCKET is required.}"
: "${LAMBDA_SG_ID:?Environment variable LAMBDA_SG_ID is required. Set via GitHub Environment secrets.}"
: "${PRIVATE_SUBNET_1_ID:?Environment variable PRIVATE_SUBNET_1_ID is required. Set via GitHub Environment secrets.}"
: "${PRIVATE_SUBNET_2_ID:?Environment variable PRIVATE_SUBNET_2_ID is required. Set via GitHub Environment secrets.}"

echo "--- Starting Webhook SAM Deployment for stage: ${STAGE} in region: ${AWS_REGION} ---"

# 2. Define Stack Name and Template File
STACK_NAME="cyphera-webhook-${STAGE}"
TEMPLATE_FILE="deployment/template-webhook.yaml"
BUILD_TEMPLATE_FILE=".aws-sam/build/template.yaml" # SAM build output

echo "Stack Name: ${STACK_NAME}"
echo "Template File: ${TEMPLATE_FILE}"
echo "Deployment Bucket: ${SAM_DEPLOYMENT_BUCKET}"
echo "Lambda SG ID: ${LAMBDA_SG_ID}"
echo "Private Subnet 1 ID: ${PRIVATE_SUBNET_1_ID}"
echo "Private Subnet 2 ID: ${PRIVATE_SUBNET_2_ID}"

# 3. Define Parameter/Secret Names (Dynamically based on STAGE)
# These are the SSM parameters that store the Terraform outputs
PARAM_RDS_SECRET_ARN_NAME="/cyphera/rds-secret-arn-${STAGE}"
PARAM_RDS_ENDPOINT_NAME="/cyphera/rds-endpoint-${STAGE}"
PARAM_WEBHOOK_SQS_QUEUE_URL_NAME="/cyphera/webhook-sqs-queue-url-${STAGE}"
PARAM_WEBHOOK_DLQ_QUEUE_URL_NAME="/cyphera/webhook-dlq-queue-url-${STAGE}"
PARAM_PAYMENT_SYNC_ENCRYPTION_KEY_ARN_NAME="/cyphera/payment-sync-encryption-key-arn-${STAGE}"
PARAM_WEBHOOK_SECRETS_POLICY_ARN_NAME="/cyphera/webhook-secrets-policy-arn-${STAGE}"

# 4. Fetch Parameter/Secret Values from AWS
echo "Fetching parameters from AWS for Webhook deployment..."

fetch_ssm_param() {
    local param_name="$1"
    local with_decryption_flag="${2:---no-with-decryption}" # Default to no decryption
    aws ssm get-parameter --name "${param_name}" --query "Parameter.Value" --output text --region "${AWS_REGION}" "${with_decryption_flag}" || { echo "ERROR: Failed to fetch SSM parameter ${param_name}" >&2; exit 1; }
}

# Fetch values - exit script if any fetch fails
PARAM_RDS_SECRET_ARN_VALUE=$(fetch_ssm_param "${PARAM_RDS_SECRET_ARN_NAME}")
PARAM_RDS_ENDPOINT_VALUE=$(fetch_ssm_param "${PARAM_RDS_ENDPOINT_NAME}")
PARAM_WEBHOOK_SQS_QUEUE_URL_VALUE=$(fetch_ssm_param "${PARAM_WEBHOOK_SQS_QUEUE_URL_NAME}")
PARAM_WEBHOOK_DLQ_QUEUE_URL_VALUE=$(fetch_ssm_param "${PARAM_WEBHOOK_DLQ_QUEUE_URL_NAME}")
PARAM_PAYMENT_SYNC_ENCRYPTION_KEY_ARN_VALUE=$(fetch_ssm_param "${PARAM_PAYMENT_SYNC_ENCRYPTION_KEY_ARN_NAME}")
PARAM_WEBHOOK_SECRETS_POLICY_ARN_VALUE=$(fetch_ssm_param "${PARAM_WEBHOOK_SECRETS_POLICY_ARN_NAME}")

echo "Successfully fetched all parameters for Webhook deployment."

# 5. Extract DB host from RDS endpoint (remove port)
# RDS endpoint format: hostname:port, we need just hostname for DB_HOST
DB_HOST_VALUE=$(echo "${PARAM_RDS_ENDPOINT_VALUE}" | cut -d':' -f1)

echo "Extracted DB Host: ${DB_HOST_VALUE}"

# 6. Convert SQS Queue URLs to ARNs for Lambda Event Source Mappings
# SQS Event Source Mappings require ARNs, not URLs
# URL format: https://sqs.region.amazonaws.com/account-id/queue-name
# ARN format: arn:aws:sqs:region:account-id:queue-name

convert_sqs_url_to_arn() {
    local queue_url="$1"
    # Extract components from URL
    # Example: https://sqs.us-east-1.amazonaws.com/699475955358/cyphera-provider-webhook-events-dev
    local queue_name=$(basename "${queue_url}")
    local account_id=$(echo "${queue_url}" | cut -d'/' -f4)
    local region=$(echo "${queue_url}" | cut -d'.' -f2)
    
    echo "arn:aws:sqs:${region}:${account_id}:${queue_name}"
}

WEBHOOK_SQS_QUEUE_ARN=$(convert_sqs_url_to_arn "${PARAM_WEBHOOK_SQS_QUEUE_URL_VALUE}")
WEBHOOK_DLQ_QUEUE_ARN=$(convert_sqs_url_to_arn "${PARAM_WEBHOOK_DLQ_QUEUE_URL_VALUE}")

echo "Converted SQS URLs to ARNs:"
echo "  Webhook Queue ARN: ${WEBHOOK_SQS_QUEUE_ARN}"
echo "  Webhook DLQ ARN: ${WEBHOOK_DLQ_QUEUE_ARN}"

# 7. Construct Parameter Overrides String for Webhook SAM template
OVERRIDES="Stage=${STAGE}"
OVERRIDES="${OVERRIDES} LambdaSecurityGroupId=${LAMBDA_SG_ID}"
OVERRIDES="${OVERRIDES} PrivateSubnet1Id=${PRIVATE_SUBNET_1_ID}"
OVERRIDES="${OVERRIDES} PrivateSubnet2Id=${PRIVATE_SUBNET_2_ID}"
# Pass the *actual fetched values* to the template parameters
OVERRIDES="${OVERRIDES} RdsSecretArnValue=${PARAM_RDS_SECRET_ARN_VALUE}"
OVERRIDES="${OVERRIDES} DbHostValue=${DB_HOST_VALUE}"
OVERRIDES="${OVERRIDES} WebhookSqsQueueUrl=${WEBHOOK_SQS_QUEUE_ARN}"
OVERRIDES="${OVERRIDES} WebhookDlqQueueUrl=${WEBHOOK_DLQ_QUEUE_ARN}"
OVERRIDES="${OVERRIDES} PaymentSyncEncryptionKeySecretArn=${PARAM_PAYMENT_SYNC_ENCRYPTION_KEY_ARN_VALUE}"
OVERRIDES="${OVERRIDES} WebhookSecretsManagerPolicyArn=${PARAM_WEBHOOK_SECRETS_POLICY_ARN_VALUE}"

echo "Constructed Parameter Overrides for Webhook deployment."
echo "Overrides: ${OVERRIDES}"

# 8. Execute SAM Deploy for Webhook Infrastructure
echo "Executing sam deploy for ${STACK_NAME}..."
sam deploy \
  --template-file "${BUILD_TEMPLATE_FILE}" \
  --stack-name "${STACK_NAME}" \
  --s3-bucket "${SAM_DEPLOYMENT_BUCKET}" \
  --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND \
  --region "${AWS_REGION}" \
  --no-confirm-changeset \
  --no-fail-on-empty-changeset \
  --parameter-overrides "${OVERRIDES}"

echo "--- Webhook SAM Deployment script finished successfully ---"

# 9. Display Deployment Information
echo ""
echo "=== Webhook Deployment Complete ==="
echo "Stack Name: ${STACK_NAME}"
echo "Region: ${AWS_REGION}"
echo "Stage: ${STAGE}"
echo ""
echo "Deployed Lambda Functions:"
echo "  - cyphera-webhook-receiver-${STAGE}"
echo "  - cyphera-webhook-processor-${STAGE}"
echo "  - cyphera-dlq-processor-${STAGE}"
echo ""
echo "API Gateway:"
echo "  - cyphera-webhook-api-${STAGE}"
echo ""
echo "To check the deployment status:"
echo "  aws cloudformation describe-stacks --stack-name ${STACK_NAME} --region ${AWS_REGION}"
echo ""
echo "To view logs:"
echo "  aws logs describe-log-groups --log-group-name-prefix '/aws/lambda/cyphera-webhook' --region ${AWS_REGION}"
echo "" 