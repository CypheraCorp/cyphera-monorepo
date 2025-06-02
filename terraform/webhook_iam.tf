# ===============================================
# Multi-Provider Webhook IAM Policies (SAM Integration)
# ===============================================

# Note: Lambda IAM roles are managed by SAM
# This file contains shared IAM policies for Terraform-managed resources

# Policy document for webhook infrastructure access to secrets
data "aws_iam_policy_document" "webhook_secrets_access" {
  statement {
    effect = "Allow"
    actions = [
      "secretsmanager:GetSecretValue"
    ]
    resources = [
      aws_secretsmanager_secret.payment_sync_encryption_key.arn,
      aws_db_instance.main.master_user_secret[0].secret_arn,
      "${aws_secretsmanager_secret.payment_sync_encryption_key.arn}:*",
      "${aws_db_instance.main.master_user_secret[0].secret_arn}:*"
    ]
  }
}

# Policy document for SQS access (for Lambda functions managed by SAM)
data "aws_iam_policy_document" "webhook_sqs_access" {
  statement {
    effect = "Allow"
    actions = [
      "sqs:SendMessage",
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:GetQueueUrl"
    ]
    resources = [
      aws_sqs_queue.provider_webhook_events.arn,
      aws_sqs_queue.provider_webhook_events_dlq.arn
    ]
  }
}

# Export these policies as IAM policies for SAM templates to reference
resource "aws_iam_policy" "webhook_secrets_policy" {
  name        = "${var.service_prefix}-webhook-secrets-policy-${var.stage}"
  description = "Policy for webhook Lambda functions to access secrets"
  policy      = data.aws_iam_policy_document.webhook_secrets_access.json

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    Purpose   = "lambda-shared-policy"
  })
}

resource "aws_iam_policy" "webhook_sqs_policy" {
  name        = "${var.service_prefix}-webhook-sqs-policy-${var.stage}"
  description = "Policy for webhook Lambda functions to access SQS"
  policy      = data.aws_iam_policy_document.webhook_sqs_access.json

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    Purpose   = "lambda-shared-policy"
  })
}

# ===============================================
# Outputs
# ===============================================

output "webhook_secrets_policy_arn" {
  description = "ARN of the webhook secrets access policy"
  value       = aws_iam_policy.webhook_secrets_policy.arn
}

output "webhook_sqs_policy_arn" {
  description = "ARN of the webhook SQS access policy"
  value       = aws_iam_policy.webhook_sqs_policy.arn
} 