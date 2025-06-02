# ===============================================
# Multi-Provider Webhook API Gateway (SAM Integration)
# ===============================================

# Data sources to reference SAM-deployed Lambda functions
data "aws_lambda_function" "webhook_receiver" {
  function_name = "${var.service_prefix}-webhook-receiver-${var.stage}"
}

data "aws_lambda_function" "webhook_processor" {
  function_name = "${var.service_prefix}-webhook-processor-${var.stage}"
}

# REST API Gateway for webhook endpoints
resource "aws_api_gateway_rest_api" "webhook_api" {
  name        = "${var.service_prefix}-webhook-api-${var.stage}"
  description = "Multi-provider webhook endpoints for ${var.stage}"

  endpoint_configuration {
    types = ["EDGE"]  # Edge-optimized for global webhook endpoints
  }

  # Enable binary media types if needed for webhook payloads
  binary_media_types = [
    "application/octet-stream",
    "application/json"
  ]

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
  })
}

# Root resource for webhook API
resource "aws_api_gateway_resource" "webhook_root" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  parent_id   = aws_api_gateway_rest_api.webhook_api.root_resource_id
  path_part   = "webhooks"
}

# Provider resource (for /webhooks/{provider})
resource "aws_api_gateway_resource" "webhook_provider" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  parent_id   = aws_api_gateway_resource.webhook_root.id
  path_part   = "{provider}"
}

# POST method for webhook endpoint
resource "aws_api_gateway_method" "webhook_post" {
  rest_api_id   = aws_api_gateway_rest_api.webhook_api.id
  resource_id   = aws_api_gateway_resource.webhook_provider.id
  http_method   = "POST"
  authorization = "NONE"

  request_parameters = {
    "method.request.path.provider" = true
  }
}

# Lambda integration for webhook receiver
resource "aws_api_gateway_integration" "webhook_lambda_integration" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_post.http_method

  integration_http_method = "POST"
  type                   = "AWS_PROXY"
  uri                    = data.aws_lambda_function.webhook_receiver.invoke_arn

  request_parameters = {
    "integration.request.path.provider" = "method.request.path.provider"
  }
}

# Method responses
resource "aws_api_gateway_method_response" "webhook_200" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_post.http_method
  status_code = "200"

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_method_response" "webhook_400" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_post.http_method
  status_code = "400"

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_method_response" "webhook_500" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_post.http_method
  status_code = "500"

  response_models = {
    "application/json" = "Empty"
  }
}

# Integration response
resource "aws_api_gateway_integration_response" "webhook_200" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_post.http_method
  status_code = aws_api_gateway_method_response.webhook_200.status_code

  depends_on = [aws_api_gateway_integration.webhook_lambda_integration]
}

# CORS preflight OPTIONS method
resource "aws_api_gateway_method" "webhook_options" {
  rest_api_id   = aws_api_gateway_rest_api.webhook_api.id
  resource_id   = aws_api_gateway_resource.webhook_provider.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "webhook_options_integration" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_options.http_method

  type = "MOCK"
  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "webhook_options_200" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_options.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }
}

resource "aws_api_gateway_integration_response" "webhook_options_200" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id
  resource_id = aws_api_gateway_resource.webhook_provider.id
  http_method = aws_api_gateway_method.webhook_options.http_method
  status_code = aws_api_gateway_method_response.webhook_options_200.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,X-Workspace-ID'"
    "method.response.header.Access-Control-Allow-Methods" = "'OPTIONS,POST'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }

  depends_on = [aws_api_gateway_integration.webhook_options_integration]
}

# API Gateway deployment
resource "aws_api_gateway_deployment" "webhook_deployment" {
  rest_api_id = aws_api_gateway_rest_api.webhook_api.id

  depends_on = [
    aws_api_gateway_integration.webhook_lambda_integration,
    aws_api_gateway_integration.webhook_options_integration,
    aws_api_gateway_method.webhook_post,
    aws_api_gateway_method.webhook_options
  ]

  # Force redeployment when configuration changes
  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.webhook_root.id,
      aws_api_gateway_resource.webhook_provider.id,
      aws_api_gateway_method.webhook_post.id,
      aws_api_gateway_method.webhook_options.id,
      aws_api_gateway_integration.webhook_lambda_integration.id,
      aws_api_gateway_integration.webhook_options_integration.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

# CloudWatch Log Group for API Gateway
resource "aws_cloudwatch_log_group" "webhook_api_logs" {
  name              = "/aws/apigateway/${var.service_prefix}-webhook-api-${var.stage}"
  retention_in_days = var.log_retention_days

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
  })
}

# API Gateway stage
resource "aws_api_gateway_stage" "webhook_stage" {
  deployment_id = aws_api_gateway_deployment.webhook_deployment.id
  rest_api_id   = aws_api_gateway_rest_api.webhook_api.id
  stage_name    = var.stage

  # Enable CloudWatch logging
  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.webhook_api_logs.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      ip             = "$context.identity.sourceIp"
      caller         = "$context.identity.caller"
      user           = "$context.identity.user"
      requestTime    = "$context.requestTime"
      httpMethod     = "$context.httpMethod"
      resourcePath   = "$context.resourcePath"
      status         = "$context.status"
      protocol       = "$context.protocol"
      responseLength = "$context.responseLength"
    })
  }

  xray_tracing_enabled = true

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
  })
}

# CloudWatch role for API Gateway
resource "aws_api_gateway_account" "webhook_api_account" {
  cloudwatch_role_arn = aws_iam_role.api_gateway_cloudwatch_role.arn
}

resource "aws_iam_role" "api_gateway_cloudwatch_role" {
  name = "${var.service_prefix}-api-gateway-cloudwatch-role-${var.stage}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "apigateway.amazonaws.com"
        }
      }
    ]
  })

  tags = merge(local.common_tags, {
    Component = "webhook-infrastructure"
  })
}

resource "aws_iam_role_policy_attachment" "api_gateway_cloudwatch" {
  role       = aws_iam_role.api_gateway_cloudwatch_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
}

# Lambda permission for API Gateway to invoke webhook receiver
resource "aws_lambda_permission" "webhook_receiver_api_gateway" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = data.aws_lambda_function.webhook_receiver.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.webhook_api.execution_arn}/*/*/*"
} 