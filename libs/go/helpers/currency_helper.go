package helpers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"go.uber.org/zap"
)

// CurrencyHelper provides utility functions for currency operations
type CurrencyHelper struct {
	queries db.Querier
}

// NewCurrencyHelper creates a new currency helper
func NewCurrencyHelper(queries db.Querier) *CurrencyHelper {
	return &CurrencyHelper{
		queries: queries,
	}
}

// CurrencyToResponse converts db.FiatCurrency to CurrencyResponse
func (h *CurrencyHelper) CurrencyToResponse(currency *db.FiatCurrency) responses.CurrencyResponse {
	response := responses.CurrencyResponse{
		Code:          currency.Code,
		Name:          currency.Name,
		Symbol:        currency.Symbol,
		DecimalPlaces: currency.DecimalPlaces,
		IsActive:      currency.IsActive.Bool,
	}

	if currency.SymbolPosition.Valid {
		response.SymbolPosition = currency.SymbolPosition.String
	} else {
		response.SymbolPosition = "before"
	}

	if currency.ThousandSeparator.Valid {
		response.ThousandSeparator = currency.ThousandSeparator.String
	} else {
		response.ThousandSeparator = ","
	}

	if currency.DecimalSeparator.Valid {
		response.DecimalSeparator = currency.DecimalSeparator.String
	} else {
		response.DecimalSeparator = "."
	}

	// Parse countries JSON
	if currency.Countries != nil {
		if err := json.Unmarshal(currency.Countries, &response.Countries); err != nil {
			logger.Log.Error("Failed to parse countries", zap.Error(err))
			response.Countries = []string{}
		}
	}

	return response
}

// ValidateCurrencyCode validates a currency code format
func (h *CurrencyHelper) ValidateCurrencyCode(code string) error {
	// Trim and convert to uppercase
	code = strings.TrimSpace(strings.ToUpper(code))

	// Check length (ISO 4217 currency codes are 3 characters)
	if len(code) != 3 {
		return fmt.Errorf("currency code must be exactly 3 characters")
	}

	// Check for valid characters (A-Z only)
	for _, char := range code {
		if char < 'A' || char > 'Z' {
			return fmt.Errorf("currency code must contain only uppercase letters")
		}
	}

	return nil
}

// ParseSupportedCurrencies parses supported currencies JSON safely
func (h *CurrencyHelper) ParseSupportedCurrencies(supportedCurrenciesJSON []byte) ([]string, error) {
	var supportedCurrencies []string

	if supportedCurrenciesJSON == nil || len(supportedCurrenciesJSON) == 0 {
		return []string{"USD"}, nil // Default fallback
	}

	if err := json.Unmarshal(supportedCurrenciesJSON, &supportedCurrencies); err != nil {
		return []string{"USD"}, fmt.Errorf("failed to parse supported currencies: %w", err)
	}

	// Validate that we have at least one currency
	if len(supportedCurrencies) == 0 {
		return []string{"USD"}, nil
	}

	return supportedCurrencies, nil
}

// SerializeSupportedCurrencies serializes supported currencies to JSON
func (h *CurrencyHelper) SerializeSupportedCurrencies(currencies []string) ([]byte, error) {
	if len(currencies) == 0 {
		currencies = []string{"USD"} // Default fallback
	}

	// Validate each currency code
	for _, code := range currencies {
		if err := h.ValidateCurrencyCode(code); err != nil {
			return nil, fmt.Errorf("invalid currency code '%s': %w", code, err)
		}
	}

	data, err := json.Marshal(currencies)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize supported currencies: %w", err)
	}

	return data, nil
}

// GetDefaultCurrencySettings returns default currency settings for new workspaces
func (h *CurrencyHelper) GetDefaultCurrencySettings() (string, []string) {
	return "USD", []string{"USD", "EUR", "GBP"}
}

// IsCurrencyInList checks if a currency code exists in a currency list
func (h *CurrencyHelper) IsCurrencyInList(currencyCode string, currencyList []string) bool {
	currencyCode = strings.ToUpper(strings.TrimSpace(currencyCode))

	for _, currency := range currencyList {
		if strings.ToUpper(strings.TrimSpace(currency)) == currencyCode {
			return true
		}
	}

	return false
}

// RemoveDuplicateCurrencies removes duplicate currency codes from a list
func (h *CurrencyHelper) RemoveDuplicateCurrencies(currencies []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, currency := range currencies {
		normalized := strings.ToUpper(strings.TrimSpace(currency))
		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}

	return result
}

// ValidateCurrencyList validates a list of currency codes
func (h *CurrencyHelper) ValidateCurrencyList(currencies []string) error {
	if len(currencies) == 0 {
		return fmt.Errorf("currency list cannot be empty")
	}

	// Check for duplicates and validate each code
	seen := make(map[string]bool)
	for _, code := range currencies {
		if err := h.ValidateCurrencyCode(code); err != nil {
			return fmt.Errorf("invalid currency code '%s': %w", code, err)
		}

		normalized := strings.ToUpper(strings.TrimSpace(code))
		if seen[normalized] {
			return fmt.Errorf("duplicate currency code: %s", normalized)
		}
		seen[normalized] = true
	}

	return nil
}

// GetCurrencyDisplayName returns a formatted display name for a currency
func (h *CurrencyHelper) GetCurrencyDisplayName(currency *db.FiatCurrency) string {
	if currency.Symbol != "" {
		return fmt.Sprintf("%s (%s)", currency.Name, currency.Symbol)
	}
	return fmt.Sprintf("%s (%s)", currency.Name, currency.Code)
}

// SortCurrenciesByCode sorts currencies alphabetically by their code
func (h *CurrencyHelper) SortCurrenciesByCode(currencies []responses.CurrencyResponse) []responses.CurrencyResponse {
	// Simple bubble sort for small lists
	n := len(currencies)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if currencies[j].Code > currencies[j+1].Code {
				currencies[j], currencies[j+1] = currencies[j+1], currencies[j]
			}
		}
	}
	return currencies
}

// GetMajorCurrencies returns a list of major world currencies
func (h *CurrencyHelper) GetMajorCurrencies() []string {
	return []string{
		"USD", // US Dollar
		"EUR", // Euro
		"GBP", // British Pound
		"JPY", // Japanese Yen
		"CAD", // Canadian Dollar
		"AUD", // Australian Dollar
		"CHF", // Swiss Franc
		"CNY", // Chinese Yuan
		"INR", // Indian Rupee
		"KRW", // South Korean Won
	}
}

// ToResponsesCurrencyResponse converts helpers.CurrencyResponse to responses.CurrencyResponse
func ToResponsesCurrencyResponse(helperResp responses.CurrencyResponse) responses.CurrencyResponse {
	return responses.CurrencyResponse{
		Code:              helperResp.Code,
		Name:              helperResp.Name,
		Symbol:            helperResp.Symbol,
		DecimalPlaces:     helperResp.DecimalPlaces,
		IsActive:          helperResp.IsActive,
		SymbolPosition:    helperResp.SymbolPosition,
		ThousandSeparator: helperResp.ThousandSeparator,
		DecimalSeparator:  helperResp.DecimalSeparator,
		Countries:         helperResp.Countries,
	}
}

// ToResponsesCurrencyResponseList converts []helpers.CurrencyResponse to []responses.CurrencyResponse
func ToResponsesCurrencyResponseList(helperResps []responses.CurrencyResponse) []responses.CurrencyResponse {
	result := make([]responses.CurrencyResponse, len(helperResps))
	for i, helperResp := range helperResps {
		result[i] = ToResponsesCurrencyResponse(helperResp)
	}
	return result
}
