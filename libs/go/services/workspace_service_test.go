package services_test

import (
	"context"
	"errors"
	"testing"

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

func TestWorkspaceService_GetWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	testID := uuid.New()
	expectedWorkspace := db.Workspace{
		ID:   testID,
		Name: "Test Workspace",
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully gets workspace",
			workspaceID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(expectedWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name:        "workspace not found",
			workspaceID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(db.Workspace{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "workspace not found",
		},
		{
			name:        "database error",
			workspaceID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(db.Workspace{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			workspace, err := service.GetWorkspace(ctx, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspace)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workspace)
				assert.Equal(t, testID, workspace.ID)
			}
		})
	}
}

func TestWorkspaceService_ListWorkspacesByAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	accountID := uuid.New()
	expectedWorkspaces := []db.Workspace{
		{ID: uuid.New(), Name: "Workspace 1", AccountID: accountID},
		{ID: uuid.New(), Name: "Workspace 2", AccountID: accountID},
	}

	tests := []struct {
		name        string
		accountID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name:      "successfully lists workspaces",
			accountID: accountID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWorkspacesByAccountID(ctx, accountID).Return(expectedWorkspaces, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "returns empty list",
			accountID: accountID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWorkspacesByAccountID(ctx, accountID).Return([]db.Workspace{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "database error",
			accountID: accountID,
			setupMocks: func() {
				mockQuerier.EXPECT().ListWorkspacesByAccountID(ctx, accountID).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to list workspaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			workspaces, err := service.ListWorkspacesByAccount(ctx, tt.accountID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspaces)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, workspaces, tt.wantCount)
			}
		})
	}
}

func TestWorkspaceService_GetAccountByWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	expectedAccount := db.Account{
		ID:   uuid.New(),
		Name: "Test Account",
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully gets account",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, workspaceID).Return(expectedAccount, nil)
			},
			wantErr: false,
		},
		{
			name:        "account not found",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, workspaceID).Return(db.Account{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "account not found for workspace",
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetAccount(ctx, workspaceID).Return(db.Account{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			account, err := service.GetAccountByWorkspace(ctx, tt.workspaceID)

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

func TestWorkspaceService_ListAllWorkspaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	expectedWorkspaces := []db.Workspace{
		{ID: uuid.New(), Name: "Workspace 1"},
		{ID: uuid.New(), Name: "Workspace 2"},
		{ID: uuid.New(), Name: "Workspace 3"},
	}

	tests := []struct {
		name        string
		setupMocks  func()
		wantErr     bool
		errorString string
		wantCount   int
	}{
		{
			name: "successfully lists all workspaces",
			setupMocks: func() {
				mockQuerier.EXPECT().GetAllWorkspaces(ctx).Return(expectedWorkspaces, nil)
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "returns empty list",
			setupMocks: func() {
				mockQuerier.EXPECT().GetAllWorkspaces(ctx).Return([]db.Workspace{}, nil)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "database error",
			setupMocks: func() {
				mockQuerier.EXPECT().GetAllWorkspaces(ctx).Return(nil, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to list all workspaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			workspaces, err := service.ListAllWorkspaces(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspaces)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, workspaces, tt.wantCount)
			}
		})
	}
}

func TestWorkspaceService_CreateWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	accountID := uuid.New()

	tests := []struct {
		name        string
		params      params.CreateWorkspaceParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully creates workspace",
			params: params.CreateWorkspaceParams{
				Name:         "Test Workspace",
				Description:  "Test Description",
				BusinessName: "Test Business",
				AccountID:    accountID,
				Metadata:     map[string]interface{}{"key": "value"},
				Livemode:     false,
			},
			setupMocks: func() {
				expectedWorkspace := db.Workspace{
					ID:        uuid.New(),
					Name:      "Test Workspace",
					AccountID: accountID,
				}
				mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(expectedWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name: "fails with empty name",
			params: params.CreateWorkspaceParams{
				AccountID: accountID,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "workspace name is required",
		},
		{
			name: "fails with nil account ID",
			params: params.CreateWorkspaceParams{
				Name:      "Test Workspace",
				AccountID: uuid.Nil,
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "account ID is required",
		},
		{
			name: "fails with invalid metadata",
			params: params.CreateWorkspaceParams{
				Name:      "Test Workspace",
				AccountID: accountID,
				Metadata:  map[string]interface{}{"key": make(chan int)}, // Invalid JSON
			},
			setupMocks:  func() {},
			wantErr:     true,
			errorString: "failed to marshal metadata",
		},
		{
			name: "handles database error",
			params: params.CreateWorkspaceParams{
				Name:      "Test Workspace",
				AccountID: accountID,
			},
			setupMocks: func() {
				mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(db.Workspace{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to create workspace",
		},
		{
			name: "creates workspace with all fields",
			params: params.CreateWorkspaceParams{
				Name:         "Full Workspace",
				Description:  "Full Description",
				BusinessName: "Full Business",
				BusinessType: "SaaS",
				WebsiteURL:   "https://test.com",
				SupportEmail: "support@test.com",
				SupportPhone: "+1234567890",
				AccountID:    accountID,
				Metadata:     map[string]interface{}{"env": "test"},
				Livemode:     true,
			},
			setupMocks: func() {
				expectedWorkspace := db.Workspace{
					ID:        uuid.New(),
					Name:      "Full Workspace",
					AccountID: accountID,
				}
				mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(expectedWorkspace, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			workspace, err := service.CreateWorkspace(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspace)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workspace)
				assert.Equal(t, tt.params.Name, workspace.Name)
				assert.Equal(t, tt.params.AccountID, workspace.AccountID)
			}
		})
	}
}

func TestWorkspaceService_UpdateWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	testID := uuid.New()
	existingWorkspace := db.Workspace{
		ID:   testID,
		Name: "Original Workspace",
	}

	tests := []struct {
		name        string
		params      params.UpdateWorkspaceParams
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully updates workspace",
			params: params.UpdateWorkspaceParams{
				ID:           testID,
				Name:         workspaceStringPtr("Updated Workspace"),
				Description:  workspaceStringPtr("Updated Description"),
				BusinessName: workspaceStringPtr("Updated Business"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(existingWorkspace, nil)
				updatedWorkspace := existingWorkspace
				updatedWorkspace.Name = "Updated Workspace"
				mockQuerier.EXPECT().UpdateWorkspace(ctx, gomock.Any()).Return(updatedWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name: "workspace not found",
			params: params.UpdateWorkspaceParams{
				ID:   testID,
				Name: workspaceStringPtr("Updated Workspace"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(db.Workspace{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "workspace not found",
		},
		{
			name: "database verification error",
			params: params.UpdateWorkspaceParams{
				ID:   testID,
				Name: workspaceStringPtr("Updated Workspace"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(db.Workspace{}, errors.New("verification error"))
			},
			wantErr:     true,
			errorString: "failed to verify workspace",
		},
		{
			name: "invalid metadata format",
			params: params.UpdateWorkspaceParams{
				ID:       testID,
				Metadata: map[string]interface{}{"key": make(chan int)},
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(existingWorkspace, nil)
			},
			wantErr:     true,
			errorString: "failed to marshal metadata",
		},
		{
			name: "database update error",
			params: params.UpdateWorkspaceParams{
				ID:   testID,
				Name: workspaceStringPtr("Updated Workspace"),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(existingWorkspace, nil)
				mockQuerier.EXPECT().UpdateWorkspace(ctx, gomock.Any()).Return(db.Workspace{}, errors.New("update error"))
			},
			wantErr:     true,
			errorString: "failed to update workspace",
		},
		{
			name: "updates with metadata",
			params: params.UpdateWorkspaceParams{
				ID:       testID,
				Metadata: map[string]interface{}{"new": "metadata"},
				Livemode: boolPtr(true),
			},
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(existingWorkspace, nil)
				updatedWorkspace := existingWorkspace
				mockQuerier.EXPECT().UpdateWorkspace(ctx, gomock.Any()).Return(updatedWorkspace, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			workspace, err := service.UpdateWorkspace(ctx, tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, workspace)
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workspace)
			}
		})
	}
}

func TestWorkspaceService_DeleteWorkspace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	testID := uuid.New()
	existingWorkspace := db.Workspace{
		ID:   testID,
		Name: "Test Workspace",
	}

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "successfully deletes workspace",
			workspaceID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(existingWorkspace, nil)
				mockQuerier.EXPECT().DeleteWorkspace(ctx, testID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "workspace not found",
			workspaceID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(db.Workspace{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "workspace not found",
		},
		{
			name:        "database verification error",
			workspaceID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(db.Workspace{}, errors.New("verification error"))
			},
			wantErr:     true,
			errorString: "failed to verify workspace",
		},
		{
			name:        "database delete error",
			workspaceID: testID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(existingWorkspace, nil)
				mockQuerier.EXPECT().DeleteWorkspace(ctx, testID).Return(errors.New("delete error"))
			},
			wantErr:     true,
			errorString: "failed to delete workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.DeleteWorkspace(ctx, tt.workspaceID)

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

func TestWorkspaceService_ValidateWorkspaceAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	workspaceID := uuid.New()
	accountID := uuid.New()
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
		workspaceID uuid.UUID
		accountID   uuid.UUID
		setupMocks  func()
		wantErr     bool
		errorString string
	}{
		{
			name:        "valid access",
			workspaceID: workspaceID,
			accountID:   accountID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(validWorkspace, nil)
			},
			wantErr: false,
		},
		{
			name:        "workspace not found",
			workspaceID: workspaceID,
			accountID:   accountID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, pgx.ErrNoRows)
			},
			wantErr:     true,
			errorString: "workspace not found",
		},
		{
			name:        "database error",
			workspaceID: workspaceID,
			accountID:   accountID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(db.Workspace{}, errors.New("database error"))
			},
			wantErr:     true,
			errorString: "failed to retrieve workspace",
		},
		{
			name:        "workspace belongs to different account",
			workspaceID: workspaceID,
			accountID:   accountID,
			setupMocks: func() {
				mockQuerier.EXPECT().GetWorkspace(ctx, workspaceID).Return(invalidWorkspace, nil)
			},
			wantErr:     true,
			errorString: "workspace does not belong to this account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := service.ValidateWorkspaceAccess(ctx, tt.workspaceID, tt.accountID)

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

func TestWorkspaceService_GetWorkspaceStats(t *testing.T) {
	service := services.NewWorkspaceService(nil) // No DB queries needed for this placeholder
	ctx := context.Background()

	tests := []struct {
		name        string
		workspaceID uuid.UUID
		wantErr     bool
	}{
		{
			name:        "returns placeholder stats",
			workspaceID: uuid.New(),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := service.GetWorkspaceStats(ctx, tt.workspaceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, stats)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stats)
				// Since this is a placeholder implementation, all stats should be 0
				assert.Equal(t, int64(0), stats.TotalCustomers)
				assert.Equal(t, int64(0), stats.TotalProducts)
				assert.Equal(t, int64(0), stats.TotalSubscriptions)
				assert.Equal(t, int64(0), stats.ActiveSubscriptions)
			}
		})
	}
}

// TestWorkspaceService_EdgeCases tests various edge cases and boundary conditions
func TestWorkspaceService_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := mocks.NewMockQuerier(ctrl)
	service := services.NewWorkspaceService(mockQuerier)
	ctx := context.Background()

	accountID := uuid.New()

	t.Run("nil metadata is handled correctly", func(t *testing.T) {
		params := params.CreateWorkspaceParams{
			Name:      "Test Workspace",
			AccountID: accountID,
			Metadata:  nil,
		}

		expectedWorkspace := db.Workspace{
			ID:        uuid.New(),
			Name:      "Test Workspace",
			AccountID: accountID,
		}

		mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(expectedWorkspace, nil)

		workspace, err := service.CreateWorkspace(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, workspace)
	})

	t.Run("empty metadata map is handled correctly", func(t *testing.T) {
		params := params.CreateWorkspaceParams{
			Name:      "Test Workspace",
			AccountID: accountID,
			Metadata:  map[string]interface{}{},
		}

		expectedWorkspace := db.Workspace{
			ID:        uuid.New(),
			Name:      "Test Workspace",
			AccountID: accountID,
		}

		mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(expectedWorkspace, nil)

		workspace, err := service.CreateWorkspace(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, workspace)
	})

	t.Run("empty string fields are handled correctly", func(t *testing.T) {
		params := params.CreateWorkspaceParams{
			Name:         "Test Workspace",
			AccountID:    accountID,
			Description:  "",
			BusinessName: "",
			WebsiteURL:   "",
			SupportEmail: "",
		}

		expectedWorkspace := db.Workspace{
			ID:        uuid.New(),
			Name:      "Test Workspace",
			AccountID: accountID,
		}

		mockQuerier.EXPECT().CreateWorkspace(ctx, gomock.Any()).Return(expectedWorkspace, nil)

		workspace, err := service.CreateWorkspace(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, workspace)
	})

	t.Run("update with nil pointers is handled correctly", func(t *testing.T) {
		testID := uuid.New()
		existingWorkspace := db.Workspace{
			ID:   testID,
			Name: "Test Workspace",
		}

		params := params.UpdateWorkspaceParams{
			ID:           testID,
			Name:         nil, // Should not update name
			Description:  nil, // Should not update description
			BusinessName: nil, // Should not update business name
			Metadata:     nil, // Should not update metadata
		}

		mockQuerier.EXPECT().GetWorkspace(ctx, testID).Return(existingWorkspace, nil)
		mockQuerier.EXPECT().UpdateWorkspace(ctx, gomock.Any()).Return(existingWorkspace, nil)

		workspace, err := service.UpdateWorkspace(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, workspace)
	})
}

func boolPtr(b bool) *bool {
	return &b
}

func workspaceStringPtr(s string) *string {
	return &s
}
