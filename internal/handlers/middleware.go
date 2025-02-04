package handlers

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/gin-gonic/gin"
)

// LogRequestBody middleware logs the request body
func LogRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read the Body content
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
		}

		// Restore the io.ReadCloser to its original state
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// uri path
		uriPath := c.Request.URL.Path
		fmt.Println("uriPath", uriPath)

		// params
		params := c.Request.URL.Query()
		for key, value := range params {
			fmt.Println("key", key, "value", value)
		}

		// Log the body
		log.Printf("Request Body: %s", string(bodyBytes))

		accountIDStr := c.GetHeader("X-Account-ID")
		workspaceIDStr := c.GetHeader("X-Workspace-ID")
		userIDStr := c.GetHeader("X-User-ID")
		apiKey := c.GetHeader("X-API-Key")
		jwtToken := c.GetHeader("Authorization")
		fmt.Println("accountIDStr", accountIDStr)
		fmt.Println("workspaceIDStr", workspaceIDStr)
		fmt.Println("userIDStr", userIDStr)
		if apiKey != "" {
			fmt.Println("apiKey", apiKey)
		} else if jwtToken != "" {
			fmt.Println("jwtToken", jwtToken)
		}

		// Continue
		c.Next()
	}
}
