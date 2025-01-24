package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetNonce godoc
// @Summary      Get authentication nonce
// @Description  Retrieves a nonce for wallet-based authentication
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Success      200  {object}  GetNonceResponse   "Returns nonce"
// @Failure      401  {object}  ErrorResponse      "Unauthorized"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /nonce [get]
func GetNonce(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/ct/nonce", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch nonce"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to get nonce from upstream"})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	var nonceResp GetNonceResponse
	if err := json.Unmarshal(body, &nonceResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, nonceResp)
}
