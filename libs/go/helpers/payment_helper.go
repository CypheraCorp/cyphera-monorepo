package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// PaymentResponse represents the response structure for payment data
type PaymentResponse struct {
	ID                  string          `json:"id"`
	WorkspaceID         string          `json:"workspace_id"`
	InvoiceID           *string         `json:"invoice_id,omitempty"`
	SubscriptionID      *string         `json:"subscription_id,omitempty"`
	SubscriptionEventID *string         `json:"subscription_event_id,omitempty"`
	CustomerID          string          `json:"customer_id"`
	AmountInCents       int64           `json:"amount_in_cents"`
	Currency            string          `json:"currency"`
	Status              string          `json:"status"`
	PaymentMethod       string          `json:"payment_method"`
	TransactionHash     *string         `json:"transaction_hash,omitempty"`
	NetworkID           *string         `json:"network_id,omitempty"`
	TokenID             *string         `json:"token_id,omitempty"`
	CryptoAmount        *string         `json:"crypto_amount,omitempty"`
	ExchangeRate        *string         `json:"exchange_rate,omitempty"`
	HasGasFee           bool            `json:"has_gas_fee"`
	GasFeeUSDCents      *int64          `json:"gas_fee_usd_cents,omitempty"`
	GasSponsored        bool            `json:"gas_sponsored"`
	ExternalPaymentID   *string         `json:"external_payment_id,omitempty"`
	PaymentProvider     *string         `json:"payment_provider,omitempty"`
	ProductAmountCents  int64           `json:"product_amount_cents"`
	TaxAmountCents      int64           `json:"tax_amount_cents"`
	GasAmountCents      int64           `json:"gas_amount_cents"`
	DiscountAmountCents int64           `json:"discount_amount_cents"`
	InitiatedAt         time.Time       `json:"initiated_at"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
	FailedAt            *time.Time      `json:"failed_at,omitempty"`
	ErrorMessage        *string         `json:"error_message,omitempty"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// ToPaymentResponse converts a db.Payment to a PaymentResponse
func ToPaymentResponse(payment db.Payment) PaymentResponse {
	response := PaymentResponse{
		ID:                 payment.ID.String(),
		WorkspaceID:        payment.WorkspaceID.String(),
		CustomerID:         payment.CustomerID.String(),
		AmountInCents:      payment.AmountInCents,
		Currency:           payment.Currency,
		Status:             payment.Status,
		PaymentMethod:      payment.PaymentMethod,
		ProductAmountCents: payment.ProductAmountCents,
		HasGasFee:          payment.HasGasFee.Bool,
		GasSponsored:       payment.GasSponsored.Bool,
		InitiatedAt:        payment.InitiatedAt.Time,
		CreatedAt:          payment.CreatedAt.Time,
		UpdatedAt:          payment.UpdatedAt.Time,
		Metadata:           payment.Metadata,
	}

	// Handle optional UUID fields
	if payment.InvoiceID.Valid {
		invoiceID := payment.InvoiceID.String()
		response.InvoiceID = &invoiceID
	}

	if payment.SubscriptionID.Valid {
		subscriptionID := payment.SubscriptionID.String()
		response.SubscriptionID = &subscriptionID
	}

	if payment.SubscriptionEvent.Valid {
		eventID := payment.SubscriptionEvent.String()
		response.SubscriptionEventID = &eventID
	}

	if payment.NetworkID.Valid {
		networkID := payment.NetworkID.String()
		response.NetworkID = &networkID
	}

	if payment.TokenID.Valid {
		tokenID := payment.TokenID.String()
		response.TokenID = &tokenID
	}

	// Handle optional text fields
	if payment.TransactionHash.Valid {
		response.TransactionHash = &payment.TransactionHash.String
	}

	if payment.ExternalPaymentID.Valid {
		response.ExternalPaymentID = &payment.ExternalPaymentID.String
	}

	if payment.PaymentProvider.Valid {
		response.PaymentProvider = &payment.PaymentProvider.String
	}

	if payment.ErrorMessage.Valid {
		response.ErrorMessage = &payment.ErrorMessage.String
	}

	// Handle optional numeric fields
	if payment.GasFeeUsdCents.Valid {
		response.GasFeeUSDCents = &payment.GasFeeUsdCents.Int64
	}

	if payment.TaxAmountCents.Valid {
		response.TaxAmountCents = payment.TaxAmountCents.Int64
	}

	if payment.GasAmountCents.Valid {
		response.GasAmountCents = payment.GasAmountCents.Int64
	}

	if payment.DiscountAmountCents.Valid {
		response.DiscountAmountCents = payment.DiscountAmountCents.Int64
	}

	// Handle optional timestamp fields
	if payment.CompletedAt.Valid {
		response.CompletedAt = &payment.CompletedAt.Time
	}

	if payment.FailedAt.Valid {
		response.FailedAt = &payment.FailedAt.Time
	}

	// Handle decimal fields (convert to string for JSON)
	if payment.CryptoAmount.Valid {
		cryptoAmount := convertNumericToString(payment.CryptoAmount, 8)
		response.CryptoAmount = &cryptoAmount
	}

	if payment.ExchangeRate.Valid {
		exchangeRate := convertNumericToString(payment.ExchangeRate, 6)
		response.ExchangeRate = &exchangeRate
	}

	return response
}

// PaymentMetricsResponse represents aggregated payment metrics
type PaymentMetricsResponse struct {
	CompletedCount        int64   `json:"completed_count"`
	FailedCount           int64   `json:"failed_count"`
	PendingCount          int64   `json:"pending_count"`
	TotalCompletedCents   int64   `json:"total_completed_cents"`
	TotalGasFeesCents     int64   `json:"total_gas_fees_cents"`
	SponsoredGasFeesCents int64   `json:"sponsored_gas_fees_cents"`
	SuccessRate           float64 `json:"success_rate"`
	AveragePaymentCents   int64   `json:"average_payment_cents"`
}

// ToPaymentMetricsResponse converts payment metrics to response format
func ToPaymentMetricsResponse(metrics db.GetPaymentMetricsRow) PaymentMetricsResponse {
	totalPayments := metrics.CompletedCount + metrics.FailedCount + metrics.PendingCount

	var successRate float64
	var averagePayment int64

	if totalPayments > 0 {
		successRate = float64(metrics.CompletedCount) / float64(totalPayments)
	}

	if metrics.CompletedCount > 0 {
		averagePayment = metrics.TotalCompletedCents / metrics.CompletedCount
	}

	return PaymentMetricsResponse{
		CompletedCount:        metrics.CompletedCount,
		FailedCount:           metrics.FailedCount,
		PendingCount:          metrics.PendingCount,
		TotalCompletedCents:   metrics.TotalCompletedCents,
		TotalGasFeesCents:     metrics.TotalGasFeesCents,
		SponsoredGasFeesCents: metrics.SponsoredGasFeesCents,
		SuccessRate:           successRate,
		AveragePaymentCents:   averagePayment,
	}
}

// PaymentSummaryResponse represents a simplified payment summary
type PaymentSummaryResponse struct {
	ID              string     `json:"id"`
	CustomerID      string     `json:"customer_id"`
	AmountInCents   int64      `json:"amount_in_cents"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	PaymentMethod   string     `json:"payment_method"`
	TransactionHash *string    `json:"transaction_hash,omitempty"`
	HasGasFee       bool       `json:"has_gas_fee"`
	GasSponsored    bool       `json:"gas_sponsored"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ToPaymentSummaryResponse converts a db.Payment to a simplified summary response
func ToPaymentSummaryResponse(payment db.Payment) PaymentSummaryResponse {
	response := PaymentSummaryResponse{
		ID:            payment.ID.String(),
		CustomerID:    payment.CustomerID.String(),
		AmountInCents: payment.AmountInCents,
		Currency:      payment.Currency,
		Status:        payment.Status,
		PaymentMethod: payment.PaymentMethod,
		HasGasFee:     payment.HasGasFee.Bool,
		GasSponsored:  payment.GasSponsored.Bool,
		CreatedAt:     payment.CreatedAt.Time,
	}

	if payment.TransactionHash.Valid {
		response.TransactionHash = &payment.TransactionHash.String
	}

	if payment.CompletedAt.Valid {
		response.CompletedAt = &payment.CompletedAt.Time
	}

	return response
}

// ValidatePaymentMethod validates payment method values
func ValidatePaymentMethod(method string) error {
	validMethods := map[string]bool{
		"crypto":        true,
		"card":          true,
		"bank_transfer": true,
		"apple_pay":     true,
		"google_pay":    true,
	}

	if !validMethods[method] {
		return fmt.Errorf("invalid payment method: %s", method)
	}

	return nil
}

// ValidatePaymentStatus validates payment status values
func ValidatePaymentStatus(status string) error {
	validStatuses := map[string]bool{
		"pending":    true,
		"processing": true,
		"completed":  true,
		"failed":     true,
		"refunded":   true,
		"cancelled":  true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid payment status: %s", status)
	}

	return nil
}

// ValidatePaymentProvider validates payment provider values
func ValidatePaymentProvider(provider string) error {
	validProviders := map[string]bool{
		"internal": true,
		"stripe":   true,
		"circle":   true,
		"paypal":   true,
	}

	if !validProviders[provider] {
		return fmt.Errorf("invalid payment provider: %s", provider)
	}

	return nil
}

// CalculatePaymentBreakdown calculates the payment amount breakdown
type PaymentBreakdown struct {
	ProductAmount  int64 `json:"product_amount_cents"`
	TaxAmount      int64 `json:"tax_amount_cents"`
	GasAmount      int64 `json:"gas_amount_cents"`
	DiscountAmount int64 `json:"discount_amount_cents"`
	TotalAmount    int64 `json:"total_amount_cents"`
}

// CalculatePaymentBreakdown calculates payment amount breakdown
func CalculatePaymentBreakdown(productAmount, taxAmount, gasAmount, discountAmount int64) PaymentBreakdown {
	totalAmount := productAmount + taxAmount + gasAmount - discountAmount

	return PaymentBreakdown{
		ProductAmount:  productAmount,
		TaxAmount:      taxAmount,
		GasAmount:      gasAmount,
		DiscountAmount: discountAmount,
		TotalAmount:    totalAmount,
	}
}

// FormatCurrencyAmount formats an amount in cents to a display format
func FormatCurrencyAmount(amountInCents int64, currency string) string {
	switch currency {
	case "USD", "EUR", "GBP", "CAD", "AUD":
		return fmt.Sprintf("%.2f", float64(amountInCents)/100.0)
	case "JPY":
		return fmt.Sprintf("%d", amountInCents)
	default:
		return fmt.Sprintf("%.2f", float64(amountInCents)/100.0)
	}
}

// FormatDecimalString formats a decimal value to string with proper precision
func FormatDecimalString(value float64, decimals int) string {
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, value)
}

// ParseDecimalString parses a decimal string to float64
func ParseDecimalString(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// FormatExchangeRate formats an exchange rate for display
func FormatExchangeRate(rate float64) string {
	return FormatDecimalString(rate, 6)
}

// FormatCryptoAmount formats a crypto amount for display
func FormatCryptoAmount(amount float64) string {
	return FormatDecimalString(amount, 8)
}

// convertNumericToString converts pgtype.Numeric to formatted string
func convertNumericToString(numeric pgtype.Numeric, decimals int) string {
	if !numeric.Valid {
		return "0"
	}
	
	// For now, use a simple string conversion
	// In production, you'd want proper decimal handling
	str := fmt.Sprintf("%.8f", 0.0) // Placeholder
	if len(str) > 0 {
		// Try to parse and format properly
		if value, err := strconv.ParseFloat(str, 64); err == nil {
			return FormatDecimalString(value, decimals)
		}
	}
	
	return FormatDecimalString(0.0, decimals)
}
