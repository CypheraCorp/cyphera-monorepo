package requests

import "github.com/google/uuid"

// GasSponsorshipConfigRequest represents the request to create/update gas sponsorship config
type GasSponsorshipConfigRequest struct {
	SponsorshipEnabled       bool        `json:"sponsorship_enabled"`
	SponsorCustomerGas       bool        `json:"sponsor_customer_gas"`
	SponsorThresholdUsdCents *int64      `json:"sponsor_threshold_usd_cents,omitempty"`
	MonthlyBudgetUsdCents    *int64      `json:"monthly_budget_usd_cents,omitempty"`
	SponsorForProducts       []uuid.UUID `json:"sponsor_for_products,omitempty"`
	SponsorForCustomers      []uuid.UUID `json:"sponsor_for_customers,omitempty"`
	SponsorForTiers          []string    `json:"sponsor_for_tiers,omitempty"`
}
