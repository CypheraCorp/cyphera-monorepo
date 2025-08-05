package services

import (
	"math"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/types/business"
)

// ProrationCalculator handles all proration calculations for subscription changes
type ProrationCalculator struct {
	// Can add timezone or calendar configurations here if needed
}

// NewProrationCalculator creates a new proration calculator
func NewProrationCalculator() *ProrationCalculator {
	return &ProrationCalculator{}
}

// CalculateUpgradeProration calculates proration for immediate upgrades
func (pc *ProrationCalculator) CalculateUpgradeProration(
	currentPeriodStart, currentPeriodEnd time.Time,
	oldAmountCents, newAmountCents int64,
	changeDate time.Time,
) *business.ProrationResult {
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

	// Handle zero-day periods to avoid division by zero
	if totalDays == 0 {
		return &business.ProrationResult{
			CreditAmount:  0,
			ChargeAmount:  0,
			NetAmount:     0,
			DaysTotal:     totalDays,
			DaysUsed:      usedDays,
			DaysRemaining: remainingDays,
			OldDailyRate:  0,
			NewDailyRate:  0,
			Calculation: map[string]interface{}{
				"total_days":       totalDays,
				"used_days":        usedDays,
				"remaining_days":   remainingDays,
				"old_daily_rate":   0.0,
				"new_daily_rate":   0.0,
				"unused_credit":    int64(0),
				"new_charge":       int64(0),
				"calculation_date": changeDate,
			},
		}
	}

	// Calculate daily rates
	dailyRateOld := float64(oldAmountCents) / float64(totalDays)
	dailyRateNew := float64(newAmountCents) / float64(totalDays)

	// Calculate credit for unused time at old rate
	// Use safe conversion to avoid overflow with very large numbers
	var unusedCredit int64
	if remainingDays == 0 {
		unusedCredit = 0
	} else if oldAmountCents > math.MaxInt64/2 {
		// For very large amounts, use integer division to avoid overflow
		unusedCredit = (oldAmountCents / int64(totalDays)) * int64(remainingDays)
	} else {
		unusedCredit = int64(math.Round(dailyRateOld * float64(remainingDays)))
	}

	// Calculate charge for remaining time at new rate
	var newCharge int64
	if remainingDays == 0 {
		newCharge = 0
	} else if newAmountCents > math.MaxInt64/2 {
		// For very large amounts, use integer division to avoid overflow
		newCharge = (newAmountCents / int64(totalDays)) * int64(remainingDays)
	} else {
		newCharge = int64(math.Round(dailyRateNew * float64(remainingDays)))
	}

	// Net amount to charge now (could be negative if downgrading)
	// Check for overflow in subtraction
	var immediateCharge int64
	if newCharge > math.MaxInt64/2 && unusedCredit < 0 {
		// Risk of overflow in subtraction
		immediateCharge = math.MaxInt64
	} else if newCharge < math.MinInt64/2 && unusedCredit > 0 {
		// Risk of underflow in subtraction
		immediateCharge = math.MinInt64
	} else {
		immediateCharge = newCharge - unusedCredit
	}

	return &business.ProrationResult{
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
) *business.ScheduleChangeResult {
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

	return &business.ScheduleChangeResult{
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
) *business.ProrationResult {
	// Similar to upgrade calculation but only credit, no new charge
	totalDays := pc.DaysBetween(currentPeriodStart, currentPeriodEnd)

	// If pause date is after period end, use period end for calculation
	var usedDays int
	if pauseDate.After(currentPeriodEnd) {
		// Entire period has been used
		usedDays = totalDays
	} else {
		usedDays = pc.DaysBetween(currentPeriodStart, pauseDate)
	}

	remainingDays := totalDays - usedDays
	if remainingDays < 0 {
		remainingDays = 0
	}

	dailyRate := float64(amountCents) / float64(totalDays)
	unusedCredit := int64(math.Round(dailyRate * float64(remainingDays)))

	return &business.ProrationResult{
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
	// Normalize both times to their respective timezones' midnight to handle DST correctly
	startNorm := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endNorm := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())

	// Convert to UTC for consistent calculation
	startUTC := startNorm.UTC()
	endUTC := endNorm.UTC()

	duration := endUTC.Sub(startUTC)
	days := int(duration.Hours() / 24)

	// Handle DST edge cases where hour difference might not be exactly divisible by 24
	if duration.Hours() >= 0 && duration.Hours() < 24 && days == 0 {
		// If it's less than 24 hours but on different calendar days, it's 1 day
		if start.Day() != end.Day() || start.Month() != end.Month() || start.Year() != end.Year() {
			days = 1
		} else if !start.Equal(end) && start.Day() == end.Day() && start.Month() == end.Month() && start.Year() == end.Year() {
			// Special case: different times on same calendar day
			// If neither time is at the very beginning or end of day (for proration purposes)
			startHour := start.Hour()*60 + start.Minute()
			endHour := end.Hour()*60 + end.Minute()

			// If both times are during "business hours" (not at exact midnight or very end of day)
			if (startHour > 0 && startHour < 1440) && (endHour > 0 && endHour < 1439) {
				days = 1
			}
		}
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
		return pc.addMonthsWithEndOfMonthHandling(start, intervalCount)
	case "yearly":
		return start.AddDate(intervalCount, 0, 0)
	default:
		// Default to monthly if unknown
		return pc.addMonthsWithEndOfMonthHandling(start, intervalCount)
	}
}

// addMonthsWithEndOfMonthHandling adds months while preserving end-of-month behavior
func (pc *ProrationCalculator) addMonthsWithEndOfMonthHandling(start time.Time, months int) time.Time {
	// Get the target month/year
	targetYear := start.Year()
	targetMonth := int(start.Month()) + months

	// Handle year overflow/underflow
	for targetMonth > 12 {
		targetYear++
		targetMonth -= 12
	}
	for targetMonth < 1 {
		targetYear--
		targetMonth += 12
	}

	// Check if original date was end of month
	nextMonth := start.AddDate(0, 1, 0)
	isEndOfMonth := start.Day() > nextMonth.Day() || start.AddDate(0, 0, 1).Day() == 1

	if isEndOfMonth {
		// Find the last day of the target month
		lastDayOfMonth := time.Date(targetYear, time.Month(targetMonth+1), 0, start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), start.Location())
		return lastDayOfMonth
	}

	// Use standard AddDate for non-end-of-month dates
	result := start.AddDate(0, months, 0)

	// If we overflowed (e.g., Jan 31 + 1 month = Mar 2), adjust to end of target month
	if result.Month() != time.Month(targetMonth) {
		lastDayOfMonth := time.Date(targetYear, time.Month(targetMonth+1), 0, start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), start.Location())
		return lastDayOfMonth
	}

	return result
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
func (pc *ProrationCalculator) FormatProrationExplanation(result *business.ProrationResult) string {
	if result == nil {
		return "No proration calculation available."
	}

	if result.NetAmount > 0 {
		return "You'll be charged for the upgraded service for the remainder of your billing period."
	} else if result.NetAmount < 0 {
		return "You'll receive a credit for the unused portion of your current billing period."
	}
	return "No additional charge for this change."
}
