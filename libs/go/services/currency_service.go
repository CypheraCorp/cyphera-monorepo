package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// CurrencyService handles currency-related operations
type CurrencyService struct {
	queries       db.Querier
	logger        *zap.Logger
	currencyHelper *helpers.CurrencyHelper
}

// NewCurrencyService creates a new currency service
func NewCurrencyService(queries db.Querier) *CurrencyService {
	return &CurrencyService{
		queries:       queries,
		logger:        logger.Log,
		currencyHelper: helpers.NewCurrencyHelper(queries),
	}
}

// FormatAmount formats an amount in cents to a human-readable string with currency symbol
func (s *CurrencyService) FormatAmount(ctx context.Context, amountCents int64, currencyCode string) (string, error) {
	currency, err := s.queries.GetFiatCurrency(ctx, currencyCode)
	if err != nil {
		return "", fmt.Errorf("failed to get currency: %w", err)
	}

	return s.FormatCurrencyAmount(amountCents, &currency), nil
}

// FormatAmountWithCode formats amount with currency code instead of symbol
func (s *CurrencyService) FormatAmountWithCode(ctx context.Context, amountCents int64, currencyCode string) (string, error) {
	currency, err := s.queries.GetFiatCurrency(ctx, currencyCode)
	if err != nil {
		return "", fmt.Errorf("failed to get currency: %w", err)
	}

	return s.FormatCurrencyAmountWithCode(amountCents, &currency), nil
}

// FormatCurrencyAmount formats an amount in cents to a human-readable string with currency symbol
func (s *CurrencyService) FormatCurrencyAmount(amountCents int64, currency *db.FiatCurrency) string {
	// Convert cents to decimal amount
	divisor := math.Pow(10, float64(currency.DecimalPlaces))
	amount := float64(amountCents) / divisor

	// Format with proper decimal places
	formatted := s.formatWithSeparators(amount, currency)

	// Add symbol in correct position
	symbol := currency.Symbol
	if currency.SymbolPosition.Valid && currency.SymbolPosition.String == "after" {
		return formatted + symbol
	}
	return symbol + formatted
}

// FormatCurrencyAmountWithCode formats amount with currency code instead of symbol
func (s *CurrencyService) FormatCurrencyAmountWithCode(amountCents int64, currency *db.FiatCurrency) string {
	// Convert cents to decimal amount
	divisor := math.Pow(10, float64(currency.DecimalPlaces))
	amount := float64(amountCents) / divisor

	// Format with proper decimal places
	formatted := s.formatWithSeparators(amount, currency)

	return fmt.Sprintf("%s %s", currency.Code, formatted)
}

// ParseAmount parses a human-readable amount string to cents
func (s *CurrencyService) ParseAmount(ctx context.Context, amountStr string, currencyCode string) (int64, error) {
	currency, err := s.queries.GetFiatCurrency(ctx, currencyCode)
	if err != nil {
		return 0, fmt.Errorf("failed to get currency: %w", err)
	}

	return s.ParseCurrencyAmount(amountStr, &currency)
}

// ParseCurrencyAmount parses a human-readable amount string to cents
func (s *CurrencyService) ParseCurrencyAmount(amountStr string, currency *db.FiatCurrency) (int64, error) {
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

// ConvertAmount converts an amount from one currency to another
func (s *CurrencyService) ConvertAmount(ctx context.Context, amountCents int64, fromCurrency, toCurrency string, exchangeRate float64) (int64, error) {
	if fromCurrency == toCurrency {
		return amountCents, nil
	}

	// Get currency details
	fromCurr, err := s.queries.GetFiatCurrency(ctx, fromCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get source currency: %w", err)
	}

	toCurr, err := s.queries.GetFiatCurrency(ctx, toCurrency)
	if err != nil {
		return 0, fmt.Errorf("failed to get target currency: %w", err)
	}

	// Convert to base units (from cents to currency units)
	fromDivisor := math.Pow(10, float64(fromCurr.DecimalPlaces))
	amount := float64(amountCents) / fromDivisor

	// Apply exchange rate
	convertedAmount := amount * exchangeRate

	// Convert to target currency cents
	toMultiplier := math.Pow(10, float64(toCurr.DecimalPlaces))
	targetCents := int64(math.Round(convertedAmount * toMultiplier))

	return targetCents, nil
}

// ValidateCurrencyForWorkspace checks if a currency is supported by a workspace
func (s *CurrencyService) ValidateCurrencyForWorkspace(ctx context.Context, workspaceID uuid.UUID, currencyCode string) error {
	// Get workspace
	workspace, err := s.queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Parse supported currencies
	var supportedCurrencies []string
	if workspace.SupportedCurrencies != nil {
		if err := json.Unmarshal(workspace.SupportedCurrencies, &supportedCurrencies); err != nil {
			return fmt.Errorf("failed to parse supported currencies: %w", err)
		}
	}

	// Check if currency is supported
	for _, supported := range supportedCurrencies {
		if supported == currencyCode {
			return nil
		}
	}

	return fmt.Errorf("currency %s is not supported by this workspace", currencyCode)
}

// GetWorkspaceDefaultCurrency returns the default currency for a workspace
func (s *CurrencyService) GetWorkspaceDefaultCurrency(ctx context.Context, workspaceID uuid.UUID) (*db.FiatCurrency, error) {
	currency, err := s.queries.GetWorkspaceDefaultCurrency(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace default currency: %w", err)
	}
	return &currency, nil
}

// formatWithSeparators formats a number with thousand and decimal separators
func (s *CurrencyService) formatWithSeparators(amount float64, currency *db.FiatCurrency) string {
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
		integerPart = s.addThousandSeparators(integerPart, currency.ThousandSeparator.String)
	}

	// Reconstruct with proper decimal separator
	if decimalPart != "" && currency.DecimalSeparator.Valid {
		return integerPart + currency.DecimalSeparator.String + decimalPart
	}

	return integerPart
}

// addThousandSeparators adds thousand separators to a number string
func (s *CurrencyService) addThousandSeparators(numStr string, separator string) string {
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

// GetSmallestUnit returns the smallest unit name for a currency (e.g., "cents" for USD)
func (s *CurrencyService) GetSmallestUnit(ctx context.Context, currencyCode string) (string, error) {
	// Common smallest unit names
	smallestUnits := map[string]string{
		"USD": "cents",
		"EUR": "cents",
		"GBP": "pence",
		"JPY": "yen", // JPY has no decimal places
		"CAD": "cents",
		"AUD": "cents",
		"CHF": "centimes",
		"CNY": "fen",
		"INR": "paise",
	}

	if unit, ok := smallestUnits[currencyCode]; ok {
		return unit, nil
	}

	// For unknown currencies, check if it has decimal places
	currency, err := s.queries.GetFiatCurrency(ctx, currencyCode)
	if err != nil {
		return "", fmt.Errorf("failed to get currency: %w", err)
	}

	if currency.DecimalPlaces == 0 {
		return strings.ToLower(currencyCode), nil
	}

	return "units", nil // Generic term for unknown currencies
}

// ListActiveCurrencies retrieves all active fiat currencies
func (s *CurrencyService) ListActiveCurrencies(ctx context.Context) ([]helpers.CurrencyResponse, error) {
	currencies, err := s.queries.ListActiveFiatCurrencies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active currencies: %w", err)
	}

	response := make([]helpers.CurrencyResponse, len(currencies))
	for i, currency := range currencies {
		response[i] = s.currencyHelper.CurrencyToResponse(&currency)
	}

	return response, nil
}

// GetCurrency retrieves a specific currency by code
func (s *CurrencyService) GetCurrency(ctx context.Context, code string) (*helpers.CurrencyResponse, error) {
	currency, err := s.queries.GetFiatCurrency(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency: %w", err)
	}

	response := s.currencyHelper.CurrencyToResponse(&currency)
	return &response, nil
}

// GetWorkspaceCurrencySettings retrieves currency settings for a workspace
func (s *CurrencyService) GetWorkspaceCurrencySettings(ctx context.Context, workspaceID uuid.UUID) (*WorkspaceCurrencySettings, error) {
	workspace, err := s.queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	supportedCurrencies, err := s.currencyHelper.ParseSupportedCurrencies(workspace.SupportedCurrencies)
	if err != nil {
		return nil, fmt.Errorf("failed to parse supported currencies: %w", err)
	}

	return &WorkspaceCurrencySettings{
		DefaultCurrency:     workspace.DefaultCurrency.String,
		SupportedCurrencies: supportedCurrencies,
	}, nil
}

// UpdateWorkspaceCurrencySettings updates currency settings for a workspace
func (s *CurrencyService) UpdateWorkspaceCurrencySettings(ctx context.Context, workspaceID uuid.UUID, req *UpdateWorkspaceCurrencyRequest) (*WorkspaceCurrencySettings, error) {
	// Update default currency if provided
	if req.DefaultCurrency != nil {
		// Validate currency exists and is active
		if _, err := s.queries.GetFiatCurrency(ctx, *req.DefaultCurrency); err != nil {
			return nil, fmt.Errorf("invalid default currency: %w", err)
		}

		if err := s.queries.UpdateWorkspaceDefaultCurrency(ctx, db.UpdateWorkspaceDefaultCurrencyParams{
			ID:              workspaceID,
			DefaultCurrency: pgtype.Text{String: *req.DefaultCurrency, Valid: true},
		}); err != nil {
			return nil, fmt.Errorf("failed to update default currency: %w", err)
		}
	}

	// Update supported currencies if provided
	if req.SupportedCurrencies != nil {
		// Validate all currencies exist and are active
		for _, code := range req.SupportedCurrencies {
			if _, err := s.queries.GetFiatCurrency(ctx, code); err != nil {
				return nil, fmt.Errorf("invalid currency %s: %w", code, err)
			}
		}

		// Serialize currencies using helper
		supportedJSON, err := s.currencyHelper.SerializeSupportedCurrencies(req.SupportedCurrencies)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize supported currencies: %w", err)
		}

		if err := s.queries.UpdateWorkspaceSupportedCurrencies(ctx, db.UpdateWorkspaceSupportedCurrenciesParams{
			ID:                  workspaceID,
			SupportedCurrencies: supportedJSON,
		}); err != nil {
			return nil, fmt.Errorf("failed to update supported currencies: %w", err)
		}
	}

	// Return updated settings
	return s.GetWorkspaceCurrencySettings(ctx, workspaceID)
}

// ListWorkspaceSupportedCurrencies retrieves currencies supported by a workspace
func (s *CurrencyService) ListWorkspaceSupportedCurrencies(ctx context.Context, workspaceID uuid.UUID) ([]helpers.CurrencyResponse, error) {
	currencies, err := s.queries.ListWorkspaceSupportedCurrencies(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace currencies: %w", err)
	}

	response := make([]helpers.CurrencyResponse, len(currencies))
	for i, currency := range currencies {
		response[i] = s.currencyHelper.CurrencyToResponse(&currency)
	}

	return response, nil
}

// WorkspaceCurrencySettings represents workspace currency settings
type WorkspaceCurrencySettings struct {
	DefaultCurrency     string   `json:"default_currency"`
	SupportedCurrencies []string `json:"supported_currencies"`
}

// UpdateWorkspaceCurrencyRequest represents a request to update workspace currency settings
type UpdateWorkspaceCurrencyRequest struct {
	DefaultCurrency     *string  `json:"default_currency,omitempty"`
	SupportedCurrencies []string `json:"supported_currencies,omitempty"`
}

// ValidateAmount validates if an amount is valid for a currency
func (s *CurrencyService) ValidateAmount(ctx context.Context, amountCents int64, currencyCode string) error {
	if amountCents < 0 {
		return fmt.Errorf("amount cannot be negative")
	}

	currency, err := s.queries.GetFiatCurrency(ctx, currencyCode)
	if err != nil {
		return fmt.Errorf("failed to get currency: %w", err)
	}

	// Check if currency is active
	if !currency.IsActive.Bool {
		return fmt.Errorf("currency %s is not active", currencyCode)
	}

	// Add any additional validation rules here
	// For example, minimum transaction amounts for certain currencies

	return nil
}
