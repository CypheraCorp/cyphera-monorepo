# ===============================================
# Multi-Provider Webhook Secrets
# ===============================================

# Global encryption key for encrypting workspace payment configurations in the database
# This is the only global secret needed - it encrypts/decrypts workspace-specific credentials
resource "aws_secretsmanager_secret" "payment_sync_encryption_key" {
  name        = "/cyphera/cyphera-api/payment-sync-encryption-key-${var.stage}"
  description = "AES-256 encryption key for payment sync configuration data - ${var.stage}"
  tags = merge(local.common_tags, {
    Service   = "cyphera-api"
    Component = "webhook-infrastructure"
  })
}

resource "aws_secretsmanager_secret_version" "payment_sync_encryption_key_version" {
  secret_id     = aws_secretsmanager_secret.payment_sync_encryption_key.id
  secret_string = var.payment_sync_encryption_key_value

  lifecycle {
    ignore_changes = [
      secret_string,
    ]
  }
}

# ===============================================
# SSM Parameters for Webhook Secrets (for SAM deployment)
# ===============================================

resource "aws_ssm_parameter" "payment_sync_encryption_key_arn" {
  name        = "/cyphera/payment-sync-encryption-key-arn-${var.stage}"
  description = "ARN of the payment sync encryption key secret for stage ${var.stage}"
  type        = "String"
  value       = aws_secretsmanager_secret.payment_sync_encryption_key.arn
  tags        = local.common_tags
}

# ===============================================
# Outputs
# ===============================================

# NOTE: Stripe API keys and webhook secrets are now managed per-workspace
# in the workspace_payment_configurations table (encrypted with the key above)
# This eliminates the need for global provider credentials. 