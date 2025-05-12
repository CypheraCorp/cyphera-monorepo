package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

type HealthResponse struct {
	Status string `json:"status"`
}

// Health godoc
// @Summary      Health check
// @Description  Checks if the server is running
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  HealthResponse   "Returns health status"
// @Router       /health [get]
// @exclude
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}
