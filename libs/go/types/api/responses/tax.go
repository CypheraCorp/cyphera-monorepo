package responses

import (
	"time"

	"github.com/cyphera/cyphera-api/libs/go/types/business"
)

// TaxCalculationResult contains the calculated tax information
type TaxCalculationResult struct {
	SubtotalCents        int64                  `json:"subtotal_cents"`
	TotalTaxCents        int64                  `json:"total_tax_cents"`
	TotalAmountCents     int64                  `json:"total_amount_cents"`
	TaxBreakdown         []business.TaxLineItem `json:"tax_breakdown"`
	AppliedJurisdictions []string               `json:"applied_jurisdictions"`
	TaxExemptReason      *string                `json:"tax_exempt_reason,omitempty"`
	CalculatedAt         time.Time              `json:"calculated_at"`
	Confidence           float64                `json:"confidence"` // 0.0 to 1.0
	AuditTrail           business.TaxAuditTrail `json:"audit_trail"`
}
