package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

// Test helper functions
func createTestAnalyticsService(ctrl *gomock.Controller) (*mocks.MockQuerier, *services.AnalyticsService) {
	mockQuerier := mocks.NewMockQuerier(ctrl)
	analyticsService := services.NewAnalyticsService(mockQuerier, nil)
	return mockQuerier, analyticsService
}

func createTestWorkspaceID() uuid.UUID {
	return uuid.New()
}

func createTestTime() time.Time {
	return time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
}

// Test GetDashboardSummary functionality
func TestAnalyticsService_GetDashboardSummary(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		currency       string
		setupMock      func()
		expectError    bool
		validateResult func(*business.DashboardSummary)
	}{
		{
			name:        "successful dashboard summary with USD",
			workspaceID: workspaceID,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, db.GetLatestDashboardMetricsParams{
						WorkspaceID:  workspaceID,
						MetricType:   "daily",
						FiatCurrency: "USD",
					}).
					Return(db.DashboardMetric{
						MrrCents:            pgtype.Int8{Int64: 100000, Valid: true},
						ArrCents:            pgtype.Int8{Int64: 1200000, Valid: true},
						TotalRevenueCents:   pgtype.Int8{Int64: 500000, Valid: true},
						ActiveSubscriptions: pgtype.Int4{Int32: 25, Valid: true},
						TotalCustomers:      pgtype.Int4{Int32: 100, Valid: true},
						ChurnRate:           pgtype.Numeric{},
						GrowthRate:          pgtype.Numeric{},
						PaymentSuccessRate:  pgtype.Numeric{},
						UpdatedAt:           pgtype.Timestamptz{Time: createTestTime(), Valid: true},
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetRevenueGrowth(ctx, gomock.Any()).
					Return(db.GetRevenueGrowthRow{
						CurrentPeriod:    pgtype.Date{Time: createTestTime(), Valid: true},
						CurrentRevenue:   pgtype.Int8{Int64: 100000, Valid: true},
						PreviousRevenue:  pgtype.Int8{Int64: 90000, Valid: true},
						GrowthPercentage: 10.0,
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(summary *business.DashboardSummary) {
				assert.Equal(t, int64(100000), summary.MRR.AmountCents)
				assert.Equal(t, "USD", summary.MRR.Currency)
				assert.Equal(t, int64(1200000), summary.ARR.AmountCents)
				assert.Equal(t, int32(25), summary.ActiveSubscriptions)
				assert.Equal(t, int32(100), summary.TotalCustomers)
				assert.NotNil(t, summary.RevenueGrowth)
				assert.Equal(t, float64(10.0), summary.RevenueGrowth.GrowthPercentage)
			},
		},
		{
			name:        "successful dashboard summary with empty currency (fallback to USD)",
			workspaceID: workspaceID,
			currency:    "",
			setupMock: func() {
				// Mock the GetWorkspaceDefaultCurrency call when currency is empty
				mockQuerier.EXPECT().
					GetWorkspaceDefaultCurrency(ctx, workspaceID).
					Return(db.FiatCurrency{
						ID:            uuid.New(),
						Code:          "USD",
						Name:          "US Dollar",
						Symbol:        "$",
						DecimalPlaces: 2,
						IsActive:      pgtype.Bool{Bool: true, Valid: true},
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, db.GetLatestDashboardMetricsParams{
						WorkspaceID:  workspaceID,
						MetricType:   "daily",
						FiatCurrency: "USD",
					}).
					Return(db.DashboardMetric{
						MrrCents:            pgtype.Int8{Int64: 50000, Valid: true},
						ArrCents:            pgtype.Int8{Int64: 600000, Valid: true},
						TotalRevenueCents:   pgtype.Int8{Int64: 250000, Valid: true},
						ActiveSubscriptions: pgtype.Int4{Int32: 15, Valid: true},
						TotalCustomers:      pgtype.Int4{Int32: 50, Valid: true},
						ChurnRate:           pgtype.Numeric{},
						GrowthRate:          pgtype.Numeric{},
						PaymentSuccessRate:  pgtype.Numeric{},
						UpdatedAt:           pgtype.Timestamptz{Time: createTestTime(), Valid: true},
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetRevenueGrowth(ctx, gomock.Any()).
					Return(db.GetRevenueGrowthRow{}, errors.New("no revenue growth data")).
					Times(1)
			},
			expectError: false,
			validateResult: func(summary *business.DashboardSummary) {
				assert.Equal(t, int64(50000), summary.MRR.AmountCents)
				assert.Equal(t, "USD", summary.MRR.Currency)
				assert.Equal(t, int32(15), summary.ActiveSubscriptions)
				assert.Nil(t, summary.RevenueGrowth)
			},
		},
		{
			name:        "error fetching latest dashboard metrics",
			workspaceID: workspaceID,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, gomock.Any()).
					Return(db.DashboardMetric{}, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetDashboardSummary(ctx, tt.workspaceID, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

// Test GetRevenueChart functionality
func TestAnalyticsService_GetRevenueChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		period         string
		days           int
		currency       string
		setupMock      func()
		expectError    bool
		validateResult func(*business.ChartData)
	}{
		{
			name:        "successful revenue chart for 30 days",
			workspaceID: workspaceID,
			period:      "daily",
			days:        30,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetDashboardMetricsByDateRange(ctx, gomock.Any()).
					Return([]db.DashboardMetric{
						{
							MetricDate:        pgtype.Date{Time: createTestTime(), Valid: true},
							TotalRevenueCents: pgtype.Int8{Int64: 10000, Valid: true},
						},
						{
							MetricDate:        pgtype.Date{Time: createTestTime().AddDate(0, 0, 1), Valid: true},
							TotalRevenueCents: pgtype.Int8{Int64: 12000, Valid: true},
						},
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(chart *business.ChartData) {
				assert.Equal(t, "line", chart.ChartType)
				assert.Equal(t, "Revenue Over Time", chart.Title)
				assert.Equal(t, "daily", chart.Period)
				assert.Len(t, chart.Data, 2)
				assert.Equal(t, 100.0, chart.Data[0].Value) // 10000 cents = 100 dollars
				assert.Equal(t, 120.0, chart.Data[1].Value) // 12000 cents = 120 dollars
			},
		},
		{
			name:        "error fetching revenue chart data",
			workspaceID: workspaceID,
			period:      "daily",
			days:        7,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetDashboardMetricsByDateRange(ctx, gomock.Any()).
					Return(nil, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetRevenueChart(ctx, tt.workspaceID, tt.period, tt.days, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

// Test GetCustomerChart functionality
func TestAnalyticsService_GetCustomerChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		metric         string
		period         string
		days           int
		currency       string
		setupMock      func()
		expectError    bool
		validateResult func(*business.ChartData)
	}{
		{
			name:        "successful customer chart for new customers",
			workspaceID: workspaceID,
			metric:      "new",
			period:      "daily",
			days:        7,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetCustomerMetricsTrend(ctx, gomock.Any()).
					Return([]db.GetCustomerMetricsTrendRow{
						{
							MetricDate:     pgtype.Date{Time: createTestTime(), Valid: true},
							NewCustomers:   pgtype.Int4{Int32: 5, Valid: true},
							TotalCustomers: pgtype.Int4{Int32: 100, Valid: true},
						},
						{
							MetricDate:     pgtype.Date{Time: createTestTime().AddDate(0, 0, 1), Valid: true},
							NewCustomers:   pgtype.Int4{Int32: 3, Valid: true},
							TotalCustomers: pgtype.Int4{Int32: 103, Valid: true},
						},
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(chart *business.ChartData) {
				assert.Equal(t, "line", chart.ChartType)
				assert.Equal(t, "New Customers", chart.Title)
				assert.Equal(t, "daily", chart.Period)
				assert.Len(t, chart.Data, 2)
				assert.Equal(t, 5.0, chart.Data[0].Value)
				assert.Equal(t, 3.0, chart.Data[1].Value)
			},
		},
		{
			name:        "successful customer chart for total customers",
			workspaceID: workspaceID,
			metric:      "total",
			period:      "daily",
			days:        7,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetCustomerMetricsTrend(ctx, gomock.Any()).
					Return([]db.GetCustomerMetricsTrendRow{
						{
							MetricDate:     pgtype.Date{Time: createTestTime(), Valid: true},
							TotalCustomers: pgtype.Int4{Int32: 100, Valid: true},
						},
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(chart *business.ChartData) {
				assert.Equal(t, "Total Customers", chart.Title)
				assert.Equal(t, 100.0, chart.Data[0].Value)
			},
		},
		{
			name:        "error fetching customer metrics",
			workspaceID: workspaceID,
			metric:      "new",
			period:      "daily",
			days:        7,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetCustomerMetricsTrend(ctx, gomock.Any()).
					Return(nil, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetCustomerChart(ctx, tt.workspaceID, tt.metric, tt.period, tt.days, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

// Test GetPaymentMetrics functionality
func TestAnalyticsService_GetPaymentMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		days           int
		currency       string
		setupMock      func()
		expectError    bool
		validateResult func(*business.PaymentMetrics)
	}{
		{
			name:        "successful payment metrics",
			workspaceID: workspaceID,
			days:        30,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetPaymentMetricsSummary(ctx, gomock.Any()).
					Return(db.GetPaymentMetricsSummaryRow{
						TotalSuccessfulPayments: 150,
						TotalFailedPayments:     10,
						TotalVolumeCents:        500000,
						AvgSuccessRate:          95.5,
						TotalGasFees:            5000,
						TotalSponsoredGas:       3000,
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(metrics *business.PaymentMetrics) {
				assert.Equal(t, int64(150), metrics.TotalSuccessful)
				assert.Equal(t, int64(10), metrics.TotalFailed)
				assert.Equal(t, int64(500000), metrics.TotalVolume.AmountCents)
				assert.Equal(t, "USD", metrics.TotalVolume.Currency)
				assert.Equal(t, 95.5, metrics.SuccessRate)
				assert.Equal(t, int64(5000), metrics.GasMetrics.TotalGasFees.AmountCents)
				assert.Equal(t, int64(3000), metrics.GasMetrics.SponsoredGasFees.AmountCents)
				assert.Equal(t, 30, metrics.Period)
			},
		},
		{
			name:        "error fetching payment metrics",
			workspaceID: workspaceID,
			days:        7,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetPaymentMetricsSummary(ctx, gomock.Any()).
					Return(db.GetPaymentMetricsSummaryRow{}, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetPaymentMetrics(ctx, tt.workspaceID, tt.days, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

// Test GetNetworkBreakdown functionality
func TestAnalyticsService_GetNetworkBreakdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()
	testDate := createTestTime()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		date           time.Time
		currency       string
		setupMock      func()
		expectError    bool
		validateResult func(*business.NetworkBreakdown)
	}{
		{
			name:        "successful network breakdown",
			workspaceID: workspaceID,
			date:        testDate,
			currency:    "USD",
			setupMock: func() {
				networkMetrics := map[string]business.NetworkMetrics{
					"ethereum": {Payments: 50, VolumeCents: 100000, GasFeeCents: 2000},
					"polygon":  {Payments: 25, VolumeCents: 50000, GasFeeCents: 500},
				}
				tokenMetrics := map[string]business.TokenMetrics{
					"USDC": {Payments: 30, VolumeCents: 75000, AvgPriceCents: 100},
					"ETH":  {Payments: 45, VolumeCents: 75000, AvgPriceCents: 200000},
				}

				networkJSON, _ := json.Marshal(networkMetrics)
				tokenJSON, _ := json.Marshal(tokenMetrics)

				mockQuerier.EXPECT().
					GetNetworkMetrics(ctx, db.GetNetworkMetricsParams{
						WorkspaceID:  workspaceID,
						MetricDate:   pgtype.Date{Time: testDate, Valid: true},
						MetricType:   "daily",
						FiatCurrency: "USD",
					}).
					Return(db.GetNetworkMetricsRow{
						NetworkMetrics: networkJSON,
						TokenMetrics:   tokenJSON,
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(breakdown *business.NetworkBreakdown) {
				assert.Equal(t, testDate.Format("2006-01-02"), breakdown.Date)
				assert.Len(t, breakdown.Networks, 2)
				assert.Len(t, breakdown.Tokens, 2)
				assert.Contains(t, breakdown.Networks, "ethereum")
				assert.Contains(t, breakdown.Networks, "polygon")
				assert.Equal(t, 50, breakdown.Networks["ethereum"].Payments)
				assert.Equal(t, int64(100000), breakdown.Networks["ethereum"].VolumeCents)
			},
		},
		{
			name:        "error fetching network metrics",
			workspaceID: workspaceID,
			date:        testDate,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetNetworkMetrics(ctx, gomock.Any()).
					Return(db.GetNetworkMetricsRow{}, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name:        "invalid network metrics JSON",
			workspaceID: workspaceID,
			date:        testDate,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetNetworkMetrics(ctx, gomock.Any()).
					Return(db.GetNetworkMetricsRow{
						NetworkMetrics: []byte("invalid json"),
						TokenMetrics:   []byte("{}"),
					}, nil).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetNetworkBreakdown(ctx, tt.workspaceID, tt.date, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

// Test GetGasFeePieChart functionality
func TestAnalyticsService_GetGasFeePieChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		days           int
		currency       string
		setupMock      func()
		expectError    bool
		validateResult func(*business.PieChartData)
	}{
		{
			name:        "successful gas fee pie chart",
			workspaceID: workspaceID,
			days:        30,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetPaymentMetricsSummary(ctx, gomock.Any()).
					Return(db.GetPaymentMetricsSummaryRow{
						TotalGasFees:      10000, // Total: $100
						TotalSponsoredGas: 6000,  // Sponsored: $60, Customer: $40
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(chart *business.PieChartData) {
				assert.Equal(t, "pie", chart.ChartType)
				assert.Equal(t, "Gas Fee Distribution", chart.Title)
				assert.Len(t, chart.Data, 2)
				assert.Equal(t, "Merchant Sponsored", chart.Data[0].Label)
				assert.Equal(t, 60.0, chart.Data[0].Value) // $60
				assert.Equal(t, "Customer Paid", chart.Data[1].Label)
				assert.Equal(t, 40.0, chart.Data[1].Value) // $40
				assert.Equal(t, int64(10000), chart.Total.AmountCents)
				assert.Equal(t, "USD", chart.Total.Currency)
			},
		},
		{
			name:        "error fetching gas fee data",
			workspaceID: workspaceID,
			days:        7,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetPaymentMetricsSummary(ctx, gomock.Any()).
					Return(db.GetPaymentMetricsSummaryRow{}, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetGasFeePieChart(ctx, tt.workspaceID, tt.days, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

// Test GetHourlyMetrics functionality
func TestAnalyticsService_GetHourlyMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		currency       string
		setupMock      func()
		expectError    bool
		validateResult func(*business.HourlyMetrics)
	}{
		{
			name:        "successful hourly metrics",
			workspaceID: workspaceID,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetHourlyMetrics(ctx, gomock.Any()).
					Return([]db.DashboardMetric{
						{
							MetricHour:         pgtype.Int4{Int32: 9, Valid: true},
							TotalRevenueCents:  pgtype.Int8{Int64: 15000, Valid: true},
							SuccessfulPayments: pgtype.Int4{Int32: 10, Valid: true},
							NewCustomers:       pgtype.Int4{Int32: 3, Valid: true},
						},
						{
							MetricHour:         pgtype.Int4{Int32: 10, Valid: true},
							TotalRevenueCents:  pgtype.Int8{Int64: 25000, Valid: true},
							SuccessfulPayments: pgtype.Int4{Int32: 15, Valid: true},
							NewCustomers:       pgtype.Int4{Int32: 5, Valid: true},
						},
					}, nil).
					Times(1)
			},
			expectError: false,
			validateResult: func(metrics *business.HourlyMetrics) {
				assert.Equal(t, "USD", metrics.Currency)
				assert.Len(t, metrics.HourlyData, 2)
				assert.Equal(t, 9, metrics.HourlyData[0].Hour)
				assert.Equal(t, 150.0, metrics.HourlyData[0].Revenue) // 15000 cents = $150
				assert.Equal(t, 10, metrics.HourlyData[0].Payments)
				assert.Equal(t, 3, metrics.HourlyData[0].NewUsers)
				assert.Equal(t, 10, metrics.HourlyData[1].Hour)
				assert.Equal(t, 250.0, metrics.HourlyData[1].Revenue) // 25000 cents = $250
			},
		},
		{
			name:        "error fetching hourly metrics",
			workspaceID: workspaceID,
			currency:    "USD",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetHourlyMetrics(ctx, gomock.Any()).
					Return(nil, errors.New("database error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetHourlyMetrics(ctx, tt.workspaceID, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(result)
				}
			}
		})
	}
}

// Test TriggerMetricsRefresh functionality
func TestAnalyticsService_TriggerMetricsRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()
	testDate := createTestTime()

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		date        time.Time
		expectError bool
	}{
		{
			name:        "successful metrics refresh trigger",
			workspaceID: workspaceID,
			date:        testDate,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TriggerMetricsRefresh(ctx, tt.workspaceID, tt.date)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test edge cases and validation scenarios
func TestAnalyticsService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name        string
		operation   func() error
		setupMock   func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil context handling",
			operation: func() error {
				_, err := service.GetDashboardSummary(nil, workspaceID, "USD")
				return err
			},
			setupMock: func() {
				// Mock the database calls that happen even with nil context
				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(nil, gomock.Any()).
					Return(db.DashboardMetric{}, errors.New("context error")).
					Times(1)
			},
			expectError: true,
		},
		{
			name: "empty workspace ID",
			operation: func() error {
				_, err := service.GetDashboardSummary(ctx, uuid.Nil, "USD")
				return err
			},
			setupMock: func() {
				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, gomock.Any()).
					Return(db.DashboardMetric{}, errors.New("invalid workspace")).
					Times(1)
			},
			expectError: true,
			errorMsg:    "invalid workspace",
		},
		{
			name: "very large date range",
			operation: func() error {
				_, err := service.GetRevenueChart(ctx, workspaceID, "daily", 10000, "USD")
				return err
			},
			setupMock: func() {
				mockQuerier.EXPECT().
					GetDashboardMetricsByDateRange(ctx, gomock.Any()).
					Return(nil, errors.New("date range too large")).
					Times(1)
			},
			expectError: true,
			errorMsg:    "date range too large",
		},
		{
			name: "negative days parameter",
			operation: func() error {
				_, err := service.GetPaymentMetrics(ctx, workspaceID, -5, "USD")
				return err
			},
			setupMock: func() {
				// Service should handle negative days gracefully
				mockQuerier.EXPECT().
					GetPaymentMetricsSummary(ctx, gomock.Any()).
					Return(db.GetPaymentMetricsSummaryRow{}, nil).
					Times(1)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := tt.operation()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test currency fallback scenarios
func TestAnalyticsService_CurrencyFallbackScenarios(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier, service := createTestAnalyticsService(ctrl)
	ctx := context.Background()
	workspaceID := createTestWorkspaceID()

	tests := []struct {
		name             string
		currency         string
		setupMock        func()
		expectError      bool
		expectedCurrency string
	}{
		{
			name:     "empty currency defaults to USD",
			currency: "",
			setupMock: func() {
				// Simulate currency service returning error (no default currency) - should fallback to USD
				mockQuerier.EXPECT().
					GetWorkspaceDefaultCurrency(ctx, workspaceID).
					Return(db.FiatCurrency{}, errors.New("no default currency")).
					Times(1)

				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, db.GetLatestDashboardMetricsParams{
						WorkspaceID:  workspaceID,
						MetricType:   "daily",
						FiatCurrency: "USD", // Should fallback to USD
					}).
					Return(db.DashboardMetric{
						MrrCents:            pgtype.Int8{Int64: 100000, Valid: true},
						ArrCents:            pgtype.Int8{Int64: 1200000, Valid: true},
						TotalRevenueCents:   pgtype.Int8{Int64: 500000, Valid: true},
						ActiveSubscriptions: pgtype.Int4{Int32: 25, Valid: true},
						TotalCustomers:      pgtype.Int4{Int32: 100, Valid: true},
						ChurnRate:           pgtype.Numeric{},
						GrowthRate:          pgtype.Numeric{},
						PaymentSuccessRate:  pgtype.Numeric{},
						UpdatedAt:           pgtype.Timestamptz{Time: createTestTime(), Valid: true},
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetRevenueGrowth(ctx, gomock.Any()).
					Return(db.GetRevenueGrowthRow{}, errors.New("no revenue growth data")).
					Times(1)
			},
			expectError:      false,
			expectedCurrency: "USD",
		},
		{
			name:     "valid currency passed through",
			currency: "EUR",
			setupMock: func() {
				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, db.GetLatestDashboardMetricsParams{
						WorkspaceID:  workspaceID,
						MetricType:   "daily",
						FiatCurrency: "EUR",
					}).
					Return(db.DashboardMetric{
						MrrCents:            pgtype.Int8{Int64: 100000, Valid: true},
						ArrCents:            pgtype.Int8{Int64: 1200000, Valid: true},
						TotalRevenueCents:   pgtype.Int8{Int64: 500000, Valid: true},
						ActiveSubscriptions: pgtype.Int4{Int32: 25, Valid: true},
						TotalCustomers:      pgtype.Int4{Int32: 100, Valid: true},
						ChurnRate:           pgtype.Numeric{},
						GrowthRate:          pgtype.Numeric{},
						PaymentSuccessRate:  pgtype.Numeric{},
						UpdatedAt:           pgtype.Timestamptz{Time: createTestTime(), Valid: true},
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetRevenueGrowth(ctx, gomock.Any()).
					Return(db.GetRevenueGrowthRow{}, errors.New("no revenue growth data")).
					Times(1)
			},
			expectError:      false,
			expectedCurrency: "EUR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.GetDashboardSummary(ctx, workspaceID, tt.currency)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedCurrency, result.MRR.Currency)
				assert.Equal(t, tt.expectedCurrency, result.ARR.Currency)
				assert.Equal(t, tt.expectedCurrency, result.TotalRevenue.Currency)
			}
		})
	}
}
