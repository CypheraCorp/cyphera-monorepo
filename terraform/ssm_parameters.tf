# SSM Parameters for Go API (Lambda)

resource "aws_ssm_parameter" "supabase_url" {
  name        = "/cyphera/supabase/url-${var.stage}"
  description = "Supabase URL for stage ${var.stage}"
  type        = "SecureString" # Store as SecureString
  value       = "dummy-value-update-manually" # Placeholder - **MUST BE UPDATED MANUALLY IN AWS CONSOLE**
  tags        = local.common_tags
  # Ignore subsequent changes to value, allow manual updates
  lifecycle {
    ignore_changes = [value]
  }
}

resource "aws_ssm_parameter" "smart_wallet_address" {
  name        = "/cyphera/wallet/smart-wallet-address-${var.stage}"
  description = "Cyphera Smart Wallet Address for stage ${var.stage}"
  type        = "SecureString" # Store as SecureString
  value       = "dummy-value-update-manually" # Placeholder - **MUST BE UPDATED MANUALLY IN AWS CONSOLE**
  tags        = local.common_tags
  # Ignore subsequent changes to value, allow manual updates
  lifecycle {
    ignore_changes = [value]
  }
}

resource "aws_ssm_parameter" "cors_allowed_origins" {
  name        = "/cyphera/cors/allowed-origins-${var.stage}"
  description = "CORS Allowed Origins for stage ${var.stage} (comma-separated)"
  type        = "String" # Can be String or SecureString
  value       = var.stage == "dev" ? "http://localhost:3000" : "https://app.cypherapay.com" # Example default, adjust prod
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "cors_allowed_methods" {
  name        = "/cyphera/cors/allowed-methods-${var.stage}"
  description = "CORS Allowed Methods for stage ${var.stage} (comma-separated)"
  type        = "String"
  value       = "GET,POST,PUT,DELETE,OPTIONS,PATCH"
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "cors_allowed_headers" {
  name        = "/cyphera/cors/allowed-headers-${var.stage}"
  description = "CORS Allowed Headers for stage ${var.stage} (comma-separated)"
  type        = "String"
  value       = "Origin,Content-Type,Accept,Authorization,X-API-Key,X-Workspace-ID,X-Account-ID"
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "cors_exposed_headers" {
  name        = "/cyphera/cors/exposed-headers-${var.stage}"
  description = "CORS Exposed Headers for stage ${var.stage} (comma-separated)"
  type        = "String"
  value       = "Content-Length,Content-Type" # Adjust if needed
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "cors_allow_credentials" {
  name        = "/cyphera/cors/allow-credentials-${var.stage}"
  description = "CORS Allow Credentials for stage ${var.stage} ('true' or 'false')"
  type        = "String"
  value       = "true"
  tags        = local.common_tags
}


# SSM Parameters for Delegation Server (ECS)

# Parameter for the manually created wildcard API certificate ARN
resource "aws_ssm_parameter" "wildcard_cert_arn" {
  name        = "/cyphera/wildcard-api-cert-arn"
  description = "ARN of the wildcard certificate for api.cypherapay.com (Managed outside this TF state initially)"
  type        = "String"
  value       = "arn:aws:acm:us-east-1:699475955358:certificate/6f8bb8d4-4200-4128-a680-d9854890993b"
  tags        = local.common_tags # Apply common tags if desired
}

# --- Secrets Manager Secret ARNs ---

# Assume you have data sources or resources for your secrets, like:
data "aws_secretsmanager_secret" "supabase_jwt" {
  name = "cyphera/cyphera-api/supabase/jwt-secret-${var.stage}"
}

# Store the Supabase JWT Secret ARN in SSM Parameter Store
resource "aws_ssm_parameter" "supabase_jwt_secret_arn" {
  name        = "/cyphera/cyphera-api/supabase-jwt-secret-arn-${var.stage}"
  description = "ARN of the Supabase JWT secret for Cyphera API - ${var.stage}"
  type        = "String"
  value       = data.aws_secretsmanager_secret.supabase_jwt.arn
  tags        = local.common_tags

  lifecycle {
    ignore_changes = [value] # Avoid unnecessary updates if ARN doesn't change
  }
}

# Store the Circle API Key Secret ARN in SSM Parameter Store
resource "aws_ssm_parameter" "circle_api_key_arn" {
  name        = "/cyphera/cyphera-api/circle-api-key-arn-${var.stage}"
  description = "ARN of the Circle API Key secret for Cyphera API - ${var.stage}"
  type        = "String"
  value       = aws_secretsmanager_secret.circle_api_key.arn # Referencing the secret defined in secrets.tf
  tags        = local.common_tags
  lifecycle {
    ignore_changes = [value] # Avoid unnecessary updates if ARN doesn't change
  }
}

# SSM Parameters for Secret ARNs

resource "aws_ssm_parameter" "coin_market_cap_api_key_arn" {
  name        = "/cyphera/cyphera-api/coin-market-cap-api-key-arn-${var.stage}"
  description = "ARN of the CoinMarketCap API Key secret for Cyphera API - ${var.stage}"
  type        = "String"
  value       = aws_secretsmanager_secret.coin_market_cap_api_key.arn
  tags = local.common_tags
  lifecycle {
    ignore_changes = [value] # Avoid unnecessary updates if ARN doesn't change
  }
}

resource "aws_ssm_parameter" "infura_api_key_arn" {
  name        = "/cyphera/delegation-server/infura-api-key-arn-${var.stage}"
  description = "ARN of the Infura API Key secret for Delegation Server - ${var.stage}"
  type        = "String"
  value       = aws_secretsmanager_secret.infura_api_key.arn
  tags        = local.common_tags
  lifecycle {
    ignore_changes = [value] # Avoid unnecessary updates if ARN doesn't change
  }
}

resource "aws_ssm_parameter" "pimlico_api_key_arn" {
  name        = "/cyphera/delegation-server/pimlico-api-key-arn-${var.stage}"
  description = "ARN of the Pimlico API Key secret for Delegation Server - ${var.stage}"
  type        = "String"
  value       = aws_secretsmanager_secret.pimlico_api_key.arn
  tags        = local.common_tags
  lifecycle {
    ignore_changes = [value] # Avoid unnecessary updates if ARN doesn't change
  }
}

resource "aws_ssm_parameter" "delegation_private_key_arn" {
  name        = "/cyphera/delegation-server/private-key-arn-${var.stage}"
  description = "ARN of the Private Key secret for Delegation Server - ${var.stage}"
  type        = "String"
  value       = aws_secretsmanager_secret.delegation_private_key.arn # Changed to reference the managed resource
  tags        = local.common_tags
  lifecycle { 
    ignore_changes = [value] 
  }
}