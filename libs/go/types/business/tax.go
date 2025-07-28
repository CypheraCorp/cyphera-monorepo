package business

import (
	"time"

	"github.com/google/uuid"
)

// Address represents a tax-relevant address
type Address struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// TaxLineItem represents a single tax component
type TaxLineItem struct {
	TaxType        string  `json:"tax_type"`     // "sales", "vat", "gst", "excise"
	Jurisdiction   string  `json:"jurisdiction"` // "US-CA", "EU-DE", "CA-ON"
	Rate           float64 `json:"rate"`         // e.g., 0.0825 for 8.25%
	TaxableAmount  int64   `json:"taxable_amount_cents"`
	TaxAmountCents int64   `json:"tax_amount_cents"`
	Description    string  `json:"description"`
	IsReversCharge bool    `json:"is_reverse_charge"`
}

// TaxAuditTrail contains audit information for tax calculations
type TaxAuditTrail struct {
	RulesVersion     string        `json:"rules_version"`
	DetectedLocation *Address      `json:"detected_location,omitempty"`
	AppliedRules     []string      `json:"applied_rules"`
	Overrides        []TaxOverride `json:"overrides,omitempty"`
	Notes            []string      `json:"notes,omitempty"`
}

// TaxOverride represents a manual tax override
type TaxOverride struct {
	Reason      string    `json:"reason"`
	OriginalTax int64     `json:"original_tax_cents"`
	NewTax      int64     `json:"new_tax_cents"`
	AppliedBy   uuid.UUID `json:"applied_by"`
	AppliedAt   time.Time `json:"applied_at"`
}

// TaxJurisdiction represents a tax jurisdiction with its rules
type TaxJurisdiction struct {
	Code          string             `json:"code"`
	Name          string             `json:"name"`
	Type          string             `json:"type"`       // "country", "state", "province", "city"
	TaxRates      map[string]float64 `json:"tax_rates"`  // product_type -> rate
	Thresholds    map[string]int64   `json:"thresholds"` // minimum amounts
	IsActive      bool               `json:"is_active"`
	EffectiveDate time.Time          `json:"effective_date"`
}
