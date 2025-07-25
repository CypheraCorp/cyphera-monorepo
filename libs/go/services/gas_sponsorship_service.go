package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// GasSponsorshipService manages gas fee sponsorship logic
type GasSponsorshipService struct {
	queries db.Querier
	logger  *zap.Logger
}

// SponsorshipCheckParams contains parameters for checking sponsorship eligibility
type SponsorshipCheckParams struct {
	WorkspaceID      uuid.UUID
	CustomerID       uuid.UUID
	ProductID        uuid.UUID
	CustomerTier     string // e.g., "bronze", "silver", "gold", "platinum"
	GasCostUSDCents  int64
	TransactionType  string // e.g., "subscription", "one_time", "refund"
}

// SponsorshipDecision contains the result of a sponsorship check
type SponsorshipDecision struct {
	ShouldSponsor    bool
	SponsorType      string    // "merchant", "platform", "third_party"
	SponsorID        uuid.UUID // ID of the sponsoring entity
	Reason           string    // Human-readable reason for the decision
	RemainingBudget  int64     // Remaining monthly budget in cents
}

// SponsorshipRecord contains details of a sponsored transaction
type SponsorshipRecord struct {
	WorkspaceID      uuid.UUID
	PaymentID        uuid.UUID
	GasCostUSDCents  int64
	SponsorType      string
	SponsorID        uuid.UUID
}

// BudgetStatus contains current sponsorship budget information
type BudgetStatus struct {
	WorkspaceID              uuid.UUID
	MonthlyBudgetCents       int64
	CurrentMonthSpentCents   int64
	RemainingBudgetCents     int64
	LastResetDate            time.Time
	SponsorshipEnabled       bool
}

// NewGasSponsorshipService creates a new gas sponsorship service
func NewGasSponsorshipService(queries db.Querier) *GasSponsorshipService {
	return &GasSponsorshipService{
		queries: queries,
		logger:  logger.Log,
	}
}

// ShouldSponsorGas determines if gas fees should be sponsored for a transaction
func (s *GasSponsorshipService) ShouldSponsorGas(ctx context.Context, params SponsorshipCheckParams) (*SponsorshipDecision, error) {
	// Default decision is no sponsorship
	decision := &SponsorshipDecision{
		ShouldSponsor: false,
		SponsorType:   "customer",
		Reason:        "No sponsorship configured",
	}

	// Get sponsorship configuration for the workspace
	config, err := s.queries.GetGasSponsorshipConfig(ctx, params.WorkspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No sponsorship config exists
			return decision, nil
		}
		return nil, fmt.Errorf("failed to get sponsorship config: %w", err)
	}

	// Check if sponsorship is enabled
	if !config.SponsorshipEnabled.Bool || !config.SponsorCustomerGas.Bool {
		decision.Reason = "Sponsorship not enabled for workspace"
		return decision, nil
	}

	// Check monthly budget
	if config.MonthlyBudgetUsdCents.Valid {
		remainingBudget := config.MonthlyBudgetUsdCents.Int64 - config.CurrentMonthSpentCents.Int64
		decision.RemainingBudget = remainingBudget
		
		if remainingBudget < params.GasCostUSDCents {
			decision.Reason = "Monthly sponsorship budget exhausted"
			return decision, nil
		}
	}

	// Check per-transaction threshold
	if config.SponsorThresholdUsdCents.Valid && params.GasCostUSDCents > config.SponsorThresholdUsdCents.Int64 {
		decision.Reason = fmt.Sprintf("Gas cost exceeds threshold (%d cents > %d cents)", 
			params.GasCostUSDCents, config.SponsorThresholdUsdCents.Int64)
		return decision, nil
	}

	// Check product-specific rules
	if len(config.SponsorForProducts) > 0 {
		var productIDs []uuid.UUID
		if err := json.Unmarshal(config.SponsorForProducts, &productIDs); err == nil && len(productIDs) > 0 {
			found := false
			for _, pid := range productIDs {
				if pid == params.ProductID {
					found = true
					break
				}
			}
			if !found {
				decision.Reason = "Product not eligible for sponsorship"
				return decision, nil
			}
		}
	}

	// Check customer-specific rules
	if len(config.SponsorForCustomers) > 0 {
		var customerIDs []uuid.UUID
		if err := json.Unmarshal(config.SponsorForCustomers, &customerIDs); err == nil && len(customerIDs) > 0 {
			found := false
			for _, cid := range customerIDs {
				if cid == params.CustomerID {
					found = true
					break
				}
			}
			if !found {
				decision.Reason = "Customer not eligible for sponsorship"
				return decision, nil
			}
		}
	}

	// Check tier-specific rules
	if len(config.SponsorForTiers) > 0 && params.CustomerTier != "" {
		var tiers []string
		if err := json.Unmarshal(config.SponsorForTiers, &tiers); err == nil && len(tiers) > 0 {
			found := false
			for _, tier := range tiers {
				if tier == params.CustomerTier {
					found = true
					break
				}
			}
			if !found {
				decision.Reason = "Customer tier not eligible for sponsorship"
				return decision, nil
			}
		}
	}

	// All checks passed - approve sponsorship
	decision.ShouldSponsor = true
	decision.SponsorType = "merchant"
	decision.SponsorID = params.WorkspaceID
	decision.Reason = "Sponsorship approved"
	
	return decision, nil
}

// RecordSponsoredTransaction records a gas sponsorship transaction
func (s *GasSponsorshipService) RecordSponsoredTransaction(ctx context.Context, record SponsorshipRecord) error {
	// First get current spending to calculate new total
	config, err := s.queries.GetGasSponsorshipConfig(ctx, record.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get current spending: %w", err)
	}
	
	newTotal := config.CurrentMonthSpentCents.Int64 + record.GasCostUSDCents
	
	// Update the current month's spending
	err = s.queries.UpdateGasSponsorshipSpending(ctx, db.UpdateGasSponsorshipSpendingParams{
		WorkspaceID:            record.WorkspaceID,
		CurrentMonthSpentCents: pgtype.Int8{Int64: newTotal, Valid: true},
		UpdatedAt:              pgtype.Timestamptz{Valid: false}, // Use CURRENT_TIMESTAMP
	})
	
	if err != nil {
		return fmt.Errorf("failed to update sponsorship spending: %w", err)
	}

	s.logger.Info("Recorded sponsored gas transaction",
		zap.String("workspace_id", record.WorkspaceID.String()),
		zap.String("payment_id", record.PaymentID.String()),
		zap.Int64("gas_cost_cents", record.GasCostUSDCents),
		zap.String("sponsor_type", record.SponsorType),
	)

	return nil
}

// GetSponsorshipBudgetStatus returns the current sponsorship budget status
func (s *GasSponsorshipService) GetSponsorshipBudgetStatus(ctx context.Context, workspaceID uuid.UUID) (*BudgetStatus, error) {
	config, err := s.queries.GetGasSponsorshipConfig(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Return a default status if no config exists
			return &BudgetStatus{
				WorkspaceID:        workspaceID,
				SponsorshipEnabled: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get sponsorship config: %w", err)
	}

	status := &BudgetStatus{
		WorkspaceID:            workspaceID,
		CurrentMonthSpentCents: config.CurrentMonthSpentCents.Int64,
		SponsorshipEnabled:     config.SponsorshipEnabled.Bool,
	}

	if config.MonthlyBudgetUsdCents.Valid {
		status.MonthlyBudgetCents = config.MonthlyBudgetUsdCents.Int64
		status.RemainingBudgetCents = config.MonthlyBudgetUsdCents.Int64 - config.CurrentMonthSpentCents.Int64
	}

	if config.LastResetDate.Valid {
		status.LastResetDate = config.LastResetDate.Time
	}

	return status, nil
}

// ResetMonthlySponsorshipBudgets resets monthly counters for all workspaces
// This should be called by a scheduled job at the start of each month
func (s *GasSponsorshipService) ResetMonthlySponsorshipBudgets(ctx context.Context) error {
	// Get all workspaces with sponsorship configs that need reset
	now := time.Now()
	dateParam := pgtype.Date{Time: now, Valid: true}
	configs, err := s.queries.GetSponsorshipConfigsNeedingReset(ctx, dateParam)
	if err != nil {
		return fmt.Errorf("failed to get configs needing reset: %w", err)
	}

	resetCount := 0
	for _, config := range configs {
		err := s.queries.ResetGasSponsorshipMonthlySpending(ctx, db.ResetGasSponsorshipMonthlySpendingParams{
			WorkspaceID:   config.WorkspaceID,
			LastResetDate: pgtype.Date{Time: time.Now(), Valid: true},
			UpdatedAt:     pgtype.Timestamptz{Valid: false}, // Use CURRENT_TIMESTAMP
		})
		
		if err != nil {
			s.logger.Error("Failed to reset sponsorship budget",
				zap.String("workspace_id", config.WorkspaceID.String()),
				zap.Error(err),
			)
			continue
		}
		
		resetCount++
	}

	s.logger.Info("Reset monthly sponsorship budgets",
		zap.Int("reset_count", resetCount),
		zap.Int("total_configs", len(configs)),
	)

	return nil
}

// CreateDefaultSponsorshipConfig creates a default sponsorship configuration for a workspace
func (s *GasSponsorshipService) CreateDefaultSponsorshipConfig(ctx context.Context, workspaceID uuid.UUID) error {
	_, err := s.queries.CreateGasSponsorshipConfig(ctx, db.CreateGasSponsorshipConfigParams{
		WorkspaceID:              workspaceID,
		SponsorshipEnabled:       pgtype.Bool{Bool: false, Valid: true},
		SponsorCustomerGas:       pgtype.Bool{Bool: false, Valid: true},
		SponsorThresholdUsdCents: pgtype.Int8{Int64: 100, Valid: true}, // Default $1.00 threshold
		MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 10000, Valid: true}, // Default $100 monthly budget
		SponsorForProducts:       []byte("[]"),
		SponsorForCustomers:      []byte("[]"),
		SponsorForTiers:          []byte("[]"),
	})

	if err != nil {
		return fmt.Errorf("failed to create default sponsorship config: %w", err)
	}

	return nil
}

// UpdateSponsorshipConfig updates sponsorship configuration for a workspace
func (s *GasSponsorshipService) UpdateSponsorshipConfig(ctx context.Context, workspaceID uuid.UUID, updates db.UpdateGasSponsorshipConfigParams) error {
	updates.WorkspaceID = workspaceID
	
	_, err := s.queries.UpdateGasSponsorshipConfig(ctx, updates)
	if err != nil {
		return fmt.Errorf("failed to update sponsorship config: %w", err)
	}

	return nil
}