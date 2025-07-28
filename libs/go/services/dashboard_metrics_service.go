package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// Internal types for dashboard metrics calculation
type networkMetrics struct {
	Payments    int   `json:"payments"`
	VolumeCents int64 `json:"volume_cents"`
	GasFeeCents int64 `json:"gas_fee_cents"`
}

type tokenMetrics struct {
	Payments      int   `json:"payments"`
	VolumeCents   int64 `json:"volume_cents"`
	AvgPriceCents int64 `json:"avg_price_cents"`
}

// DashboardMetricsService handles calculation and storage of dashboard metrics
type DashboardMetricsService struct {
	queries         db.Querier
	pool            *pgx.Conn
	logger          *zap.Logger
	currencyService *CurrencyService
}

// NewDashboardMetricsService creates a new dashboard metrics service
func NewDashboardMetricsService(queries db.Querier, pool *pgx.Conn) *DashboardMetricsService {
	return &DashboardMetricsService{
		queries:         queries,
		pool:            pool,
		logger:          logger.Log,
		currencyService: NewCurrencyService(queries),
	}
}

// MetricType represents different aggregation periods
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

// CalculateAndStoreMetrics calculates and stores metrics for a given period
func (s *DashboardMetricsService) CalculateAndStoreMetrics(ctx context.Context, opts CalculateMetricsOptions) (*db.DashboardMetric, error) {
	s.logger.Info("Calculating dashboard metrics",
		zap.String("workspace_id", opts.WorkspaceID.String()),
		zap.Time("date", opts.Date),
		zap.String("metric_type", string(opts.MetricType)),
		zap.String("currency", opts.Currency),
	)

	// Calculate date range based on metric type
	startDate, endDate := s.getDateRange(opts.Date, opts.MetricType, opts.Hour)

	// Initialize metrics
	metrics := &db.CreateDashboardMetricParams{
		WorkspaceID:  opts.WorkspaceID,
		MetricDate:   pgtype.Date{Time: opts.Date, Valid: true},
		MetricType:   string(opts.MetricType),
		FiatCurrency: opts.Currency,
	}

	// Set hour if applicable
	if opts.Hour != nil {
		metrics.MetricHour = pgtype.Int4{Int32: int32(*opts.Hour), Valid: true}
	} else {
		metrics.MetricHour = pgtype.Int4{Valid: false}
	}

	// Calculate revenue metrics
	if err := s.calculateRevenueMetrics(ctx, metrics, opts.WorkspaceID, startDate, endDate, opts.Currency); err != nil {
		return nil, fmt.Errorf("failed to calculate revenue metrics: %w", err)
	}

	// Calculate customer metrics
	if err := s.calculateCustomerMetrics(ctx, metrics, opts.WorkspaceID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to calculate customer metrics: %w", err)
	}

	// Calculate subscription metrics
	if err := s.calculateSubscriptionMetrics(ctx, metrics, opts.WorkspaceID, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to calculate subscription metrics: %w", err)
	}

	// Calculate payment metrics
	if err := s.calculatePaymentMetrics(ctx, metrics, opts.WorkspaceID, startDate, endDate, opts.Currency); err != nil {
		return nil, fmt.Errorf("failed to calculate payment metrics: %w", err)
	}

	// Calculate gas fee metrics
	if err := s.calculateGasFeeMetrics(ctx, metrics, opts.WorkspaceID, startDate, endDate, opts.Currency); err != nil {
		return nil, fmt.Errorf("failed to calculate gas fee metrics: %w", err)
	}

	// Calculate network and token metrics
	if err := s.calculateNetworkTokenMetrics(ctx, metrics, opts.WorkspaceID, startDate, endDate, opts.Currency); err != nil {
		return nil, fmt.Errorf("failed to calculate network/token metrics: %w", err)
	}

	// Calculate derived metrics (rates, averages)
	s.calculateDerivedMetrics(metrics)

	// Store metrics in database
	metric, err := s.queries.CreateDashboardMetric(ctx, *metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to store dashboard metrics: %w", err)
	}

	s.logger.Info("Successfully stored dashboard metrics",
		zap.String("metric_id", metric.ID.String()),
		zap.Int64("mrr_cents", metric.MrrCents.Int64),
		zap.Int64("total_revenue", metric.TotalRevenueCents.Int64),
		zap.Int32("active_subscriptions", metric.ActiveSubscriptions.Int32),
	)

	return &metric, nil
}

// calculateRevenueMetrics calculates MRR, ARR, and revenue breakdown
func (s *DashboardMetricsService) calculateRevenueMetrics(ctx context.Context, metrics *db.CreateDashboardMetricParams, workspaceID uuid.UUID, startDate, endDate time.Time, currency string) error {
	// For MRR/ARR, we need to look at active subscriptions as of the end date
	// This query should get all active recurring subscriptions and their monthly values
	query := `
		SELECT 
			SUM(CASE 
				WHEN p.interval_type = 'month' THEN p.unit_amount_in_pennies
				WHEN p.interval_type = 'year' THEN p.unit_amount_in_pennies / 12
				WHEN p.interval_type = 'week' THEN p.unit_amount_in_pennies * 4.33
				WHEN p.interval_type = 'daily' THEN p.unit_amount_in_pennies * 30
				ELSE 0
			END) as mrr_cents,
			COUNT(DISTINCT s.id) as subscription_count
		FROM subscriptions s
		JOIN prices p ON s.price_id = p.id
		JOIN products pr ON s.product_id = pr.id
		WHERE s.status = 'active'
			AND pr.workspace_id = $1
			AND p.currency = $2
			AND s.current_period_start <= $3
			AND (s.current_period_end IS NULL OR s.current_period_end > $3)
			AND s.deleted_at IS NULL
			AND p.deleted_at IS NULL
			AND pr.deleted_at IS NULL
	`

	var mrrCents pgtype.Int8
	var subCount pgtype.Int8
	err := s.pool.QueryRow(ctx, query, workspaceID, currency, endDate).Scan(&mrrCents, &subCount)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate MRR: %w", err)
	}

	if mrrCents.Valid {
		metrics.MrrCents = pgtype.Int8{Int64: mrrCents.Int64, Valid: true}
		metrics.ArrCents = pgtype.Int8{Int64: mrrCents.Int64 * 12, Valid: true}
	}

	// Calculate total revenue for the period
	revenueQuery := `
		SELECT 
			COALESCE(SUM(amount_in_cents), 0) as total_revenue,
			COALESCE(SUM(CASE WHEN p.created_at >= $3 THEN amount_in_cents ELSE 0 END), 0) as new_revenue
		FROM payments p
		WHERE p.workspace_id = $1
			AND p.currency = $2
			AND p.status = 'completed'
			AND p.created_at >= $3
			AND p.created_at < $4
	`

	var totalRevenue, newRevenue pgtype.Int8
	err = s.pool.QueryRow(ctx, revenueQuery, workspaceID, currency, startDate, endDate).Scan(&totalRevenue, &newRevenue)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate revenue: %w", err)
	}

	if totalRevenue.Valid {
		metrics.TotalRevenueCents = pgtype.Int8{Int64: totalRevenue.Int64, Valid: true}
	}
	if newRevenue.Valid {
		metrics.NewRevenueCents = pgtype.Int8{Int64: newRevenue.Int64, Valid: true}
	}

	// TODO: Calculate expansion and contraction revenue by comparing subscription changes

	return nil
}

// calculateCustomerMetrics calculates customer counts and changes
func (s *DashboardMetricsService) calculateCustomerMetrics(ctx context.Context, metrics *db.CreateDashboardMetricParams, workspaceID uuid.UUID, startDate, endDate time.Time) error {
	// Total customers with active subscriptions
	query := `
		SELECT COUNT(DISTINCT c.id) as total_customers
		FROM customers c
		JOIN subscriptions s ON c.id = s.customer_id
		JOIN products p ON s.product_id = p.id
		WHERE p.workspace_id = $1
			AND s.status IN ('active', 'trial')
			AND s.deleted_at IS NULL
			AND c.deleted_at IS NULL
			AND p.deleted_at IS NULL
	`

	var totalCustomers pgtype.Int8
	err := s.pool.QueryRow(ctx, query, workspaceID).Scan(&totalCustomers)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate total customers: %w", err)
	}

	if totalCustomers.Valid {
		metrics.TotalCustomers = pgtype.Int4{Int32: int32(totalCustomers.Int64), Valid: true}
	}

	// New customers in period
	newCustomersQuery := `
		SELECT COUNT(DISTINCT c.id) as new_customers
		FROM customers c
		JOIN subscriptions s ON c.id = s.customer_id
		JOIN products p ON s.product_id = p.id
		WHERE p.workspace_id = $1
			AND c.created_at >= $2
			AND c.created_at < $3
			AND s.deleted_at IS NULL
			AND c.deleted_at IS NULL
			AND p.deleted_at IS NULL
	`

	var newCustomers pgtype.Int8
	err = s.pool.QueryRow(ctx, newCustomersQuery, workspaceID, startDate, endDate).Scan(&newCustomers)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate new customers: %w", err)
	}

	if newCustomers.Valid {
		metrics.NewCustomers = pgtype.Int4{Int32: int32(newCustomers.Int64), Valid: true}
	}

	// Churned customers (had active subscription that was cancelled in period)
	churnedQuery := `
		SELECT COUNT(DISTINCT c.id) as churned_customers
		FROM customers c
		JOIN subscriptions s ON c.id = s.customer_id
		JOIN products p ON s.product_id = p.id
		WHERE p.workspace_id = $1
			AND s.status IN ('canceled', 'expired')
			AND s.updated_at >= $2
			AND s.updated_at < $3
			AND NOT EXISTS (
				SELECT 1 FROM subscriptions s2
				JOIN products p2 ON s2.product_id = p2.id
				WHERE s2.customer_id = c.id
					AND p2.workspace_id = $1
					AND s2.status = 'active'
					AND s2.deleted_at IS NULL
			)
	`

	var churnedCustomers pgtype.Int8
	err = s.pool.QueryRow(ctx, churnedQuery, workspaceID, startDate, endDate).Scan(&churnedCustomers)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate churned customers: %w", err)
	}

	if churnedCustomers.Valid {
		metrics.ChurnedCustomers = pgtype.Int4{Int32: int32(churnedCustomers.Int64), Valid: true}
	}

	return nil
}

// calculateSubscriptionMetrics calculates subscription counts and changes
func (s *DashboardMetricsService) calculateSubscriptionMetrics(ctx context.Context, metrics *db.CreateDashboardMetricParams, workspaceID uuid.UUID, startDate, endDate time.Time) error {
	// Active subscriptions
	query := `
		SELECT 
			COUNT(CASE WHEN s.status = 'active' THEN 1 END) as active_subs,
			COUNT(CASE WHEN s.status = 'active' AND s.created_at >= $2 AND s.created_at < $3 THEN 1 END) as new_subs,
			COUNT(CASE WHEN s.status = 'canceled' AND s.updated_at >= $2 AND s.updated_at < $3 THEN 1 END) as cancelled_subs,
			COUNT(CASE WHEN s.status = 'suspended' THEN 1 END) as paused_subs,
			COUNT(CASE WHEN s.status = 'trial' THEN 1 END) as trial_subs
		FROM subscriptions s
		JOIN products p ON s.product_id = p.id
		WHERE p.workspace_id = $1
			AND s.deleted_at IS NULL
			AND p.deleted_at IS NULL
	`

	var activeSubs, newSubs, cancelledSubs, pausedSubs, trialSubs pgtype.Int8
	err := s.pool.QueryRow(ctx, query, workspaceID, startDate, endDate).Scan(
		&activeSubs, &newSubs, &cancelledSubs, &pausedSubs, &trialSubs,
	)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate subscription metrics: %w", err)
	}

	if activeSubs.Valid {
		metrics.ActiveSubscriptions = pgtype.Int4{Int32: int32(activeSubs.Int64), Valid: true}
	}
	if newSubs.Valid {
		metrics.NewSubscriptions = pgtype.Int4{Int32: int32(newSubs.Int64), Valid: true}
	}
	if cancelledSubs.Valid {
		metrics.CancelledSubscriptions = pgtype.Int4{Int32: int32(cancelledSubs.Int64), Valid: true}
	}
	if pausedSubs.Valid {
		metrics.PausedSubscriptions = pgtype.Int4{Int32: int32(pausedSubs.Int64), Valid: true}
	}
	if trialSubs.Valid {
		metrics.TrialSubscriptions = pgtype.Int4{Int32: int32(trialSubs.Int64), Valid: true}
	}

	return nil
}

// calculatePaymentMetrics calculates payment success rates and volumes
func (s *DashboardMetricsService) calculatePaymentMetrics(ctx context.Context, metrics *db.CreateDashboardMetricParams, workspaceID uuid.UUID, startDate, endDate time.Time, currency string) error {
	query := `
		SELECT 
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			SUM(CASE WHEN status = 'completed' THEN amount_in_cents ELSE 0 END) as total_volume,
			AVG(CASE WHEN status = 'completed' THEN amount_in_cents ELSE NULL END) as avg_payment_size
		FROM payments
		WHERE workspace_id = $1
			AND currency = $2
			AND created_at >= $3
			AND created_at < $4
	`

	var successful, failed, pending, totalVolume, avgPaymentSize pgtype.Int8
	err := s.pool.QueryRow(ctx, query, workspaceID, currency, startDate, endDate).Scan(
		&successful, &failed, &pending, &totalVolume, &avgPaymentSize,
	)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate payment metrics: %w", err)
	}

	if successful.Valid {
		metrics.SuccessfulPayments = pgtype.Int4{Int32: int32(successful.Int64), Valid: true}
	}
	if failed.Valid {
		metrics.FailedPayments = pgtype.Int4{Int32: int32(failed.Int64), Valid: true}
	}
	if pending.Valid {
		metrics.PendingPayments = pgtype.Int4{Int32: int32(pending.Int64), Valid: true}
	}
	if totalVolume.Valid {
		metrics.TotalPaymentVolumeCents = pgtype.Int8{Int64: totalVolume.Int64, Valid: true}
	}
	if avgPaymentSize.Valid {
		metrics.AvgPaymentSizeCents = pgtype.Int8{Int64: avgPaymentSize.Int64, Valid: true}
	}

	return nil
}

// calculateGasFeeMetrics calculates gas fee statistics
func (s *DashboardMetricsService) calculateGasFeeMetrics(ctx context.Context, metrics *db.CreateDashboardMetricParams, workspaceID uuid.UUID, startDate, endDate time.Time, currency string) error {
	query := `
		SELECT 
			SUM(p.gas_fee_usd_cents) as total_gas_fees,
			SUM(CASE WHEN p.gas_sponsored = true THEN p.gas_fee_usd_cents ELSE 0 END) as sponsored_gas,
			SUM(CASE WHEN p.gas_sponsored = false THEN p.gas_fee_usd_cents ELSE 0 END) as customer_gas,
			AVG(p.gas_fee_usd_cents) as avg_gas_fee,
			COUNT(CASE WHEN p.gas_sponsored = true THEN 1 END)::FLOAT / NULLIF(COUNT(*), 0) as sponsorship_rate
		FROM payments p
		WHERE p.workspace_id = $1
			AND p.currency = $2
			AND p.has_gas_fee = true
			AND p.created_at >= $3
			AND p.created_at < $4
			AND p.status = 'completed'
	`

	var totalGas, sponsoredGas, customerGas, avgGasFee pgtype.Int8
	var sponsorshipRate pgtype.Float8
	err := s.pool.QueryRow(ctx, query, workspaceID, currency, startDate, endDate).Scan(
		&totalGas, &sponsoredGas, &customerGas, &avgGasFee, &sponsorshipRate,
	)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to calculate gas fee metrics: %w", err)
	}

	if totalGas.Valid {
		metrics.TotalGasFeesCents = pgtype.Int8{Int64: totalGas.Int64, Valid: true}
	}
	if sponsoredGas.Valid {
		metrics.SponsoredGasFeesCents = pgtype.Int8{Int64: sponsoredGas.Int64, Valid: true}
	}
	if customerGas.Valid {
		metrics.CustomerGasFeesCents = pgtype.Int8{Int64: customerGas.Int64, Valid: true}
	}
	if avgGasFee.Valid {
		metrics.AvgGasFeeCents = pgtype.Int8{Int64: avgGasFee.Int64, Valid: true}
	}
	if sponsorshipRate.Valid {
		metrics.GasSponsorshipRate = pgtype.Numeric{}
		metrics.GasSponsorshipRate.Scan(sponsorshipRate.Float64)
	}

	return nil
}

// calculateNetworkTokenMetrics calculates metrics broken down by network and token
func (s *DashboardMetricsService) calculateNetworkTokenMetrics(ctx context.Context, metrics *db.CreateDashboardMetricParams, workspaceID uuid.UUID, startDate, endDate time.Time, currency string) error {
	// Network metrics
	networkQuery := `
		SELECT 
			n.name as network_name,
			COUNT(p.id) as payment_count,
			SUM(p.amount_in_cents) as volume_cents,
			SUM(p.gas_fee_usd_cents) as gas_fee_cents
		FROM payments p
		JOIN networks n ON p.network_id = n.id
		WHERE p.workspace_id = $1
			AND p.currency = $2
			AND p.created_at >= $3
			AND p.created_at < $4
			AND p.status = 'completed'
		GROUP BY n.name
	`

	rows, err := s.pool.Query(ctx, networkQuery, workspaceID, currency, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to query network metrics: %w", err)
	}
	defer rows.Close()

	networkMetrics := make(map[string]business.NetworkMetrics)
	for rows.Next() {
		var networkName string
		var paymentCount pgtype.Int8
		var volumeCents, gasFeeCents pgtype.Int8

		if err := rows.Scan(&networkName, &paymentCount, &volumeCents, &gasFeeCents); err != nil {
			return fmt.Errorf("failed to scan network metrics: %w", err)
		}

		nm := business.NetworkMetrics{}
		if paymentCount.Valid {
			nm.Payments = int(paymentCount.Int64)
		}
		if volumeCents.Valid {
			nm.VolumeCents = volumeCents.Int64
		}
		if gasFeeCents.Valid {
			nm.GasFeeCents = gasFeeCents.Int64
		}
		networkMetrics[networkName] = nm
	}

	// Token metrics
	tokenQuery := `
		SELECT 
			t.symbol as token_symbol,
			COUNT(p.id) as payment_count,
			SUM(p.amount_in_cents) as volume_cents
		FROM payments p
		JOIN tokens t ON p.token_id = t.id
		WHERE p.workspace_id = $1
			AND p.currency = $2
			AND p.created_at >= $3
			AND p.created_at < $4
			AND p.status = 'completed'
		GROUP BY t.symbol
	`

	rows, err = s.pool.Query(ctx, tokenQuery, workspaceID, currency, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to query token metrics: %w", err)
	}
	defer rows.Close()

	tokenMetrics := make(map[string]business.TokenMetrics)
	for rows.Next() {
		var tokenSymbol string
		var paymentCount pgtype.Int8
		var volumeCents pgtype.Int8

		if err := rows.Scan(&tokenSymbol, &paymentCount, &volumeCents); err != nil {
			return fmt.Errorf("failed to scan token metrics: %w", err)
		}

		tm := business.TokenMetrics{}
		if paymentCount.Valid {
			tm.Payments = int(paymentCount.Int64)
		}
		if volumeCents.Valid {
			tm.VolumeCents = volumeCents.Int64
		}
		tokenMetrics[tokenSymbol] = tm
	}

	// Convert to JSON
	networkJSON, _ := json.Marshal(networkMetrics)
	tokenJSON, _ := json.Marshal(tokenMetrics)

	metrics.NetworkMetrics = networkJSON
	metrics.TokenMetrics = tokenJSON

	return nil
}

// calculateDerivedMetrics calculates rates and averages from other metrics
func (s *DashboardMetricsService) calculateDerivedMetrics(metrics *db.CreateDashboardMetricParams) {
	// Churn rate
	if metrics.TotalCustomers.Valid && metrics.TotalCustomers.Int32 > 0 {
		churnRate := float64(metrics.ChurnedCustomers.Int32) / float64(metrics.TotalCustomers.Int32)
		metrics.ChurnRate = pgtype.Numeric{}
		metrics.ChurnRate.Scan(churnRate)
	}

	// Growth rate
	if metrics.TotalCustomers.Valid && metrics.TotalCustomers.Int32 > 0 {
		newCount := int32(0)
		churnedCount := int32(0)
		if metrics.NewCustomers.Valid {
			newCount = metrics.NewCustomers.Int32
		}
		if metrics.ChurnedCustomers.Valid {
			churnedCount = metrics.ChurnedCustomers.Int32
		}
		growthRate := float64(newCount-churnedCount) / float64(metrics.TotalCustomers.Int32)
		metrics.GrowthRate = pgtype.Numeric{}
		metrics.GrowthRate.Scan(growthRate)
	}

	// Payment success rate
	var totalPayments int32
	if metrics.SuccessfulPayments.Valid {
		totalPayments += metrics.SuccessfulPayments.Int32
	}
	if metrics.FailedPayments.Valid {
		totalPayments += metrics.FailedPayments.Int32
	}
	if totalPayments > 0 && metrics.SuccessfulPayments.Valid {
		successRate := float64(metrics.SuccessfulPayments.Int32) / float64(totalPayments)
		metrics.PaymentSuccessRate = pgtype.Numeric{}
		metrics.PaymentSuccessRate.Scan(successRate)
	}

	// Calculate LTV if we have enough data
	if metrics.TotalCustomers.Valid && metrics.TotalCustomers.Int32 > 0 &&
		metrics.ChurnRate.Valid &&
		metrics.TotalRevenueCents.Valid {
		avgRevPerCustomer := float64(metrics.TotalRevenueCents.Int64) / float64(metrics.TotalCustomers.Int32)
		churnRateValue, _ := metrics.ChurnRate.Value()
		churnFloat, _ := churnRateValue.(float64)
		if churnFloat > 0 {
			ltv := avgRevPerCustomer / churnFloat
			metrics.LtvAvgCents = pgtype.Int8{Int64: int64(ltv), Valid: true}
		}
	}
}

// getDateRange calculates the start and end date for a given metric type
func (s *DashboardMetricsService) getDateRange(date time.Time, metricType MetricType, hour *int) (startDate, endDate time.Time) {
	switch metricType {
	case MetricTypeHourly:
		if hour != nil {
			startDate = time.Date(date.Year(), date.Month(), date.Day(), *hour, 0, 0, 0, date.Location())
			endDate = startDate.Add(time.Hour)
		} else {
			startDate = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
			endDate = startDate.Add(time.Hour)
		}
	case MetricTypeDaily:
		startDate = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endDate = startDate.AddDate(0, 0, 1)
	case MetricTypeWeekly:
		// Start of week (Sunday)
		weekday := int(date.Weekday())
		startDate = time.Date(date.Year(), date.Month(), date.Day()-weekday, 0, 0, 0, 0, date.Location())
		endDate = startDate.AddDate(0, 0, 7)
	case MetricTypeMonthly:
		startDate = time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
		endDate = startDate.AddDate(0, 1, 0)
	case MetricTypeYearly:
		startDate = time.Date(date.Year(), 1, 1, 0, 0, 0, 0, date.Location())
		endDate = startDate.AddDate(1, 0, 0)
	default:
		// Default to daily
		startDate = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endDate = startDate.AddDate(0, 0, 1)
	}
	return
}

// CalculateAllMetricsForWorkspace calculates all metric types for a workspace
func (s *DashboardMetricsService) CalculateAllMetricsForWorkspace(ctx context.Context, workspaceID uuid.UUID, date time.Time) error {
	// Get workspace default currency
	defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspaceID)
	var currency string
	if err != nil {
		// Fallback to USD if no default currency is set
		currency = "USD"
	} else {
		currency = defaultCurrency.Code
	}

	// Calculate hourly metrics for today
	if date.Day() == time.Now().Day() {
		for hour := 0; hour < 24; hour++ {
			if hour > time.Now().Hour() {
				break // Don't calculate future hours
			}
			h := hour
			_, err := s.CalculateAndStoreMetrics(ctx, CalculateMetricsOptions{
				WorkspaceID: workspaceID,
				Date:        date,
				MetricType:  MetricTypeHourly,
				Hour:        &h,
				Currency:    currency,
			})
			if err != nil {
				s.logger.Error("Failed to calculate hourly metrics",
					zap.Error(err),
					zap.Int("hour", hour),
				)
			}
		}
	}

	// Calculate daily metrics
	_, err = s.CalculateAndStoreMetrics(ctx, CalculateMetricsOptions{
		WorkspaceID: workspaceID,
		Date:        date,
		MetricType:  MetricTypeDaily,
		Currency:    currency,
	})
	if err != nil {
		return fmt.Errorf("failed to calculate daily metrics: %w", err)
	}

	// Calculate weekly metrics if it's Sunday
	if date.Weekday() == time.Sunday {
		_, err = s.CalculateAndStoreMetrics(ctx, CalculateMetricsOptions{
			WorkspaceID: workspaceID,
			Date:        date,
			MetricType:  MetricTypeWeekly,
			Currency:    currency,
		})
		if err != nil {
			return fmt.Errorf("failed to calculate weekly metrics: %w", err)
		}
	}

	// Calculate monthly metrics if it's the first of the month
	if date.Day() == 1 {
		_, err = s.CalculateAndStoreMetrics(ctx, CalculateMetricsOptions{
			WorkspaceID: workspaceID,
			Date:        date,
			MetricType:  MetricTypeMonthly,
			Currency:    currency,
		})
		if err != nil {
			return fmt.Errorf("failed to calculate monthly metrics: %w", err)
		}
	}

	// Calculate yearly metrics if it's January 1st
	if date.Month() == time.January && date.Day() == 1 {
		_, err = s.CalculateAndStoreMetrics(ctx, CalculateMetricsOptions{
			WorkspaceID: workspaceID,
			Date:        date,
			MetricType:  MetricTypeYearly,
			Currency:    currency,
		})
		if err != nil {
			return fmt.Errorf("failed to calculate yearly metrics: %w", err)
		}
	}

	return nil
}

// GetMetricsForDashboard retrieves various metrics for dashboard display
func (s *DashboardMetricsService) GetMetricsForDashboard(ctx context.Context, workspaceID uuid.UUID, currency string) (*DashboardData, error) {
	data := &DashboardData{}

	// Get latest daily metrics
	latest, err := s.queries.GetLatestDashboardMetrics(ctx, db.GetLatestDashboardMetricsParams{
		WorkspaceID:  workspaceID,
		MetricType:   string(MetricTypeDaily),
		FiatCurrency: currency,
	})
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get latest metrics: %w", err)
	}

	if err != pgx.ErrNoRows {
		data.CurrentMetrics = &latest
	}

	// Get revenue growth
	growth, err := s.queries.GetRevenueGrowth(ctx, db.GetRevenueGrowthParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: time.Now(), Valid: true},
		MetricType:   string(MetricTypeMonthly),
		FiatCurrency: currency,
	})
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get revenue growth: %w", err)
	}

	if err != pgx.ErrNoRows {
		data.RevenueGrowth = &growth
	}

	// Get customer trends for last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)
	trends, err := s.queries.GetCustomerMetricsTrend(ctx, db.GetCustomerMetricsTrendParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   string(MetricTypeDaily),
		FiatCurrency: currency,
	})
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get customer trends: %w", err)
	}

	data.CustomerTrends = trends

	// Get payment summary for last 30 days
	paymentSummary, err := s.queries.GetPaymentMetricsSummary(ctx, db.GetPaymentMetricsSummaryParams{
		WorkspaceID:  workspaceID,
		MetricDate:   pgtype.Date{Time: startDate, Valid: true},
		MetricDate_2: pgtype.Date{Time: endDate, Valid: true},
		MetricType:   string(MetricTypeDaily),
		FiatCurrency: currency,
	})
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to get payment summary: %w", err)
	}

	if err != pgx.ErrNoRows {
		data.PaymentSummary = &paymentSummary
	}

	return data, nil
}

// DashboardData represents comprehensive dashboard data
type DashboardData struct {
	CurrentMetrics *db.DashboardMetric
	RevenueGrowth  *db.GetRevenueGrowthRow
	CustomerTrends []db.GetCustomerMetricsTrendRow
	PaymentSummary *db.GetPaymentMetricsSummaryRow
}

// RecalculateHistoricalMetrics recalculates metrics for a date range
func (s *DashboardMetricsService) RecalculateHistoricalMetrics(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time) error {
	current := startDate
	for current.Before(endDate) || current.Equal(endDate) {
		if err := s.CalculateAllMetricsForWorkspace(ctx, workspaceID, current); err != nil {
			s.logger.Error("Failed to calculate metrics for date",
				zap.Time("date", current),
				zap.Error(err),
			)
		}
		current = current.AddDate(0, 0, 1)
	}
	return nil
}
