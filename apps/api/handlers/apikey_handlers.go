package handlers

import (
	"net/http"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIKeyHandler handles API key related operations
type APIKeyHandler struct {
	common        *CommonServices
	apiKeyService *services.APIKeyService
}

// NewAPIKeyHandler creates a new instance of APIKeyHandler
// @Summary Create new API key handler
// @Description Creates a new handler for API key operations with common services
func NewAPIKeyHandler(common *CommonServices) *APIKeyHandler {
	return &APIKeyHandler{
		common:        common,
		apiKeyService: services.NewAPIKeyService(common.db),
	}
}

// Use types from helpers
type APIKeyResponse = helpers.APIKeyResponse

// CreateAPIKeyRequest represents the request body for creating an API key
type CreateAPIKeyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description,omitempty"`
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

// Use types from helpers
type ListAPIKeysResponse = helpers.ListAPIKeysResponse

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
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	apiKeyId := c.Param("api_key_id")
	parsedUUID, err := uuid.Parse(apiKeyId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid UUID format", err)
		return
	}

	apiKey, err := h.common.APIKeyService.GetAPIKey(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		handleDBError(c, err, "API key not found")
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToAPIKeyResponse(apiKey))
}

// ListAPIKeys godoc
// @Summary List API keys
// @Description Retrieves all API keys for the current workspace
// @Tags api-keys
// @Accept json
// @Produce json
// @Success 200 {object} ListAPIKeysResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /api-keys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	apiKeys, err := h.common.APIKeyService.ListAPIKeys(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve API keys", err)
		return
	}

	response := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = helpers.ToAPIKeyResponse(key)
	}

	listAPIKeysResponse := ListAPIKeysResponse{
		Object:  "list",
		Data:    response,
		HasMore: false,
		Total:   int64(len(apiKeys)),
	}

	sendSuccess(c, http.StatusOK, listAPIKeysResponse)
}

// CreateAPIKey godoc
// @Summary Create a new API key
// @Description Creates a new API key with the specified name and access level
// @Tags api-keys
// @Accept json
// @Tags exclude
func (h *APIKeyHandler) GetAllAPIKeys(c *gin.Context) {
	apiKeys, err := h.common.APIKeyService.GetAllAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	response := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = helpers.ToAPIKeyResponse(key)
	}

	listAPIKeysResponse := ListAPIKeysResponse{
		Object:  "list",
		Data:    response,
		HasMore: false,
		Total:   int64(len(apiKeys)),
	}

	sendList(c, listAPIKeysResponse)
}

// CreateAPIKey godoc
// @Summary Create a new API key
// @Description Creates a new API key with the specified name and access level
// @Tags api-keys
// @Accept json
// @Tags exclude
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Create API key using service
	apiKey, fullKey, keyPrefix, err := h.common.APIKeyService.CreateAPIKey(c.Request.Context(), services.CreateAPIKeyParams{
		WorkspaceID: parsedWorkspaceID,
		Name:        req.Name,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
		AccessLevel: req.AccessLevel,
		Metadata:    req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create API key", err)
		return
	}

	// Include the full key in the response (only time it's shown)
	response := helpers.ToAPIKeyResponse(apiKey)
	response.Key = fullKey
	response.KeyPrefix = keyPrefix

	sendSuccess(c, http.StatusCreated, response)
}

// UpdateAPIKey godoc
// @Summary Update an API key
// @Description Updates an existing API key with the specified name and access level
// @Tags api-keys
// @Accept json
// @Tags exclude
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	apiKeyId := c.Param("api_key_id")
	parsedUUID, err := uuid.Parse(apiKeyId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid UUID format", err)
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	apiKey, err := h.common.APIKeyService.UpdateAPIKey(c.Request.Context(), services.UpdateAPIKeyParams{
		WorkspaceID: parsedWorkspaceID,
		ID:          parsedUUID,
		Name:        req.Name,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
		AccessLevel: req.AccessLevel,
		Metadata:    req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update API key", err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToAPIKeyResponse(apiKey))
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
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	apiKeyId := c.Param("api_key_id")
	parsedUUID, err := uuid.Parse(apiKeyId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid UUID format", err)
		return
	}

	err = h.common.APIKeyService.DeleteAPIKey(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		handleDBError(c, err, "Failed to delete API key")
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// Helper functions moved to helpers.ToAPIKeyResponse and services.APIKeyService
