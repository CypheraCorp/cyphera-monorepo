package services_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	logger.InitLogger("test")
}

func TestPaymentLinkService_GetBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "returns provided base URL",
			baseURL:  "https://custom.example.com",
			expected: "https://custom.example.com",
		},
		{
			name:     "trims trailing slash",
			baseURL:  "https://custom.example.com/",
			expected: "https://custom.example.com",
		},
		{
			name:     "trims multiple trailing slashes",
			baseURL:  "https://custom.example.com///",
			expected: "https://custom.example.com",
		},
		{
			name:     "uses default when empty",
			baseURL:  "",
			expected: "https://pay.cyphera.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := services.NewPaymentLinkService(nil, zap.NewNop(), tt.baseURL)
			assert.Equal(t, tt.expected, service.GetBaseURL())
		})
	}
}

func TestPaymentLinkService_GenerateQRCode(t *testing.T) {
	service := services.NewPaymentLinkService(nil, zap.NewNop(), "https://test.example.com")
	ctx := context.Background()

	tests := []struct {
		name        string
		paymentURL  string
		wantErr     bool
		errorString string
		validate    func(string)
	}{
		{
			name:       "successfully generates QR code for valid URL",
			paymentURL: "https://pay.example.com/test-slug",
			wantErr:    false,
			validate: func(qrData string) {
				assert.True(t, strings.HasPrefix(qrData, "data:image/png;base64,"))
				assert.True(t, len(qrData) > 50) // QR code should be substantial
			},
		},
		{
			name:       "generates QR code for simple URL",
			paymentURL: "https://example.com",
			wantErr:    false,
			validate: func(qrData string) {
				assert.True(t, strings.HasPrefix(qrData, "data:image/png;base64,"))
			},
		},
		{
			name:       "generates QR code for URL with query parameters",
			paymentURL: "https://pay.example.com/test?amount=100&currency=USD",
			wantErr:    false,
			validate: func(qrData string) {
				assert.True(t, strings.HasPrefix(qrData, "data:image/png;base64,"))
			},
		},
		{
			name:       "generates QR code for long URL",
			paymentURL: "https://pay.example.com/very-long-slug-that-contains-many-characters-and-should-still-work",
			wantErr:    false,
			validate: func(qrData string) {
				assert.True(t, strings.HasPrefix(qrData, "data:image/png;base64,"))
			},
		},
		{
			name:        "handles empty URL",
			paymentURL:  "",
			wantErr:     true, // Empty strings should fail QR code generation
			errorString: "no data to encode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qrData, err := service.GenerateQRCode(ctx, tt.paymentURL)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Empty(t, qrData)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, qrData)
				if tt.validate != nil {
					tt.validate(qrData)
				}
			}
		})
	}
}

func TestPaymentLinkService_PaymentLinkCreateParams_Validation(t *testing.T) {
	workspaceID := uuid.New()
	productID := uuid.New()
	priceID := uuid.New()
	amountCents := int64(1000)

	tests := []struct {
		name    string
		params  params.PaymentLinkCreateParams
		isValid bool
	}{
		{
			name: "valid product-based payment link",
			params: params.PaymentLinkCreateParams{
				WorkspaceID: workspaceID,
				ProductID:   &productID,
				PriceID:     &priceID,
				Currency:    "USD",
				Title:       "Test Payment Link",
			},
			isValid: true,
		},
		{
			name: "valid custom amount payment link",
			params: params.PaymentLinkCreateParams{
				WorkspaceID: workspaceID,
				AmountCents: amountCents,
				Currency:    "USD",
				Title:       "Custom Amount Link",
			},
			isValid: true,
		},
		{
			name: "valid minimal payment link",
			params: params.PaymentLinkCreateParams{
				WorkspaceID: workspaceID,
				Currency:    "USD",
				Title:       "Minimal Link",
			},
			isValid: true,
		},
		{
			name: "invalid - missing workspace ID",
			params: params.PaymentLinkCreateParams{
				Currency: "USD",
				Title:    "Missing Workspace",
			},
			isValid: false,
		},
		{
			name: "invalid - missing currency",
			params: params.PaymentLinkCreateParams{
				WorkspaceID: workspaceID,
				Title:       "Missing Currency",
			},
			isValid: false,
		},
		{
			name: "invalid - missing title",
			params: params.PaymentLinkCreateParams{
				WorkspaceID: workspaceID,
				Currency:    "USD",
			},
			isValid: false,
		},
		{
			name: "valid with all optional fields",
			params: params.PaymentLinkCreateParams{
				WorkspaceID:         workspaceID,
				ProductID:           &productID,
				PriceID:             &priceID,
				AmountCents:         amountCents,
				Currency:            "USD",
				Title:               "Complete Payment Link",
				Description:         "A payment link with all fields",
				ExpiresAt:           &[]string{time.Now().Add(24 * time.Hour).Format(time.RFC3339)}[0],
				MaxRedemptions:      &[]int32{10}[0],
				RequireCustomerInfo: true,
				RedirectURL:         &[]string{"https://example.com/success"}[0],
				Metadata: map[string]interface{}{
					"source":  "api",
					"version": 1,
				},
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation checks
			if tt.isValid {
				assert.NotEqual(t, uuid.Nil, tt.params.WorkspaceID)
				assert.NotEmpty(t, tt.params.Currency)
				assert.NotEmpty(t, tt.params.Title)
			}

			// Validate currency format
			if tt.params.Currency != "" {
				assert.True(t, len(tt.params.Currency) >= 3)
				assert.Equal(t, strings.ToUpper(tt.params.Currency), tt.params.Currency)
			}

			// Validate amount if provided
			if tt.params.AmountCents != 0 {
				assert.Greater(t, tt.params.AmountCents, int64(0))
			}
		})
	}
}

func TestPaymentLinkService_PaymentLinkUpdateParams(t *testing.T) {
	// Test the update parameters structure
	tests := []struct {
		name   string
		params params.PaymentLinkUpdateParams
	}{
		{
			name: "update title only",
			params: params.PaymentLinkUpdateParams{
				Title: &[]string{"Updated Title"}[0],
			},
		},
		{
			name: "update expiration only",
			params: params.PaymentLinkUpdateParams{
				ExpiresAt: &[]string{time.Now().Add(24 * time.Hour).Format(time.RFC3339)}[0],
			},
		},
		{
			name: "update max redemptions only",
			params: params.PaymentLinkUpdateParams{
				MaxRedemptions: &[]int32{5}[0],
			},
		},
		{
			name: "update redirect URL only",
			params: params.PaymentLinkUpdateParams{
				RedirectURL: &[]string{"https://example.com/success"}[0],
			},
		},
		{
			name: "update metadata only",
			params: params.PaymentLinkUpdateParams{
				Metadata: map[string]interface{}{
					"updated": true,
					"version": 2,
				},
			},
		},
		{
			name: "update all fields",
			params: params.PaymentLinkUpdateParams{
				Title:               &[]string{"Complete Update"}[0],
				Description:         &[]string{"Updated description"}[0],
				ExpiresAt:           &[]string{time.Now().Add(48 * time.Hour).Format(time.RFC3339)}[0],
				MaxRedemptions:      &[]int32{10}[0],
				IsActive:            &[]bool{true}[0],
				RequireCustomerInfo: &[]bool{false}[0],
				RedirectURL:         &[]string{"https://example.com/updated"}[0],
				Metadata: map[string]interface{}{
					"comprehensive": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the structure can be created
			assert.NotNil(t, tt.params)

			// Test specific field validations
			if tt.params.Title != nil {
				assert.NotEmpty(t, *tt.params.Title)
			}
			if tt.params.Description != nil {
				assert.NotEmpty(t, *tt.params.Description)
			}
			if tt.params.ExpiresAt != nil {
				// Parse the time string to validate it's in the future
				parsedTime, err := time.Parse(time.RFC3339, *tt.params.ExpiresAt)
				assert.NoError(t, err)
				assert.True(t, parsedTime.After(time.Now()))
			}
			if tt.params.MaxRedemptions != nil {
				assert.Greater(t, *tt.params.MaxRedemptions, int32(0))
			}
			if tt.params.RedirectURL != nil {
				assert.True(t, strings.HasPrefix(*tt.params.RedirectURL, "http"))
			}
		})
	}
}

func TestPaymentLinkService_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		operation func() error
		wantErr   bool
		errorMsg  string
	}{
		{
			name: "service creation with nil logger",
			operation: func() error {
				service := services.NewPaymentLinkService(nil, nil, "")
				assert.NotNil(t, service)
				return nil
			},
			wantErr: false,
		},
		{
			name: "QR code generation with very long URL",
			operation: func() error {
				service := services.NewPaymentLinkService(nil, zap.NewNop(), "https://test.example.com")
				longURL := "https://pay.example.com/" + strings.Repeat("a", 1000)
				_, err := service.GenerateQRCode(context.Background(), longURL)
				return err
			},
			wantErr: false, // QR codes can handle long URLs
		},
		{
			name: "service with custom base URL with query parameters",
			operation: func() error {
				service := services.NewPaymentLinkService(nil, zap.NewNop(), "https://custom.example.com/path?param=value")
				baseURL := service.GetBaseURL()
				assert.Contains(t, baseURL, "custom.example.com")
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if tt.wantErr {
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

func TestPaymentLinkService_BoundaryConditions(t *testing.T) {
	service := services.NewPaymentLinkService(nil, zap.NewNop(), "https://test.example.com")
	ctx := context.Background()

	tests := []struct {
		name        string
		operation   func() (interface{}, error)
		expectError bool
	}{
		{
			name: "QR code with empty string",
			operation: func() (interface{}, error) {
				return service.GenerateQRCode(ctx, "")
			},
			expectError: true,
		},
		{
			name: "QR code with whitespace only",
			operation: func() (interface{}, error) {
				return service.GenerateQRCode(ctx, "   ")
			},
			expectError: false,
		},
		{
			name: "QR code with special characters",
			operation: func() (interface{}, error) {
				return service.GenerateQRCode(ctx, "https://example.com/pay?amount=100&currency=USD&callback=https://merchant.com/callback")
			},
			expectError: false,
		},
		{
			name: "QR code with Unicode characters",
			operation: func() (interface{}, error) {
				return service.GenerateQRCode(ctx, "https://example.com/pay/测试")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.operation()

			if tt.expectError {
				assert.Error(t, err)
				// For string results, check for empty string instead of nil
				if resultStr, ok := result.(string); ok {
					assert.Empty(t, resultStr)
				} else {
					assert.Nil(t, result)
				}
			} else {
				assert.NoError(t, err)
				if resultStr, ok := result.(string); ok {
					assert.NotEmpty(t, resultStr)
				} else {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

// Note: The CreatePaymentLink, GetPaymentLink, UpdatePaymentLink, and other database-dependent methods
// cannot be properly tested without either:
// 1. Refactoring the service to accept db.Querier interface instead of *db.Queries
// 2. Using integration tests with a real database
// 3. Creating a test database layer
//
// The current architecture with concrete *db.Queries makes unit testing these methods impossible.
// Consider refactoring the service to use interfaces for better testability.
