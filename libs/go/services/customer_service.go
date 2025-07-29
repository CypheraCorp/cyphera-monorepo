package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
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

// CreateCustomer creates a new customer
func (s *CustomerService) CreateCustomer(ctx context.Context, createParams params.CreateCustomerParams) (*db.Customer, error) {
	metadataBytes, err := json.Marshal(createParams.Metadata)
	if err != nil {
		s.logger.Error("Failed to marshal customer metadata", zap.Error(err))
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	// Handle optional fields
	var name, description, phone string
	if createParams.Name != nil {
		name = *createParams.Name
	}
	if createParams.Description != nil {
		description = *createParams.Description
	}
	if createParams.Phone != nil {
		phone = *createParams.Phone
	}

	customer, err := s.queries.CreateCustomer(ctx, db.CreateCustomerParams{
		ExternalID:         pgtype.Text{String: createParams.Email, Valid: createParams.Email != ""}, // Using email as external ID
		Email:              pgtype.Text{String: createParams.Email, Valid: createParams.Email != ""},
		Name:               pgtype.Text{String: name, Valid: name != ""},
		Description:        pgtype.Text{String: description, Valid: description != ""},
		Phone:              pgtype.Text{String: phone, Valid: phone != ""},
		Metadata:           metadataBytes,
		FinishedOnboarding: createParams.FinishedOnboarding,
		PaymentSyncStatus:  "pending",
		PaymentProvider:    pgtype.Text{},
	})
	if err != nil {
		s.logger.Error("Failed to create customer",
			zap.String("email", createParams.Email),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	s.logger.Info("Customer created successfully",
		zap.String("customer_id", customer.ID.String()),
		zap.String("email", createParams.Email))

	return &customer, nil
}

// UpdateCustomer updates an existing customer
func (s *CustomerService) UpdateCustomer(ctx context.Context, updateParams params.UpdateCustomerParams) (*db.Customer, error) {
	dbParams := db.UpdateCustomerParams{
		ID: updateParams.ID,
	}

	// Update basic text fields
	if updateParams.Email != nil {
		dbParams.Email = pgtype.Text{String: *updateParams.Email, Valid: true}
	}
	if updateParams.Name != nil {
		dbParams.Name = pgtype.Text{String: *updateParams.Name, Valid: true}
	}
	if updateParams.Phone != nil {
		dbParams.Phone = pgtype.Text{String: *updateParams.Phone, Valid: true}
	}
	if updateParams.Description != nil {
		dbParams.Description = pgtype.Text{String: *updateParams.Description, Valid: true}
	}
	if updateParams.FinishedOnboarding != nil {
		dbParams.FinishedOnboarding = pgtype.Bool{Bool: *updateParams.FinishedOnboarding, Valid: true}
	}

	// Update JSON fields
	if updateParams.Metadata != nil {
		metadata, err := json.Marshal(updateParams.Metadata)
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
			zap.String("customer_id", updateParams.ID.String()),
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

// ListCustomers retrieves a paginated list of customers
func (s *CustomerService) ListCustomers(ctx context.Context, listParams params.ListCustomersParams) (*responses.ListCustomersResult, error) {
	customers, err := s.queries.ListCustomersWithPagination(ctx, db.ListCustomersWithPaginationParams{
		Limit:  listParams.Limit,
		Offset: listParams.Offset,
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

	// Convert customers to response format
	customerResponses := make([]responses.CustomerResponse, len(customers))
	for i, customer := range customers {
		customerResponses[i] = helpers.ToCustomerResponse(customer)
	}

	return &responses.ListCustomersResult{
		Customers: customerResponses,
		Total:     totalCount,
	}, nil
}

// ListWorkspaceCustomers retrieves a paginated list of customers for a workspace
func (s *CustomerService) ListWorkspaceCustomers(ctx context.Context, listParams params.ListWorkspaceCustomersParams) (*responses.ListWorkspaceCustomersResult, error) {
	customers, err := s.queries.ListWorkspaceCustomersWithPagination(ctx, db.ListWorkspaceCustomersWithPaginationParams{
		WorkspaceID: listParams.WorkspaceID,
		Limit:       listParams.Limit,
		Offset:      listParams.Offset,
	})
	if err != nil {
		s.logger.Error("Failed to list workspace customers",
			zap.String("workspace_id", listParams.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve workspace customers: %w", err)
	}

	totalCount, err := s.queries.CountWorkspaceCustomers(ctx, listParams.WorkspaceID)
	if err != nil {
		s.logger.Error("Failed to count workspace customers",
			zap.String("workspace_id", listParams.WorkspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to count workspace customers: %w", err)
	}

	// Convert customers to response format
	customerResponses := make([]responses.CustomerResponse, len(customers))
	for i, customer := range customers {
		customerResponses[i] = helpers.ToCustomerResponse(customer)
	}

	return &responses.ListWorkspaceCustomersResult{
		Customers: customerResponses,
		Total:     totalCount,
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

// CreateCustomerWithWeb3Auth creates a new customer with Web3Auth ID
func (s *CustomerService) CreateCustomerWithWeb3Auth(ctx context.Context, createParams params.CreateCustomerWithWeb3AuthParams) (*db.Customer, error) {
	metadataBytes, err := json.Marshal(createParams.Metadata)
	if err != nil {
		s.logger.Error("Failed to marshal customer metadata", zap.Error(err))
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	// Handle optional fields
	var name string
	if createParams.Name != nil {
		name = *createParams.Name
	}

	customer, err := s.queries.CreateCustomerWithWeb3Auth(ctx, db.CreateCustomerWithWeb3AuthParams{
		Web3authID:         pgtype.Text{String: createParams.Web3AuthID, Valid: createParams.Web3AuthID != ""},
		Email:              pgtype.Text{String: createParams.Email, Valid: createParams.Email != ""},
		Name:               pgtype.Text{String: name, Valid: name != ""},
		Phone:              pgtype.Text{}, // Phone not in params type
		Description:        pgtype.Text{},
		Metadata:           metadataBytes,
		FinishedOnboarding: false, // Not in params type
	})
	if err != nil {
		s.logger.Error("Failed to create customer with Web3Auth",
			zap.String("email", createParams.Email),
			zap.String("web3auth_id", createParams.Web3AuthID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	s.logger.Info("Customer created with Web3Auth successfully",
		zap.String("customer_id", customer.ID.String()),
		zap.String("email", createParams.Email))

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

// CreateCustomerWallet creates a new customer wallet
func (s *CustomerService) CreateCustomerWallet(ctx context.Context, createParams params.CreateCustomerWalletParams) (*db.CustomerWallet, error) {
	walletMetadata, err := json.Marshal(createParams.Metadata)
	if err != nil {
		s.logger.Error("Failed to marshal wallet metadata", zap.Error(err))
		return nil, fmt.Errorf("invalid wallet metadata format: %w", err)
	}

	// Parse network type
	networkType, err := parseNetworkType(createParams.NetworkType)
	if err != nil {
		return nil, fmt.Errorf("invalid network type: %w", err)
	}

	// Handle optional fields
	var nickname, ens string
	if createParams.Nickname != nil {
		nickname = *createParams.Nickname
	}
	if createParams.ENS != nil {
		ens = *createParams.ENS
	}

	wallet, err := s.queries.CreateCustomerWallet(ctx, db.CreateCustomerWalletParams{
		CustomerID:    createParams.CustomerID,
		WalletAddress: createParams.WalletAddress,
		NetworkType:   networkType,
		Nickname:      pgtype.Text{String: nickname, Valid: nickname != ""},
		Ens:           pgtype.Text{String: ens, Valid: ens != ""},
		IsPrimary:     pgtype.Bool{Bool: createParams.IsPrimary, Valid: true},
		Verified:      pgtype.Bool{Bool: createParams.Verified, Valid: true},
		Metadata:      walletMetadata,
	})
	if err != nil {
		s.logger.Error("Failed to create customer wallet",
			zap.String("customer_id", createParams.CustomerID.String()),
			zap.String("wallet_address", createParams.WalletAddress),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create customer wallet: %w", err)
	}

	s.logger.Info("Customer wallet created successfully",
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("customer_id", createParams.CustomerID.String()))

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

// ProcessCustomerAndWallet handles customer lookup/creation and wallet association
func (s *CustomerService) ProcessCustomerAndWallet(
	ctx context.Context,
	tx pgx.Tx,
	processParams params.ProcessCustomerWalletParams,
) (*db.Customer, *db.CustomerWallet, error) {
	// Create queries instance with transaction
	var qtx db.Querier
	if tx != nil {
		qtx = db.New(tx)
	} else {
		qtx = s.queries
	}

	customers, err := qtx.GetCustomersByWalletAddress(ctx, processParams.WalletAddress)
	if err != nil {
		s.logger.Error("Failed to check for existing customers",
			zap.Error(err),
			zap.String("wallet_address", processParams.WalletAddress))
		return nil, nil, err
	}

	if len(customers) == 0 {
		return s.CreateCustomerFromWallet(ctx, tx, params.CreateCustomerFromWalletParams{
			WalletAddress: processParams.WalletAddress,
			WorkspaceID:   processParams.WorkspaceID,
			ProductID:     processParams.ProductID,
			NetworkType:   processParams.NetworkType,
		})
	}

	customer := customers[0]

	// Ensure customer is associated with the current workspace
	isAssociated, err := qtx.IsCustomerInWorkspace(ctx, db.IsCustomerInWorkspaceParams{
		WorkspaceID: processParams.WorkspaceID,
		CustomerID:  customer.ID,
	})
	if err != nil {
		s.logger.Error("Failed to check customer workspace association",
			zap.Error(err),
			zap.String("customer_id", customer.ID.String()),
			zap.String("workspace_id", processParams.WorkspaceID.String()))
		return nil, nil, err
	}

	// If customer is not associated with this workspace, create the association
	if !isAssociated {
		_, err = qtx.AddCustomerToWorkspace(ctx, db.AddCustomerToWorkspaceParams{
			WorkspaceID: processParams.WorkspaceID,
			CustomerID:  customer.ID,
		})
		if err != nil {
			s.logger.Error("Failed to associate customer with workspace",
				zap.Error(err),
				zap.String("customer_id", customer.ID.String()),
				zap.String("workspace_id", processParams.WorkspaceID.String()))
			return nil, nil, err
		}

		s.logger.Info("Associated existing customer with workspace",
			zap.String("customer_id", customer.ID.String()),
			zap.String("workspace_id", processParams.WorkspaceID.String()),
			zap.String("wallet_address", processParams.WalletAddress))
	}

	customerWallet, err := s.FindOrCreateCustomerWallet(ctx, tx, params.FindOrCreateWalletParams{
		CustomerID:    customer.ID,
		WalletAddress: processParams.WalletAddress,
		NetworkType:   processParams.NetworkType,
		ProductID:     processParams.ProductID,
		IsPrimary:     true,
		Verified:      true,
	})

	return &customer, customerWallet, err
}

// CreateCustomerFromWallet creates a new customer and associated wallet
func (s *CustomerService) CreateCustomerFromWallet(
	ctx context.Context,
	tx pgx.Tx,
	params params.CreateCustomerFromWalletParams,
) (*db.Customer, *db.CustomerWallet, error) {
	var qtx db.Querier
	if tx != nil {
		qtx = db.New(tx)
	} else {
		qtx = s.queries
	}

	s.logger.Info("Creating new customer for wallet address",
		zap.String("wallet_address", params.WalletAddress),
		zap.String("product_id", params.ProductID.String()))

	metadata := map[string]interface{}{
		"source":                  "product_subscription",
		"created_from_product_id": params.ProductID.String(),
		"wallet_address":          params.WalletAddress,
	}

	// Merge with provided metadata
	if params.Metadata != nil {
		for k, v := range params.Metadata {
			metadata[k] = v
		}
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, nil, err
	}

	createCustomerParams := db.CreateCustomerParams{
		Email: pgtype.Text{
			String: "",
			Valid:  false,
		},
		Name: pgtype.Text{
			String: "Wallet Customer: " + params.WalletAddress,
			Valid:  true,
		},
		Description: pgtype.Text{
			String: "Customer created from product subscription",
			Valid:  true,
		},
		Metadata: metadataBytes,
	}

	customer, err := qtx.CreateCustomer(ctx, createCustomerParams)
	if err != nil {
		return nil, nil, err
	}

	// Associate customer with workspace using the new association table
	_, err = qtx.AddCustomerToWorkspace(ctx, db.AddCustomerToWorkspaceParams{
		WorkspaceID: params.WorkspaceID,
		CustomerID:  customer.ID,
	})
	if err != nil {
		return nil, nil, err
	}

	walletMetadata := map[string]interface{}{
		"source":     "product_subscription",
		"product_id": params.ProductID.String(),
		"created_at": time.Now().Format(time.RFC3339),
	}
	walletMetadataBytes, err := json.Marshal(walletMetadata)
	if err != nil {
		return nil, nil, err
	}

	createWalletParams := db.CreateCustomerWalletParams{
		CustomerID:    customer.ID,
		WalletAddress: params.WalletAddress,
		NetworkType:   db.NetworkType(params.NetworkType),
		Nickname: pgtype.Text{
			String: "Subscription Wallet",
			Valid:  true,
		},
		IsPrimary: pgtype.Bool{
			Bool:  true,
			Valid: true,
		},
		Verified: pgtype.Bool{
			Bool:  true,
			Valid: true,
		},
		Metadata: walletMetadataBytes,
	}

	customerWallet, err := qtx.CreateCustomerWallet(ctx, createWalletParams)
	return &customer, &customerWallet, err
}

// FindOrCreateCustomerWallet finds an existing wallet or creates a new one
func (s *CustomerService) FindOrCreateCustomerWallet(
	ctx context.Context,
	tx pgx.Tx,
	params params.FindOrCreateWalletParams,
) (*db.CustomerWallet, error) {
	var qtx db.Querier
	if tx != nil {
		qtx = db.New(tx)
	} else {
		qtx = s.queries
	}

	wallets, err := qtx.ListCustomerWallets(ctx, params.CustomerID)
	if err != nil {
		return nil, err
	}

	for _, wallet := range wallets {
		if strings.EqualFold(wallet.WalletAddress, params.WalletAddress) {
			updatedWallet, err := qtx.UpdateCustomerWalletUsageTime(ctx, wallet.ID)
			if err != nil {
				s.logger.Warn("Failed to update wallet usage time",
					zap.Error(err),
					zap.String("wallet_id", wallet.ID.String()))
				return &wallet, nil
			}
			return &updatedWallet, nil
		}
	}

	walletMetadata := map[string]interface{}{
		"source":     "product_subscription",
		"product_id": params.ProductID.String(),
		"created_at": time.Now().Format(time.RFC3339),
	}
	walletMetadataBytes, err := json.Marshal(walletMetadata)
	if err != nil {
		return nil, err
	}

	nickname := "Subscription Wallet"
	if params.Nickname != nil {
		nickname = *params.Nickname
	}

	createWalletParams := db.CreateCustomerWalletParams{
		CustomerID:    params.CustomerID,
		WalletAddress: params.WalletAddress,
		NetworkType:   db.NetworkType(params.NetworkType),
		Nickname: pgtype.Text{
			String: nickname,
			Valid:  true,
		},
		IsPrimary: pgtype.Bool{
			Bool:  params.IsPrimary || len(wallets) == 0,
			Valid: true,
		},
		Verified: pgtype.Bool{
			Bool:  params.Verified,
			Valid: true,
		},
		Metadata: walletMetadataBytes,
	}

	wallet, err := qtx.CreateCustomerWallet(ctx, createWalletParams)
	return &wallet, err
}
