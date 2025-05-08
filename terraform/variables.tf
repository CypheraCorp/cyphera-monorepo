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

variable "db_master_username" {
  description = "Master username for the RDS database (password managed by Secrets Manager)"
  type        = string
  default     = "postgres" # Or another standard name like 'masteruser'
}

variable "db_name" {
  description = "The name of the initial database created in the RDS instance"
  type        = string
  default     = "cyphera" # Default name
}

variable "prod_backup_retention_period" {
  description = "Backup retention period in days for the production RDS instance"
  type        = number
  default     = 7 # Default to 7 days for prod backups
}

variable "nate_machine_ip" {
  description = "Development machine IP address for RDS access"
  default     = "151.204.139.74/32"  # Your current IP
}

variable "service_prefix" {
  description = "Prefix for naming resources (e.g., 'cyphera')"
  type        = string
  default     = "cyphera" # Or set based on your naming convention
}

variable "stage" {
  description = "Deployment stage (e.g., 'dev', 'prod')"
  type        = string
  # Default can be set, or passed via TF_VAR_stage or terraform.tfvars
}

variable "log_retention_days" {
  description = "Number of days to retain CloudWatch logs"
  type        = number
  default     = 7
}

variable "circle_api_key_value" {
  description = "The value for the Circle API Key (only used for initial set, then ignored)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "coin_market_cap_api_key_value" {
  description = "The value for the CoinMarketCap API Key (only used for initial set, then ignored)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "infura_api_key_value" {
  description = "The value for the Infura API Key (only used for initial set, then ignored)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "pimlico_api_key_value" {
  description = "The value for the Pimlico API Key (only used for initial set, then ignored)"
  type        = string
  sensitive   = true
  default     = ""
}

variable "delegation_private_key_value" {
  description = "The actual private key string for the delegation server. Used to create the secret in AWS Secrets Manager."
  type        = string
  sensitive   = true
  default     = ""
}

# ACM / Route53 Variables 