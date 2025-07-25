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

// CustomerService handles business logic for customer operations
type CustomerService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewCustomerService creates a new customer service
func NewCustomerService(queries db.Querier) *CustomerService {
	return &CustomerService{
		queries: queries,
		logger:  logger.Log,
	}
}

// GetCustomer retrieves a customer by ID
func (s *CustomerService) GetCustomer(ctx context.Context, customerID uuid.UUID) (*db.Customer, error) {
	customer, err := s.queries.GetCustomer(ctx, customerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		s.logger.Error("Failed to get customer",
			zap.String("customer_id", customerID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve customer: %w", err)
	}

	return &customer, nil
}

// CreateCustomerParams contains parameters for creating a customer
type CreateCustomerParams struct {
	ExternalID         string
	Email              string
	Name               string
	Phone              string
	Description        string
	FinishedOnboarding bool
	Metadata           map[string]interface{}
}

// CreateCustomer creates a new customer
func (s *CustomerService) CreateCustomer(ctx context.Context, params CreateCustomerParams) (*db.Customer, error) {
	metadataBytes, err := json.Marshal(params.Metadata)
	if err != nil {
		s.logger.Error("Failed to marshal customer metadata", zap.Error(err))
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	customer, err := s.queries.CreateCustomer(ctx, db.CreateCustomerParams{
		ExternalID:         pgtype.Text{String: params.ExternalID, Valid: params.ExternalID != ""},
		Email:              pgtype.Text{String: params.Email, Valid: params.Email != ""},
		Name:               pgtype.Text{String: params.Name, Valid: params.Name != ""},
		Description:        pgtype.Text{String: params.Description, Valid: params.Description != ""},
		Phone:              pgtype.Text{String: params.Phone, Valid: params.Phone != ""},
		Metadata:           metadataBytes,
		FinishedOnboarding: params.FinishedOnboarding,
		PaymentSyncStatus:  "pending",
		PaymentProvider:    pgtype.Text{},
	})
	if err != nil {
		s.logger.Error("Failed to create customer",
			zap.String("email", params.Email),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	s.logger.Info("Customer created successfully",
		zap.String("customer_id", customer.ID.String()),
		zap.String("email", params.Email))

	return &customer, nil
}

// UpdateCustomerParams contains parameters for updating a customer
type UpdateCustomerParams struct {
	ID                 uuid.UUID
	ExternalID         *string
	Email              *string
	Name               *string
	Phone              *string
	Description        *string
	FinishedOnboarding *bool
	Metadata           map[string]interface{}
}

// UpdateCustomer updates an existing customer
func (s *CustomerService) UpdateCustomer(ctx context.Context, params UpdateCustomerParams) (*db.Customer, error) {
	dbParams := db.UpdateCustomerParams{
		ID: params.ID,
	}

	// Update basic text fields
	if params.Email != nil {
		dbParams.Email = pgtype.Text{String: *params.Email, Valid: true}
	}
	if params.Name != nil {
		dbParams.Name = pgtype.Text{String: *params.Name, Valid: true}
	}
	if params.Phone != nil {
		dbParams.Phone = pgtype.Text{String: *params.Phone, Valid: true}
	}
	if params.Description != nil {
		dbParams.Description = pgtype.Text{String: *params.Description, Valid: true}
	}
	if params.FinishedOnboarding != nil {
		dbParams.FinishedOnboarding = pgtype.Bool{Bool: *params.FinishedOnboarding, Valid: true}
	}

	// Update JSON fields
	if params.Metadata != nil {
		metadata, err := json.Marshal(params.Metadata)
		if err != nil {
			s.logger.Error("Failed to marshal customer metadata", zap.Error(err))
			return nil, fmt.Errorf("invalid metadata format: %w", err)
		}
		dbParams.Metadata = metadata
	}

	customer, err := s.queries.UpdateCustomer(ctx, dbParams)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		s.logger.Error("Failed to update customer",
			zap.String("customer_id", params.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	s.logger.Info("Customer updated successfully",
		zap.String("customer_id", customer.ID.String()))

	return &customer, nil
}

// DeleteCustomer deletes a customer by ID
func (s *CustomerService) DeleteCustomer(ctx context.Context, customerID uuid.UUID) error {
	err := s.queries.DeleteCustomer(ctx, customerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("customer not found")
		}
		s.logger.Error("Failed to delete customer",
			zap.String("customer_id", customerID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	s.logger.Info("Customer deleted successfully",
		zap.String("customer_id", customerID.String()))

	return nil
}

// ListCustomersParams contains parameters for listing customers
type ListCustomersParams struct {
	Limit  int
	Offset int
}

// ListCustomersResult contains the result of listing customers
type ListCustomersResult struct {
	Customers  []db.Customer
	TotalCount int64
}

// ListCustomers retrieves a paginated list of customers
func (s *CustomerService) ListCustomers(ctx context.Context, params ListCustomersParams) (*ListCustomersResult, error) {
	customers, err := s.queries.ListCustomersWithPagination(ctx, db.ListCustomersWithPaginationParams{
		Limit:  int32(params.Limit),
		Offset: int32(params.Offset),
	})
	if err != nil {
		s.logger.Error("Failed to list customers", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve customers: %w", err)
	}

	totalCount, err := s.queries.CountCustomers(ctx)
	if err != nil {
		s.logger.Error("Failed to count customers", zap.Error(err))
		return nil, fmt.Errorf("failed to count customers: %w", err)
	}

	return &ListCustomersResult{
		Customers:  customers,
		TotalCount: totalCount,
	}, nil
}

// ListWorkspaceCustomersParams contains parameters for listing workspace customers
type ListWorkspaceCustomersParams struct {
	WorkspaceID uuid.UUID
	Limit       int
	Offset      int
}

// ListWorkspaceCustomersResult contains the result of listing workspace customers
type ListWorkspaceCustomersResult struct {
	Customers  []db.Customer
	TotalCount int64
}

// ListWorkspaceCustomers retrieves a paginated list of customers for a workspace
func (s *CustomerService) ListWorkspaceCustomers(ctx context.Context, params ListWorkspaceCustomersParams) (*ListWorkspaceCustomersResult, error) {
	customers, err := s.queries.ListWorkspaceCustomersWithPagination(ctx, db.ListWorkspaceCustomersWithPaginationParams{
		WorkspaceID: params.WorkspaceID,
		Limit:       int32(params.Limit),
		Offset:      int32(params.Offset),
	})
	if err != nil {
		s.logger.Error("Failed to list workspace customers",
			zap.String("workspace_id", params.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve workspace customers: %w", err)
	}

	totalCount, err := s.queries.CountWorkspaceCustomers(ctx, params.WorkspaceID)
	if err != nil {
		s.logger.Error("Failed to count workspace customers",
			zap.String("workspace_id", params.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to count workspace customers: %w", err)
	}

	return &ListWorkspaceCustomersResult{
		Customers:  customers,
		TotalCount: totalCount,
	}, nil
}

// UpdateCustomerOnboardingStatus updates the onboarding status for a customer
func (s *CustomerService) UpdateCustomerOnboardingStatus(ctx context.Context, customerID uuid.UUID, finishedOnboarding bool) (*db.Customer, error) {
	customer, err := s.queries.UpdateCustomerOnboardingStatus(ctx, db.UpdateCustomerOnboardingStatusParams{
		ID:                 customerID,
		FinishedOnboarding: pgtype.Bool{Bool: finishedOnboarding, Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		s.logger.Error("Failed to update customer onboarding status",
			zap.String("customer_id", customerID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update customer onboarding status: %w", err)
	}

	s.logger.Info("Customer onboarding status updated successfully",
		zap.String("customer_id", customer.ID.String()),
		zap.Bool("finished_onboarding", finishedOnboarding))

	return &customer, nil
}

// AddCustomerToWorkspace associates a customer with a workspace
func (s *CustomerService) AddCustomerToWorkspace(ctx context.Context, workspaceID, customerID uuid.UUID) error {
	_, err := s.queries.AddCustomerToWorkspace(ctx, db.AddCustomerToWorkspaceParams{
		WorkspaceID: workspaceID,
		CustomerID:  customerID,
	})
	if err != nil {
		s.logger.Error("Failed to add customer to workspace",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("customer_id", customerID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to add customer to workspace: %w", err)
	}

	s.logger.Info("Customer added to workspace successfully",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("customer_id", customerID.String()))

	return nil
}

// GetCustomerByWeb3AuthID retrieves a customer by Web3Auth ID
func (s *CustomerService) GetCustomerByWeb3AuthID(ctx context.Context, web3authID string) (*db.Customer, error) {
	customer, err := s.queries.GetCustomerByWeb3AuthID(ctx, pgtype.Text{String: web3authID, Valid: web3authID != ""})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		s.logger.Error("Failed to get customer by Web3Auth ID",
			zap.String("web3auth_id", web3authID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve customer: %w", err)
	}

	return &customer, nil
}

// CreateCustomerWithWeb3AuthParams contains parameters for creating a customer with Web3Auth
type CreateCustomerWithWeb3AuthParams struct {
	Web3AuthID         string
	Email              string
	Name               string
	Phone              string
	Description        string
	Metadata           map[string]interface{}
	FinishedOnboarding bool
}

// CreateCustomerWithWeb3Auth creates a new customer with Web3Auth ID
func (s *CustomerService) CreateCustomerWithWeb3Auth(ctx context.Context, params CreateCustomerWithWeb3AuthParams) (*db.Customer, error) {
	metadataBytes, err := json.Marshal(params.Metadata)
	if err != nil {
		s.logger.Error("Failed to marshal customer metadata", zap.Error(err))
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	customer, err := s.queries.CreateCustomerWithWeb3Auth(ctx, db.CreateCustomerWithWeb3AuthParams{
		Web3authID:         pgtype.Text{String: params.Web3AuthID, Valid: params.Web3AuthID != ""},
		Email:              pgtype.Text{String: params.Email, Valid: params.Email != ""},
		Name:               pgtype.Text{String: params.Name, Valid: params.Name != ""},
		Phone:              pgtype.Text{String: params.Phone, Valid: params.Phone != ""},
		Description:        pgtype.Text{},
		Metadata:           metadataBytes,
		FinishedOnboarding: params.FinishedOnboarding,
	})
	if err != nil {
		s.logger.Error("Failed to create customer with Web3Auth",
			zap.String("email", params.Email),
			zap.String("web3auth_id", params.Web3AuthID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	s.logger.Info("Customer created with Web3Auth successfully",
		zap.String("customer_id", customer.ID.String()),
		zap.String("email", params.Email))

	return &customer, nil
}

// ListCustomerWallets retrieves wallets for a customer
func (s *CustomerService) ListCustomerWallets(ctx context.Context, customerID uuid.UUID) ([]db.CustomerWallet, error) {
	wallets, err := s.queries.ListCustomerWallets(ctx, customerID)
	if err != nil {
		s.logger.Error("Failed to list customer wallets",
			zap.String("customer_id", customerID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve customer wallets: %w", err)
	}

	return wallets, nil
}

// CreateCustomerWalletParams contains parameters for creating a customer wallet
type CreateCustomerWalletParams struct {
	CustomerID    uuid.UUID
	WalletAddress string
	NetworkType   string
	Nickname      string
	ENS           string
	IsPrimary     bool
	Verified      bool
	Metadata      map[string]interface{}
}

// CreateCustomerWallet creates a new customer wallet
func (s *CustomerService) CreateCustomerWallet(ctx context.Context, params CreateCustomerWalletParams) (*db.CustomerWallet, error) {
	walletMetadata, err := json.Marshal(params.Metadata)
	if err != nil {
		s.logger.Error("Failed to marshal wallet metadata", zap.Error(err))
		return nil, fmt.Errorf("invalid wallet metadata format: %w", err)
	}

	// Parse network type
	networkType, err := parseNetworkType(params.NetworkType)
	if err != nil {
		return nil, fmt.Errorf("invalid network type: %w", err)
	}

	wallet, err := s.queries.CreateCustomerWallet(ctx, db.CreateCustomerWalletParams{
		CustomerID:    params.CustomerID,
		WalletAddress: params.WalletAddress,
		NetworkType:   networkType,
		Nickname:      pgtype.Text{String: params.Nickname, Valid: params.Nickname != ""},
		Ens:           pgtype.Text{String: params.ENS, Valid: params.ENS != ""},
		IsPrimary:     pgtype.Bool{Bool: params.IsPrimary, Valid: true},
		Verified:      pgtype.Bool{Bool: params.Verified, Valid: true},
		Metadata:      walletMetadata,
	})
	if err != nil {
		s.logger.Error("Failed to create customer wallet",
			zap.String("customer_id", params.CustomerID.String()),
			zap.String("wallet_address", params.WalletAddress),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create customer wallet: %w", err)
	}

	s.logger.Info("Customer wallet created successfully",
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("customer_id", params.CustomerID.String()))

	return &wallet, nil
}

// parseNetworkType converts string to NetworkType
func parseNetworkType(networkType string) (db.NetworkType, error) {
	switch networkType {
	case "evm":
		return db.NetworkTypeEvm, nil
	case "solana":
		return db.NetworkTypeSolana, nil
	case "cosmos":
		return db.NetworkTypeCosmos, nil
	case "bitcoin":
		return db.NetworkTypeBitcoin, nil
	case "polkadot":
		return db.NetworkTypePolkadot, nil
	default:
		return "", fmt.Errorf("unsupported network type: %s", networkType)
	}
}