resource "aws_acm_certificate" "wildcard_api" {
  domain_name       = "*.api.cypherapay.com"
  validation_method = "DNS"

  # We are not managing validation records in this TF state currently
  # We assume the certificate is already validated externally or previously.
  # Including lifecycle block prevents Terraform from trying to recreate
  # the certificate if minor changes occur outside its control.
  lifecycle {
    ignore_changes = [
      # Ignore changes to validation options as we aren't managing validation records here
      # This prevents errors if the validation CNAME changes externally
      # validation_options,
    ]
    prevent_destroy = false # Set to true in prod if you want extra safety
  }

  tags = merge(local.common_tags, {
    Name = "wildcard-api-cypherapay-com"
  })
}

resource "aws_acm_certificate" "dev_app" {
  domain_name       = "dev-app.cypherapay.com"
  validation_method = "DNS"

  # Assume already validated, similar lifecycle policy
  lifecycle {
    ignore_changes = [
      # validation_options,
    ]
    prevent_destroy = false # Set to true in prod if desired
  }

  tags = merge(local.common_tags, {
    Name = "dev-app-cypherapay-com"
  })
}

resource "aws_acm_certificate" "root_api" {
  domain_name       = "api.cypherapay.com"
  validation_method = "DNS"

  # Assume already validated, similar lifecycle policy
  lifecycle {
    ignore_changes = [
      # validation_options,
    ]
    prevent_destroy = false # Set to true in prod if desired
  }

  tags = merge(local.common_tags, {
    Name = "api-cypherapay-com"
  })
} 