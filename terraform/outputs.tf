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
  value       = aws_secretsmanager_secret.rds_master_password.arn # From rds.tf
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
}

# Keep existing outputs if needed, and potentially add others (like ECR repo URL) 