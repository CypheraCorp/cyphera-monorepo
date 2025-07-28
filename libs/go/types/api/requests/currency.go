package requests

// FormatAmountRequest represents a request to format an amount
type FormatAmountRequest struct {
	AmountCents  int64  `json:"amount_cents" binding:"required"`
	CurrencyCode string `json:"currency_code" binding:"required,len=3"`
	UseSymbol    bool   `json:"use_symbol"`
}

// UpdateWorkspaceCurrencyRequest represents a request to update workspace currency settings
type UpdateWorkspaceCurrencyRequest struct {
	DefaultCurrency     *string  `json:"default_currency,omitempty"`
	SupportedCurrencies []string `json:"supported_currencies,omitempty"`
}
