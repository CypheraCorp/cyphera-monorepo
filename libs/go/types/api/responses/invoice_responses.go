package responses

import (
	"time"

	"github.com/google/uuid"
)

// BulkInvoiceError represents an error that occurred during bulk invoice generation
type BulkInvoiceError struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	Error          string    `json:"error"`
}

// BulkInvoiceGenerationResult represents the result of bulk invoice generation
type BulkInvoiceGenerationResult struct {
	Success        []InvoiceResponse     `json:"success"`
	Failed         []BulkInvoiceError    `json:"failed"`
	TotalProcessed int                   `json:"total_processed"`
	SuccessCount   int                   `json:"success_count"`
	FailedCount    int                   `json:"failed_count"`
}

// InvoiceStatsResponse represents invoice statistics for a workspace
type InvoiceStatsResponse struct {
	DraftCount               int64     `json:"draft_count"`
	OpenCount                int64     `json:"open_count"`
	PaidCount                int64     `json:"paid_count"`
	VoidCount                int64     `json:"void_count"`
	UncollectibleCount       int64     `json:"uncollectible_count"`
	TotalCount               int64     `json:"total_count"`
	TotalOutstandingCents    int64     `json:"total_outstanding_cents"`
	TotalPaidCents           int64     `json:"total_paid_cents"`
	TotalUncollectibleCents  int64     `json:"total_uncollectible_cents"`
	Currency                 string    `json:"currency"`
	PeriodStart              time.Time `json:"period_start"`
	PeriodEnd                time.Time `json:"period_end"`
}