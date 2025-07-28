package services

import (
	"context"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"go.uber.org/zap"
)

// DiscountService handles discount code validation and application logic
// NOTE: This is a temporary implementation until database schema is created
type DiscountService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewDiscountService creates a new discount service
func NewDiscountService(queries db.Querier) *DiscountService {
	return &DiscountService{
		queries: queries,
		logger:  logger.Log,
	}
}

// ApplyDiscount validates and applies a discount code
func (s *DiscountService) ApplyDiscount(ctx context.Context, params params.DiscountApplicationParams) (*responses.DiscountApplicationResult, error) {
	if s.logger != nil {
		s.logger.Info("Discount service not yet implemented",
			zap.String("workspace_id", params.WorkspaceID.String()),
			zap.String("discount_code", params.DiscountCode))
	}

	// Return no discount applied
	reason := "Discount functionality not yet implemented"
	return &responses.DiscountApplicationResult{
		DiscountCode:        params.DiscountCode,
		OriginalAmountCents: params.AmountCents,
		FinalAmountCents:    params.AmountCents,
		IsValid:             false,
		ReasonForInvalidity: &reason,
		ApplicationDetails: business.DiscountDetails{
			AppliedAt:         time.Now(),
			ApplicationMethod: "code",
		},
	}, nil
}
