# ===============================================
# Multi-Provider Webhook Monitoring (Terraform-managed resources only)
# ===============================================

# NOTE: API Gateway monitoring is now handled by SAM template
# This file only covers SQS queue monitoring for Terraform-managed resources

# SQS Queue Depth Alarm
resource "aws_cloudwatch_metric_alarm" "webhook_queue_high_depth" {
  alarm_name          = "${var.service_prefix}-webhook-queue-high-depth-${var.stage}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ApproximateNumberOfVisibleMessages"
  namespace           = "AWS/SQS"
  period              = "300"
  statistic           = "Average"
  threshold           = "100"
  alarm_description   = "This metric monitors webhook SQS queue depth"
  treat_missing_data  = "missing"

  dimensions = {
    QueueName = aws_sqs_queue.provider_webhook_events.name
  }

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    AlertType = "operational"
  })
}

# DLQ Messages Alarm (Critical - should always be zero)
resource "aws_cloudwatch_metric_alarm" "webhook_dlq_messages" {
  alarm_name          = "${var.service_prefix}-webhook-dlq-messages-${var.stage}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ApproximateNumberOfVisibleMessages"
  namespace           = "AWS/SQS"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "This metric monitors messages in webhook DLQ"
  treat_missing_data  = "missing"

  dimensions = {
    QueueName = aws_sqs_queue.provider_webhook_events_dlq.name
  }

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    AlertType = "critical"
  })
}

# CloudWatch Dashboard for SQS metrics
resource "aws_cloudwatch_dashboard" "webhook_sqs_dashboard" {
  dashboard_name = "${var.service_prefix}-webhook-sqs-dashboard-${var.stage}"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6

        properties = {
          metrics = [
            ["AWS/SQS", "ApproximateNumberOfVisibleMessages", "QueueName", aws_sqs_queue.provider_webhook_events.name],
            [".", "NumberOfMessagesSent", ".", "."],
            [".", "NumberOfMessagesReceived", ".", "."],
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "Webhook SQS Queue Metrics"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 0
        width  = 12
        height = 6

        properties = {
          metrics = [
            ["AWS/SQS", "ApproximateNumberOfVisibleMessages", "QueueName", aws_sqs_queue.provider_webhook_events_dlq.name],
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "Webhook DLQ Messages (Should be Zero)"
          period  = 300
          yAxis = {
            left = {
              min = 0
            }
          }
        }
      }
    ]
  })
}

# ===============================================
# Note about SAM-managed monitoring
# ===============================================

output "webhook_monitoring_note" {
  description = "Information about webhook monitoring coverage"
  value = "SQS monitoring managed by Terraform. API Gateway and Lambda monitoring managed by SAM template."
} 