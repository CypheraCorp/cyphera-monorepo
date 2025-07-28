package handlers

import (
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Use types from the centralized packages
type HealthResponse = responses.HealthResponse

// Health godoc
// @Summary Check the health of the server
// @Description Returns a simple "ok" status
// @Tags health
// @Accept json
// @Produce json
// @Tags exclude
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}
