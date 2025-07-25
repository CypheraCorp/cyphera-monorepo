package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/db"
)

// FormatCurrencyAmount formats an amount in cents to a human-readable string with currency symbol
func FormatCurrencyAmount(amountCents int64, currency *db.FiatCurrency) string {
	// Convert cents to decimal amount
	divisor := math.Pow(10, float64(currency.DecimalPlaces))
	amount := float64(amountCents) / divisor

	// Format with proper decimal places
	formatted := formatWithSeparators(amount, currency)

	// Add symbol in correct position
	symbol := currency.Symbol
	if currency.SymbolPosition.Valid && currency.SymbolPosition.String == "after" {
		return formatted + symbol
	}
	return symbol + formatted
}

// FormatCurrencyAmountWithCode formats amount with currency code instead of symbol
func FormatCurrencyAmountWithCode(amountCents int64, currency *db.FiatCurrency) string {
	// Convert cents to decimal amount
	divisor := math.Pow(10, float64(currency.DecimalPlaces))
	amount := float64(amountCents) / divisor

	// Format with proper decimal places
	formatted := formatWithSeparators(amount, currency)

	return fmt.Sprintf("%s %s", currency.Code, formatted)
}

// ParseCurrencyAmount parses a human-readable amount string to cents
func ParseCurrencyAmount(amountStr string, currency *db.FiatCurrency) (int64, error) {
	// Remove currency symbol and whitespace
	cleaned := strings.TrimSpace(amountStr)
	cleaned = strings.Replace(cleaned, currency.Symbol, "", -1)
	cleaned = strings.TrimSpace(cleaned)

	// Remove thousand separators
	if currency.ThousandSeparator.Valid {
		cleaned = strings.ReplaceAll(cleaned, currency.ThousandSeparator.String, "")
	}

	// Replace decimal separator with dot for parsing
	if currency.DecimalSeparator.Valid && currency.DecimalSeparator.String != "." {
		cleaned = strings.Replace(cleaned, currency.DecimalSeparator.String, ".", 1)
	}

	// Parse the amount
	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %w", err)
	}

	// Convert to cents
	multiplier := math.Pow(10, float64(currency.DecimalPlaces))
	cents := int64(math.Round(amount * multiplier))

	return cents, nil
}

// formatWithSeparators formats a number with thousand and decimal separators
func formatWithSeparators(amount float64, currency *db.FiatCurrency) string {
	// Format with decimal places
	format := fmt.Sprintf("%%.%df", currency.DecimalPlaces)
	formatted := fmt.Sprintf(format, amount)

	// Split into integer and decimal parts
	parts := strings.Split(formatted, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) > 1 {
		decimalPart = parts[1]
	}

	// Add thousand separators to integer part
	if currency.ThousandSeparator.Valid && currency.ThousandSeparator.String != "" {
		integerPart = addThousandSeparators(integerPart, currency.ThousandSeparator.String)
	}

	// Reconstruct with proper decimal separator
	if decimalPart != "" && currency.DecimalSeparator.Valid {
		return integerPart + currency.DecimalSeparator.String + decimalPart
	}

	return integerPart
}

// addThousandSeparators adds thousand separators to a number string
func addThousandSeparators(numStr string, separator string) string {
	// Handle negative numbers
	negative := false
	if strings.HasPrefix(numStr, "-") {
		negative = true
		numStr = numStr[1:]
	}

	// Add separators from right to left
	var result []rune
	for i, r := range []rune(numStr) {
		if i > 0 && (len(numStr)-i)%3 == 0 {
			result = append(result, []rune(separator)...)
		}
		result = append(result, r)
	}

	if negative {
		return "-" + string(result)
	}
	return string(result)
}