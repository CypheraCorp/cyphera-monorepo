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

// NetworkService handles business logic for network operations
type NetworkService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewNetworkService creates a new network service
func NewNetworkService(queries db.Querier) *NetworkService {
	return &NetworkService{
		queries: queries,
		logger:  logger.Log,
	}
}

// GetNetwork retrieves a network by ID
func (s *NetworkService) GetNetwork(ctx context.Context, networkID uuid.UUID) (*db.Network, error) {
	network, err := s.queries.GetNetwork(ctx, networkID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("network not found")
		}
		s.logger.Error("Failed to get network",
			zap.String("network_id", networkID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve network: %w", err)
	}

	return &network, nil
}

// GetNetworkByChainID retrieves a network by chain ID
func (s *NetworkService) GetNetworkByChainID(ctx context.Context, chainID int32) (*db.Network, error) {
	network, err := s.queries.GetNetworkByChainID(ctx, chainID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("network not found")
		}
		s.logger.Error("Failed to get network by chain ID",
			zap.Int32("chain_id", chainID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve network: %w", err)
	}

	return &network, nil
}

// CreateNetworkParams contains parameters for creating a network
type CreateNetworkParams struct {
	Name              string
	Type              string
	NetworkType       string
	CircleNetworkType string
	BlockExplorerURL  string
	ChainID           int32
	IsTestnet         bool
	Active            bool
	LogoURL           string
	DisplayName       string
	ChainNamespace    string
	GasConfig         *CreateGasConfigParams
}

// CreateGasConfigParams contains gas configuration parameters
type CreateGasConfigParams struct {
	BaseFeeMultiplier     float64
	PriorityFeeMultiplier float64
	DeploymentGasLimit    string
	TokenTransferGasLimit string
	SupportsEIP1559       bool
	GasOracleURL          string
	GasRefreshIntervalMs  int32
	GasPriorityLevels     map[string]interface{}
	AverageBlockTimeMs    int32
	PeakHoursMultiplier   float64
}

// CreateNetwork creates a new network
func (s *NetworkService) CreateNetwork(ctx context.Context, params CreateNetworkParams) (*db.Network, error) {
	dbParams := db.CreateNetworkParams{
		Name:              params.Name,
		Type:              params.Type,
		NetworkType:       db.NetworkType(params.NetworkType),
		CircleNetworkType: db.CircleNetworkType(params.CircleNetworkType),
		BlockExplorerUrl:  nullableString(params.BlockExplorerURL),
		ChainID:           params.ChainID,
		IsTestnet:         params.IsTestnet,
		Active:            params.Active,
		LogoUrl:           nullableString(params.LogoURL),
		DisplayName:       nullableString(params.DisplayName),
		ChainNamespace:    nullableString(params.ChainNamespace),
	}

	// Set gas config if provided
	if params.GasConfig != nil {
		dbParams.BaseFeeMultiplier = nullableNumeric(params.GasConfig.BaseFeeMultiplier)
		dbParams.PriorityFeeMultiplier = nullableNumeric(params.GasConfig.PriorityFeeMultiplier)
		dbParams.DeploymentGasLimit = nullableString(params.GasConfig.DeploymentGasLimit)
		dbParams.TokenTransferGasLimit = nullableString(params.GasConfig.TokenTransferGasLimit)
		dbParams.SupportsEip1559 = pgtype.Bool{Bool: params.GasConfig.SupportsEIP1559, Valid: true}
		dbParams.GasOracleUrl = nullableString(params.GasConfig.GasOracleURL)
		dbParams.GasRefreshIntervalMs = pgtype.Int4{Int32: params.GasConfig.GasRefreshIntervalMs, Valid: true}
		dbParams.AverageBlockTimeMs = pgtype.Int4{Int32: params.GasConfig.AverageBlockTimeMs, Valid: true}
		dbParams.PeakHoursMultiplier = nullableNumeric(params.GasConfig.PeakHoursMultiplier)

		if params.GasConfig.GasPriorityLevels != nil {
			levelsJSON, err := json.Marshal(params.GasConfig.GasPriorityLevels)
			if err != nil {
				s.logger.Error("Failed to marshal gas priority levels",
					zap.Error(err))
				return nil, fmt.Errorf("failed to process gas priority levels: %w", err)
			}
			dbParams.GasPriorityLevels = levelsJSON
		}
	}

	network, err := s.queries.CreateNetwork(ctx, dbParams)
	if err != nil {
		s.logger.Error("Failed to create network",
			zap.String("name", params.Name),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	s.logger.Info("Network created successfully",
		zap.String("network_id", network.ID.String()),
		zap.String("name", params.Name))

	return &network, nil
}

// UpdateNetworkParams contains parameters for updating a network
type UpdateNetworkParams struct {
	ID                uuid.UUID
	Name              string
	Type              string
	NetworkType       string
	CircleNetworkType string
	BlockExplorerURL  string
	ChainID           int32
	IsTestnet         *bool
	Active            *bool
	LogoURL           string
	DisplayName       string
	ChainNamespace    string
	GasConfig         *UpdateGasConfigParams
}

// UpdateGasConfigParams contains gas configuration parameters for updates
type UpdateGasConfigParams struct {
	BaseFeeMultiplier     *float64
	PriorityFeeMultiplier *float64
	DeploymentGasLimit    *string
	TokenTransferGasLimit *string
	SupportsEIP1559       *bool
	GasOracleURL          *string
	GasRefreshIntervalMs  *int32
	GasPriorityLevels     map[string]interface{}
	AverageBlockTimeMs    *int32
	PeakHoursMultiplier   *float64
}

// UpdateNetwork updates an existing network
func (s *NetworkService) UpdateNetwork(ctx context.Context, params UpdateNetworkParams) (*db.Network, error) {
	dbParams := db.UpdateNetworkParams{
		ID:                params.ID,
		Name:              params.Name,
		Type:              params.Type,
		NetworkType:       db.NetworkType(params.NetworkType),
		CircleNetworkType: db.CircleNetworkType(params.CircleNetworkType),
		BlockExplorerUrl:  nullableString(params.BlockExplorerURL),
		ChainID:           params.ChainID,
		LogoUrl:           nullableString(params.LogoURL),
		DisplayName:       nullableString(params.DisplayName),
		ChainNamespace:    nullableString(params.ChainNamespace),
	}

	if params.IsTestnet != nil {
		dbParams.IsTestnet = *params.IsTestnet
	}
	if params.Active != nil {
		dbParams.Active = *params.Active
	}

	// Set gas config if provided
	if params.GasConfig != nil {
		if params.GasConfig.BaseFeeMultiplier != nil {
			dbParams.BaseFeeMultiplier = nullableNumeric(*params.GasConfig.BaseFeeMultiplier)
		}
		if params.GasConfig.PriorityFeeMultiplier != nil {
			dbParams.PriorityFeeMultiplier = nullableNumeric(*params.GasConfig.PriorityFeeMultiplier)
		}
		if params.GasConfig.DeploymentGasLimit != nil {
			dbParams.DeploymentGasLimit = nullableString(*params.GasConfig.DeploymentGasLimit)
		}
		if params.GasConfig.TokenTransferGasLimit != nil {
			dbParams.TokenTransferGasLimit = nullableString(*params.GasConfig.TokenTransferGasLimit)
		}
		if params.GasConfig.SupportsEIP1559 != nil {
			dbParams.SupportsEip1559 = pgtype.Bool{Bool: *params.GasConfig.SupportsEIP1559, Valid: true}
		}
		if params.GasConfig.GasOracleURL != nil {
			dbParams.GasOracleUrl = nullableString(*params.GasConfig.GasOracleURL)
		}
		if params.GasConfig.GasRefreshIntervalMs != nil {
			dbParams.GasRefreshIntervalMs = pgtype.Int4{Int32: *params.GasConfig.GasRefreshIntervalMs, Valid: true}
		}
		if params.GasConfig.AverageBlockTimeMs != nil {
			dbParams.AverageBlockTimeMs = pgtype.Int4{Int32: *params.GasConfig.AverageBlockTimeMs, Valid: true}
		}
		if params.GasConfig.PeakHoursMultiplier != nil {
			dbParams.PeakHoursMultiplier = nullableNumeric(*params.GasConfig.PeakHoursMultiplier)
		}

		if params.GasConfig.GasPriorityLevels != nil {
			levelsJSON, err := json.Marshal(params.GasConfig.GasPriorityLevels)
			if err != nil {
				s.logger.Error("Failed to marshal gas priority levels",
					zap.Error(err))
				return nil, fmt.Errorf("failed to process gas priority levels: %w", err)
			}
			dbParams.GasPriorityLevels = levelsJSON
		}
	}

	network, err := s.queries.UpdateNetwork(ctx, dbParams)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("network not found")
		}
		s.logger.Error("Failed to update network",
			zap.String("network_id", params.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update network: %w", err)
	}

	s.logger.Info("Network updated successfully",
		zap.String("network_id", network.ID.String()))

	return &network, nil
}

// DeleteNetwork deletes a network by ID
func (s *NetworkService) DeleteNetwork(ctx context.Context, networkID uuid.UUID) error {
	err := s.queries.DeleteNetwork(ctx, networkID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("network not found")
		}
		s.logger.Error("Failed to delete network",
			zap.String("network_id", networkID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete network: %w", err)
	}

	s.logger.Info("Network deleted successfully",
		zap.String("network_id", networkID.String()))

	return nil
}

// ListNetworksParams contains parameters for listing networks
type ListNetworksParams struct {
	IsTestnet *bool
	IsActive  *bool
}

// ListNetworks retrieves networks with optional filtering
func (s *NetworkService) ListNetworks(ctx context.Context, params ListNetworksParams) ([]db.Network, error) {
	dbParams := db.ListNetworksParams{}

	if params.IsTestnet != nil {
		dbParams.IsTestnet.Valid = true
		dbParams.IsTestnet.Bool = *params.IsTestnet
	}
	if params.IsActive != nil {
		dbParams.IsActive.Valid = true
		dbParams.IsActive.Bool = *params.IsActive
	}

	networks, err := s.queries.ListNetworks(ctx, dbParams)
	if err != nil {
		s.logger.Error("Failed to list networks", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve networks: %w", err)
	}

	return networks, nil
}

// ListActiveTokensByNetwork retrieves active tokens for a network
func (s *NetworkService) ListActiveTokensByNetwork(ctx context.Context, networkID uuid.UUID) ([]db.Token, error) {
	tokens, err := s.queries.ListActiveTokensByNetwork(ctx, networkID)
	if err != nil {
		s.logger.Error("Failed to list tokens by network",
			zap.String("network_id", networkID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve tokens: %w", err)
	}

	return tokens, nil
}

// Helper functions for nullable types
func nullableString(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// nullableNumeric converts float64 to pgtype.Numeric
func nullableNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	// Convert float to string and scan it
	strVal := fmt.Sprintf("%f", f)
	n.Scan(strVal)
	return n
}
