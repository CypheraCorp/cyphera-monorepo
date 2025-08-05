package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// GetDashboardSummary retrieves the main dashboard metrics
func (s *AnalyticsService) GetDashboardSummary(ctx context.Context, workspaceID uuid.UUID, currency string) (*business.DashboardSummary, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
		} else {
			currency = defaultCurrency.Code
		}
	}

	// Get latest metrics for MRR/ARR from monthly metrics
	monthlyMetrics, err := s.queries.GetLatestDashboardMetrics(ctx, db.GetLatestDashboardMetricsParams{
		WorkspaceID:  workspaceID,
		MetricType:   "monthly",
		FiatCurrency: currency,
	})
	
	var mrrCents, arrCents int64
	var lastUpdated time.Time
	
	if err == nil {
		mrrCents = monthlyMetrics.MrrCents.Int64
		arrCents = monthlyMetrics.ArrCents.Int64
		lastUpdated = monthlyMetrics.UpdatedAt.Time
	} else if err != pgx.ErrNoRows {
		// If it's not a "no rows" error, return it
		return nil, err
	}
	
	// Get all-time totals directly from source tables
	var totalRevenueCents int64
	var totalCustomers int32
	var activeSubscriptions int32
	
	// Query for all-time total revenue
	err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount_in_cents), 0) 
		FROM payments 
		WHERE workspace_id = $1 
			AND currency = $2 
			AND status = 'completed'
	`, workspaceID, currency).Scan(&totalRevenueCents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %w", err)
	}
	
	// Query for total unique customers
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT customer_id) 
		FROM payments 
		WHERE workspace_id = $1 
			AND status = 'completed'
	`, workspaceID).Scan(&totalCustomers)
	if err != nil {
		return nil, fmt.Errorf("failed to get total customers: %w", err)
	}
	
	// Query for active subscriptions (currently active, not completed)
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT s.id) 
		FROM subscriptions s
		JOIN products p ON s.product_id = p.id
		WHERE p.workspace_id = $1
			AND s.status = 'active'
			AND p.deleted_at IS NULL
	`, workspaceID).Scan(&activeSubscriptions)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get active subscriptions: %w", err)
	}
	
	// If no monthly metrics exist yet, use current time for last updated
	if lastUpdated.IsZero() {
		lastUpdated = time.Now()
	}

	summary := &business.DashboardSummary{
		MRR: business.MoneyAmount{
			AmountCents: mrrCents,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(mrrCents, currency),
		},
		ARR: business.MoneyAmount{
			AmountCents: arrCents,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(arrCents, currency),
		},
		TotalRevenue: business.MoneyAmount{
			AmountCents: totalRevenueCents,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(totalRevenueCents, currency),
		},
		ActiveSubscriptions: activeSubscriptions,
		TotalCustomers:      totalCustomers,
		ChurnRate:           0, // Will be calculated from metrics when available
		GrowthRate:          0, // Will be calculated from metrics when available
		PaymentSuccessRate:  0, // Will be calculated from metrics when available
		LastUpdated:         lastUpdated,
	}

	// Get revenue growth
	growth, err := s.queries.GetRevenueGrowth(ctx, db.GetRevenueGrowthParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: time.Now(), Valid: true},
		MetricType:   "monthly",
		FiatCurrency: currency,
	})
	if err == nil {
		summary.RevenueGrowth = &business.RevenueGrowth{
			CurrentPeriod: growth.CurrentPeriod.Time,
			CurrentRevenue: business.MoneyAmount{
				AmountCents: growth.CurrentRevenue.Int64,
				Currency:    currency,
				Formatted:   helpers.FormatMoney(growth.CurrentRevenue.Int64, currency),
			},
			PreviousRevenue: business.MoneyAmount{
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
func (s *AnalyticsService) GetRevenueChart(ctx context.Context, workspaceID uuid.UUID, period string, days int, currency string) (*business.ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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
	chartData := make([]business.ChartDataPoint, len(metrics))
	for i, metric := range metrics {
		chartData[i] = business.ChartDataPoint{
			Date:  metric.MetricDate.Time.Format("2006-01-02"),
			Value: float64(metric.TotalRevenueCents.Int64) / 100.0, // Convert cents to dollars
			Label: helpers.FormatMoney(metric.TotalRevenueCents.Int64, currency),
		}
	}

	return &business.ChartData{
		ChartType: "line",
		Title:     "Revenue Over Time",
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetCustomerChart returns customer growth data for charting
func (s *AnalyticsService) GetCustomerChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, days int, currency string) (*business.ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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
	chartData := make([]business.ChartDataPoint, len(trends))
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

		chartData[i] = business.ChartDataPoint{
			Date:  trend.MetricDate.Time.Format("2006-01-02"),
			Value: value,
		}
	}

	return &business.ChartData{
		ChartType: "line",
		Title:     title,
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetPaymentMetrics returns payment success and failure metrics
func (s *AnalyticsService) GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, days int, currency string) (*business.PaymentMetrics, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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

	return &business.PaymentMetrics{
		TotalSuccessful: summary.TotalSuccessfulPayments,
		TotalFailed:     summary.TotalFailedPayments,
		TotalVolume: business.MoneyAmount{
			AmountCents: summary.TotalVolumeCents,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(summary.TotalVolumeCents, currency),
		},
		SuccessRate: summary.AvgSuccessRate,
		GasMetrics: business.GasMetrics{
			TotalGasFees: business.MoneyAmount{
				AmountCents: summary.TotalGasFees,
				Currency:    currency,
				Formatted:   helpers.FormatMoney(summary.TotalGasFees, currency),
			},
			SponsoredGasFees: business.MoneyAmount{
				AmountCents: summary.TotalSponsoredGas,
				Currency:    currency,
				Formatted:   helpers.FormatMoney(summary.TotalSponsoredGas, currency),
			},
		},
		Period: days,
	}, nil
}

// GetNetworkBreakdown returns payment volume breakdown by network
func (s *AnalyticsService) GetNetworkBreakdown(ctx context.Context, workspaceID uuid.UUID, date time.Time, currency string) (*business.NetworkBreakdown, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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
	var networkData map[string]business.NetworkMetrics
	if err := json.Unmarshal(metrics.NetworkMetrics, &networkData); err != nil {
		return nil, err
	}

	// Parse token metrics JSON
	var tokenData map[string]business.TokenMetrics
	if err := json.Unmarshal(metrics.TokenMetrics, &tokenData); err != nil {
		return nil, err
	}

	return &business.NetworkBreakdown{
		Date:     date.Format("2006-01-02"),
		Networks: networkData,
		Tokens:   tokenData,
	}, nil
}

// GetSubscriptionChart returns subscription metrics for charting
func (s *AnalyticsService) GetSubscriptionChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, days int, currency string) (*business.ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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
	chartData := make([]business.ChartDataPoint, len(metrics))
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

		chartData[i] = business.ChartDataPoint{
			Date:  m.MetricDate.Time.Format("2006-01-02"),
			Value: value,
		}
	}

	return &business.ChartData{
		ChartType: "line",
		Title:     title,
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetMRRChart returns MRR/ARR growth over time
func (s *AnalyticsService) GetMRRChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, months int, currency string) (*business.ChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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
	chartData := make([]business.ChartDataPoint, len(metrics))
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

		chartData[i] = business.ChartDataPoint{
			Date:  m.MetricDate.Time.Format("2006-01-02"),
			Value: float64(cents) / 100.0, // Convert cents to dollars
			Label: helpers.FormatMoney(cents, currency),
		}
	}

	return &business.ChartData{
		ChartType: "line",
		Title:     title,
		Data:      chartData,
		Period:    period,
	}, nil
}

// GetGasFeePieChart returns gas fee breakdown as a pie chart
func (s *AnalyticsService) GetGasFeePieChart(ctx context.Context, workspaceID uuid.UUID, days int, currency string) (*business.PieChartData, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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

	return &business.PieChartData{
		ChartType: "pie",
		Title:     "Gas Fee Distribution",
		Data: []business.PieChartDataPoint{
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
		Total: business.MoneyAmount{
			AmountCents: summary.TotalGasFees,
			Currency:    currency,
			Formatted:   helpers.FormatMoney(summary.TotalGasFees, currency),
		},
	}, nil
}

// GetHourlyMetrics returns hourly metrics for today
func (s *AnalyticsService) GetHourlyMetrics(ctx context.Context, workspaceID uuid.UUID, currency string) (*business.HourlyMetrics, error) {
	// Get workspace default currency if not provided
	if currency == "" {
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
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
	hourlyData := make([]business.HourlyDataPoint, 0, len(metrics))
	for _, m := range metrics {
		if m.MetricHour.Valid {
			hourlyData = append(hourlyData, business.HourlyDataPoint{
				Hour:     int(m.MetricHour.Int32),
				Revenue:  float64(m.TotalRevenueCents.Int64) / 100.0,
				Payments: int(m.SuccessfulPayments.Int32),
				NewUsers: int(m.NewCustomers.Int32),
			})
		}
	}

	return &business.HourlyMetrics{
		Date:       today.Format("2006-01-02"),
		HourlyData: hourlyData,
		Currency:   currency,
	}, nil
}

// TriggerMetricsRefresh triggers async metrics recalculation
func (s *AnalyticsService) TriggerMetricsRefresh(ctx context.Context, workspaceID uuid.UUID, date time.Time) error {
	// Get a connection from the pool
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Create a DashboardMetricsService instance
	metricsService := NewDashboardMetricsService(s.queries, conn.Conn())

	// Calculate all metrics for the workspace
	if err := metricsService.CalculateAllMetricsForWorkspace(ctx, workspaceID, date); err != nil {
		return fmt.Errorf("failed to calculate metrics: %w", err)
	}

	return nil
}
