package helpers

import (
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// FormatMoney formats cents to a currency string
func FormatMoney(cents int64, currency string) string {
	dollars := float64(cents) / 100.0
	return fmt.Sprintf("%s%.2f", GetCurrencySymbol(currency), dollars)
}

// GetCurrencySymbol returns the symbol for a given currency code
func GetCurrencySymbol(currency string) string {
	symbols := map[string]string{
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		"JPY": "¥",
		"CAD": "C$",
		"AUD": "A$",
	}
	if symbol, ok := symbols[currency]; ok {
		return symbol
	}
	return currency + " "
}

// GetNumericFloat safely converts pgtype.Numeric to float64
func GetNumericFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	// Try to get the value as float64
	f, err := n.Value()
	if err != nil {
		return 0
	}
	// Check if it's nil
	if f == nil {
		return 0
	}
	// Type assert to get the actual numeric value
	switch v := f.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case string:
		// Try to parse string representation
		val, _ := strconv.ParseFloat(v, 64)
		return val
	default:
		return 0
	}
}
