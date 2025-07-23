# ===============================================
# Multi-Provider Webhook SQS Infrastructure
# ===============================================

# Main SQS Queue for webhook events
resource "aws_sqs_queue" "provider_webhook_events" {
  name = "${var.service_prefix}-provider-webhook-events-${var.stage}"

  # Enhanced configuration for multi-provider multi-workspace
  visibility_timeout_seconds = 300     # 5 minutes - enough time for Lambda processing
  message_retention_seconds  = 1209600 # 14 days
  receive_wait_time_seconds  = 20      # Enable long polling for efficiency

  # Redrive policy for failed messages
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.provider_webhook_events_dlq.arn
    maxReceiveCount     = 3
  })

  tags = merge(local.common_tags, {
    Name      = "${var.service_prefix}-provider-webhook-events-${var.stage}"
    Purpose   = "Multi-provider multi-workspace webhook event processing"
    Component = "webhook-infrastructure"
  })
}

# Dead Letter Queue for failed webhook events
resource "aws_sqs_queue" "provider_webhook_events_dlq" {
  name = "${var.service_prefix}-provider-webhook-events-dlq-${var.stage}"

  # DLQ configuration
  message_retention_seconds = 1209600 # 14 days retention for analysis

  tags = merge(local.common_tags, {
    Name      = "${var.service_prefix}-provider-webhook-events-dlq-${var.stage}"
    Purpose   = "Dead letter queue for failed webhook events"
    Component = "webhook-infrastructure"
  })
}

# ===============================================
# SSM Parameters for Webhook SQS (for SAM deployment)
# ===============================================

resource "aws_ssm_parameter" "webhook_sqs_queue_url" {
  name        = "/cyphera/webhook-sqs-queue-url-${var.stage}"
  description = "URL of the webhook SQS queue for stage ${var.stage}"
  type        = "String"
  value       = aws_sqs_queue.provider_webhook_events.url
  tags        = local.common_tags
}

resource "aws_ssm_parameter" "webhook_dlq_queue_url" {
  name        = "/cyphera/webhook-dlq-queue-url-${var.stage}"
  description = "URL of the webhook DLQ for stage ${var.stage}"
  type        = "String"
  value       = aws_sqs_queue.provider_webhook_events_dlq.url
  tags        = local.common_tags
}

# ===============================================
# Outputs
# ===============================================

output "webhook_sqs_queue_url" {
  description = "URL of the webhook SQS queue"
  value       = aws_sqs_queue.provider_webhook_events.url
}

output "webhook_sqs_queue_arn" {
  description = "ARN of the webhook SQS queue"
  value       = aws_sqs_queue.provider_webhook_events.arn
}

output "webhook_dlq_queue_url" {
  description = "URL of the webhook DLQ"
  value       = aws_sqs_queue.provider_webhook_events_dlq.url
}

output "webhook_dlq_queue_arn" {
  description = "ARN of the webhook DLQ"
  value       = aws_sqs_queue.provider_webhook_events_dlq.arn
}

# Note: CloudWatch alarms and queue policies are managed in webhook_monitoring.tf
# Note: SQS access permissions are managed via IAM policies attached to SAM Lambda roles 