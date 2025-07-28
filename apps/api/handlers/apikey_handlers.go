package handlers

import (
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// APIKeyHandler handles API key operations
type APIKeyHandler struct {
	common        *CommonServices
	logger        *zap.Logger
	apiKeyService interfaces.APIKeyService
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(common *CommonServices, logger *zap.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		common:        common,
		logger:        logger,
		apiKeyService: common.GetAPIKeyService(),
	}
}

// Use types from the centralized packages
type CreateAPIKeyRequest = requests.CreateAPIKeyRequest
type UpdateAPIKeyRequest = requests.UpdateAPIKeyRequest
type APIKeyResponse = responses.APIKeyResponse
type ListAPIKeysResponse = responses.ListAPIKeysResponse

// convertAPIKeyResponse converts helpers.APIKeyResponse to responses.APIKeyResponse
func convertAPIKeyResponse(helperResponse responses.APIKeyResponse) APIKeyResponse {
	return APIKeyResponse{
		ID:          helperResponse.ID,
		Object:      "api_key",
		Name:        helperResponse.Name,
		AccessLevel: helperResponse.AccessLevel,
		ExpiresAt:   helperResponse.ExpiresAt,
		LastUsedAt:  helperResponse.LastUsedAt,
		Metadata:    helperResponse.Metadata,
		CreatedAt:   helperResponse.CreatedAt,
		UpdatedAt:   helperResponse.UpdatedAt,
		KeyPrefix:   helperResponse.KeyPrefix,
	}
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

	apiKey, err := h.apiKeyService.GetAPIKey(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		handleDBError(c, err, "API key not found")
		return
	}

	sendSuccess(c, http.StatusOK, convertAPIKeyResponse(helpers.ToAPIKeyResponse(apiKey)))
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

	apiKeys, err := h.apiKeyService.ListAPIKeys(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve API keys", err)
		return
	}

	// Convert to API response format
	response := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = convertAPIKeyResponse(helpers.ToAPIKeyResponse(key))
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
	apiKeys, err := h.apiKeyService.GetAllAPIKeys(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve API keys", err)
		return
	}

	// Convert to API response format
	response := make([]APIKeyResponse, len(apiKeys))
	for i, key := range apiKeys {
		response[i] = convertAPIKeyResponse(helpers.ToAPIKeyResponse(key))
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
	apiKey, fullKey, keyPrefix, err := h.apiKeyService.CreateAPIKey(c.Request.Context(), params.CreateAPIKeyParams{
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
	response := convertAPIKeyResponse(helpers.ToAPIKeyResponse(apiKey))
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

	updatedAPIKey, err := h.apiKeyService.UpdateAPIKey(c.Request.Context(), params.UpdateAPIKeyParams{
		WorkspaceID: parsedWorkspaceID,
		ID:          parsedUUID,
		Name:        &req.Name,
		Description: &req.Description,
		ExpiresAt:   req.ExpiresAt,
		AccessLevel: req.AccessLevel,
		Metadata:    req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update API key", err)
		return
	}

	sendSuccess(c, http.StatusOK, convertAPIKeyResponse(helpers.ToAPIKeyResponse(updatedAPIKey)))
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

	err = h.apiKeyService.DeleteAPIKey(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		handleDBError(c, err, "Failed to delete API key")
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// Helper functions moved to helpers.ToAPIKeyResponse and services.APIKeyService
