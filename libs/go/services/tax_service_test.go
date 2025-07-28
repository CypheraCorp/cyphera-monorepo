package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestTaxService_CalculateTax(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTaxService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	productID := uuid.New()

	tests := []struct {
		name                 string
		params               params.TaxCalculationParams
		setupMocks           func()
		wantErr              bool
		errorString          string
		expectedTaxCents     int64
		expectedTotalCents   int64
		expectedJurisdiction string
		expectedTaxType      string
		expectedExempt       bool
	}{
		{
			name: "tax exempt customer",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       &productID,
				AmountCents:     10000, // $100.00
				Currency:        "USD",
				TransactionType: "subscription",
				ProductType:     "digital",
				TaxExempt:       true,
			},
			setupMocks:         func() {},
			wantErr:            false,
			expectedTaxCents:   0,
			expectedTotalCents: 10000,
			expectedExempt:     true,
		},
		{
			name: "US California sales tax calculation",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       &productID,
				AmountCents:     10000, // $100.00
				Currency:        "USD",
				TransactionType: "one_time",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Street1:    "123 Main St",
					City:       "San Francisco",
					State:      "CA",
					PostalCode: "94102",
					Country:    "US",
				},
				TaxExempt: false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     725, // 7.25% of $100
			expectedTotalCents:   10725,
			expectedJurisdiction: "US-CA",
			expectedTaxType:      "sales",
		},
		{
			name: "UK VAT calculation for digital service",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       &productID,
				AmountCents:     10000, // $100.00
				Currency:        "GBP",
				TransactionType: "subscription",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Street1:    "10 Downing Street",
					City:       "London",
					PostalCode: "SW1A 2AA",
					Country:    "GB",
				},
				TaxExempt: false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     2000, // 20% VAT
			expectedTotalCents:   12000,
			expectedJurisdiction: "UK",
			expectedTaxType:      "vat",
		},
		{
			name: "EU B2B reverse charge scenario",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       &productID,
				AmountCents:     10000,
				Currency:        "EUR",
				TransactionType: "subscription",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Street1:    "Unter den Linden 1",
					City:       "Berlin",
					PostalCode: "10117",
					Country:    "DE",
				},
				IsB2B:             true,
				CustomerVATNumber: taxStringPtr("DE123456789"),
				TaxExempt:         false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     0, // Reverse charge
			expectedTotalCents:   10000,
			expectedJurisdiction: "EU-", // Will be partial match
			expectedTaxType:      "vat",
		},
		{
			name: "Canadian GST/PST calculation",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				ProductID:       &productID,
				AmountCents:     10000,
				Currency:        "CAD",
				TransactionType: "one_time",
				ProductType:     "service",
				CustomerAddress: &business.Address{
					Street1:    "123 Main St",
					City:       "Toronto",
					State:      "ON",
					PostalCode: "M5V 3A8",
					Country:    "CA",
				},
				TaxExempt: false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     1300, // 13% HST
			expectedTotalCents:   11300,
			expectedJurisdiction: "CA-ON",
			expectedTaxType:      "gst",
		},
		{
			name: "no customer address uses workspace default",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     5000,
				Currency:        "USD",
				TransactionType: "subscription",
				ProductType:     "digital",
				TaxExempt:       false,
			},
			setupMocks: func() {
				workspace := db.Workspace{
					ID:   workspaceID,
					Name: "Test Workspace",
				}
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(workspace, nil)
			},
			wantErr:              false,
			expectedTaxCents:     0, // Default US jurisdiction has no federal tax
			expectedTotalCents:   5000,
			expectedJurisdiction: "US",
			expectedTaxType:      "sales",
		},
		{
			name: "workspace lookup fails",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     5000,
				Currency:        "USD",
				TransactionType: "subscription",
				ProductType:     "digital",
				TaxExempt:       false,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, errors.New("workspace not found"))
			},
			wantErr:     true,
			errorString: "failed to determine jurisdiction",
		},
		{
			name: "unknown jurisdiction defaults to no tax",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     10000,
				Currency:        "USD",
				TransactionType: "one_time",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Street1: "123 Test St",
					City:    "Test City",
					Country: "XX", // Unknown country
				},
				TaxExempt: false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     0,
			expectedTotalCents:   10000,
			expectedJurisdiction: "XX",
		},
		{
			name: "zero amount calculation",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     0,
				Currency:        "USD",
				TransactionType: "one_time",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Country: "US",
					State:   "CA",
				},
				TaxExempt: false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     0,
			expectedTotalCents:   0,
			expectedJurisdiction: "US-CA",
			expectedTaxType:      "sales",
		},
		{
			name: "large amount calculation",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     100000000, // $1,000,000
				Currency:        "USD",
				TransactionType: "one_time",
				ProductType:     "physical",
				CustomerAddress: &business.Address{
					Country: "US",
					State:   "NY",
				},
				TaxExempt: false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     8000000, // 8% of $1M
			expectedTotalCents:   108000000,
			expectedJurisdiction: "US-NY",
			expectedTaxType:      "sales",
		},
		{
			name: "B2B without VAT number - standard tax",
			params: params.TaxCalculationParams{
				WorkspaceID:     workspaceID,
				CustomerID:      customerID,
				AmountCents:     10000,
				Currency:        "EUR",
				TransactionType: "one_time",
				ProductType:     "service",
				CustomerAddress: &business.Address{
					Country: "DE",
					State:   "BY",
				},
				IsB2B:     true,
				TaxExempt: false,
			},
			setupMocks:           func() {},
			wantErr:              false,
			expectedTaxCents:     1900, // 19% VAT on 10000 cents
			expectedTotalCents:   11900,
			expectedJurisdiction: "EU-DE", // Updated to match new jurisdiction format
			expectedTaxType:      "vat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.CalculateTax(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Verify basic calculation results
				assert.Equal(t, tt.params.AmountCents, result.SubtotalCents)
				assert.Equal(t, tt.expectedTaxCents, result.TotalTaxCents)
				assert.Equal(t, tt.expectedTotalCents, result.TotalAmountCents)

				// Verify audit trail
				assert.NotEmpty(t, result.AuditTrail.RulesVersion)
				assert.NotZero(t, result.CalculatedAt)
				assert.GreaterOrEqual(t, result.Confidence, 0.0)
				assert.LessOrEqual(t, result.Confidence, 1.0)

				// Verify tax exemption
				if tt.expectedExempt {
					assert.NotNil(t, result.TaxExemptReason)
					assert.Contains(t, *result.TaxExemptReason, "tax exempt")
				} else {
					assert.Nil(t, result.TaxExemptReason)
				}

				// Verify jurisdiction
				if tt.expectedJurisdiction != "" {
					assert.NotEmpty(t, result.AppliedJurisdictions)
					if len(tt.expectedJurisdiction) > 3 {
						assert.Equal(t, tt.expectedJurisdiction, result.AppliedJurisdictions[0])
					} else {
						// Partial match for cases like "EU-"
						assert.Contains(t, result.AppliedJurisdictions[0], tt.expectedJurisdiction)
					}
				}

				// Verify tax breakdown for non-exempt calculations
				if !tt.expectedExempt && tt.expectedTaxCents > 0 {
					assert.NotEmpty(t, result.TaxBreakdown)
					taxItem := result.TaxBreakdown[0]
					if tt.expectedTaxType != "" {
						assert.Equal(t, tt.expectedTaxType, taxItem.TaxType)
					}
					assert.Equal(t, tt.expectedTaxCents, taxItem.TaxAmountCents)
					assert.Equal(t, tt.params.AmountCents, taxItem.TaxableAmount)
				}
			}
		})
	}
}

func TestTaxService_StoreTaxCalculation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTaxService(mockQuerier)
	ctx := context.Background()

	paymentID := uuid.New()
	calculation := &responses.TaxCalculationResult{
		SubtotalCents:    10000,
		TotalTaxCents:    800,
		TotalAmountCents: 10800,
		TaxBreakdown: []business.TaxLineItem{
			{
				TaxType:        "sales",
				Jurisdiction:   "US-CA",
				Rate:           0.08,
				TaxableAmount:  10000,
				TaxAmountCents: 800,
				Description:    "California Sales Tax",
			},
		},
		AppliedJurisdictions: []string{"US-CA"},
		CalculatedAt:         time.Now(),
		Confidence:           1.0,
		AuditTrail: business.TaxAuditTrail{
			RulesVersion: "v2024.1",
			AppliedRules: []string{"STANDARD_TAX_US-CA"},
		},
	}

	tests := []struct {
		name        string
		paymentID   uuid.UUID
		calculation *responses.TaxCalculationResult
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully stores tax calculation (placeholder)",
			paymentID:   paymentID,
			calculation: calculation,
			setupMocks:  func() {}, // No database calls in current implementation
			wantErr:     false,
		},
		{
			name:        "handles nil calculation",
			paymentID:   paymentID,
			calculation: nil,
			setupMocks:  func() {},
			wantErr:     false, // Current implementation doesn't validate
		},
		{
			name:      "handles calculation with empty breakdown",
			paymentID: paymentID,
			calculation: &responses.TaxCalculationResult{
				SubtotalCents:    5000,
				TotalTaxCents:    0,
				TotalAmountCents: 5000,
				TaxBreakdown:     []business.TaxLineItem{},
				CalculatedAt:     time.Now(),
			},
			setupMocks: func() {},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.StoreTaxCalculation(ctx, tt.paymentID, tt.calculation)

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

func TestTaxService_GetTaxRatesForJurisdiction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTaxService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name             string
		jurisdictionCode string
		setupMocks       func()
		wantErr          bool
		expectedTaxRates map[string]float64
	}{
		{
			name:             "US state jurisdiction",
			jurisdictionCode: "US-CA",
			setupMocks:       func() {},
			wantErr:          false,
			expectedTaxRates: map[string]float64{
				"digital":  0.0725,
				"physical": 0.0725,
				"service":  0.0725,
				"default":  0.0725,
			},
		},
		{
			name:             "unknown jurisdiction returns default",
			jurisdictionCode: "XX-UNKNOWN",
			setupMocks:       func() {},
			wantErr:          false,
			expectedTaxRates: map[string]float64{
				"default": 0.0,
			},
		},
		{
			name:             "empty jurisdiction code",
			jurisdictionCode: "",
			setupMocks:       func() {},
			wantErr:          false,
			expectedTaxRates: map[string]float64{
				"default": 0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			jurisdiction, err := service.GetTaxRatesForJurisdiction(ctx, tt.jurisdictionCode)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, jurisdiction)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, jurisdiction)

				// Verify tax rates
				for productType, expectedRate := range tt.expectedTaxRates {
					actualRate, exists := jurisdiction.TaxRates[productType]
					assert.True(t, exists, "Tax rate for %s should exist", productType)
					assert.Equal(t, expectedRate, actualRate, "Tax rate for %s should match", productType)
				}

				// Verify jurisdiction properties
				assert.NotEmpty(t, jurisdiction.Code)
				assert.NotEmpty(t, jurisdiction.Name)
				assert.NotEmpty(t, jurisdiction.Type)
				assert.True(t, jurisdiction.IsActive)
				assert.NotZero(t, jurisdiction.EffectiveDate)
			}
		})
	}
}

func TestTaxService_ValidateVATNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTaxService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name          string
		vatNumber     string
		countryCode   string
		setupMocks    func()
		wantErr       bool
		expectedValid bool
	}{
		{
			name:          "valid EU VAT number",
			vatNumber:     "DE123456789",
			countryCode:   "DE",
			setupMocks:    func() {},
			wantErr:       false,
			expectedValid: true,
		},
		{
			name:          "valid long EU VAT number",
			vatNumber:     "FR1234567890", // 12 characters - within validation range
			countryCode:   "FR",
			setupMocks:    func() {},
			wantErr:       false,
			expectedValid: true,
		},
		{
			name:          "too short VAT number",
			vatNumber:     "DE123",
			countryCode:   "DE",
			setupMocks:    func() {},
			wantErr:       false,
			expectedValid: false,
		},
		{
			name:          "too long VAT number",
			vatNumber:     "DE1234567890123",
			countryCode:   "DE",
			setupMocks:    func() {},
			wantErr:       false,
			expectedValid: false,
		},
		{
			name:          "empty VAT number",
			vatNumber:     "",
			countryCode:   "DE",
			setupMocks:    func() {},
			wantErr:       false,
			expectedValid: false,
		},
		{
			name:          "minimum length VAT number",
			vatNumber:     "12345678",
			countryCode:   "XX",
			setupMocks:    func() {},
			wantErr:       false,
			expectedValid: true,
		},
		{
			name:          "maximum length VAT number",
			vatNumber:     "123456789012",
			countryCode:   "XX",
			setupMocks:    func() {},
			wantErr:       false,
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			isValid, err := service.ValidateVATNumber(ctx, tt.vatNumber, tt.countryCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValid, isValid)
			}
		})
	}
}

func TestTaxService_JurisdictionHelpers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTaxService(mockQuerier)

	t.Run("getUSStateTaxRates returns correct rates", func(t *testing.T) {
		// Test known states
		testCases := []struct {
			state        string
			expectedRate float64
		}{
			{"CA", 0.0725},
			{"NY", 0.08},
			{"TX", 0.0625},
			{"FL", 0.06},
			{"UNKNOWN", 0.0},
		}

		for _, tc := range testCases {
			// This would require exposing the helper method or testing indirectly
			// For now, we test through the main CalculateTax function
			params := params.TaxCalculationParams{
				WorkspaceID: uuid.New(),
				CustomerID:  uuid.New(),
				AmountCents: 10000,
				Currency:    "USD",
				CustomerAddress: &business.Address{
					Country: "US",
					State:   tc.state,
				},
				TransactionType: "one_time",
				ProductType:     "digital",
			}

			result, err := service.CalculateTax(context.Background(), params)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			expectedTax := int64(float64(10000) * tc.expectedRate)
			assert.Equal(t, expectedTax, result.TotalTaxCents,
				"Tax for state %s should be %d cents", tc.state, expectedTax)
		}
	})

	t.Run("getCanadaTaxRates returns correct rates", func(t *testing.T) {
		testCases := []struct {
			province     string
			expectedRate float64
		}{
			{"ON", 0.13},      // HST
			{"BC", 0.12},      // GST + PST
			{"AB", 0.05},      // GST only
			{"QC", 0.14975},   // GST + QST
			{"UNKNOWN", 0.05}, // Default GST
		}

		for _, tc := range testCases {
			params := params.TaxCalculationParams{
				WorkspaceID: uuid.New(),
				CustomerID:  uuid.New(),
				AmountCents: 10000,
				Currency:    "CAD",
				CustomerAddress: &business.Address{
					Country: "CA",
					State:   tc.province,
				},
				TransactionType: "one_time",
				ProductType:     "digital",
			}

			result, err := service.CalculateTax(context.Background(), params)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			expectedTax := int64(float64(10000) * tc.expectedRate)
			assert.Equal(t, expectedTax, result.TotalTaxCents,
				"Tax for province %s should be %d cents", tc.province, expectedTax)
		}
	})
}

func TestTaxService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTaxService(mockQuerier)
	ctx := context.Background()

	t.Run("nil service handles gracefully", func(t *testing.T) {
		// Service should not panic with nil querier
		nilService := services.NewTaxService(nil)
		assert.NotNil(t, nilService)
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		params := params.TaxCalculationParams{
			WorkspaceID: uuid.New(),
			CustomerID:  uuid.New(),
			AmountCents: 10000,
			Currency:    "USD",
			CustomerAddress: &business.Address{
				Country: "US",
				State:   "CA",
			},
			TransactionType: "one_time",
			ProductType:     "digital",
		}

		// Should handle cancelled context gracefully
		result, err := service.CalculateTax(cancelCtx, params)
		// Current implementation doesn't check context, so it should succeed
		// In a real implementation with external API calls, this might fail
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("negative amount calculation", func(t *testing.T) {
		params := params.TaxCalculationParams{
			WorkspaceID:     uuid.New(),
			CustomerID:      uuid.New(),
			AmountCents:     -10000, // Negative amount (refund)
			Currency:        "USD",
			TransactionType: "refund",
			ProductType:     "digital",
			CustomerAddress: &business.Address{
				Country: "US",
				State:   "CA",
			},
		}

		result, err := service.CalculateTax(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Tax should be negative for refunds
		assert.Equal(t, int64(-10000), result.SubtotalCents)
		assert.Less(t, result.TotalTaxCents, int64(0))
		assert.Less(t, result.TotalAmountCents, int64(-10000))
	})

	t.Run("very large amounts", func(t *testing.T) {
		params := params.TaxCalculationParams{
			WorkspaceID:     uuid.New(),
			CustomerID:      uuid.New(),
			AmountCents:     9223372036854775807, // Max int64
			Currency:        "USD",
			TransactionType: "one_time",
			ProductType:     "digital",
			CustomerAddress: &business.Address{
				Country: "US",
				State:   "CA",
			},
		}

		result, err := service.CalculateTax(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		// Should handle large numbers without overflow (no negative values)
		assert.GreaterOrEqual(t, result.TotalAmountCents, int64(0))
		assert.GreaterOrEqual(t, result.TotalTaxCents, int64(0))
		assert.Equal(t, params.AmountCents, result.SubtotalCents)
		// For max int64 values, total might be capped to prevent overflow
		assert.GreaterOrEqual(t, result.TotalAmountCents, result.SubtotalCents)
	})

	t.Run("nil address pointers", func(t *testing.T) {
		mockQuerier.EXPECT().GetWorkspace(ctx, gomock.Any()).Return(db.Workspace{}, nil)

		params := params.TaxCalculationParams{
			WorkspaceID:     uuid.New(),
			CustomerID:      uuid.New(),
			AmountCents:     10000,
			Currency:        "USD",
			TransactionType: "one_time",
			ProductType:     "digital",
			CustomerAddress: nil,
			BusinessAddress: nil,
		}

		result, err := service.CalculateTax(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("empty string fields", func(t *testing.T) {
		params := params.TaxCalculationParams{
			WorkspaceID:     uuid.New(),
			CustomerID:      uuid.New(),
			AmountCents:     10000,
			Currency:        "", // Empty currency
			TransactionType: "", // Empty transaction type
			ProductType:     "", // Empty product type
			CustomerAddress: &business.Address{
				Street1:    "",
				City:       "",
				State:      "",
				PostalCode: "",
				Country:    "",
			},
		}

		result, err := service.CalculateTax(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestTaxService_ReverseChargeScenarios(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewTaxService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name                  string
		params                params.TaxCalculationParams
		setupMocks            func()
		expectedReverseCharge bool
		expectedTaxCents      int64
	}{
		{
			name: "EU B2B with valid VAT - reverse charge applies",
			params: params.TaxCalculationParams{
				WorkspaceID:     uuid.New(),
				CustomerID:      uuid.New(),
				AmountCents:     10000,
				Currency:        "EUR",
				TransactionType: "subscription",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Country: "DE",
				},
				IsB2B:             true,
				CustomerVATNumber: taxStringPtr("DE123456789"),
			},
			setupMocks:            func() {},
			expectedReverseCharge: true,
			expectedTaxCents:      0,
		},
		{
			name: "EU B2B without VAT number - standard tax",
			params: params.TaxCalculationParams{
				WorkspaceID:     uuid.New(),
				CustomerID:      uuid.New(),
				AmountCents:     10000,
				Currency:        "EUR",
				TransactionType: "subscription",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Country: "DE",
				},
				IsB2B: true,
				// No VAT number
			},
			setupMocks:            func() {},
			expectedReverseCharge: false,
		},
		{
			name: "EU B2C - standard tax applies",
			params: params.TaxCalculationParams{
				WorkspaceID:     uuid.New(),
				CustomerID:      uuid.New(),
				AmountCents:     10000,
				Currency:        "EUR",
				TransactionType: "subscription",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Country: "DE",
				},
				IsB2B: false,
			},
			setupMocks:            func() {},
			expectedReverseCharge: false,
		},
		{
			name: "Non-EU B2B - no reverse charge",
			params: params.TaxCalculationParams{
				WorkspaceID:     uuid.New(),
				CustomerID:      uuid.New(),
				AmountCents:     10000,
				Currency:        "USD",
				TransactionType: "subscription",
				ProductType:     "digital",
				CustomerAddress: &business.Address{
					Country: "US",
					State:   "CA",
				},
				IsB2B:             true,
				CustomerVATNumber: taxStringPtr("US123456789"),
			},
			setupMocks:            func() {},
			expectedReverseCharge: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.CalculateTax(ctx, tt.params)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.expectedReverseCharge {
				assert.Equal(t, tt.expectedTaxCents, result.TotalTaxCents)
				assert.NotEmpty(t, result.TaxBreakdown)
				assert.True(t, result.TaxBreakdown[0].IsReversCharge)
				assert.Contains(t, result.AuditTrail.AppliedRules, "B2B_REVERSE_CHARGE")
			} else {
				// Should have normal tax calculation
				if !tt.params.TaxExempt {
					assert.NotEmpty(t, result.TaxBreakdown)
					if len(result.TaxBreakdown) > 0 {
						assert.False(t, result.TaxBreakdown[0].IsReversCharge)
					}
				}
			}
		})
	}
}

// Helper functions
func taxStringPtr(s string) *string {
	return &s
}
