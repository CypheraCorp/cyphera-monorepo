# ===============================================
# Multi-Provider Webhook Monitoring & Alerting
# ===============================================

# CloudWatch Log Groups for Webhook API Gateway
# (Note: Lambda log groups are managed by SAM)

# CloudWatch Alarms for SQS Queue Monitoring
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
  alarm_actions       = []

  dimensions = {
    QueueName = aws_sqs_queue.provider_webhook_events.name
  }

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    AlertType = "operational"
  })
}

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
  alarm_actions       = []

  dimensions = {
    QueueName = aws_sqs_queue.provider_webhook_events_dlq.name
  }

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    AlertType = "critical"
  })
}

# API Gateway Monitoring
resource "aws_cloudwatch_metric_alarm" "webhook_api_4xx_errors" {
  alarm_name          = "${var.service_prefix}-webhook-api-4xx-errors-${var.stage}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "4XXError"
  namespace           = "AWS/ApiGateway"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "This metric monitors 4XX errors from webhook API Gateway"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ApiName = aws_api_gateway_rest_api.webhook_api.name
    Stage   = var.stage
  }

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    AlertType = "operational"
  })
}

resource "aws_cloudwatch_metric_alarm" "webhook_api_5xx_errors" {
  alarm_name          = "${var.service_prefix}-webhook-api-5xx-errors-${var.stage}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "5XXError"
  namespace           = "AWS/ApiGateway"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "This metric monitors 5XX errors from webhook API Gateway"
  treat_missing_data  = "notBreaching"

  dimensions = {
    ApiName = aws_api_gateway_rest_api.webhook_api.name
    Stage   = var.stage
  }

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
    AlertType = "critical"
  })
}

# Custom CloudWatch Dashboard for Webhook Infrastructure
resource "aws_cloudwatch_dashboard" "webhook_dashboard" {
  dashboard_name = "${var.service_prefix}-webhook-dashboard-${var.stage}"

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
            [".", "NumberOfMessagesReceived", ".", "."]
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
            ["AWS/ApiGateway", "Count", "ApiName", aws_api_gateway_rest_api.webhook_api.name, "Stage", var.stage],
            [".", "4XXError", ".", ".", ".", "."],
            [".", "5XXError", ".", ".", ".", "."],
            [".", "Latency", ".", ".", ".", "."]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "API Gateway Metrics"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 24
        height = 6

        properties = {
          metrics = [
            ["AWS/SQS", "ApproximateNumberOfVisibleMessages", "QueueName", aws_sqs_queue.provider_webhook_events_dlq.name]
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

# Log Insights Saved Queries for troubleshooting
resource "aws_cloudwatch_query_definition" "webhook_api_errors" {
  name = "${var.service_prefix}-webhook-api-errors-${var.stage}"

  log_group_names = [
    aws_cloudwatch_log_group.webhook_api_logs.name
  ]

  query_string = <<EOF
fields @timestamp, @message, @requestId
| filter @message like /ERROR/
| sort @timestamp desc
| limit 100
EOF
}

resource "aws_cloudwatch_query_definition" "webhook_provider_analysis" {
  name = "${var.service_prefix}-webhook-provider-analysis-${var.stage}"

  log_group_names = [
    aws_cloudwatch_log_group.webhook_api_logs.name
  ]

  query_string = <<EOF
fields @timestamp, resourcePath, httpMethod, status, responseLength
| filter resourcePath like /webhooks/
| stats count() by resourcePath
| sort count desc
EOF
} 