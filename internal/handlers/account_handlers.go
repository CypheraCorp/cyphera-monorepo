package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AccountHandler struct {
	common *CommonServices
}

func NewAccountHandler(common *CommonServices) *AccountHandler {
	return &AccountHandler{common: common}
}

// GetAccount retrieves a specific account by its ID
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")

	// Validate UUID format
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	account, err := h.common.db.GetAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	c.JSON(http.StatusOK, account)
}

// ListAccounts retrieves all non-deleted accounts
func (h *AccountHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.common.db.ListAccounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve accounts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   accounts,
	})
}

// GetAllAccounts retrieves all accounts including deleted ones
func (h *AccountHandler) GetAllAccounts(c *gin.Context) {
	accounts, err := h.common.db.GetAllAccounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve accounts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   accounts,
	})
}

// CreateAccount creates a new account
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req struct {
		UserID       string                 `json:"user_id" binding:"required"`
		Name         string                 `json:"name" binding:"required"`
		Description  string                 `json:"description,omitempty"`
		BusinessName string                 `json:"business_name,omitempty"`
		BusinessType string                 `json:"business_type,omitempty"`
		WebsiteURL   string                 `json:"website_url,omitempty"`
		SupportEmail string                 `json:"support_email,omitempty"`
		SupportPhone string                 `json:"support_phone,omitempty"`
		Metadata     map[string]interface{} `json:"metadata,omitempty"`
		Livemode     *bool                  `json:"livemode,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
		return
	}

	var livemode pgtype.Bool
	if req.Livemode != nil {
		livemode.Bool = *req.Livemode
		livemode.Valid = true
	}

	account, err := h.common.db.CreateAccount(c.Request.Context(), db.CreateAccountParams{
		UserID:       userUUID,
		Name:         req.Name,
		Description:  pgtype.Text{String: req.Description, Valid: req.Description != ""},
		BusinessName: pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
		BusinessType: pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
		WebsiteUrl:   pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
		SupportEmail: pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
		SupportPhone: pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
		Metadata:     metadata,
		Livemode:     livemode,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}

	c.JSON(http.StatusCreated, account)
}

// UpdateAccount updates an existing account
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	var req struct {
		Name         string                 `json:"name,omitempty"`
		Description  string                 `json:"description,omitempty"`
		BusinessName string                 `json:"business_name,omitempty"`
		BusinessType string                 `json:"business_type,omitempty"`
		WebsiteURL   string                 `json:"website_url,omitempty"`
		SupportEmail string                 `json:"support_email,omitempty"`
		SupportPhone string                 `json:"support_phone,omitempty"`
		Metadata     map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metadata format"})
		return
	}

	account, err := h.common.db.UpdateAccount(c.Request.Context(), db.UpdateAccountParams{
		ID:           parsedUUID,
		Name:         req.Name,
		Description:  pgtype.Text{String: req.Description, Valid: req.Description != ""},
		BusinessName: pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
		BusinessType: pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
		WebsiteUrl:   pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
		SupportEmail: pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
		SupportPhone: pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
		Metadata:     metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account"})
		return
	}

	c.JSON(http.StatusOK, account)
}

// DeleteAccount soft deletes an account
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	err = h.common.db.DeleteAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	c.Status(http.StatusNoContent)
}

// HardDeleteAccount permanently deletes an account
func (h *AccountHandler) HardDeleteAccount(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	err = h.common.db.HardDeleteAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to permanently delete account"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAccountCustomers retrieves all customers for a specific account
func (h *AccountHandler) ListAccountCustomers(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	customers, err := h.common.db.ListAccountCustomers(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account customers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   customers,
	})
}
