package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestAccountService_CreateAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	tests := []struct {
		name        string
		params      params.CreateAccountParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates account",
			params: params.CreateAccountParams{
				Name:         "Test Account",
				AccountType:  "merchant",
				BusinessName: "Test Business",
				WebsiteURL:   "https://test.com",
				Metadata:     map[string]interface{}{"key": "value"},
			},
			setupMocks: func() {
				expectedAccount := db.Account{
					ID:          uuid.New(),
					Name:        "Test Account",
					AccountType: "merchant",
				}
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(expectedAccount, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty name",
			params: params.CreateAccountParams{
				AccountType: "merchant",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "account name is required",
		},
		{
			name: "fails with empty account type",
			params: params.CreateAccountParams{
				Name: "Test Account",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "account type is required",
		},
		{
			name: "fails with invalid account type",
			params: params.CreateAccountParams{
				Name:        "Test Account",
				AccountType: "invalid",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "invalid account type: invalid. Must be 'admin' or 'merchant'",
		},
		{
			name: "fails with invalid metadata",
			params: params.CreateAccountParams{
				Name:        "Test Account",
				AccountType: "admin",
				Metadata:    map[string]interface{}{"key": make(chan int)}, // Invalid JSON
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "invalid metadata format",
		},
		{
			name: "handles database error",
			params: params.CreateAccountParams{
				Name:        "Test Account",
				AccountType: "merchant",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(db.Account{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create account",
		},
		{
			name: "creates admin account successfully",
			params: params.CreateAccountParams{
				Name:               "Admin Account",
				AccountType:        "admin",
				FinishedOnboarding: true,
			},
			setupMocks: func() {
				expectedAccount := db.Account{
					ID:          uuid.New(),
					Name:        "Admin Account",
					AccountType: "admin",
				}
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(expectedAccount, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			account, err := service.CreateAccount(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, account)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, account)
				assert.Equal(t, tt.params.Name, account.Name)
				assert.Equal(t, tt.params.AccountType, string(account.AccountType))
			}
		})
	}
}

func TestAccountService_GetAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	testID := uuid.New()
	expectedAccount := db.Account{
		ID:   testID,
		Name: "Test Account",
	}

	tests := []struct {
		name        string
		accountID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:      "successfully gets account",
			accountID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(expectedAccount, nil)
			},
			wantErr: false,
		},
		{
			name:      "account not found",
			accountID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(db.Account{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "account not found",
		},
		{
			name:      "database error",
			accountID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(db.Account{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			account, err := service.GetAccount(ctx, tt.accountID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, account)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, account)
				assert.Equal(t, testID, account.ID)
			}
		})
	}
}

func TestAccountService_ListAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	expectedAccounts := []db.Account{
		{ID: uuid.New(), Name: "Account 1"},
		{ID: uuid.New(), Name: "Account 2"},
	}

	tests := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name: "successfully lists accounts",
			setupMocks: func() {
				mockQuerier.EXPECT().ListAccounts(ctx).Return(expectedAccounts, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "returns empty list",
			setupMocks: func() {
				mockQuerier.EXPECT().ListAccounts(ctx).Return([]db.Account{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "database error",
			setupMocks: func() {
				mockQuerier.EXPECT().ListAccounts(ctx).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve accounts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			accounts, err := service.ListAccounts(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, accounts)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, accounts, tt.wantCount)
			}
		})
	}
}

func TestAccountService_UpdateAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	testID := uuid.New()
	existingAccount := db.Account{
		ID:          testID,
		Name:        "Original Name",
		AccountType: "merchant",
		Metadata:    []byte(`{"existing": "data"}`),
	}

	tests := []struct {
		name        string
		params      params.UpdateAccountParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully updates account",
			params: params.UpdateAccountParams{
				ID:           testID,
				Name:         aws.String("Updated Name"),
				BusinessName: aws.String("Updated Business"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
				updatedAccount := existingAccount
				updatedAccount.Name = "Updated Name"
				mockQuerier.EXPECT().UpdateAccount(ctx, gomock.Any()).Return(updatedAccount, nil)
			},
			wantErr: false,
		},
		{
			name: "account not found",
			params: params.UpdateAccountParams{
				ID:           testID,
				Name:         aws.String("Updated Name"),
				BusinessName: aws.String("Updated Business"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(db.Account{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "account not found",
		},
		{
			name: "invalid account type",
			params: params.UpdateAccountParams{
				ID:          testID,
				AccountType: aws.String("invalid"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
			},
			wantErr:     true,
			errorString: "invalid account type: invalid",
		},
		{
			name: "invalid metadata format",
			params: params.UpdateAccountParams{
				ID:       testID,
				Metadata: map[string]interface{}{"key": make(chan int)},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
			},
			wantErr:     true,
			errorString: "invalid metadata format",
		},
		{
			name: "database update error",
			params: params.UpdateAccountParams{
				ID:   testID,
				Name: aws.String("Updated Name"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
				mockQuerier.EXPECT().UpdateAccount(ctx, gomock.Any()).Return(db.Account{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update account",
		},
		{
			name: "updates with new metadata",
			params: params.UpdateAccountParams{
				ID:       testID,
				Metadata: map[string]interface{}{"new": "metadata"},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
				updatedAccount := existingAccount
				mockQuerier.EXPECT().UpdateAccount(ctx, gomock.Any()).Return(updatedAccount, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			account, err := service.UpdateAccount(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, account)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, account)
			}
		})
	}
}

func TestAccountService_DeleteAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	testID := uuid.New()
	existingAccount := db.Account{
		ID:   testID,
		Name: "Test Account",
	}

	tests := []struct {
		name        string
		accountID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:      "successfully deletes account",
			accountID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
				mockQuerier.EXPECT().DeleteAccount(ctx, testID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "account not found",
			accountID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(db.Account{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "account not found",
		},
		{
			name:      "database delete error",
			accountID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
				mockQuerier.EXPECT().DeleteAccount(ctx, testID).Return(errors.New("delete error"))
			},
			wantErr:     true,
			errorString: "failed to delete account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteAccount(ctx, tt.accountID)

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

func TestAccountService_ValidateSignInRequest(t *testing.T) {
	service := services.NewAccountService(nil) // No DB queries needed for this test

	tests := []struct {
		name        string
		metadata    map[string]interface{}
		wantWeb3ID  string
		wantEmail   string
		wantErr     bool
		errorString string
	}{
		{
			name: "valid metadata",
			metadata: map[string]interface{}{
				"ownerWeb3AuthId": "web3-auth-123",
				"email":           "test@example.com",
			},
			wantWeb3ID: "web3-auth-123",
			wantEmail:  "test@example.com",
			wantErr:    false,
		},
		{
			name:        "missing ownerWeb3AuthId",
			metadata:    map[string]interface{}{"email": "test@example.com"},
			wantErr:     true,
			errorString: "ownerWeb3AuthId is required",
		},
		{
			name: "empty ownerWeb3AuthId",
			metadata: map[string]interface{}{
				"ownerWeb3AuthId": "",
				"email":           "test@example.com",
			},
			wantErr:     true,
			errorString: "ownerWeb3AuthId is required",
		},
		{
			name: "ownerWeb3AuthId not string",
			metadata: map[string]interface{}{
				"ownerWeb3AuthId": 123,
				"email":           "test@example.com",
			},
			wantErr:     true,
			errorString: "ownerWeb3AuthId is required",
		},
		{
			name: "missing email",
			metadata: map[string]interface{}{
				"ownerWeb3AuthId": "web3-auth-123",
			},
			wantErr:     true,
			errorString: "email is required",
		},
		{
			name: "empty email",
			metadata: map[string]interface{}{
				"ownerWeb3AuthId": "web3-auth-123",
				"email":           "",
			},
			wantErr:     true,
			errorString: "email is required",
		},
		{
			name: "email not string",
			metadata: map[string]interface{}{
				"ownerWeb3AuthId": "web3-auth-123",
				"email":           123,
			},
			wantErr:     true,
			errorString: "email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			web3ID, email, err := service.ValidateSignInRequest(tt.metadata)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, web3ID)
				assert.Empty(t, email)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantWeb3ID, web3ID)
				assert.Equal(t, tt.wantEmail, email)
			}
		})
	}
}

func TestAccountService_SignInOrRegisterAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	web3AuthID := "web3-auth-123"
	email := "test@example.com"
	accountID := uuid.New()
	userID := uuid.New()

	existingUser := db.User{
		ID:        userID,
		AccountID: accountID,
		Email:     email,
	}

	existingAccount := db.Account{
		ID:   accountID,
		Name: "Existing Account",
	}

	workspaces := []db.Workspace{
		{ID: uuid.New(), Name: "Workspace 1", AccountID: accountID},
	}

	tests := []struct {
		name         string
		createParams params.CreateAccountParams
		web3AuthID   string
		email        string
		setupMocks   func()
		wantErr      bool
		errorString  string
		wantNewUser  bool
	}{
		{
			name: "existing user signs in",
			createParams: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(existingUser, nil)
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().ListWorkspacesByAccountID(ctx, accountID).Return(workspaces, nil)
			},
			wantErr:     false,
			wantNewUser: false,
		},
		{
			name: "new user registration",
			createParams: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
				Metadata: map[string]interface{}{
					"verifier":   "google",
					"verifierId": "google-123",
				},
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(db.User{}, pgx.ErrNoRows)
				// CreateAccount call
				newAccount := db.Account{ID: uuid.New(), Name: "New Account"}
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(newAccount, nil)
				// CreateUser call
				newUser := db.User{ID: uuid.New(), AccountID: newAccount.ID, Email: email}
				mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(newUser, nil)
				// CreateWorkspace call
				newWorkspace := db.Workspace{ID: uuid.New(), Name: "Default", AccountID: newAccount.ID}
				mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(newWorkspace, nil)
			},
			wantErr:     false,
			wantNewUser: true,
		},
		{
			name: "database error checking user",
			createParams: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(db.User{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to check existing user",
		},
		{
			name: "error getting existing account",
			createParams: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(existingUser, nil)
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(db.Account{}, errors.New("account error"))
			},
			wantErr:     true,
			errorString: "failed to get account",
		},
		{
			name: "error getting workspaces",
			createParams: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(existingUser, nil)
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().ListWorkspacesByAccountID(ctx, accountID).Return(nil, errors.New("workspace error"))
			},
			wantErr:     true,
			errorString: "failed to get workspaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.SignInOrRegisterAccount(ctx, tt.createParams, tt.web3AuthID, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantNewUser, result.IsNewUser)
				assert.NotNil(t, result.Account)
				assert.NotNil(t, result.User)
				assert.NotEmpty(t, result.Workspaces)
			}
		})
	}
}

func TestAccountService_OnboardAccount(t *testing.T) {
	accountID := uuid.New()
	userID := uuid.New()

	existingAccount := db.Account{
		ID:   accountID,
		Name: "Test Account",
	}

	existingUser := db.User{
		ID:        userID,
		AccountID: accountID,
		Email:     "test@example.com",
	}

	tests := []struct {
		name        string
		params      params.OnboardAccountParams
		setupMocks  func(*mocks.MockQuerier, context.Context)
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully onboards account",
			params: params.OnboardAccountParams{
				AccountID:    accountID,
				UserID:       userID,
				FirstName:    "John",
				LastName:     "Doe",
				AddressLine1: "123 Main St",
				City:         "New York",
				State:        "NY",
				PostalCode:   "10001",
				Country:      "US",
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier, ctx context.Context) {
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				// UpdateAccount calls GetAccount internally again
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().UpdateAccount(ctx, gomock.Any()).Return(existingAccount, nil)
				mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(existingUser, nil)
			},
			wantErr: false,
		},
		{
			name: "account not found",
			params: params.OnboardAccountParams{
				AccountID: accountID,
				UserID:    userID,
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier, ctx context.Context) {
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(db.Account{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "account not found",
		},
		{
			name: "user not found",
			params: params.OnboardAccountParams{
				AccountID: accountID,
				UserID:    userID,
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier, ctx context.Context) {
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(db.User{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "user not found",
		},
		{
			name: "account update error",
			params: params.OnboardAccountParams{
				AccountID: accountID,
				UserID:    userID,
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier, ctx context.Context) {
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				// UpdateAccount calls GetAccount internally again
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().UpdateAccount(ctx, gomock.Any()).Return(db.Account{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update account",
		},
		{
			name: "user update error",
			params: params.OnboardAccountParams{
				AccountID: accountID,
				UserID:    userID,
			},
			setupMocks: func(mockQuerier *mocks.MockQuerier, ctx context.Context) {
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				// UpdateAccount calls GetAccount internally again
				mockQuerier.EXPECT().GetAccount(ctx, accountID).Return(existingAccount, nil)
				mockQuerier.EXPECT().UpdateAccount(ctx, gomock.Any()).Return(existingAccount, nil)
				mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(db.User{}, errors.New("user update error"))
			},
			wantErr:     true,
			errorString: "failed to update user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockQuerier := mocks.NewMockQuerier(ctrl)
			service := services.NewAccountService(mockQuerier)
			ctx := context.Background()

			tt.setupMocks(mockQuerier, ctx)

			err := service.OnboardAccount(ctx, tt.params)

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

func TestAccountService_ValidateAccountAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	accountID := uuid.New()
	workspaceID := uuid.New()
	otherAccountID := uuid.New()

	validWorkspace := db.Workspace{
		ID:        workspaceID,
		AccountID: accountID,
		Name:      "Test Workspace",
	}

	invalidWorkspace := db.Workspace{
		ID:        workspaceID,
		AccountID: otherAccountID,
		Name:      "Other Workspace",
	}

	tests := []struct {
		name        string
		accountID   uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "valid access",
			accountID:   accountID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(validWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name:        "workspace not found",
			accountID:   accountID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "workspace not found",
		},
		{
			name:        "database error",
			accountID:   accountID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to get workspace",
		},
		{
			name:        "workspace belongs to different account",
			accountID:   accountID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(invalidWorkspace, nil)
			},
			wantErr:     true,
			errorString: "you are not authorized to access this account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.ValidateAccountAccess(ctx, tt.accountID, tt.workspaceID)

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

func TestAccountService_CreateNewAccountWithUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	web3AuthID := "web3-auth-123"
	email := "test@example.com"

	tests := []struct {
		name        string
		params      params.CreateAccountParams
		web3AuthID  string
		email       string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates account with user",
			params: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
				Metadata: map[string]interface{}{
					"verifier":   "google",
					"verifierId": "google-123",
				},
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				// First check if user exists (should return not found)
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(db.User{}, pgx.ErrNoRows)

				newAccount := db.Account{ID: uuid.New(), Name: "New Account"}
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(newAccount, nil)

				newUser := db.User{ID: uuid.New(), AccountID: newAccount.ID, Email: email}
				mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(newUser, nil)

				newWorkspace := db.Workspace{ID: uuid.New(), Name: "Default", AccountID: newAccount.ID}
				mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(newWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name: "account creation fails",
			params: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				// First check if user exists (should return not found)
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(db.User{}, pgx.ErrNoRows)
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(db.Account{}, errors.New("account creation error"))
			},
			wantErr:     true,
			errorString: "failed to create account",
		},
		{
			name: "user creation fails",
			params: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				// First check if user exists (should return not found)
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(db.User{}, pgx.ErrNoRows)

				newAccount := db.Account{ID: uuid.New(), Name: "New Account"}
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(newAccount, nil)
				mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(db.User{}, errors.New("user creation error"))
			},
			wantErr:     true,
			errorString: "failed to create user",
		},
		{
			name: "workspace creation fails",
			params: params.CreateAccountParams{
				Name:        "New Account",
				AccountType: "merchant",
			},
			web3AuthID: web3AuthID,
			email:      email,
			setupMocks: func() {
				// First check if user exists (should return not found)
				mockQuerier.EXPECT().GetUserByWeb3AuthID(ctx, gomock.Any()).Return(db.User{}, pgx.ErrNoRows)

				newAccount := db.Account{ID: uuid.New(), Name: "New Account"}
				mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(newAccount, nil)

				newUser := db.User{ID: uuid.New(), AccountID: newAccount.ID, Email: email}
				mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(newUser, nil)

				mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(db.Workspace{}, errors.New("workspace creation error"))
			},
			wantErr:     true,
			errorString: "failed to create workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			// Note: This tests the private method indirectly through SignInOrRegisterAccount
			result, err := service.SignInOrRegisterAccount(ctx, tt.params, tt.web3AuthID, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsNewUser)
				assert.NotNil(t, result.Account)
				assert.NotNil(t, result.User)
				assert.Len(t, result.Workspaces, 1)
				assert.Equal(t, "Default", result.Workspaces[0].Name)
			}
		})
	}
}

// TestAccountService_EdgeCases tests various edge cases and boundary conditions
func TestAccountService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewAccountService(mockQuerier)
	ctx := context.Background()

	t.Run("nil metadata is handled correctly", func(t *testing.T) {
		params := params.CreateAccountParams{
			Name:        "Test Account",
			AccountType: "merchant",
			Metadata:    nil,
		}

		expectedAccount := db.Account{
			ID:          uuid.New(),
			Name:        "Test Account",
			AccountType: "merchant",
		}

		mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(expectedAccount, nil)

		account, err := service.CreateAccount(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, account)
	})

	t.Run("empty metadata map is handled correctly", func(t *testing.T) {
		params := params.CreateAccountParams{
			Name:        "Test Account",
			AccountType: "merchant",
			Metadata:    map[string]interface{}{},
		}

		expectedAccount := db.Account{
			ID:          uuid.New(),
			Name:        "Test Account",
			AccountType: "merchant",
		}

		mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(expectedAccount, nil)

		account, err := service.CreateAccount(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, account)
	})

	t.Run("update with nil metadata preserves existing", func(t *testing.T) {
		testID := uuid.New()
		existingMetadata, _ := json.Marshal(map[string]interface{}{"existing": "data"})
		existingAccount := db.Account{
			ID:       testID,
			Name:     "Test Account",
			Metadata: existingMetadata,
		}

		params := params.UpdateAccountParams{
			ID:       testID,
			Name:     aws.String("Updated Name"),
			Metadata: nil, // Should preserve existing metadata
		}

		mockQuerier.EXPECT().GetAccount(ctx, testID).Return(existingAccount, nil)
		mockQuerier.EXPECT().UpdateAccount(ctx, gomock.Any()).Return(existingAccount, nil)

		account, err := service.UpdateAccount(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, account)
	})

	t.Run("empty string fields are handled correctly", func(t *testing.T) {
		params := params.CreateAccountParams{
			Name:         "Test Account",
			AccountType:  "merchant",
			BusinessName: "",
			WebsiteURL:   "",
			SupportEmail: "",
		}

		expectedAccount := db.Account{
			ID:          uuid.New(),
			Name:        "Test Account",
			AccountType: "merchant",
		}

		mockQuerier.EXPECT().CreateAccount(ctx, gomock.Any()).Return(expectedAccount, nil)

		account, err := service.CreateAccount(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, account)
	})
}
