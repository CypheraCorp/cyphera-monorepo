package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"encoding/base64"

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

// APIKeyResponse represents the standardized API response for API key operations
type APIKeyResponse struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	Name        string                 `json:"name"`
	AccessLevel string                 `json:"access_level"`
	ExpiresAt   *int64                 `json:"expires_at,omitempty"`
	LastUsedAt  *int64                 `json:"last_used_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Created     int64                  `json:"created"`
	Updated     int64                  `json:"updated"`
}

// CreateAPIKeyRequest represents the request body for creating an API key
type CreateAPIKeyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description" binding:"required"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	AccessLevel string                 `json:"access_level" binding:"required,oneof=read write admin"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateAPIKeyRequest represents the request body for updating an API key
type UpdateAPIKeyRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	AccessLevel string                 `json:"access_level,omitempty" binding:"omitempty,oneof=read write admin"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GetAPIKeyByID godoc
// @Summary Get an API key
// @Description Retrieves a specific API key by its ID
// @Tags api-keys
// @Accept json
// @Produce json
// @Param api_key_id path string true "API Key ID"
// @Success 200 {object} APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api-keys/{api_key_id} [get]
func (h *APIKeyHandler) GetAPIKeyByID(c *gin.Context) {
	apiKeyId := c.Param("api_key_id")
	parsedUUID, err := uuid.Parse(apiKeyId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	apiKey, err := h.common.db.GetAPIKey(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	c.JSON(http.StatusOK, toAPIKeyResponse(apiKey))
}

// ListAPIKeys godoc
// @Summary List API keys
// @Description Retrieves all API keys for the current workspace
// @Tags api-keys
// @Accept json
// @Produce json
// @Success 200 {array} APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api-keys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	workspaceID := c.GetString("workspaceID")
	parsedUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID format"})
		return
	}

	apiKeys, err := h.common.db.ListAPIKeys(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	response := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = toAPIKeyResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// GetAllAPIKeys godoc
// @Summary Get all API keys
// @Description Retrieves all API keys across all workspaces (admin only)
// @Tags api-keys
// @Accept json
// @Produce json
// @Success 200 {array} APIKeyResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/api-keys [get]
func (h *APIKeyHandler) GetAllAPIKeys(c *gin.Context) {
	apiKeys, err := h.common.db.GetAllAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	response := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = toAPIKeyResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// CreateAPIKey godoc
// @Summary Create API key
// @Description Creates a new API key for the current workspace
// @Tags api-keys
// @Accept json
// @Produce json
// @Param key body CreateAPIKeyRequest true "API key creation data"
// @Success 201 {object} APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api-keys [post]
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspaceID := c.GetString("workspaceID")
	parsedUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID format"})
		return
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

	apiKey, err := h.common.db.CreateAPIKey(c.Request.Context(), db.CreateAPIKeyParams{
		WorkspaceID: parsedUUID,
		Name:        req.Name,
		KeyHash:     generateAPIKeyHash(),
		AccessLevel: db.ApiKeyLevel(req.AccessLevel),
		ExpiresAt:   expiresAt,
		Metadata:    metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key"})
		return
	}

	c.JSON(http.StatusCreated, toAPIKeyResponse(apiKey))
}

// UpdateAPIKey godoc
// @Summary Update API key
// @Description Updates an existing API key
// @Tags api-keys
// @Accept json
// @Produce json
// @Param api_key_id path string true "API Key ID"
// @Param key body UpdateAPIKeyRequest true "API key update data"
// @Success 200 {object} APIKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api-keys/{api_key_id} [put]
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	apiKeyId := c.Param("api_key_id")
	parsedUUID, err := uuid.Parse(apiKeyId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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

	apiKey, err := h.common.db.UpdateAPIKey(c.Request.Context(), db.UpdateAPIKeyParams{
		ID:          parsedUUID,
		Name:        req.Name,
		AccessLevel: db.ApiKeyLevel(req.AccessLevel),
		ExpiresAt:   expiresAt,
		Metadata:    metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key"})
		return
	}

	c.JSON(http.StatusOK, toAPIKeyResponse(apiKey))
}

// DeleteAPIKey godoc
// @Summary Delete API key
// @Description Soft deletes an API key
// @Tags api-keys
// @Accept json
// @Produce json
// @Param api_key_id path string true "API Key ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api-keys/{api_key_id} [delete]
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	apiKeyId := c.Param("api_key_id")
	parsedUUID, err := uuid.Parse(apiKeyId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	err = h.common.db.DeleteAPIKey(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetExpiredAPIKeys godoc
// @Summary Get expired API keys
// @Description Retrieves all expired API keys
// @Tags api-keys
// @Accept json
// @Produce json
// @Success 200 {array} APIKeyResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/api-keys/expired [get]
func (h *APIKeyHandler) GetExpiredAPIKeys(c *gin.Context) {
	apiKeys, err := h.common.db.GetExpiredAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve expired API keys"})
		return
	}

	response := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = toAPIKeyResponse(key)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// GetActiveAPIKeysCount godoc
// @Summary Get active API key count
// @Description Gets the count of active API keys for a workspace
// @Tags api-keys
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int32
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api-keys/count [get]
func (h *APIKeyHandler) GetActiveAPIKeysCount(c *gin.Context) {
	workspaceID := c.GetString("workspaceID")
	parsedUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID format"})
		return
	}

	count, err := h.common.db.GetActiveAPIKeysCount(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// Helper function to convert database model to API response
func toAPIKeyResponse(k db.ApiKey) APIKeyResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(k.Metadata, &metadata); err != nil {
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	var expiresAt *int64
	if k.ExpiresAt.Valid {
		unix := k.ExpiresAt.Time.Unix()
		expiresAt = &unix
	}

	var lastUsedAt *int64
	if k.LastUsedAt.Valid {
		unix := k.LastUsedAt.Time.Unix()
		lastUsedAt = &unix
	}

	return APIKeyResponse{
		ID:          k.ID.String(),
		Object:      "api_key",
		Name:        k.Name,
		AccessLevel: string(k.AccessLevel),
		ExpiresAt:   expiresAt,
		LastUsedAt:  lastUsedAt,
		Metadata:    metadata,
		Created:     k.CreatedAt.Time.Unix(),
		Updated:     k.UpdatedAt.Time.Unix(),
	}
}

// Helper function to generate a secure API key hash
// generateAPIKeyHash creates a secure, unique API key using UUID v4 and base64 encoding
// The format is: prefix_base64(uuid)_timestamp
// Example: cyk_dj8kDjf9sKq0pLm3nO7_1234567890
func generateAPIKeyHash() string {
	// Generate a UUID v4
	keyUUID := uuid.New()

	// Get current timestamp
	timestamp := time.Now().Unix()

	// Create the key components
	prefix := "cyk" // Cyphera Key prefix
	uuidStr := base64.RawURLEncoding.EncodeToString(keyUUID[:])
	timestampStr := strconv.FormatInt(timestamp, 10)

	// Combine components with underscores
	return fmt.Sprintf("%s_%s_%s", prefix, uuidStr, timestampStr)
}
