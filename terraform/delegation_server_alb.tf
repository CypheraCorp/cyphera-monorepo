# Manages the Application Load Balancer and associated resources for the Delegation Server

# --- Security Groups ---

# Security Group for the ALB
resource "aws_security_group" "delegation_server_alb" {
  name        = "${var.service_prefix}-ds-alb-sg-${var.stage}"
  description = "Allow gRPC traffic from Lambda to Delegation Server ALB"
  vpc_id      = module.vpc.vpc_id # Use VPC module output

  # Allow inbound gRPC (port 50051) from Lambda Security Group
  ingress {
    description     = "Allow gRPC from Lambda Functions"
    from_port       = 50051
    to_port         = 50051
    protocol        = "tcp"
    security_groups = [aws_security_group.lambda.id] # Reference Lambda SG defined in main.tf
  }

  # Allow all outbound (Can be restricted if needed, but generally okay for ALB)
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${var.service_prefix}-ds-alb-sg-${var.stage}"
  })
}

# Security Group for the Fargate Tasks
resource "aws_security_group" "delegation_server_task" {
  name        = "${var.service_prefix}-ds-task-sg-${var.stage}"
  description = "Allow gRPC traffic from ALB and potentially egress to RDS"
  vpc_id      = module.vpc.vpc_id # Use VPC module output

  # Allow inbound gRPC (port 50051) ONLY from the ALB Security Group
  ingress {
    description     = "Allow gRPC from ALB"
    from_port       = 50051
    to_port         = 50051
    protocol        = "tcp"
    security_groups = [aws_security_group.delegation_server_alb.id] # Reference the ALB SG
  }

  # Allow all outbound (Needed for pulling images, CloudWatch, etc.)
  # If delegation server needs to talk to RDS, add specific egress rule below
  egress {
    description = "Allow all outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Example: Allow egress to RDS if needed by delegation server
  # egress {
  #   description     = "Allow outbound traffic to RDS"
  #   from_port       = 5432
  #   to_port         = 5432
  #   protocol        = "tcp"
  #   security_groups = [aws_security_group.rds.id] # Reference RDS SG from rds.tf
  # }

  tags = merge(local.common_tags, {
    Name = "${var.service_prefix}-ds-task-sg-${var.stage}"
  })
}

# --- Application Load Balancer --- 
resource "aws_lb" "delegation_server" {
  name               = "${var.service_prefix}-ds-alb-${var.stage}"
  internal           = true # Crucial: Keep it private within the VPC
  load_balancer_type = "application"
  security_groups    = [aws_security_group.delegation_server_alb.id] # Attach ALB SG
  subnets            = module.vpc.private_subnets # Place ALB in private subnets

  enable_deletion_protection = false # Set to true for prod if desired
  tags                       = local.common_tags
}

# --- ALB Target Group --- 
resource "aws_lb_target_group" "delegation_server" {
  name        = "${var.service_prefix}-ds-tg-${var.stage}"
  port        = 50051    # Port tasks listen on
  protocol    = "HTTP"     # ALB talks HTTP to tasks for gRPC health checks
  target_type = "ip"       # Required for Fargate
  vpc_id      = module.vpc.vpc_id

  # gRPC Health Check (requires server implementation: https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
  health_check {
    enabled             = true
    interval            = 30
    path                = "/grpc.health.v1.Health/Check" # Standard gRPC path
    protocol            = "HTTP" # Health checks use HTTP
    matcher             = "0-19" # Match gRPC status codes (0 = OK, others might be unready but not unhealthy for ALB)
    healthy_threshold   = 2
    unhealthy_threshold = 3 # Increase slightly to tolerate transient issues
    timeout             = 10
    port                = "traffic-port"
  }

  # Enable gRPC support on the target group
  protocol_version = "GRPC"

  tags = local.common_tags
}

# --- ALB Listener --- 
resource "aws_lb_listener" "delegation_server_grpc" {
  load_balancer_arn = aws_lb.delegation_server.arn
  # Use standard HTTPS port
  port              = 443
  # Use HTTPS protocol for gRPC via ALB
  protocol          = "HTTPS"
  # Attach the wildcard certificate managed in acm.tf
  certificate_arn   = aws_acm_certificate.wildcard_api.arn 

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.delegation_server.arn
  }
}

# --- Outputs --- 
output "delegation_server_alb_dns_name" {
  description = "The private DNS name of the Delegation Server ALB (use this in Lambda env var)"
  value       = aws_lb.delegation_server.dns_name
}

output "delegation_server_alb_zone_id" {
  description = "The zone ID of the Delegation Server ALB (for Route 53 alias records if needed)"
  value       = aws_lb.delegation_server.zone_id
} 