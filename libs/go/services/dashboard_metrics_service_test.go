package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestDashboardMetricsService_NewService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)

	// Test service creation
	service := services.NewDashboardMetricsService(mockQuerier, nil)
	assert.NotNil(t, service)
}

func TestDashboardMetricsService_MetricTypes(t *testing.T) {
	tests := []struct {
		name       string
		metricType services.MetricType
		expected   string
	}{
		{
			name:       "hourly metric type",
			metricType: services.MetricTypeHourly,
			expected:   "hourly",
		},
		{
			name:       "daily metric type",
			metricType: services.MetricTypeDaily,
			expected:   "daily",
		},
		{
			name:       "weekly metric type",
			metricType: services.MetricTypeWeekly,
			expected:   "weekly",
		},
		{
			name:       "monthly metric type",
			metricType: services.MetricTypeMonthly,
			expected:   "monthly",
		},
		{
			name:       "yearly metric type",
			metricType: services.MetricTypeYearly,
			expected:   "yearly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.metricType))
		})
	}
}

func TestDashboardMetricsService_CalculateMetricsOptions(t *testing.T) {
	workspaceID := uuid.New()
	testDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	hour := 14

	tests := []struct {
		name     string
		opts     services.CalculateMetricsOptions
		validate func(services.CalculateMetricsOptions)
	}{
		{
			name: "daily metrics options",
			opts: services.CalculateMetricsOptions{
				WorkspaceID: workspaceID,
				Date:        testDate,
				MetricType:  services.MetricTypeDaily,
				Currency:    "USD",
			},
			validate: func(opts services.CalculateMetricsOptions) {
				assert.Equal(t, workspaceID, opts.WorkspaceID)
				assert.Equal(t, testDate, opts.Date)
				assert.Equal(t, services.MetricTypeDaily, opts.MetricType)
				assert.Equal(t, "USD", opts.Currency)
				assert.Nil(t, opts.Hour)
			},
		},
		{
			name: "hourly metrics options with hour",
			opts: services.CalculateMetricsOptions{
				WorkspaceID: workspaceID,
				Date:        testDate,
				MetricType:  services.MetricTypeHourly,
				Hour:        &hour,
				Currency:    "EUR",
			},
			validate: func(opts services.CalculateMetricsOptions) {
				assert.Equal(t, workspaceID, opts.WorkspaceID)
				assert.Equal(t, testDate, opts.Date)
				assert.Equal(t, services.MetricTypeHourly, opts.MetricType)
				assert.Equal(t, "EUR", opts.Currency)
				assert.NotNil(t, opts.Hour)
				assert.Equal(t, 14, *opts.Hour)
			},
		},
		{
			name: "weekly metrics options",
			opts: services.CalculateMetricsOptions{
				WorkspaceID: workspaceID,
				Date:        testDate,
				MetricType:  services.MetricTypeWeekly,
				Currency:    "GBP",
			},
			validate: func(opts services.CalculateMetricsOptions) {
				assert.Equal(t, services.MetricTypeWeekly, opts.MetricType)
				assert.Equal(t, "GBP", opts.Currency)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(tt.opts)
		})
	}
}

func TestDashboardMetricsService_DashboardData(t *testing.T) {
	tests := []struct {
		name     string
		data     services.DashboardData
		validate func(services.DashboardData)
	}{
		{
			name: "empty dashboard data",
			data: services.DashboardData{},
			validate: func(data services.DashboardData) {
				assert.Nil(t, data.CurrentMetrics)
				assert.Nil(t, data.RevenueGrowth)
				assert.Nil(t, data.CustomerTrends)
				assert.Nil(t, data.PaymentSummary)
			},
		},
		{
			name: "dashboard data with metrics",
			data: services.DashboardData{
				CurrentMetrics: &db.DashboardMetric{
					ID:         uuid.New(),
					MetricType: "daily",
					MrrCents:   pgtype.Int8{Int64: 50000, Valid: true},
				},
				RevenueGrowth: &db.GetRevenueGrowthRow{
					CurrentRevenue: pgtype.Int8{Int64: 100000, Valid: true},
				},
				CustomerTrends: []db.GetCustomerMetricsTrendRow{
					{
						MetricDate:     pgtype.Date{Time: time.Now(), Valid: true},
						TotalCustomers: pgtype.Int4{Int32: 100, Valid: true},
					},
				},
				PaymentSummary: &db.GetPaymentMetricsSummaryRow{
					TotalSuccessfulPayments: 50,
				},
			},
			validate: func(data services.DashboardData) {
				assert.NotNil(t, data.CurrentMetrics)
				assert.Equal(t, "daily", data.CurrentMetrics.MetricType)
				assert.True(t, data.CurrentMetrics.MrrCents.Valid)
				assert.Equal(t, int64(50000), data.CurrentMetrics.MrrCents.Int64)

				assert.NotNil(t, data.RevenueGrowth)
				assert.True(t, data.RevenueGrowth.CurrentRevenue.Valid)
				assert.Equal(t, int64(100000), data.RevenueGrowth.CurrentRevenue.Int64)

				assert.Len(t, data.CustomerTrends, 1)
				assert.True(t, data.CustomerTrends[0].TotalCustomers.Valid)
				assert.Equal(t, int32(100), data.CustomerTrends[0].TotalCustomers.Int32)

				assert.NotNil(t, data.PaymentSummary)
				assert.Equal(t, int64(50), data.PaymentSummary.TotalSuccessfulPayments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(tt.data)
		})
	}
}

func TestDashboardMetricsService_GetMetricsForDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewDashboardMetricsService(mockQuerier, nil)
	ctx := context.Background()

	workspaceID := uuid.New()
	currency := "USD"

	tests := []struct {
		name           string
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(*services.DashboardData)
	}{
		{
			name: "successfully retrieve dashboard data",
			setupMocks: func() {
				// Mock latest dashboard metrics
				expectedMetric := db.DashboardMetric{
					ID:                  uuid.New(),
					WorkspaceID:         workspaceID,
					MetricType:          "daily",
					FiatCurrency:        currency,
					MrrCents:            pgtype.Int8{Int64: 50000, Valid: true},
					ActiveSubscriptions: pgtype.Int4{Int32: 10, Valid: true},
					TotalCustomers:      pgtype.Int4{Int32: 100, Valid: true},
				}

				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, gomock.Any()).
					Return(expectedMetric, nil)

				// Mock revenue growth
				expectedGrowth := db.GetRevenueGrowthRow{
					CurrentRevenue:   pgtype.Int8{Int64: 100000, Valid: true},
					PreviousRevenue:  pgtype.Int8{Int64: 90000, Valid: true},
					GrowthPercentage: 11,
				}

				mockQuerier.EXPECT().
					GetRevenueGrowth(ctx, gomock.Any()).
					Return(expectedGrowth, nil)

				// Mock customer trends
				expectedTrends := []db.GetCustomerMetricsTrendRow{
					{
						MetricDate:     pgtype.Date{Time: time.Now().AddDate(0, 0, -1), Valid: true},
						TotalCustomers: pgtype.Int4{Int32: 95, Valid: true},
						NewCustomers:   pgtype.Int4{Int32: 5, Valid: true},
					},
					{
						MetricDate:     pgtype.Date{Time: time.Now(), Valid: true},
						TotalCustomers: pgtype.Int4{Int32: 100, Valid: true},
						NewCustomers:   pgtype.Int4{Int32: 5, Valid: true},
					},
				}

				mockQuerier.EXPECT().
					GetCustomerMetricsTrend(ctx, gomock.Any()).
					Return(expectedTrends, nil)

				// Mock payment summary
				expectedSummary := db.GetPaymentMetricsSummaryRow{
					TotalSuccessfulPayments: 95,
					TotalFailedPayments:     5,
					TotalVolumeCents:        100000,
					AvgSuccessRate:          0.95,
				}

				mockQuerier.EXPECT().
					GetPaymentMetricsSummary(ctx, gomock.Any()).
					Return(expectedSummary, nil)
			},
			wantErr: false,
			validateResult: func(data *services.DashboardData) {
				assert.NotNil(t, data.CurrentMetrics)
				assert.Equal(t, workspaceID, data.CurrentMetrics.WorkspaceID)
				assert.NotNil(t, data.RevenueGrowth)
				assert.Len(t, data.CustomerTrends, 2)
				assert.NotNil(t, data.PaymentSummary)
			},
		},
		{
			name: "handle no metrics found",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, gomock.Any()).
					Return(db.DashboardMetric{}, pgx.ErrNoRows)

				mockQuerier.EXPECT().
					GetRevenueGrowth(ctx, gomock.Any()).
					Return(db.GetRevenueGrowthRow{}, pgx.ErrNoRows)

				mockQuerier.EXPECT().
					GetCustomerMetricsTrend(ctx, gomock.Any()).
					Return([]db.GetCustomerMetricsTrendRow{}, nil)

				mockQuerier.EXPECT().
					GetPaymentMetricsSummary(ctx, gomock.Any()).
					Return(db.GetPaymentMetricsSummaryRow{}, pgx.ErrNoRows)
			},
			wantErr: false,
			validateResult: func(data *services.DashboardData) {
				assert.Nil(t, data.CurrentMetrics)
				assert.Nil(t, data.RevenueGrowth)
				assert.Empty(t, data.CustomerTrends)
				assert.Nil(t, data.PaymentSummary)
			},
		},
		{
			name: "handle database error",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetLatestDashboardMetrics(ctx, gomock.Any()).
					Return(db.DashboardMetric{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to get latest metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetMetricsForDashboard(ctx, workspaceID, currency)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
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
