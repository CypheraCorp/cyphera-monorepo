package handlers

import (
	"bytes"
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

		// Log the body
		log.Printf("Request Body: %s", string(bodyBytes))

		// Continue
		c.Next()
	}
}
