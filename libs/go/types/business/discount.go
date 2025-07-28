package business

import "time"

// DiscountDetails contains detailed information about the discount application
type DiscountDetails struct {
	AppliedAt             time.Time `json:"applied_at"`
	ApplicationMethod     string    `json:"application_method"` // "code", "automatic", "admin"
	DurationMonths        *int32    `json:"duration_months,omitempty"`
	RecurringDiscount     bool      `json:"recurring_discount"`
	FirstTimeCustDiscount bool      `json:"first_time_customer_discount"`
}
