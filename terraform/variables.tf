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
  default     = "cyphera"
}

variable "db_username" {
  description = "Database username"
  default     = "apiuser"
}

variable "db_password" {
  description = "Database password"
  sensitive   = true
}

variable "nate_machine_ip" {
  description = "Development machine IP address for RDS access"
  default     = "151.204.139.74/32"  # Your current IP
} 