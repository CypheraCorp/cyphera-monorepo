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

// GetAPIKeyByID retrieves a specific API key by its ID
func (h *HandlerClient) GetAPIKeyByID(c *gin.Context) {
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

	apiKey, err := h.db.GetAPIKey(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	// Check if the API key belongs to the account (unless admin)
	if c.GetString("userRole") != "admin" {
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

// ListAPIKeys retrieves all API keys for the current account
func (h *HandlerClient) ListAPIKeys(c *gin.Context) {
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

	apiKeys, err := h.db.ListAPIKeys(c.Request.Context(), pgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   apiKeys,
	})
}

// GetAllAPIKeys retrieves all API keys (admin only)
func (h *HandlerClient) GetAllAPIKeys(c *gin.Context) {
	// This endpoint should only be accessible by admins (handled by middleware)
	apiKeys, err := h.db.GetAllAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   apiKeys,
	})
}

// CreateAPIKey creates a new API key
func (h *HandlerClient) CreateAPIKey(c *gin.Context) {
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

	apiKey, err := h.db.CreateAPIKey(c.Request.Context(), db.CreateAPIKeyParams{
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

// UpdateAPIKey updates an existing API key
func (h *HandlerClient) UpdateAPIKey(c *gin.Context) {
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
	if c.GetString("userRole") != "admin" {
		accountID := c.GetString("accountID")
		if accountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
			return
		}

		apiKey, err := h.db.GetAPIKey(c.Request.Context(), parsedUUID)
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

	apiKey, err := h.db.UpdateAPIKey(c.Request.Context(), db.UpdateAPIKeyParams{
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

// DeleteAPIKey soft deletes an API key
func (h *HandlerClient) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}

	// Check ownership unless admin
	if c.GetString("userRole") != "admin" {
		accountID := c.GetString("accountID")
		if accountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Account ID not found in context"})
			return
		}

		apiKey, err := h.db.GetAPIKey(c.Request.Context(), parsedUUID)
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

	err = h.db.DeleteAPIKey(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetExpiredAPIKeys retrieves all expired API keys (admin only)
func (h *HandlerClient) GetExpiredAPIKeys(c *gin.Context) {
	// This endpoint should only be accessible by admins (handled by middleware)
	apiKeys, err := h.db.GetExpiredAPIKeys(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve expired API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   apiKeys,
	})
}

// GetActiveAPIKeysCount gets the count of active API keys for an account
func (h *HandlerClient) GetActiveAPIKeysCount(c *gin.Context) {
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

	count, err := h.db.GetActiveAPIKeysCount(c.Request.Context(), pgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}
