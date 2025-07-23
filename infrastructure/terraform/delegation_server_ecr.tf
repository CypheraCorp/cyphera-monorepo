# Manages the ECR repository for the delegation server Docker images

resource "aws_ecr_repository" "delegation_server" {
  # Construct name using variables for consistency
  name                 = "${var.service_prefix}-delegation-server-${var.stage}"
  image_tag_mutability = "MUTABLE" # Allows overwriting tags like 'latest', change to IMMUTABLE for stricter versioning

  image_scanning_configuration {
    scan_on_push = true # Recommended for security
  }

  # Apply common tags defined in locals.tf or main.tf
  tags = merge(local.common_tags, {
    Name = "${var.service_prefix}-delegation-server-${var.stage}-ecr"
  })
}

output "delegation_server_ecr_repository_url" {
  description = "The URL of the ECR repository for the delegation server"
  value       = aws_ecr_repository.delegation_server.repository_url
}

output "delegation_server_ecr_repository_name" {
  description = "The Name of the ECR repository for the delegation server"
  value       = aws_ecr_repository.delegation_server.name
} 