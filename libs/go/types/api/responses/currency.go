package responses

// CurrencyResponse represents a currency in API responses
type CurrencyResponse struct {
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	Symbol            string   `json:"symbol"`
	DecimalPlaces     int32    `json:"decimal_places"`
	IsActive          bool     `json:"is_active"`
	SymbolPosition    string   `json:"symbol_position"`
	ThousandSeparator string   `json:"thousand_separator"`
	DecimalSeparator  string   `json:"decimal_separator"`
	Countries         []string `json:"countries"`
}

// ListCurrenciesResponse represents the response for listing currencies
type ListCurrenciesResponse struct {
	Currencies []CurrencyResponse `json:"currencies"`
}

// FormatAmountResponse represents the response for amount formatting
type FormatAmountResponse struct {
	Formatted string `json:"formatted"`
	Currency  string `json:"currency"`
}
