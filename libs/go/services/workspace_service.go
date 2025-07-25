package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
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

// CreateWorkspaceParams contains parameters for creating a workspace
type CreateWorkspaceParams struct {
	Name         string
	Description  string
	BusinessName string
	BusinessType string
	WebsiteURL   string
	SupportEmail string
	SupportPhone string
	AccountID    uuid.UUID
	Metadata     map[string]interface{}
	Livemode     bool
}

// CreateWorkspace creates a new workspace
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, params CreateWorkspaceParams) (*db.Workspace, error) {
	// Validate required fields
	if params.Name == "" {
		return nil, fmt.Errorf("workspace name is required")
	}
	if params.AccountID == uuid.Nil {
		return nil, fmt.Errorf("account ID is required")
	}

	// Convert metadata to JSON
	metadataJSON := []byte("{}")
	if params.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(params.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Create workspace
	workspace, err := s.queries.CreateWorkspace(ctx, db.CreateWorkspaceParams{
		Name:         params.Name,
		AccountID:    params.AccountID,
		Description:  pgtype.Text{String: params.Description, Valid: params.Description != ""},
		BusinessName: pgtype.Text{String: params.BusinessName, Valid: params.BusinessName != ""},
		BusinessType: pgtype.Text{String: params.BusinessType, Valid: params.BusinessType != ""},
		WebsiteUrl:   pgtype.Text{String: params.WebsiteURL, Valid: params.WebsiteURL != ""},
		SupportEmail: pgtype.Text{String: params.SupportEmail, Valid: params.SupportEmail != ""},
		SupportPhone: pgtype.Text{String: params.SupportPhone, Valid: params.SupportPhone != ""},
		Metadata:     metadataJSON,
		Livemode:     pgtype.Bool{Bool: params.Livemode, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create workspace", 
			zap.String("name", params.Name),
			zap.String("account_id", params.AccountID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	s.logger.Info("Workspace created successfully",
		zap.String("workspace_id", workspace.ID.String()),
		zap.String("name", workspace.Name))

	return &workspace, nil
}

// UpdateWorkspaceParams contains parameters for updating a workspace
type UpdateWorkspaceParams struct {
	ID           uuid.UUID
	Name         *string
	Description  *string
	BusinessName *string
	BusinessType *string
	WebsiteURL   *string
	SupportEmail *string
	SupportPhone *string
	Metadata     map[string]interface{}
	Livemode     *bool
}

// UpdateWorkspace updates an existing workspace
func (s *WorkspaceService) UpdateWorkspace(ctx context.Context, params UpdateWorkspaceParams) (*db.Workspace, error) {
	// First check if workspace exists
	_, err := s.queries.GetWorkspace(ctx, params.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, fmt.Errorf("failed to verify workspace: %w", err)
	}

	// Prepare update params
	updateParams := db.UpdateWorkspaceParams{
		ID: params.ID,
	}

	// Set optional fields
	if params.Name != nil {
		updateParams.Name = *params.Name
	}
	if params.Description != nil {
		updateParams.Description = pgtype.Text{String: *params.Description, Valid: true}
	}
	if params.BusinessName != nil {
		updateParams.BusinessName = pgtype.Text{String: *params.BusinessName, Valid: true}
	}
	if params.BusinessType != nil {
		updateParams.BusinessType = pgtype.Text{String: *params.BusinessType, Valid: true}
	}
	if params.WebsiteURL != nil {
		updateParams.WebsiteUrl = pgtype.Text{String: *params.WebsiteURL, Valid: true}
	}
	if params.SupportEmail != nil {
		updateParams.SupportEmail = pgtype.Text{String: *params.SupportEmail, Valid: true}
	}
	if params.SupportPhone != nil {
		updateParams.SupportPhone = pgtype.Text{String: *params.SupportPhone, Valid: true}
	}
	if params.Livemode != nil {
		updateParams.Livemode = pgtype.Bool{Bool: *params.Livemode, Valid: true}
	}

	// Convert metadata to JSON if provided
	if params.Metadata != nil {
		metadataJSON, err := json.Marshal(params.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		updateParams.Metadata = metadataJSON
	}

	// Update workspace
	workspace, err := s.queries.UpdateWorkspace(ctx, updateParams)
	if err != nil {
		s.logger.Error("Failed to update workspace", 
			zap.String("workspace_id", params.ID.String()),
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
type WorkspaceStats struct {
	TotalCustomers     int64
	TotalProducts      int64
	TotalSubscriptions int64
	ActiveSubscriptions int64
}

// GetWorkspaceStats retrieves statistics for a workspace
func (s *WorkspaceService) GetWorkspaceStats(ctx context.Context, workspaceID uuid.UUID) (*WorkspaceStats, error) {
	// This is a placeholder - you would implement actual queries here
	// For now, returning empty stats
	stats := &WorkspaceStats{
		TotalCustomers:     0,
		TotalProducts:      0,
		TotalSubscriptions: 0,
		ActiveSubscriptions: 0,
	}

	// TODO: Implement actual statistics queries
	// stats.TotalCustomers = s.queries.CountCustomersByWorkspace(ctx, workspaceID)
	// stats.TotalProducts = s.queries.CountProductsByWorkspace(ctx, workspaceID)
	// etc.

	return stats, nil
}