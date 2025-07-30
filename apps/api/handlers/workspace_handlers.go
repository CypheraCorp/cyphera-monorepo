package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/cyphera/cyphera-api/apps/api/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WorkspaceHandler handles workspace-related operations
type WorkspaceHandler struct {
	common           *CommonServices
	workspaceService interfaces.WorkspaceService
}

// Use types from the centralized packages
type CreateWorkspaceRequest = requests.CreateWorkspaceRequest
type UpdateWorkspaceRequest = requests.UpdateWorkspaceRequest
type WorkspaceResponse = responses.WorkspaceResponse
type ListWorkspacesResponse = responses.ListWorkspacesResponse

// NewWorkspaceHandler creates a handler with interface dependencies
func NewWorkspaceHandler(
	common *CommonServices,
	workspaceService interfaces.WorkspaceService,
) *WorkspaceHandler {
	return &WorkspaceHandler{
		common:           common,
		workspaceService: workspaceService,
	}
}

// GetWorkspace retrieves a specific workspace by its ID
// @Summary Get a workspace
// @Description Retrieves a specific workspace by its ID
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Success 200 {object} WorkspaceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{workspace_id} [get]
func (h *WorkspaceHandler) GetWorkspace(c *gin.Context) {
	workspaceId := c.Param("workspace_id")
	parsedUUID, err := uuid.Parse(workspaceId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	workspace, err := h.workspaceService.GetWorkspace(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == constants.WorkspaceNotFound {
			sendError(c, http.StatusNotFound, "Workspace not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, "Failed to retrieve workspace", err)
		return
	}

	sendSuccess(c, http.StatusOK, toWorkspaceResponse(*workspace))
}

// ListWorkspaces retrieves all workspaces for the current account
// @Summary List workspaces
// @Description Retrieves all workspaces for the current account
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {object} ListWorkspacesResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces [get]
func (h *WorkspaceHandler) ListWorkspaces(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		sendError(c, http.StatusBadRequest, "Workspace ID is required", nil)
		return
	}

	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Get account for the workspace using service
	account, err := h.workspaceService.GetAccountByWorkspace(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve account", err)
		return
	}

	// List workspaces for the account using service
	workspaces, err := h.workspaceService.ListWorkspacesByAccount(c.Request.Context(), account.ID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve workspaces", err)
		return
	}

	response := make([]WorkspaceResponse, len(workspaces))
	for i, workspace := range workspaces {
		response[i] = toWorkspaceResponse(workspace)
	}

	sendList(c, response)
}

func (h *WorkspaceHandler) GetAllWorkspaces(c *gin.Context) {
	workspaces, err := h.workspaceService.ListAllWorkspaces(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve workspaces", err)
		return
	}

	response := make([]WorkspaceResponse, len(workspaces))
	for i, workspace := range workspaces {
		response[i] = toWorkspaceResponse(workspace)
	}

	sendList(c, response)
}

// CreateWorkspace godoc
// @Summary Create a new workspace
// @Description Creates a new workspace with the specified details
// @Tags workspaces
// @Accept json
// @Produce json
// @Tags exclude
func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get account ID from context
	accountID := c.GetString("accountID")
	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID format", err)
		return
	}

	// Create workspace using service
	workspace, err := h.workspaceService.CreateWorkspace(c.Request.Context(), params.CreateWorkspaceParams{
		Name:         req.Name,
		Description:  req.Description,
		BusinessName: req.BusinessName,
		BusinessType: req.BusinessType,
		WebsiteURL:   req.WebsiteURL,
		SupportEmail: req.SupportEmail,
		SupportPhone: req.SupportPhone,
		AccountID:    parsedAccountID,
		Metadata:     req.Metadata,
		Livemode:     req.Livemode,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create workspace", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toWorkspaceResponse(*workspace))
}

// UpdateWorkspace godoc
// @Summary Update a workspace
// @Description Updates an existing workspace with the specified details
// @Tags workspaces
// @Accept json
// @Produce json
// @Tags exclude
func (h *WorkspaceHandler) UpdateWorkspace(c *gin.Context) {
	workspaceId := c.Param("workspace_id")
	parsedUUID, err := uuid.Parse(workspaceId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	var req UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Build update params
	updateParams := params.UpdateWorkspaceParams{
		ID:       parsedUUID,
		Metadata: req.Metadata,
	}

	// Set optional fields if provided
	if req.Name != "" {
		updateParams.Name = &req.Name
	}
	if req.Description != "" {
		updateParams.Description = &req.Description
	}
	if req.BusinessName != "" {
		updateParams.BusinessName = &req.BusinessName
	}
	if req.BusinessType != "" {
		updateParams.BusinessType = &req.BusinessType
	}
	if req.WebsiteURL != "" {
		updateParams.WebsiteURL = &req.WebsiteURL
	}
	if req.SupportEmail != "" {
		updateParams.SupportEmail = &req.SupportEmail
	}
	if req.SupportPhone != "" {
		updateParams.SupportPhone = &req.SupportPhone
	}
	// For boolean fields, always update if the request method is PUT
	if c.Request.Method == "PUT" {
		updateParams.Livemode = &req.Livemode
	}

	// Update workspace using service
	workspace, err := h.workspaceService.UpdateWorkspace(c.Request.Context(), updateParams)
	if err != nil {
		if err.Error() == constants.WorkspaceNotFound {
			sendError(c, http.StatusNotFound, "Workspace not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, "Failed to update workspace", err)
		return
	}

	sendSuccess(c, http.StatusOK, toWorkspaceResponse(*workspace))
}

// DeleteWorkspace godoc
// @Summary Delete a workspace
// @Description Deletes a workspace with the specified ID
// @Tags workspaces
// @Accept json
// @Produce json
// @Tags exclude
func (h *WorkspaceHandler) DeleteWorkspace(c *gin.Context) {
	workspaceId := c.Param("workspace_id")
	parsedUUID, err := uuid.Parse(workspaceId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Delete workspace using service
	err = h.workspaceService.DeleteWorkspace(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == constants.WorkspaceNotFound {
			sendError(c, http.StatusNotFound, "Workspace not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, "Failed to delete workspace", err)
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// toWorkspaceResponse converts db.Workspace to WorkspaceResponse
func toWorkspaceResponse(w db.Workspace) WorkspaceResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling workspace metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return WorkspaceResponse{
		ID:           w.ID.String(),
		Object:       "workspace",
		Name:         w.Name,
		Description:  w.Description.String,
		BusinessName: w.BusinessName.String,
		BusinessType: w.BusinessType.String,
		WebsiteURL:   w.WebsiteUrl.String,
		SupportEmail: w.SupportEmail.String,
		SupportPhone: w.SupportPhone.String,
		AccountID:    w.AccountID.String(),
		Metadata:     metadata,
		Livemode:     w.Livemode.Bool,
		CreatedAt:    w.CreatedAt.Time.Unix(),
		UpdatedAt:    w.UpdatedAt.Time.Unix(),
	}
}

// GetWorkspaceStats retrieves statistics for a workspace
// @Summary Get workspace statistics
// @Description Retrieves statistics for a specific workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace_id path string true "Workspace ID"
// @Success 200 {object} services.WorkspaceStats
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{workspace_id}/stats [get]
func (h *WorkspaceHandler) GetWorkspaceStats(c *gin.Context) {
	workspaceId := c.Param("workspace_id")
	parsedUUID, err := uuid.Parse(workspaceId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Get workspace stats using service
	stats, err := h.workspaceService.GetWorkspaceStats(c.Request.Context(), parsedUUID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve workspace statistics", err)
		return
	}

	sendSuccess(c, http.StatusOK, stats)
}
