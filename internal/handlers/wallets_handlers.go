package handlers

import (
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// WalletHandler handles wallet-related operations
type WalletHandler struct {
	common *CommonServices
}

// NewWalletHandler creates a new WalletHandler instance
func NewWalletHandler(common *CommonServices) *WalletHandler {
	return &WalletHandler{common: common}
}

// WalletResponse represents the standardized API response for wallet operations
type WalletResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	AccountID     string                 `json:"account_id"`
	WalletAddress string                 `json:"wallet_address"`
	NetworkType   string                 `json:"network_type"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	LastUsedAt    *int64                 `json:"last_used_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// WalletListResponse represents the paginated response for wallet list operations
type WalletListResponse struct {
	Object string           `json:"object"`
	Data   []WalletResponse `json:"data"`
}

// WalletStatsResponse represents the response for wallet statistics
type WalletStatsResponse struct {
	Object            string `json:"object"`
	TotalWallets      int64  `json:"total_wallets"`
	VerifiedWallets   int64  `json:"verified_wallets"`
	PrimaryWallets    int64  `json:"primary_wallets"`
	NetworkTypesCount int64  `json:"network_types_count"`
}

// CreateWalletRequest represents the request body for creating a wallet
type CreateWalletRequest struct {
	WalletAddress string                 `json:"wallet_address" binding:"required"`
	NetworkType   string                 `json:"network_type" binding:"required"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateWalletRequest represents the request body for updating a wallet
type UpdateWalletRequest struct {
	Nickname  string                 `json:"nickname,omitempty"`
	ENS       string                 `json:"ens,omitempty"`
	IsPrimary bool                   `json:"is_primary,omitempty"`
	Verified  bool                   `json:"verified,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
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

	return WalletResponse{
		ID:            w.ID.String(),
		Object:        "wallet",
		AccountID:     w.AccountID.String(),
		WalletAddress: w.WalletAddress,
		NetworkType:   string(w.NetworkType),
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

// CreateWallet godoc
// @Summary Create a new wallet
// @Description Creates a new wallet for the authenticated account
// @Tags wallets
// @Accept json
// @Produce json
// @Param body body CreateWalletRequest true "Wallet creation request"
// @Success 201 {object} WalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets [post]
func (h *WalletHandler) CreateWallet(c *gin.Context) {
	var req CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get account ID from context
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	// Marshal metadata
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	// Create wallet
	wallet, err := h.common.db.CreateWallet(c.Request.Context(), db.CreateWalletParams{
		AccountID:     accountID,
		WalletAddress: req.WalletAddress,
		NetworkType:   db.NetworkType(req.NetworkType),
		Nickname:      pgtype.Text{String: req.Nickname, Valid: req.Nickname != ""},
		Ens:           pgtype.Text{String: req.ENS, Valid: req.ENS != ""},
		IsPrimary:     pgtype.Bool{Bool: req.IsPrimary, Valid: true},
		Verified:      pgtype.Bool{Bool: req.Verified, Valid: true},
		Metadata:      metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create wallet", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toWalletResponse(wallet))
}

// GetWallet godoc
// @Summary Get wallet by ID
// @Description Get wallet details by wallet ID
// @Tags wallets
// @Accept json
// @Produce json
// @Param wallet_id path string true "Wallet ID"
// @Success 200 {object} WalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/{wallet_id} [get]
func (h *WalletHandler) GetWallet(c *gin.Context) {
	walletID := c.Param("wallet_id")
	parsedUUID, err := uuid.Parse(walletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid wallet ID format", err)
		return
	}

	wallet, err := h.common.db.GetWalletByID(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Wallet not found")
		return
	}

	// Verify account access
	accountID := c.GetString("account_id")
	if wallet.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	sendSuccess(c, http.StatusOK, toWalletResponse(wallet))
}

// GetWalletByAddress godoc
// @Summary Get wallet by address
// @Description Get wallet details by wallet address and network type
// @Tags wallets
// @Accept json
// @Produce json
// @Param wallet_address path string true "Wallet address"
// @Param network_type query string true "Network type"
// @Success 200 {object} WalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/address/{wallet_address} [get]
func (h *WalletHandler) GetWalletByAddress(c *gin.Context) {
	walletAddress := c.Param("wallet_address")
	networkType := c.Query("network_type")

	if walletAddress == "" || networkType == "" {
		sendError(c, http.StatusBadRequest, "Wallet address and network type are required", nil)
		return
	}

	wallet, err := h.common.db.GetWalletByAddress(c.Request.Context(), db.GetWalletByAddressParams{
		WalletAddress: walletAddress,
		NetworkType:   db.NetworkType(networkType),
	})
	if err != nil {
		handleDBError(c, err, "Wallet not found")
		return
	}

	// Verify account access
	accountID := c.GetString("account_id")
	if wallet.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	sendSuccess(c, http.StatusOK, toWalletResponse(wallet))
}

// ListWallets godoc
// @Summary List all wallets
// @Description List all wallets for the authenticated account
// @Tags wallets
// @Accept json
// @Produce json
// @Success 200 {object} WalletListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets [get]
func (h *WalletHandler) ListWallets(c *gin.Context) {
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	wallets, err := h.common.db.ListWalletsByAccountID(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list wallets", err)
		return
	}

	// Convert to response format
	response := WalletListResponse{
		Object: "list",
		Data:   make([]WalletResponse, len(wallets)),
	}
	for i, wallet := range wallets {
		response.Data[i] = toWalletResponse(wallet)
	}

	sendSuccess(c, http.StatusOK, response)
}

// ListWalletsByNetworkType godoc
// @Summary List wallets by network type
// @Description List all wallets for the authenticated account filtered by network type
// @Tags wallets
// @Accept json
// @Produce json
// @Param network_type path string true "Network type"
// @Success 200 {object} WalletListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/network/{network_type} [get]
func (h *WalletHandler) ListWalletsByNetworkType(c *gin.Context) {
	accountID, err := uuid.Parse(c.GetString("account_id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	networkType := c.Param("network_type")
	if networkType == "" {
		sendError(c, http.StatusBadRequest, "Network type is required", nil)
		return
	}

	wallets, err := h.common.db.ListWalletsByNetworkType(c.Request.Context(), db.ListWalletsByNetworkTypeParams{
		AccountID:   accountID,
		NetworkType: db.NetworkType(networkType),
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to list wallets", err)
		return
	}

	// Convert to response format
	response := WalletListResponse{
		Object: "list",
		Data:   make([]WalletResponse, len(wallets)),
	}
	for i, wallet := range wallets {
		response.Data[i] = toWalletResponse(wallet)
	}

	sendSuccess(c, http.StatusOK, response)
}

// SearchWallets godoc
// @Summary Search wallets
// @Description Search wallets by address, nickname, or ENS name
// @Tags wallets
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param limit query int false "Number of results to return (default 10)"
// @Param offset query int false "Number of results to skip (default 0)"
// @Success 200 {object} WalletListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/search [get]
func (h *WalletHandler) SearchWallets(c *gin.Context) {
	accountID, err := uuid.Parse(c.GetString("account_id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	query := c.Query("query")
	if query == "" {
		sendError(c, http.StatusBadRequest, "Search query is required", nil)
		return
	}

	// Parse pagination parameters
	limit := 10
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Add wildcards for ILIKE search
	searchPattern := "%" + query + "%"

	wallets, err := h.common.db.SearchWallets(c.Request.Context(), db.SearchWalletsParams{
		AccountID:     accountID,
		WalletAddress: searchPattern,
		Limit:         int32(limit),
		Offset:        int32(offset),
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to search wallets", err)
		return
	}

	// Convert to response format
	response := WalletListResponse{
		Object: "list",
		Data:   make([]WalletResponse, len(wallets)),
	}
	for i, wallet := range wallets {
		response.Data[i] = toWalletResponse(wallet)
	}

	sendSuccess(c, http.StatusOK, response)
}

// UpdateWallet godoc
// @Summary Update a wallet
// @Description Update wallet details
// @Tags wallets
// @Accept json
// @Produce json
// @Param wallet_id path string true "Wallet ID"
// @Param body body UpdateWalletRequest true "Wallet update request"
// @Success 200 {object} WalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/{wallet_id} [patch]
func (h *WalletHandler) UpdateWallet(c *gin.Context) {
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

	// Get current wallet to verify ownership
	currentWallet, err := h.common.db.GetWalletByID(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Wallet not found")
		return
	}

	// Verify account access
	accountID := c.GetString("account_id")
	if currentWallet.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Marshal metadata if provided
	var metadata []byte
	if req.Metadata != nil {
		metadata, err = json.Marshal(req.Metadata)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
			return
		}
	}

	// Update wallet
	wallet, err := h.common.db.UpdateWallet(c.Request.Context(), db.UpdateWalletParams{
		ID:        parsedUUID,
		Nickname:  pgtype.Text{String: req.Nickname, Valid: req.Nickname != ""},
		Ens:       pgtype.Text{String: req.ENS, Valid: req.ENS != ""},
		IsPrimary: pgtype.Bool{Bool: req.IsPrimary, Valid: true},
		Verified:  pgtype.Bool{Bool: req.Verified, Valid: true},
		Metadata:  metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update wallet", err)
		return
	}

	sendSuccess(c, http.StatusOK, toWalletResponse(wallet))
}

// SetWalletAsPrimary godoc
// @Summary Set wallet as primary
// @Description Set a wallet as the primary wallet for its network type
// @Tags wallets
// @Accept json
// @Produce json
// @Param wallet_id path string true "Wallet ID"
// @Success 200 {object} WalletResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/{wallet_id}/primary [post]
func (h *WalletHandler) SetWalletAsPrimary(c *gin.Context) {
	walletID := c.Param("wallet_id")
	parsedUUID, err := uuid.Parse(walletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid wallet ID format", err)
		return
	}

	// Get current wallet to verify ownership and get network type
	currentWallet, err := h.common.db.GetWalletByID(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Wallet not found")
		return
	}

	// Verify account access
	accountID, err := uuid.Parse(c.GetString("account_id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}
	if currentWallet.AccountID != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Set as primary
	_, err = h.common.db.SetWalletAsPrimary(c.Request.Context(), db.SetWalletAsPrimaryParams{
		AccountID:   accountID,
		NetworkType: currentWallet.NetworkType,
		ID:          parsedUUID,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to set wallet as primary", err)
		return
	}

	// Get updated wallet
	updatedWallet, err := h.common.db.GetWalletByID(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get updated wallet", err)
		return
	}

	sendSuccess(c, http.StatusOK, toWalletResponse(updatedWallet))
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
	walletID := c.Param("wallet_id")
	parsedUUID, err := uuid.Parse(walletID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid wallet ID format", err)
		return
	}

	// Get current wallet to verify ownership
	currentWallet, err := h.common.db.GetWalletByID(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Wallet not found")
		return
	}

	// Verify account access
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}
	if currentWallet.AccountID.String() != accountID.String() {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Delete wallet
	err = h.common.db.SoftDeleteWallet(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete wallet", err)
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// GetWalletStats godoc
// @Summary Get wallet statistics
// @Description Get statistics about wallets for the authenticated account
// @Tags wallets
// @Accept json
// @Produce json
// @Success 200 {object} WalletStatsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/stats [get]
func (h *WalletHandler) GetWalletStats(c *gin.Context) {
	accountID, err := uuid.Parse(c.GetString("account_id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	stats, err := h.common.db.GetWalletStats(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get wallet statistics", err)
		return
	}

	response := WalletStatsResponse{
		Object:            "wallet_stats",
		TotalWallets:      stats.TotalWallets,
		VerifiedWallets:   stats.VerifiedWallets,
		PrimaryWallets:    stats.PrimaryWallets,
		NetworkTypesCount: stats.NetworkTypesCount,
	}

	sendSuccess(c, http.StatusOK, response)
}

// GetRecentlyUsedWallets godoc
// @Summary Get recently used wallets
// @Description Get a list of recently used wallets for the authenticated account
// @Tags wallets
// @Accept json
// @Produce json
// @Param limit query int false "Number of results to return (default 5)"
// @Success 200 {object} WalletListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/recent [get]
func (h *WalletHandler) GetRecentlyUsedWallets(c *gin.Context) {
	accountID, err := uuid.Parse(c.GetString("account_id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	// Parse limit parameter
	limit := 5 // default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	wallets, err := h.common.db.GetRecentlyUsedWallets(c.Request.Context(), db.GetRecentlyUsedWalletsParams{
		AccountID: accountID,
		Limit:     int32(limit),
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get recently used wallets", err)
		return
	}

	// Convert to response format
	response := WalletListResponse{
		Object: "list",
		Data:   make([]WalletResponse, len(wallets)),
	}
	for i, wallet := range wallets {
		response.Data[i] = toWalletResponse(wallet)
	}

	sendSuccess(c, http.StatusOK, response)
}

// GetWalletsByENS godoc
// @Summary Get wallets with ENS names
// @Description Get a list of wallets that have ENS names for the authenticated account
// @Tags wallets
// @Accept json
// @Produce json
// @Success 200 {object} WalletListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /wallets/ens [get]
func (h *WalletHandler) GetWalletsByENS(c *gin.Context) {
	accountID, err := uuid.Parse(c.GetString("account_id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	wallets, err := h.common.db.GetWalletsByENS(c.Request.Context(), accountID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get wallets with ENS names", err)
		return
	}

	// Convert to response format
	response := WalletListResponse{
		Object: "list",
		Data:   make([]WalletResponse, len(wallets)),
	}
	for i, wallet := range wallets {
		response.Data[i] = toWalletResponse(wallet)
	}

	sendSuccess(c, http.StatusOK, response)
}
