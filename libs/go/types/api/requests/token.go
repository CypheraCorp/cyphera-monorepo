package requests

// CreateTokenRequest represents the request body for creating a token
type CreateTokenRequest struct {
	NetworkID       string `json:"network_id" binding:"required"`
	GasToken        bool   `json:"gas_token"`
	Name            string `json:"name" binding:"required"`
	Symbol          string `json:"symbol" binding:"required"`
	ContractAddress string `json:"contract_address" binding:"required"`
	Decimals        int32  `json:"decimals" binding:"required,gte=0"`
	Active          bool   `json:"active"`
}

// UpdateTokenRequest represents the request body for updating a token
type UpdateTokenRequest struct {
	Name            string `json:"name,omitempty"`
	Symbol          string `json:"symbol,omitempty"`
	ContractAddress string `json:"contract_address,omitempty"`
	Decimals        *int32 `json:"decimals,omitempty,gte=0"`
	GasToken        *bool  `json:"gas_token,omitempty"`
	Active          *bool  `json:"active,omitempty"`
}

// GetTokenQuoteRequest represents the request for getting a token quote
type GetTokenQuoteRequest struct {
	TokenID    string `json:"token_id" binding:"required,uuid"`
	NetworkID  string `json:"network_id" binding:"required,uuid"`
	AmountWei  string `json:"amount_wei" binding:"required"`
	ToCurrency string `json:"to_currency" binding:"required"`
}
