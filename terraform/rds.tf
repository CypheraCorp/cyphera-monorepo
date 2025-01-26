resource "aws_db_subnet_group" "main" {
  name       = "${var.app_name}-db-subnet"
  subnet_ids = module.vpc.private_subnets

  tags = {
    Name = "${var.app_name}-db-subnet"
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
  allocated_storage = 20  # Minimum storage for gp3
  storage_type      = "gp3"  # More cost-effective than gp2
  max_allocated_storage = 20  # Disable storage autoscaling to control costs

  db_name  = "cyphera"
  username = var.db_username
  password = var.db_password

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # Cost optimization settings
  backup_retention_period = 0  # No automated backups for dev/test
  skip_final_snapshot    = true
  multi_az              = false
  publicly_accessible    = false
  storage_encrypted     = true  # Important for security, minimal cost impact

  # Performance Insights is free for 7 days retention
  performance_insights_enabled = true
  performance_insights_retention_period = 7

  # Auto minor version upgrades for security
  auto_minor_version_upgrade = true

  # Maintenance window during off-hours
  maintenance_window = "Sun:03:00-Sun:04:00"

  # Cost optimization: Stop database during non-business hours
  deletion_protection = false  # Allow stopping/starting for cost savings

  tags = {
    Environment = var.environment
    App         = var.app_name
  }
} 