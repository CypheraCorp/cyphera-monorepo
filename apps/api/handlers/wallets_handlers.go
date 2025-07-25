package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WalletHandler handles wallet-related operations
type WalletHandler struct {
	common        *CommonServices
	walletService *services.WalletService
}

// NewWalletHandler creates a new WalletHandler instance
func NewWalletHandler(common *CommonServices) *WalletHandler {
	return &WalletHandler{
		common:        common,
		walletService: services.NewWalletService(common.db),
	}
}

// WalletResponse represents the standardized API response for wallet operations
type WalletResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	WorkspaceID   string                 `json:"workspace_id"`
	WalletType    string                 `json:"wallet_type"` // 'wallet' or 'circle_wallet'
	WalletAddress string                 `json:"wallet_address"`
	NetworkType   string                 `json:"network_type"`
	NetworkID     string                 `json:"network_id,omitempty"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	LastUsedAt    *int64                 `json:"last_used_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CircleData    *CircleWalletData      `json:"circle_data,omitempty"` // Only present for circle wallets
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// CircleWalletData represents Circle-specific wallet data
type CircleWalletData struct {
	CircleWalletID string `json:"circle_wallet_id"`
	CircleUserID   string `json:"circle_user_id"`
	ChainID        int32  `json:"chain_id"`
	State          string `json:"state"`
}

// WalletListResponse represents the paginated response for wallet list operations
type WalletListResponse struct {
	Object string           `json:"object"`
	Data   []WalletResponse `json:"data"`
}

// CreateWalletRequest represents the request body for creating a wallet
type CreateWalletRequest struct {
	WalletType    string                 `json:"wallet_type" binding:"required"` // 'wallet' or 'circle_wallet'
	WalletAddress string                 `json:"wallet_address" binding:"required"`
	NetworkType   string                 `json:"network_type" binding:"required"` // 'evm' or 'solana'
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	// Circle wallet specific fields
	CircleUserID   string `json:"circle_user_id,omitempty"`   // Only for circle wallets
	CircleWalletID string `json:"circle_wallet_id,omitempty"` // Only for circle wallets
	ChainID        int32  `json:"chain_id,omitempty"`         // Only for circle wallets
	State          string `json:"state,omitempty"`            // Only for circle wallets
}

// UpdateWalletRequest represents the request body for updating a wallet
type UpdateWalletRequest struct {
	Nickname  string                 `json:"nickname,omitempty"`
	ENS       string                 `json:"ens,omitempty"`
	IsPrimary bool                   `json:"is_primary,omitempty"`
	Verified  bool                   `json:"verified,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	// Circle wallet specific fields
	State string `json:"state,omitempty"` // Only for circle wallets
}

// Helper function to convert database model to API response
func toWalletResponse(w db.Wallet) WalletResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Metadata, &metadata); err != nil {
		logger.Error("Error unmarshaling wallet metadata", zap.Error(err))
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	var lastUsedAt *int64
	if w.LastUsedAt.Valid {
		unix := w.LastUsedAt.Time.Unix()
		lastUsedAt = &unix
	}

	networkID := ""
	if w.NetworkID.Valid {
		networkID = w.NetworkID.String()
	}

	return WalletResponse{
		ID:            w.ID.String(),
		Object:        "wallet",
		WorkspaceID:   w.WorkspaceID.String(),
		WalletType:    w.WalletType,
		WalletAddress: w.WalletAddress,
		NetworkType:   string(w.NetworkType),
		NetworkID:     networkID,
		Nickname:      w.Nickname.String,
		ENS:           w.Ens.String,
		IsPrimary:     w.IsPrimary.Bool,
		Verified:      w.Verified.Bool,
		LastUsedAt:    lastUsedAt,
		Metadata:      metadata,
		CreatedAt:     w.CreatedAt.Time.Unix(),
		UpdatedAt:     w.UpdatedAt.Time.Unix(),
	}
}

func toWalletResponseFromWalletRow(w db.GetWalletByAddressAndCircleNetworkTypeRow) WalletResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Metadata, &metadata); err != nil {
		logger.Error("Error unmarshaling wallet metadata", zap.Error(err))
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	var lastUsedAt *int64
	if w.LastUsedAt.Valid {
		unix := w.LastUsedAt.Time.Unix()
		lastUsedAt = &unix
	}

	networkID := ""
	if w.NetworkID.Valid {
		networkID = w.NetworkID.String()
	}

	return WalletResponse{
		ID:            w.ID.String(),
		Object:        "wallet",
		WorkspaceID:   w.WorkspaceID.String(),
		WalletType:    w.WalletType,
		WalletAddress: w.WalletAddress,
		NetworkType:   string(w.NetworkType),
		NetworkID:     networkID,
		Nickname:      w.Nickname.String,
		ENS:           w.Ens.String,
		IsPrimary:     w.IsPrimary.Bool,
		Verified:      w.Verified.Bool,
		LastUsedAt:    lastUsedAt,
		Metadata:      metadata,
		CreatedAt:     w.CreatedAt.Time.Unix(),
		UpdatedAt:     w.UpdatedAt.Time.Unix(),
	}
}

// Helper function for GetWalletWithCircleDataByID results
func toWalletWithCircleDataByIDResponse(w db.GetWalletWithCircleDataByIDRow) WalletResponse {
	response := toWalletResponse(db.Wallet{
		ID:            w.ID,
		WorkspaceID:   w.WorkspaceID,
		WalletType:    w.WalletType,
		WalletAddress: w.WalletAddress,
		NetworkType:   w.NetworkType,
		NetworkID:     w.NetworkID,
		Nickname:      w.Nickname,
		Ens:           w.Ens,
		IsPrimary:     w.IsPrimary,
		Verified:      w.Verified,
		LastUsedAt:    w.LastUsedAt,
		Metadata:      w.Metadata,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	})

	// Add Circle data if it exists
	if w.CircleWalletID.Valid && w.CircleUserID.Valid {
		response.CircleData = &CircleWalletData{
			CircleWalletID: w.CircleWalletID.String(),
			CircleUserID:   w.CircleUserID.String(),
			ChainID:        w.ChainID.Int32,
			State:          w.CircleState.String,
		}
	}

	return response
}

// Helper function for ListWalletsWithCircleDataByWorkspaceID results
func toListWalletsWithCircleDataResponse(w db.ListWalletsWithCircleDataByWorkspaceIDRow) WalletResponse {
	response := toWalletResponse(db.Wallet{
		ID:            w.ID,
		WorkspaceID:   w.WorkspaceID,
		WalletType:    w.WalletType,
		WalletAddress: w.WalletAddress,
		NetworkType:   w.NetworkType,
		NetworkID:     w.NetworkID,
		Nickname:      w.Nickname,
		Ens:           w.Ens,
		IsPrimary:     w.IsPrimary,
		Verified:      w.Verified,
		LastUsedAt:    w.LastUsedAt,
		Metadata:      w.Metadata,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	})

	// Add Circle data if it exists
	if w.CircleWalletID.Valid && w.CircleUserID.Valid {
		response.CircleData = &CircleWalletData{
			CircleWalletID: w.CircleWalletID.String(),
			CircleUserID:   w.CircleUserID.String(),
			ChainID:        w.ChainID.Int32,
			State:          w.CircleState.String,
		}
	}

	return response
}

// Helper function for ListCircleWalletsByWorkspaceID results
func toListCircleWalletsResponse(w db.ListCircleWalletsByWorkspaceIDRow) WalletResponse {
	response := toWalletResponse(db.Wallet{
		ID:            w.ID,
		WorkspaceID:   w.WorkspaceID,
		WalletType:    w.WalletType,
		WalletAddress: w.WalletAddress,
		NetworkType:   w.NetworkType,
		NetworkID:     w.NetworkID,
		Nickname:      w.Nickname,
		Ens:           w.Ens,
		IsPrimary:     w.IsPrimary,
		Verified:      w.Verified,
		LastUsedAt:    w.LastUsedAt,
		Metadata:      w.Metadata,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	})

	// Add Circle data
	response.CircleData = &CircleWalletData{
		CircleWalletID: w.CircleID,
		CircleUserID:   w.CircleUserID.String(),
		ChainID:        w.ChainID,
		State:          w.CircleState,
	}

	return response
}

// CreateWallet godoc
// @Summary Create a new wallet
// @Description Creates a new wallet for each active network in the authenticated workspace
// @Tags wallets
// @Accept json
// @Produce json
// @Param body body CreateWalletRequest true "Wallet creation request"
// @Success 201 {object} WalletListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets [post]
func (h *WalletHandler) CreateWallet(c *gin.Context) {
	// Get workspace ID from header (Assuming X-Workspace-ID is used now)
	workspaceIDStr := c.GetHeader("X-Workspace-ID")
	workspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid or missing X-Workspace-ID header", err)
		return
	}

	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate wallet type - only allow non-Circle wallets
	if req.WalletType != "wallet" {
		sendError(c, http.StatusBadRequest, "Invalid wallet type. Circle wallets should be created using the Circle API endpoints.", nil)
		return
	}

	// Create wallets for all active networks using the service
	wallets, err := h.walletService.CreateWalletsForAllNetworks(c.Request.Context(), services.CreateWalletParams{
		WorkspaceID:   workspaceID,
		WalletType:    req.WalletType,
		WalletAddress: req.WalletAddress,
		NetworkType:   req.NetworkType,
		Nickname:      req.Nickname,
		ENS:           req.ENS,
		IsPrimary:     req.IsPrimary,
		Verified:      req.Verified,
		Metadata:      req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Convert to response format
	var createdWallets []WalletResponse
	for _, wallet := range wallets {
		createdWallets = append(createdWallets, toWalletResponse(wallet))
	}

	// Return list of created wallets
	listResponse := WalletListResponse{
		Object: "list",
		Data:   createdWallets,
	}

	sendSuccess(c, http.StatusCreated, listResponse)
}

// GetWallet godoc
// @Summary Get wallet by ID
// @Description Get wallet details by wallet ID
// @Tags wallets
// @Accept json
// @Produce json
// @Param wallet_id path string true "Wallet ID"
// @Param include_circle_data query boolean false "Include Circle wallet data"
// @Success 200 {object} WalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/{wallet_id} [get]
func (h *WalletHandler) GetWallet(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	walletID := c.Param("wallet_id")
	parsedUUID, err := uuid.Parse(walletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid wallet ID format", err)
		return
	}

	// Check if Circle data should be included
	includeCircleData := c.Query("include_circle_data") == "true"

	if includeCircleData {
		walletData, err := h.walletService.GetWalletWithCircleData(c.Request.Context(), parsedUUID, parsedWorkspaceID)
		if err != nil {
			if err.Error() == "wallet not found" {
				sendError(c, http.StatusNotFound, "Wallet not found", nil)
				return
			}
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		// Convert to response format
		response := toWalletResponse(walletData.Wallet)
		if walletData.CircleData != nil {
			response.CircleData = &CircleWalletData{
				CircleWalletID: walletData.CircleData.CircleWalletID,
				CircleUserID:   walletData.CircleData.CircleUserID,
				ChainID:        walletData.CircleData.ChainID,
				State:          walletData.CircleData.State,
			}
		}

		sendSuccess(c, http.StatusOK, response)
		return
	}

	wallet, err := h.walletService.GetWallet(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if err.Error() == "wallet not found" {
			sendError(c, http.StatusNotFound, "Wallet not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, toWalletResponse(*wallet))
}

// ListWallets godoc
// @Summary List all wallets
// @Description List all wallets for the authenticated workspace
// @Tags wallets
// @Accept json
// @Produce json
// @Param include_circle_data query boolean false "Include Circle wallet data"
// @Param wallet_type query string false "Filter by wallet type (wallet, circle_wallet)"
// @Success 200 {object} WalletListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets [get]
func (h *WalletHandler) ListWallets(c *gin.Context) {
	// FIRST: Log that we entered the handler
	logger.Log.Info("=== ENTERED ListWallets handler ===",
		zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		zap.String("path", c.Request.URL.Path),
	)

	// Get workspace ID from header - try both cases
	workspaceIDStr := c.GetHeader("X-Workspace-ID")
	if workspaceIDStr == "" {
		workspaceIDStr = c.GetHeader("X-Workspace-Id")
	}

	// Add debug logging with all header variations
	logger.Log.Info("ListWallets called",
		zap.String("workspace_id_header", workspaceIDStr),
		zap.String("X-Workspace-ID", c.GetHeader("X-Workspace-ID")),
		zap.String("X-Workspace-Id", c.GetHeader("X-Workspace-Id")),
		zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		zap.String("auth_header_preview", func() string {
			auth := c.GetHeader("Authorization")
			if len(auth) > 20 {
				return auth[:20] + "..."
			}
			return auth
		}()),
	)

	parsedWorkspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		logger.Log.Error("Failed to parse workspace ID",
			zap.String("workspace_id_str", workspaceIDStr),
			zap.Error(err),
		)
		sendError(c, http.StatusBadRequest, "Invalid or missing X-Workspace-ID header", err)
		return
	}

	// Check if Circle data should be included
	includeCircleData := c.Query("include_circle_data") == "true"
	walletType := c.Query("wallet_type")

	// Build response based on params
	var response []WalletResponse

	if walletType == "circle_wallet" {
		// Get only Circle wallets
		circleWallets, err := h.walletService.ListCircleWallets(c.Request.Context(), parsedWorkspaceID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		response = make([]WalletResponse, len(circleWallets))
		for i, wallet := range circleWallets {
			response[i] = toListCircleWalletsResponse(wallet)
		}
	} else if walletType == "wallet" {
		// Get only standard wallets
		wallets, err := h.walletService.ListWalletsByType(c.Request.Context(), parsedWorkspaceID, "wallet")
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		response = make([]WalletResponse, len(wallets))
		for i, wallet := range wallets {
			response[i] = toWalletResponse(wallet)
		}
	} else if includeCircleData {
		// Get all wallets with Circle data
		walletsWithCircleData, err := h.walletService.ListWalletsWithCircleData(c.Request.Context(), parsedWorkspaceID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		response = make([]WalletResponse, len(walletsWithCircleData))
		for i, wallet := range walletsWithCircleData {
			response[i] = toListWalletsWithCircleDataResponse(wallet)
		}
	} else {
		// Get all wallets without Circle data
		wallets, err := h.walletService.ListWalletsByWorkspace(c.Request.Context(), parsedWorkspaceID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		response = make([]WalletResponse, len(wallets))
		for i, wallet := range wallets {
			response[i] = toWalletResponse(wallet)
		}
	}

	listResponse := WalletListResponse{
		Object: "list",
		Data:   response,
	}

	sendSuccess(c, http.StatusOK, listResponse)
}

// UpdateWallet godoc
// @Summary Update a wallet
// @Description Updates an existing wallet with the specified details
// @Tags wallets
// @Accept json
// @Produce json
// @Tags exclude
func (h *WalletHandler) UpdateWallet(c *gin.Context) {
	workspaceIDStr := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid or missing X-Workspace-ID header", err)
		return
	}

	walletID := c.Param("wallet_id")
	parsedUUID, err := uuid.Parse(walletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid wallet ID format", err)
		return
	}

	var req UpdateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Update wallet using the service
	updateParams := services.UpdateWalletParams{
		ID:       parsedUUID,
		Metadata: req.Metadata,
	}

	// Set optional fields only if provided
	if req.Nickname != "" {
		updateParams.Nickname = &req.Nickname
	}
	if req.ENS != "" {
		updateParams.ENS = &req.ENS
	}
	updateParams.IsPrimary = &req.IsPrimary
	updateParams.Verified = &req.Verified

	wallet, err := h.walletService.UpdateWallet(c.Request.Context(), parsedWorkspaceID, updateParams)
	if err != nil {
		if err.Error() == "wallet not found" {
			sendError(c, http.StatusNotFound, "Wallet not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, toWalletResponse(*wallet))
}

// DeleteWallet godoc
// @Summary Delete a wallet
// @Description Soft delete a wallet
// @Tags wallets
// @Accept json
// @Produce json
// @Param wallet_id path string true "Wallet ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/{wallet_id} [delete]
func (h *WalletHandler) DeleteWallet(c *gin.Context) {
	// Get workspace ID from header (Assuming X-Workspace-ID is used now)
	workspaceIDStr := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid or missing X-Workspace-ID header", err)
		return
	}

	walletID := c.Param("wallet_id")
	parsedUUID, err := uuid.Parse(walletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid wallet ID format", err)
		return
	}

	// Delete wallet using the service
	err = h.walletService.DeleteWallet(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if err.Error() == "wallet not found" {
			sendError(c, http.StatusNotFound, "Wallet not found", nil)
			return
		}
		// Check if it's a product usage error
		if err.Error() != "" && err.Error()[:6] == "cannot" {
			sendError(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}
