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

// WalletService handles business logic for wallet operations
type WalletService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewWalletService creates a new wallet service
func NewWalletService(queries db.Querier) *WalletService {
	return &WalletService{
		queries: queries,
		logger:  logger.Log,
	}
}

// CircleWalletData represents Circle-specific wallet data
type CircleWalletData struct {
	CircleWalletID string
	CircleUserID   string
	ChainID        int32
	State          string
}

// WalletWithCircleData represents a wallet with optional Circle data
type WalletWithCircleData struct {
	Wallet     db.Wallet
	CircleData *CircleWalletData
}

// CreateWalletParams contains parameters for creating a wallet
type CreateWalletParams struct {
	WorkspaceID   uuid.UUID
	WalletType    string
	WalletAddress string
	NetworkType   string
	NetworkID     *uuid.UUID
	Nickname      string
	ENS           string
	IsPrimary     bool
	Verified      bool
	Metadata      map[string]interface{}
}

// CreateWallet creates a new wallet
func (s *WalletService) CreateWallet(ctx context.Context, params CreateWalletParams) (*db.Wallet, error) {
	// Validate required fields
	if params.WalletType == "" {
		return nil, fmt.Errorf("wallet type is required")
	}
	if params.WalletAddress == "" {
		return nil, fmt.Errorf("wallet address is required")
	}
	if params.NetworkType == "" {
		return nil, fmt.Errorf("network type is required")
	}

	// Validate wallet type
	if params.WalletType != "wallet" && params.WalletType != "circle_wallet" {
		return nil, fmt.Errorf("invalid wallet type: %s", params.WalletType)
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

	// Prepare network ID
	var networkID pgtype.UUID
	if params.NetworkID != nil {
		networkID = pgtype.UUID{Bytes: *params.NetworkID, Valid: true}
	}

	// Create wallet
	wallet, err := s.queries.CreateWallet(ctx, db.CreateWalletParams{
		WorkspaceID:   params.WorkspaceID,
		WalletType:    params.WalletType,
		WalletAddress: params.WalletAddress,
		NetworkType:   db.NetworkType(params.NetworkType),
		NetworkID:     networkID,
		Nickname:      pgtype.Text{String: params.Nickname, Valid: params.Nickname != ""},
		Ens:           pgtype.Text{String: params.ENS, Valid: params.ENS != ""},
		IsPrimary:     pgtype.Bool{Bool: params.IsPrimary, Valid: true},
		Verified:      pgtype.Bool{Bool: params.Verified, Valid: true},
		Metadata:      metadataJSON,
	})
	if err != nil {
		s.logger.Error("Failed to create wallet",
			zap.String("workspace_id", params.WorkspaceID.String()),
			zap.String("wallet_type", params.WalletType),
			zap.String("wallet_address", params.WalletAddress),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	s.logger.Info("Wallet created successfully",
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("wallet_type", wallet.WalletType),
		zap.String("wallet_address", wallet.WalletAddress))

	return &wallet, nil
}

// CreateWalletsForAllNetworks creates wallets for all active networks
func (s *WalletService) CreateWalletsForAllNetworks(ctx context.Context, params CreateWalletParams) ([]db.Wallet, error) {
	// Get all active networks
	networks, err := s.queries.ListNetworks(ctx, db.ListNetworksParams{
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active networks: %w", err)
	}

	if len(networks) == 0 {
		return nil, fmt.Errorf("no active networks found")
	}

	// Create wallets for each network
	var createdWallets []db.Wallet
	for _, network := range networks {
		params.NetworkType = string(network.NetworkType)
		params.NetworkID = &network.ID

		wallet, err := s.CreateWallet(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet for network %s: %w", network.Name, err)
		}

		createdWallets = append(createdWallets, *wallet)
	}

	return createdWallets, nil
}

// GetWallet retrieves a wallet by ID
func (s *WalletService) GetWallet(ctx context.Context, walletID, workspaceID uuid.UUID) (*db.Wallet, error) {
	wallet, err := s.queries.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          walletID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		s.logger.Error("Failed to get wallet",
			zap.String("wallet_id", walletID.String()),
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve wallet: %w", err)
	}

	return &wallet, nil
}

// GetWalletWithCircleData retrieves a wallet with Circle data
func (s *WalletService) GetWalletWithCircleData(ctx context.Context, walletID, workspaceID uuid.UUID) (*WalletWithCircleData, error) {
	result, err := s.queries.GetWalletWithCircleDataByID(ctx, db.GetWalletWithCircleDataByIDParams{
		ID:          walletID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		s.logger.Error("Failed to get wallet with Circle data",
			zap.String("wallet_id", walletID.String()),
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve wallet: %w", err)
	}

	// Convert to wallet with Circle data
	walletData := &WalletWithCircleData{
		Wallet: db.Wallet{
			ID:            result.ID,
			WorkspaceID:   result.WorkspaceID,
			WalletType:    result.WalletType,
			WalletAddress: result.WalletAddress,
			NetworkType:   result.NetworkType,
			NetworkID:     result.NetworkID,
			Nickname:      result.Nickname,
			Ens:           result.Ens,
			IsPrimary:     result.IsPrimary,
			Verified:      result.Verified,
			LastUsedAt:    result.LastUsedAt,
			Metadata:      result.Metadata,
			CreatedAt:     result.CreatedAt,
			UpdatedAt:     result.UpdatedAt,
		},
	}

	// Add Circle data if present
	if result.CircleWalletID.Valid && result.CircleUserID.Valid {
		walletData.CircleData = &CircleWalletData{
			CircleWalletID: result.CircleWalletID.String(),
			CircleUserID:   result.CircleUserID.String(),
			ChainID:        result.ChainID.Int32,
			State:          result.CircleState.String,
		}
	}

	return walletData, nil
}

// ListWalletsByWorkspace lists all wallets for a workspace
func (s *WalletService) ListWalletsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.Wallet, error) {
	wallets, err := s.queries.ListWalletsByWorkspaceID(ctx, workspaceID)
	if err != nil {
		s.logger.Error("Failed to list wallets",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve wallets: %w", err)
	}

	return wallets, nil
}

// ListWalletsByType lists wallets filtered by type
func (s *WalletService) ListWalletsByType(ctx context.Context, workspaceID uuid.UUID, walletType string) ([]db.Wallet, error) {
	wallets, err := s.queries.ListWalletsByWalletType(ctx, db.ListWalletsByWalletTypeParams{
		WorkspaceID: workspaceID,
		WalletType:  walletType,
	})
	if err != nil {
		s.logger.Error("Failed to list wallets by type",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("wallet_type", walletType),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve wallets: %w", err)
	}

	return wallets, nil
}

// ListCircleWallets lists Circle wallets for a workspace
func (s *WalletService) ListCircleWallets(ctx context.Context, workspaceID uuid.UUID) ([]db.ListCircleWalletsByWorkspaceIDRow, error) {
	wallets, err := s.queries.ListCircleWalletsByWorkspaceID(ctx, workspaceID)
	if err != nil {
		s.logger.Error("Failed to list Circle wallets",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve Circle wallets: %w", err)
	}

	return wallets, nil
}

// ListWalletsWithCircleData lists all wallets with Circle data
func (s *WalletService) ListWalletsWithCircleData(ctx context.Context, workspaceID uuid.UUID) ([]db.ListWalletsWithCircleDataByWorkspaceIDRow, error) {
	wallets, err := s.queries.ListWalletsWithCircleDataByWorkspaceID(ctx, workspaceID)
	if err != nil {
		s.logger.Error("Failed to list wallets with Circle data",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve wallets: %w", err)
	}

	return wallets, nil
}

// UpdateWalletParams contains parameters for updating a wallet
type UpdateWalletParams struct {
	ID        uuid.UUID
	Nickname  *string
	ENS       *string
	IsPrimary *bool
	Verified  *bool
	Metadata  map[string]interface{}
}

// UpdateWallet updates an existing wallet
func (s *WalletService) UpdateWallet(ctx context.Context, workspaceID uuid.UUID, params UpdateWalletParams) (*db.Wallet, error) {
	// First verify the wallet belongs to the workspace
	_, err := s.GetWallet(ctx, params.ID, workspaceID)
	if err != nil {
		return nil, err
	}

	// Prepare update params
	updateParams := db.UpdateWalletParams{
		ID: params.ID,
	}

	// Set optional fields
	if params.Nickname != nil {
		updateParams.Nickname = pgtype.Text{String: *params.Nickname, Valid: true}
	}
	if params.ENS != nil {
		updateParams.Ens = pgtype.Text{String: *params.ENS, Valid: true}
	}
	if params.IsPrimary != nil {
		updateParams.IsPrimary = pgtype.Bool{Bool: *params.IsPrimary, Valid: true}
	}
	if params.Verified != nil {
		updateParams.Verified = pgtype.Bool{Bool: *params.Verified, Valid: true}
	}

	// Convert metadata to JSON if provided
	if params.Metadata != nil {
		metadataJSON, err := json.Marshal(params.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		updateParams.Metadata = metadataJSON
	}

	// Update wallet
	wallet, err := s.queries.UpdateWallet(ctx, updateParams)
	if err != nil {
		s.logger.Error("Failed to update wallet",
			zap.String("wallet_id", params.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update wallet: %w", err)
	}

	s.logger.Info("Wallet updated successfully",
		zap.String("wallet_id", wallet.ID.String()))

	return &wallet, nil
}

// DeleteWallet soft deletes a wallet
func (s *WalletService) DeleteWallet(ctx context.Context, walletID, workspaceID uuid.UUID) error {
	// First verify the wallet belongs to the workspace
	_, err := s.GetWallet(ctx, walletID, workspaceID)
	if err != nil {
		return err
	}

	// Check if wallet is being used by any active products
	products, err := s.queries.GetActiveProductsByWalletID(ctx, walletID)
	if err != nil {
		return fmt.Errorf("failed to check product usage: %w", err)
	}

	if len(products) > 0 {
		return fmt.Errorf("cannot delete wallet: used by %d active product(s)", len(products))
	}

	// Soft delete the wallet
	err = s.queries.SoftDeleteWallet(ctx, walletID)
	if err != nil {
		s.logger.Error("Failed to delete wallet",
			zap.String("wallet_id", walletID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete wallet: %w", err)
	}

	s.logger.Info("Wallet deleted successfully",
		zap.String("wallet_id", walletID.String()))

	return nil
}

// GetWalletByAddressAndNetwork retrieves a wallet by address and network
func (s *WalletService) GetWalletByAddressAndNetwork(ctx context.Context, workspaceID uuid.UUID, walletAddress, networkType string) (*db.Wallet, error) {
	wallet, err := s.queries.GetWalletByAddressAndCircleNetworkType(ctx, db.GetWalletByAddressAndCircleNetworkTypeParams{
		WalletAddress:     walletAddress,
		CircleNetworkType: db.CircleNetworkType(networkType),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("failed to retrieve wallet: %w", err)
	}

	// Convert result to wallet
	walletResult := &db.Wallet{
		ID:            wallet.ID,
		WorkspaceID:   wallet.WorkspaceID,
		WalletType:    wallet.WalletType,
		WalletAddress: wallet.WalletAddress,
		NetworkType:   wallet.NetworkType,
		NetworkID:     wallet.NetworkID,
		Nickname:      wallet.Nickname,
		Ens:           wallet.Ens,
		IsPrimary:     wallet.IsPrimary,
		Verified:      wallet.Verified,
		LastUsedAt:    wallet.LastUsedAt,
		Metadata:      wallet.Metadata,
		CreatedAt:     wallet.CreatedAt,
		UpdatedAt:     wallet.UpdatedAt,
	}

	return walletResult, nil
}

// UpdateWalletLastUsed updates the last used timestamp for a wallet
func (s *WalletService) UpdateWalletLastUsed(ctx context.Context, walletID uuid.UUID) error {
	err := s.queries.UpdateWalletLastUsed(ctx, walletID)
	if err != nil {
		s.logger.Error("Failed to update wallet last used",
			zap.String("wallet_id", walletID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to update wallet last used: %w", err)
	}

	return nil
}

// ValidateWalletAccess checks if a wallet belongs to a workspace
func (s *WalletService) ValidateWalletAccess(ctx context.Context, walletID, workspaceID uuid.UUID) error {
	wallet, err := s.GetWallet(ctx, walletID, workspaceID)
	if err != nil {
		return err
	}

	if wallet.WorkspaceID != workspaceID {
		return fmt.Errorf("wallet does not belong to this workspace")
	}

	return nil
}
