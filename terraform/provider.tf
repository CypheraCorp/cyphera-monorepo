terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.96.0" # Pin to the specific v5 version we are targeting
    }
  }
  backend "s3" {
    bucket = "cyphera-terraform-state"
    key    = "cyphera-api/terraform.tfstate"
    region = "us-east-1"
  }
  required_version = ">= 1.0"
} 

# Configure the AWS provider
provider "aws" {
  region = var.aws_region
} 