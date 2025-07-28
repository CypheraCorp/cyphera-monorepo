package services_test

import (
	"context"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func init() {
	logger.InitLogger("test")
}

func TestInvoiceService_CreateInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)

	// Create mock services
	taxService := services.NewTaxService(mockQuerier)
	discountService := services.NewDiscountService(mockQuerier)
	gasSponsorshipService := services.NewGasSponsorshipService(mockQuerier)
	currencyService := services.NewCurrencyService(mockQuerier)
	exchangeRateService := services.NewExchangeRateService(mockQuerier, "test-api-key")

	service := services.NewInvoiceService(
		mockQuerier,
		zap.NewNop(),
		taxService,
		discountService,
		gasSponsorshipService,
		currencyService,
		exchangeRateService,
	)
	ctx := context.Background()

	workspaceID := uuid.New()
	customerID := uuid.New()
	subscriptionID := uuid.New()
	productID := uuid.New()
	priceID := uuid.New()

	tests := []struct {
		name        string
		params      params.InvoiceCreateParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates invoice with product line items",
			params: params.InvoiceCreateParams{
				WorkspaceID:    workspaceID,
				CustomerID:     customerID,
				SubscriptionID: &subscriptionID,
				Currency:       "USD",
				LineItems: []params.LineItemCreateParams{
					{
						Description:     "Premium Subscription",
						Quantity:        1.0,
						UnitAmountCents: 2000,
						ProductID:       &productID,
						PriceID:         &priceID,
						LineItemType:    "product",
					},
				},
				Metadata: map[string]interface{}{"source": "api"},
			},
			setupMocks: func() {
				// Mock invoice number generation
				mockQuerier.EXPECT().
					GetNextInvoiceNumber(ctx, workspaceID).
					Return(int32(1), nil).
					Times(1)

				// Mock customer retrieval
				mockQuerier.EXPECT().
					GetCustomer(ctx, customerID).
					Return(db.Customer{
						ID:    customerID,
						TaxID: pgtype.Text{String: "TAX123", Valid: true},
					}, nil).
					Times(1)

				// Mock workspace retrieval for tax calculation
				mockQuerier.EXPECT().
					GetWorkspace(ctx, workspaceID).
					Return(db.Workspace{
						ID:   workspaceID,
						Name: "Test Workspace",
					}, nil).
					AnyTimes()

				// Mock invoice creation
				mockQuerier.EXPECT().
					CreateInvoiceWithDetails(ctx, gomock.Any()).
					Return(db.Invoice{
						ID:          uuid.New(),
						WorkspaceID: workspaceID,
						CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
						AmountDue:   2000,
						Currency:    "USD",
						Status:      "draft",
					}, nil).
					Times(1)

				// Mock line item creation for product
				mockQuerier.EXPECT().
					CreateInvoiceLineItem(ctx, gomock.Any()).
					Return(db.InvoiceLineItem{
						ID:                uuid.New(),
						Description:       "Premium Subscription",
						UnitAmountInCents: 2000,
						AmountInCents:     2000,
						LineItemType:      pgtype.Text{String: "product", Valid: true},
					}, nil).
					Times(1)

				// Mock tax line item creation (service automatically creates tax line items)
				mockQuerier.EXPECT().
					CreateInvoiceLineItem(ctx, gomock.Any()).
					Return(db.InvoiceLineItem{
						ID:                uuid.New(),
						Description:       "Tax (United States)",
						UnitAmountInCents: 0,
						AmountInCents:     0,
						LineItemType:      pgtype.Text{String: "tax", Valid: true},
					}, nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "successfully creates invoice with discount",
			params: params.InvoiceCreateParams{
				WorkspaceID:  workspaceID,
				CustomerID:   customerID,
				Currency:     "USD",
				DiscountCode: &[]string{"SAVE10"}[0],
				LineItems: []params.LineItemCreateParams{
					{
						Description:     "Basic Plan",
						Quantity:        1.0,
						UnitAmountCents: 1000,
						LineItemType:    "product",
					},
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetNextInvoiceNumber(ctx, workspaceID).
					Return(int32(2), nil).
					Times(1)

				mockQuerier.EXPECT().
					GetCustomer(ctx, customerID).
					Return(db.Customer{
						ID: customerID,
					}, nil).
					Times(1)

				// Mock workspace retrieval for tax calculation
				mockQuerier.EXPECT().
					GetWorkspace(ctx, workspaceID).
					Return(db.Workspace{
						ID:   workspaceID,
						Name: "Test Workspace",
					}, nil).
					AnyTimes()

				mockQuerier.EXPECT().
					CreateInvoiceWithDetails(ctx, gomock.Any()).
					Return(db.Invoice{
						ID:          uuid.New(),
						WorkspaceID: workspaceID,
						AmountDue:   900,
					}, nil).
					Times(1)

				// Mock line item creation (product + discount + tax)
				mockQuerier.EXPECT().
					CreateInvoiceLineItem(ctx, gomock.Any()).
					Return(db.InvoiceLineItem{
						ID: uuid.New(),
					}, nil).
					AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "fails when invoice number generation fails",
			params: params.InvoiceCreateParams{
				WorkspaceID: workspaceID,
				CustomerID:  customerID,
				Currency:    "USD",
				LineItems: []params.LineItemCreateParams{
					{
						Description:     "Test Product",
						Quantity:        1.0,
						UnitAmountCents: 1000,
						LineItemType:    "product",
					},
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetNextInvoiceNumber(ctx, workspaceID).
					Return(int32(0), assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to generate invoice number",
		},
		{
			name: "fails when customer not found",
			params: params.InvoiceCreateParams{
				WorkspaceID: workspaceID,
				CustomerID:  customerID,
				Currency:    "USD",
				LineItems: []params.LineItemCreateParams{
					{
						Description:     "Test Product",
						Quantity:        1.0,
						UnitAmountCents: 1000,
						LineItemType:    "product",
					},
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetNextInvoiceNumber(ctx, workspaceID).
					Return(int32(1), nil).
					Times(1)

				mockQuerier.EXPECT().
					GetCustomer(ctx, customerID).
					Return(db.Customer{}, assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to get customer",
		},
		{
			name: "fails when database creation fails",
			params: params.InvoiceCreateParams{
				WorkspaceID: workspaceID,
				CustomerID:  customerID,
				Currency:    "USD",
				LineItems: []params.LineItemCreateParams{
					{
						Description:     "Test Product",
						Quantity:        1.0,
						UnitAmountCents: 1000,
						LineItemType:    "product",
					},
				},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetNextInvoiceNumber(ctx, workspaceID).
					Return(int32(1), nil).
					Times(1)

				mockQuerier.EXPECT().
					GetCustomer(ctx, customerID).
					Return(db.Customer{ID: customerID}, nil).
					Times(1)

				// Mock workspace retrieval for tax calculation
				mockQuerier.EXPECT().
					GetWorkspace(ctx, workspaceID).
					Return(db.Workspace{
						ID:   workspaceID,
						Name: "Test Workspace",
					}, nil).
					AnyTimes()

				mockQuerier.EXPECT().
					CreateInvoiceWithDetails(ctx, gomock.Any()).
					Return(db.Invoice{}, assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to create invoice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.CreateInvoice(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.params.WorkspaceID, result.WorkspaceID)
			}
		})
	}
}

func TestInvoiceService_GetInvoiceWithDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewInvoiceService(
		mockQuerier,
		zap.NewNop(),
		nil, nil, nil, nil, nil,
	)
	ctx := context.Background()

	workspaceID := uuid.New()
	invoiceID := uuid.New()

	tests := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully retrieves invoice with details",
			setupMocks: func() {
				// Mock invoice retrieval
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
						AmountDue:   2000,
						Currency:    "USD",
						Status:      "open",
						TaxDetails:  []byte(`[{"jurisdiction_id":"US","tax_rate":0.10,"tax_amount_cents":200}]`),
					}, nil).
					Times(1)

				// Mock line items retrieval
				mockQuerier.EXPECT().
					GetInvoiceLineItems(ctx, invoiceID).
					Return([]db.InvoiceLineItem{
						{
							ID:                uuid.New(),
							Description:       "Test Product",
							UnitAmountInCents: 1800,
							AmountInCents:     1800,
							LineItemType:      pgtype.Text{String: "product", Valid: true},
						},
						{
							ID:                uuid.New(),
							Description:       "Tax (Sales Tax)",
							UnitAmountInCents: 200,
							AmountInCents:     200,
							LineItemType:      pgtype.Text{String: "tax", Valid: true},
						},
					}, nil).
					Times(1)

				// Mock subtotals retrieval
				mockQuerier.EXPECT().
					GetInvoiceSubtotal(ctx, invoiceID).
					Return(db.GetInvoiceSubtotalRow{
						ProductSubtotal:  1800,
						CustomerGasFees:  0,
						SponsoredGasFees: 0,
						TotalTax:         200,
						TotalDiscount:    0,
						CustomerTotal:    2000,
					}, nil).
					Times(1)

				// Mock crypto amounts retrieval
				mockQuerier.EXPECT().
					GetInvoiceCryptoAmounts(ctx, invoiceID).
					Return([]db.GetInvoiceCryptoAmountsRow{}, nil).
					Times(1)
			},
			wantErr: false,
		},
		{
			name: "fails when invoice not found",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{}, assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to get invoice",
		},
		{
			name: "fails when line items retrieval fails",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetInvoiceLineItems(ctx, invoiceID).
					Return(nil, assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to get line items",
		},
		{
			name: "fails when subtotals retrieval fails",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetInvoiceLineItems(ctx, invoiceID).
					Return([]db.InvoiceLineItem{}, nil).
					Times(1)

				mockQuerier.EXPECT().
					GetInvoiceSubtotal(ctx, invoiceID).
					Return(db.GetInvoiceSubtotalRow{}, assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to get invoice subtotals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetInvoiceWithDetails(ctx, workspaceID, invoiceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, invoiceID, result.ID)
				assert.Equal(t, workspaceID, result.WorkspaceID)
			}
		})
	}
}

func TestInvoiceService_FinalizeInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewInvoiceService(
		mockQuerier,
		zap.NewNop(),
		nil, nil, nil, nil, nil,
	)
	ctx := context.Background()

	workspaceID := uuid.New()
	invoiceID := uuid.New()
	customerID := uuid.New()

	tests := []struct {
		name           string
		setupMocks     func()
		wantErr        bool
		errorString    string
		expectedStatus string
	}{
		{
			name: "successfully finalizes invoice",
			setupMocks: func() {
				// Mock getting current invoice
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
						CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
						Status:      "draft",
						AmountDue:   2000,
						Currency:    "USD",
					}, nil).
					Times(1)

				// Mock updating invoice status
				mockQuerier.EXPECT().
					UpdateInvoice(ctx, gomock.Any()).
					Return(db.Invoice{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
						Status:      "open",
						AmountDue:   2000,
					}, nil).
					Times(1)
			},
			wantErr:        false,
			expectedStatus: "open",
		},
		{
			name: "fails when invoice not found",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{}, assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to get invoice",
		},
		{
			name: "fails when invoice cannot be finalized - already open",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
						Status:      "open",
					}, nil).
					Times(1)
			},
			wantErr:     true,
			errorString: "invoice cannot be finalized: current status is open",
		},
		{
			name: "fails when database update fails",
			setupMocks: func() {
				mockQuerier.EXPECT().
					GetInvoiceByID(ctx, db.GetInvoiceByIDParams{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
					}).
					Return(db.Invoice{
						ID:          invoiceID,
						WorkspaceID: workspaceID,
						CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
						Status:      "draft",
						AmountDue:   2000,
					}, nil).
					Times(1)

				mockQuerier.EXPECT().
					UpdateInvoice(ctx, gomock.Any()).
					Return(db.Invoice{}, assert.AnError).
					Times(1)
			},
			wantErr:     true,
			errorString: "failed to finalize invoice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.FinalizeInvoice(ctx, workspaceID, invoiceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, invoiceID, result.ID)
				if tt.expectedStatus != "" {
					assert.Equal(t, tt.expectedStatus, result.Status)
				}
			}
		})
	}
}
