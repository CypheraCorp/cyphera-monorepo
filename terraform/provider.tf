terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.40"
    }
  }
  backend "s3" {
    bucket  = "cyphera-terraform-state"
    key     = "cyphera-api/terraform.tfstate"
    region  = "us-east-1"
    encrypt = true # Enable server-side encryption for the state file
  }
  required_version = ">= 1.0"
}

# Configure the AWS provider
provider "aws" {
  region = var.aws_region
} 