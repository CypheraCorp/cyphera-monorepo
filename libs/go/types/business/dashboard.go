package business

import (
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// MetricType represents the type of metric (hourly, daily, etc.)
type MetricType string

const (
	MetricTypeHourly  MetricType = "hourly"
	MetricTypeDaily   MetricType = "daily"
	MetricTypeWeekly  MetricType = "weekly"
	MetricTypeMonthly MetricType = "monthly"
	MetricTypeYearly  MetricType = "yearly"
)

// CalculateMetricsOptions provides options for metric calculation
type CalculateMetricsOptions struct {
	WorkspaceID uuid.UUID
	Date        time.Time
	MetricType  MetricType
	Hour        *int // For hourly metrics
	Currency    string
}

// DashboardData represents comprehensive dashboard data
type DashboardData struct {
	CurrentMetrics *db.DashboardMetric
	RevenueGrowth  *db.GetRevenueGrowthRow
	CustomerTrends []db.GetCustomerMetricsTrendRow
	PaymentSummary *db.GetPaymentMetricsSummaryRow
}

// DashboardSummary represents the main dashboard overview
type DashboardSummary struct {
	MRR                 MoneyAmount    `json:"mrr"`
	ARR                 MoneyAmount    `json:"arr"`
	TotalRevenue        MoneyAmount    `json:"total_revenue"`
	ActiveSubscriptions int32          `json:"active_subscriptions"`
	TotalCustomers      int32          `json:"total_customers"`
	ChurnRate           float64        `json:"churn_rate"`
	GrowthRate          float64        `json:"growth_rate"`
	PaymentSuccessRate  float64        `json:"payment_success_rate"`
	LastUpdated         time.Time      `json:"last_updated"`
	RevenueGrowth       *RevenueGrowth `json:"revenue_growth,omitempty"`
}

// MoneyAmount represents a monetary value with currency
type MoneyAmount struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Formatted   string `json:"formatted"`
}

// RevenueGrowth represents revenue growth data
type RevenueGrowth struct {
	CurrentPeriod    time.Time   `json:"current_period"`
	CurrentRevenue   MoneyAmount `json:"current_revenue"`
	PreviousRevenue  MoneyAmount `json:"previous_revenue"`
	GrowthPercentage float64     `json:"growth_percentage"`
}

// PaymentMetrics represents payment metrics data
type PaymentMetrics struct {
	TotalSuccessful int64       `json:"total_successful"`
	TotalFailed     int64       `json:"total_failed"`
	TotalVolume     MoneyAmount `json:"total_volume"`
	SuccessRate     float64     `json:"success_rate"`
	GasMetrics      GasMetrics  `json:"gas_metrics"`
	Period          int         `json:"period_days"`
}

// GasMetrics represents gas fee metrics
type GasMetrics struct {
	TotalGasFees     MoneyAmount `json:"total_gas_fees"`
	SponsoredGasFees MoneyAmount `json:"sponsored_gas_fees"`
}

// NetworkMetrics represents payment metrics for a specific network
type NetworkMetrics struct {
	Payments    int   `json:"payments"`
	VolumeCents int64 `json:"volume_cents"`
	GasFeeCents int64 `json:"gas_fee_cents"`
}

// TokenMetrics represents payment metrics for a specific token
type TokenMetrics struct {
	Payments      int   `json:"payments"`
	VolumeCents   int64 `json:"volume_cents"`
	AvgPriceCents int64 `json:"avg_price_cents"`
}

// HourlyMetrics represents hourly breakdown
type HourlyMetrics struct {
	Date       string            `json:"date"`
	HourlyData []HourlyDataPoint `json:"hourly_data"`
	Currency   string            `json:"currency"`
}

// HourlyDataPoint represents metrics for a specific hour
type HourlyDataPoint struct {
	Hour     int     `json:"hour"`
	Revenue  float64 `json:"revenue"`
	Payments int     `json:"payments"`
	NewUsers int     `json:"new_users"`
}

// NetworkBreakdown represents network and token breakdown
type NetworkBreakdown struct {
	Date     string                    `json:"date"`
	Networks map[string]NetworkMetrics `json:"networks"`
	Tokens   map[string]TokenMetrics   `json:"tokens"`
}
