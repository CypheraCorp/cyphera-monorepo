package responses

// TokenResponse represents the standardized API response for token operations
type TokenResponse struct {
	ID              string `json:"id"`
	Object          string `json:"object"`
	NetworkID       string `json:"network_id"`
	GasToken        bool   `json:"gas_token"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	ContractAddress string `json:"contract_address"`
	Decimals        int32  `json:"decimals"`
	Active          bool   `json:"active"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
	DeletedAt       *int64 `json:"deleted_at,omitempty"`
}

// ListTokensResponse represents a paginated list of tokens
type ListTokensResponse struct {
	Object  string          `json:"object"`
	Data    []TokenResponse `json:"data"`
	HasMore bool            `json:"has_more,omitempty"`
	Total   int64           `json:"total,omitempty"`
}

// GetTokenQuoteResponse represents a token quote response
type GetTokenQuoteResponse struct {
	FiatSymbol        string  `json:"fiat_symbol"`
	TokenSymbol       string  `json:"token_symbol"`
	TokenAmountInFiat float64 `json:"token_amount_in_fiat"`
}

// TokenQuoteResult represents the result of a token quote from services
type TokenQuoteResult struct {
	TokenAmount   string  `json:"token_amount"`  // Amount in token units
	FiatAmount    float64 `json:"fiat_amount"`   // Amount in fiat
	ExchangeRate  float64 `json:"exchange_rate"` // Token price in fiat
	TokenDecimals int32   `json:"token_decimals"`
	QuotedAt      string  `json:"quoted_at"`
	ExpiresAt     string  `json:"expires_at"`
	PriceSource   string  `json:"price_source"` // "coingecko", "chainlink", etc
}
