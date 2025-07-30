package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger("test")
}

func TestCustomerService_GetCustomer(t *testing.T) {
	customerID := uuid.New()

	tests := []struct {
		name        string
		customerID  uuid.UUID
		mockSetup   func(m *mocks.MockQuerier)
		want        *db.Customer
		wantErr     bool
		errorString string
	}{
		{
			name:       "successfully retrieves customer",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				expectedCustomer := db.Customer{
					ID:    customerID,
					Email: pgtype.Text{String: "test@example.com", Valid: true},
					Name:  pgtype.Text{String: "Test User", Valid: true},
				}
				m.EXPECT().GetCustomer(gomock.Any(), customerID).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:    customerID,
				Email: pgtype.Text{String: "test@example.com", Valid: true},
				Name:  pgtype.Text{String: "Test User", Valid: true},
			},
			wantErr: false,
		},
		{
			name:       "customer not found",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().GetCustomer(gomock.Any(), customerID).Return(db.Customer{}, pgx.ErrNoRows)
			},
			want:        nil,
			wantErr:     true,
			errorString: "customer not found",
		},
		{
			name:       "database error",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().GetCustomer(gomock.Any(), customerID).Return(db.Customer{}, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to retrieve customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.GetCustomer(context.Background(), tt.customerID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_CreateCustomer(t *testing.T) {
	customerID := uuid.New()
	metadata := map[string]interface{}{
		"source": "api",
		"tier":   "premium",
	}
	metadataBytes, _ := json.Marshal(metadata)

	tests := []struct {
		name        string
		params      params.CreateCustomerParams
		mockSetup   func(m *mocks.MockQuerier)
		want        *db.Customer
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates customer with all fields",
			params: params.CreateCustomerParams{
				Email:              "test@example.com",
				Name:               aws.String("Test User"),
				Phone:              aws.String("+1234567890"),
				Description:        aws.String("Test customer"),
				FinishedOnboarding: true,
				Metadata:           metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.CreateCustomerParams{
					ExternalID:         pgtype.Text{String: "test@example.com", Valid: true},
					Email:              pgtype.Text{String: "test@example.com", Valid: true},
					Name:               pgtype.Text{String: "Test User", Valid: true},
					Phone:              pgtype.Text{String: "+1234567890", Valid: true},
					Description:        pgtype.Text{String: "Test customer", Valid: true},
					Metadata:           metadataBytes,
					FinishedOnboarding: true,
					PaymentSyncStatus:  "pending",
					PaymentProvider:    pgtype.Text{},
				}
				expectedCustomer := db.Customer{
					ID:                 customerID,
					Email:              pgtype.Text{String: "test@example.com", Valid: true},
					Name:               pgtype.Text{String: "Test User", Valid: true},
					FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
				}
				m.EXPECT().CreateCustomer(gomock.Any(), expectedParams).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:                 customerID,
				Email:              pgtype.Text{String: "test@example.com", Valid: true},
				Name:               pgtype.Text{String: "Test User", Valid: true},
				FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
			},
			wantErr: false,
		},
		{
			name: "successfully creates customer with minimal fields",
			params: params.CreateCustomerParams{
				Email:              "minimal@example.com",
				FinishedOnboarding: false,
				Metadata:           nil,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				nullMetadata, _ := json.Marshal(map[string]interface{}(nil))
				expectedParams := db.CreateCustomerParams{
					ExternalID:         pgtype.Text{String: "minimal@example.com", Valid: true},
					Email:              pgtype.Text{String: "minimal@example.com", Valid: true},
					Name:               pgtype.Text{String: "", Valid: false},
					Phone:              pgtype.Text{String: "", Valid: false},
					Description:        pgtype.Text{String: "", Valid: false},
					Metadata:           nullMetadata,
					FinishedOnboarding: false,
					PaymentSyncStatus:  "pending",
					PaymentProvider:    pgtype.Text{},
				}
				expectedCustomer := db.Customer{
					ID:    customerID,
					Email: pgtype.Text{String: "minimal@example.com", Valid: true},
				}
				m.EXPECT().CreateCustomer(gomock.Any(), expectedParams).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:    customerID,
				Email: pgtype.Text{String: "minimal@example.com", Valid: true},
			},
			wantErr: false,
		},
		{
			name: "handles invalid metadata",
			params: params.CreateCustomerParams{
				Email: "test@example.com",
				Metadata: map[string]interface{}{
					"invalid": make(chan int), // This will cause json.Marshal to fail
				},
			},
			mockSetup: func(m *mocks.MockQuerier) {
				// No mock expectation since it should fail before database call
			},
			want:        nil,
			wantErr:     true,
			errorString: "invalid metadata format",
		},
		{
			name: "database error during creation",
			params: params.CreateCustomerParams{
				Email:    "test@example.com",
				Metadata: metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().CreateCustomer(gomock.Any(), gomock.Any()).Return(db.Customer{}, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to create customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.CreateCustomer(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_UpdateCustomer(t *testing.T) {
	customerID := uuid.New()
	updatedEmail := "updated@example.com"
	updatedName := "Updated Name"
	updatedPhone := "+0987654321"
	updatedDescription := "Updated description"
	updatedOnboarding := true
	metadata := map[string]interface{}{
		"updated": true,
		"version": 2,
	}

	tests := []struct {
		name        string
		params      params.UpdateCustomerParams
		mockSetup   func(m *mocks.MockQuerier)
		want        *db.Customer
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully updates all fields",
			params: params.UpdateCustomerParams{
				ID:                 customerID,
				Email:              &updatedEmail,
				Name:               &updatedName,
				Phone:              &updatedPhone,
				Description:        &updatedDescription,
				FinishedOnboarding: &updatedOnboarding,
				Metadata:           metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				metadataBytes, _ := json.Marshal(metadata)
				expectedParams := db.UpdateCustomerParams{
					ID:                 customerID,
					Email:              pgtype.Text{String: updatedEmail, Valid: true},
					Name:               pgtype.Text{String: updatedName, Valid: true},
					Phone:              pgtype.Text{String: updatedPhone, Valid: true},
					Description:        pgtype.Text{String: updatedDescription, Valid: true},
					FinishedOnboarding: pgtype.Bool{Bool: updatedOnboarding, Valid: true},
					Metadata:           metadataBytes,
				}
				expectedCustomer := db.Customer{
					ID:    customerID,
					Email: pgtype.Text{String: updatedEmail, Valid: true},
					Name:  pgtype.Text{String: updatedName, Valid: true},
				}
				m.EXPECT().UpdateCustomer(gomock.Any(), expectedParams).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:    customerID,
				Email: pgtype.Text{String: updatedEmail, Valid: true},
				Name:  pgtype.Text{String: updatedName, Valid: true},
			},
			wantErr: false,
		},
		{
			name: "successfully updates partial fields",
			params: params.UpdateCustomerParams{
				ID:    customerID,
				Email: &updatedEmail,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.UpdateCustomerParams{
					ID:    customerID,
					Email: pgtype.Text{String: updatedEmail, Valid: true},
				}
				expectedCustomer := db.Customer{
					ID:    customerID,
					Email: pgtype.Text{String: updatedEmail, Valid: true},
				}
				m.EXPECT().UpdateCustomer(gomock.Any(), expectedParams).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:    customerID,
				Email: pgtype.Text{String: updatedEmail, Valid: true},
			},
			wantErr: false,
		},
		{
			name: "customer not found",
			params: params.UpdateCustomerParams{
				ID:    customerID,
				Email: &updatedEmail,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().UpdateCustomer(gomock.Any(), gomock.Any()).Return(db.Customer{}, pgx.ErrNoRows)
			},
			want:        nil,
			wantErr:     true,
			errorString: "customer not found",
		},
		{
			name: "handles invalid metadata",
			params: params.UpdateCustomerParams{
				ID:    customerID,
				Email: &updatedEmail,
				Metadata: map[string]interface{}{
					"invalid": make(chan int),
				},
			},
			mockSetup: func(m *mocks.MockQuerier) {
				// No mock expectation since it should fail before database call
			},
			want:        nil,
			wantErr:     true,
			errorString: "invalid metadata format",
		},
		{
			name: "database error during update",
			params: params.UpdateCustomerParams{
				ID:    customerID,
				Email: &updatedEmail,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().UpdateCustomer(gomock.Any(), gomock.Any()).Return(db.Customer{}, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to update customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.UpdateCustomer(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_DeleteCustomer(t *testing.T) {
	customerID := uuid.New()

	tests := []struct {
		name        string
		customerID  uuid.UUID
		mockSetup   func(m *mocks.MockQuerier)
		wantErr     bool
		errorString string
	}{
		{
			name:       "successfully deletes customer",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().DeleteCustomer(gomock.Any(), customerID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "customer not found",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().DeleteCustomer(gomock.Any(), customerID).Return(pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "customer not found",
		},
		{
			name:       "database error during deletion",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().DeleteCustomer(gomock.Any(), customerID).Return(errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to delete customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			err := service.DeleteCustomer(context.Background(), tt.customerID)

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

func TestCustomerService_ListCustomers(t *testing.T) {
	dbCustomers := []db.Customer{
		{
			ID:        uuid.New(),
			Email:     pgtype.Text{String: "user1@example.com", Valid: true},
			Metadata:  []byte("{}"),
			CreatedAt: pgtype.Timestamptz{},
			UpdatedAt: pgtype.Timestamptz{},
		},
		{
			ID:        uuid.New(),
			Email:     pgtype.Text{String: "user2@example.com", Valid: true},
			Metadata:  []byte("{}"),
			CreatedAt: pgtype.Timestamptz{},
			UpdatedAt: pgtype.Timestamptz{},
		},
	}

	// Convert to response format for expected result using the actual helper function
	customerResponses := make([]responses.CustomerResponse, len(dbCustomers))
	for i, c := range dbCustomers {
		customerResponses[i] = helpers.ToCustomerResponse(c)
	}

	tests := []struct {
		name        string
		params      params.ListCustomersParams
		mockSetup   func(m *mocks.MockQuerier)
		want        *responses.ListCustomersResult
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully lists customers",
			params: params.ListCustomersParams{
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.ListCustomersWithPaginationParams{
					Limit:  10,
					Offset: 0,
				}
				m.EXPECT().ListCustomersWithPagination(gomock.Any(), expectedParams).Return(dbCustomers, nil)
				m.EXPECT().CountCustomers(gomock.Any()).Return(int64(25), nil)
			},
			want: &responses.ListCustomersResult{
				Customers: customerResponses,
				Total:     25,
			},
			wantErr: false,
		},
		{
			name: "database error during listing",
			params: params.ListCustomersParams{
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().ListCustomersWithPagination(gomock.Any(), gomock.Any()).Return(nil, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to retrieve customers",
		},
		{
			name: "database error during counting",
			params: params.ListCustomersParams{
				Limit:  10,
				Offset: 0,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().ListCustomersWithPagination(gomock.Any(), gomock.Any()).Return(dbCustomers, nil)
				m.EXPECT().CountCustomers(gomock.Any()).Return(int64(0), errors.New("count error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to count customers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.ListCustomers(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_ListWorkspaceCustomers(t *testing.T) {
	workspaceID := uuid.New()
	dbCustomers := []db.Customer{
		{
			ID:        uuid.New(),
			Email:     pgtype.Text{String: "user1@example.com", Valid: true},
			Metadata:  []byte("{}"),
			CreatedAt: pgtype.Timestamptz{},
			UpdatedAt: pgtype.Timestamptz{},
		},
		{
			ID:        uuid.New(),
			Email:     pgtype.Text{String: "user2@example.com", Valid: true},
			Metadata:  []byte("{}"),
			CreatedAt: pgtype.Timestamptz{},
			UpdatedAt: pgtype.Timestamptz{},
		},
	}

	// Convert to response format for expected result using the actual helper function
	customerResponses := make([]responses.CustomerResponse, len(dbCustomers))
	for i, c := range dbCustomers {
		customerResponses[i] = helpers.ToCustomerResponse(c)
	}

	tests := []struct {
		name        string
		params      params.ListWorkspaceCustomersParams
		mockSetup   func(m *mocks.MockQuerier)
		want        *responses.ListWorkspaceCustomersResult
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully lists workspace customers",
			params: params.ListWorkspaceCustomersParams{
				WorkspaceID: workspaceID,
				Limit:       10,
				Offset:      0,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.ListWorkspaceCustomersWithPaginationParams{
					WorkspaceID: workspaceID,
					Limit:       10,
					Offset:      0,
				}
				m.EXPECT().ListWorkspaceCustomersWithPagination(gomock.Any(), expectedParams).Return(dbCustomers, nil)
				m.EXPECT().CountWorkspaceCustomers(gomock.Any(), workspaceID).Return(int64(15), nil)
			},
			want: &responses.ListWorkspaceCustomersResult{
				Customers: customerResponses,
				Total:     15,
			},
			wantErr: false,
		},
		{
			name: "database error during listing",
			params: params.ListWorkspaceCustomersParams{
				WorkspaceID: workspaceID,
				Limit:       10,
				Offset:      0,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().ListWorkspaceCustomersWithPagination(gomock.Any(), gomock.Any()).Return(nil, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to retrieve workspace customers",
		},
		{
			name: "database error during counting",
			params: params.ListWorkspaceCustomersParams{
				WorkspaceID: workspaceID,
				Limit:       10,
				Offset:      0,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().ListWorkspaceCustomersWithPagination(gomock.Any(), gomock.Any()).Return(dbCustomers, nil)
				m.EXPECT().CountWorkspaceCustomers(gomock.Any(), workspaceID).Return(int64(0), errors.New("count error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to count workspace customers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.ListWorkspaceCustomers(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_UpdateCustomerOnboardingStatus(t *testing.T) {
	customerID := uuid.New()

	tests := []struct {
		name               string
		customerID         uuid.UUID
		finishedOnboarding bool
		mockSetup          func(m *mocks.MockQuerier)
		want               *db.Customer
		wantErr            bool
		errorString        string
	}{
		{
			name:               "successfully updates onboarding status to true",
			customerID:         customerID,
			finishedOnboarding: true,
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.UpdateCustomerOnboardingStatusParams{
					ID:                 customerID,
					FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
				}
				expectedCustomer := db.Customer{
					ID:                 customerID,
					FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
				}
				m.EXPECT().UpdateCustomerOnboardingStatus(gomock.Any(), expectedParams).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:                 customerID,
				FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
			},
			wantErr: false,
		},
		{
			name:               "successfully updates onboarding status to false",
			customerID:         customerID,
			finishedOnboarding: false,
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.UpdateCustomerOnboardingStatusParams{
					ID:                 customerID,
					FinishedOnboarding: pgtype.Bool{Bool: false, Valid: true},
				}
				expectedCustomer := db.Customer{
					ID:                 customerID,
					FinishedOnboarding: pgtype.Bool{Bool: false, Valid: true},
				}
				m.EXPECT().UpdateCustomerOnboardingStatus(gomock.Any(), expectedParams).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:                 customerID,
				FinishedOnboarding: pgtype.Bool{Bool: false, Valid: true},
			},
			wantErr: false,
		},
		{
			name:               "customer not found",
			customerID:         customerID,
			finishedOnboarding: true,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().UpdateCustomerOnboardingStatus(gomock.Any(), gomock.Any()).Return(db.Customer{}, pgx.ErrNoRows)
			},
			want:        nil,
			wantErr:     true,
			errorString: "customer not found",
		},
		{
			name:               "database error during update",
			customerID:         customerID,
			finishedOnboarding: true,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().UpdateCustomerOnboardingStatus(gomock.Any(), gomock.Any()).Return(db.Customer{}, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to update customer onboarding status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.UpdateCustomerOnboardingStatus(context.Background(), tt.customerID, tt.finishedOnboarding)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_AddCustomerToWorkspace(t *testing.T) {
	workspaceID := uuid.New()
	customerID := uuid.New()

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		customerID  uuid.UUID
		mockSetup   func(m *mocks.MockQuerier)
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully adds customer to workspace",
			workspaceID: workspaceID,
			customerID:  customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.AddCustomerToWorkspaceParams{
					WorkspaceID: workspaceID,
					CustomerID:  customerID,
				}
				m.EXPECT().AddCustomerToWorkspace(gomock.Any(), expectedParams).Return(db.WorkspaceCustomer{}, nil)
			},
			wantErr: false,
		},
		{
			name:        "database error during addition",
			workspaceID: workspaceID,
			customerID:  customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().AddCustomerToWorkspace(gomock.Any(), gomock.Any()).Return(db.WorkspaceCustomer{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to add customer to workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			err := service.AddCustomerToWorkspace(context.Background(), tt.workspaceID, tt.customerID)

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

func TestCustomerService_GetCustomerByWeb3AuthID(t *testing.T) {
	web3authID := "web3auth-12345"
	customerID := uuid.New()

	tests := []struct {
		name        string
		web3authID  string
		mockSetup   func(m *mocks.MockQuerier)
		want        *db.Customer
		wantErr     bool
		errorString string
	}{
		{
			name:       "successfully retrieves customer by web3auth ID",
			web3authID: web3authID,
			mockSetup: func(m *mocks.MockQuerier) {
				expectedCustomer := db.Customer{
					ID:         customerID,
					Web3authID: pgtype.Text{String: web3authID, Valid: true},
					Email:      pgtype.Text{String: "test@example.com", Valid: true},
				}
				m.EXPECT().GetCustomerByWeb3AuthID(gomock.Any(), pgtype.Text{String: web3authID, Valid: true}).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:         customerID,
				Web3authID: pgtype.Text{String: web3authID, Valid: true},
				Email:      pgtype.Text{String: "test@example.com", Valid: true},
			},
			wantErr: false,
		},
		{
			name:       "customer not found",
			web3authID: web3authID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().GetCustomerByWeb3AuthID(gomock.Any(), pgtype.Text{String: web3authID, Valid: true}).Return(db.Customer{}, pgx.ErrNoRows)
			},
			want:        nil,
			wantErr:     true,
			errorString: "customer not found",
		},
		{
			name:       "database error",
			web3authID: web3authID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().GetCustomerByWeb3AuthID(gomock.Any(), pgtype.Text{String: web3authID, Valid: true}).Return(db.Customer{}, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to retrieve customer",
		},
		{
			name:       "handles empty web3auth ID",
			web3authID: "",
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().GetCustomerByWeb3AuthID(gomock.Any(), pgtype.Text{String: "", Valid: false}).Return(db.Customer{}, pgx.ErrNoRows)
			},
			want:        nil,
			wantErr:     true,
			errorString: "customer not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.GetCustomerByWeb3AuthID(context.Background(), tt.web3authID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_CreateCustomerWithWeb3Auth(t *testing.T) {
	customerID := uuid.New()
	metadata := map[string]interface{}{
		"provider": "web3auth",
		"tier":     "standard",
	}
	metadataBytes, _ := json.Marshal(metadata)

	tests := []struct {
		name        string
		params      params.CreateCustomerWithWeb3AuthParams
		mockSetup   func(m *mocks.MockQuerier)
		want        *db.Customer
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates customer with web3auth",
			params: params.CreateCustomerWithWeb3AuthParams{
				Web3AuthID: "web3auth-12345",
				Email:      "test@example.com",
				Name:       aws.String("Test User"),
				Metadata:   metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.CreateCustomerWithWeb3AuthParams{
					Web3authID:         pgtype.Text{String: "web3auth-12345", Valid: true},
					Email:              pgtype.Text{String: "test@example.com", Valid: true},
					Name:               pgtype.Text{String: "Test User", Valid: true},
					Phone:              pgtype.Text{}, // Phone not in params type
					Description:        pgtype.Text{},
					Metadata:           metadataBytes,
					FinishedOnboarding: false, // Not in params type
				}
				expectedCustomer := db.Customer{
					ID:         customerID,
					Web3authID: pgtype.Text{String: "web3auth-12345", Valid: true},
					Email:      pgtype.Text{String: "test@example.com", Valid: true},
					Name:       pgtype.Text{String: "Test User", Valid: true},
				}
				m.EXPECT().CreateCustomerWithWeb3Auth(gomock.Any(), expectedParams).Return(expectedCustomer, nil)
			},
			want: &db.Customer{
				ID:         customerID,
				Web3authID: pgtype.Text{String: "web3auth-12345", Valid: true},
				Email:      pgtype.Text{String: "test@example.com", Valid: true},
				Name:       pgtype.Text{String: "Test User", Valid: true},
			},
			wantErr: false,
		},
		{
			name: "handles invalid metadata",
			params: params.CreateCustomerWithWeb3AuthParams{
				Web3AuthID: "web3auth-12345",
				Email:      "test@example.com",
				Metadata: map[string]interface{}{
					"invalid": make(chan int),
				},
			},
			mockSetup: func(m *mocks.MockQuerier) {
				// No mock expectation since it should fail before database call
			},
			want:        nil,
			wantErr:     true,
			errorString: "invalid metadata format",
		},
		{
			name: "database error during creation",
			params: params.CreateCustomerWithWeb3AuthParams{
				Web3AuthID: "web3auth-12345",
				Email:      "test@example.com",
				Metadata:   metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().CreateCustomerWithWeb3Auth(gomock.Any(), gomock.Any()).Return(db.Customer{}, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to create customer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.CreateCustomerWithWeb3Auth(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_ListCustomerWallets(t *testing.T) {
	customerID := uuid.New()
	wallets := []db.CustomerWallet{
		{
			ID:            uuid.New(),
			CustomerID:    customerID,
			WalletAddress: "0x1234567890abcdef",
			NetworkType:   db.NetworkTypeEvm,
		},
		{
			ID:            uuid.New(),
			CustomerID:    customerID,
			WalletAddress: "0xfedcba0987654321",
			NetworkType:   db.NetworkTypeEvm,
		},
	}

	tests := []struct {
		name        string
		customerID  uuid.UUID
		mockSetup   func(m *mocks.MockQuerier)
		want        []db.CustomerWallet
		wantErr     bool
		errorString string
	}{
		{
			name:       "successfully lists customer wallets",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().ListCustomerWallets(gomock.Any(), customerID).Return(wallets, nil)
			},
			want:    wallets,
			wantErr: false,
		},
		{
			name:       "returns empty list when no wallets found",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().ListCustomerWallets(gomock.Any(), customerID).Return([]db.CustomerWallet{}, nil)
			},
			want:    []db.CustomerWallet{},
			wantErr: false,
		},
		{
			name:       "database error during listing",
			customerID: customerID,
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().ListCustomerWallets(gomock.Any(), customerID).Return(nil, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to retrieve customer wallets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.ListCustomerWallets(context.Background(), tt.customerID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCustomerService_CreateCustomerWallet(t *testing.T) {
	customerID := uuid.New()
	walletID := uuid.New()
	metadata := map[string]interface{}{
		"source": "metamask",
		"index":  0,
	}
	metadataBytes, _ := json.Marshal(metadata)

	tests := []struct {
		name        string
		params      params.CreateCustomerWalletParams
		mockSetup   func(m *mocks.MockQuerier)
		want        *db.CustomerWallet
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates customer wallet",
			params: params.CreateCustomerWalletParams{
				CustomerID:    customerID,
				WalletAddress: "0x1234567890abcdef",
				NetworkType:   "evm",
				Nickname:      aws.String("Primary Wallet"),
				ENS:           aws.String("test.eth"),
				IsPrimary:     true,
				Verified:      true,
				Metadata:      metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				expectedParams := db.CreateCustomerWalletParams{
					CustomerID:    customerID,
					WalletAddress: "0x1234567890abcdef",
					NetworkType:   db.NetworkTypeEvm,
					Nickname:      pgtype.Text{String: "Primary Wallet", Valid: true},
					Ens:           pgtype.Text{String: "test.eth", Valid: true},
					IsPrimary:     pgtype.Bool{Bool: true, Valid: true},
					Verified:      pgtype.Bool{Bool: true, Valid: true},
					Metadata:      metadataBytes,
				}
				expectedWallet := db.CustomerWallet{
					ID:            walletID,
					CustomerID:    customerID,
					WalletAddress: "0x1234567890abcdef",
					NetworkType:   db.NetworkTypeEvm,
					IsPrimary:     pgtype.Bool{Bool: true, Valid: true},
				}
				m.EXPECT().CreateCustomerWallet(gomock.Any(), expectedParams).Return(expectedWallet, nil)
			},
			want: &db.CustomerWallet{
				ID:            walletID,
				CustomerID:    customerID,
				WalletAddress: "0x1234567890abcdef",
				NetworkType:   db.NetworkTypeEvm,
				IsPrimary:     pgtype.Bool{Bool: true, Valid: true},
			},
			wantErr: false,
		},
		{
			name: "handles invalid metadata",
			params: params.CreateCustomerWalletParams{
				CustomerID:    customerID,
				WalletAddress: "0x1234567890abcdef",
				NetworkType:   "evm",
				Metadata: map[string]interface{}{
					"invalid": make(chan int),
				},
			},
			mockSetup: func(m *mocks.MockQuerier) {
				// No mock expectation since it should fail before database call
			},
			want:        nil,
			wantErr:     true,
			errorString: "invalid wallet metadata format",
		},
		{
			name: "handles invalid network type",
			params: params.CreateCustomerWalletParams{
				CustomerID:    customerID,
				WalletAddress: "0x1234567890abcdef",
				NetworkType:   "invalid",
				Metadata:      metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				// No mock expectation since it should fail before database call
			},
			want:        nil,
			wantErr:     true,
			errorString: "invalid network type",
		},
		{
			name: "database error during creation",
			params: params.CreateCustomerWalletParams{
				CustomerID:    customerID,
				WalletAddress: "0x1234567890abcdef",
				NetworkType:   "evm",
				Metadata:      metadata,
			},
			mockSetup: func(m *mocks.MockQuerier) {
				m.EXPECT().CreateCustomerWallet(gomock.Any(), gomock.Any()).Return(db.CustomerWallet{}, errors.New("database error"))
			},
			want:        nil,
			wantErr:     true,
			errorString: "failed to create customer wallet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			tt.mockSetup(mockQuerier)

			service := services.NewCustomerService(mockQuerier)
			got, err := service.CreateCustomerWallet(context.Background(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseNetworkType(t *testing.T) {
	tests := []struct {
		name        string
		networkType string
		want        db.NetworkType
		wantErr     bool
	}{
		{
			name:        "valid evm network type",
			networkType: "evm",
			want:        db.NetworkTypeEvm,
			wantErr:     false,
		},
		{
			name:        "valid solana network type",
			networkType: "solana",
			want:        db.NetworkTypeSolana,
			wantErr:     false,
		},
		{
			name:        "valid cosmos network type",
			networkType: "cosmos",
			want:        db.NetworkTypeCosmos,
			wantErr:     false,
		},
		{
			name:        "valid bitcoin network type",
			networkType: "bitcoin",
			want:        db.NetworkTypeBitcoin,
			wantErr:     false,
		},
		{
			name:        "valid polkadot network type",
			networkType: "polkadot",
			want:        db.NetworkTypePolkadot,
			wantErr:     false,
		},
		{
			name:        "invalid network type",
			networkType: "invalid",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "empty network type",
			networkType: "",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "case sensitive - uppercase should fail",
			networkType: "EVM",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since parseNetworkType is not exported, we test it indirectly
			// through CreateCustomerWallet which uses it internally
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)

			// Only set up mock expectation if we expect the function to reach the database call
			if !tt.wantErr {
				mockQuerier.EXPECT().CreateCustomerWallet(gomock.Any(), gomock.Any()).Return(db.CustomerWallet{}, nil)
			}

			service := services.NewCustomerService(mockQuerier)
			params := params.CreateCustomerWalletParams{
				CustomerID:    uuid.New(),
				WalletAddress: "0x1234567890abcdef",
				NetworkType:   tt.networkType,
				Metadata:      map[string]interface{}{},
			}

			_, err := service.CreateCustomerWallet(context.Background(), params)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.networkType != "" && tt.networkType != "EVM" {
					assert.Contains(t, err.Error(), "invalid network type")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCustomerService_ErrorHandling tests error scenarios and edge cases
func TestCustomerService_ErrorHandling(t *testing.T) {
	t.Run("service creation with nil querier", func(t *testing.T) {
		// Service should handle nil querier gracefully during creation
		assert.NotPanics(t, func() {
			service := services.NewCustomerService(nil)
			assert.NotNil(t, service)
		})
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewCustomerService(mockQuerier)

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Mock should expect the call but return context error
		mockQuerier.EXPECT().GetCustomer(gomock.Any(), gomock.Any()).Return(db.Customer{}, context.Canceled)

		_, err := service.GetCustomer(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve customer")
	})
}

// TestCustomerService_BoundaryConditions tests boundary conditions and edge cases
func TestCustomerService_BoundaryConditions(t *testing.T) {
	t.Run("empty metadata", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewCustomerService(mockQuerier)

		// Test with nil metadata
		params := params.CreateCustomerParams{
			Email:    "test@example.com",
			Metadata: nil,
		}

		expectedMetadata, _ := json.Marshal(map[string]interface{}(nil))
		expectedParams := db.CreateCustomerParams{
			ExternalID:         pgtype.Text{String: "test@example.com", Valid: true},
			Email:              pgtype.Text{String: "test@example.com", Valid: true},
			Name:               pgtype.Text{String: "", Valid: false},
			Phone:              pgtype.Text{String: "", Valid: false},
			Description:        pgtype.Text{String: "", Valid: false},
			Metadata:           expectedMetadata,
			FinishedOnboarding: false,
			PaymentSyncStatus:  "pending",
			PaymentProvider:    pgtype.Text{},
		}

		mockQuerier.EXPECT().CreateCustomer(gomock.Any(), expectedParams).Return(db.Customer{}, nil)

		_, err := service.CreateCustomer(context.Background(), params)
		assert.NoError(t, err)
	})

	t.Run("large metadata object", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewCustomerService(mockQuerier)

		// Create large metadata object
		largeMetadata := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largeMetadata[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}

		params := params.CreateCustomerParams{
			Email:    "test@example.com",
			Metadata: largeMetadata,
		}

		mockQuerier.EXPECT().CreateCustomer(gomock.Any(), gomock.Any()).Return(db.Customer{}, nil)

		_, err := service.CreateCustomer(context.Background(), params)
		assert.NoError(t, err)
	})

	t.Run("zero limit and offset", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewCustomerService(mockQuerier)

		params := params.ListCustomersParams{
			Limit:  0,
			Offset: 0,
		}

		expectedParams := db.ListCustomersWithPaginationParams{
			Limit:  0,
			Offset: 0,
		}

		mockQuerier.EXPECT().ListCustomersWithPagination(gomock.Any(), expectedParams).Return([]db.Customer{}, nil)
		mockQuerier.EXPECT().CountCustomers(gomock.Any()).Return(int64(0), nil)

		result, err := service.ListCustomers(context.Background(), params)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(result.Customers))
		assert.Equal(t, int64(0), result.Total)
	})
}

func TestCustomerService_ParseNetworkType(t *testing.T) {
	tests := []struct {
		name        string
		networkType string
		want        db.NetworkType
		wantErr     bool
	}{
		{
			name:        "valid evm network",
			networkType: "evm",
			want:        db.NetworkTypeEvm,
			wantErr:     false,
		},
		{
			name:        "valid solana network",
			networkType: "solana",
			want:        db.NetworkTypeSolana,
			wantErr:     false,
		},
		{
			name:        "valid cosmos network",
			networkType: "cosmos",
			want:        db.NetworkTypeCosmos,
			wantErr:     false,
		},
		{
			name:        "valid bitcoin network",
			networkType: "bitcoin",
			want:        db.NetworkTypeBitcoin,
			wantErr:     false,
		},
		{
			name:        "valid polkadot network",
			networkType: "polkadot",
			want:        db.NetworkTypePolkadot,
			wantErr:     false,
		},
		{
			name:        "invalid network type",
			networkType: "invalid",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "empty network type",
			networkType: "",
			want:        "",
			wantErr:     true,
		},
		{
			name:        "case sensitive - EVM uppercase",
			networkType: "EVM",
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parseNetworkType function indirectly through CreateCustomerWallet
			// since parseNetworkType is not exported
			walletParams := params.CreateCustomerWalletParams{
				CustomerID:    uuid.New(),
				WalletAddress: "0x123",
				NetworkType:   tt.networkType,
				IsPrimary:     true,
				Verified:      true,
				Metadata:      map[string]interface{}{},
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockQuerier := mocks.NewMockQuerier(ctrl)
			service := services.NewCustomerService(mockQuerier)

			if tt.wantErr {
				// Don't set up any mocks - should fail at parseNetworkType
				_, err := service.CreateCustomerWallet(context.Background(), walletParams)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported network type")
			} else {
				// Mock the database call since we just want to test parsing
				mockQuerier.EXPECT().CreateCustomerWallet(gomock.Any(), gomock.Any()).Return(db.CustomerWallet{}, nil)
				_, err := service.CreateCustomerWallet(context.Background(), walletParams)
				assert.NoError(t, err)
			}
		})
	}
}

// NOTE: ProcessCustomerAndWallet, CreateCustomerFromWallet, and FindOrCreateCustomerWallet
// are transaction-based methods that take pgx.Tx as parameters. These are integration methods
// that should be tested with integration tests rather than unit tests with mocks.
// The complexity of properly mocking transaction behavior makes unit testing these methods
// less valuable than integration testing them with a real database.

// TODO: Add integration tests for:
// - ProcessCustomerAndWallet
// - CreateCustomerFromWallet
// - FindOrCreateCustomerWallet
