package handlers

import (
	"cyphera-api/internal/client/circle"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CircleHandler struct {
	common       *CommonServices
	circleClient *circle.CircleClient
}

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
// @Description Creates a user token for Circle API operations
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

	// Call Circle API to create user token
	tokenResponse, err := h.circleClient.CreateUserToken(c.Request.Context(), userID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create user token", err)
		return
	}

	sendSuccess(c, http.StatusOK, tokenResponse)
}

// GetUserByToken godoc
// @Summary Get Circle user by token
// @Description Retrieves user details using a Circle user token
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

	// Call Circle API to get user details
	userResponse, err := h.circleClient.GetUserByToken(c.Request.Context(), userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get user details", err)
		return
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
// @Description Creates a challenge for user initialization and wallet creation
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
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
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

	// Prepare the request for Circle API
	circleRequest := circle.InitializeUserRequest{
		IdempotencyKey: req.IdempotencyKey,
		AccountType:    req.AccountType,
		Blockchains:    req.Blockchains,
	}

	if len(req.Metadata) > 0 {
		circleRequest.Metadata = make([]struct {
			Name  string `json:"name"`
			RefID string `json:"refId"`
		}, len(req.Metadata))

		for i, meta := range req.Metadata {
			circleRequest.Metadata[i].Name = meta.Name
			circleRequest.Metadata[i].RefID = meta.RefID
		}
	}

	// Call Circle API to initialize user
	initResponse, err := h.circleClient.InitializeUser(c.Request.Context(), circleRequest, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to initialize user", err)
		return
	}

	sendSuccess(c, http.StatusOK, initResponse)
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
// @Description Creates a challenge for creating new user-controlled wallets
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
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
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

	// Prepare the request for Circle API
	circleRequest := circle.CreateWalletsRequest{
		IdempotencyKey: req.IdempotencyKey,
		Blockchains:    req.Blockchains,
		AccountType:    req.AccountType,
	}

	if len(req.Metadata) > 0 {
		circleRequest.Metadata = make([]struct {
			Name  string `json:"name"`
			RefID string `json:"refId"`
		}, len(req.Metadata))

		for i, meta := range req.Metadata {
			circleRequest.Metadata[i].Name = meta.Name
			circleRequest.Metadata[i].RefID = meta.RefID
		}
	}

	// Call Circle API to create wallets
	walletsResponse, err := h.circleClient.CreateWallets(c.Request.Context(), circleRequest, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create wallets", err)
		return
	}

	sendSuccess(c, http.StatusOK, walletsResponse)
}

// GetWallet godoc
// @Summary Get Circle wallet by ID
// @Description Retrieves details about a specific wallet by its ID
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param wallet_id path string true "Wallet ID"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/wallets/{wallet_id} [get]
func (h *CircleHandler) GetWallet(c *gin.Context) {
	// Validate account ID
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
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

	// Call Circle API to get wallet
	walletResponse, err := h.circleClient.GetWallet(c.Request.Context(), walletID, userToken)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get wallet", err)
		return
	}

	sendSuccess(c, http.StatusOK, walletResponse)
}

// GetWalletBalance godoc
// @Summary Get Circle wallet balance
// @Description Retrieves token balances for a specific wallet
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param wallet_id path string true "Wallet ID"
// @Param include_all query boolean false "Include all tokens (even with zero balance)"
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
	_, err := uuid.Parse(c.GetHeader("X-Account-ID"))
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
		address := params.TokenAddress
		circleParams.TokenAddress = &address
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

	// Call Circle API to get wallet balance
	balanceResponse, err := h.circleClient.GetWalletBalance(c.Request.Context(), walletID, userToken, circleParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get wallet balance", err)
		return
	}

	sendSuccess(c, http.StatusOK, balanceResponse)
}

// ListWallets godoc
// @Summary List Circle wallets
// @Description Retrieves a list of wallets that match the specified parameters
// @Tags circle
// @Accept json
// @Produce json
// @Param user_token header string true "Circle User Token"
// @Param address query string false "Filter by wallet address"
// @Param blockchain query string false "Filter by blockchain"
// @Param sca_core query string false "Filter by SCA core"
// @Param wallet_set_id query string false "Filter by wallet set ID"
// @Param ref_id query string false "Filter by reference ID"
// @Param from query string false "Filter by creation date from (RFC3339 format)"
// @Param to query string false "Filter by creation date to (RFC3339 format)"
// @Param page_size query integer false "Page size for pagination"
// @Param page_before query string false "Page before cursor for pagination"
// @Param page_after query string false "Page after cursor for pagination"
// @Success 200 {object} interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle/wallets [get]
func (h *CircleHandler) ListWallets(c *gin.Context) {
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
	var params ListWalletsParams
	if err := c.ShouldBindQuery(&params); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid query parameters", err)
		return
	}

	// Prepare Circle API parameters
	circleParams := &circle.ListWalletsParams{}

	if params.Address != "" {
		address := params.Address
		circleParams.Address = &address
	}

	if params.Blockchain != "" {
		blockchain := params.Blockchain
		circleParams.Blockchain = &blockchain
	}

	if params.ScaCore != "" {
		scaCore := params.ScaCore
		circleParams.ScaCore = &scaCore
	}

	if params.WalletSetID != "" {
		walletSetID := params.WalletSetID
		circleParams.WalletSetID = &walletSetID
	}

	if params.RefID != "" {
		refID := params.RefID
		circleParams.RefID = &refID
	}

	if params.From != "" {
		fromTime, err := time.Parse(time.RFC3339, params.From)
		if err == nil {
			circleParams.From = &fromTime
		}
	}

	if params.To != "" {
		toTime, err := time.Parse(time.RFC3339, params.To)
		if err == nil {
			circleParams.To = &toTime
		}
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

	// Call Circle API to list wallets
	walletsResponse, err := h.circleClient.ListWallets(c.Request.Context(), userToken, circleParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list wallets", err)
		return
	}

	sendSuccess(c, http.StatusOK, walletsResponse)
}
