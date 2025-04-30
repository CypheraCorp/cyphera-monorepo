#!/bin/bash
set -eo pipefail # Exit on error, treat unset variables as error, propagate pipeline errors

# --- Deployment Script for Subscription Processor Lambda ---

# 1. Input Validation
# Check required environment variables passed from GitHub Actions
: "${STAGE:?Environment variable STAGE is required. Should be 'dev' or 'prod'.}"
: "${AWS_REGION:?Environment variable AWS_REGION is required.}"
: "${SAM_DEPLOYMENT_BUCKET:?Environment variable SAM_DEPLOYMENT_BUCKET is required.}"
: "${LAMBDA_SG_ID:?Environment variable LAMBDA_SG_ID is required. Set via GitHub Environment secrets.}"
: "${PRIVATE_SUBNET_1_ID:?Environment variable PRIVATE_SUBNET_1_ID is required. Set via GitHub Environment secrets.}"
: "${PRIVATE_SUBNET_2_ID:?Environment variable PRIVATE_SUBNET_2_ID is required. Set via GitHub Environment secrets.}"

echo "--- Starting SAM Deployment for Subscription Processor: ${STAGE} in region: ${AWS_REGION} ---"

# 2. Define Stack Name and Template File
STACK_NAME="cyphera-subprocessor-${STAGE}"
TEMPLATE_FILE="template-subprocessor.yaml"
# SAM build output template path
BUILD_TEMPLATE_FILE=".aws-sam/build/template.yaml" # Default SAM build output

echo "Stack Name: ${STACK_NAME}"
echo "Template File: ${TEMPLATE_FILE}"
echo "Deployment Bucket: ${SAM_DEPLOYMENT_BUCKET}"
echo "Lambda SG ID: ${LAMBDA_SG_ID}"
echo "Private Subnet 1 ID: ${PRIVATE_SUBNET_1_ID}"
echo "Private Subnet 2 ID: ${PRIVATE_SUBNET_2_ID}"

# 3. Define Parameter/Secret Names (Dynamically based on STAGE)
# These are the parameters expected by template-subprocessor.yaml
PARAM_RDS_SECRET_ARN_NAME="/cyphera/rds-secret-arn-${STAGE}" # Name of the SSM param holding the RDS Secret ARN
PARAM_RDS_ENDPOINT_NAME="/cyphera/rds-endpoint-${STAGE}"   # Name of the SSM param holding the RDS Host:Port
PARAM_DB_NAME_NAME="/cyphera/rds-db-name-${STAGE}"        # Name of the SSM param holding the DB Name (Optional, could be env var)
PARAM_SMART_WALLET_NAME="/cyphera/wallet/smart-wallet-address-${STAGE}"
PARAM_DELEGATION_DNS_NAME="/cyphera/delegation-server-alb-dns-${STAGE}" # ADDED - ALB DNS Name parameter

# Add DB Name parameter if it doesn't exist (using default 'cyphera_api' if not set via TF)
# You might manage this via Terraform instead
aws ssm put-parameter --name "${PARAM_DB_NAME_NAME}" --value "cyphera_api_${STAGE}" --type String --overwrite --region "${AWS_REGION}" 2>/dev/null || echo "DB Name parameter potentially exists."

# 4. Fetch Parameter/Secret Values from AWS
echo "Fetching parameters from AWS for Subscription Processor..."

fetch_ssm_param() {
    local param_name="$1"
    local with_decryption_flag="${2:---no-with-decryption}" # Default to no decryption
    aws ssm get-parameter --name "${param_name}" --query "Parameter.Value" --output text --region "${AWS_REGION}" "${with_decryption_flag}" || { echo "ERROR: Failed to fetch SSM parameter ${param_name}" >&2; exit 1; }
}

# Fetch values - exit script if any fetch fails
PARAM_RDS_SECRET_ARN_VALUE=$(fetch_ssm_param "${PARAM_RDS_SECRET_ARN_NAME}")
PARAM_RDS_ENDPOINT_VALUE=$(fetch_ssm_param "${PARAM_RDS_ENDPOINT_NAME}")
PARAM_DB_NAME_VALUE=$(fetch_ssm_param "${PARAM_DB_NAME_NAME}")
PARAM_SMART_WALLET_VALUE=$(fetch_ssm_param "${PARAM_SMART_WALLET_NAME}" "--with-decryption")
PARAM_DELEGATION_DNS_VALUE=$(fetch_ssm_param "${PARAM_DELEGATION_DNS_NAME}")

echo "Successfully fetched all parameters for Subscription Processor."

# 5. Construct Parameter Overrides String for Subscription Processor
OVERRIDES="Stage=${STAGE}"
OVERRIDES="${OVERRIDES} LambdaSecurityGroupId=${LAMBDA_SG_ID}"
OVERRIDES="${OVERRIDES} PrivateSubnet1Id=${PRIVATE_SUBNET_1_ID}"
OVERRIDES="${OVERRIDES} PrivateSubnet2Id=${PRIVATE_SUBNET_2_ID}"
# Pass the *actual fetched values* to the template parameters
OVERRIDES="${OVERRIDES} RdsSecretArnValue=${PARAM_RDS_SECRET_ARN_VALUE}"
OVERRIDES="${OVERRIDES} DbHostValue=${PARAM_RDS_ENDPOINT_VALUE}"
OVERRIDES="${OVERRIDES} DbNameValue=${PARAM_DB_NAME_VALUE}"
OVERRIDES="${OVERRIDES} SmartWalletAddressValue=${PARAM_SMART_WALLET_VALUE}"
# Pass the ALB DNS name parameter instead of the constructed address
OVERRIDES="${OVERRIDES} ParamDelegationServerAlbDns=${PARAM_DELEGATION_DNS_VALUE}"

# Add other parameters required by template-subprocessor.yaml if any

echo "Constructed Parameter Overrides for Subscription Processor."
echo "Overrides: ${OVERRIDES}" # Be cautious logging sensitive values if any are passed directly

# 6. Execute SAM Deploy for Subscription Processor
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

echo "--- SAM Deployment script for Subscription Processor finished successfully --- " 