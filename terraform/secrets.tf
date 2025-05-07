# --- Secrets for cyphera-api ---

resource "aws_secretsmanager_secret" "circle_api_key" {
  name        = "/cyphera/cyphera-api/circle-api-key-${var.stage}"
  description = "Circle API Key for Cyphera API - ${var.stage}"
  tags = {
    Environment = var.stage
    Project     = "Cyphera"
    Service     = "cyphera-api"
  }
}

resource "aws_secretsmanager_secret_version" "circle_api_key_version" {
  secret_id     = aws_secretsmanager_secret.circle_api_key.id
  secret_string = var.circle_api_key_value

  lifecycle {
    ignore_changes = [
      secret_string,
    ]
  }
}

resource "aws_secretsmanager_secret" "coin_market_cap_api_key" {
  name        = "/cyphera/cyphera-api/coin-market-cap-api-key-${var.stage}"
  description = "CoinMarketCap API Key for Cyphera API - ${var.stage}"
  tags = {
    Environment = var.stage
    Project     = "Cyphera"
    Service     = "cyphera-api"
  }
}

resource "aws_secretsmanager_secret_version" "coin_market_cap_api_key_version" {
  secret_id     = aws_secretsmanager_secret.coin_market_cap_api_key.id
  secret_string = var.coin_market_cap_api_key_value

  lifecycle {
    ignore_changes = [
      secret_string,
    ]
  }
}

# --- Secrets for delegation-server ---

resource "aws_secretsmanager_secret" "infura_api_key" {
  name        = "/cyphera/delegation-server/infura-api-key-${var.stage}"
  description = "Infura API Key for Delegation Server - ${var.stage}"
  tags = {
    Environment = var.stage
    Project     = "Cyphera"
    Service     = "delegation-server"
  }
}

resource "aws_secretsmanager_secret_version" "infura_api_key_version" {
  secret_id     = aws_secretsmanager_secret.infura_api_key.id
  secret_string = var.infura_api_key_value

  lifecycle {
    ignore_changes = [
      secret_string,
    ]
  }
}

resource "aws_secretsmanager_secret" "pimlico_api_key" {
  name        = "/cyphera/delegation-server/pimlico-api-key-${var.stage}"
  description = "Pimlico API Key for Delegation Server - ${var.stage}"
  tags = {
    Environment = var.stage
    Project     = "Cyphera"
    Service     = "delegation-server"
  }
}

resource "aws_secretsmanager_secret_version" "pimlico_api_key_version" {
  secret_id     = aws_secretsmanager_secret.pimlico_api_key.id
  secret_string = var.pimlico_api_key_value

  lifecycle {
    ignore_changes = [
      secret_string,
    ]
  }
} 