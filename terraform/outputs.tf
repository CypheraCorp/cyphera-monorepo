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

# Delegation server ALB outputs
output "delegation_server_alb_dns_name" {
  description = "DNS name of the delegation server ALB"
  value       = aws_lb.delegation_server.dns_name
}

output "delegation_server_alb_zone_id" {
  description = "Zone ID of the delegation server ALB"
  value       = aws_lb.delegation_server.zone_id
}

# ===============================================
# Stripe Webhook Infrastructure Outputs
# ===============================================

output "stripe_webhook_api_url" {
  description = "URL for the Stripe webhook API Gateway endpoint"
  value       = "${aws_api_gateway_stage.stripe_webhooks.invoke_url}/webhooks/stripe"
}

output "stripe_webhook_api_gateway_id" {
  description = "ID of the Stripe webhooks API Gateway"
  value       = aws_api_gateway_rest_api.stripe_webhooks.id
}

output "stripe_webhook_events_queue_url" {
  description = "URL of the main Stripe webhook events SQS queue"
  value       = aws_sqs_queue.stripe_webhook_events.url
}

output "stripe_webhook_events_queue_arn" {
  description = "ARN of the main Stripe webhook events SQS queue"
  value       = aws_sqs_queue.stripe_webhook_events.arn
}

output "stripe_webhook_events_dlq_url" {
  description = "URL of the Stripe webhook events dead letter queue"
  value       = aws_sqs_queue.stripe_webhook_events_dlq.url
}

output "stripe_webhook_receiver_lambda_arn" {
  description = "ARN of the Stripe webhook receiver Lambda function"
  value       = aws_lambda_function.stripe_webhook_receiver.arn
}

output "stripe_webhook_processor_lambda_arn" {
  description = "ARN of the Stripe webhook processor Lambda function"
  value       = aws_lambda_function.stripe_webhook_processor.arn
}

output "stripe_webhook_receiver_lambda_name" {
  description = "Name of the Stripe webhook receiver Lambda function"
  value       = aws_lambda_function.stripe_webhook_receiver.function_name
}

output "stripe_webhook_processor_lambda_name" {
  description = "Name of the Stripe webhook processor Lambda function"
  value       = aws_lambda_function.stripe_webhook_processor.function_name
}

# ===============================================
# Webhook Infrastructure Outputs (for SAM)
# ===============================================

# SQS Outputs
output "webhook_sqs_queue_url" {
  description = "URL of the webhook events SQS queue"
  value       = aws_sqs_queue.provider_webhook_events.url
}

output "webhook_sqs_queue_arn" {
  description = "ARN of the webhook events SQS queue"
  value       = aws_sqs_queue.provider_webhook_events.arn
}

output "webhook_sqs_dlq_arn" {
  description = "ARN of the webhook events DLQ"
  value       = aws_sqs_queue.provider_webhook_events_dlq.arn
}

# Secrets Manager Outputs
output "stripe_api_key_secret_arn" {
  description = "ARN of Stripe API key secret"
  value       = aws_secretsmanager_secret.stripe_api_key.arn
}

output "stripe_webhook_secret_arn" {
  description = "ARN of Stripe webhook secret"
  value       = aws_secretsmanager_secret.stripe_webhook_secret.arn
}

output "payment_sync_encryption_key_secret_arn" {
  description = "ARN of payment sync encryption key secret"
  value       = aws_secretsmanager_secret.payment_sync_encryption_key.arn
}

# IAM Policy Outputs (for SAM to attach to Lambda roles)
output "webhook_secrets_policy_arn" {
  description = "ARN of webhook secrets access policy"
  value       = aws_iam_policy.webhook_secrets_policy.arn
}

output "webhook_sqs_policy_arn" {
  description = "ARN of webhook SQS access policy"
  value       = aws_iam_policy.webhook_sqs_policy.arn
}

# API Gateway Outputs
output "webhook_api_gateway_id" {
  description = "ID of the webhook API Gateway"
  value       = aws_api_gateway_rest_api.webhook_api.id
}

output "webhook_api_gateway_root_resource_id" {
  description = "Root resource ID of the webhook API Gateway"
  value       = aws_api_gateway_rest_api.webhook_api.root_resource_id
}

output "webhook_api_endpoint" {
  description = "Webhook API Gateway endpoint URL"
  value       = "https://${aws_api_gateway_rest_api.webhook_api.id}.execute-api.${var.aws_region}.amazonaws.com/${var.stage}"
}

# VPC/Network Outputs (that SAM Lambda functions might need)
# Note: VPC configuration should be passed via SAM parameters from your deployment script
output "webhook_vpc_config" {
  description = "VPC configuration placeholder for webhook Lambda functions"
  value = {
    note = "VPC configuration should be provided via SAM template parameters"
  }
} 