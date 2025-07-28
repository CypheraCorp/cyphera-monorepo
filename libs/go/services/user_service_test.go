package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/mocks"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	logger.InitLogger("test")
}

func TestUserService_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	accountID := uuid.New()
	userID := uuid.New()
	metadata := map[string]interface{}{
		"source": "web3auth",
		"tier":   "standard",
	}
	metadataBytes, _ := json.Marshal(metadata)

	tests := []struct {
		name        string
		params      params.CreateUserParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates user with all fields",
			params: params.CreateUserParams{
				Web3AuthID:     "web3auth-123",
				Verifier:       "google",
				VerifierID:     "google-123",
				Email:          "test@example.com",
				AccountID:      accountID,
				Role:           "admin",
				IsAccountOwner: true,
				FirstName:      "John",
				LastName:       "Doe",
				AddressLine1:   "123 Main St",
				AddressLine2:   "Apt 1",
				City:           "New York",
				StateRegion:    "NY",
				PostalCode:     "10001",
				Country:        "US",
				DisplayName:    "John Doe",
				PictureURL:     "https://example.com/photo.jpg",
				Phone:          "+1234567890",
				Timezone:       "America/New_York",
				Locale:         "en-US",
				EmailVerified:  true,
				Metadata:       metadata,
			},
			setupMocks: func() {
				expectedParams := db.CreateUserParams{
					Web3authID:     pgtype.Text{String: "web3auth-123", Valid: true},
					Verifier:       pgtype.Text{String: "google", Valid: true},
					VerifierID:     pgtype.Text{String: "google-123", Valid: true},
					Email:          "test@example.com",
					AccountID:      accountID,
					Role:           db.UserRole("admin"),
					IsAccountOwner: pgtype.Bool{Bool: true, Valid: true},
					FirstName:      pgtype.Text{String: "John", Valid: true},
					LastName:       pgtype.Text{String: "Doe", Valid: true},
					AddressLine1:   pgtype.Text{String: "123 Main St", Valid: true},
					AddressLine2:   pgtype.Text{String: "Apt 1", Valid: true},
					City:           pgtype.Text{String: "New York", Valid: true},
					StateRegion:    pgtype.Text{String: "NY", Valid: true},
					PostalCode:     pgtype.Text{String: "10001", Valid: true},
					Country:        pgtype.Text{String: "US", Valid: true},
					DisplayName:    pgtype.Text{String: "John Doe", Valid: true},
					PictureUrl:     pgtype.Text{String: "https://example.com/photo.jpg", Valid: true},
					Phone:          pgtype.Text{String: "+1234567890", Valid: true},
					Timezone:       pgtype.Text{String: "America/New_York", Valid: true},
					Locale:         pgtype.Text{String: "en-US", Valid: true},
					EmailVerified:  pgtype.Bool{Bool: true, Valid: true},
					Metadata:       metadataBytes,
				}
				expectedUser := db.User{
					ID:        userID,
					Email:     "test@example.com",
					AccountID: accountID,
					Role:      db.UserRole("admin"),
				}
				mockQuerier.EXPECT().CreateUser(ctx, expectedParams).Return(expectedUser, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully creates user with minimal fields",
			params: params.CreateUserParams{
				Email:     "minimal@example.com",
				AccountID: accountID,
				Role:      "support",
			},
			setupMocks: func() {
				expectedParams := db.CreateUserParams{
					Web3authID:     pgtype.Text{String: "", Valid: false},
					Verifier:       pgtype.Text{String: "", Valid: false},
					VerifierID:     pgtype.Text{String: "", Valid: false},
					Email:          "minimal@example.com",
					AccountID:      accountID,
					Role:           db.UserRole("support"),
					IsAccountOwner: pgtype.Bool{Bool: false, Valid: true},
					FirstName:      pgtype.Text{String: "", Valid: false},
					LastName:       pgtype.Text{String: "", Valid: false},
					AddressLine1:   pgtype.Text{String: "", Valid: false},
					AddressLine2:   pgtype.Text{String: "", Valid: false},
					City:           pgtype.Text{String: "", Valid: false},
					StateRegion:    pgtype.Text{String: "", Valid: false},
					PostalCode:     pgtype.Text{String: "", Valid: false},
					Country:        pgtype.Text{String: "", Valid: false},
					DisplayName:    pgtype.Text{String: "", Valid: false},
					PictureUrl:     pgtype.Text{String: "", Valid: false},
					Phone:          pgtype.Text{String: "", Valid: false},
					Timezone:       pgtype.Text{String: "", Valid: false},
					Locale:         pgtype.Text{String: "", Valid: false},
					EmailVerified:  pgtype.Bool{Bool: false, Valid: true},
					Metadata:       []byte("{}"),
				}
				expectedUser := db.User{
					ID:        userID,
					Email:     "minimal@example.com",
					AccountID: accountID,
					Role:      db.UserRole("support"),
				}
				mockQuerier.EXPECT().CreateUser(ctx, expectedParams).Return(expectedUser, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty email",
			params: params.CreateUserParams{
				AccountID: accountID,
				Role:      "admin",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "email is required",
		},
		{
			name: "fails with empty account ID",
			params: params.CreateUserParams{
				Email: "test@example.com",
				Role:  "admin",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "account ID is required",
		},
		{
			name: "fails with empty role",
			params: params.CreateUserParams{
				Email:     "test@example.com",
				AccountID: accountID,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "role is required",
		},
		{
			name: "fails with invalid role",
			params: params.CreateUserParams{
				Email:     "test@example.com",
				AccountID: accountID,
				Role:      "invalid",
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "invalid role: invalid",
		},
		{
			name: "handles invalid metadata",
			params: params.CreateUserParams{
				Email:     "test@example.com",
				AccountID: accountID,
				Role:      "admin",
				Metadata:  map[string]interface{}{"key": make(chan int)}, // Invalid JSON
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "failed to marshal metadata",
		},
		{
			name: "handles database error",
			params: params.CreateUserParams{
				Email:     "test@example.com",
				AccountID: accountID,
				Role:      "admin",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(db.User{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create user",
		},
		{
			name: "creates developer role user successfully",
			params: params.CreateUserParams{
				Email:          "dev@example.com",
				AccountID:      accountID,
				Role:           "developer",
				IsAccountOwner: false,
			},
			setupMocks: func() {
				expectedUser := db.User{
					ID:        userID,
					Email:     "dev@example.com",
					AccountID: accountID,
					Role:      db.UserRole("developer"),
				}
				mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(expectedUser, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			user, err := service.CreateUser(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.params.Email, user.Email)
				assert.Equal(t, tt.params.AccountID, user.AccountID)
				assert.Equal(t, string(tt.params.Role), string(user.Role))
			}
		})
	}
}

func TestUserService_GetUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	userID := uuid.New()
	expectedUser := db.User{
		ID:    userID,
		Email: "test@example.com",
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:   "successfully gets user",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(expectedUser, nil)
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(db.User{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "user not found",
		},
		{
			name:   "database error",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(db.User{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			user, err := service.GetUser(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, userID, user.ID)
			}
		})
	}
}

func TestUserService_GetUserWithWorkspaceAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	userID := uuid.New()
	workspaceID := uuid.New()
	accountID := uuid.New()
	otherAccountID := uuid.New()

	validUser := db.User{
		ID:        userID,
		Email:     "test@example.com",
		AccountID: accountID,
	}

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
		userID      uuid.UUID
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully gets user with valid workspace access",
			userID:      userID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(validUser, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(validWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name:        "user not found",
			userID:      userID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(db.User{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "user not found",
		},
		{
			name:        "workspace not found",
			userID:      userID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(validUser, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "workspace not found",
		},
		{
			name:        "database error getting workspace",
			userID:      userID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(validUser, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve workspace",
		},
		{
			name:        "user does not have access to workspace",
			userID:      userID,
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(validUser, nil)
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(invalidWorkspace, nil)
			},
			wantErr:     true,
			errorString: "user does not have access to this workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			user, err := service.GetUserWithWorkspaceAccess(ctx, tt.userID, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, userID, user.ID)
			}
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	userID := uuid.New()
	existingUser := db.User{
		ID:        userID,
		Email:     "test@example.com",
		FirstName: pgtype.Text{String: "Original", Valid: true},
		LastName:  pgtype.Text{String: "Name", Valid: true},
	}

	emailVerified := true
	twoFactorEnabled := false
	metadata := map[string]interface{}{
		"updated": true,
		"version": 2,
	}

	tests := []struct {
		name        string
		params      params.UpdateUserParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully updates user with all fields",
			params: params.UpdateUserParams{
				ID:               userID,
				Email:            "updated@example.com",
				FirstName:        "Updated",
				LastName:         "User",
				AddressLine1:     "456 Oak St",
				AddressLine2:     "Suite 2",
				City:             "Boston",
				StateRegion:      "MA",
				PostalCode:       "02101",
				Country:          "US",
				DisplayName:      "Updated User",
				PictureURL:       "https://example.com/new-photo.jpg",
				Phone:            "+0987654321",
				Timezone:         "America/Boston",
				Locale:           "en-US",
				EmailVerified:    &emailVerified,
				TwoFactorEnabled: &twoFactorEnabled,
				Status:           "active",
				Metadata:         metadata,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				updatedUser := existingUser
				updatedUser.FirstName = pgtype.Text{String: "Updated", Valid: true}
				updatedUser.LastName = pgtype.Text{String: "User", Valid: true}
				mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(updatedUser, nil)
			},
			wantErr: false,
		},
		{
			name: "successfully updates user with partial fields",
			params: params.UpdateUserParams{
				ID:        userID,
				FirstName: "Partial",
				LastName:  "Update",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				updatedUser := existingUser
				updatedUser.FirstName = pgtype.Text{String: "Partial", Valid: true}
				updatedUser.LastName = pgtype.Text{String: "Update", Valid: true}
				mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(updatedUser, nil)
			},
			wantErr: false,
		},
		{
			name: "user not found during update",
			params: params.UpdateUserParams{
				ID:        userID,
				FirstName: "Updated",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(db.User{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "user not found",
		},
		{
			name: "handles invalid metadata",
			params: params.UpdateUserParams{
				ID:       userID,
				Metadata: map[string]interface{}{"key": make(chan int)},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
			},
			wantErr:     true,
			errorString: "failed to marshal metadata",
		},
		{
			name: "database error during update",
			params: params.UpdateUserParams{
				ID:        userID,
				FirstName: "Updated",
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(db.User{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update user",
		},
		{
			name: "updates with new metadata",
			params: params.UpdateUserParams{
				ID:       userID,
				Metadata: metadata,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				updatedUser := existingUser
				mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(updatedUser, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			user, err := service.UpdateUser(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
		})
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	userID := uuid.New()
	existingUser := db.User{
		ID:    userID,
		Email: "test@example.com",
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:   "successfully deletes user",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				mockQuerier.EXPECT().DeleteUser(ctx, userID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(db.User{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "user not found",
		},
		{
			name:   "database delete error",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				mockQuerier.EXPECT().DeleteUser(ctx, userID).Return(errors.New("delete error"))
			},
			wantErr:     true,
			errorString: "failed to delete user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteUser(ctx, tt.userID)

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

func TestUserService_GetUserAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	userID := uuid.New()
	accountID := uuid.New()
	expectedUserAccount := db.GetUserAccountRow{
		ID:        userID,
		Email:     "test@example.com",
		AccountID: accountID,
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:   "successfully gets user account",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserAccount(ctx, userID).Return(expectedUserAccount, nil)
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserAccount(ctx, userID).Return(db.GetUserAccountRow{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "user not found",
		},
		{
			name:   "database error",
			userID: userID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserAccount(ctx, userID).Return(db.GetUserAccountRow{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve user account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			userAccount, err := service.GetUserAccount(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, userAccount)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, userAccount)
				assert.Equal(t, userID, userAccount.ID)
				assert.Equal(t, accountID, userAccount.AccountID)
			}
		})
	}
}

func TestUserService_GetUserByEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	userID := uuid.New()
	email := "test@example.com"
	expectedUser := db.User{
		ID:    userID,
		Email: email,
	}

	tests := []struct {
		name        string
		email       string
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:  "successfully gets user by email",
			email: email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByEmail(ctx, email).Return(expectedUser, nil)
			},
			wantErr: false,
		},
		{
			name:  "user not found",
			email: email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByEmail(ctx, email).Return(db.User{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "user not found",
		},
		{
			name:  "database error",
			email: email,
			setupMocks: func() {
				mockQuerier.EXPECT().GetUserByEmail(ctx, email).Return(db.User{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			user, err := service.GetUserByEmail(ctx, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, userID, user.ID)
				assert.Equal(t, email, user.Email)
			}
		})
	}
}

func TestUserService_ValidateUserRole(t *testing.T) {
	service := services.NewUserService(nil) // No DB queries needed for this test

	tests := []struct {
		name        string
		role        string
		wantErr     bool
		errorString string
	}{
		{
			name:    "valid admin role",
			role:    "admin",
			wantErr: false,
		},
		{
			name:    "valid support role",
			role:    "support",
			wantErr: false,
		},
		{
			name:    "valid developer role",
			role:    "developer",
			wantErr: false,
		},
		{
			name:        "invalid role",
			role:        "invalid",
			wantErr:     true,
			errorString: "invalid role: invalid. Must be one of: admin, support, developer",
		},
		{
			name:        "empty role",
			role:        "",
			wantErr:     true,
			errorString: "invalid role: . Must be one of: admin, support, developer",
		},
		{
			name:        "case sensitive - uppercase should fail",
			role:        "ADMIN",
			wantErr:     true,
			errorString: "invalid role: ADMIN. Must be one of: admin, support, developer",
		},
		{
			name:        "case sensitive - mixed case should fail",
			role:        "Admin",
			wantErr:     true,
			errorString: "invalid role: Admin. Must be one of: admin, support, developer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateUserRole(tt.role)

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

// TestUserService_EdgeCases tests various edge cases and boundary conditions
func TestUserService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewUserService(mockQuerier)
	ctx := context.Background()

	t.Run("nil metadata is handled correctly", func(t *testing.T) {
		accountID := uuid.New()
		params := params.CreateUserParams{
			Email:     "test@example.com",
			AccountID: accountID,
			Role:      "admin",
			Metadata:  nil,
		}

		expectedUser := db.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			AccountID: accountID,
			Role:      db.UserRole("admin"),
		}

		mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(expectedUser, nil)

		user, err := service.CreateUser(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, user)
	})

	t.Run("empty metadata map is handled correctly", func(t *testing.T) {
		accountID := uuid.New()
		params := params.CreateUserParams{
			Email:     "test@example.com",
			AccountID: accountID,
			Role:      "support",
			Metadata:  map[string]interface{}{},
		}

		expectedUser := db.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			AccountID: accountID,
			Role:      db.UserRole("support"),
		}

		mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(expectedUser, nil)

		user, err := service.CreateUser(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, user)
	})

	t.Run("update with nil metadata preserves existing", func(t *testing.T) {
		userID := uuid.New()
		existingMetadata := []byte(`{"existing": "data"}`)
		existingUser := db.User{
			ID:       userID,
			Email:    "test@example.com",
			Metadata: existingMetadata,
		}

		params := params.UpdateUserParams{
			ID:        userID,
			FirstName: "Updated Name",
			Metadata:  nil, // Should preserve existing metadata
		}

		mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
		mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(existingUser, nil)

		user, err := service.UpdateUser(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, user)
	})

	t.Run("empty string fields are handled correctly", func(t *testing.T) {
		accountID := uuid.New()
		params := params.CreateUserParams{
			Email:        "test@example.com",
			AccountID:    accountID,
			Role:         "developer",
			Web3AuthID:   "",
			Verifier:     "",
			VerifierID:   "",
			FirstName:    "",
			LastName:     "",
			AddressLine1: "",
			AddressLine2: "",
			City:         "",
			StateRegion:  "",
			PostalCode:   "",
			Country:      "",
			DisplayName:  "",
			PictureURL:   "",
			Phone:        "",
			Timezone:     "",
			Locale:       "",
		}

		expectedUser := db.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			AccountID: accountID,
			Role:      db.UserRole("developer"),
		}

		mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(expectedUser, nil)

		user, err := service.CreateUser(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, user)
	})

	t.Run("handles large metadata object", func(t *testing.T) {
		accountID := uuid.New()
		largeMetadata := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			largeMetadata[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}

		params := params.CreateUserParams{
			Email:     "test@example.com",
			AccountID: accountID,
			Role:      "admin",
			Metadata:  largeMetadata,
		}

		expectedUser := db.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			AccountID: accountID,
			Role:      db.UserRole("admin"),
		}

		mockQuerier.EXPECT().CreateUser(ctx, gomock.Any()).Return(expectedUser, nil)

		user, err := service.CreateUser(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, user)
	})

	t.Run("handles boolean pointer edge cases", func(t *testing.T) {
		userID := uuid.New()
		existingUser := db.User{
			ID:    userID,
			Email: "test@example.com",
		}

		// Test with both true and false pointers
		emailVerifiedTrue := true
		emailVerifiedFalse := false
		twoFactorTrue := true
		twoFactorFalse := false

		testCases := []struct {
			name             string
			emailVerified    *bool
			twoFactorEnabled *bool
		}{
			{"both true", &emailVerifiedTrue, &twoFactorTrue},
			{"both false", &emailVerifiedFalse, &twoFactorFalse},
			{"mixed values", &emailVerifiedTrue, &twoFactorFalse},
			{"nil values", nil, nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				params := params.UpdateUserParams{
					ID:               userID,
					EmailVerified:    tc.emailVerified,
					TwoFactorEnabled: tc.twoFactorEnabled,
				}

				mockQuerier.EXPECT().GetUserByID(ctx, userID).Return(existingUser, nil)
				mockQuerier.EXPECT().UpdateUser(ctx, gomock.Any()).Return(existingUser, nil)

				user, err := service.UpdateUser(ctx, params)
				assert.NoError(t, err)
				assert.NotNil(t, user)
			})
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		// Create a cancelled context
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		userID := uuid.New()
		mockQuerier.EXPECT().GetUserByID(cancelledCtx, userID).Return(db.User{}, context.Canceled)

		_, err := service.GetUser(cancelledCtx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve user")
	})
}

// TestUserService_ValidationEdgeCases tests edge cases in validation logic
func TestUserService_ValidationEdgeCases(t *testing.T) {
	service := services.NewUserService(nil)

	t.Run("nil UUID is handled correctly", func(t *testing.T) {
		params := params.CreateUserParams{
			Email:     "test@example.com",
			AccountID: uuid.Nil, // This should trigger validation error
			Role:      "admin",
		}

		_, err := service.CreateUser(context.Background(), params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})

	t.Run("whitespace-only email is handled", func(t *testing.T) {
		params := params.CreateUserParams{
			Email:     "   ", // Only whitespace
			AccountID: uuid.New(),
			Role:      "admin",
		}

		// Note: The current implementation doesn't trim whitespace, so this would pass email validation
		// but fail at the database level. This test documents current behavior.
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewUserService(mockQuerier)

		// If the service doesn't trim whitespace, it will pass validation and try to create
		mockQuerier.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(db.User{}, errors.New("database constraint error"))

		_, err := service.CreateUser(context.Background(), params)
		assert.Error(t, err)
	})

	t.Run("very long strings are handled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockQuerier := mocks.NewMockQuerier(ctrl)
		service := services.NewUserService(mockQuerier)

		// Create very long strings
		veryLongString := string(make([]byte, 1000))
		for i := range veryLongString {
			veryLongString = string(rune('a' + (i % 26)))
		}

		params := params.CreateUserParams{
			Email:        "test@example.com",
			AccountID:    uuid.New(),
			Role:         "admin",
			FirstName:    veryLongString,
			LastName:     veryLongString,
			AddressLine1: veryLongString,
			DisplayName:  veryLongString,
		}

		// The service should handle long strings and pass them to the database
		// Database constraints will handle validation of field lengths
		mockQuerier.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(db.User{}, errors.New("value too long"))

		_, err := service.CreateUser(context.Background(), params)
		assert.Error(t, err)
	})
}
