#!/bin/bash
set -eo pipefail # Exit on error, treat unset variables as error, propagate pipeline errors

# 1. Input Validation
# Check required environment variables passed from GitHub Actions
: "${STAGE:?Environment variable STAGE is required. Should be 'dev' or 'prod'.}"
: "${AWS_REGION:?Environment variable AWS_REGION is required.}"
: "${SAM_DEPLOYMENT_BUCKET:?Environment variable SAM_DEPLOYMENT_BUCKET is required.}"
: "${LAMBDA_SG_ID:?Environment variable LAMBDA_SG_ID is required. Set via GitHub Environment secrets.}"
: "${PRIVATE_SUBNET_1_ID:?Environment variable PRIVATE_SUBNET_1_ID is required. Set via GitHub Environment secrets.}"
: "${PRIVATE_SUBNET_2_ID:?Environment variable PRIVATE_SUBNET_2_ID is required. Set via GitHub Environment secrets.}"

echo "--- Starting SAM Deployment for stage: ${STAGE} in region: ${AWS_REGION} ---"

# 2. Define Stack Name
STACK_NAME="cyphera-api-${STAGE}"
echo "Stack Name: ${STACK_NAME}"
echo "Deployment Bucket: ${SAM_DEPLOYMENT_BUCKET}"
echo "Lambda SG ID: ${LAMBDA_SG_ID}"
echo "Private Subnet 1 ID: ${PRIVATE_SUBNET_1_ID}"
echo "Private Subnet 2 ID: ${PRIVATE_SUBNET_2_ID}"

# 3. Define Parameter/Secret Names (Dynamically based on STAGE)
PARAM_DELEGATION_DNS_NAME="/cyphera/delegation-server-alb-dns-${STAGE}"
PARAM_RDS_SECRET_ARN_NAME="/cyphera/rds-secret-arn-${STAGE}"
PARAM_RDS_ENDPOINT_NAME="/cyphera/rds-endpoint-${STAGE}"
PARAM_SUPABASE_URL_NAME="/cyphera/supabase/url-${STAGE}"
PARAM_SUPABASE_JWT_SECRET_ARN_NAME="/cyphera/cyphera-api/supabase-jwt-secret-arn-${STAGE}"
PARAM_CIRCLE_API_KEY_ARN_NAME="/cyphera/cyphera-api/circle-api-key-arn-${STAGE}"
PARAM_SMART_WALLET_NAME="/cyphera/wallet/smart-wallet-address-${STAGE}"
PARAM_CORS_ORIGINS_NAME="/cyphera/cors/allowed-origins-${STAGE}"
PARAM_CORS_METHODS_NAME="/cyphera/cors/allowed-methods-${STAGE}"
PARAM_CORS_HEADERS_NAME="/cyphera/cors/allowed-headers-${STAGE}"
PARAM_CORS_EXPOSED_HEADERS_NAME="/cyphera/cors/exposed-headers-${STAGE}"
PARAM_CORS_CREDENTIALS_NAME="/cyphera/cors/allow-credentials-${STAGE}"

# 4. Fetch Parameter/Secret Values from AWS
echo "Fetching parameters from AWS..."

fetch_ssm_param() {
    local param_name="$1"
    local with_decryption_flag="${2:---no-with-decryption}" # Default to no decryption
    aws ssm get-parameter --name "${param_name}" --query "Parameter.Value" --output text --region "${AWS_REGION}" "${with_decryption_flag}" || { echo "ERROR: Failed to fetch SSM parameter ${param_name}" >&2; exit 1; }
}

# Fetch values - exit script if any fetch fails
PARAM_DELEGATION_DNS_VALUE=$(fetch_ssm_param "${PARAM_DELEGATION_DNS_NAME}")
PARAM_RDS_SECRET_ARN_VALUE=$(fetch_ssm_param "${PARAM_RDS_SECRET_ARN_NAME}")
PARAM_RDS_ENDPOINT_VALUE=$(fetch_ssm_param "${PARAM_RDS_ENDPOINT_NAME}")
PARAM_SUPABASE_URL_VALUE=$(fetch_ssm_param "${PARAM_SUPABASE_URL_NAME}" "--with-decryption")
PARAM_SUPABASE_JWT_SECRET_ARN_VALUE=$(fetch_ssm_param "${PARAM_SUPABASE_JWT_SECRET_ARN_NAME}")
PARAM_CIRCLE_API_KEY_ARN_VALUE=$(fetch_ssm_param "${PARAM_CIRCLE_API_KEY_ARN_NAME}")
PARAM_SMART_WALLET_VALUE=$(fetch_ssm_param "${PARAM_SMART_WALLET_NAME}" "--with-decryption")
PARAM_CORS_ORIGINS_VALUE=$(fetch_ssm_param "${PARAM_CORS_ORIGINS_NAME}")
PARAM_CORS_METHODS_VALUE=$(fetch_ssm_param "${PARAM_CORS_METHODS_NAME}")
PARAM_CORS_HEADERS_VALUE=$(fetch_ssm_param "${PARAM_CORS_HEADERS_NAME}")
PARAM_CORS_EXPOSED_HEADERS_VALUE=$(fetch_ssm_param "${PARAM_CORS_EXPOSED_HEADERS_NAME}")
PARAM_CORS_CREDENTIALS_VALUE=$(fetch_ssm_param "${PARAM_CORS_CREDENTIALS_NAME}")

# Construct gRPC address for Main API
DELEGATION_GRPC_ADDR_VALUE="${PARAM_DELEGATION_DNS_VALUE}:50051"
echo "Constructed Delegation gRPC Address for Main API: ${DELEGATION_GRPC_ADDR_VALUE}"

echo "Successfully fetched all parameters."

# 5. Construct Parameter Overrides String
OVERRIDES="Stage=${STAGE}"
OVERRIDES="${OVERRIDES} LambdaSecurityGroupId=${LAMBDA_SG_ID}"
OVERRIDES="${OVERRIDES} PrivateSubnet1Id=${PRIVATE_SUBNET_1_ID}"
OVERRIDES="${OVERRIDES} PrivateSubnet2Id=${PRIVATE_SUBNET_2_ID}"
OVERRIDES="${OVERRIDES} ParamDelegationServerAlbDns=${PARAM_DELEGATION_DNS_VALUE}"
OVERRIDES="${OVERRIDES} ParamRdsSecretArn=${PARAM_RDS_SECRET_ARN_VALUE}"
OVERRIDES="${OVERRIDES} ParamRdsEndpoint=${PARAM_RDS_ENDPOINT_VALUE}"
OVERRIDES="${OVERRIDES} ParamSupabaseUrl=${PARAM_SUPABASE_URL_VALUE}"
OVERRIDES="${OVERRIDES} ParamSupabaseJwtSecretArn=${PARAM_SUPABASE_JWT_SECRET_ARN_VALUE}"
OVERRIDES="${OVERRIDES} ParamCircleApiKeyArn=${PARAM_CIRCLE_API_KEY_ARN_VALUE}"
OVERRIDES="${OVERRIDES} ParamSmartWalletAddress=${PARAM_SMART_WALLET_VALUE}"
OVERRIDES="${OVERRIDES} ParamCorsAllowedOrigins=${PARAM_CORS_ORIGINS_VALUE}"
OVERRIDES="${OVERRIDES} ParamCorsAllowedMethods=${PARAM_CORS_METHODS_VALUE}"
OVERRIDES="${OVERRIDES} ParamCorsAllowedHeaders=${PARAM_CORS_HEADERS_VALUE}"
OVERRIDES="${OVERRIDES} ParamCorsExposedHeaders=${PARAM_CORS_EXPOSED_HEADERS_VALUE}"
OVERRIDES="${OVERRIDES} ParamCorsAllowCredentials=${PARAM_CORS_CREDENTIALS_VALUE}"
OVERRIDES="${OVERRIDES} paramDelegationGrpcAddr=${DELEGATION_GRPC_ADDR_VALUE}"
# DeploymentBucketName is passed via --s3-bucket, not parameter override

echo "Constructed Parameter Overrides."
# Use placeholder for sensitive values in log - Removed masking as secrets aren't passed directly
echo "Overrides: ${OVERRIDES}"


# 6. Execute SAM Deploy
echo "Executing sam deploy..."
sam deploy \
  --template-file .aws-sam/build/template.yaml \
  --stack-name "${STACK_NAME}" \
  --s3-bucket "${SAM_DEPLOYMENT_BUCKET}" \
  --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND \
  --region "${AWS_REGION}" \
  --no-confirm-changeset \
  --no-fail-on-empty-changeset \
  --parameter-overrides "${OVERRIDES}"

echo "--- SAM Deployment script finished successfully ---" 