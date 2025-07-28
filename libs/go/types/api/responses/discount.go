package responses

import (
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
)

// DiscountApplicationResult contains the result of applying a discount
type DiscountApplicationResult struct {
	DiscountCode        string                   `json:"discount_code"`
	DiscountID          *uuid.UUID               `json:"discount_id,omitempty"`
	OriginalAmountCents int64                    `json:"original_amount_cents"`
	DiscountAmountCents int64                    `json:"discount_amount_cents"`
	FinalAmountCents    int64                    `json:"final_amount_cents"`
	DiscountPercentage  float64                  `json:"discount_percentage"`
	IsValid             bool                     `json:"is_valid"`
	ReasonForInvalidity *string                  `json:"reason_for_invalidity,omitempty"`
	ApplicationDetails  business.DiscountDetails `json:"application_details"`
}
