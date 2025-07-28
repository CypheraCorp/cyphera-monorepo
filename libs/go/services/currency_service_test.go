package services_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewCurrencyService(t *testing.T) {
	tests := []struct {
		name    string
		queries db.Querier
		want    bool
	}{
		{
			name:    "valid initialization",
			queries: &db.Queries{},
			want:    true,
		},
		{
			name:    "with nil queries",
			queries: nil,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := services.NewCurrencyService(tt.queries)
			assert.Equal(t, tt.want, service != nil)
		})
	}
}

func TestCurrencyService_FormatAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueries := mocks.NewMockQuerier(ctrl)
	service := services.NewCurrencyService(mockQueries)
	ctx := context.Background()

	tests := []struct {
		name           string
		amountCents    int64
		currencyCode   string
		setupMock      func()
		expectedResult string
		expectError    bool
	}{
		{
			name:         "format USD amount",
			amountCents:  12345,
			currencyCode: "USD",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "USD").Return(db.FiatCurrency{
					ID:            uuid.New(),
					Code:          "USD",
					Name:          "US Dollar",
					Symbol:        "$",
					DecimalPlaces: 2,
					IsActive:      pgtype.Bool{Bool: true, Valid: true},
					SymbolPosition: pgtype.Text{String: "before", Valid: true},
					ThousandSeparator: pgtype.Text{String: ",", Valid: true},
					DecimalSeparator:  pgtype.Text{String: ".", Valid: true},
				}, nil)
			},
			expectedResult: "$123.45",
			expectError:    false,
		},
		{
			name:         "format EUR amount with after symbol position",
			amountCents:  98765,
			currencyCode: "EUR",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "EUR").Return(db.FiatCurrency{
					ID:            uuid.New(),
					Code:          "EUR",
					Name:          "Euro",
					Symbol:        "€",
					DecimalPlaces: 2,
					IsActive:      pgtype.Bool{Bool: true, Valid: true},
					SymbolPosition: pgtype.Text{String: "after", Valid: true},
					ThousandSeparator: pgtype.Text{String: ",", Valid: true},
					DecimalSeparator:  pgtype.Text{String: ".", Valid: true},
				}, nil)
			},
			expectedResult: "987.65€",
			expectError:    false,
		},
		{
			name:         "database error",
			amountCents:  12345,
			currencyCode: "INVALID",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "INVALID").Return(db.FiatCurrency{}, assert.AnError)
			},
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.FormatAmount(ctx, tt.amountCents, tt.currencyCode)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestCurrencyService_ParseAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueries := mocks.NewMockQuerier(ctrl)
	service := services.NewCurrencyService(mockQueries)
	ctx := context.Background()

	tests := []struct {
		name           string
		amountStr      string
		currencyCode   string
		setupMock      func()
		expectedResult int64
		expectError    bool
	}{
		{
			name:         "parse USD amount with symbol",
			amountStr:    "$123.45",
			currencyCode: "USD",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "USD").Return(db.FiatCurrency{
					ID:            uuid.New(),
					Code:          "USD",
					Symbol:        "$",
					DecimalPlaces: 2,
					ThousandSeparator: pgtype.Text{String: ",", Valid: true},
					DecimalSeparator:  pgtype.Text{String: ".", Valid: true},
				}, nil)
			},
			expectedResult: 12345,
			expectError:    false,
		},
		{
			name:         "parse invalid amount format",
			amountStr:    "invalid",
			currencyCode: "USD",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "USD").Return(db.FiatCurrency{
					ID:            uuid.New(),
					Code:          "USD",
					Symbol:        "$",
					DecimalPlaces: 2,
					ThousandSeparator: pgtype.Text{String: ",", Valid: true},
					DecimalSeparator:  pgtype.Text{String: ".", Valid: true},
				}, nil)
			},
			expectedResult: 0,
			expectError:    true,
		},
		{
			name:         "database error",
			amountStr:    "$123.45",
			currencyCode: "INVALID",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "INVALID").Return(db.FiatCurrency{}, assert.AnError)
			},
			expectedResult: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			result, err := service.ParseAmount(ctx, tt.amountStr, tt.currencyCode)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestCurrencyService_ValidateCurrencyForWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueries := mocks.NewMockQuerier(ctrl)
	service := services.NewCurrencyService(mockQueries)
	ctx := context.Background()
	workspaceID := uuid.New()

	tests := []struct {
		name         string
		workspaceID  uuid.UUID
		currencyCode string
		setupMock    func()
		expectError  bool
	}{
		{
			name:         "valid currency for workspace",
			workspaceID:  workspaceID,
			currencyCode: "USD",
			setupMock: func() {
				supportedCurrencies := []string{"USD", "EUR", "GBP"}
				supportedJSON, _ := json.Marshal(supportedCurrencies)
				mockQueries.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{
					ID:                  workspaceID,
					SupportedCurrencies: supportedJSON,
				}, nil)
			},
			expectError: false,
		},
		{
			name:         "invalid currency for workspace",
			workspaceID:  workspaceID,
			currencyCode: "JPY",
			setupMock: func() {
				supportedCurrencies := []string{"USD", "EUR", "GBP"}
				supportedJSON, _ := json.Marshal(supportedCurrencies)
				mockQueries.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{
					ID:                  workspaceID,
					SupportedCurrencies: supportedJSON,
				}, nil)
			},
			expectError: true,
		},
		{
			name:         "workspace not found",
			workspaceID:  workspaceID,
			currencyCode: "USD",
			setupMock: func() {
				mockQueries.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := service.ValidateCurrencyForWorkspace(ctx, tt.workspaceID, tt.currencyCode)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCurrencyService_ValidateAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQueries := mocks.NewMockQuerier(ctrl)
	service := services.NewCurrencyService(mockQueries)
	ctx := context.Background()

	tests := []struct {
		name         string
		amountCents  int64
		currencyCode string
		setupMock    func()
		expectError  bool
	}{
		{
			name:         "valid amount for active currency",
			amountCents:  12345,
			currencyCode: "USD",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "USD").Return(db.FiatCurrency{
					ID:       uuid.New(),
					Code:     "USD",
					IsActive: pgtype.Bool{Bool: true, Valid: true},
				}, nil)
			},
			expectError: false,
		},
		{
			name:         "negative amount",
			amountCents:  -100,
			currencyCode: "USD",
			setupMock:    func() {},
			expectError:  true,
		},
		{
			name:         "inactive currency",
			amountCents:  12345,
			currencyCode: "USD",
			setupMock: func() {
				mockQueries.EXPECT().GetFiatCurrency(ctx, "USD").Return(db.FiatCurrency{
					ID:       uuid.New(),
					Code:     "USD",
					IsActive: pgtype.Bool{Bool: false, Valid: true},
				}, nil)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := service.ValidateAmount(ctx, tt.amountCents, tt.currencyCode)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCurrencyService_FormatCurrencyAmount(t *testing.T) {
	service := services.NewCurrencyService(nil)

	tests := []struct {
		name           string
		amountCents    int64
		currency       *db.FiatCurrency
		expectedResult string
	}{
		{
			name:        "USD with before symbol position",
			amountCents: 12345,
			currency: &db.FiatCurrency{
				Symbol:        "$",
				DecimalPlaces: 2,
				SymbolPosition: pgtype.Text{String: "before", Valid: true},
				ThousandSeparator: pgtype.Text{String: ",", Valid: true},
				DecimalSeparator:  pgtype.Text{String: ".", Valid: true},
			},
			expectedResult: "$123.45",
		},
		{
			name:        "EUR with after symbol position",
			amountCents: 98765,
			currency: &db.FiatCurrency{
				Symbol:        "€",
				DecimalPlaces: 2,
				SymbolPosition: pgtype.Text{String: "after", Valid: true},
				ThousandSeparator: pgtype.Text{String: ",", Valid: true},
				DecimalSeparator:  pgtype.Text{String: ".", Valid: true},
			},
			expectedResult: "987.65€",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.FormatCurrencyAmount(tt.amountCents, tt.currency)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}