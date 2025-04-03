package handlers

import (
	"cyphera-api/internal/db"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CircleUserHandler handles circle user-related operations
type CircleUserHandler struct {
	common *CommonServices
}

// NewCircleUserHandler creates a new CircleUserHandler instance
func NewCircleUserHandler(common *CommonServices) *CircleUserHandler {
	return &CircleUserHandler{common: common}
}

// CircleUserResponse represents the standardized API response for circle user operations
type CircleUserResponse struct {
	ID            string `json:"id"`
	Object        string `json:"object"`
	AccountID     string `json:"account_id"`
	Token         string `json:"token"`
	EncryptionKey string `json:"encryption_key"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

// CircleUserWithWalletsResponse extends CircleUserResponse with wallet count
type CircleUserWithWalletsResponse struct {
	CircleUserResponse
	WalletCount int64 `json:"wallet_count"`
}

// CircleUserListResponse represents the paginated response for circle user list operations
type CircleUserListResponse struct {
	Object string               `json:"object"`
	Data   []CircleUserResponse `json:"data"`
}

// CreateCircleUserRequest represents the request body for creating a circle user
type CreateCircleUserRequest struct {
	Token         string `json:"token" binding:"required"`
	EncryptionKey string `json:"encryption_key" binding:"required"`
}

// UpdateCircleUserRequest represents the request body for updating a circle user
type UpdateCircleUserRequest struct {
	Token         string `json:"token,omitempty"`
	EncryptionKey string `json:"encryption_key,omitempty"`
}

// Helper function to convert database model to API response
func toCircleUserResponse(user db.CircleUser) CircleUserResponse {
	return CircleUserResponse{
		ID:            user.ID.String(),
		Object:        "circle_user",
		AccountID:     user.AccountID.String(),
		Token:         user.Token,
		EncryptionKey: user.EncryptionKey,
		CreatedAt:     user.CreatedAt.Time.Unix(),
		UpdatedAt:     user.UpdatedAt.Time.Unix(),
	}
}

// Helper function to convert database model with wallet count to API response
func toCircleUserWithWalletsResponse(userWithWallets db.GetCircleUserWithWalletsRow) CircleUserWithWalletsResponse {
	return CircleUserWithWalletsResponse{
		CircleUserResponse: CircleUserResponse{
			ID:            userWithWallets.ID.String(),
			Object:        "circle_user",
			AccountID:     userWithWallets.AccountID.String(),
			Token:         userWithWallets.Token,
			EncryptionKey: userWithWallets.EncryptionKey,
			CreatedAt:     userWithWallets.CreatedAt.Time.Unix(),
			UpdatedAt:     userWithWallets.UpdatedAt.Time.Unix(),
		},
		WalletCount: userWithWallets.WalletCount,
	}
}

// Same helper for GetCircleUserWithWalletsByAccountID
func toCircleUserWithWalletsByAccountIDResponse(userWithWallets db.GetCircleUserWithWalletsByAccountIDRow) CircleUserWithWalletsResponse {
	return CircleUserWithWalletsResponse{
		CircleUserResponse: CircleUserResponse{
			ID:            userWithWallets.ID.String(),
			Object:        "circle_user",
			AccountID:     userWithWallets.AccountID.String(),
			Token:         userWithWallets.Token,
			EncryptionKey: userWithWallets.EncryptionKey,
			CreatedAt:     userWithWallets.CreatedAt.Time.Unix(),
			UpdatedAt:     userWithWallets.UpdatedAt.Time.Unix(),
		},
		WalletCount: userWithWallets.WalletCount,
	}
}

// CreateCircleUser godoc
// @Summary Create a new circle user
// @Description Creates a new circle user for the authenticated account
// @Tags circle-users
// @Accept json
// @Produce json
// @Param body body CreateCircleUserRequest true "Circle user creation request"
// @Success 201 {object} CircleUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users [post]
func (h *CircleUserHandler) CreateCircleUser(c *gin.Context) {
	var req CreateCircleUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get account ID from header
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	// Generate a new ID for the circle user
	circleUserID := uuid.New()

	// Create circle user
	circleUser, err := h.common.db.CreateCircleUser(c.Request.Context(), db.CreateCircleUserParams{
		ID:            circleUserID,
		AccountID:     accountID,
		Token:         req.Token,
		EncryptionKey: req.EncryptionKey,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create circle user", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toCircleUserResponse(circleUser))
}

// GetCircleUserByID godoc
// @Summary Get circle user by ID
// @Description Get circle user details by ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Param id path string true "Circle User ID"
// @Success 200 {object} CircleUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/{id} [get]
func (h *CircleUserHandler) GetCircleUserByID(c *gin.Context) {
	userID := c.Param("id")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid circle user ID format", err)
		return
	}

	// Get account ID from header
	accountID := c.GetHeader("X-Account-ID")
	if accountID == "" {
		sendError(c, http.StatusBadRequest, "Account ID is required", nil)
		return
	}

	circleUser, err := h.common.db.GetCircleUserByID(c.Request.Context(), parsedUserID)
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	// Verify account access
	if circleUser.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	sendSuccess(c, http.StatusOK, toCircleUserResponse(circleUser))
}

// GetCircleUserByAccountID godoc
// @Summary Get circle user by account ID
// @Description Get circle user details by account ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Success 200 {object} CircleUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/account [get]
func (h *CircleUserHandler) GetCircleUserByAccountID(c *gin.Context) {
	// Get account ID from header
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	circleUser, err := h.common.db.GetCircleUserByAccountID(c.Request.Context(), accountID)
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	sendSuccess(c, http.StatusOK, toCircleUserResponse(circleUser))
}

// ListCircleUsers godoc
// @Summary List all circle users
// @Description List all circle users (admin only)
// @Tags circle-users
// @Accept json
// @Produce json
// @Success 200 {object} CircleUserListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users [get]
func (h *CircleUserHandler) ListCircleUsers(c *gin.Context) {
	// This endpoint should be restricted to admin users
	// TODO: Add admin check here

	circleUsers, err := h.common.db.ListCircleUsers(c.Request.Context())
	if err != nil {
		handleDBError(c, err, "Failed to list circle users")
		return
	}

	// Convert to response format
	response := make([]CircleUserResponse, len(circleUsers))
	for i, user := range circleUsers {
		response[i] = toCircleUserResponse(user)
	}

	listResponse := CircleUserListResponse{
		Object: "list",
		Data:   response,
	}

	sendSuccess(c, http.StatusOK, listResponse)
}

// UpdateCircleUser godoc
// @Summary Update circle user by ID
// @Description Update circle user details by ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Param id path string true "Circle User ID"
// @Param body body UpdateCircleUserRequest true "Circle user update request"
// @Success 200 {object} CircleUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/{id} [patch]
func (h *CircleUserHandler) UpdateCircleUser(c *gin.Context) {
	userID := c.Param("id")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid circle user ID format", err)
		return
	}

	var req UpdateCircleUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get account ID from header
	accountID := c.GetHeader("X-Account-ID")
	if accountID == "" {
		sendError(c, http.StatusBadRequest, "Account ID is required", nil)
		return
	}

	// Get current circle user to verify ownership
	currentUser, err := h.common.db.GetCircleUserByID(c.Request.Context(), parsedUserID)
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	// Verify account access
	if currentUser.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Update circle user
	updatedUser, err := h.common.db.UpdateCircleUser(c.Request.Context(), db.UpdateCircleUserParams{
		Token:         req.Token,
		EncryptionKey: req.EncryptionKey,
		ID:            parsedUserID,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update circle user", err)
		return
	}

	sendSuccess(c, http.StatusOK, toCircleUserResponse(updatedUser))
}

// UpdateCircleUserByAccountID godoc
// @Summary Update circle user by account ID
// @Description Update circle user details by account ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Param body body UpdateCircleUserRequest true "Circle user update request"
// @Success 200 {object} CircleUserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/account [patch]
func (h *CircleUserHandler) UpdateCircleUserByAccountID(c *gin.Context) {
	var req UpdateCircleUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get account ID from header
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	// Update circle user by account ID
	updatedUser, err := h.common.db.UpdateCircleUserByAccountID(c.Request.Context(), db.UpdateCircleUserByAccountIDParams{
		Token:         req.Token,
		EncryptionKey: req.EncryptionKey,
		AccountID:     accountID,
	})
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	sendSuccess(c, http.StatusOK, toCircleUserResponse(updatedUser))
}

// DeleteCircleUser godoc
// @Summary Delete circle user by ID
// @Description Delete circle user by ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Param id path string true "Circle User ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/{id} [delete]
func (h *CircleUserHandler) DeleteCircleUser(c *gin.Context) {
	userID := c.Param("id")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid circle user ID format", err)
		return
	}

	// Get account ID from header
	accountID := c.GetHeader("X-Account-ID")
	if accountID == "" {
		sendError(c, http.StatusBadRequest, "Account ID is required", nil)
		return
	}

	// Get current circle user to verify ownership
	currentUser, err := h.common.db.GetCircleUserByID(c.Request.Context(), parsedUserID)
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	// Verify account access
	if currentUser.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Delete circle user
	err = h.common.db.DeleteCircleUser(c.Request.Context(), parsedUserID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to delete circle user", err)
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// DeleteCircleUserByAccountID godoc
// @Summary Delete circle user by account ID
// @Description Delete circle user by account ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/account [delete]
func (h *CircleUserHandler) DeleteCircleUserByAccountID(c *gin.Context) {
	// Get account ID from header
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	// Delete circle user by account ID
	err = h.common.db.DeleteCircleUserByAccountID(c.Request.Context(), accountID)
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// GetCircleUserWithWallets godoc
// @Summary Get circle user with wallet count by ID
// @Description Get circle user details with wallet count by ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Param id path string true "Circle User ID"
// @Success 200 {object} CircleUserWithWalletsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/{id}/wallets [get]
func (h *CircleUserHandler) GetCircleUserWithWallets(c *gin.Context) {
	userID := c.Param("id")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid circle user ID format", err)
		return
	}

	// Get account ID from header
	accountID := c.GetHeader("X-Account-ID")
	if accountID == "" {
		sendError(c, http.StatusBadRequest, "Account ID is required", nil)
		return
	}

	userWithWallets, err := h.common.db.GetCircleUserWithWallets(c.Request.Context(), parsedUserID)
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	// Verify account access
	if userWithWallets.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	sendSuccess(c, http.StatusOK, toCircleUserWithWalletsResponse(userWithWallets))
}

// GetCircleUserWithWalletsByAccountID godoc
// @Summary Get circle user with wallet count by account ID
// @Description Get circle user details with wallet count by account ID
// @Tags circle-users
// @Accept json
// @Produce json
// @Success 200 {object} CircleUserWithWalletsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /circle-users/account/wallets [get]
func (h *CircleUserHandler) GetCircleUserWithWalletsByAccountID(c *gin.Context) {
	// Get account ID from header
	accountID, err := uuid.Parse(c.GetHeader("X-Account-ID"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID", err)
		return
	}

	userWithWallets, err := h.common.db.GetCircleUserWithWalletsByAccountID(c.Request.Context(), accountID)
	if err != nil {
		handleDBError(c, err, "Circle user not found")
		return
	}

	sendSuccess(c, http.StatusOK, toCircleUserWithWalletsByAccountIDResponse(userWithWallets))
}
