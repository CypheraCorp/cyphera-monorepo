package responses

import "github.com/google/uuid"

// GasSponsorshipConfigResponse represents a gas sponsorship configuration
type GasSponsorshipConfigResponse struct {
	WorkspaceID              uuid.UUID   `json:"workspace_id"`
	SponsorshipEnabled       bool        `json:"sponsorship_enabled"`
	SponsorCustomerGas       bool        `json:"sponsor_customer_gas"`
	SponsorThresholdUsdCents *int64      `json:"sponsor_threshold_usd_cents,omitempty"`
	MonthlyBudgetUsdCents    *int64      `json:"monthly_budget_usd_cents,omitempty"`
	SponsorForProducts       []uuid.UUID `json:"sponsor_for_products"`
	SponsorForCustomers      []uuid.UUID `json:"sponsor_for_customers"`
	SponsorForTiers          []string    `json:"sponsor_for_tiers"`
	CurrentMonthSpentCents   int64       `json:"current_month_spent_cents"`
	RemainingBudgetCents     *int64      `json:"remaining_budget_cents,omitempty"`
}
