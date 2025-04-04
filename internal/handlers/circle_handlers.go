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
	// We will store the wallets in our database once the challenge is completed and
	// we confirm the wallets were created, typically by calling ListWallets later

	sendSuccess(c, http.StatusOK, map[string]interface{}{
		"challenge_id": challengeID,
		"user_id":      circleUser.ID.String(),
		"message":      "Wallet creation challenge initiated. Check challenge status to confirm wallet creation.",
	})
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

// lookupNetworkID finds a Network in the database by Circle network type.
// It returns the network ID and chain ID if found, or returns a best-guess chain ID
// based on common networks if not found.
func (h *CircleHandler) lookupNetworkID(ctx context.Context, blockchain string) (uuid.UUID, int32, error) {
	circleNetworkType, err := getCircleNetworkType(blockchain)
	if err != nil {
		return uuid.Nil, 0, err
	}

	// Look up network in our database
	networks, err := h.common.db.ListNetworks(ctx)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("failed to list networks: %w", err)
	}

	// Find matching network
	for _, network := range networks {
		if network.CircleNetworkType == circleNetworkType {
			return network.ID, network.ChainID, nil
		}
	}

	// For backwards compatibility, fallback to looking up by chain ID for common networks
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
	network, err := h.common.db.GetNetworkByChainID(ctx, chainID)
	if err == nil {
		return network.ID, network.ChainID, nil
	}

	// If we couldn't find a match, return nil UUID and the best guess chain ID
	return uuid.Nil, chainID, fmt.Errorf("no matching network found for Circle blockchain: %s", blockchain)
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
