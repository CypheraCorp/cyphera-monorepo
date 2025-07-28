package business

import (
	"time"

	"github.com/google/uuid"
)

// SponsorshipDecision contains the result of a sponsorship check
type SponsorshipDecision struct {
	ShouldSponsor   bool
	SponsorType     string    // "merchant", "platform", "third_party"
	SponsorID       uuid.UUID // ID of the sponsoring entity
	Reason          string    // Human-readable reason for the decision
	RemainingBudget int64     // Remaining monthly budget in cents
}

// SponsorshipRecord contains details of a sponsored transaction
type SponsorshipRecord struct {
	WorkspaceID     uuid.UUID
	PaymentID       uuid.UUID
	GasCostUSDCents int64
	SponsorType     string
	SponsorID       uuid.UUID
}

// BudgetStatus contains current sponsorship budget information
type BudgetStatus struct {
	WorkspaceID            uuid.UUID
	MonthlyBudgetCents     int64
	CurrentMonthSpentCents int64
	RemainingBudgetCents   int64
	LastResetDate          time.Time
	SponsorshipEnabled     bool
}

// SponsorshipConfigUpdates contains fields that can be updated in sponsorship config
type SponsorshipConfigUpdates struct {
	SponsorshipEnabled       *bool
	SponsorCustomerGas       *bool
	MonthlyBudgetUSDCents    *int64
	SponsorThresholdUSDCents *int64
	SponsorForProducts       *[]uuid.UUID
	SponsorForCustomers      *[]uuid.UUID
	SponsorForTiers          *[]string
}

// SponsorshipAnalytics contains analytics data for gas sponsorship
type SponsorshipAnalytics struct {
	TotalTransactions     int64   `json:"total_transactions"`
	SponsoredTransactions int64   `json:"sponsored_transactions"`
	TotalGasCostCents     int64   `json:"total_gas_cost_cents"`
	SponsoredCostCents    int64   `json:"sponsored_cost_cents"`
	SavingsPercentage     float64 `json:"savings_percentage"`
	Period                int     `json:"period_days"`
}
