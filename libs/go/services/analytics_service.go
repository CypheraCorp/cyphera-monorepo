package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AnalyticsService handles analytics business logic
type AnalyticsService struct {
	queries         db.Querier
	pool            *pgxpool.Pool
	currencyService *CurrencyService
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(queries db.Querier, pool *pgxpool.Pool) *AnalyticsService {
	return &AnalyticsService{
		queries:         queries,
		pool:            pool,
		currencyService: NewCurrencyService(queries),
	}
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

// ChartDataPoint represents a single data point for charts
type ChartDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
	Label string  `json:"label,omitempty"`
}

// ChartData represents data for various chart types
type ChartData struct {
	ChartType string           `json:"chart_type"`
	Title     string           `json:"title"`
	Data      []ChartDataPoint `json:"data"`
	Period    string           `json:"period"`
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

// NetworkBreakdown represents network and token breakdown
type NetworkBreakdown struct {
	Date     string                    `json:"date"`
	Networks map[string]NetworkMetrics `json:"networks"`
	Tokens   map[string]TokenMetrics   `json:"tokens"`
}

// PieChartData represents data for pie charts
type PieChartData struct {
	ChartType string              `json:"chart_type"`
	Title     string              `json:"title"`
	Data      []PieChartDataPoint `json:"data"`
	Total     MoneyAmount         `json:"total"`
}

// PieChartDataPoint represents a single pie slice
type PieChartDataPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color string  `json:"color,omitempty"`
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

// GetDashboardSummary retrieves the main dashboard metrics
func (s *AnalyticsService) GetDashboardSummary(ctx context.Context, workspaceID uuid.UUID, currency string) (*DashboardSummary, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	// Get latest metrics
	latest, err := s.queries.GetLatestDashboardMetrics(ctx, db.GetLatestDashboardMetricsParams{
		WorkspaceID:  workspaceID,
		MetricType:   "daily",
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	summary := &DashboardSummary{
		MRR: MoneyAmount{
			AmountCents: latest.MrrCents.Int64,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(latest.MrrCents.Int64, currency),
		},
		ARR: MoneyAmount{
			AmountCents: latest.ArrCents.Int64,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(latest.ArrCents.Int64, currency),
		},
		TotalRevenue: MoneyAmount{
			AmountCents: latest.TotalRevenueCents.Int64,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(latest.TotalRevenueCents.Int64, currency),
		},
		ActiveSubscriptions: latest.ActiveSubscriptions.Int32,
		TotalCustomers:      latest.TotalCustomers.Int32,
		ChurnRate:           helpers.GetNumericFloat(latest.ChurnRate),
		GrowthRate:          helpers.GetNumericFloat(latest.GrowthRate),
		PaymentSuccessRate:  helpers.GetNumericFloat(latest.PaymentSuccessRate),
		LastUpdated:         latest.UpdatedAt.Time,
	}

	// Get revenue growth
	growth, err := s.queries.GetRevenueGrowth(ctx, db.GetRevenueGrowthParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: time.Now(), Valid: true},
		MetricType:   "monthly",
		FiatCurrency: currency,
	})
	if err == nil {
		summary.RevenueGrowth = &RevenueGrowth{
			CurrentPeriod: growth.CurrentPeriod.Time,
			CurrentRevenue: MoneyAmount{
				AmountCents: growth.CurrentRevenue.Int64,
				Currency:    currency,
				Formatted:   helpers.FormatMoney(growth.CurrentRevenue.Int64, currency),
			},
			PreviousRevenue: MoneyAmount{
				AmountCents: growth.PreviousRevenue.Int64,
				Currency:    currency,
				Formatted:   helpers.FormatMoney(growth.PreviousRevenue.Int64, currency),
			},
			GrowthPercentage: float64(growth.GrowthPercentage),
		}
	}

	return summary, nil
}

// GetRevenueChart returns revenue data for charting
func (s *AnalyticsService) GetRevenueChart(ctx context.Context, workspaceID uuid.UUID, period string, days int, currency string) (*ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get metrics for the period
	metrics, err := s.queries.GetDashboardMetricsByDateRange(ctx, db.GetDashboardMetricsByDateRangeParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   period,
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	// Convert to chart data
	chartData := make([]ChartDataPoint, len(metrics))
	for i, metric := range metrics {
		chartData[i] = ChartDataPoint{
			Date:  metric.MetricDate.Time.Format("2006-01-02"),
			Value: float64(metric.TotalRevenueCents.Int64) / 100.0, // Convert cents to dollars
			Label: helpers.FormatMoney(metric.TotalRevenueCents.Int64, currency),
		}
	}

	return &ChartData{
		ChartType: "line",
		Title:     "Revenue Over Time",
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetCustomerChart returns customer growth data for charting
func (s *AnalyticsService) GetCustomerChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, days int, currency string) (*ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get customer trends
	trends, err := s.queries.GetCustomerMetricsTrend(ctx, db.GetCustomerMetricsTrendParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   period,
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	// Convert to chart data based on selected metric
	chartData := make([]ChartDataPoint, len(trends))
	title := "Customer Metrics"

	for i, trend := range trends {
		var value float64
		switch metric {
		case "new":
			value = float64(trend.NewCustomers.Int32)
			title = "New Customers"
		case "churned":
			value = float64(trend.ChurnedCustomers.Int32)
			title = "Churned Customers"
		case "growth_rate":
			value = helpers.GetNumericFloat(trend.GrowthRate) * 100 // Convert to percentage
			title = "Growth Rate (%)"
		default: // total
			value = float64(trend.TotalCustomers.Int32)
			title = "Total Customers"
		}

		chartData[i] = ChartDataPoint{
			Date:  trend.MetricDate.Time.Format("2006-01-02"),
			Value: value,
		}
	}

	return &ChartData{
		ChartType: "line",
		Title:     title,
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetPaymentMetrics returns payment success and failure metrics
func (s *AnalyticsService) GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, days int, currency string) (*PaymentMetrics, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get payment summary
	summary, err := s.queries.GetPaymentMetricsSummary(ctx, db.GetPaymentMetricsSummaryParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   "daily",
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	return &PaymentMetrics{
		TotalSuccessful: summary.TotalSuccessfulPayments,
		TotalFailed:     summary.TotalFailedPayments,
		TotalVolume: MoneyAmount{
			AmountCents: summary.TotalVolumeCents,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(summary.TotalVolumeCents, currency),
		},
		SuccessRate: summary.AvgSuccessRate,
		GasMetrics: GasMetrics{
			TotalGasFees: MoneyAmount{
				AmountCents: summary.TotalGasFees,
				Currency:    currency,
				Formatted:   helpers.FormatMoney(summary.TotalGasFees, currency),
			},
			SponsoredGasFees: MoneyAmount{
				AmountCents: summary.TotalSponsoredGas,
				Currency:    currency,
				Formatted:   helpers.FormatMoney(summary.TotalSponsoredGas, currency),
			},
		},
		Period: days,
	}, nil
}

// GetNetworkBreakdown returns payment volume breakdown by network
func (s *AnalyticsService) GetNetworkBreakdown(ctx context.Context, workspaceID uuid.UUID, date time.Time, currency string) (*NetworkBreakdown, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	// Get network metrics
	metrics, err := s.queries.GetNetworkMetrics(ctx, db.GetNetworkMetricsParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: date, Valid: true},
		MetricType:   "daily",
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	// Parse network metrics JSON
	var networkData map[string]NetworkMetrics
	if err := json.Unmarshal(metrics.NetworkMetrics, &networkData); err != nil {
		return nil, err
	}

	// Parse token metrics JSON
	var tokenData map[string]TokenMetrics
	if err := json.Unmarshal(metrics.TokenMetrics, &tokenData); err != nil {
		return nil, err
	}

	return &NetworkBreakdown{
		Date:     date.Format("2006-01-02"),
		Networks: networkData,
		Tokens:   tokenData,
	}, nil
}

// GetSubscriptionChart returns subscription metrics for charting
func (s *AnalyticsService) GetSubscriptionChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, days int, currency string) (*ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get metrics for the period
	metrics, err := s.queries.GetDashboardMetricsByDateRange(ctx, db.GetDashboardMetricsByDateRangeParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   period,
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	// Convert to chart data based on selected metric
	chartData := make([]ChartDataPoint, len(metrics))
	title := "Subscription Metrics"

	for i, m := range metrics {
		var value float64
		switch metric {
		case "new":
			value = float64(m.NewSubscriptions.Int32)
			title = "New Subscriptions"
		case "cancelled":
			value = float64(m.CancelledSubscriptions.Int32)
			title = "Cancelled Subscriptions"
		case "churn_rate":
			value = helpers.GetNumericFloat(m.ChurnRate) * 100 // Convert to percentage
			title = "Churn Rate (%)"
		default: // active
			value = float64(m.ActiveSubscriptions.Int32)
			title = "Active Subscriptions"
		}

		chartData[i] = ChartDataPoint{
			Date:  m.MetricDate.Time.Format("2006-01-02"),
			Value: value,
		}
	}

	return &ChartData{
		ChartType: "line",
		Title:     title,
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetMRRChart returns MRR/ARR growth over time
func (s *AnalyticsService) GetMRRChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, months int, currency string) (*ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, -months, 0)

	// Get metrics for the period
	metrics, err := s.queries.GetDashboardMetricsByDateRange(ctx, db.GetDashboardMetricsByDateRangeParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   period,
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	// Convert to chart data
	chartData := make([]ChartDataPoint, len(metrics))
	title := "Monthly Recurring Revenue"
	if metric == "arr" {
		title = "Annual Recurring Revenue"
	}

	for i, m := range metrics {
		var cents int64
		if metric == "arr" {
			cents = m.ArrCents.Int64
		} else {
			cents = m.MrrCents.Int64
		}

		chartData[i] = ChartDataPoint{
			Date:  m.MetricDate.Time.Format("2006-01-02"),
			Value: float64(cents) / 100.0, // Convert cents to dollars
			Label: helpers.FormatMoney(cents, currency),
		}
	}

	return &ChartData{
		ChartType: "line",
		Title:     title,
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetGasFeePieChart returns gas fee breakdown as a pie chart
func (s *AnalyticsService) GetGasFeePieChart(ctx context.Context, workspaceID uuid.UUID, days int, currency string) (*PieChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get payment summary
	summary, err := s.queries.GetPaymentMetricsSummary(ctx, db.GetPaymentMetricsSummaryParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   "daily",
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	sponsoredCents := summary.TotalSponsoredGas
	customerCents := summary.TotalGasFees - sponsoredCents

	return &PieChartData{
		ChartType: "pie",
		Title:     "Gas Fee Distribution",
		Data: []PieChartDataPoint{
			{
				Label: "Merchant Sponsored",
				Value: float64(sponsoredCents) / 100.0,
				Color: "#10B981", // Green
			},
			{
				Label: "Customer Paid",
				Value: float64(customerCents) / 100.0,
				Color: "#3B82F6", // Blue
			},
		},
		Total: MoneyAmount{
			AmountCents: summary.TotalGasFees,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(summary.TotalGasFees, currency),
		},
	}, nil
}

// GetHourlyMetrics returns hourly metrics for today
func (s *AnalyticsService) GetHourlyMetrics(ctx context.Context, workspaceID uuid.UUID, currency string) (*HourlyMetrics, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = "USD"
		} else {
			currency = defaultCurrency.Code
		}
	}

	today := time.Now()

	// Get hourly metrics for today
	metrics, err := s.queries.GetHourlyMetrics(ctx, db.GetHourlyMetricsParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: today, Valid: true},
		FiatCurrency: currency,
	})
	if err != nil {
		return nil, err
	}

	// Convert to response format
	hourlyData := make([]HourlyDataPoint, 0, len(metrics))
	for _, m := range metrics {
		if m.MetricHour.Valid {
			hourlyData = append(hourlyData, HourlyDataPoint{
				Hour:     int(m.MetricHour.Int32),
				Revenue:  float64(m.TotalRevenueCents.Int64) / 100.0,
				Payments: int(m.SuccessfulPayments.Int32),
				NewUsers: int(m.NewCustomers.Int32),
			})
		}
	}

	return &HourlyMetrics{
		Date:       today.Format("2006-01-02"),
		HourlyData: hourlyData,
		Currency:   currency,
	}, nil
}

// TriggerMetricsRefresh triggers async metrics recalculation
func (s *AnalyticsService) TriggerMetricsRefresh(ctx context.Context, workspaceID uuid.UUID, date time.Time) error {
	// TODO: Implement metrics refresh using background job system
	// This should be handled by a separate background worker service that:
	// 1. Gets a connection from the database pool
	// 2. Creates a DashboardMetricsService instance
	// 3. Calls CalculateAllMetricsForWorkspace
	// For now, return nil to avoid circular dependencies
	return nil
}
