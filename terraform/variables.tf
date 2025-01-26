variable "aws_region" {
  description = "AWS region"
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name"
  default     = "development"
}

variable "app_name" {
  description = "Application name"
  default     = "cyphera-api"
}

variable "db_username" {
  description = "Database username"
  default     = "apiuser"
}

variable "db_password" {
  description = "Database password"
  sensitive   = true
} 