terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"  # Use an appropriate version
    }
  }
}

provider "aws" {
  region = "us-east-1"
} 