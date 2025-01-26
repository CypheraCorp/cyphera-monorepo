package handlers

import (
	"cyphera-api/internal/constants"
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// APIKeyHandler handles API key related operations
type APIKeyHandler struct {
	common *CommonServices
}

// NewAPIKeyHandler creates a new instance of APIKeyHandler
// @Summary Create new API key handler
// @Description Creates a new handler for API key operations with common services
func NewAPIKeyHandler(common *CommonServices) *APIKeyHandler {
	return &APIKeyHandler{common: common}
}

// GetAPIKeyByID godoc
// @Summary Get API key by ID
// @Description Retrieves a specific API key by its unique identifier
// @Tags api_keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} APIKeyResponse
// @Failure 400 {object} ErrorResponse "Invalid UUID format or account ID"
// @Failure 403 {object} ErrorResponse "API key does not belong to your account"
// @Failure 404 {object} ErrorResponse "API key not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Security ApiKeyAuth
// @Router /api_keys/{id} [get]
func (h *APIKeyHandler) GetAPIKeyByID(c *gin.Context) {
	id := c.Param("id")

	// Validate UUID format
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	// Get account ID from context for permission checking
	accountID := c.GetString("accountID")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
		return
	}

	apiKey, err := h.common.db.GetAPIKey(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Check if the API key belongs to the account (unless admin)
	if c.GetString("userRole") != constants.RoleAdmin {
		accountUUID, err := uuid.Parse(accountID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID format"})
			return
		}

		keyAccountUUID, err := uuid.FromBytes(apiKey.AccountID.Bytes[:])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid API key account ID"})
			return
		}

		if keyAccountUUID != accountUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "API key does not belong to your account"})
			return
		}
	}

	c.JSON(http.StatusOK, apiKey)
}

// APIKeyResponse is a swagger-friendly representation of db.ApiKey
type APIKeyResponse struct {
	ID         uuid.UUID              `json:"id"`
	AccountID  uuid.UUID              `json:"account_id"`
	Name       string                 `json:"name"`
	Level      string                 `json:"level"`
	CreatedAt  time.Time              `json:"created_at"`
	DeletedAt  *time.Time             `json:"deleted_at,omitempty"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	LastUsedAt *time.Time             `json:"last_used_at,omitempty"`
	IsActive   bool                   `json:"is_active"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Livemode   bool                   `json:"livemode"`
}

type ListAPIKeyResponse struct {
	Object string           `json:"object"`
	Data   []APIKeyResponse `json:"data"`
}

// ListAPIKeys godoc
// @Summary List API keys
// @Description Returns a list of API keys for the authenticated user or account
// @Tags api_keys
// @Accept json
// @Produce json
// @Success 200 {object} ListAPIKeyResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api_keys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	// Only admin users can list API keys
	if c.GetString("userRole") != constants.RoleAdmin {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only admin users can list API keys"})
		return
	}

	accountID := c.GetString("accountID")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
		return
	}

	parsedUUID, err := uuid.Parse(accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID format"})
		return
	}

	var pgUUID pgtype.UUID
	pgUUID.Bytes = parsedUUID
	pgUUID.Valid = true

	apiKeys, err := h.common.db.ListAPIKeys(c.Request.Context(), pgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	// Convert db.ApiKey to APIKeyResponse
	apiKeyResponses := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		accountUUID, _ := uuid.FromBytes(key.AccountID.Bytes[:])
		apiKeyResponses[i] = APIKeyResponse{
			ID:        key.ID,
			AccountID: accountUUID,
			Name:      key.Name,
			Level:     string(key.Level),
			CreatedAt: key.CreatedAt.Time,
			IsActive:  key.IsActive.Bool,
			Livemode:  key.Livemode.Bool,
		}
		if key.DeletedAt.Valid {
			deletedAt := key.DeletedAt.Time
			apiKeyResponses[i].DeletedAt = &deletedAt
		}
		if key.ExpiresAt.Valid {
			expiresAt := key.ExpiresAt.Time
			apiKeyResponses[i].ExpiresAt = &expiresAt
		}
		if key.LastUsedAt.Valid {
			lastUsedAt := key.LastUsedAt.Time
			apiKeyResponses[i].LastUsedAt = &lastUsedAt
		}
		if len(key.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(key.Metadata, &metadata); err == nil {
				apiKeyResponses[i].Metadata = metadata
			}
		}
	}

	c.JSON(http.StatusOK, ListAPIKeyResponse{
		Object: "list",
		Data:   apiKeyResponses,
	})
}

// GetAllAPIKeys godoc
// @Summary Get all API keys
// @Description Retrieves all API keys (admin only)
// @Tags api_keys
// @Accept json
// @Produce json
// @Success 200 {object} ListAPIKeyResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Failed to retrieve API keys"
// @Security ApiKeyAuth
// @Router /api_keys/all [get]
func (h *APIKeyHandler) GetAllAPIKeys(c *gin.Context) {
	// This endpoint should only be accessible by admins (handled by middleware)
	apiKeys, err := h.common.db.GetAllAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	// Convert db.ApiKey to APIKeyResponse
	apiKeyResponses := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		accountUUID, _ := uuid.FromBytes(key.AccountID.Bytes[:])
		apiKeyResponses[i] = APIKeyResponse{
			ID:        key.ID,
			AccountID: accountUUID,
			Name:      key.Name,
			Level:     string(key.Level),
			CreatedAt: key.CreatedAt.Time,
			IsActive:  key.IsActive.Bool,
			Livemode:  key.Livemode.Bool,
		}
		if key.DeletedAt.Valid {
			deletedAt := key.DeletedAt.Time
			apiKeyResponses[i].DeletedAt = &deletedAt
		}
		if key.ExpiresAt.Valid {
			expiresAt := key.ExpiresAt.Time
			apiKeyResponses[i].ExpiresAt = &expiresAt
		}
		if key.LastUsedAt.Valid {
			lastUsedAt := key.LastUsedAt.Time
			apiKeyResponses[i].LastUsedAt = &lastUsedAt
		}
		if len(key.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(key.Metadata, &metadata); err == nil {
				apiKeyResponses[i].Metadata = metadata
			}
		}
	}

	c.JSON(http.StatusOK, ListAPIKeyResponse{
		Object: "list",
		Data:   apiKeyResponses,
	})
}

// CreateAPIKey godoc
// @Summary Create new API key
// @Description Creates a new API key for the authenticated account
// @Tags api_keys
// @Accept json
// @Produce json
// @Param request body object true "API Key creation request"
// @Success 200 {object} map[string]interface{} "Returns API key object and the actual key"
// @Failure 400 {object} ErrorResponse "Invalid request parameters"
// @Failure 500 {object} ErrorResponse "Failed to create API key"
// @Security ApiKeyAuth
// @Router /api_keys [post]
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req struct {
		Name      string                 `json:"name" binding:"required"`
		Level     string                 `json:"level" binding:"required,oneof=read write admin"`
		ExpiresAt *time.Time             `json:"expires_at,omitempty"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID := c.GetString("accountID")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
		return
	}

	parsedUUID, err := uuid.Parse(accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID format"})
		return
	}

	// Generate a new API key (in production, use a secure method)
	keyHash := uuid.New().String()

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
		return
	}

	var pgUUID pgtype.UUID
	pgUUID.Bytes = parsedUUID
	pgUUID.Valid = true

	var expiresAt pgtype.Timestamptz
	if req.ExpiresAt != nil {
		expiresAt.Time = *req.ExpiresAt
		expiresAt.Valid = true
	}

	apiKey, err := h.common.db.CreateAPIKey(c.Request.Context(), db.CreateAPIKeyParams{
		AccountID: pgUUID,
		Name:      req.Name,
		KeyHash:   keyHash,
		Level:     db.ApiKeyLevel(req.Level),
		ExpiresAt: expiresAt,
		Metadata:  metadata,
		Livemode:  pgtype.Bool{Bool: false, Valid: true}, // Set based on environment
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	// Return both the API key object and the actual key (which won't be retrievable again)
	c.JSON(http.StatusOK, gin.H{
		"api_key": apiKey,
		"key":     keyHash, // In production, this would be the actual API key
	})
}

// UpdateAPIKey godoc
// @Summary Update API key
// @Description Updates an existing API key's properties
// @Tags api_keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Param request body object true "API Key update request"
// @Success 200 {object} APIKeyResponse
// @Failure 400 {object} ErrorResponse "Invalid request parameters"
// @Failure 403 {object} ErrorResponse "API key does not belong to your account"
// @Failure 404 {object} ErrorResponse "API key not found"
// @Failure 500 {object} ErrorResponse "Failed to update API key"
// @Security ApiKeyAuth
// @Router /api_keys/{id} [put]
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req struct {
		Name      string                 `json:"name,omitempty"`
		Level     string                 `json:"level,omitempty" binding:"omitempty,oneof=read write admin"`
		ExpiresAt *time.Time             `json:"expires_at,omitempty"`
		IsActive  *bool                  `json:"is_active,omitempty"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check ownership unless admin
	if c.GetString("userRole") != constants.RoleAdmin {
		accountID := c.GetString("accountID")
		if accountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
			return
		}

		apiKey, err := h.common.db.GetAPIKey(c.Request.Context(), parsedUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}

		accountUUID, err := uuid.Parse(accountID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID format"})
			return
		}

		keyAccountUUID, err := uuid.FromBytes(apiKey.AccountID.Bytes[:])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid API key account ID"})
			return
		}

		if keyAccountUUID != accountUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "API key does not belong to your account"})
			return
		}
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
		return
	}

	var expiresAt pgtype.Timestamptz
	if req.ExpiresAt != nil {
		expiresAt.Time = *req.ExpiresAt
		expiresAt.Valid = true
	}

	var isActive pgtype.Bool
	if req.IsActive != nil {
		isActive.Bool = *req.IsActive
		isActive.Valid = true
	}

	apiKey, err := h.common.db.UpdateAPIKey(c.Request.Context(), db.UpdateAPIKeyParams{
		ID:        parsedUUID,
		Name:      req.Name,
		Level:     db.ApiKeyLevel(req.Level),
		ExpiresAt: expiresAt,
		IsActive:  isActive,
		Metadata:  metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key"})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// DeleteAPIKey godoc
// @Summary Delete API key
// @Description Soft deletes an API key
// @Tags api_keys
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse "Invalid UUID format"
// @Failure 403 {object} ErrorResponse "API key does not belong to your account"
// @Failure 404 {object} ErrorResponse "API key not found"
// @Failure 500 {object} ErrorResponse "Failed to delete API key"
// @Security ApiKeyAuth
// @Router /api_keys/{id} [delete]
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	// Check ownership unless admin
	if c.GetString("userRole") != constants.RoleAdmin {
		accountID := c.GetString("accountID")
		if accountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
			return
		}

		apiKey, err := h.common.db.GetAPIKey(c.Request.Context(), parsedUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}

		accountUUID, err := uuid.Parse(accountID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID format"})
			return
		}

		keyAccountUUID, err := uuid.FromBytes(apiKey.AccountID.Bytes[:])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid API key account ID"})
			return
		}

		if keyAccountUUID != accountUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "API key does not belong to your account"})
			return
		}
	}

	err = h.common.db.DeleteAPIKey(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetExpiredAPIKeys godoc
// @Summary Get expired API keys
// @Description Retrieves all expired API keys (admin only)
// @Tags api_keys
// @Accept json
// @Produce json
// @Success 200 {object} ListAPIKeyResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Failed to retrieve expired API keys"
// @Security ApiKeyAuth
// @Router /api_keys/expired [get]
func (h *APIKeyHandler) GetExpiredAPIKeys(c *gin.Context) {
	// This endpoint should only be accessible by admins (handled by middleware)
	apiKeys, err := h.common.db.GetExpiredAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve expired API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   apiKeys,
	})
}

// GetActiveAPIKeysCount godoc
// @Summary Get active API keys count
// @Description Gets the count of active API keys for an account
// @Tags api_keys
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int64 "Returns count of active API keys"
// @Failure 400 {object} ErrorResponse "Invalid account ID"
// @Failure 500 {object} ErrorResponse "Failed to get API key count"
// @Security ApiKeyAuth
// @Router /api_keys/count [get]
func (h *APIKeyHandler) GetActiveAPIKeysCount(c *gin.Context) {
	accountID := c.GetString("accountID")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
		return
	}

	parsedUUID, err := uuid.Parse(accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID format"})
		return
	}

	var pgUUID pgtype.UUID
	pgUUID.Bytes = parsedUUID
	pgUUID.Valid = true

	count, err := h.common.db.GetActiveAPIKeysCount(c.Request.Context(), pgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}
