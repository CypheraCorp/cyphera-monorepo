package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// AccountService handles business logic for account operations
type AccountService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewAccountService creates a new account service
func NewAccountService(queries db.Querier) *AccountService {
	return &AccountService{
		queries: queries,
		logger:  logger.Log,
	}
}

// CreateAccount creates a new account with validation
func (s *AccountService) CreateAccount(ctx context.Context, createParams params.CreateAccountParams) (*db.Account, error) {
	// Validate required fields
	if createParams.Name == "" {
		return nil, fmt.Errorf("account name is required")
	}
	if createParams.AccountType == "" {
		return nil, fmt.Errorf("account type is required")
	}

	// Validate account type
	if createParams.AccountType != constants.AdminRole && createParams.AccountType != constants.MerchantRole {
		return nil, fmt.Errorf("invalid account type: %s. Must be 'admin' or 'merchant'", createParams.AccountType)
	}

	// Convert metadata to JSON
	metadata, err := json.Marshal(createParams.Metadata)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	// Create account
	account, err := s.queries.CreateAccount(ctx, db.CreateAccountParams{
		Name:               createParams.Name,
		AccountType:        db.AccountType(createParams.AccountType),
		BusinessName:       pgtype.Text{String: createParams.BusinessName, Valid: createParams.BusinessName != ""},
		BusinessType:       pgtype.Text{String: createParams.BusinessType, Valid: createParams.BusinessType != ""},
		WebsiteUrl:         pgtype.Text{String: createParams.WebsiteURL, Valid: createParams.WebsiteURL != ""},
		SupportEmail:       pgtype.Text{String: createParams.SupportEmail, Valid: createParams.SupportEmail != ""},
		SupportPhone:       pgtype.Text{String: createParams.SupportPhone, Valid: createParams.SupportPhone != ""},
		FinishedOnboarding: pgtype.Bool{Bool: createParams.FinishedOnboarding, Valid: true},
		Metadata:           metadata,
	})
	if err != nil {
		s.logger.Error("Failed to create account",
			zap.String("name", createParams.Name),
			zap.String("account_type", createParams.AccountType),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	s.logger.Info("Account created successfully",
		zap.String("account_id", account.ID.String()),
		zap.String("name", account.Name))

	return &account, nil
}

// GetAccount retrieves an account by ID
func (s *AccountService) GetAccount(ctx context.Context, accountID uuid.UUID) (*db.Account, error) {
	account, err := s.queries.GetAccount(ctx, accountID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		s.logger.Error("Failed to get account",
			zap.String("account_id", accountID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve account: %w", err)
	}

	return &account, nil
}

// ListAccounts retrieves all accounts
func (s *AccountService) ListAccounts(ctx context.Context) ([]db.Account, error) {
	accounts, err := s.queries.ListAccounts(ctx)
	if err != nil {
		s.logger.Error("Failed to list accounts", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve accounts: %w", err)
	}

	return accounts, nil
}

// UpdateAccount updates an existing account
func (s *AccountService) UpdateAccount(ctx context.Context, updateParams params.UpdateAccountParams) (*db.Account, error) {
	// Verify account exists
	existingAccount, err := s.GetAccount(ctx, updateParams.ID)
	if err != nil {
		return nil, err
	}

	// Convert metadata to JSON if provided
	var metadata []byte
	if updateParams.Metadata != nil {
		metadata, err = json.Marshal(updateParams.Metadata)
		if err != nil {
			return nil, fmt.Errorf("invalid metadata format: %w", err)
		}
	} else {
		metadata = existingAccount.Metadata
	}

	// Use existing values if not provided
	name := updateParams.Name
	if name == nil {
		name = &existingAccount.Name
	}

	accountType := existingAccount.AccountType
	if updateParams.AccountType != nil {
		if *updateParams.AccountType != constants.AdminRole && *updateParams.AccountType != constants.MerchantRole {
			return nil, fmt.Errorf("invalid account type: %s", *updateParams.AccountType)
		}
		accountType = db.AccountType(*updateParams.AccountType)
	}

	// Prepare update params with existing values as defaults
	finishedOnboarding := existingAccount.FinishedOnboarding
	if updateParams.FinishedOnboarding != nil {
		finishedOnboarding = pgtype.Bool{Bool: *updateParams.FinishedOnboarding, Valid: true}
	}

	dbUpdateParams := db.UpdateAccountParams{
		ID:                 updateParams.ID,
		Name:               *name,
		AccountType:        accountType,
		BusinessName:       existingAccount.BusinessName,
		BusinessType:       existingAccount.BusinessType,
		WebsiteUrl:         existingAccount.WebsiteUrl,
		SupportEmail:       existingAccount.SupportEmail,
		SupportPhone:       existingAccount.SupportPhone,
		FinishedOnboarding: finishedOnboarding,
		Metadata:           metadata,
	}

	// Update optional fields if provided
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

	// Update account
	account, err := s.queries.UpdateAccount(ctx, dbUpdateParams)
	if err != nil {
		s.logger.Error("Failed to update account",
			zap.String("account_id", updateParams.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	s.logger.Info("Account updated successfully",
		zap.String("account_id", account.ID.String()))

	return &account, nil
}

// DeleteAccount deletes an account
func (s *AccountService) DeleteAccount(ctx context.Context, accountID uuid.UUID) error {
	// Verify account exists
	_, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return err
	}

	// Delete account
	err = s.queries.DeleteAccount(ctx, accountID)
	if err != nil {
		s.logger.Error("Failed to delete account",
			zap.String("account_id", accountID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete account: %w", err)
	}

	s.logger.Info("Account deleted successfully",
		zap.String("account_id", accountID.String()))

	return nil
}

// ValidateSignInRequest validates sign-in request metadata
func (s *AccountService) ValidateSignInRequest(metadata map[string]interface{}) (string, string, error) {
	// Check for Web3Auth metadata format
	web3authId, ok := metadata["ownerWeb3AuthId"].(string)
	if !ok || web3authId == "" {
		return "", "", fmt.Errorf("ownerWeb3AuthId is required")
	}

	email, ok := metadata["email"].(string)
	if !ok || email == "" {
		return "", "", fmt.Errorf("email is required")
	}

	return web3authId, email, nil
}

// SignInOrRegisterAccount handles both sign-in and registration logic
func (s *AccountService) SignInOrRegisterAccount(ctx context.Context, createParams params.CreateAccountParams, web3authId, email string) (*business.SignInRegisterData, error) {
	// Check if user already exists by Web3Auth ID
	user, err := s.queries.GetUserByWeb3AuthID(ctx, pgtype.Text{String: web3authId, Valid: web3authId != ""})
	if err != nil && err != pgx.ErrNoRows {
		s.logger.Error("Failed to check existing user",
			zap.String("web3auth_id", web3authId),
			zap.Error(err))
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	var result *business.SignInRegisterData

	if err == pgx.ErrNoRows {
		// User doesn't exist, create new account and user
		result, err = s.createNewAccountWithUser(ctx, createParams, web3authId, email)
		if err != nil {
			return nil, err
		}
		result.IsNewUser = true
	} else {
		// User exists, get existing account details
		account, err := s.queries.GetAccount(ctx, user.AccountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get account: %w", err)
		}

		workspaces, err := s.queries.ListWorkspacesByAccountID(ctx, account.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workspaces: %w", err)
		}

		result = &business.SignInRegisterData{
			Account:    &account,
			User:       &user,
			Workspaces: workspaces,
			IsNewUser:  false,
		}
	}

	return result, nil
}

// createNewAccountWithUser creates both account and user in a transaction-like manner
func (s *AccountService) createNewAccountWithUser(ctx context.Context, createParams params.CreateAccountParams, web3authId, email string) (*business.SignInRegisterData, error) {
	// Extract verifier and verifierId from metadata
	var verifier, verifierId string
	if createParams.Metadata != nil {
		if v, ok := createParams.Metadata["verifier"].(string); ok {
			verifier = v
		}
		if v, ok := createParams.Metadata["verifierId"].(string); ok {
			verifierId = v
		}
	}

	// Create account
	account, err := s.CreateAccount(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Create user
	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:          email,
		AccountID:      account.ID,
		Role:           db.UserRoleAdmin,
		IsAccountOwner: pgtype.Bool{Bool: true, Valid: true},
		FirstName:      pgtype.Text{Valid: false},
		LastName:       pgtype.Text{Valid: false},
		AddressLine1:   pgtype.Text{Valid: false},
		AddressLine2:   pgtype.Text{Valid: false},
		City:           pgtype.Text{Valid: false},
		StateRegion:    pgtype.Text{Valid: false},
		PostalCode:     pgtype.Text{Valid: false},
		Country:        pgtype.Text{Valid: false},
		DisplayName:    pgtype.Text{Valid: false},
		PictureUrl:     pgtype.Text{Valid: false},
		Phone:          pgtype.Text{Valid: false},
		Timezone:       pgtype.Text{Valid: false},
		Locale:         pgtype.Text{Valid: false},
		EmailVerified:  pgtype.Bool{Bool: true, Valid: true}, // Assume Web3Auth emails are verified
		Metadata:       []byte("{}"),
		Web3authID:     pgtype.Text{String: web3authId, Valid: web3authId != ""},
		Verifier:       pgtype.Text{String: verifier, Valid: verifier != ""},
		VerifierID:     pgtype.Text{String: verifierId, Valid: verifierId != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create default workspace
	workspace, err := s.queries.CreateWorkspace(ctx, db.CreateWorkspaceParams{
		Name:      "Default",
		AccountID: account.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return &business.SignInRegisterData{
		Account:    account,
		User:       &user,
		Workspaces: []db.Workspace{workspace},
		IsNewUser:  true,
	}, nil
}

// OnboardAccount handles account onboarding process
func (s *AccountService) OnboardAccount(ctx context.Context, onboardParams params.OnboardAccountParams) error {
	// Verify account exists
	_, err := s.GetAccount(ctx, onboardParams.AccountID)
	if err != nil {
		return err
	}

	// Get existing user
	user, err := s.queries.GetUserByID(ctx, onboardParams.UserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	updateParams := params.UpdateAccountParams{
		ID:                 onboardParams.AccountID,
		FinishedOnboarding: aws.Bool(true),
	}

	// Update account to mark onboarding as complete
	_, err = s.UpdateAccount(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	// Update user with onboarding information
	_, err = s.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:                 onboardParams.UserID,
		Email:              user.Email,
		FirstName:          pgtype.Text{String: onboardParams.FirstName, Valid: onboardParams.FirstName != ""},
		LastName:           pgtype.Text{String: onboardParams.LastName, Valid: onboardParams.LastName != ""},
		AddressLine1:       pgtype.Text{String: onboardParams.AddressLine1, Valid: onboardParams.AddressLine1 != ""},
		AddressLine2:       pgtype.Text{String: onboardParams.AddressLine2, Valid: onboardParams.AddressLine2 != ""},
		City:               pgtype.Text{String: onboardParams.City, Valid: onboardParams.City != ""},
		StateRegion:        pgtype.Text{String: onboardParams.State, Valid: onboardParams.State != ""},
		PostalCode:         pgtype.Text{String: onboardParams.PostalCode, Valid: onboardParams.PostalCode != ""},
		Country:            pgtype.Text{String: onboardParams.Country, Valid: onboardParams.Country != ""},
		DisplayName:        user.DisplayName,
		PictureUrl:         user.PictureUrl,
		Phone:              user.Phone,
		Timezone:           user.Timezone,
		Locale:             user.Locale,
		EmailVerified:      pgtype.Bool{Bool: true, Valid: true},
		TwoFactorEnabled:   user.TwoFactorEnabled,
		FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
		Status:             user.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("Account onboarded successfully",
		zap.String("account_id", onboardParams.AccountID.String()),
		zap.String("user_id", onboardParams.UserID.String()))

	return nil
}

// ValidateAccountAccess validates if user has access to account through workspace
func (s *AccountService) ValidateAccountAccess(ctx context.Context, accountID, workspaceID uuid.UUID) error {
	// Get the workspace
	workspace, err := s.queries.GetWorkspace(ctx, workspaceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("workspace not found")
		}
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Verify workspace belongs to account
	if workspace.AccountID != accountID {
		return fmt.Errorf("you are not authorized to access this account")
	}

	return nil
}
