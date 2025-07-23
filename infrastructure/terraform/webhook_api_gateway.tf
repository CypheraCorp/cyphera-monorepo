# ===============================================
# Multi-Provider Webhook API Gateway (SAM Integration)
# ===============================================

# NOTE: API Gateway and Lambda functions are now managed by SAM template
# This eliminates the chicken-and-egg dependency issue where Terraform
# tried to reference Lambda functions that didn't exist yet.

# All webhook-related API Gateway resources are defined in:
# deployment/template-webhook.yaml

# ===============================================
# Outputs needed for SAM integration
# ===============================================

output "webhook_api_gateway_note" {
  description = "API Gateway will be managed by SAM template"
  value       = "API Gateway resources are defined in deployment/template-webhook.yaml"
} 