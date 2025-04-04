package handlers

import (
	"context"
	"cyphera-api/internal/client/circle"
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// CircleHandler handles API requests related to Circle's wallet and user management features.
// It manages user initialization, wallet creation, and token management through Circle's API.
type CircleHandler struct {
	common       *CommonServices
	circleClient *circle.CircleClient
}

// NewCircleHandler creates a new instance of CircleHandler for handling Circle API operations.
// It requires CommonServices for database operations and a CircleClient for API interactions.
func NewCircleHandler(common *CommonServices, circleClient *circle.CircleClient) *CircleHandler {
	return &CircleHandler{
		common:       common,
		circleClient: circleClient,
	}
}

// RequestWithIdempotencyKey represents a request containing an idempotencyKey
type RequestWithIdempotencyKey struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// InitializeUserRequest represents the request to initialize a Circle user
type InitializeUserRequest struct {
	IdempotencyKey string   `json:"idempotency_key" binding:"required"`
	AccountType    string   `json:"account_type,omitempty"`
	Blockchains    []string `json:"blockchains" binding:"required"`
	Metadata       []struct {
		Name  string `json:"name"`
		RefID string `json:"ref_id"`
	} `json:"metadata,omitempty"`
}

// CreateWalletsRequest represents the request to create Circle wallets
type CreateWalletsRequest struct {
	IdempotencyKey string   `json:"idempotency_key" binding:"required"`
	Blockchains    []string `json:"blockchains" binding:"required"`
	AccountType    string   `json:"account_type,omitempty"`
	Metadata       []struct {
		Name  string `json:"name"`
		RefID string `json:"ref_id"`
	} `json:"metadata,omitempty"`
}

// GetWalletBalanceParams represents query parameters for the wallet balance endpoint
type GetWalletBalanceParams struct {
	IncludeAll   bool   `form:"include_all"`
	Name         string `form:"name"`
	TokenAddress string `form:"token_address"`
	Standard     string `form:"standard"`
	PageSize     int    `form:"page_size"`
	PageBefore   string `form:"page_before"`
	PageAfter    string `form:"page_after"`
}

// ListWalletsParams represents query parameters for listing wallets
type ListWalletsParams struct {
	Address     string `form:"address"`
	Blockchain  string `form:"blockchain"`
	ScaCore     string `form:"sca_core"`
	WalletSetID string `form:"wallet_set_id"`
	RefID       string `form:"ref_id"`
	From        string `form:"from"`
	To          string `form:"to"`
	PageSize    int    `form:"page_size"`
	PageBefore  string `form:"page_before"`
	PageAfter   string `form:"page_after"`
}

// ValidateAddressRequest represents the request to validate a blockchain address
type ValidateAddressRequest struct {
	Blockchain string `json:"blockchain" binding:"required"`
	Address    string `json:"address" binding:"required"`
}

// EstimateTransferFeeRequest represents the request to estimate transfer transaction fees
type EstimateTransferFeeRequest struct {
	DestinationAddress string   `json:"destination_address" binding:"required"`
	Amounts            []string `json:"amounts" binding:"required"`
	WalletID           string   `json:"wallet_id,omitempty"`
	SourceAddress      string   `json:"source_address,omitempty"`
	Blockchain         string   `json:"blockchain,omitempty"`
	TokenID            string   `json:"token_id,omitempty"`
	TokenAddress       string   `json:"token_address,omitempty"`
	NftTokenIds        []string `json:"nft_token_ids,omitempty"`
}

// CreateTransferRequest represents the request to create a transfer transaction challenge
type CreateTransferRequest struct {
	IdempotencyKey     string   `json:"idempotency_key" binding:"required"`
	DestinationAddress string   `json:"destination_address" binding:"required"`
	WalletID           string   `json:"wallet_id,omitempty"`
	SourceAddress      string   `json:"source_address,omitempty"`
	Blockchain         string   `json:"blockchain,omitempty"`
	Amounts            []string `json:"amounts,omitempty"`
	TokenID            string   `json:"token_id,omitempty"`
	TokenAddress       string   `json:"token_address,omitempty"`
	FeeLevel           string   `json:"fee_level,omitempty"`
	GasLimit           string   `json:"gas_limit,omitempty"`
	GasPrice           string   `json:"gas_price,omitempty"`
	MaxFee             string   `json:"max_fee,omitempty"`
	PriorityFee        string   `json:"priority_fee,omitempty"`
	NftTokenIds        []string `json:"nft_token_ids,omitempty"`
	RefID              string   `json:"ref_id,omitempty"`
}

// ListTransactionsParams represents query parameters for listing transactions
type ListTransactionsParams struct {
	Blockchain         string `form:"blockchain"`
	DestinationAddress string `form:"destination_address"`
	IncludeAll         bool   `form:"include_all"`
	Operation          string `form:"operation"`
	State              string `form:"state"`
	TxHash             string `form:"tx_hash"`
	TxType             string `form:"tx_type"`
	UserID             string `form:"user_id"`
	WalletIDs          string `form:"wallet_ids"`
	From               string `form:"from"`
	To                 string `form:"to"`
	PageSize           int    `form:"page_size"`
	PageBefore         string `form:"page_before"`
	PageAfter          string `form:"page_after"`
}

// CreateUserToken godoc
// @Summary Create a Circle user token
// @Description Creates a user token for Circle API operations and stores or updates the user in the database
// @Tags circle
// @Accept json
// @Produce json
// @Param user_id path string true "Circle User ID"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/users/{user_id}/token [post]
func (h *CircleHandler) CreateUserToken(c *gin.Context) {
	// Validate account ID
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userID := c.Param("user_id")
	if userID == "" {
		sendError(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	// Call Circle API to create user token
	tokenResponse, err := h.circleClient.CreateUserToken(c.Request.Context(), userID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create user token", err)
		return
	}

	// Check if a circle user already exists for this account
	_, err = h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)
	if err == nil {
		// User exists, update the token and encryption key
		_, err = h.common.db.UpdateCircleUserByAccountID(c.Request.Context(), db.UpdateCircleUserByAccountIDParams{
			Token:         tokenResponse.Data.UserToken,
			EncryptionKey: tokenResponse.Data.EncryptionKey,
			AccountID:     accountID,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to update circle user", err)
			return
		}
	} else {
		// User doesn't exist, create a new one
		_, err = h.common.db.CreateCircleUser(c.Request.Context(), db.CreateCircleUserParams{
			ID:            uuid.New(),
			AccountID:     accountID,
			Token:         tokenResponse.Data.UserToken,
			EncryptionKey: tokenResponse.Data.EncryptionKey,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create circle user", err)
			return
		}
	}

	sendSuccess(c, http.StatusOK, tokenResponse)
}

// GetUserByToken godoc
// @Summary Get Circle user by token
// @Description Retrieves user details using a Circle user token and updates database if token changed
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/users/token [get]
func (h *CircleHandler) GetUserByToken(c *gin.Context) {
	// Validate account ID
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Call Circle API to get user details
	userResponse, err := h.circleClient.GetUserByToken(c.Request.Context(), userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get user details", err)
		return
	}

	// Check if a circle user exists for this account
	circleUser, err := h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)

	// If user exists but token doesn't match, update the token
	if err == nil && circleUser.Token != userToken {
		// Update token to ensure our database is in sync
		_, err = h.common.db.UpdateCircleUserByAccountID(c.Request.Context(), db.UpdateCircleUserByAccountIDParams{
			Token:     userToken,
			AccountID: accountID,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to update circle user token", err)
			return
		}
	}

	sendSuccess(c, http.StatusOK, userResponse)
}

// GetUserByID godoc
// @Summary Get Circle user by ID
// @Description Retrieves user details by Circle user ID
// @Tags circle
// @Accept json
// @Produce json
// @Param user_id path string true "Circle User ID"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/users/{user_id} [get]
func (h *CircleHandler) GetUserByID(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userID := c.Param("user_id")
	if userID == "" {
		sendError(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	// Call Circle API to get user details
	userResponse, err := h.circleClient.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get user details", err)
		return
	}

	sendSuccess(c, http.StatusOK, userResponse)
}

// GetChallenge godoc
// @Summary Get Circle challenge by ID
// @Description Retrieves challenge details by Circle challenge ID
// @Tags circle
// @Accept json
// @Produce json
// @Param challenge_id path string true "Circle Challenge ID"
// @Param user_token header string true "Circle User Token"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/challenges/{challenge_id} [get]
func (h *CircleHandler) GetChallenge(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	challengeID := c.Param("challenge_id")
	if challengeID == "" {
		sendError(c, http.StatusBadRequest, "Challenge ID is required", nil)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Call Circle API to get challenge details
	challengeResponse, err := h.circleClient.GetChallenge(c.Request.Context(), challengeID, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get challenge details", err)
		return
	}

	sendSuccess(c, http.StatusOK, challengeResponse)
}

// InitializeUser godoc
// @Summary Initialize Circle user and create wallets
// @Description Creates a challenge for user initialization and wallet creation (requires existing Circle user in database)
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param body body InitializeUserRequest true "User initialization request"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/users/initialize [post]
func (h *CircleHandler) InitializeUser(c *gin.Context) {
	// Validate account ID
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	var req InitializeUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Ensure the Circle user is in our database
	_, err = h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Circle user not found for this account", err)
		return
	}

	// Convert our req to the client's required format
	initRequest := circle.InitializeUserRequest{
		IdempotencyKey: req.IdempotencyKey,
		AccountType:    req.AccountType,
		Blockchains:    req.Blockchains,
	}

	// Map metadata if provided
	if len(req.Metadata) > 0 {
		initRequest.Metadata = make([]struct {
			Name  string `json:"name"`
			RefID string `json:"refId"`
		}, len(req.Metadata))

		for i, meta := range req.Metadata {
			initRequest.Metadata[i] = struct {
				Name  string `json:"name"`
				RefID string `json:"refId"`
			}{
				Name:  meta.Name,
				RefID: meta.RefID,
			}
		}
	}

	// Call Circle API to initialize user
	initResponse, err := h.circleClient.InitializeUser(c.Request.Context(), initRequest, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to initialize user", err)
		return
	}

	// Store the challenge ID for future reference
	// This is useful for checking challenge status later
	challengeID := initResponse.Data.ChallengeID

	sendSuccess(c, http.StatusOK, map[string]interface{}{
		"challenge_id": challengeID,
		"message":      "User initialization started. The wallets will be created once the challenge is completed.",
	})
}

// CreatePinChallenge godoc
// @Summary Create a PIN challenge
// @Description Creates a challenge for PIN setup
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param body body RequestWithIdempotencyKey true "Request with idempotency key"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/users/pin/create [post]
func (h *CircleHandler) CreatePinChallenge(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	var req RequestWithIdempotencyKey
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Call Circle API to create PIN challenge
	pinResponse, err := h.circleClient.CreatePinChallenge(c.Request.Context(), req.IdempotencyKey, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create PIN challenge", err)
		return
	}

	sendSuccess(c, http.StatusOK, pinResponse)
}

// UpdatePinChallenge godoc
// @Summary Update a PIN challenge
// @Description Creates a challenge to update a user's PIN via the existing PIN
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param body body RequestWithIdempotencyKey true "Request with idempotency key"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/users/pin/update [put]
func (h *CircleHandler) UpdatePinChallenge(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	var req RequestWithIdempotencyKey
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Call Circle API to update PIN challenge
	pinResponse, err := h.circleClient.UpdatePinChallenge(c.Request.Context(), req.IdempotencyKey, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update PIN challenge", err)
		return
	}

	sendSuccess(c, http.StatusOK, pinResponse)
}

// CreatePinRestoreChallenge godoc
// @Summary Create a PIN restore challenge
// @Description Creates a challenge to change a user's PIN via Security Questions
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param body body RequestWithIdempotencyKey true "Request with idempotency key"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/users/pin/restore [post]
func (h *CircleHandler) CreatePinRestoreChallenge(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	var req RequestWithIdempotencyKey
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Call Circle API to create PIN restore challenge
	pinResponse, err := h.circleClient.CreatePinRestoreChallenge(c.Request.Context(), req.IdempotencyKey, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create PIN restore challenge", err)
		return
	}

	sendSuccess(c, http.StatusOK, pinResponse)
}

// CreateWallets godoc
// @Summary Create Circle wallets
// @Description Creates wallets for a Circle user and tracks wallet creation challenge
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param body body CreateWalletsRequest true "Wallet creation request"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/wallets [post]
func (h *CircleHandler) CreateWallets(c *gin.Context) {
	// Validate account ID
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	var req CreateWalletsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Get the Circle user from our database
	circleUser, err := h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Circle user not found for this account", err)
		return
	}

	// Prepare the Circle API request
	circleRequest := circle.CreateWalletsRequest{
		IdempotencyKey: req.IdempotencyKey,
		Blockchains:    req.Blockchains,
		AccountType:    req.AccountType,
	}

	// Map metadata if provided
	if len(req.Metadata) > 0 {
		circleRequest.Metadata = make([]struct {
			Name  string `json:"name"`
			RefID string `json:"refId"`
		}, len(req.Metadata))

		for i, meta := range req.Metadata {
			circleRequest.Metadata[i] = struct {
				Name  string `json:"name"`
				RefID string `json:"refId"`
			}{
				Name:  meta.Name,
				RefID: meta.RefID,
			}
		}
	}

	// Call Circle API to create wallets
	walletsResponse, err := h.circleClient.CreateWallets(c.Request.Context(), circleRequest, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create wallets", err)
		return
	}

	// Store the challenge ID for future wallet creation tracking
	challengeID := walletsResponse.Data.ChallengeID

	// Note: Actual wallet creation happens asynchronously after the challenge is completed
	// We should attempt to check if the challenge is already complete, and if so, create wallet entries

	// Get the challenge status to see if it's already completed (unlikely but possible)
	challengeResponse, err := h.circleClient.GetChallenge(c.Request.Context(), challengeID, userToken)
	if err != nil {
		// If we can't get challenge details, we'll just continue; wallets will be created later when ListWallets is called
		logger.Warn("Could not check challenge status",
			zap.String("challenge_id", challengeID),
			zap.Error(err))
	} else if challengeResponse.Data.Challenge.Status == "complete" {
		// Challenge already completed, immediately list wallets and create them in our database
		// This is unusual but possible for very fast challenge completion

		// List wallets to get their details
		listParams := &circle.ListWalletsParams{}
		walletsListResponse, err := h.circleClient.ListWallets(c.Request.Context(), userToken, listParams)
		if err == nil && len(walletsListResponse.Data.Wallets) > 0 {
			// Begin a transaction for database operations
			tx, qtx, err := h.common.BeginTx(c.Request.Context())
			if err != nil {
				logger.Error("Failed to begin transaction for wallet creation",
					zap.Error(err))
			} else {
				defer tx.Rollback(c.Request.Context())

				// Process each wallet and create Cyphera wallet entries
				for _, walletData := range walletsListResponse.Data.Wallets {
					err = h.createCypheraWalletEntry(c.Request.Context(), qtx, walletData, accountID, circleUser.ID)
					if err != nil {
						logger.Error("Failed to create Cyphera wallet entry",
							zap.String("address", walletData.Address),
							zap.Error(err))
						// Continue processing other wallets
						continue
					}
				}

				// Commit transaction
				if err := tx.Commit(c.Request.Context()); err != nil {
					logger.Error("Failed to commit transaction for wallet creation",
						zap.Error(err))
				}
			}
		}
	}

	// Return the challenge info regardless of whether we've already created the wallets
	sendSuccess(c, http.StatusOK, map[string]interface{}{
		"challenge_id": challengeID,
		"user_id":      circleUser.ID.String(),
		"message":      "Wallet creation challenge initiated. Check challenge status to confirm wallet creation.",
	})
}

// createCypheraWalletEntry creates a Cyphera wallet entry for a Circle wallet in our database
// This is used when we receive wallet data from Circle and need to store it in our system
func (h *CircleHandler) createCypheraWalletEntry(ctx context.Context, qtx *db.Queries, walletData circle.Wallet, accountID uuid.UUID, circleUserID uuid.UUID) error {
	// Look up network ID and chain ID
	networkID, chainID, err := h.lookupNetworkID(ctx, walletData.Blockchain)
	if err != nil {
		// Log the error but continue - we can still process with an unknown network
		logger.Warn("Could not find network for blockchain",
			zap.String("blockchain", walletData.Blockchain),
			zap.Error(err))
	}

	var networkIDPgType pgtype.UUID
	if networkID != uuid.Nil {
		networkIDPgType.Bytes = networkID
		networkIDPgType.Valid = true
	}

	// Check if wallet already exists in our database
	var dbWallet db.Wallet
	dbWallet, err = h.common.db.GetWalletByAddress(ctx, db.GetWalletByAddressParams{
		WalletAddress: walletData.Address,
		NetworkType:   getNetworkType(walletData.Blockchain),
	})

	// Check if the error is "no rows found" or another error
	walletExists := true
	if err != nil {
		if err == pgx.ErrNoRows {
			// Wallet doesn't exist, that's fine
			walletExists = false
		} else {
			// This is a real database error
			return fmt.Errorf("failed to check if wallet exists: %w", err)
		}
	}

	// Create metadata with wallet information
	metadata := map[string]interface{}{
		"circle_wallet_set_id": walletData.WalletSetID,
		"circle_custody_type":  walletData.CustodyType,
		"circle_ref_id":        walletData.RefID,
		"circle_blockchain":    walletData.Blockchain,
		"circle_user_id":       walletData.UserID,
		"circle_account_type":  walletData.AccountType,
		"circle_name":          walletData.Name,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if walletExists {
		// Wallet exists, update it
		_, err = qtx.UpdateWallet(ctx, db.UpdateWalletParams{
			ID:       dbWallet.ID,
			Metadata: metadataJSON,
		})
		if err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}

		// Update Circle wallet entry
		_, err = qtx.UpdateCircleWalletState(ctx, db.UpdateCircleWalletStateParams{
			WalletID: dbWallet.ID,
			State:    walletData.State,
		})
		if err != nil {
			return fmt.Errorf("failed to update Circle wallet entry: %w", err)
		}
	} else {
		// Wallet doesn't exist, create it
		nicknamePgText := pgtype.Text{}
		nicknamePgText.String = walletData.Name
		nicknamePgText.Valid = walletData.Name != ""

		// First create the wallet
		newWallet, err := qtx.CreateWallet(ctx, db.CreateWalletParams{
			AccountID:     accountID,
			WalletType:    "circle_wallet",
			WalletAddress: walletData.Address,
			NetworkType:   getNetworkType(walletData.Blockchain),
			NetworkID:     networkIDPgType,
			Nickname:      nicknamePgText,
			Metadata:      metadataJSON,
		})
		if err != nil {
			return fmt.Errorf("failed to create wallet: %w", err)
		}

		// Then create the Circle wallet entry
		_, err = qtx.CreateCircleWalletEntry(ctx, db.CreateCircleWalletEntryParams{
			WalletID:       newWallet.ID,
			CircleUserID:   circleUserID,
			CircleWalletID: walletData.ID,
			ChainID:        chainID,
			State:          walletData.State,
		})
		if err != nil {
			return fmt.Errorf("failed to create Circle wallet entry: %w", err)
		}
	}

	return nil
}

// GetWallet godoc
// @Summary Get Circle wallet by ID
// @Description Retrieves wallet details by Circle wallet ID and stores/updates wallet in database
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param wallet_id path string true "Circle Wallet ID"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/wallets/{wallet_id} [get]
func (h *CircleHandler) GetWallet(c *gin.Context) {
	// Validate account ID
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	walletID := c.Param("wallet_id")
	if walletID == "" {
		sendError(c, http.StatusBadRequest, "Wallet ID is required", nil)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Get the Circle user from our database
	circleUser, err := h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Circle user not found for this account", err)
		return
	}

	// Call Circle API to get wallet details
	walletResponse, err := h.circleClient.GetWallet(c.Request.Context(), walletID, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get wallet details", err)
		return
	}

	// Extract wallet data from response
	walletData := walletResponse.Data.Wallet

	// Look up network ID and chain ID
	networkID, chainID, err := h.lookupNetworkID(c.Request.Context(), walletData.Blockchain)
	if err != nil {
		// Log the error but continue - we can still process with an unknown network
		logger.Warn("Could not find network for blockchain",
			zap.String("blockchain", walletData.Blockchain),
			zap.Error(err))
		// Don't return, continue processing
	}

	var networkIDPgType pgtype.UUID
	if networkID != uuid.Nil {
		networkIDPgType.Bytes = networkID
		networkIDPgType.Valid = true
	}

	// Begin a transaction
	tx, qtx, err := h.common.BeginTx(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to begin transaction", err)
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Check if wallet already exists in our database
	var dbWallet db.Wallet
	dbWallet, err = h.common.db.GetWalletByAddress(c.Request.Context(), db.GetWalletByAddressParams{
		WalletAddress: walletData.Address,
		NetworkType:   getNetworkType(walletData.Blockchain),
	})
	// Check if the error is "no rows found" or another error
	walletExists := true
	if err != nil {
		if err == pgx.ErrNoRows {
			// Wallet doesn't exist, that's fine
			walletExists = false
		} else {
			// This is a real database error
			sendError(c, http.StatusInternalServerError, "Failed to check if wallet exists", err)
			return
		}
	}

	// Create metadata with wallet information
	metadata := map[string]interface{}{
		"circle_wallet_set_id": walletData.WalletSetID,
		"circle_custody_type":  walletData.CustodyType,
		"circle_ref_id":        walletData.RefID,
		"circle_blockchain":    walletData.Blockchain,
		"circle_user_id":       walletData.UserID,
		"circle_account_type":  walletData.AccountType,
		"circle_name":          walletData.Name,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to marshal metadata", err)
		return
	}

	if walletExists {
		// Wallet exists, update it
		_, err = qtx.UpdateWallet(c.Request.Context(), db.UpdateWalletParams{
			ID:       dbWallet.ID,
			Metadata: metadataJSON,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to update wallet", err)
			return
		}

		// Update Circle wallet entry
		_, err = qtx.UpdateCircleWalletState(c.Request.Context(), db.UpdateCircleWalletStateParams{
			WalletID: dbWallet.ID,
			State:    walletData.State,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to update Circle wallet entry", err)
			return
		}
	} else {
		// Wallet doesn't exist, create it
		nicknamePgText := pgtype.Text{}
		nicknamePgText.String = walletData.Name
		nicknamePgText.Valid = walletData.Name != ""

		// First create the wallet
		newWallet, err := qtx.CreateWallet(c.Request.Context(), db.CreateWalletParams{
			AccountID:     accountID,
			WalletType:    "circle_wallet",
			WalletAddress: walletData.Address,
			NetworkType:   getNetworkType(walletData.Blockchain),
			NetworkID:     networkIDPgType,
			Nickname:      nicknamePgText,
			Metadata:      metadataJSON,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create wallet", err)
			return
		}

		// Then create the Circle wallet entry
		_, err = qtx.CreateCircleWalletEntry(c.Request.Context(), db.CreateCircleWalletEntryParams{
			WalletID:       newWallet.ID,
			CircleUserID:   circleUser.ID,
			CircleWalletID: walletData.ID,
			ChainID:        chainID,
			State:          walletData.State,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create Circle wallet entry", err)
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit(c.Request.Context()); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to commit transaction", err)
		return
	}

	// Return the wallet data from Circle
	sendSuccess(c, http.StatusOK, walletResponse)
}

// getNetworkType determines the NetworkType enum value from a Circle blockchain identifier.
// It returns NetworkTypeEvm for Ethereum-based chains and NetworkTypeSolana for Solana chains.
func getNetworkType(blockchain string) db.NetworkType {
	switch blockchain {
	case "ETH", "ETH-SEPOLIA", "ARB", "ARB-SEPOLIA", "MATIC", "MATIC-AMOY", "BASE", "BASE-SEPOLIA", "UNICHAIN", "UNICHAIN-SEPOLIA":
		return db.NetworkTypeEvm
	case "SOL", "SOL-DEVNET":
		return db.NetworkTypeSolana
	default:
		return db.NetworkTypeEvm // Default to EVM
	}
}

// getCircleNetworkType converts a Circle blockchain identifier to the corresponding CircleNetworkType enum.
// It returns an error if the blockchain is not supported.
func getCircleNetworkType(blockchain string) (db.CircleNetworkType, error) {
	switch blockchain {
	case "ETH":
		return db.CircleNetworkTypeETH, nil
	case "ETH-SEPOLIA":
		return db.CircleNetworkTypeETHSEPOLIA, nil
	case "ARB":
		return db.CircleNetworkTypeARB, nil
	case "ARB-SEPOLIA":
		return db.CircleNetworkTypeARBSEPOLIA, nil
	case "MATIC":
		return db.CircleNetworkTypeMATIC, nil
	case "MATIC-AMOY":
		return db.CircleNetworkTypeMATICAMOY, nil
	case "BASE":
		return db.CircleNetworkTypeBASE, nil
	case "BASE-SEPOLIA":
		return db.CircleNetworkTypeBASESEPOLIA, nil
	case "UNICHAIN":
		return db.CircleNetworkTypeUNICHAIN, nil
	case "UNICHAIN-SEPOLIA":
		return db.CircleNetworkTypeUNICHAINSEPOLIA, nil
	case "SOL":
		return db.CircleNetworkTypeSOL, nil
	case "SOL-DEVNET":
		return db.CircleNetworkTypeSOLDEVNET, nil
	default:
		return "", fmt.Errorf("unsupported blockchain: %s", blockchain)
	}
}

// GetWalletBalance godoc
// @Summary Get Circle wallet balance
// @Description Retrieves token balances for a Circle wallet and ensures wallet is stored in database
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param wallet_id path string true "Circle Wallet ID"
// @Param include_all query boolean false "Include all token balances"
// @Param name query string false "Filter by token name"
// @Param token_address query string false "Filter by token address"
// @Param standard query string false "Filter by token standard"
// @Param page_size query integer false "Page size for pagination"
// @Param page_before query string false "Page before cursor for pagination"
// @Param page_after query string false "Page after cursor for pagination"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/wallets/{wallet_id}/balances [get]
func (h *CircleHandler) GetWalletBalance(c *gin.Context) {
	// Validate account ID
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	walletID := c.Param("wallet_id")
	if walletID == "" {
		sendError(c, http.StatusBadRequest, "Wallet ID is required", nil)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Parse query parameters
	var params GetWalletBalanceParams
	if err := c.ShouldBindQuery(&params); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid query parameters", err)
		return
	}

	// First, get the wallet details to ensure we have it in our database
	walletResponse, err := h.circleClient.GetWallet(c.Request.Context(), walletID, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get wallet details", err)
		return
	}

	// Extract wallet data
	walletData := walletResponse.Data.Wallet

	// Get the Circle user from our database
	circleUser, err := h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Circle user not found for this account", err)
		return
	}

	// Look up network ID and chain ID
	networkID, chainID, err := h.lookupNetworkID(c.Request.Context(), walletData.Blockchain)
	if err != nil {
		// Log the error but continue - we can still process with an unknown network
		logger.Warn("Could not find network for blockchain",
			zap.String("blockchain", walletData.Blockchain),
			zap.Error(err))
		// Don't return, continue processing
	}

	var networkIDPgType pgtype.UUID
	if networkID != uuid.Nil {
		networkIDPgType.Bytes = networkID
		networkIDPgType.Valid = true
	}

	// Check if this wallet already exists in our database
	_, err = h.common.db.GetWalletByAddress(c.Request.Context(), db.GetWalletByAddressParams{
		WalletAddress: walletData.Address,
		NetworkType:   getNetworkType(walletData.Blockchain),
	})

	// If wallet doesn't exist, create it
	if err != nil {
		// Begin transaction for creating wallet
		tx, qtx, err := h.common.BeginTx(c.Request.Context())
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to begin transaction", err)
			return
		}
		defer tx.Rollback(c.Request.Context())

		// Create metadata with wallet information
		metadata := map[string]interface{}{
			"circle_wallet_set_id": walletData.WalletSetID,
			"circle_custody_type":  walletData.CustodyType,
			"circle_ref_id":        walletData.RefID,
			"circle_blockchain":    walletData.Blockchain,
			"circle_user_id":       walletData.UserID,
			"circle_account_type":  walletData.AccountType,
			"circle_name":          walletData.Name,
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to marshal metadata", err)
			return
		}

		nicknamePgText := pgtype.Text{}
		nicknamePgText.String = walletData.Name
		nicknamePgText.Valid = walletData.Name != ""

		// Create the wallet
		newWallet, err := qtx.CreateWallet(c.Request.Context(), db.CreateWalletParams{
			AccountID:     accountID,
			WalletType:    "circle_wallet",
			WalletAddress: walletData.Address,
			NetworkType:   getNetworkType(walletData.Blockchain),
			NetworkID:     networkIDPgType,
			Nickname:      nicknamePgText,
			Metadata:      metadataJSON,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create wallet", err)
			return
		}

		// Create the Circle wallet entry
		_, err = qtx.CreateCircleWalletEntry(c.Request.Context(), db.CreateCircleWalletEntryParams{
			WalletID:       newWallet.ID,
			CircleUserID:   circleUser.ID,
			CircleWalletID: walletData.ID,
			ChainID:        chainID,
			State:          walletData.State,
		})
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create Circle wallet entry", err)
			return
		}

		// Commit transaction
		if err := tx.Commit(c.Request.Context()); err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to commit transaction", err)
			return
		}
	}

	// Prepare Circle API parameters
	circleParams := &circle.GetWalletBalanceParams{}

	if params.IncludeAll {
		includeAll := params.IncludeAll
		circleParams.IncludeAll = &includeAll
	}

	if params.Name != "" {
		name := params.Name
		circleParams.Name = &name
	}

	if params.TokenAddress != "" {
		tokenAddress := params.TokenAddress
		circleParams.TokenAddress = &tokenAddress
	}

	if params.Standard != "" {
		standard := params.Standard
		circleParams.Standard = &standard
	}

	if params.PageSize > 0 {
		pageSize := params.PageSize
		circleParams.PageSize = &pageSize
	}

	if params.PageBefore != "" {
		pageBefore := params.PageBefore
		circleParams.PageBefore = &pageBefore
	}

	if params.PageAfter != "" {
		pageAfter := params.PageAfter
		circleParams.PageAfter = &pageAfter
	}

	// Call Circle API to get wallet balances
	balanceResponse, err := h.circleClient.GetWalletBalance(c.Request.Context(), walletID, userToken, circleParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get wallet balances", err)
		return
	}

	sendSuccess(c, http.StatusOK, balanceResponse)
}

// ListWallets godoc
// @Summary List Circle wallets
// @Description Lists wallets for a Circle user and stores them in the database
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param address query string false "Filter by wallet address"
// @Param blockchain query string false "Filter by blockchain"
// @Param page_size query int false "Number of results per page"
// @Param page_before query string false "Cursor for previous page"
// @Param page_after query string false "Cursor for next page"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/wallets [get]
func (h *CircleHandler) ListWallets(c *gin.Context) {
	// Validate account ID
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Get the Circle user from our database
	circleUser, err := h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Circle user not found for this account", err)
		return
	}

	// Parse query parameters
	var address, blockchain, pageBefore, pageAfter string
	var pageSize int

	if val := c.Query("address"); val != "" {
		address = val
	}
	if val := c.Query("blockchain"); val != "" {
		blockchain = val
	}
	if val := c.Query("page_size"); val != "" {
		var parseErr error
		pageSize, parseErr = strconv.Atoi(val)
		if parseErr != nil {
			sendError(c, http.StatusBadRequest, "Invalid page_size parameter", parseErr)
			return
		}
	}
	if val := c.Query("page_before"); val != "" {
		pageBefore = val
	}
	if val := c.Query("page_after"); val != "" {
		pageAfter = val
	}

	// Prepare Circle API params
	params := &circle.ListWalletsParams{}
	if address != "" {
		params.Address = &address
	}
	if blockchain != "" {
		params.Blockchain = &blockchain
	}
	if pageSize > 0 {
		params.PageSize = &pageSize
	}
	if pageBefore != "" {
		params.PageBefore = &pageBefore
	}
	if pageAfter != "" {
		params.PageAfter = &pageAfter
	}

	// Call Circle API to list wallets
	walletsResponse, err := h.circleClient.ListWallets(c.Request.Context(), userToken, params)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list wallets", err)
		return
	}

	// Begin a transaction for database operations
	tx, qtx, err := h.common.BeginTx(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to begin transaction", err)
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Process each wallet and store/update in database
	for _, walletData := range walletsResponse.Data.Wallets {
		// Look up network ID and chain ID
		networkID, chainID, err := h.lookupNetworkID(c.Request.Context(), walletData.Blockchain)
		if err != nil {
			// Log the error but continue - we can still process with an unknown network
			logger.Warn("Could not find network for blockchain",
				zap.String("blockchain", walletData.Blockchain),
				zap.Error(err))
			// Don't return, continue processing
		}
		var networkIDPgType pgtype.UUID
		if networkID != uuid.Nil {
			networkIDPgType.Bytes = networkID
			networkIDPgType.Valid = true
		}

		// Check if wallet already exists in our database
		var dbWallet db.Wallet
		dbWallet, err = h.common.db.GetWalletByAddress(c.Request.Context(), db.GetWalletByAddressParams{
			WalletAddress: walletData.Address,
			NetworkType:   getNetworkType(walletData.Blockchain),
		})

		// Check if the error is "no rows found" or another error
		walletExists := true
		if err != nil {
			if err == pgx.ErrNoRows {
				// Wallet doesn't exist, that's fine
				walletExists = false
			} else {
				// This is a real database error
				sendError(c, http.StatusInternalServerError, "Failed to check if wallet exists", err)
				return
			}
		}

		// Create metadata with wallet information
		metadata := map[string]interface{}{
			"circle_wallet_set_id": walletData.WalletSetID,
			"circle_custody_type":  walletData.CustodyType,
			"circle_ref_id":        walletData.RefID,
			"circle_blockchain":    walletData.Blockchain,
			"circle_user_id":       walletData.UserID,
			"circle_account_type":  walletData.AccountType,
			"circle_name":          walletData.Name,
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to marshal metadata", err)
			return
		}

		if walletExists {
			// Wallet exists, update it
			_, err = qtx.UpdateWallet(c.Request.Context(), db.UpdateWalletParams{
				ID:       dbWallet.ID,
				Metadata: metadataJSON,
			})
			if err != nil {
				sendError(c, http.StatusInternalServerError, "Failed to update wallet", err)
				return
			}

			// Update Circle wallet entry
			_, err = qtx.UpdateCircleWalletState(c.Request.Context(), db.UpdateCircleWalletStateParams{
				WalletID: dbWallet.ID,
				State:    walletData.State,
			})
			if err != nil {
				sendError(c, http.StatusInternalServerError, "Failed to update Circle wallet entry", err)
				return
			}
		} else {
			// Wallet doesn't exist, create it
			nicknamePgText := pgtype.Text{}
			nicknamePgText.String = walletData.Name
			nicknamePgText.Valid = walletData.Name != ""

			// First create the wallet
			newWallet, err := qtx.CreateWallet(c.Request.Context(), db.CreateWalletParams{
				AccountID:     accountID,
				WalletType:    "circle_wallet",
				WalletAddress: walletData.Address,
				NetworkType:   getNetworkType(walletData.Blockchain),
				NetworkID:     networkIDPgType,
				Nickname:      nicknamePgText,
				Metadata:      metadataJSON,
			})
			if err != nil {
				sendError(c, http.StatusInternalServerError, "Failed to create wallet", err)
				return
			}

			// Then create the Circle wallet entry
			_, err = qtx.CreateCircleWalletEntry(c.Request.Context(), db.CreateCircleWalletEntryParams{
				WalletID:       newWallet.ID,
				CircleUserID:   circleUser.ID,
				CircleWalletID: walletData.ID,
				ChainID:        chainID,
				State:          walletData.State,
			})
			if err != nil {
				sendError(c, http.StatusInternalServerError, "Failed to create Circle wallet entry", err)
				return
			}
		}
	}

	// Commit all database changes
	if err := tx.Commit(c.Request.Context()); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to commit transaction", err)
		return
	}

	sendSuccess(c, http.StatusOK, walletsResponse)
}

// lookupNetworkID finds a Network in the database by Circle network type.
// It returns the network ID and chain ID if found, or returns a best-guess chain ID
// based on common networks if not found.
func (h *CircleHandler) lookupNetworkID(ctx context.Context, blockchain string) (uuid.UUID, int32, error) {
	circleNetworkType, err := getCircleNetworkType(blockchain)
	if err != nil {
		return uuid.Nil, 0, err
	}

	// Use the dedicated query for looking up networks by Circle network type
	network, err := h.common.db.GetNetworkByCircleNetworkType(ctx, circleNetworkType)
	if err == nil {
		return network.ID, network.ChainID, nil
	}

	// If not found, fallback to looking up by chain ID for common networks
	var chainID int32
	switch blockchain {
	case "ETH":
		chainID = 1
	case "ETH-SEPOLIA":
		chainID = 11155111
	case "MATIC":
		chainID = 137
	case "ARB":
		chainID = 42161
	case "SOL":
		chainID = 999999 // Placeholder - use actual Solana chain ID
	default:
		chainID = 1 // Default to Ethereum mainnet
	}

	// Try to find by chain ID
	network, err = h.common.db.GetNetworkByChainID(ctx, chainID)
	if err == nil {
		return network.ID, network.ChainID, nil
	}

	// If we couldn't find a match, return nil UUID and the best guess chain ID
	return uuid.Nil, chainID, fmt.Errorf("no matching network found for Circle blockchain: %s", blockchain)
}

// ValidateAddress godoc
// @Summary Validate a blockchain address
// @Description Confirms that a specified address is valid for a given blockchain
// @Tags circle
// @Accept json
// @Produce json
// @Param body body ValidateAddressRequest true "Address validation request"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/transactions/validate-address [post]
func (h *CircleHandler) ValidateAddress(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	var req ValidateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Call Circle API to validate the address
	validateRequest := circle.ValidateAddressRequest{
		Blockchain: req.Blockchain,
		Address:    req.Address,
	}

	validateResponse, err := h.circleClient.ValidateAddress(c.Request.Context(), validateRequest)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to validate address", err)
		return
	}

	sendSuccess(c, http.StatusOK, validateResponse)
}

// EstimateTransferFee godoc
// @Summary Estimate fee for a transfer transaction
// @Description Calculates estimated gas fees for a transfer transaction based on network conditions
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param body body EstimateTransferFeeRequest true "Fee estimation request"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/transactions/transfer/estimate-fee [post]
func (h *CircleHandler) EstimateTransferFee(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	var req EstimateTransferFeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert request to Circle client format
	estimateRequest := circle.EstimateTransferFeeRequest{
		DestinationAddress: req.DestinationAddress,
		Amounts:            req.Amounts,
		WalletID:           req.WalletID,
		SourceAddress:      req.SourceAddress,
		Blockchain:         req.Blockchain,
		TokenID:            req.TokenID,
		TokenAddress:       req.TokenAddress,
		NftTokenIds:        req.NftTokenIds,
	}

	// Check mutual exclusivity for source parameters
	if req.WalletID != "" && (req.SourceAddress != "" || req.Blockchain != "") {
		sendError(c, http.StatusBadRequest, "Either wallet_id or both source_address and blockchain must be provided, not both", nil)
		return
	}

	// Check token identification parameters
	if req.TokenID != "" && req.TokenAddress != "" {
		sendError(c, http.StatusBadRequest, "token_id and token_address are mutually exclusive", nil)
		return
	}

	// Call Circle API to estimate transfer fee
	estimateResponse, err := h.circleClient.EstimateTransferFee(c.Request.Context(), estimateRequest, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to estimate transfer fee", err)
		return
	}

	sendSuccess(c, http.StatusOK, estimateResponse)
}

// CreateTransfer godoc
// @Summary Create a transfer transaction challenge
// @Description Generates a challenge for initiating an on-chain digital asset transfer
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param body body CreateTransferRequest true "Transfer creation request"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/transactions/transfer [post]
func (h *CircleHandler) CreateTransfer(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	var req CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert request to Circle client format
	transferRequest := circle.TransferChallengeRequest{
		IdempotencyKey:     req.IdempotencyKey,
		DestinationAddress: req.DestinationAddress,
		WalletID:           req.WalletID,
		SourceAddress:      req.SourceAddress,
		Blockchain:         req.Blockchain,
		Amounts:            req.Amounts,
		TokenID:            req.TokenID,
		TokenAddress:       req.TokenAddress,
		FeeLevel:           req.FeeLevel,
		GasLimit:           req.GasLimit,
		GasPrice:           req.GasPrice,
		MaxFee:             req.MaxFee,
		PriorityFee:        req.PriorityFee,
		NftTokenIds:        req.NftTokenIds,
		RefID:              req.RefID,
	}

	// Check mutual exclusivity for source parameters
	if req.WalletID != "" && (req.SourceAddress != "" || req.Blockchain != "") {
		sendError(c, http.StatusBadRequest, "Either wallet_id or both source_address and blockchain must be provided, not both", nil)
		return
	}

	// Check token identification parameters
	if req.TokenID != "" && req.TokenAddress != "" {
		sendError(c, http.StatusBadRequest, "token_id and token_address are mutually exclusive", nil)
		return
	}

	// Call Circle API to create transfer challenge
	transferResponse, err := h.circleClient.CreateTransferChallenge(c.Request.Context(), transferRequest, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create transfer challenge", err)
		return
	}

	sendSuccess(c, http.StatusOK, transferResponse)
}

// GetTransaction godoc
// @Summary Get transaction details
// @Description Retrieves detailed information about a specific transaction
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param transaction_id path string true "Transaction ID"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/transactions/{transaction_id} [get]
func (h *CircleHandler) GetTransaction(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	transactionID := c.Param("transaction_id")
	if transactionID == "" {
		sendError(c, http.StatusBadRequest, "Transaction ID is required", nil)
		return
	}

	// Call Circle API to get transaction details
	transactionResponse, err := h.circleClient.GetTransaction(c.Request.Context(), transactionID, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get transaction details", err)
		return
	}

	sendSuccess(c, http.StatusOK, transactionResponse)
}

// ListTransactions godoc
// @Summary List transactions
// @Description Retrieves a list of transactions with optional filtering
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param blockchain query string false "Filter by blockchain"
// @Param destination_address query string false "Filter by destination address"
// @Param include_all query boolean false "Include all tokens including zero balances"
// @Param operation query string false "Filter by operation type"
// @Param state query string false "Filter by transaction state"
// @Param tx_hash query string false "Filter by transaction hash"
// @Param tx_type query string false "Filter by transaction type"
// @Param user_id query string false "Filter by user ID"
// @Param wallet_ids query string false "Filter by wallet IDs (comma separated)"
// @Param from query string false "Filter by start date (RFC3339)"
// @Param to query string false "Filter by end date (RFC3339)"
// @Param page_size query int false "Number of results per page"
// @Param page_before query string false "Cursor for previous page"
// @Param page_after query string false "Cursor for next page"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/transactions [get]
func (h *CircleHandler) ListTransactions(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Parse query parameters
	var params ListTransactionsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid query parameters", err)
		return
	}

	// Create Circle API params
	circleParams := &circle.ListTransactionsParams{}

	// Map the parameters
	if params.Blockchain != "" {
		blockchain := params.Blockchain
		circleParams.Blockchain = &blockchain
	}
	if params.DestinationAddress != "" {
		destinationAddress := params.DestinationAddress
		circleParams.DestinationAddress = &destinationAddress
	}
	if params.IncludeAll {
		includeAll := params.IncludeAll
		circleParams.IncludeAll = &includeAll
	}
	if params.Operation != "" {
		operation := params.Operation
		circleParams.Operation = &operation
	}
	if params.State != "" {
		state := params.State
		circleParams.State = &state
	}
	if params.TxHash != "" {
		txHash := params.TxHash
		circleParams.TxHash = &txHash
	}
	if params.TxType != "" {
		txType := params.TxType
		circleParams.TxType = &txType
	}
	if params.UserID != "" {
		userID := params.UserID
		circleParams.UserID = &userID
	}
	if params.WalletIDs != "" {
		walletIDs := params.WalletIDs
		circleParams.WalletIDs = &walletIDs
	}

	// Handle time-based parameters
	if params.From != "" {
		fromTime, err := time.Parse(time.RFC3339, params.From)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid 'from' date format, use RFC3339", err)
			return
		}
		circleParams.From = &fromTime
	}

	if params.To != "" {
		toTime, err := time.Parse(time.RFC3339, params.To)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid 'to' date format, use RFC3339", err)
			return
		}
		circleParams.To = &toTime
	}

	// Handle pagination parameters
	if params.PageSize > 0 {
		pageSize := params.PageSize
		circleParams.PageSize = &pageSize
	}
	if params.PageBefore != "" {
		pageBefore := params.PageBefore
		circleParams.PageBefore = &pageBefore
	}
	if params.PageAfter != "" {
		pageAfter := params.PageAfter
		circleParams.PageAfter = &pageAfter
	}

	// Call Circle API to list transactions
	transactionsResponse, err := h.circleClient.ListTransactions(c.Request.Context(), userToken, circleParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list transactions", err)
		return
	}

	sendSuccess(c, http.StatusOK, transactionsResponse)
}
