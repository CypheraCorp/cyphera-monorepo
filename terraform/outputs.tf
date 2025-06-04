# Output for the main RDS instance endpoint
output "rds_endpoint" {
  description = "The endpoint of the RDS instance"
  value       = aws_db_instance.main.endpoint
}

# Output for the main RDS instance ARN
output "rds_arn" {
  description = "The ARN of the RDS instance"
  value       = aws_db_instance.main.arn
}

# Output for RDS secret ARN (required by SAM)
output "rds_secret_arn" {
  description = "The ARN of the RDS master user secret"
  value       = aws_db_instance.main.master_user_secret[0].secret_arn
}

# Output for RDS host (without port)
output "db_host" {
  description = "The RDS host without port"
  value       = aws_db_instance.main.address
}

# Output for Lambda Security Group ID (required by SAM)
output "lambda_security_group_id" {
  description = "The Security Group ID for Lambda functions"
  value       = aws_security_group.lambda.id
}

# Output for Private Subnet IDs (required by SAM)
output "private_subnet_1_id" {
  description = "The ID of the first private subnet"
  value       = module.vpc.private_subnets[0]
}

output "private_subnet_2_id" {
  description = "The ID of the second private subnet"
  value       = module.vpc.private_subnets[1]
}

# Output for Payment Sync Encryption Key ARN (required by SAM)
output "payment_sync_encryption_key_secret_arn" {
  description = "The ARN of the payment sync encryption key secret"
  value       = aws_secretsmanager_secret.payment_sync_encryption_key.arn
}

# Output for Webhook Secrets Manager Policy ARN (required by SAM)
output "webhook_secrets_manager_policy_arn" {
  description = "The ARN of the webhook secrets manager policy"
  value       = aws_iam_policy.webhook_secrets_policy.arn
}

# --- SSM Parameter for Lambda Security Group ID ---
# Store the Lambda SG ID in SSM for lookup by Serverless Framework
resource "aws_ssm_parameter" "lambda_sg_id" {
  name        = "/cyphera/lambda-security-group-id-${var.stage}"
  description = "The Security Group ID for the Lambda functions for stage ${var.stage}"
  type        = "String"
  value       = aws_security_group.lambda.id # Reference the SG created in main.tf
  tags        = local.common_tags
}

# --- SSM Parameters for Private Subnet IDs ---
# Store the Private Subnet IDs used by Lambda for lookup by Serverless Framework
# Assumes module.vpc.private_subnets provides at least two subnets consistently.
resource "aws_ssm_parameter" "private_subnet_1" {
  name        = "/cyphera/private-subnet-1-${var.stage}"
  description = "The ID of the first private subnet for stage ${var.stage}"
  type        = "String"
  value       = module.vpc.private_subnets[0] # Reference the first private subnet
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "private_subnet_2" {
  name        = "/cyphera/private-subnet-2-${var.stage}"
  description = "The ID of the second private subnet for stage ${var.stage}"
  type        = "String"
  value       = module.vpc.private_subnets[1] # Reference the second private subnet
  tags        = local.common_tags
}

# --- SSM Parameter for RDS Secret ARN ---
resource "aws_ssm_parameter" "rds_secret_arn" {
  name        = "/cyphera/rds-secret-arn-${var.stage}"
  description = "The ARN of the Secrets Manager secret for RDS credentials for stage ${var.stage}"
  type        = "String"
  value       = aws_db_instance.main.master_user_secret[0].secret_arn
  tags        = local.common_tags
}

# --- SSM Parameter for RDS Endpoint ---
resource "aws_ssm_parameter" "rds_endpoint" {
  name        = "/cyphera/rds-endpoint-${var.stage}"
  description = "The connection endpoint for the RDS instance for stage ${var.stage}"
  type        = "String"
  value       = aws_db_instance.main.endpoint # Endpoint includes host:port
  tags        = local.common_tags
}

# --- SSM Parameter for Delegation Server ALB DNS Name ---
# Store the ALB DNS name in SSM for easy lookup by Serverless Framework
resource "aws_ssm_parameter" "delegation_server_alb_dns" {
  name        = "/cyphera/delegation-server-alb-dns-${var.stage}" # Stage-specific name
  description = "The private DNS name for the Delegation Server ALB for stage ${var.stage}"
  type        = "String"
  value       = aws_lb.delegation_server.dns_name # Reference the ALB output
  tags        = local.common_tags

  lifecycle {
    ignore_changes = [value]
  }
}

output "circle_api_key_secret_arn" {
  description = "ARN of the Circle API Key secret"
  value       = aws_secretsmanager_secret.circle_api_key.arn
}

output "circle_api_key_ssm_parameter_name" {
  description = "Name of the SSM parameter storing the Circle API Key secret ARN"
  value       = aws_ssm_parameter.circle_api_key_arn.name
}

output "coin_market_cap_api_key_secret_arn" {
  description = "ARN of the CoinMarketCap API Key secret"
  value       = aws_secretsmanager_secret.coin_market_cap_api_key.arn
}

output "coin_market_cap_api_key_ssm_parameter_name" {
  description = "Name of the SSM parameter storing the CoinMarketCap API Key secret ARN"
  value       = aws_ssm_parameter.coin_market_cap_api_key_arn.name
}

output "infura_api_key_secret_arn" {
  description = "ARN of the Infura API Key secret"
  value       = aws_secretsmanager_secret.infura_api_key.arn
}

output "infura_api_key_ssm_parameter_name" {
  description = "Name of the SSM parameter storing the Infura API Key secret ARN"
  value       = aws_ssm_parameter.infura_api_key_arn.name
}

output "pimlico_api_key_secret_arn" {
  description = "ARN of the Pimlico API Key secret"
  value       = aws_secretsmanager_secret.pimlico_api_key.arn
}

output "pimlico_api_key_ssm_parameter_name" {
  description = "Name of the SSM parameter storing the Pimlico API Key secret ARN"
  value       = aws_ssm_parameter.pimlico_api_key_arn.name
}

# ===============================================
# Note: Webhook infrastructure outputs moved to SAM template
# ===============================================

output "webhook_infrastructure_note" {
  description = "Webhook infrastructure deployment information"
  value = "Webhook API Gateway, Lambda functions, and related resources are deployed via SAM. See deployment/template-webhook.yaml"
} 