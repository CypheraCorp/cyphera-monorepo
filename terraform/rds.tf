resource "aws_db_subnet_group" "main" {
  name       = "${var.app_name}-db-subnet-private"
  subnet_ids = module.vpc.private_subnets

  tags = {
    Name = "${var.app_name}-db-subnet-private"
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
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${var.app_name}-rds-sg"
    Environment = var.environment
    App         = var.app_name
  }
}

resource "aws_db_instance" "main" {
  identifier        = "${var.app_name}-db-${var.stage}"
  engine           = "postgres"
  engine_version   = "15.10"
  # Use smaller instance for dev
  instance_class   = var.stage == "dev" ? "db.t3.micro" : "db.t4g.micro" 
  # Use minimal storage for dev
  allocated_storage = var.stage == "dev" ? 20 : 50 # Example prod size: 50GB
  storage_type      = "gp3"
  # Prevent auto-scaling for dev to cap costs
  max_allocated_storage = var.stage == "dev" ? 20 : 100 # Example prod max: 100GB

  db_name  = var.db_name
  username = var.db_master_username

  manage_master_user_password = true
  # master_user_secret_kms_key_id = null # Keep if you have it

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # --- Production vs Non-Production Settings ---
  # Apply production settings only if var.stage is "prod"
  backup_retention_period = var.stage == "prod" ? var.prod_backup_retention_period : 0
  multi_az              = var.stage == "prod" ? true : false
  skip_final_snapshot    = var.stage == "prod" ? false : true
  deletion_protection = var.stage == "prod" ? true : false
  # ---------------------------------------------

  publicly_accessible    = false
  storage_encrypted     = true
  # Disable Performance Insights for dev
  performance_insights_enabled = var.stage == "dev" ? false : true
  # Conditionally set retention period (only relevant if enabled)
  performance_insights_retention_period = var.stage == "dev" ? null : 7 
  auto_minor_version_upgrade = true
  maintenance_window = "Sun:03:00-Sun:04:00"

  tags = merge(local.common_tags, {
    Name = "${var.service_prefix}-rds-${var.stage}"
  })

  # Remove depends_on as the secret resource is being removed
  lifecycle {
    ignore_changes = [password]
  }
} 