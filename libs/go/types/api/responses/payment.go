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
	Customer            *CustomerBasic  `json:"customer,omitempty"`
	AmountInCents       int64           `json:"amount_in_cents"`
	FormattedAmount     string          `json:"formatted_amount"`
	Currency            string          `json:"currency"`
	Status              string          `json:"status"`
	PaymentMethod       string          `json:"payment_method"`
	TransactionHash     *string         `json:"transaction_hash,omitempty"`
	NetworkID           *string         `json:"network_id,omitempty"`
	Network             *NetworkBasic   `json:"network,omitempty"`
	TokenID             *string         `json:"token_id,omitempty"`
	Token               *TokenBasic     `json:"token,omitempty"`
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
	ProductName         *string         `json:"product_name,omitempty"`
	ProductID           *string         `json:"product_id,omitempty"`
	InitiatedAt         time.Time       `json:"initiated_at"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
	FailedAt            *time.Time      `json:"failed_at,omitempty"`
	ErrorMessage        *string         `json:"error_message,omitempty"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// CustomerBasic represents basic customer info for embedded responses
type CustomerBasic struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NetworkBasic represents basic network info for embedded responses
type NetworkBasic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChainID     int64  `json:"chain_id"`
	DisplayName string `json:"display_name"`
}

// TokenBasic represents basic token info for embedded responses
type TokenBasic struct {
	ID              string `json:"id"`
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	ContractAddress string `json:"contract_address"`
	Decimals        int    `json:"decimals"`
}

// PaymentListResponse represents a paginated list of payments
type PaymentListResponse struct {
	Data       []PaymentResponse `json:"data"`
	Pagination PaginationMeta    `json:"pagination"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int  `json:"page"`
	PerPage    int  `json:"per_page"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasPrev    bool `json:"has_prev"`
	HasNext    bool `json:"has_next"`
}
