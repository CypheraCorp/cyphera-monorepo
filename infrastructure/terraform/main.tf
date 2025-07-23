// Terraform configuration

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = "${var.app_name}-vpc"
  cidr = "10.0.0.0/16"

  azs             = ["us-east-1a", "us-east-1b"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]

  # Enable NAT Gateway for dev (using single for cost), enable for prod
  enable_nat_gateway = true # Always true now based on requirement
  # If NAT Gateway is enabled use single NAT gateway for cost saving, especially in dev
  single_nat_gateway = true # Always true to prefer single NAT

  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Environment = var.environment
    App         = var.app_name
  }
}

# Lambda Security Group
resource "aws_security_group" "lambda" {
  name        = "${var.app_name}-lambda-sg"
  description = "Security group for Lambda functions"
  vpc_id      = module.vpc.vpc_id

  # Default egress: Allow all outbound
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Add specific egress rule for Delegation Server ALB (Optional but Recommended)
  # Alternatively, the default egress rule above covers this.
  # Keeping the default is simpler, but adding specific rule is more secure if you remove default.
  # egress {
  #   description     = "Allow outbound gRPC to Delegation Server ALB"
  #   from_port       = 50051
  #   to_port         = 50051
  #   protocol        = "tcp"
  #   security_groups = [aws_security_group.delegation_server_alb.id] # Reference ALB SG (needs depends_on if removing default egress)
  # }

  tags = {
    Name        = "${var.app_name}-lambda-sg"
    Environment = var.environment
    App         = var.app_name
  }
}

# --- Add depends_on if adding specific egress and removing default ---
# resource "aws_security_group_rule" "lambda_egress_to_ds_alb" {
#   type                     = "egress"
#   from_port                = 50051
#   to_port                  = 50051
#   protocol                 = "tcp"
#   source_security_group_id = aws_security_group.delegation_server_alb.id
#   security_group_id        = aws_security_group.lambda.id
#   description              = "Allow outbound gRPC to Delegation Server ALB"
# }

locals {
  # Common tags applied to all resources for organization and cost tracking
  common_tags = {
    Project     = var.service_prefix
    Environment = var.stage
    ManagedBy   = "Terraform"
  }
}

# --- ECS Cluster --- 
# Define a cluster for the delegation server workloads.
# If you have an existing cluster you want to reuse, replace this resource
# with a data source: data "aws_ecs_cluster" "existing" { name = "your-cluster-name" }
resource "aws_ecs_cluster" "delegation_server_cluster" {
  name = "${var.service_prefix}-delegation-cluster-${var.stage}"
  tags = local.common_tags

  setting {
    name  = "containerInsights"
    value = "enabled" # Enable enhanced monitoring
  }
}

# Existing VPC Peering Connection (Keep if used, ensure correct vpc_id)
# data "aws_vpc_peering_connection" "default" {
#   vpc_id    = module.vpc.vpc_id # Corrected reference
#   peer_vpc_id = "vpc-0f048363b051a1a62"
# }

# --- REMOVED: IAM Policy/Attachment for Lambda to access RDS Secret ---
# This permission is now managed within serverless.yml under provider.iam.role.statements
# data "aws_iam_policy_document" "lambda_read_rds_secret" { ... }
# resource "aws_iam_policy" "lambda_read_rds_secret" { ... }
# resource "aws_iam_role_policy_attachment" "lambda_read_rds_secret" { ... }

# --- S3 Bucket for SAM Deployments ---
# Bucket to store SAM deployment artifacts (e.g., packaged Lambda code)
resource "aws_s3_bucket" "sam_deployment_bucket" {
  bucket = "cyphera-api-sam-deployments-${var.stage}" # Using stage variable for uniqueness per env

  tags = local.common_tags
}

resource "aws_s3_bucket_versioning" "sam_deployment_bucket_versioning" {
  bucket = aws_s3_bucket.sam_deployment_bucket.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "sam_deployment_bucket_sse" {
  bucket = aws_s3_bucket.sam_deployment_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "sam_deployment_bucket_public_access" {
  bucket = aws_s3_bucket.sam_deployment_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
} 