package services

import (
	"math"
	"time"
)

// ProrationCalculator handles all proration calculations for subscription changes
type ProrationCalculator struct {
	// Can add timezone or calendar configurations here if needed
}

// NewProrationCalculator creates a new proration calculator
func NewProrationCalculator() *ProrationCalculator {
	return &ProrationCalculator{}
}

// ProrationResult contains the result of a proration calculation
type ProrationResult struct {
	CreditAmount          int64                  `json:"credit_amount"`
	ChargeAmount          int64                  `json:"charge_amount"`
	NetAmount             int64                  `json:"net_amount"`
	DaysTotal             int                    `json:"days_total"`
	DaysUsed              int                    `json:"days_used"`
	DaysRemaining         int                    `json:"days_remaining"`
	OldDailyRate          float64                `json:"old_daily_rate"`
	NewDailyRate          float64                `json:"new_daily_rate"`
	Calculation           map[string]interface{} `json:"calculation"`
}

// ScheduleChangeResult contains information about a scheduled change
type ScheduleChangeResult struct {
	ScheduledFor time.Time `json:"scheduled_for"`
	ChangeType   string    `json:"change_type"`
	NoProration  bool      `json:"no_proration"`
	Message      string    `json:"message"`
}

// CalculateUpgradeProration calculates proration for immediate upgrades
func (pc *ProrationCalculator) CalculateUpgradeProration(
	currentPeriodStart, currentPeriodEnd time.Time,
	oldAmountCents, newAmountCents int64,
	changeDate time.Time,
) *ProrationResult {
	// Calculate days in the billing period
	totalDays := pc.DaysBetween(currentPeriodStart, currentPeriodEnd)
	usedDays := pc.DaysBetween(currentPeriodStart, changeDate)
	remainingDays := totalDays - usedDays

	// Ensure we don't have negative days
	if remainingDays < 0 {
		remainingDays = 0
	}
	if usedDays > totalDays {
		usedDays = totalDays
	}

	// Calculate daily rates
	dailyRateOld := float64(oldAmountCents) / float64(totalDays)
	dailyRateNew := float64(newAmountCents) / float64(totalDays)

	// Calculate credit for unused time at old rate
	unusedCredit := int64(math.Round(dailyRateOld * float64(remainingDays)))

	// Calculate charge for remaining time at new rate
	newCharge := int64(math.Round(dailyRateNew * float64(remainingDays)))

	// Net amount to charge now (could be negative if downgrading)
	immediateCharge := newCharge - unusedCredit

	return &ProrationResult{
		CreditAmount:  unusedCredit,
		ChargeAmount:  newCharge,
		NetAmount:     immediateCharge,
		DaysTotal:     totalDays,
		DaysUsed:      usedDays,
		DaysRemaining: remainingDays,
		OldDailyRate:  dailyRateOld,
		NewDailyRate:  dailyRateNew,
		Calculation: map[string]interface{}{
			"total_days":       totalDays,
			"used_days":        usedDays,
			"remaining_days":   remainingDays,
			"old_daily_rate":   dailyRateOld,
			"new_daily_rate":   dailyRateNew,
			"unused_credit":    unusedCredit,
			"new_charge":       newCharge,
			"calculation_date": changeDate,
		},
	}
}

// ScheduleDowngrade schedules a downgrade for the end of the current period
func (pc *ProrationCalculator) ScheduleDowngrade(
	currentPeriodEnd time.Time,
	changeType string,
) *ScheduleChangeResult {
	// Downgrades and cancellations happen at end of current period
	// No proration needed as customer keeps current service until then
	message := ""
	switch changeType {
	case "downgrade":
		message = "Downgrade scheduled for end of billing period. You'll continue with current plan until then."
	case "cancel":
		message = "Cancellation scheduled for end of billing period. You'll have access until then."
	default:
		message = "Change scheduled for end of billing period."
	}

	return &ScheduleChangeResult{
		ScheduledFor: currentPeriodEnd,
		ChangeType:   changeType,
		NoProration:  true,
		Message:      message,
	}
}

// CalculatePauseCredit calculates any credit due when pausing a subscription
func (pc *ProrationCalculator) CalculatePauseCredit(
	currentPeriodStart, currentPeriodEnd time.Time,
	amountCents int64,
	pauseDate time.Time,
) *ProrationResult {
	// Similar to upgrade calculation but only credit, no new charge
	totalDays := pc.DaysBetween(currentPeriodStart, currentPeriodEnd)
	usedDays := pc.DaysBetween(currentPeriodStart, pauseDate)
	remainingDays := totalDays - usedDays

	if remainingDays < 0 {
		remainingDays = 0
	}

	dailyRate := float64(amountCents) / float64(totalDays)
	unusedCredit := int64(math.Round(dailyRate * float64(remainingDays)))

	return &ProrationResult{
		CreditAmount:  unusedCredit,
		ChargeAmount:  0,
		NetAmount:     -unusedCredit, // Negative because it's a credit
		DaysTotal:     totalDays,
		DaysUsed:      usedDays,
		DaysRemaining: remainingDays,
		OldDailyRate:  dailyRate,
		NewDailyRate:  0,
		Calculation: map[string]interface{}{
			"total_days":     totalDays,
			"used_days":      usedDays,
			"remaining_days": remainingDays,
			"daily_rate":     dailyRate,
			"pause_credit":   unusedCredit,
			"pause_date":     pauseDate,
		},
	}
}

// DaysBetween calculates the number of days between two dates
func (pc *ProrationCalculator) DaysBetween(start, end time.Time) int {
	// Normalize times to beginning of day for consistent calculation
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	
	duration := end.Sub(start)
	days := int(duration.Hours() / 24)
	
	// Ensure at least 1 day for same-day calculations
	if days == 0 && !start.Equal(end) {
		days = 1
	}
	
	return days
}

// AddBillingPeriod adds the appropriate billing period to a date
func (pc *ProrationCalculator) AddBillingPeriod(start time.Time, intervalType string, intervalCount int) time.Time {
	switch intervalType {
	case "daily":
		return start.AddDate(0, 0, intervalCount)
	case "weekly":
		return start.AddDate(0, 0, 7*intervalCount)
	case "monthly":
		return start.AddDate(0, intervalCount, 0)
	case "yearly":
		return start.AddDate(intervalCount, 0, 0)
	default:
		// Default to monthly if unknown
		return start.AddDate(0, intervalCount, 0)
	}
}

// CalculateTrialEndDate calculates when a trial period should end
func (pc *ProrationCalculator) CalculateTrialEndDate(start time.Time, trialDays int) time.Time {
	return start.AddDate(0, 0, trialDays)
}

// IsInTrial checks if a subscription is currently in trial period
func (pc *ProrationCalculator) IsInTrial(trialEnd *time.Time) bool {
	if trialEnd == nil {
		return false
	}
	return time.Now().Before(*trialEnd)
}

// GetDailyRate calculates the daily rate for a given amount and period
func (pc *ProrationCalculator) GetDailyRate(amountCents int64, periodStart, periodEnd time.Time) float64 {
	days := pc.DaysBetween(periodStart, periodEnd)
	if days == 0 {
		return 0
	}
	return float64(amountCents) / float64(days)
}

// FormatProrationExplanation creates a human-readable explanation of the proration
func (pc *ProrationCalculator) FormatProrationExplanation(result *ProrationResult) string {
	if result.NetAmount > 0 {
		return "You'll be charged for the upgraded service for the remainder of your billing period."
	} else if result.NetAmount < 0 {
		return "You'll receive a credit for the unused portion of your current billing period."
	}
	return "No additional charge for this change."
}