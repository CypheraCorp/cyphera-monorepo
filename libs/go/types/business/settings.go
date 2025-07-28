package business

// WorkspaceCurrencySettings represents workspace currency settings
type WorkspaceCurrencySettings struct {
	DefaultCurrency     string   `json:"default_currency"`
	SupportedCurrencies []string `json:"supported_currencies"`
}
