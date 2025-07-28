package services_test

import (
	"context"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestExchangeRateService_GetExchangeRate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewExchangeRateService(mockQuerier, "test-api-key")
	ctx := context.Background()

	tokenID := uuid.New()
	networkID := uuid.New()

	tests := []struct {
		name       string
		params     params.ExchangeRateParams
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name: "successful rate from database fallback",
			params: params.ExchangeRateParams{
				FromSymbol: "ETH",
				ToSymbol:   "USD",
				TokenID:    &tokenID,
				NetworkID:  &networkID,
			},
			setupMocks: func() {
				// Clear cache to force database fallback
				service.ClearCache()
			},
			wantErr: false,
		},
		{
			name: "successful rate from database for BTC",
			params: params.ExchangeRateParams{
				FromSymbol: "BTC",
				ToSymbol:   "USD",
			},
			setupMocks: func() {
				service.ClearCache()
			},
			wantErr: false,
		},
		{
			name: "successful rate from database for USDC",
			params: params.ExchangeRateParams{
				FromSymbol: "USDC",
				ToSymbol:   "USD",
			},
			setupMocks: func() {
				service.ClearCache()
			},
			wantErr: false,
		},
		{
			name: "error when no fallback available",
			params: params.ExchangeRateParams{
				FromSymbol: "UNKNOWN",
				ToSymbol:   "USD",
			},
			setupMocks: func() {
				service.ClearCache()
			},
			wantErr:   true,
			errString: "no fallback rate available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			result, err := service.GetExchangeRate(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.params.FromSymbol, result.FromSymbol)
				assert.Equal(t, tt.params.ToSymbol, result.ToSymbol)
				assert.Greater(t, result.Rate, 0.0)
			}
		})
	}
}

func TestExchangeRateService_ConvertAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewExchangeRateService(mockQuerier, "test-api-key")
	ctx := context.Background()

	tests := []struct {
		name       string
		amount     float64
		from       string
		to         string
		setupMocks func()
		wantErr    bool
		errString  string
	}{
		{
			name:       "same currency conversion",
			amount:     100.0,
			from:       "USD",
			to:         "USD",
			setupMocks: func() {},
			wantErr:    false,
		},
		{
			name:   "convert ETH to USD",
			amount: 1.5,
			from:   "ETH",
			to:     "USD",
			setupMocks: func() {
				service.ClearCache()
			},
			wantErr: false,
		},
		{
			name:   "convert BTC to USD",
			amount: 0.5,
			from:   "BTC",
			to:     "USD",
			setupMocks: func() {
				service.ClearCache()
			},
			wantErr: false,
		},
		{
			name:   "error on unknown currency",
			amount: 100.0,
			from:   "UNKNOWN",
			to:     "USD",
			setupMocks: func() {
				service.ClearCache()
			},
			wantErr:   true,
			errString: "failed to get exchange rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			amount, rateResult, err := service.ConvertAmount(ctx, tt.amount, tt.from, tt.to)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Equal(t, float64(0), amount)
				assert.Nil(t, rateResult)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rateResult)
				assert.Equal(t, tt.from, rateResult.FromSymbol)
				assert.Equal(t, tt.to, rateResult.ToSymbol)
				if tt.from == tt.to {
					assert.Equal(t, tt.amount, amount)
					assert.Equal(t, 1.0, rateResult.Rate)
				} else {
					assert.Greater(t, rateResult.Rate, 0.0)
				}
			}
		})
	}
}

func TestExchangeRateService_UtilityMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewExchangeRateService(mockQuerier, "test-api-key")

	t.Run("format and parse decimal operations", func(t *testing.T) {
		// Format tests
		assert.Equal(t, "123.46", service.FormatDecimalString(123.456789, 2))
		assert.Equal(t, "0.123457", service.FormatDecimalString(0.123456789, 6))

		// Parse tests
		parsed, err := service.ParseDecimalString("123.456")
		assert.NoError(t, err)
		assert.Equal(t, 123.456, parsed)

		_, err = service.ParseDecimalString("invalid")
		assert.Error(t, err)
	})

	t.Run("supported tokens and currencies", func(t *testing.T) {
		tokens := service.GetSupportedTokens()
		assert.Contains(t, tokens, "BTC")
		assert.Contains(t, tokens, "ETH")
		assert.Contains(t, tokens, "USDC")

		currencies := service.GetSupportedCurrencies()
		assert.Contains(t, currencies, "USD")
		assert.Contains(t, currencies, "EUR")
		assert.Contains(t, currencies, "GBP")
	})

	t.Run("cache operations", func(t *testing.T) {
		service.ClearCache()
		stats := service.GetCacheStats()
		assert.Equal(t, 0, stats["total_entries"])
		assert.Equal(t, float64(5), stats["cache_ttl_minutes"])
	})
}

func TestExchangeRateService_GetMultipleExchangeRates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewExchangeRateService(mockQuerier, "test-api-key")
	ctx := context.Background()

	tests := []struct {
		name       string
		requests   []params.ExchangeRateParams
		setupMocks func()
		wantCount  int
		wantErr    bool
	}{
		{
			name: "multiple rates with fallback",
			requests: []params.ExchangeRateParams{
				{FromSymbol: "ETH", ToSymbol: "USD"},
				{FromSymbol: "BTC", ToSymbol: "USD"},
				{FromSymbol: "USDC", ToSymbol: "USD"},
			},
			setupMocks: func() {
				service.ClearCache()
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "mixed success and failure",
			requests: []params.ExchangeRateParams{
				{FromSymbol: "ETH", ToSymbol: "USD"},
				{FromSymbol: "UNKNOWN", ToSymbol: "USD"},
			},
			setupMocks: func() {
				service.ClearCache()
			},
			wantCount: 1, // Only ETH should succeed
			wantErr:   false,
		},
		{
			name:       "empty request list",
			requests:   []params.ExchangeRateParams{},
			setupMocks: func() {},
			wantCount:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			results, err := service.GetMultipleExchangeRates(ctx, tt.requests)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
				assert.Equal(t, tt.wantCount, len(results))
			}
		})
	}
}
