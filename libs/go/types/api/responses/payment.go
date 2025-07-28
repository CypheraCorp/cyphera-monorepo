package responses

import (
	"encoding/json"
	"time"
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
