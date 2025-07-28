package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// TokenService handles business logic for token operations
type TokenService struct {
	queries   db.Querier
	cmcClient *coinmarketcap.Client
	logger    *zap.Logger
}

// NewTokenService creates a new token service
func NewTokenService(queries db.Querier, cmcClient *coinmarketcap.Client) *TokenService {
	return &TokenService{
		queries:   queries,
		cmcClient: cmcClient,
		logger:    logger.Log,
	}
}

// GetToken retrieves a token by ID
func (s *TokenService) GetToken(ctx context.Context, tokenID uuid.UUID) (*db.Token, error) {
	token, err := s.queries.GetToken(ctx, tokenID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("token not found")
		}
		s.logger.Error("Failed to get token",
			zap.String("token_id", tokenID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	return &token, nil
}

// GetTokenByAddress retrieves a token by network ID and contract address
func (s *TokenService) GetTokenByAddress(ctx context.Context, networkID uuid.UUID, contractAddress string) (*db.Token, error) {
	token, err := s.queries.GetTokenByAddress(ctx, db.GetTokenByAddressParams{
		NetworkID:       networkID,
		ContractAddress: contractAddress,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("token not found")
		}
		s.logger.Error("Failed to get token by address",
			zap.String("network_id", networkID.String()),
			zap.String("contract_address", contractAddress),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	return &token, nil
}

// ListTokens retrieves all tokens
func (s *TokenService) ListTokens(ctx context.Context) ([]db.Token, error) {
	tokens, err := s.queries.ListTokens(ctx)
	if err != nil {
		s.logger.Error("Failed to list tokens", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve tokens: %w", err)
	}

	return tokens, nil
}

// ListTokensByNetwork retrieves all tokens for a specific network
func (s *TokenService) ListTokensByNetwork(ctx context.Context, networkID uuid.UUID) ([]db.Token, error) {
	tokens, err := s.queries.ListTokensByNetwork(ctx, networkID)
	if err != nil {
		s.logger.Error("Failed to list tokens by network",
			zap.String("network_id", networkID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve tokens: %w", err)
	}

	return tokens, nil
}

// GetTokenQuote retrieves the price of a token in the specified fiat currency
func (s *TokenService) GetTokenQuote(ctx context.Context, quoteParams params.TokenQuoteParams) (*responses.TokenQuoteResult, error) {
	// Validate CMC client is available
	if s.cmcClient == nil {
		s.logger.Error("CoinMarketCap client is not initialized")
		return nil, fmt.Errorf("price service is unavailable")
	}

	// Get token information to get the symbol
	token, err := s.queries.GetToken(ctx, quoteParams.TokenID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("token not found")
		}
		s.logger.Error("Failed to get token",
			zap.String("token_id", quoteParams.TokenID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	// Prepare request using token symbol
	tokenSymbols := []string{token.Symbol}
	fiatSymbols := []string{quoteParams.ToCurrency}

	// Get quotes from CoinMarketCap
	cmcResponse, err := s.cmcClient.GetLatestQuotes(tokenSymbols, fiatSymbols)
	if err != nil {
		s.logger.Error("Failed to get quotes from CoinMarketCap",
			zap.String("token", token.Symbol),
			zap.String("fiat", quoteParams.ToCurrency),
			zap.Error(err))

		// Handle specific CMC client errors
		var cmcErr *coinmarketcap.Error
		if errors.As(err, &cmcErr) {
			return nil, fmt.Errorf("failed to get price: %s", cmcErr.Message)
		}
		return nil, fmt.Errorf("failed to fetch price data: %w", err)
	}

	// Extract the price from the response
	upperTokenSymbol := strings.ToUpper(token.Symbol)
	upperFiatSymbol := strings.ToUpper(quoteParams.ToCurrency)

	var exchangeRate float64
	found := false

	if tokenDataList, ok := cmcResponse.Data[upperTokenSymbol]; ok && len(tokenDataList) > 0 {
		tokenData := tokenDataList[0]
		if quoteData, ok := tokenData.Quote[upperFiatSymbol]; ok {
			exchangeRate = quoteData.Price
			found = true
		}
	}

	if !found {
		s.logger.Warn("Price not found in CoinMarketCap response",
			zap.String("token", upperTokenSymbol),
			zap.String("fiat", upperFiatSymbol),
			zap.Any("cmc_response_data", cmcResponse.Data))
		return nil, fmt.Errorf("price data not found for %s in %s", upperTokenSymbol, upperFiatSymbol)
	}

	// Convert AmountWei to token amount and calculate fiat amount
	// For simplicity, assuming AmountWei is already in token units (would need proper Wei conversion in real implementation)
	tokenAmountFloat := 1.0 // This would need proper conversion from Wei based on token decimals
	fiatAmount := tokenAmountFloat * exchangeRate

	return &responses.TokenQuoteResult{
		TokenAmount:   "1.0", // This should be properly calculated from AmountWei and token decimals
		FiatAmount:    fiatAmount,
		ExchangeRate:  exchangeRate,
		TokenDecimals: token.Decimals,
		QuotedAt:      "", // Would need current timestamp
		ExpiresAt:     "", // Would need expiration timestamp  
		PriceSource:   "coinmarketcap",
	}, nil
}

// CreateTokenParams contains parameters for creating a token
type CreateTokenParams struct {
	NetworkID       uuid.UUID
	GasToken        bool
	Name            string
	Symbol          string
	ContractAddress string
	Decimals        int32
	Active          bool
}

// CreateToken creates a new token
func (s *TokenService) CreateToken(ctx context.Context, params CreateTokenParams) (*db.Token, error) {
	// Validate required fields
	if params.Name == "" {
		return nil, fmt.Errorf("token name is required")
	}
	if params.Symbol == "" {
		return nil, fmt.Errorf("token symbol is required")
	}
	if params.ContractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}
	if params.Decimals < 0 {
		return nil, fmt.Errorf("decimals must be non-negative")
	}

	// Check if token already exists for this network and address
	existing, err := s.GetTokenByAddress(ctx, params.NetworkID, params.ContractAddress)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("token already exists for this network and contract address")
	}

	// Create the token
	token, err := s.queries.CreateToken(ctx, db.CreateTokenParams{
		NetworkID:       params.NetworkID,
		GasToken:        params.GasToken,
		Name:            params.Name,
		Symbol:          params.Symbol,
		ContractAddress: params.ContractAddress,
		Decimals:        params.Decimals,
		Active:          params.Active,
	})
	if err != nil {
		s.logger.Error("Failed to create token",
			zap.String("name", params.Name),
			zap.String("symbol", params.Symbol),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	s.logger.Info("Token created successfully",
		zap.String("token_id", token.ID.String()),
		zap.String("symbol", token.Symbol))

	return &token, nil
}

// UpdateTokenParams contains parameters for updating a token
type UpdateTokenParams struct {
	ID              uuid.UUID
	Name            *string
	Symbol          *string
	ContractAddress *string
	Decimals        *int32
	GasToken        *bool
	Active          *bool
}

// UpdateToken updates an existing token
func (s *TokenService) UpdateToken(ctx context.Context, params UpdateTokenParams) (*db.Token, error) {
	// Verify token exists
	_, err := s.GetToken(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Update the token
	// Note: This assumes an UpdateToken query exists in the database layer
	// If it doesn't, you'll need to add it to your SQL queries
	/*
	token, err := s.queries.UpdateToken(ctx, db.UpdateTokenParams{
		ID:              params.ID,
		Name:            params.Name,
		Symbol:          params.Symbol,
		ContractAddress: params.ContractAddress,
		Decimals:        params.Decimals,
		GasToken:        params.GasToken,
		Active:          params.Active,
	})
	if err != nil {
		s.logger.Error("Failed to update token",
			zap.String("token_id", params.ID.String()),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update token: %w", err)
	}

	s.logger.Info("Token updated successfully",
		zap.String("token_id", token.ID.String()))

	return &token, nil
	*/

	// For now, return an error since the update query doesn't exist
	return nil, fmt.Errorf("token update not implemented")
}

// DeleteToken soft deletes a token
func (s *TokenService) DeleteToken(ctx context.Context, tokenID uuid.UUID) error {
	// Verify token exists
	_, err := s.GetToken(ctx, tokenID)
	if err != nil {
		return err
	}

	// Delete the token
	// Note: This assumes a DeleteToken query exists in the database layer
	/*
	err = s.queries.DeleteToken(ctx, tokenID)
	if err != nil {
		s.logger.Error("Failed to delete token",
			zap.String("token_id", tokenID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to delete token: %w", err)
	}

	s.logger.Info("Token deleted successfully",
		zap.String("token_id", tokenID.String()))
	*/

	// For now, return an error since the delete query doesn't exist
	return fmt.Errorf("token deletion not implemented")
}

// ValidateTokenSymbol validates if a token symbol is valid
func (s *TokenService) ValidateTokenSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("token symbol cannot be empty")
	}
	if len(symbol) > 10 {
		return fmt.Errorf("token symbol too long (max 10 characters)")
	}
	// Additional validation can be added here
	return nil
}