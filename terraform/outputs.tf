resource "aws_ssm_parameter" "lambda_sg" {
  name  = "/cyphera/lambda-security-group-id"
  type  = "String"
  value = aws_security_group.lambda.id
}

resource "aws_ssm_parameter" "private_subnet_1" {
  name  = "/cyphera/private-subnet-1"
  type  = "String"
  value = module.vpc.private_subnets[0]
}

resource "aws_ssm_parameter" "private_subnet_2" {
  name  = "/cyphera/private-subnet-2"
  type  = "String"
  value = module.vpc.private_subnets[1]
} 