package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/client/circle"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Define constants for Circle blockchain identifiers to satisfy goconst
const (
	circleEthSepolia      = "ETH-SEPOLIA"
	circleEth             = "ETH"
	circleArb             = "ARB"
	circleArbSepolia      = "ARB-SEPOLIA"
	circleMatic           = "MATIC"
	circleMaticAmoy       = "MATIC-AMOY"
	circleBase            = "BASE"
	circleBaseSepolia     = "BASE-SEPOLIA"
	circleUnichain        = "UNICHAIN"
	circleUnichainSepolia = "UNICHAIN-SEPOLIA"
	circleOp              = "OP"
	circleOPSepolia       = "OP-SEPOLIA"
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
	UserToken      string `json:"user_token" binding:"required"`
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
	Blockchains    []string `json:"blockchains"` // Optional - backend will auto-select if not provided
	AccountType    string   `json:"account_type" binding:"required"`
	UserToken      string   `json:"user_token" binding:"required"`
	Metadata       []struct {
		Name  string `json:"name"`
		RefID string `json:"ref_id"`
	} `json:"metadata,omitempty"`
}

// CreateUserPinWithWalletsRequest represents the request to create user PIN and wallets together
type CreateUserPinWithWalletsRequest struct {
	Blockchains []string `json:"blockchains"` // Optional - backend will auto-select if not provided
	AccountType string   `json:"account_type,omitempty"`
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

// CreateUserRequest represents the request to create a Circle user
type CreateUserWithPinAuthRequest struct {
	ExternalUserID string `json:"external_user_id" binding:"required"`
}

// PinDetails represents the PIN information for a Circle user
type PinDetails struct {
	FailedAttempts       int    `json:"failedAttempts"`
	LockedDate           string `json:"lockedDate"`
	LockedExpiryDate     string `json:"lockedExpiryDate"`
	LastLockOverrideDate string `json:"lastLockOverrideDate"`
}

// CircleUserData represents user data returned from the Circle API
type CircleUserData struct {
	ID                      string     `json:"id"`
	CreateDate              time.Time  `json:"createDate"`
	PinStatus               string     `json:"pinStatus"`
	Status                  string     `json:"status"`
	SecurityQuestionStatus  string     `json:"securityQuestionStatus"`
	PinDetails              PinDetails `json:"pinDetails"`
	SecurityQuestionDetails PinDetails `json:"securityQuestionDetails"`
}

// TokenData represents the token information for Circle API access
type TokenData struct {
	UserToken     string `json:"user_token"`
	EncryptionKey string `json:"encryption_key"`
}

// CreateUserResponse represents the response returned by the CreateUser endpoint
type CreateUserResponse struct {
	ID                     string `json:"id"`
	CreateDate             string `json:"createDate"`
	PinStatus              string `json:"pinStatus"`
	Status                 string `json:"status"`
	SecurityQuestionStatus string `json:"securityQuestionStatus"`
	PinDetails             struct {
		FailedAttempts       int    `json:"failedAttempts"`
		LockedDate           string `json:"lockedDate"`
		LockedExpiryDate     string `json:"lockedExpiryDate"`
		LastLockOverrideDate string `json:"lastLockOverrideDate"`
	} `json:"pinDetails"`
	SecurityQuestionDetails struct {
		FailedAttempts       int    `json:"failedAttempts"`
		LockedDate           string `json:"lockedDate"`
		LockedExpiryDate     string `json:"lockedExpiryDate"`
		LastLockOverrideDate string `json:"lastLockOverrideDate"`
	} `json:"securityQuestionDetails"`
}

// CreateUser godoc
// @Summary Create a new Circle user
// @Description Creates a new Circle user with the specified external user ID
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) CreateUser(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	// Bind request body
	var req CreateUserWithPinAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Check if user already exists in our database using workspaceID
	// Assuming GetCircleUserByWorkspaceID exists now
	existingUser, err := h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Log.Error("Failed to check for existing circle user",
			zap.Error(err),
			// Use workspaceID for logging if accountID is removed
			zap.String("workspace_id", workspaceID.String()))
		sendError(c, http.StatusInternalServerError, "Failed to check for existing circle user", err)
		return
	}

	var userID string
	var createDate time.Time
	var pinStatus string
	var status string
	var securityQuestionStatus string
	var statusCode int
	var pinDetails struct {
		FailedAttempts       int    `json:"failedAttempts"`
		LockedDate           string `json:"lockedDate"`
		LockedExpiryDate     string `json:"lockedExpiryDate"`
		LastLockOverrideDate string `json:"lastLockOverrideDate"`
	}
	var securityQuestionDetails struct {
		FailedAttempts       int    `json:"failedAttempts"`
		LockedDate           string `json:"lockedDate"`
		LockedExpiryDate     string `json:"lockedExpiryDate"`
		LastLockOverrideDate string `json:"lastLockOverrideDate"`
	}

	if err == nil {
		// User exists in our database, verify with Circle
		userByIDResponse, err := h.circleClient.GetUserByID(c.Request.Context(), existingUser.ID.String())
		if err != nil {
			logger.Log.Error("Failed to get user from Circle",
				zap.Error(err),
				zap.String("circle_user_id", existingUser.ID.String()))
			sendError(c, http.StatusInternalServerError, "Failed to get user from Circle", err)
			return
		}

		userID = userByIDResponse.Data.User.ID
		createDate = userByIDResponse.Data.User.CreateDate
		pinStatus = userByIDResponse.Data.User.PinStatus
		status = userByIDResponse.Data.User.Status
		securityQuestionStatus = userByIDResponse.Data.User.SecurityQuestionStatus
		pinDetails = userByIDResponse.Data.User.PinDetails
		securityQuestionDetails = userByIDResponse.Data.User.SecurityQuestionDetails
		statusCode = http.StatusOK // User already exists

		// Check if any status fields have changed
		if existingUser.PinStatus != pinStatus ||
			existingUser.Status != status ||
			existingUser.SecurityQuestionStatus != securityQuestionStatus {

			// Update the user in our database with new status values
			_, err = h.common.db.UpdateCircleUser(c.Request.Context(), db.UpdateCircleUserParams{
				PinStatus:              pinStatus,
				Status:                 status,
				SecurityQuestionStatus: securityQuestionStatus,
				ID:                     existingUser.ID,
			})
			if err != nil {
				logger.Log.Error("Failed to update circle user status",
					zap.Error(err),
					zap.String("circle_user_id", existingUser.ID.String()))
				sendError(c, http.StatusInternalServerError, "Failed to update circle user status", err)
				return
			}
		}
	} else {
		// Create new user in Circle
		userResponse, err := h.circleClient.CreateUserWithPinAuth(c.Request.Context(), req.ExternalUserID)
		if err != nil {
			if errors.Is(err, circle.ErrUserAlreadyExists) {
				sendError(c, http.StatusConflict, "User already exists in Circle", err)
				return
			}
			logger.Log.Error("Failed to create user in Circle",
				zap.Error(err),
				zap.String("external_user_id", req.ExternalUserID))
			sendError(c, http.StatusInternalServerError, "Failed to create user in Circle", err)
			return
		}
		userID = userResponse.Data.ID
		createDate = userResponse.Data.CreateDate
		pinStatus = userResponse.Data.PinStatus
		status = userResponse.Data.Status
		securityQuestionStatus = userResponse.Data.SecurityQuestionStatus
		pinDetails = userResponse.Data.PinDetails
		securityQuestionDetails = userResponse.Data.SecurityQuestionDetails
		statusCode = http.StatusCreated // New user created

		// Store user in our database
		// Assuming CreateCircleUserParams now takes WorkspaceID
		_, err = h.common.db.CreateCircleUser(c.Request.Context(), db.CreateCircleUserParams{
			ID:                     uuid.MustParse(userID),
			WorkspaceID:            workspaceID, // Use WorkspaceID
			CircleCreateDate:       pgtype.Timestamptz{Time: createDate, Valid: true},
			PinStatus:              pinStatus,
			Status:                 status,
			SecurityQuestionStatus: securityQuestionStatus,
		})
		if err != nil {
			logger.Log.Error("Failed to store circle user in database",
				zap.Error(err),
				zap.String("circle_user_id", userID))
			sendError(c, http.StatusInternalServerError, "Failed to store circle user in database", err)
			return
		}
	}

	sendSuccess(c, statusCode, CreateUserResponse{
		ID:                      userID,
		CreateDate:              createDate.Format(time.RFC3339),
		PinStatus:               pinStatus,
		Status:                  status,
		SecurityQuestionStatus:  securityQuestionStatus,
		PinDetails:              pinDetails,
		SecurityQuestionDetails: securityQuestionDetails,
	})
}

// CreateUserToken godoc
// @Summary Create a new Circle user token
// @Description Creates a new Circle user token with the specified external user ID
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) CreateUserToken(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	_, err := uuid.Parse(workspaceIDStr) // Validate format
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Bind request body
	var req CreateUserWithPinAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Call Circle API to create user token
	tokenResponse, err := h.circleClient.CreateUserToken(c.Request.Context(), req.ExternalUserID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create user token", err)
		return
	}

	sendSuccess(c, http.StatusOK, tokenResponse)
}

// GetUserByToken godoc
// @Summary Get Circle user details by token
// @Description Retrieves user details using Circle user token
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) GetUserByToken(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	userToken := c.GetHeader("User-Token") // Keep reading user token from header
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

	// Check if a circle user exists for this workspace
	// Assuming GetCircleUserByWorkspaceID exists now
	_, err = h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Only log and return error if it's not a "not found" error
		logger.Log.Error("Failed to check for existing circle user",
			zap.Error(err),
			// Use workspaceID for logging
			zap.String("workspace_id", workspaceID.String()))
		sendError(c, http.StatusInternalServerError, "Failed to check for existing circle user", err)
		return
	}

	sendSuccess(c, http.StatusOK, userResponse)
}

// GetUserByID godoc
// @Summary Get Circle user details by ID
// @Description Retrieves user details using Circle user ID
// @Tags circle
// @Accept json
// @Tags exclude
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
// @Summary Get Circle challenge details
// @Description Retrieves challenge details using Circle challenge ID
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) GetChallenge(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr) // Assign to workspaceID
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
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
// @Summary Initialize a new Circle user
// @Description Initializes a new Circle user with the specified idempotency key and account type
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) InitializeUser(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
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

	// Ensure the Circle user is in our database for this workspace
	// Assuming GetCircleUserByWorkspaceID exists now
	_, err = h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil {
		// Use specific error message if not found
		if errors.Is(err, pgx.ErrNoRows) {
			sendError(c, http.StatusNotFound, "Circle user not found for this workspace", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to check Circle user", err)
		}
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
// @Summary Create a new Circle PIN challenge
// @Description Creates a new Circle PIN challenge with the specified idempotency key
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) CreatePinChallenge(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr) // Validate format
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	var req RequestWithIdempotencyKey
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := req.UserToken
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the body", nil)
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

// CreateUserPinWithWallets godoc
// @Summary Create user PIN and wallets together
// @Description Creates a Circle user PIN and initial wallets in a single operation
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) CreateUserPinWithWallets(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	var req CreateUserPinWithWalletsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get the Circle user from our database using workspaceID
	circleUser, err := h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			sendError(c, http.StatusNotFound, "Circle user not found for this workspace", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to check Circle user", err)
		}
		return
	}

	// If blockchains not provided, automatically select from active networks
	blockchains := req.Blockchains
	if len(blockchains) == 0 {
		// Get all active networks with Circle support
		networks, err := h.common.db.ListActiveCircleNetworks(c.Request.Context())
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to fetch active Circle networks", err)
			return
		}

		if len(networks) == 0 {
			sendError(c, http.StatusBadRequest, "No active Circle networks configured", nil)
			return
		}

		// Extract Circle network types
		blockchains = make([]string, 0, len(networks))
		for _, network := range networks {
			if network.CircleNetworkType != "" {
				blockchains = append(blockchains, string(network.CircleNetworkType))
			}
		}

		if len(blockchains) == 0 {
			sendError(c, http.StatusBadRequest, "No valid Circle network types found in active networks", nil)
			return
		}

		logger.Log.Info("Auto-selected Circle networks for PIN with wallets",
			zap.Strings("blockchains", blockchains),
			zap.String("workspace_id", workspaceID.String()))
	}

	// Set default account type if not provided
	accountType := req.AccountType
	if accountType == "" {
		accountType = "SCA"
	}

	// Get user token for the Initialize User call
	tokenResponse, err := h.circleClient.CreateUserToken(c.Request.Context(), circleUser.ID.String())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create user token", err)
		return
	}
	userToken := tokenResponse.Data.UserToken

	// Use InitializeUser which combines PIN setup and wallet creation
	initRequest := circle.InitializeUserRequest{
		IdempotencyKey: uuid.New().String(),
		AccountType:    accountType,
		Blockchains:    blockchains,
	}

	challengeResponse, err := h.circleClient.InitializeUser(c.Request.Context(), initRequest, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create user PIN with wallets", err)
		return
	}

	// Store the challenge ID for future wallet creation tracking
	challengeID := challengeResponse.Data.ChallengeID

	// Poll every second for up to one minute to check if the challenge completes
	go func() {
		ctx := context.Background() // Create a new context to avoid cancellation issues
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		timeout := time.After(60 * time.Second)

		// Get user token for checking challenge status
		pollingTokenResponse, err := h.circleClient.CreateUserToken(ctx, circleUser.ID.String())
		if err != nil {
			logger.Error("Failed to create user token for challenge polling",
				zap.String("challenge_id", challengeID),
				zap.Error(err))
			return
		}
		pollingUserToken := pollingTokenResponse.Data.UserToken

		for {
			select {
			case <-ticker.C:
				// Check challenge status
				challengeResp, err := h.circleClient.GetChallenge(ctx, challengeID, pollingUserToken)
				if err != nil {
					logger.Warn("Failed to check challenge status during polling",
						zap.String("challenge_id", challengeID),
						zap.Error(err))
					continue
				}

				logger.Debug("Checking PIN with wallets challenge status",
					zap.String("challenge_id", challengeID),
					zap.String("status", challengeResp.Data.Challenge.Status))

				if challengeResp.Data.Challenge.Status == "COMPLETE" {
					logger.Info("PIN with wallets challenge completed successfully",
						zap.String("challenge_id", challengeID),
						zap.String("user_id", circleUser.ID.String()))

					// List wallets to get their details
					listParams := &circle.ListWalletsParams{}
					walletsListResponse, err := h.circleClient.ListWallets(ctx, userToken, listParams)
					if err != nil {
						logger.Error("Failed to list wallets after PIN with wallets challenge completion",
							zap.String("challenge_id", challengeID),
							zap.Error(err))
						return
					}

					if len(walletsListResponse.Data.Wallets) == 0 {
						logger.Warn("No wallets returned after PIN with wallets challenge completion",
							zap.String("challenge_id", challengeID))
						return
					}

					// Get database pool
					pool, poolErr := h.common.GetDBPool()
					if poolErr != nil {
						logger.Error("Failed to get database pool for wallet creation after challenge completion",
							zap.Error(poolErr))
						return
					}

					// Execute within transaction
					walletCount := 0
					txErr := helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
						qtx := h.common.WithTx(tx)

						// Process each wallet and create Cyphera wallet entries
						for _, walletData := range walletsListResponse.Data.Wallets {
							err = h.createCypheraWalletEntry(ctx, qtx, walletData, workspaceID, circleUser.ID)
							if err != nil {
								logger.Error("Failed to create Cyphera wallet entry during PIN with wallets challenge polling",
									zap.String("address", walletData.Address),
									zap.Error(err))
								continue
							}
							walletCount++
						}

						return nil
					})

					if txErr != nil {
						logger.Error("Failed to process wallet creation transaction",
							zap.Error(txErr))
						return
					}

					logger.Info("Successfully created wallets after PIN with wallets challenge completion",
						zap.String("challenge_id", challengeID),
						zap.Int("wallet_count", walletCount))
					return
				} else if challengeResp.Data.Challenge.Status == "FAILED" {
					logger.Error("PIN with wallets challenge failed",
						zap.String("challenge_id", challengeID),
						zap.Int("error_code", challengeResp.Data.Challenge.ErrorCode),
						zap.String("error_message", challengeResp.Data.Challenge.ErrorMessage))
					return
				}

			case <-timeout:
				logger.Warn("Timed out waiting for PIN with wallets challenge to complete",
					zap.String("challenge_id", challengeID),
					zap.String("user_id", circleUser.ID.String()))
				return
			}
		}
	}()

	// Return the challenge info
	sendSuccess(c, http.StatusOK, map[string]interface{}{
		"challenge_id": challengeID,
		"user_id":      circleUser.ID.String(),
		"message":      "PIN setup with wallet creation challenge initiated. Execute the challenge to set PIN and create wallets.",
	})
}

// UpdatePinChallenge godoc
// @Summary Update a Circle PIN challenge
// @Description Updates an existing Circle PIN challenge with the specified idempotency key
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) UpdatePinChallenge(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr) // Assign to workspaceID
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	var req RequestWithIdempotencyKey
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := req.UserToken
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the body", nil)
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
// @Summary Create a new Circle PIN restore challenge
// @Description Creates a new Circle PIN restore challenge with the specified idempotency key
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) CreatePinRestoreChallenge(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr) // Assign to workspaceID
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	var req RequestWithIdempotencyKey
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := req.UserToken
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the body", nil)
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
// @Summary Create a new Circle wallets
// @Description Creates a new Circle wallets with the specified idempotency key
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) CreateWallets(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	var req CreateWalletsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	userToken := req.UserToken
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the body", nil)
		return
	}

	// Get the Circle user from our database using workspaceID
	// Assuming GetCircleUserByWorkspaceID exists now
	circleUser, err := h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil {
		// Use specific error message if not found
		if errors.Is(err, pgx.ErrNoRows) {
			sendError(c, http.StatusNotFound, "Circle user not found for this workspace", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to check Circle user", err)
		}
		return
	}

	// If blockchains not provided, automatically select from active networks
	blockchains := req.Blockchains
	if len(blockchains) == 0 {
		// Get all active networks with Circle support
		networks, err := h.common.db.ListActiveCircleNetworks(c.Request.Context())
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to fetch active Circle networks", err)
			return
		}

		if len(networks) == 0 {
			sendError(c, http.StatusBadRequest, "No active Circle networks configured", nil)
			return
		}

		// Extract Circle network types
		blockchains = make([]string, 0, len(networks))
		for _, network := range networks {
			if network.CircleNetworkType != "" {
				blockchains = append(blockchains, string(network.CircleNetworkType))
			}
		}

		if len(blockchains) == 0 {
			sendError(c, http.StatusBadRequest, "No valid Circle network types found in active networks", nil)
			return
		}

		logger.Log.Info("Auto-selected Circle networks",
			zap.Strings("blockchains", blockchains),
			zap.String("workspace_id", workspaceID.String()))
	}

	// Prepare the Circle API request
	circleRequest := circle.CreateWalletsRequest{
		IdempotencyKey: req.IdempotencyKey,
		Blockchains:    blockchains,
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

	// Poll every second for up to one minute to check if the challenge completes
	go func() {
		ctx := context.Background() // Create a new context to avoid cancellation issues
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		timeout := time.After(60 * time.Second)

		for {
			select {
			case <-ticker.C:
				// Check challenge status
				challengeResp, err := h.circleClient.GetChallenge(ctx, challengeID, userToken)
				if err != nil {
					logger.Warn("Failed to check challenge status during polling",
						zap.String("challenge_id", challengeID),
						zap.Error(err))
					continue
				}

				logger.Debug("Checking challenge status",
					zap.String("challenge_id", challengeID),
					zap.String("status", challengeResp.Data.Challenge.Status))

				if challengeResp.Data.Challenge.Status == "COMPLETE" {
					logger.Info("Challenge completed successfully",
						zap.String("challenge_id", challengeID),
						zap.String("user_id", circleUser.ID.String()))

					// List wallets to get their details
					listParams := &circle.ListWalletsParams{}
					walletsListResponse, err := h.circleClient.ListWallets(ctx, userToken, listParams)
					if err != nil {
						logger.Error("Failed to list wallets after challenge completion",
							zap.String("challenge_id", challengeID),
							zap.Error(err))
						return
					}

					spew.Dump(walletsListResponse)

					if len(walletsListResponse.Data.Wallets) == 0 {
						logger.Warn("No wallets returned after challenge completion",
							zap.String("challenge_id", challengeID))
						return
					}

					// Get database pool
					pool, poolErr := h.common.GetDBPool()
					if poolErr != nil {
						logger.Error("Failed to get database pool for wallet creation after challenge completion",
							zap.Error(poolErr))
						return
					}

					// Execute within transaction
					walletCount := 0
					txErr := helpers.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
						qtx := h.common.WithTx(tx)

						spew.Dump(walletsListResponse.Data.Wallets)

						// Process each wallet and create Cyphera wallet entries
						for _, walletData := range walletsListResponse.Data.Wallets {
							err = h.createCypheraWalletEntry(ctx, qtx, walletData, workspaceID, circleUser.ID)
							if err != nil {
								logger.Error("Failed to create Cyphera wallet entry during challenge polling",
									zap.String("address", walletData.Address),
									zap.Error(err))
								continue
							}
							walletCount++
						}

						return nil
					})

					if txErr != nil {
						logger.Error("Failed to process wallet creation transaction",
							zap.Error(txErr))
						return
					}

					logger.Info("Successfully created wallets after challenge completion",
						zap.String("challenge_id", challengeID),
						zap.Int("wallet_count", walletCount))
					return
				} else if challengeResp.Data.Challenge.Status == "FAILED" {
					logger.Error("Challenge failed",
						zap.String("challenge_id", challengeID),
						zap.Int("error_code", challengeResp.Data.Challenge.ErrorCode),
						zap.String("error_message", challengeResp.Data.Challenge.ErrorMessage))
					return
				}

			case <-timeout:
				logger.Warn("Timed out waiting for challenge to complete",
					zap.String("challenge_id", challengeID),
					zap.String("user_id", circleUser.ID.String()))

				sendError(c, http.StatusInternalServerError, "Timed out waiting for challenge to complete", nil)
				return
			}
		}
	}()

	// Return the challenge info regardless of whether we've already created the wallets
	sendSuccess(c, http.StatusOK, map[string]interface{}{
		"challenge_id": challengeID,
		"user_id":      circleUser.ID.String(),
		"message":      "Wallet creation challenge initiated. Check challenge status to confirm wallet creation.",
	})
}

// createCypheraWalletEntry creates a Cyphera wallet entry for a Circle wallet in our database
// This is used when we receive wallet data from Circle and need to store it in our system
// Uses accountID because CreateWallet query requires it.
func (h *CircleHandler) createCypheraWalletEntry(ctx context.Context, qtx *db.Queries, walletData circle.Wallet, workspaceID uuid.UUID, circleUserID uuid.UUID) error {
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

	network, err := h.common.db.GetNetwork(ctx, networkID)
	if err != nil {
		return fmt.Errorf("failed to get network: %w", err)
	}

	spew.Dump(walletData)

	// Check if wallet already exists in our database
	dbWallet, err := h.common.db.GetWalletByAddressAndCircleNetworkType(ctx, db.GetWalletByAddressAndCircleNetworkTypeParams{
		WalletAddress:     walletData.Address,
		CircleNetworkType: network.CircleNetworkType,
	})

	spew.Dump(dbWallet)

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

	spew.Dump(walletExists, walletData.Blockchain)

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
			WorkspaceID:   workspaceID,
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
// @Summary Get a Circle wallet
// @Description Retrieves wallet details using Circle wallet ID
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) GetWallet(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id") // Assuming path is now /admin/circle/wallets/{workspace_id}/{wallet_id}
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
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

	// Get the Circle user from our database using workspaceID
	// Assuming GetCircleUserByWorkspaceID exists now
	circleUser, err := h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil {
		// Use specific error message if not found
		if errors.Is(err, pgx.ErrNoRows) {
			sendError(c, http.StatusNotFound, "Circle user not found for this workspace", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to check Circle user", err)
		}
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

	network, err := h.common.db.GetNetwork(c.Request.Context(), networkID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get network", err)
		return
	}

	// Get database pool
	pool, err := h.common.GetDBPool()
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get database pool", err)
		return
	}

	// Execute within transaction
	err = helpers.WithTransaction(c.Request.Context(), pool, func(tx pgx.Tx) error {
		qtx := h.common.WithTx(tx)

		// Check if wallet already exists in our database
		dbWallet, err := h.common.db.GetWalletByAddressAndCircleNetworkType(c.Request.Context(), db.GetWalletByAddressAndCircleNetworkTypeParams{
			WalletAddress:     walletData.Address,
			CircleNetworkType: network.CircleNetworkType,
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
			_, err = qtx.UpdateWallet(c.Request.Context(), db.UpdateWalletParams{
				ID:       dbWallet.ID,
				Metadata: metadataJSON,
			})
			if err != nil {
				return fmt.Errorf("failed to update wallet: %w", err)
			}

			// Update Circle wallet entry
			_, err = qtx.UpdateCircleWalletState(c.Request.Context(), db.UpdateCircleWalletStateParams{
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
			newWallet, err := qtx.CreateWallet(c.Request.Context(), db.CreateWalletParams{
				WorkspaceID:   workspaceID, // Use WorkspaceID
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
			_, err = qtx.CreateCircleWalletEntry(c.Request.Context(), db.CreateCircleWalletEntryParams{
				WalletID:       newWallet.ID,
				CircleUserID:   circleUser.ID,
				CircleWalletID: walletData.ID,
				ChainID:        chainID,
				State:          walletData.State,
			})
			if err != nil {
				return fmt.Errorf("failed to create Circle wallet entry: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		sendError(c, http.StatusInternalServerError, "Transaction failed", err)
		return
	}

	// Return the wallet data from Circle
	sendSuccess(c, http.StatusOK, walletResponse)
}

// getNetworkType determines the NetworkType enum value from a Circle blockchain identifier.
// It returns NetworkTypeEvm for Ethereum-based chains and NetworkTypeSolana for Solana chains.
func getNetworkType(blockchain string) db.NetworkType {
	switch blockchain {
	case circleEth, circleEthSepolia, circleArb, circleArbSepolia, circleMatic, circleMaticAmoy, circleBase, circleBaseSepolia, circleUnichain, circleUnichainSepolia:
		return db.NetworkTypeEvm
	default:
		return db.NetworkTypeEvm // Default to EVM
	}
}

// getCircleNetworkType converts a Circle blockchain identifier to the corresponding CircleNetworkType enum.
// It returns an error if the blockchain is not supported.
func getCircleNetworkType(blockchain string) (db.CircleNetworkType, error) {
	switch blockchain {
	case circleEth:
		return db.CircleNetworkTypeETH, nil
	case circleEthSepolia:
		return db.CircleNetworkTypeETHSEPOLIA, nil
	case circleArb:
		return db.CircleNetworkTypeARB, nil
	case circleArbSepolia:
		return db.CircleNetworkTypeARBSEPOLIA, nil
	case circleMatic:
		return db.CircleNetworkTypeMATIC, nil
	case circleMaticAmoy:
		return db.CircleNetworkTypeMATICAMOY, nil
	case circleBase:
		return db.CircleNetworkTypeBASE, nil
	case circleBaseSepolia:
		return db.CircleNetworkTypeBASESEPOLIA, nil
	case circleUnichain:
		return db.CircleNetworkTypeUNI, nil
	case circleUnichainSepolia:
		return db.CircleNetworkTypeUNI, nil
	case circleOp:
		return db.CircleNetworkTypeOP, nil
	case circleOPSepolia:
		return db.CircleNetworkTypeOPSEPOLIA, nil
	default:
		return "", fmt.Errorf("unsupported blockchain: %s", blockchain)
	}
}

// GetWalletBalance godoc
// @Summary Get a Circle wallet balance
// @Description Retrieves wallet balance details using Circle wallet ID
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) GetWalletBalance(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id") // Assuming path is now /admin/circle/wallets/{workspace_id}/{wallet_id}/balances
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
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

	// Get the Circle user from our database using workspaceID
	// Assuming GetCircleUserByWorkspaceID exists now
	circleUser, err := h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil {
		// Use specific error message if not found
		if errors.Is(err, pgx.ErrNoRows) {
			sendError(c, http.StatusNotFound, "Circle user not found for this workspace", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to check Circle user", err)
		}
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

	network, err := h.common.db.GetNetwork(c.Request.Context(), networkID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get network", err)
		return
	}

	var networkIDPgType pgtype.UUID
	if networkID != uuid.Nil {
		networkIDPgType.Bytes = networkID
		networkIDPgType.Valid = true
	}

	// Check if this wallet already exists in our database
	_, err = h.common.db.GetWalletByAddressAndCircleNetworkType(c.Request.Context(), db.GetWalletByAddressAndCircleNetworkTypeParams{
		WalletAddress:     walletData.Address,
		CircleNetworkType: network.CircleNetworkType,
	})

	// If wallet doesn't exist, create it
	if err != nil {
		// Begin transaction for creating wallet
		tx, qtx, err := h.common.BeginTx(c.Request.Context())
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to begin transaction", err)
			return
		}
		defer func() {
			if rErr := tx.Rollback(c.Request.Context()); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
				logger.Error("Failed to rollback transaction in GetWalletBalance", zap.Error(rErr))
			}
		}()

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
			WorkspaceID:   workspaceID,
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
// @Description Retrieves a list of Circle wallets for a given workspace
// @Tags circle
// @Accept json
// @Tags exclude
func (h *CircleHandler) ListWallets(c *gin.Context) {
	// Validate workspace ID from path parameter
	workspaceIDStr := c.Param("workspace_id")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format in path", err)
		return
	}

	// Validate that the workspace exists
	_, err = h.common.db.GetWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	userToken := c.GetHeader("User-Token")
	if userToken == "" {
		sendError(c, http.StatusBadRequest, "User token is required in the User-Token header", nil)
		return
	}

	// Get the Circle user from our database using workspaceID
	// Assuming GetCircleUserByWorkspaceID exists now
	circleUser, err := h.common.db.GetCircleUserByWorkspaceID(c.Request.Context(), workspaceID)
	if err != nil {
		// Use specific error message if not found
		if errors.Is(err, pgx.ErrNoRows) {
			sendError(c, http.StatusNotFound, "Circle user not found for this workspace", err)
		} else {
			sendError(c, http.StatusInternalServerError, "Failed to check Circle user", err)
		}
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
	defer func() {
		if rErr := tx.Rollback(c.Request.Context()); rErr != nil && !errors.Is(rErr, pgx.ErrTxClosed) {
			logger.Error("Failed to rollback transaction in ListWallets", zap.Error(rErr))
		}
	}()

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

		network, err := h.common.db.GetNetwork(c.Request.Context(), networkID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to get network", err)
			return
		}

		// Check if wallet already exists in our database
		dbWallet, err := h.common.db.GetWalletByAddressAndCircleNetworkType(c.Request.Context(), db.GetWalletByAddressAndCircleNetworkTypeParams{
			WalletAddress:     walletData.Address,
			CircleNetworkType: network.CircleNetworkType,
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
				WorkspaceID:   workspaceID,
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
// @Summary Validate a Circle address
// @Description Validates an address for a given blockchain
// @Tags circle
// @Accept json
// @Tags exclude
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
// @Summary Estimate a Circle transfer fee
// @Description Estimates the transfer fee for a given transfer request
// @Tags circle
// @Accept json
// @Tags exclude
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
// @Summary Create a new Circle transfer
// @Description Creates a new Circle transfer with the specified idempotency key
// @Tags circle
// @Accept json
// @Tags exclude
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
// @Summary Get a Circle transaction
// @Description Retrieves transaction details using Circle transaction ID
// @Tags circle
// @Accept json
// @Tags exclude
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
// @Summary List Circle transactions
// @Description Retrieves a list of Circle transactions for a given user
// @Tags circle
// @Accept json
// @Tags exclude
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
