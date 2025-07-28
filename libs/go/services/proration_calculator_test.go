package services_test

import (
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger.InitLogger("test")
}

func TestProrationCalculator_CalculateUpgradeProration(t *testing.T) {
	calculator := services.NewProrationCalculator()

	// Base test dates for consistency
	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC) // 30 days

	tests := []struct {
		name                  string
		currentPeriodStart    time.Time
		currentPeriodEnd      time.Time
		oldAmountCents        int64
		newAmountCents        int64
		changeDate            time.Time
		expectedNetAmount     int64
		expectedDaysTotal     int
		expectedDaysUsed      int
		expectedDaysRemaining int
		expectedCreditAmount  int64
		expectedChargeAmount  int64
	}{
		{
			name:                  "upgrade halfway through monthly period",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        1000,                                         // $10.00
			newAmountCents:        2000,                                         // $20.00
			changeDate:            time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), // Day 15
			expectedNetAmount:     500,                                          // $10.00 more for remaining days (1000 - 500)
			expectedDaysTotal:     30,
			expectedDaysUsed:      15,
			expectedDaysRemaining: 15,
			expectedCreditAmount:  500,  // 15 days * ($10/30 days)
			expectedChargeAmount:  1000, // 15 days * ($20/30 days)
		},
		{
			name:                  "upgrade at beginning of period",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        1000,
			newAmountCents:        3000,
			changeDate:            periodStart,
			expectedNetAmount:     2000, // Full difference for entire period
			expectedDaysTotal:     30,
			expectedDaysUsed:      0,
			expectedDaysRemaining: 30,
			expectedCreditAmount:  1000, // Full old amount
			expectedChargeAmount:  3000, // Full new amount
		},
		{
			name:                  "upgrade at end of period",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        1000,
			newAmountCents:        2000,
			changeDate:            periodEnd,
			expectedNetAmount:     0, // No remaining days
			expectedDaysTotal:     30,
			expectedDaysUsed:      30,
			expectedDaysRemaining: 0,
			expectedCreditAmount:  0,
			expectedChargeAmount:  0,
		},
		{
			name:                  "downgrade (negative net amount)",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        2000,
			newAmountCents:        1000,
			changeDate:            time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			expectedNetAmount:     -500, // Credit back the difference
			expectedDaysTotal:     30,
			expectedDaysUsed:      15,
			expectedDaysRemaining: 15,
			expectedCreditAmount:  1000, // 15 days * ($20/30 days)
			expectedChargeAmount:  500,  // 15 days * ($10/30 days)
		},
		{
			name:                  "same price (no net change)",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        1500,
			newAmountCents:        1500,
			changeDate:            time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			expectedNetAmount:     0,
			expectedDaysTotal:     30,
			expectedDaysUsed:      15,
			expectedDaysRemaining: 15,
			expectedCreditAmount:  750,
			expectedChargeAmount:  750,
		},
		{
			name:                  "change date after period end",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        1000,
			newAmountCents:        2000,
			changeDate:            time.Date(2024, 2, 5, 0, 0, 0, 0, time.UTC), // After period
			expectedNetAmount:     0,                                           // No remaining days
			expectedDaysTotal:     30,
			expectedDaysUsed:      30,
			expectedDaysRemaining: 0,
			expectedCreditAmount:  0,
			expectedChargeAmount:  0,
		},
		{
			name:                  "single day period",
			currentPeriodStart:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			currentPeriodEnd:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			oldAmountCents:        100,
			newAmountCents:        200,
			changeDate:            time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedNetAmount:     0, // Same day, no proration
			expectedDaysTotal:     0,
			expectedDaysUsed:      0,
			expectedDaysRemaining: 0,
		},
		{
			name:                  "weekly billing period upgrade",
			currentPeriodStart:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			currentPeriodEnd:      time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC), // 7 days
			oldAmountCents:        700,                                         // $7.00
			newAmountCents:        1400,                                        // $14.00
			changeDate:            time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), // Day 3
			expectedNetAmount:     400,                                         // 4 days * ($7 difference per day)
			expectedDaysTotal:     7,
			expectedDaysUsed:      3,
			expectedDaysRemaining: 4,
			expectedCreditAmount:  400, // 4 days * ($7/7 days)
			expectedChargeAmount:  800, // 4 days * ($14/7 days)
		},
		{
			name:                  "zero amount subscription",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        0,
			newAmountCents:        1000,
			changeDate:            time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			expectedNetAmount:     500, // Just the new charge
			expectedDaysTotal:     30,
			expectedDaysUsed:      15,
			expectedDaysRemaining: 15,
			expectedCreditAmount:  0,
			expectedChargeAmount:  500,
		},
		{
			name:                  "upgrade to zero amount (cancellation)",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			oldAmountCents:        1000,
			newAmountCents:        0,
			changeDate:            time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			expectedNetAmount:     -500, // Credit back unused portion
			expectedDaysTotal:     30,
			expectedDaysUsed:      15,
			expectedDaysRemaining: 15,
			expectedCreditAmount:  500,
			expectedChargeAmount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.CalculateUpgradeProration(
				tt.currentPeriodStart,
				tt.currentPeriodEnd,
				tt.oldAmountCents,
				tt.newAmountCents,
				tt.changeDate,
			)

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedNetAmount, result.NetAmount)
			assert.Equal(t, tt.expectedDaysTotal, result.DaysTotal)
			assert.Equal(t, tt.expectedDaysUsed, result.DaysUsed)
			assert.Equal(t, tt.expectedDaysRemaining, result.DaysRemaining)
			assert.Equal(t, tt.expectedCreditAmount, result.CreditAmount)
			assert.Equal(t, tt.expectedChargeAmount, result.ChargeAmount)

			// Verify calculation breakdown
			assert.NotNil(t, result.Calculation)
			assert.Contains(t, result.Calculation, "total_days")
			assert.Contains(t, result.Calculation, "used_days")
			assert.Contains(t, result.Calculation, "remaining_days")
			assert.Contains(t, result.Calculation, "old_daily_rate")
			assert.Contains(t, result.Calculation, "new_daily_rate")
			assert.Contains(t, result.Calculation, "calculation_date")

			// Verify daily rates are calculated correctly
			if tt.expectedDaysTotal > 0 {
				expectedOldRate := float64(tt.oldAmountCents) / float64(tt.expectedDaysTotal)
				expectedNewRate := float64(tt.newAmountCents) / float64(tt.expectedDaysTotal)
				assert.InDelta(t, expectedOldRate, result.OldDailyRate, 0.01)
				assert.InDelta(t, expectedNewRate, result.NewDailyRate, 0.01)
			}
		})
	}
}

func TestProrationCalculator_ScheduleDowngrade(t *testing.T) {
	calculator := services.NewProrationCalculator()

	periodEnd := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)

	tests := []struct {
		name                string
		currentPeriodEnd    time.Time
		changeType          string
		expectedScheduled   time.Time
		expectedNoProration bool
		expectedMessage     string
	}{
		{
			name:                "schedule downgrade",
			currentPeriodEnd:    periodEnd,
			changeType:          "downgrade",
			expectedScheduled:   periodEnd,
			expectedNoProration: true,
			expectedMessage:     "Downgrade scheduled for end of billing period. You'll continue with current plan until then.",
		},
		{
			name:                "schedule cancellation",
			currentPeriodEnd:    periodEnd,
			changeType:          "cancel",
			expectedScheduled:   periodEnd,
			expectedNoProration: true,
			expectedMessage:     "Cancellation scheduled for end of billing period. You'll have access until then.",
		},
		{
			name:                "schedule unknown change type",
			currentPeriodEnd:    periodEnd,
			changeType:          "pause",
			expectedScheduled:   periodEnd,
			expectedNoProration: true,
			expectedMessage:     "Change scheduled for end of billing period.",
		},
		{
			name:                "schedule change with empty type",
			currentPeriodEnd:    periodEnd,
			changeType:          "",
			expectedScheduled:   periodEnd,
			expectedNoProration: true,
			expectedMessage:     "Change scheduled for end of billing period.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.ScheduleDowngrade(tt.currentPeriodEnd, tt.changeType)

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedScheduled, result.ScheduledFor)
			assert.Equal(t, tt.changeType, result.ChangeType)
			assert.Equal(t, tt.expectedNoProration, result.NoProration)
			assert.Equal(t, tt.expectedMessage, result.Message)
		})
	}
}

func TestProrationCalculator_CalculatePauseCredit(t *testing.T) {
	calculator := services.NewProrationCalculator()

	periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC) // 30 days

	tests := []struct {
		name                  string
		currentPeriodStart    time.Time
		currentPeriodEnd      time.Time
		amountCents           int64
		pauseDate             time.Time
		expectedCreditAmount  int64
		expectedChargeAmount  int64
		expectedNetAmount     int64
		expectedDaysTotal     int
		expectedDaysUsed      int
		expectedDaysRemaining int
	}{
		{
			name:                  "pause halfway through period",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			amountCents:           3000,                                         // $30.00
			pauseDate:             time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC), // Day 15
			expectedCreditAmount:  1500,                                         // 15 days * ($30/30 days)
			expectedChargeAmount:  0,
			expectedNetAmount:     -1500, // Negative because it's a credit
			expectedDaysTotal:     30,
			expectedDaysUsed:      15,
			expectedDaysRemaining: 15,
		},
		{
			name:                  "pause at beginning of period",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			amountCents:           3000,
			pauseDate:             periodStart,
			expectedCreditAmount:  3000, // Full credit
			expectedChargeAmount:  0,
			expectedNetAmount:     -3000,
			expectedDaysTotal:     30,
			expectedDaysUsed:      0,
			expectedDaysRemaining: 30,
		},
		{
			name:                  "pause at end of period",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			amountCents:           3000,
			pauseDate:             periodEnd,
			expectedCreditAmount:  0, // No credit
			expectedChargeAmount:  0,
			expectedNetAmount:     0,
			expectedDaysTotal:     30,
			expectedDaysUsed:      30,
			expectedDaysRemaining: 0,
		},
		{
			name:                  "pause zero amount subscription",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			amountCents:           0,
			pauseDate:             time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			expectedCreditAmount:  0,
			expectedChargeAmount:  0,
			expectedNetAmount:     0,
			expectedDaysTotal:     30,
			expectedDaysUsed:      15,
			expectedDaysRemaining: 15,
		},
		{
			name:                  "pause after period end",
			currentPeriodStart:    periodStart,
			currentPeriodEnd:      periodEnd,
			amountCents:           3000,
			pauseDate:             time.Date(2024, 2, 5, 0, 0, 0, 0, time.UTC),
			expectedCreditAmount:  0,
			expectedChargeAmount:  0,
			expectedNetAmount:     0,
			expectedDaysTotal:     30,
			expectedDaysUsed:      30,
			expectedDaysRemaining: 0,
		},
		{
			name:                  "pause weekly subscription",
			currentPeriodStart:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			currentPeriodEnd:      time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC), // 7 days
			amountCents:           700,                                         // $7.00
			pauseDate:             time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), // Day 3
			expectedCreditAmount:  400,                                         // 4 days * ($7/7 days)
			expectedChargeAmount:  0,
			expectedNetAmount:     -400,
			expectedDaysTotal:     7,
			expectedDaysUsed:      3,
			expectedDaysRemaining: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.CalculatePauseCredit(
				tt.currentPeriodStart,
				tt.currentPeriodEnd,
				tt.amountCents,
				tt.pauseDate,
			)

			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedCreditAmount, result.CreditAmount)
			assert.Equal(t, tt.expectedChargeAmount, result.ChargeAmount)
			assert.Equal(t, tt.expectedNetAmount, result.NetAmount)
			assert.Equal(t, tt.expectedDaysTotal, result.DaysTotal)
			assert.Equal(t, tt.expectedDaysUsed, result.DaysUsed)
			assert.Equal(t, tt.expectedDaysRemaining, result.DaysRemaining)

			// Verify calculation breakdown
			assert.NotNil(t, result.Calculation)
			assert.Contains(t, result.Calculation, "total_days")
			assert.Contains(t, result.Calculation, "used_days")
			assert.Contains(t, result.Calculation, "remaining_days")
			assert.Contains(t, result.Calculation, "daily_rate")
			assert.Contains(t, result.Calculation, "pause_credit")
			assert.Contains(t, result.Calculation, "pause_date")

			// Verify daily rates
			if tt.expectedDaysTotal > 0 {
				expectedRate := float64(tt.amountCents) / float64(tt.expectedDaysTotal)
				assert.InDelta(t, expectedRate, result.OldDailyRate, 0.01)
			}
			assert.Equal(t, float64(0), result.NewDailyRate)
		})
	}
}

func TestProrationCalculator_DaysBetween(t *testing.T) {
	calculator := services.NewProrationCalculator()

	tests := []struct {
		name         string
		start        time.Time
		end          time.Time
		expectedDays int
	}{
		{
			name:         "same day",
			start:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC),
			expectedDays: 0,
		},
		{
			name:         "one day apart",
			start:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			expectedDays: 1,
		},
		{
			name:         "one week apart",
			start:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			expectedDays: 7,
		},
		{
			name:         "one month apart (January)",
			start:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			expectedDays: 31,
		},
		{
			name:         "leap year February",
			start:        time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedDays: 29, // 2024 is a leap year
		},
		{
			name:         "across year boundary",
			start:        time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			expectedDays: 2,
		},
		{
			name:         "with different time zones",
			start:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 2, 0, 0, 0, 0, time.FixedZone("EST", -5*3600)),
			expectedDays: 1,
		},
		{
			name:         "end before start",
			start:        time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedDays: -1,
		},
		{
			name:         "different times same day becomes 1 day",
			start:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			expectedDays: 1, // Different times on same day should be 1 day
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days := calculator.DaysBetween(tt.start, tt.end)
			assert.Equal(t, tt.expectedDays, days)
		})
	}
}

func TestProrationCalculator_AddBillingPeriod(t *testing.T) {
	calculator := services.NewProrationCalculator()

	baseDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name          string
		start         time.Time
		intervalType  string
		intervalCount int
		expected      time.Time
	}{
		{
			name:          "add daily interval",
			start:         baseDate,
			intervalType:  "daily",
			intervalCount: 5,
			expected:      time.Date(2024, 1, 20, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "add weekly interval",
			start:         baseDate,
			intervalType:  "weekly",
			intervalCount: 2,
			expected:      time.Date(2024, 1, 29, 10, 30, 0, 0, time.UTC), // 14 days later
		},
		{
			name:          "add monthly interval",
			start:         baseDate,
			intervalType:  "monthly",
			intervalCount: 1,
			expected:      time.Date(2024, 2, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "add multiple months",
			start:         baseDate,
			intervalType:  "monthly",
			intervalCount: 3,
			expected:      time.Date(2024, 4, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "add yearly interval",
			start:         baseDate,
			intervalType:  "yearly",
			intervalCount: 1,
			expected:      time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "add multiple years",
			start:         baseDate,
			intervalType:  "yearly",
			intervalCount: 2,
			expected:      time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "unknown interval defaults to monthly",
			start:         baseDate,
			intervalType:  "unknown",
			intervalCount: 1,
			expected:      time.Date(2024, 2, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "empty interval type defaults to monthly",
			start:         baseDate,
			intervalType:  "",
			intervalCount: 1,
			expected:      time.Date(2024, 2, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "zero interval count",
			start:         baseDate,
			intervalType:  "monthly",
			intervalCount: 0,
			expected:      baseDate, // No change
		},
		{
			name:          "negative interval count",
			start:         baseDate,
			intervalType:  "monthly",
			intervalCount: -1,
			expected:      time.Date(2023, 12, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:          "month end handling",
			start:         time.Date(2024, 1, 31, 10, 30, 0, 0, time.UTC),
			intervalType:  "monthly",
			intervalCount: 1,
			expected:      time.Date(2024, 2, 29, 10, 30, 0, 0, time.UTC), // Leap year Feb 29
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.AddBillingPeriod(tt.start, tt.intervalType, tt.intervalCount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProrationCalculator_CalculateTrialEndDate(t *testing.T) {
	calculator := services.NewProrationCalculator()

	tests := []struct {
		name      string
		start     time.Time
		trialDays int
		expected  time.Time
	}{
		{
			name:      "7 day trial",
			start:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			trialDays: 7,
			expected:  time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "30 day trial",
			start:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			trialDays: 30,
			expected:  time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "zero day trial",
			start:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			trialDays: 0,
			expected:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "negative trial days",
			start:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			trialDays: -5,
			expected:  time.Date(2023, 12, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "trial across month boundary",
			start:     time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC),
			trialDays: 10,
			expected:  time.Date(2024, 2, 4, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.CalculateTrialEndDate(tt.start, tt.trialDays)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProrationCalculator_IsInTrial(t *testing.T) {
	calculator := services.NewProrationCalculator()

	now := time.Now()
	futureDate := now.Add(24 * time.Hour)
	pastDate := now.Add(-24 * time.Hour)

	tests := []struct {
		name     string
		trialEnd *time.Time
		expected bool
	}{
		{
			name:     "trial in future",
			trialEnd: &futureDate,
			expected: true,
		},
		{
			name:     "trial in past",
			trialEnd: &pastDate,
			expected: false,
		},
		{
			name:     "nil trial end",
			trialEnd: nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.IsInTrial(tt.trialEnd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProrationCalculator_GetDailyRate(t *testing.T) {
	calculator := services.NewProrationCalculator()

	tests := []struct {
		name        string
		amountCents int64
		periodStart time.Time
		periodEnd   time.Time
		expected    float64
	}{
		{
			name:        "monthly rate",
			amountCents: 3000, // $30.00
			periodStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), // 30 days
			expected:    100.0,                                        // $1.00 per day
		},
		{
			name:        "weekly rate",
			amountCents: 700, // $7.00
			periodStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC), // 7 days
			expected:    100.0,                                       // $1.00 per day
		},
		{
			name:        "zero amount",
			amountCents: 0,
			periodStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			expected:    0.0,
		},
		{
			name:        "zero days period",
			amountCents: 1000,
			periodStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected:    0.0,
		},
		{
			name:        "single day period",
			amountCents: 100,
			periodStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			periodEnd:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), // 1 day
			expected:    100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.GetDailyRate(tt.amountCents, tt.periodStart, tt.periodEnd)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestProrationCalculator_FormatProrationExplanation(t *testing.T) {
	calculator := services.NewProrationCalculator()

	tests := []struct {
		name     string
		result   *business.ProrationResult
		expected string
	}{
		{
			name: "positive net amount (charge)",
			result: &business.ProrationResult{
				NetAmount: 500,
			},
			expected: "You'll be charged for the upgraded service for the remainder of your billing period.",
		},
		{
			name: "negative net amount (credit)",
			result: &business.ProrationResult{
				NetAmount: -300,
			},
			expected: "You'll receive a credit for the unused portion of your current billing period.",
		},
		{
			name: "zero net amount",
			result: &business.ProrationResult{
				NetAmount: 0,
			},
			expected: "No additional charge for this change.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculator.FormatProrationExplanation(tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProrationCalculator_EdgeCases(t *testing.T) {
	calculator := services.NewProrationCalculator()

	t.Run("nil proration result explanation", func(t *testing.T) {
		// Should not panic with nil result
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("FormatProrationExplanation panicked with nil input: %v", r)
			}
		}()
		calculator.FormatProrationExplanation(nil)
	})

	t.Run("very large amounts", func(t *testing.T) {
		// Test with maximum int64 values
		periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		periodEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
		changeDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

		result := calculator.CalculateUpgradeProration(
			periodStart,
			periodEnd,
			9223372036854775000, // Large old amount
			9223372036854775800, // Slightly larger new amount
			changeDate,
		)

		assert.NotNil(t, result)
		// Should handle large numbers without overflow
		assert.Greater(t, result.NetAmount, int64(0))
	})

	t.Run("extreme date ranges", func(t *testing.T) {
		// Test with very old dates
		oldStart := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
		oldEnd := time.Date(1900, 12, 31, 0, 0, 0, 0, time.UTC)

		days := calculator.DaysBetween(oldStart, oldEnd)
		assert.Greater(t, days, 0)

		// Test with far future dates
		futureStart := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
		futureEnd := time.Date(2100, 1, 31, 0, 0, 0, 0, time.UTC)

		futureDays := calculator.DaysBetween(futureStart, futureEnd)
		assert.Equal(t, 30, futureDays)
	})

	t.Run("daylight saving time transitions", func(t *testing.T) {
		// Test around DST transitions
		loc, _ := time.LoadLocation("America/New_York")

		// Spring forward (lose an hour)
		start := time.Date(2024, 3, 10, 0, 0, 0, 0, loc)
		end := time.Date(2024, 3, 11, 0, 0, 0, 0, loc)

		days := calculator.DaysBetween(start, end)
		assert.Equal(t, 1, days) // Should still be 1 day regardless of DST
	})

	t.Run("leap year calculations", func(t *testing.T) {
		// Test February in leap year vs non-leap year
		leapYear := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
		leapYearEnd := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		leapDays := calculator.DaysBetween(leapYear, leapYearEnd)
		assert.Equal(t, 29, leapDays)

		nonLeapYear := time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)
		nonLeapYearEnd := time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)
		nonLeapDays := calculator.DaysBetween(nonLeapYear, nonLeapYearEnd)
		assert.Equal(t, 28, nonLeapDays)
	})

	t.Run("concurrent access safety", func(t *testing.T) {
		// ProrationCalculator should be safe for concurrent use
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				periodStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				periodEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
				changeDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

				result := calculator.CalculateUpgradeProration(
					periodStart, periodEnd, 1000, 2000, changeDate,
				)
				assert.NotNil(t, result)
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
