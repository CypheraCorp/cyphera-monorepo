resource "aws_db_subnet_group" "main" {
  name       = "${var.app_name}-db-subnet-private"
  subnet_ids = module.vpc.private_subnets

  tags = {
    Name = "${var.app_name}-db-subnet-private"
  }
}

resource "aws_db_subnet_group" "public" {
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

  # Ingress rules should be managed by separate aws_security_group_rule resources
  # to allow for conditional logic (like dev-only public access).
  # Therefore, we ignore changes to the inline ingress block here.
  lifecycle {
    ignore_changes = [
      ingress, # Ignore inline ingress rules
    ]
  }

  # This ingress rule (allowing Lambda access) should also be moved
  # to a separate aws_security_group_rule resource for consistency.
  # Keeping it here temporarily might work, but moving it is cleaner.
  ingress {
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.lambda.id]
    description     = "PostgreSQL access from Lambda"
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

resource "aws_security_group_rule" "rds_public_ingress_dev" {
  count = var.stage == "dev" ? 1 : 0

  type              = "ingress"
  from_port         = 5432
  to_port           = 5432
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.rds.id
  description       = "Public PostgreSQL access for dev"
}

resource "aws_db_instance" "main" {
  identifier        = "${var.app_name}-db-${var.stage}"
  engine           = "postgres"
  engine_version   = "15.10"
  instance_class   = var.stage == "dev" ? "db.t3.micro" : "db.t4g.micro" 
  allocated_storage = var.stage == "dev" ? 20 : 50
  storage_type      = "gp3"
  max_allocated_storage = var.stage == "dev" ? 20 : 100

  db_name  = var.db_name
  username = var.db_master_username

  manage_master_user_password = true

  # STEP 2: Uncomment the subnet group assignment
  db_subnet_group_name   = var.stage == "dev" ? aws_db_subnet_group.public.name : aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # --- Production vs Non-Production Settings ---
  backup_retention_period = var.stage == "prod" ? var.prod_backup_retention_period : 0
  multi_az              = var.stage == "prod" ? true : false
  skip_final_snapshot    = var.stage == "prod" ? false : true
  deletion_protection = var.stage == "prod" ? true : false
  # ---------------------------------------------

  # Set public access based on stage (This is the only change we want in Step 1)
  publicly_accessible    = var.stage == "dev" ? true : false
  storage_encrypted     = true
  performance_insights_enabled = var.stage == "dev" ? false : true
  performance_insights_retention_period = var.stage == "dev" ? null : 7 
  auto_minor_version_upgrade = true
  maintenance_window = "Sun:03:00-Sun:04:00"

  tags = merge(local.common_tags, {
    Name = "${var.service_prefix}-rds-${var.stage}"
  })

  lifecycle {
    ignore_changes = [password]
  }
} 