resource "aws_db_subnet_group" "main" {
  name       = "${var.app_name}-db-subnet-public"
  subnet_ids = module.vpc.public_subnets

  tags = {
    Name = "${var.app_name}-db-subnet-public"
  }
}

resource "aws_security_group" "rds" {
  name        = "${var.app_name}-rds-sg"
  description = "RDS security group"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.lambda.id]
  }

  # Allow access from development machine
  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [var.nate_machine_ip]
    description = "PostgreSQL access from development machine"
  }

  egress {
    from_port       = 0
    to_port         = 0
    protocol        = "-1"
    security_groups = [aws_security_group.lambda.id]
  }

  tags = {
    Name        = "${var.app_name}-rds-sg"
    Environment = var.environment
    App         = var.app_name
  }
}

resource "aws_db_instance" "main" {
  identifier        = "${var.app_name}-db"
  engine           = "postgres"
  engine_version   = "15.10"
  instance_class   = "db.t4g.micro"
  allocated_storage = 20
  storage_type      = "gp3"
  max_allocated_storage = 20

  db_name  = "cyphera"
  username = var.db_username
  password = var.db_password

  # Using the new public subnet group
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # Cost optimization settings
  backup_retention_period = 0
  skip_final_snapshot    = true
  multi_az              = false
  publicly_accessible    = true
  storage_encrypted     = true

  # Performance Insights settings
  performance_insights_enabled = true
  performance_insights_retention_period = 7

  auto_minor_version_upgrade = true
  maintenance_window = "Sun:03:00-Sun:04:00"
  deletion_protection = false

  tags = {
    Environment = var.environment
    App         = var.app_name
  }
} 