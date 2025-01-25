package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *HandlerClient) GetCustomerByID(c *gin.Context) {
	id := c.Param("id")

	// Validate UUID format
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}
	customer, err := h.db.GetCustomer(context.Background(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, customer)
}

func (h *HandlerClient) GetAPIKeyByID(c *gin.Context) {
	id := c.Param("id")

	// Validate UUID format
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID format"})
		return
	}
	apiKey, err := h.db.GetAPIKey(context.Background(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, apiKey)
}
