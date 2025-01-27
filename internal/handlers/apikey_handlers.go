package handlers

import (
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

// GetAPIKeyByID retrieves a specific API key by its ID
func (h *APIKeyHandler) GetAPIKeyByID(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
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

// ListAPIKeys retrieves all API keys for the current workspace
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

// GetAllAPIKeys retrieves all API keys (admin only)
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

// CreateAPIKey creates a new API key
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

// UpdateAPIKey updates an existing API key
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
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

// DeleteAPIKey soft deletes an API key
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
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

// GetExpiredAPIKeys retrieves all expired API keys
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

// GetActiveAPIKeysCount gets the count of active API keys for a workspace
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
func generateAPIKeyHash() string {
	// TODO: Implement secure API key generation
	return "test_key_hash"
}
