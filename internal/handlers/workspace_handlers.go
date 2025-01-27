package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// WorkspaceHandler handles workspace related operations
type WorkspaceHandler struct {
	common *CommonServices
}

// NewWorkspaceHandler creates a new instance of WorkspaceHandler
func NewWorkspaceHandler(common *CommonServices) *WorkspaceHandler {
	return &WorkspaceHandler{common: common}
}

// GetWorkspace retrieves a specific workspace by its ID
// @Summary Get a workspace
// @Description Retrieves a specific workspace by its ID
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {object} WorkspaceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{id} [get]
func (h *WorkspaceHandler) GetWorkspace(c *gin.Context) {
	id := c.Param("id")

	// Validate UUID format
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
		return
	}

	c.JSON(http.StatusOK, toWorkspaceResponse(workspace))
}

// ListWorkspaces retrieves all non-deleted workspaces
// @Summary List workspaces
// @Description Retrieves all non-deleted workspaces
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} WorkspaceResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces [get]
func (h *WorkspaceHandler) ListWorkspaces(c *gin.Context) {
	workspaces, err := h.common.db.ListWorkspaces(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workspaces"})
		return
	}

	response := make([]WorkspaceResponse, len(workspaces))
	for i, workspace := range workspaces {
		response[i] = toWorkspaceResponse(workspace)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// GetAllWorkspaces retrieves all workspaces including deleted ones
// @Summary Get all workspaces
// @Description Retrieves all workspaces including deleted ones (admin only)
// @Tags workspaces
// @Accept json
// @Produce json
// @Success 200 {array} WorkspaceResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/workspaces/all [get]
func (h *WorkspaceHandler) GetAllWorkspaces(c *gin.Context) {
	workspaces, err := h.common.db.GetAllWorkspaces(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workspaces"})
		return
	}

	response := make([]WorkspaceResponse, len(workspaces))
	for i, workspace := range workspaces {
		response[i] = toWorkspaceResponse(workspace)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// CreateWorkspaceRequest represents the request body for creating a workspace
// @Summary Create workspace
// @Description Creates a new workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace body CreateWorkspaceRequest true "Workspace creation data"
// @Success 201 {object} WorkspaceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces [post]
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

// UpdateWorkspaceRequest represents the request body for updating a workspace
// @Summary Update workspace
// @Description Updates an existing workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param workspace body UpdateWorkspaceRequest true "Workspace update data"
// @Success 200 {object} WorkspaceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{id} [put]
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
	Created      int64                  `json:"created"`
	Updated      int64                  `json:"updated"`
}

// CreateWorkspace creates a new workspace
// @Summary Create workspace
// @Description Creates a new workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param workspace body CreateWorkspaceRequest true "Workspace creation data"
// @Success 201 {object} WorkspaceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces [post]
func (h *WorkspaceHandler) CreateWorkspace(c *gin.Context) {
	var req CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountUUID, err := uuid.Parse(req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID format"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
		return
	}

	workspace, err := h.common.db.CreateWorkspace(c.Request.Context(), db.CreateWorkspaceParams{
		AccountID:    accountUUID,
		Name:         req.Name,
		Description:  pgtype.Text{String: req.Description, Valid: req.Description != ""},
		BusinessName: pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
		BusinessType: pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
		WebsiteUrl:   pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
		SupportEmail: pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
		SupportPhone: pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
		Metadata:     metadata,
		Livemode:     pgtype.Bool{Bool: req.Livemode, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workspace"})
		return
	}

	c.JSON(http.StatusCreated, toWorkspaceResponse(workspace))
}

// UpdateWorkspace updates an existing workspace
// @Summary Update workspace
// @Description Updates an existing workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Param workspace body UpdateWorkspaceRequest true "Workspace update data"
// @Success 200 {object} WorkspaceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{id} [put]
func (h *WorkspaceHandler) UpdateWorkspace(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
		return
	}

	workspace, err := h.common.db.UpdateWorkspace(c.Request.Context(), db.UpdateWorkspaceParams{
		ID:           parsedUUID,
		Name:         req.Name,
		Description:  pgtype.Text{String: req.Description, Valid: req.Description != ""},
		BusinessName: pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
		BusinessType: pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
		WebsiteUrl:   pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
		SupportEmail: pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
		SupportPhone: pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
		Metadata:     metadata,
		Livemode:     pgtype.Bool{Bool: req.Livemode, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workspace"})
		return
	}

	c.JSON(http.StatusOK, toWorkspaceResponse(workspace))
}

// DeleteWorkspace godoc
// @Summary Delete workspace
// @Description Soft deletes a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{id} [delete]
func (h *WorkspaceHandler) DeleteWorkspace(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	err = h.common.db.DeleteWorkspace(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// HardDeleteWorkspace godoc
// @Summary Hard delete workspace
// @Description Permanently deletes a workspace (admin only)
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/workspaces/{id}/hard [delete]
func (h *WorkspaceHandler) HardDeleteWorkspace(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	err = h.common.db.HardDeleteWorkspace(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workspace not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListWorkspaceCustomers retrieves all customers for a workspace
// @Summary List workspace customers
// @Description Retrieves all customers for a workspace
// @Tags workspaces
// @Accept json
// @Produce json
// @Param id path string true "Workspace ID"
// @Success 200 {array} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /workspaces/{id}/customers [get]
func (h *WorkspaceHandler) ListWorkspaceCustomers(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	customers, err := h.common.db.ListWorkspaceCustomers(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve customers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   customers,
	})
}

// Helper function to convert database model to API response
func toWorkspaceResponse(w db.Workspace) WorkspaceResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Metadata, &metadata); err != nil {
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
		Created:      w.CreatedAt.Time.Unix(),
		Updated:      w.UpdatedAt.Time.Unix(),
	}
}
