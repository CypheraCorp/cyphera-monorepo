package business

import "time"

// ProrationResult contains the result of a proration calculation
type ProrationResult struct {
	CreditAmount  int64                  `json:"credit_amount"`
	ChargeAmount  int64                  `json:"charge_amount"`
	NetAmount     int64                  `json:"net_amount"`
	DaysTotal     int                    `json:"days_total"`
	DaysUsed      int                    `json:"days_used"`
	DaysRemaining int                    `json:"days_remaining"`
	OldDailyRate  float64                `json:"old_daily_rate"`
	NewDailyRate  float64                `json:"new_daily_rate"`
	Calculation   map[string]interface{} `json:"calculation"`
}

// ScheduleChangeResult contains information about a scheduled change
type ScheduleChangeResult struct {
	ScheduledFor time.Time `json:"scheduled_for"`
	ChangeType   string    `json:"change_type"`
	NoProration  bool      `json:"no_proration"`
	Message      string    `json:"message"`
}
