# SSM Parameters for Go API (Lambda)

resource "aws_ssm_parameter" "supabase_url" {
  name        = "/cyphera/supabase/url-${var.stage}"
  description = "Supabase URL for stage ${var.stage}"
  type        = "SecureString" # Store as SecureString
  value       = "dummy-value-update-manually" # Placeholder - **MUST BE UPDATED MANUALLY IN AWS CONSOLE**
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "smart_wallet_address" {
  name        = "/cyphera/wallet/smart-wallet-address-${var.stage}"
  description = "Cyphera Smart Wallet Address for stage ${var.stage}"
  type        = "SecureString" # Store as SecureString
  value       = "dummy-value-update-manually" # Placeholder - **MUST BE UPDATED MANUALLY IN AWS CONSOLE**
  tags        = local.common_tags
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

resource "aws_ssm_parameter" "delegation_rpc_url" {
  name        = "/cyphera/delegation-server/rpc-url-${var.stage}"
  description = "Blockchain RPC URL for Delegation Server stage ${var.stage}"
  type        = "SecureString"
  value       = "dummy-value-update-manually" # Placeholder - **MUST BE UPDATED MANUALLY IN AWS CONSOLE**
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "delegation_bundler_url" {
  name        = "/cyphera/delegation-server/bundler-url-${var.stage}"
  description = "Bundler URL for Delegation Server stage ${var.stage}"
  type        = "SecureString"
  value       = "dummy-value-update-manually" # Placeholder - **MUST BE UPDATED MANUALLY IN AWS CONSOLE**
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "delegation_chain_id" {
  name        = "/cyphera/delegation-server/chain-id-${var.stage}"
  description = "Blockchain Chain ID for Delegation Server stage ${var.stage}"
  type        = "String"
  value       = "11155111" # Sepolia default, adjust if needed per stage
  tags        = local.common_tags
} 