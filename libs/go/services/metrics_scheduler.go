package services

import (
	"context"
	"sync"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// MetricsScheduler runs periodic metric calculations
type MetricsScheduler struct {
	metricsService  *DashboardMetricsService
	queries         db.Querier
	pool            *pgx.Conn
	logger          *zap.Logger
	currencyService *CurrencyService
	stopCh          chan struct{}
	wg              sync.WaitGroup
	stopOnce        sync.Once
	stopped         bool
	mu              sync.RWMutex
}

// NewMetricsScheduler creates a new metrics scheduler
func NewMetricsScheduler(queries db.Querier, pool *pgx.Conn) *MetricsScheduler {
	return &MetricsScheduler{
		metricsService:  NewDashboardMetricsService(queries, pool),
		queries:         queries,
		pool:            pool,
		logger:          logger.Log,
		currencyService: NewCurrencyService(queries),
		stopCh:          make(chan struct{}),
	}
}

// Start begins the scheduled metric calculations
func (s *MetricsScheduler) Start() {
	s.logger.Info("Starting metrics scheduler")

	// Calculate metrics immediately on startup
	go s.calculateAllWorkspaceMetrics()

	// Schedule hourly calculations
	s.wg.Add(1)
	go s.runHourlySchedule()

	// Schedule daily calculations
	s.wg.Add(1)
	go s.runDailySchedule()
}

// Stop gracefully shuts down the scheduler
func (s *MetricsScheduler) Stop() {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		if s.stopped {
			s.mu.Unlock()
			return
		}
		s.stopped = true
		s.mu.Unlock()

		s.logger.Info("Stopping metrics scheduler")
		close(s.stopCh)
		s.wg.Wait()
	})
}

// runHourlySchedule runs metric calculations every hour
func (s *MetricsScheduler) runHourlySchedule() {
	defer s.wg.Done()

	// Calculate initial delay to start at the next hour
	now := time.Now()
	nextHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()+1, 0, 0, 0, now.Location())
	initialDelay := nextHour.Sub(now)

	// Wait for initial delay
	select {
	case <-time.After(initialDelay):
		// Proceed after delay
	case <-s.stopCh:
		return
	}

	// Run every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.calculateHourlyMetrics()
		case <-s.stopCh:
			return
		}
	}
}

// runDailySchedule runs metric calculations once per day
func (s *MetricsScheduler) runDailySchedule() {
	defer s.wg.Done()

	// Calculate initial delay to start at midnight
	now := time.Now()
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	initialDelay := tomorrow.Sub(now)

	// Wait for initial delay
	select {
	case <-time.After(initialDelay):
		// Proceed after delay
	case <-s.stopCh:
		return
	}

	// Run every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.calculateDailyMetrics()
		case <-s.stopCh:
			return
		}
	}
}

// calculateHourlyMetrics calculates hourly metrics for all workspaces
func (s *MetricsScheduler) calculateHourlyMetrics() {
	ctx := context.Background()
	now := time.Now()
	hour := now.Hour() - 1 // Calculate for the previous hour
	if hour < 0 {
		hour = 23
		now = now.AddDate(0, 0, -1)
	}

	s.logger.Info("Starting hourly metrics calculation", zap.Time("time", now), zap.Int("hour", hour))

	workspaces, err := s.queries.ListWorkspaces(ctx)
	if err != nil {
		s.logger.Error("Failed to list workspaces", zap.Error(err))
		return
	}

	for _, workspace := range workspaces {
		// Get workspace currency
		defaultCurrency, err := s.currencyService.GetWorkspaceDefaultCurrency(ctx, workspace.ID)
		var currency string
		if err != nil {
			// Fallback to USD if no default currency is set
			currency = constants.USDCurrency
		} else {
			currency = defaultCurrency.Code
		}

		// Calculate hourly metrics
		_, err = s.metricsService.CalculateAndStoreMetrics(ctx, CalculateMetricsOptions{
			WorkspaceID: workspace.ID,
			Date:        now,
			MetricType:  MetricTypeHourly,
			Hour:        &hour,
			Currency:    currency,
		})
		if err != nil {
			s.logger.Error("Failed to calculate hourly metrics",
				zap.String("workspace_id", workspace.ID.String()),
				zap.Error(err),
			)
		}
	}
}

// calculateDailyMetrics calculates daily, weekly, monthly metrics as needed
func (s *MetricsScheduler) calculateDailyMetrics() {
	ctx := context.Background()
	yesterday := time.Now().AddDate(0, 0, -1)

	s.logger.Info("Starting daily metrics calculation", zap.Time("date", yesterday))

	workspaces, err := s.queries.ListWorkspaces(ctx)
	if err != nil {
		s.logger.Error("Failed to list workspaces", zap.Error(err))
		return
	}

	for _, workspace := range workspaces {
		// Calculate all metric types for yesterday
		if err := s.metricsService.CalculateAllMetricsForWorkspace(ctx, workspace.ID, yesterday); err != nil {
			s.logger.Error("Failed to calculate daily metrics",
				zap.String("workspace_id", workspace.ID.String()),
				zap.Error(err),
			)
		}
	}

	// Clean up old metrics (keep last 90 days of hourly, 1 year of daily, 3 years of monthly)
	s.cleanupOldMetrics(ctx)
}

// calculateAllWorkspaceMetrics calculates metrics for all workspaces (used on startup)
func (s *MetricsScheduler) calculateAllWorkspaceMetrics() {
	ctx := context.Background()
	now := time.Now()

	s.logger.Info("Calculating initial metrics for all workspaces")

	workspaces, err := s.queries.ListWorkspaces(ctx)
	if err != nil {
		s.logger.Error("Failed to list workspaces", zap.Error(err))
		return
	}

	for _, workspace := range workspaces {
		// Calculate today's metrics
		if err := s.metricsService.CalculateAllMetricsForWorkspace(ctx, workspace.ID, now); err != nil {
			s.logger.Error("Failed to calculate metrics",
				zap.String("workspace_id", workspace.ID.String()),
				zap.Error(err),
			)
		}

		// Also calculate yesterday's metrics if missing
		yesterday := now.AddDate(0, 0, -1)
		if err := s.metricsService.CalculateAllMetricsForWorkspace(ctx, workspace.ID, yesterday); err != nil {
			s.logger.Error("Failed to calculate yesterday's metrics",
				zap.String("workspace_id", workspace.ID.String()),
				zap.Error(err),
			)
		}
	}
}

// cleanupOldMetrics removes old metric data to prevent database bloat
func (s *MetricsScheduler) cleanupOldMetrics(ctx context.Context) {
	// Delete hourly metrics older than 90 days
	hourlyDate := time.Now().AddDate(0, 0, -90)
	if err := s.queries.DeleteOldMetrics(ctx, db.DeleteOldMetricsParams{
		MetricDate: pgtype.Date{Time: hourlyDate, Valid: true},
		MetricType: "hourly",
	}); err != nil {
		s.logger.Error("Failed to cleanup old hourly metrics", zap.Error(err))
	}

	// Delete daily metrics older than 1 year
	dailyDate := time.Now().AddDate(-1, 0, 0)
	if err := s.queries.DeleteOldMetrics(ctx, db.DeleteOldMetricsParams{
		MetricDate: pgtype.Date{Time: dailyDate, Valid: true},
		MetricType: "daily",
	}); err != nil {
		s.logger.Error("Failed to cleanup old daily metrics", zap.Error(err))
	}

	// Delete weekly metrics older than 2 years
	weeklyDate := time.Now().AddDate(-2, 0, 0)
	if err := s.queries.DeleteOldMetrics(ctx, db.DeleteOldMetricsParams{
		MetricDate: pgtype.Date{Time: weeklyDate, Valid: true},
		MetricType: "weekly",
	}); err != nil {
		s.logger.Error("Failed to cleanup old weekly metrics", zap.Error(err))
	}

	// Delete monthly metrics older than 3 years
	monthlyDate := time.Now().AddDate(-3, 0, 0)
	if err := s.queries.DeleteOldMetrics(ctx, db.DeleteOldMetricsParams{
		MetricDate: pgtype.Date{Time: monthlyDate, Valid: true},
		MetricType: "monthly",
	}); err != nil {
		s.logger.Error("Failed to cleanup old monthly metrics", zap.Error(err))
	}
}

// RecalculateWorkspaceMetrics recalculates all metrics for a specific workspace
func (s *MetricsScheduler) RecalculateWorkspaceMetrics(workspaceID uuid.UUID, startDate, endDate time.Time) error {
	ctx := context.Background()
	return s.metricsService.RecalculateHistoricalMetrics(ctx, workspaceID, startDate, endDate)
}
