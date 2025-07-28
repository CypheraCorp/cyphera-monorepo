package business

import (
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// InvoiceWithDetails contains invoice with all line items and calculations
type InvoiceWithDetails struct {
	Invoice          db.Invoice
	LineItems        []db.InvoiceLineItem
	ProductSubtotal  int64
	GasFeesSubtotal  int64
	SponsoredGasFees int64
	TaxAmount        int64
	DiscountAmount   int64
	TotalAmount      int64
	CustomerTotal    int64 // Total after gas sponsorship
	TaxDetails       []TaxDetail
	CryptoAmounts    []CryptoAmount
}

// TaxDetail contains tax calculation details
type TaxDetail struct {
	JurisdictionID   string  `json:"jurisdiction_id"`
	JurisdictionName string  `json:"jurisdiction_name"`
	TaxRate          float64 `json:"tax_rate"`
	TaxAmountCents   int64   `json:"tax_amount_cents"`
	TaxType          string  `json:"tax_type"` // "vat", "sales_tax", etc.
}

// CryptoAmount contains crypto payment amounts by token
type CryptoAmount struct {
	TokenID         uuid.UUID `json:"token_id"`
	NetworkID       uuid.UUID `json:"network_id"`
	CryptoAmount    string    `json:"crypto_amount"`
	TaxCryptoAmount string    `json:"tax_crypto_amount"`
}
