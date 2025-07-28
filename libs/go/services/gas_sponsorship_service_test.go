package services_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestGasSponsorshipService_ShouldSponsorGas(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasSponsorshipService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	productID := uuid.New()

	tests := []struct {
		name           string
		params         params.SponsorshipCheckParams
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(*business.SponsorshipDecision)
	}{
		{
			name: "no sponsorship config exists",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(db.GasSponsorshipConfig{}, pgx.ErrNoRows)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.False(t, decision.ShouldSponsor)
				assert.Equal(t, "customer", decision.SponsorType)
				assert.Equal(t, "No sponsorship configured", decision.Reason)
			},
		},
		{
			name: "sponsorship disabled",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				config := db.GasSponsorshipConfig{
					WorkspaceID:        workspaceID,
					SponsorshipEnabled: pgtype.Bool{Bool: false, Valid: true},
					SponsorCustomerGas: pgtype.Bool{Bool: false, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.False(t, decision.ShouldSponsor)
				assert.Equal(t, "Sponsorship not enabled for workspace", decision.Reason)
			},
		},
		{
			name: "monthly budget exhausted",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				config := db.GasSponsorshipConfig{
					WorkspaceID:            workspaceID,
					SponsorshipEnabled:     pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:     pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents:  pgtype.Int8{Int64: 1000, Valid: true},
					CurrentMonthSpentCents: pgtype.Int8{Int64: 950, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.False(t, decision.ShouldSponsor)
				assert.Equal(t, "Monthly sponsorship budget exhausted", decision.Reason)
				assert.Equal(t, int64(50), decision.RemainingBudget)
			},
		},
		{
			name: "gas cost exceeds threshold",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 500,
			},
			setupMocks: func() {
				config := db.GasSponsorshipConfig{
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 10000, Valid: true},
					CurrentMonthSpentCents:   pgtype.Int8{Int64: 1000, Valid: true},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 300, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.False(t, decision.ShouldSponsor)
				assert.Contains(t, decision.Reason, "Gas cost exceeds threshold")
			},
		},
		{
			name: "product not eligible for sponsorship",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				otherProductID := uuid.New()
				sponsorProducts, _ := json.Marshal([]uuid.UUID{otherProductID})
				config := db.GasSponsorshipConfig{
					WorkspaceID:           workspaceID,
					SponsorshipEnabled:    pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:    pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents: pgtype.Int8{Int64: 10000, Valid: true},
					SponsorForProducts:    sponsorProducts,
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.False(t, decision.ShouldSponsor)
				assert.Equal(t, "Product not eligible for sponsorship", decision.Reason)
			},
		},
		{
			name: "customer not eligible for sponsorship",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				otherCustomerID := uuid.New()
				sponsorCustomers, _ := json.Marshal([]uuid.UUID{otherCustomerID})
				config := db.GasSponsorshipConfig{
					WorkspaceID:           workspaceID,
					SponsorshipEnabled:    pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:    pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents: pgtype.Int8{Int64: 10000, Valid: true},
					SponsorForCustomers:   sponsorCustomers,
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.False(t, decision.ShouldSponsor)
				assert.Equal(t, "Customer not eligible for sponsorship", decision.Reason)
			},
		},
		{
			name: "customer tier not eligible for sponsorship",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				sponsorTiers, _ := json.Marshal([]string{"gold", "platinum"})
				config := db.GasSponsorshipConfig{
					WorkspaceID:           workspaceID,
					SponsorshipEnabled:    pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:    pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents: pgtype.Int8{Int64: 10000, Valid: true},
					SponsorForTiers:       sponsorTiers,
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.False(t, decision.ShouldSponsor)
				assert.Equal(t, "Customer tier not eligible for sponsorship", decision.Reason)
			},
		},
		{
			name: "sponsorship approved - all checks pass",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "gold",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				sponsorTiers, _ := json.Marshal([]string{"gold", "platinum"})
				config := db.GasSponsorshipConfig{
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 10000, Valid: true},
					CurrentMonthSpentCents:   pgtype.Int8{Int64: 2000, Valid: true},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 200, Valid: true},
					SponsorForTiers:          sponsorTiers,
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(decision *business.SponsorshipDecision) {
				assert.True(t, decision.ShouldSponsor)
				assert.Equal(t, "merchant", decision.SponsorType)
				assert.Equal(t, workspaceID, decision.SponsorID)
				assert.Equal(t, "Sponsorship approved", decision.Reason)
				assert.Equal(t, int64(8000), decision.RemainingBudget)
			},
		},
		{
			name: "database error retrieving config",
			params: params.SponsorshipCheckParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       productID,
				CustomerTier:    "bronze",
				GasCostUSDCents: 100,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(db.GasSponsorshipConfig{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to get sponsorship config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.ShouldSponsorGas(ctx, tt.params)

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

func TestGasSponsorshipService_RecordSponsoredTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasSponsorshipService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	paymentID := uuid.New()

	tests := []struct {
		name        string
		record      business.SponsorshipRecord
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successful transaction recording",
			record: business.SponsorshipRecord{
				WorkspaceID:     workspaceID,
				PaymentID:       paymentID,
				GasCostUSDCents: 150,
				SponsorType:     "merchant",
				SponsorID:       workspaceID,
			},
			setupMocks: func() {
				config := db.GasSponsorshipConfig{
					WorkspaceID:            workspaceID,
					CurrentMonthSpentCents: pgtype.Int8{Int64: 1000, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)

				mockQuerier.EXPECT().
					UpdateGasSponsorshipSpending(ctx, db.UpdateGasSponsorshipSpendingParams{
						WorkspaceID:            workspaceID,
						CurrentMonthSpentCents: pgtype.Int8{Int64: 1150, Valid: true},
						UpdatedAt:              pgtype.Timestamptz{Valid: false},
					}).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error getting current spending",
			record: business.SponsorshipRecord{
				WorkspaceID:     workspaceID,
				PaymentID:       paymentID,
				GasCostUSDCents: 150,
				SponsorType:     "merchant",
				SponsorID:       workspaceID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(db.GasSponsorshipConfig{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to get current spending",
		},
		{
			name: "error updating spending",
			record: business.SponsorshipRecord{
				WorkspaceID:     workspaceID,
				PaymentID:       paymentID,
				GasCostUSDCents: 150,
				SponsorType:     "merchant",
				SponsorID:       workspaceID,
			},
			setupMocks: func() {
				config := db.GasSponsorshipConfig{
					WorkspaceID:            workspaceID,
					CurrentMonthSpentCents: pgtype.Int8{Int64: 1000, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)

				mockQuerier.EXPECT().
					UpdateGasSponsorshipSpending(ctx, gomock.Any()).
					Return(assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to update sponsorship spending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.RecordSponsoredTransaction(ctx, tt.record)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGasSponsorshipService_GetSponsorshipBudgetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasSponsorshipService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		setupMocks     func()
		wantErr        bool
		errorString    string
		validateResult func(*business.BudgetStatus)
	}{
		{
			name:        "successful budget status retrieval",
			workspaceID: workspaceID,
			setupMocks: func() {
				lastReset := time.Now().AddDate(0, -1, 0)
				config := db.GasSponsorshipConfig{
					WorkspaceID:            workspaceID,
					SponsorshipEnabled:     pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents:  pgtype.Int8{Int64: 5000, Valid: true},
					CurrentMonthSpentCents: pgtype.Int8{Int64: 2000, Valid: true},
					LastResetDate:          pgtype.Date{Time: lastReset, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(config, nil)
			},
			wantErr: false,
			validateResult: func(status *business.BudgetStatus) {
				assert.Equal(t, workspaceID, status.WorkspaceID)
				assert.True(t, status.SponsorshipEnabled)
				assert.Equal(t, int64(5000), status.MonthlyBudgetCents)
				assert.Equal(t, int64(2000), status.CurrentMonthSpentCents)
				assert.Equal(t, int64(3000), status.RemainingBudgetCents)
			},
		},
		{
			name:        "no config exists - default status",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(db.GasSponsorshipConfig{}, pgx.ErrNoRows)
			},
			wantErr: false,
			validateResult: func(status *business.BudgetStatus) {
				assert.Equal(t, workspaceID, status.WorkspaceID)
				assert.False(t, status.SponsorshipEnabled)
				assert.Equal(t, int64(0), status.MonthlyBudgetCents)
				assert.Equal(t, int64(0), status.CurrentMonthSpentCents)
			},
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(db.GasSponsorshipConfig{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to get sponsorship config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetSponsorshipBudgetStatus(ctx, tt.workspaceID)

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

func TestGasSponsorshipService_ResetMonthlySponsorshipBudgets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasSponsorshipService(mockQuerier)
	ctx := context.Background()

	workspaceID1 := uuid.New()
	workspaceID2 := uuid.New()

	tests := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successful budget reset for multiple workspaces",
			setupMocks: func() {
				configs := []db.GasSponsorshipConfig{
					{WorkspaceID: workspaceID1},
					{WorkspaceID: workspaceID2},
				}
				mockQuerier.EXPECT().
					GetSponsorshipConfigsNeedingReset(ctx, gomock.Any()).
					Return(configs, nil)

				mockQuerier.EXPECT().
					ResetGasSponsorshipMonthlySpending(ctx, gomock.Any()).
					Return(nil).
					Times(2)
			},
			wantErr: false,
		},
		{
			name: "partial success with one error",
			setupMocks: func() {
				configs := []db.GasSponsorshipConfig{
					{WorkspaceID: workspaceID1},
					{WorkspaceID: workspaceID2},
				}
				mockQuerier.EXPECT().
					GetSponsorshipConfigsNeedingReset(ctx, gomock.Any()).
					Return(configs, nil)

				gomock.InOrder(
					mockQuerier.EXPECT().
						ResetGasSponsorshipMonthlySpending(ctx, gomock.Any()).
						Return(nil),
					mockQuerier.EXPECT().
						ResetGasSponsorshipMonthlySpending(ctx, gomock.Any()).
						Return(assert.AnError),
				)
			},
			wantErr: false, // Method continues on individual errors
		},
		{
			name: "error getting configs needing reset",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetSponsorshipConfigsNeedingReset(ctx, gomock.Any()).
					Return(nil, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to get configs needing reset",
		},
		{
			name: "no configs need reset",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetSponsorshipConfigsNeedingReset(ctx, gomock.Any()).
					Return([]db.GasSponsorshipConfig{}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.ResetMonthlySponsorshipBudgets(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGasSponsorshipService_CreateDefaultSponsorshipConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasSponsorshipService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successful default config creation",
			workspaceID: workspaceID,
			setupMocks: func() {
				expectedParams := db.CreateGasSponsorshipConfigParams{
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: false, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: false, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 0, Valid: false},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 0, Valid: false},
				}
				mockQuerier.EXPECT().
					CreateGasSponsorshipConfig(ctx, expectedParams).
					Return(db.GasSponsorshipConfig{WorkspaceID: workspaceID}, nil)
			},
			wantErr: false,
		},
		{
			name:        "database error creating config",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().
					CreateGasSponsorshipConfig(ctx, gomock.Any()).
					Return(db.GasSponsorshipConfig{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to create default sponsorship config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.CreateDefaultSponsorshipConfig(ctx, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGasSponsorshipService_UpdateSponsorshipConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasSponsorshipService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		updates     business.SponsorshipConfigUpdates
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successful config update with all fields",
			workspaceID: workspaceID,
			updates: business.SponsorshipConfigUpdates{
				SponsorshipEnabled:       gasBoolPtr(true),
				SponsorCustomerGas:       gasBoolPtr(true),
				MonthlyBudgetUSDCents:    gasInt64Ptr(5000),
				SponsorThresholdUSDCents: gasInt64Ptr(200),
			},
			setupMocks: func() {
				existingConfig := db.GasSponsorshipConfig{
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: false, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: false, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 0, Valid: false},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 0, Valid: false},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(existingConfig, nil)

				expectedParams := db.UpdateGasSponsorshipConfigParams{
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: true, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 5000, Valid: true},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 200, Valid: true},
				}
				mockQuerier.EXPECT().
					UpdateGasSponsorshipConfig(ctx, expectedParams).
					Return(db.GasSponsorshipConfig{}, nil)
			},
			wantErr: false,
		},
		{
			name:        "partial config update",
			workspaceID: workspaceID,
			updates: business.SponsorshipConfigUpdates{
				SponsorshipEnabled: gasBoolPtr(true),
			},
			setupMocks: func() {
				existingConfig := db.GasSponsorshipConfig{
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: false, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: false, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 1000, Valid: true},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 100, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(existingConfig, nil)

				expectedParams := db.UpdateGasSponsorshipConfigParams{
					WorkspaceID:              workspaceID,
					SponsorshipEnabled:       pgtype.Bool{Bool: true, Valid: true},
					SponsorCustomerGas:       pgtype.Bool{Bool: false, Valid: true},
					MonthlyBudgetUsdCents:    pgtype.Int8{Int64: 1000, Valid: true},
					SponsorThresholdUsdCents: pgtype.Int8{Int64: 100, Valid: true},
				}
				mockQuerier.EXPECT().
					UpdateGasSponsorshipConfig(ctx, expectedParams).
					Return(db.GasSponsorshipConfig{}, nil)
			},
			wantErr: false,
		},
		{
			name:        "error getting existing config",
			workspaceID: workspaceID,
			updates:     business.SponsorshipConfigUpdates{},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(db.GasSponsorshipConfig{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to get existing config",
		},
		{
			name:        "error updating config",
			workspaceID: workspaceID,
			updates: business.SponsorshipConfigUpdates{
				SponsorshipEnabled: gasBoolPtr(true),
			},
			setupMocks: func() {
				existingConfig := db.GasSponsorshipConfig{
					WorkspaceID:        workspaceID,
					SponsorshipEnabled: pgtype.Bool{Bool: false, Valid: true},
				}
				mockQuerier.EXPECT().
					GetGasSponsorshipConfig(ctx, workspaceID).
					Return(existingConfig, nil)

				mockQuerier.EXPECT().
					UpdateGasSponsorshipConfig(ctx, gomock.Any()).
					Return(db.GasSponsorshipConfig{}, assert.AnError)
			},
			wantErr:     true,
			errorString: "failed to update sponsorship config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.UpdateSponsorshipConfig(ctx, tt.workspaceID, tt.updates)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGasSponsorshipService_GetSponsorshipAnalytics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewGasSponsorshipService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()

	tests := []struct {
		name           string
		workspaceID    uuid.UUID
		days           int
		setupMocks     func()
		wantErr        bool
		validateResult func(*business.SponsorshipAnalytics)
	}{
		{
			name:        "successful analytics retrieval",
			workspaceID: workspaceID,
			days:        30,
			setupMocks:  func() {}, // No mocks needed for current implementation
			wantErr:     false,
			validateResult: func(analytics *business.SponsorshipAnalytics) {
				assert.Equal(t, int64(0), analytics.TotalTransactions)
				assert.Equal(t, int64(0), analytics.SponsoredTransactions)
				assert.Equal(t, int64(0), analytics.TotalGasCostCents)
				assert.Equal(t, int64(0), analytics.SponsoredCostCents)
				assert.Equal(t, 0.0, analytics.SavingsPercentage)
				assert.Equal(t, 30, analytics.Period)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetSponsorshipAnalytics(ctx, tt.workspaceID, tt.days)

			if tt.wantErr {
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

// Helper functions for creating pointers
func gasBoolPtr(b bool) *bool {
	return &b
}

func gasInt64Ptr(i int64) *int64 {
	return &i
}
