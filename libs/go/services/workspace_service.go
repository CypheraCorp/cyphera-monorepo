package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// WorkspaceService handles business logic for workspace operations
type WorkspaceService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewWorkspaceService creates a new workspace service
func NewWorkspaceService(queries db.Querier) *WorkspaceService {
	return &WorkspaceService{
		queries: queries,
		logger:  logger.Log,
	}
}

// GetWorkspace retrieves a workspace by ID
func (s *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID uuid.UUID) (*db.Workspace, error) {
	workspace, err := s.queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workspace not found")
		}
		s.logger.Error("Failed to get workspace",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve workspace: %w", err)
	}

	return &workspace, nil
}

// ListWorkspacesByAccount retrieves all workspaces for a given account
func (s *WorkspaceService) ListWorkspacesByAccount(ctx context.Context, accountID uuid.UUID) ([]db.Workspace, error) {
	workspaces, err := s.queries.ListWorkspacesByAccountID(ctx, accountID)
	if err != nil {
		s.logger.Error("Failed to list workspaces by account",
			zap.String("account_id", accountID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	return workspaces, nil
}

// GetAccountByWorkspace retrieves account information for a workspace
func (s *WorkspaceService) GetAccountByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*db.Account, error) {
	account, err := s.queries.GetAccount(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("account not found for workspace")
		}
		s.logger.Error("Failed to get account for workspace",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	return &account, nil
}

// ListAllWorkspaces retrieves all workspaces (admin function)
func (s *WorkspaceService) ListAllWorkspaces(ctx context.Context) ([]db.Workspace, error) {
	workspaces, err := s.queries.GetAllWorkspaces(ctx)
	if err != nil {
		s.logger.Error("Failed to list all workspaces", zap.Error(err))
		return nil, fmt.Errorf("failed to list all workspaces: %w", err)
	}

	return workspaces, nil
}

// CreateWorkspace creates a new workspace
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, createParams params.CreateWorkspaceParams) (*db.Workspace, error) {
	// Validate required fields
	if createParams.Name == "" {
		return nil, fmt.Errorf("workspace name is required")
	}
	if createParams.AccountID == uuid.Nil {
		return nil, fmt.Errorf("account ID is required")
	}

	// Convert metadata to JSON
	metadataJSON := []byte("{}")
	if createParams.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(createParams.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Create workspace
	workspace, err := s.queries.CreateWorkspace(ctx, db.CreateWorkspaceParams{
		Name:         createParams.Name,
		AccountID:    createParams.AccountID,
		Description:  pgtype.Text{String: createParams.Description, Valid: createParams.Description != ""},
		BusinessName: pgtype.Text{String: createParams.BusinessName, Valid: createParams.BusinessName != ""},
		BusinessType: pgtype.Text{String: createParams.BusinessType, Valid: createParams.BusinessType != ""},
		WebsiteUrl:   pgtype.Text{String: createParams.WebsiteURL, Valid: createParams.WebsiteURL != ""},
		SupportEmail: pgtype.Text{String: createParams.SupportEmail, Valid: createParams.SupportEmail != ""},
		SupportPhone: pgtype.Text{String: createParams.SupportPhone, Valid: createParams.SupportPhone != ""},
		Metadata:     metadataJSON,
		Livemode:     pgtype.Bool{Bool: createParams.Livemode, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create workspace",
			zap.String("name", createParams.Name),
			zap.String("account_id", createParams.AccountID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	s.logger.Info("Workspace created successfully",
		zap.String("workspace_id", workspace.ID.String()),
		zap.String("name", workspace.Name))

	return &workspace, nil
}

// UpdateWorkspace updates an existing workspace
func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, updateParams params.UpdateWorkspaceParams) (*db.Workspace, error) {
	// First check if workspace exists
	_, err := s.queries.GetWorkspace(ctx, updateParams.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, fmt.Errorf("failed to verify workspace: %w", err)
	}

	// Prepare update params
	dbUpdateParams := db.UpdateWorkspaceParams{
		ID: updateParams.ID,
	}

	// Set optional fields
	if updateParams.Name != nil {
		dbUpdateParams.Name = *updateParams.Name
	}
	if updateParams.Description != nil {
		dbUpdateParams.Description = pgtype.Text{String: *updateParams.Description, Valid: true}
	}
	if updateParams.BusinessName != nil {
		dbUpdateParams.BusinessName = pgtype.Text{String: *updateParams.BusinessName, Valid: true}
	}
	if updateParams.BusinessType != nil {
		dbUpdateParams.BusinessType = pgtype.Text{String: *updateParams.BusinessType, Valid: true}
	}
	if updateParams.WebsiteURL != nil {
		dbUpdateParams.WebsiteUrl = pgtype.Text{String: *updateParams.WebsiteURL, Valid: true}
	}
	if updateParams.SupportEmail != nil {
		dbUpdateParams.SupportEmail = pgtype.Text{String: *updateParams.SupportEmail, Valid: true}
	}
	if updateParams.SupportPhone != nil {
		dbUpdateParams.SupportPhone = pgtype.Text{String: *updateParams.SupportPhone, Valid: true}
	}
	if updateParams.Livemode != nil {
		dbUpdateParams.Livemode = pgtype.Bool{Bool: *updateParams.Livemode, Valid: true}
	}

	// Convert metadata to JSON if provided
	if updateParams.Metadata != nil {
		metadataJSON, err := json.Marshal(updateParams.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		dbUpdateParams.Metadata = metadataJSON
	}

	// Update workspace
	workspace, err := s.queries.UpdateWorkspace(ctx, dbUpdateParams)
	if err != nil {
		s.logger.Error("Failed to update workspace",
			zap.String("workspace_id", updateParams.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	s.logger.Info("Workspace updated successfully",
		zap.String("workspace_id", workspace.ID.String()))

	return &workspace, nil
}

// DeleteWorkspace deletes a workspace
func (s *WorkspaceService) DeleteWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	// First check if workspace exists
	_, err := s.queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("workspace not found")
		}
		return fmt.Errorf("failed to verify workspace: %w", err)
	}

	// Delete workspace (soft delete)
	err = s.queries.DeleteWorkspace(ctx, workspaceID)
	if err != nil {
		s.logger.Error("Failed to delete workspace",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	s.logger.Info("Workspace deleted successfully",
		zap.String("workspace_id", workspaceID.String()))

	return nil
}

// ValidateWorkspaceAccess checks if a workspace belongs to a given account
func (s *WorkspaceService) ValidateWorkspaceAccess(ctx context.Context, workspaceID, accountID uuid.UUID) error {
	workspace, err := s.queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("workspace not found")
		}
		return fmt.Errorf("failed to retrieve workspace: %w", err)
	}

	if workspace.AccountID != accountID {
		return fmt.Errorf("workspace does not belong to this account")
	}

	return nil
}

// GetWorkspaceStats retrieves statistics for a workspace
func (s *WorkspaceService) GetWorkspaceStats(ctx context.Context, workspaceID uuid.UUID) (*business.WorkspaceStats, error) {
	// This is a placeholder - you would implement actual queries here
	// For now, returning empty stats
	stats := &business.WorkspaceStats{
		TotalCustomers:      0,
		TotalProducts:       0,
		TotalSubscriptions:  0,
		ActiveSubscriptions: 0,
	}

	// TODO: Implement actual statistics queries
	// stats.TotalCustomers = s.queries.CountCustomersByWorkspace(ctx, workspaceID)
	// stats.TotalProducts = s.queries.CountProductsByWorkspace(ctx, workspaceID)
	// etc.

	return stats, nil
}
