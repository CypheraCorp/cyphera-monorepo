package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WorkspaceHandler handles workspace related operations
type WorkspaceHandler struct {
	common           *CommonServices
	workspaceService *services.WorkspaceService
}

// NewWorkspaceHandler creates a new instance of WorkspaceHandler
func NewWorkspaceHandler(common *CommonServices) *WorkspaceHandler {
	return &WorkspaceHandler{
		common:           common,
		workspaceService: services.NewWorkspaceService(common.db),
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
		if err.Error() == "workspace not found" {
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

type CreateWorkspaceRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Description  string                 `json:"description,omitempty"`
	BusinessName string                 `json:"business_name" binding:"required"`
	BusinessType string                 `json:"business_type,omitempty"`
	WebsiteURL   string                 `json:"website_url,omitempty"`
	SupportEmail string                 `json:"support_email,omitempty"`
	SupportPhone string                 `json:"support_phone,omitempty"`
	AccountID    string                 `json:"account_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Livemode     bool                   `json:"livemode,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	BusinessName string                 `json:"business_name,omitempty"`
	BusinessType string                 `json:"business_type,omitempty"`
	WebsiteURL   string                 `json:"website_url,omitempty"`
	SupportEmail string                 `json:"support_email,omitempty"`
	SupportPhone string                 `json:"support_phone,omitempty"`
	AccountID    string                 `json:"account_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Livemode     bool                   `json:"livemode,omitempty"`
}

// WorkspaceResponse represents the standardized API response for workspace operations
// @Summary Workspace response
// @Description Workspace response
type WorkspaceResponse struct {
	ID           string                 `json:"id"`
	Object       string                 `json:"object"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	BusinessName string                 `json:"business_name"`
	BusinessType string                 `json:"business_type,omitempty"`
	WebsiteURL   string                 `json:"website_url,omitempty"`
	SupportEmail string                 `json:"support_email,omitempty"`
	SupportPhone string                 `json:"support_phone,omitempty"`
	AccountID    string                 `json:"account_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Livemode     bool                   `json:"livemode"`
	CreatedAt    int64                  `json:"created_at"`
	UpdatedAt    int64                  `json:"updated_at"`
}

// ListWorkspacesResponse represents the response for listing workspaces
type ListWorkspacesResponse struct {
	Object string              `json:"object"`
	Data   []WorkspaceResponse `json:"data"`
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
	workspace, err := h.workspaceService.CreateWorkspace(c.Request.Context(), services.CreateWorkspaceParams{
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
	updateParams := services.UpdateWorkspaceParams{
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
		if err.Error() == "workspace not found" {
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
		if err.Error() == "workspace not found" {
			sendError(c, http.StatusNotFound, "Workspace not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, "Failed to delete workspace", err)
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// safeParseInt32 safely parses a string to int32, checking for overflow
func safeParseInt32(s string) (int32, error) {
	// Parse as int64 first to check for overflow
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}

	// Check if value fits in int32
	if val > math.MaxInt32 || val < math.MinInt32 {
		return 0, fmt.Errorf("value %d overflows int32", val)
	}

	return int32(val), nil
}

// parsePaginationParams parses and validates pagination parameters
func parsePaginationParams(c *gin.Context) (limit int32, offset int32, err error) {
	const maxLimit int32 = 100
	const defaultLimit int32 = 10
	const defaultOffset int32 = 0

	limitStr := c.DefaultQuery("limit", strconv.Itoa(int(defaultLimit)))
	offsetStr := c.DefaultQuery("offset", strconv.Itoa(int(defaultOffset)))

	// Parse limit
	limit, err = safeParseInt32(limitStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid limit: %w", err)
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	// Parse offset
	offset, err = safeParseInt32(offsetStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid offset: %w", err)
	}
	if offset < 0 {
		offset = defaultOffset
	}

	return limit, offset, nil
}

// Helper function to convert database model to API response
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
