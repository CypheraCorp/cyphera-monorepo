package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// UserService handles business logic for user operations
type UserService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewUserService creates a new user service
func NewUserService(queries db.Querier) *UserService {
	return &UserService{
		queries: queries,
		logger:  logger.Log,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, params params.CreateUserParams) (*db.User, error) {
	// Validate required fields
	if params.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if params.AccountID == uuid.Nil {
		return nil, fmt.Errorf("account ID is required")
	}
	if params.Role == "" {
		return nil, fmt.Errorf("role is required")
	}

	// Validate role
	if params.Role != "admin" && params.Role != "support" && params.Role != "developer" {
		return nil, fmt.Errorf("invalid role: %s", params.Role)
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

	// Create user
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Web3authID:     pgtype.Text{String: params.Web3AuthID, Valid: params.Web3AuthID != ""},
		Verifier:       pgtype.Text{String: params.Verifier, Valid: params.Verifier != ""},
		VerifierID:     pgtype.Text{String: params.VerifierID, Valid: params.VerifierID != ""},
		Email:          params.Email,
		AccountID:      params.AccountID,
		Role:           db.UserRole(params.Role),
		IsAccountOwner: pgtype.Bool{Bool: params.IsAccountOwner, Valid: true},
		FirstName:      pgtype.Text{String: params.FirstName, Valid: params.FirstName != ""},
		LastName:       pgtype.Text{String: params.LastName, Valid: params.LastName != ""},
		AddressLine1:   pgtype.Text{String: params.AddressLine1, Valid: params.AddressLine1 != ""},
		AddressLine2:   pgtype.Text{String: params.AddressLine2, Valid: params.AddressLine2 != ""},
		City:           pgtype.Text{String: params.City, Valid: params.City != ""},
		StateRegion:    pgtype.Text{String: params.StateRegion, Valid: params.StateRegion != ""},
		PostalCode:     pgtype.Text{String: params.PostalCode, Valid: params.PostalCode != ""},
		Country:        pgtype.Text{String: params.Country, Valid: params.Country != ""},
		DisplayName:    pgtype.Text{String: params.DisplayName, Valid: params.DisplayName != ""},
		PictureUrl:     pgtype.Text{String: params.PictureURL, Valid: params.PictureURL != ""},
		Phone:          pgtype.Text{String: params.Phone, Valid: params.Phone != ""},
		Timezone:       pgtype.Text{String: params.Timezone, Valid: params.Timezone != ""},
		Locale:         pgtype.Text{String: params.Locale, Valid: params.Locale != ""},
		EmailVerified:  pgtype.Bool{Bool: params.EmailVerified, Valid: true},
		Metadata:       metadataJSON,
	})
	if err != nil {
		s.logger.Error("Failed to create user",
			zap.String("email", params.Email),
			zap.String("account_id", params.AccountID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User created successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email))

	return &user, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*db.User, error) {
	user, err := s.queries.GetUserByID(ctx, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.Error("Failed to get user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return &user, nil
}

// GetUserWithWorkspaceAccess retrieves a user and validates workspace access
func (s *UserService) GetUserWithWorkspaceAccess(ctx context.Context, userID, workspaceID uuid.UUID) (*db.User, error) {
	// Get user
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get workspace to verify account access
	workspace, err := s.queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workspace not found")
		}
		return nil, fmt.Errorf("failed to retrieve workspace: %w", err)
	}

	// Verify user has access to this workspace through account
	if user.AccountID != workspace.AccountID {
		return nil, fmt.Errorf("user does not have access to this workspace")
	}

	return user, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, params params.UpdateUserParams) (*db.User, error) {
	// Verify user exists
	_, err := s.GetUser(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Convert metadata to JSON if provided
	var metadataJSON []byte
	if params.Metadata != nil {
		metadataJSON, err = json.Marshal(params.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Prepare update params
	updateParams := db.UpdateUserParams{
		Email:        params.ID.String(), // The query uses this field for ID
		FirstName:    pgtype.Text{String: params.FirstName, Valid: params.FirstName != ""},
		LastName:     pgtype.Text{String: params.LastName, Valid: params.LastName != ""},
		AddressLine1: pgtype.Text{String: params.AddressLine1, Valid: params.AddressLine1 != ""},
		AddressLine2: pgtype.Text{String: params.AddressLine2, Valid: params.AddressLine2 != ""},
		City:         pgtype.Text{String: params.City, Valid: params.City != ""},
		StateRegion:  pgtype.Text{String: params.StateRegion, Valid: params.StateRegion != ""},
		PostalCode:   pgtype.Text{String: params.PostalCode, Valid: params.PostalCode != ""},
		Country:      pgtype.Text{String: params.Country, Valid: params.Country != ""},
		DisplayName:  pgtype.Text{String: params.DisplayName, Valid: params.DisplayName != ""},
		PictureUrl:   pgtype.Text{String: params.PictureURL, Valid: params.PictureURL != ""},
		Phone:        pgtype.Text{String: params.Phone, Valid: params.Phone != ""},
		Timezone:     pgtype.Text{String: params.Timezone, Valid: params.Timezone != ""},
		Locale:       pgtype.Text{String: params.Locale, Valid: params.Locale != ""},
		Metadata:     metadataJSON,
	}

	// Set optional boolean fields
	if params.EmailVerified != nil {
		updateParams.EmailVerified = pgtype.Bool{Bool: *params.EmailVerified, Valid: true}
	}
	if params.TwoFactorEnabled != nil {
		updateParams.TwoFactorEnabled = pgtype.Bool{Bool: *params.TwoFactorEnabled, Valid: true}
	}

	// Set status if provided
	if params.Status != "" {
		updateParams.Status = db.NullUserStatus{
			UserStatus: db.UserStatus(params.Status),
			Valid:      true,
		}
	}

	// Update user
	user, err := s.queries.UpdateUser(ctx, updateParams)
	if err != nil {
		s.logger.Error("Failed to update user",
			zap.String("user_id", params.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("User updated successfully",
		zap.String("user_id", user.ID.String()))

	return &user, nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// Verify user exists
	_, err := s.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Delete user
	err = s.queries.DeleteUser(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to delete user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.Info("User deleted successfully",
		zap.String("user_id", userID.String()))

	return nil
}

// GetUserAccount retrieves user account information
func (s *UserService) GetUserAccount(ctx context.Context, userID uuid.UUID) (*db.GetUserAccountRow, error) {
	userAccount, err := s.queries.GetUserAccount(ctx, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.Error("Failed to get user account",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve user account: %w", err)
	}

	return &userAccount, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*db.User, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		s.logger.Error("Failed to get user by email",
			zap.String("email", email),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return &user, nil
}

// ValidateUserRole validates if a role is valid
func (s *UserService) ValidateUserRole(role string) error {
	validRoles := []string{"admin", "support", "developer"}
	for _, validRole := range validRoles {
		if role == validRole {
			return nil
		}
	}
	return fmt.Errorf("invalid role: %s. Must be one of: admin, support, developer", role)
}
