package services

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
)

// GasSponsorshipHelper provides convenience methods for gas sponsorship
type GasSponsorshipHelper struct {
	service *GasSponsorshipService
}

// NewGasSponsorshipHelper creates a new gas sponsorship helper
func NewGasSponsorshipHelper(queries db.Querier) *GasSponsorshipHelper {
	return &GasSponsorshipHelper{
		service: NewGasSponsorshipService(queries),
	}
}

// QuickSponsorshipCheck performs a simple sponsorship check for common use cases
func (h *GasSponsorshipHelper) QuickSponsorshipCheck(
	ctx context.Context,
	workspaceID uuid.UUID,
	customerID uuid.UUID,
	productID uuid.UUID,
	gasCostCents int64,
) (shouldSponsor bool, sponsorType string, err error) {
	params := params.SponsorshipCheckParams{
		WorkspaceID:     workspaceID,
		CustomerID:      customerID,
		ProductID:       productID,
		GasCostUSDCents: gasCostCents,
		TransactionType: "subscription",
	}

	decision, err := h.service.ShouldSponsorGas(ctx, params)
	if err != nil {
		return false, "customer", err
	}

	return decision.ShouldSponsor, decision.SponsorType, nil
}

// ApplySponsorshipToPayment applies sponsorship decision to payment parameters
func (h *GasSponsorshipHelper) ApplySponsorshipToPayment(
	ctx context.Context,
	workspaceID uuid.UUID,
	customerID uuid.UUID,
	productID uuid.UUID,
	gasCostCents int64,
) (gasSponsored bool, sponsorWorkspaceID *uuid.UUID, err error) {
	// Check if gas should be sponsored
	shouldSponsor, sponsorType, err := h.QuickSponsorshipCheck(
		ctx,
		workspaceID,
		customerID,
		productID,
		gasCostCents,
	)

	if err != nil {
		return false, nil, err
	}

	if shouldSponsor && sponsorType == "merchant" {
		return true, &workspaceID, nil
	}

	return false, nil, nil
}

// RecordSponsorship records a sponsored transaction
func (h *GasSponsorshipHelper) RecordSponsorship(
	ctx context.Context,
	workspaceID uuid.UUID,
	paymentID uuid.UUID,
	gasCostCents int64,
) error {
	record := business.SponsorshipRecord{
		WorkspaceID:     workspaceID,
		PaymentID:       paymentID,
		GasCostUSDCents: gasCostCents,
		SponsorType:     "merchant",
		SponsorID:       workspaceID,
	}

	return h.service.RecordSponsoredTransaction(ctx, record)
}

// GetService returns the underlying gas sponsorship service
func (h *GasSponsorshipHelper) GetService() *GasSponsorshipService {
	return h.service
}
