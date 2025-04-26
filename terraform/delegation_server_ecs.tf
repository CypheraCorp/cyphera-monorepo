# Manages core ECS resources for the Delegation Server (Role, Logging)
# Cluster, Task Definition, and Service will be added in later steps.

# --- Data sources to get ARNs for Secrets/Parameters ---
data "aws_secretsmanager_secret" "delegation_private_key" {
  name = "cyphera/delegation-server/private-key-${var.stage}"
}

# --- IAM Role for ECS Task Execution ---
data "aws_iam_policy_document" "ecs_task_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "delegation_server_task_execution_role" {
  name               = "${var.service_prefix}-delegation-exec-role-${var.stage}"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "delegation_server_task_execution_policy" {
  role       = aws_iam_role.delegation_server_task_execution_role.name
  # Grants permissions to pull images from ECR and send logs to CloudWatch
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Add inline policy to allow fetching specific secrets/parameters
resource "aws_iam_role_policy" "delegation_server_fetch_config" {
  name = "${var.service_prefix}-delegation-fetch-config-${var.stage}"
  role = aws_iam_role.delegation_server_task_execution_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [data.aws_secretsmanager_secret.delegation_private_key.arn]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameters",
          "ssm:GetParameter"
        ]
        # Reference the specific parameter ARNs using resource attributes
        Resource = [
          aws_ssm_parameter.delegation_rpc_url.arn,
          aws_ssm_parameter.delegation_bundler_url.arn,
          aws_ssm_parameter.delegation_chain_id.arn
        ]
      }
    ]
  })
}

# --- CloudWatch Log Group ---
resource "aws_cloudwatch_log_group" "delegation_server" {
  name              = "/ecs/${var.service_prefix}-delegation-server-${var.stage}"
  retention_in_days = var.log_retention_days # Use a variable for retention period
  tags              = local.common_tags
}

# --- ECS Task Definition ---
resource "aws_ecs_task_definition" "delegation_server" {
  family                   = "${var.service_prefix}-delegation-server-${var.stage}"
  network_mode             = "awsvpc"         # Required for Fargate
  requires_compatibilities = ["FARGATE"]      # Specify Fargate compatibility
  # Use minimal resources for dev
  cpu                      = var.stage == "dev" ? "256" : "1024" # Example prod: 1 vCPU
  memory                   = var.stage == "dev" ? "512" : "2048" # Example prod: 2GB Memory
  execution_role_arn       = aws_iam_role.delegation_server_task_execution_role.arn
  # task_role_arn          = Optional: Define if the application itself needs AWS permissions

  # Define the container(s) for the task
  container_definitions = jsonencode([
    {
      name      = "${var.service_prefix}-delegation-server-${var.stage}" # Unique name for the container
      # Use the ECR repo URL. Tag will be specified/updated by CI/CD or Service definition.
      image     = "${aws_ecr_repository.delegation_server.repository_url}:latest" # Placeholder image, CI/CD updates service
      essential = true
      portMappings = [
        {
          containerPort = 50051 # Port exposed in Dockerfile and used by the app
          hostPort      = 50051 # Required for awsvpc
          protocol      = "tcp"
          name          = "${var.service_prefix}-ds-50051-tcp" # Optional: Name for the port mapping
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.delegation_server.name
          "awslogs-region"        = var.aws_region # Use region variable
          "awslogs-stream-prefix" = "ecs" # Prefix for log streams
        }
      }
      secrets = [ # Inject sensitive secrets
        {
          name      = "PRIVATE_KEY" # Env var name inside container
          valueFrom = data.aws_secretsmanager_secret.delegation_private_key.arn
        }
      ],
      environment = [ # Inject less sensitive config and non-secrets
        {
          name      = "RPC_URL"
          valueFrom = aws_ssm_parameter.delegation_rpc_url.arn
        },
        {
          name      = "BUNDLER_URL"
          valueFrom = aws_ssm_parameter.delegation_bundler_url.arn
        },
        {
          name      = "CHAIN_ID"
          valueFrom = aws_ssm_parameter.delegation_chain_id.arn
        },
        # Non-secret plain values
        { name = "GRPC_PORT", value = "50051" },
        { name = "GRPC_HOST", value = "0.0.0.0" },
        { name = "LOG_LEVEL", value = var.stage == "dev" ? "debug" : "info" }
      ]
    }
  ])

  tags = local.common_tags
}

# --- ECS Service ---
resource "aws_ecs_service" "delegation_server" {
  name            = "${var.service_prefix}-delegation-server-${var.stage}"
  cluster         = aws_ecs_cluster.delegation_server_cluster.id
  task_definition = aws_ecs_task_definition.delegation_server.arn
  launch_type     = "FARGATE"
  # Run only one task for dev
  desired_count   = var.stage == "dev" ? 1 : 2 # Example prod: 2 tasks for HA

  # Configure networking to place tasks in private subnets
  network_configuration {
    subnets = module.vpc.private_subnets
    security_groups = [
      aws_security_group.delegation_server_task.id
    ]
    assign_public_ip = false
  }

  # Connect the service to the Application Load Balancer
  load_balancer {
    target_group_arn = aws_lb_target_group.delegation_server.arn
    container_name   = "${var.service_prefix}-delegation-server-${var.stage}"
    container_port   = 50051
  }

  # Wait for the service to stabilize after deployments triggered by ALB health checks
  health_check_grace_period_seconds = 120 # Give tasks time to start and pass health checks

  # Prevent Terraform from replacing the service just because the task definition changes
  # (CI/CD will handle updating the service with the new task definition revision)
  lifecycle {
    ignore_changes = [task_definition]
  }

  # Ensure ALB resources are created before the service
  depends_on = [
    aws_lb_listener.delegation_server_grpc,
    aws_iam_role_policy_attachment.delegation_server_task_execution_policy
  ]

  tags = local.common_tags
} 