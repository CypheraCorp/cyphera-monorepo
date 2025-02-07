package handlers

import (
	"bytes"
	"io"
	"log"

	"github.com/gin-gonic/gin"
)

// responseBodyWriter is a custom response writer that captures the response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response body while writing it to the original writer
func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LogRequestBody middleware logs the request and response body
func LogRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read the Request Body content
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
		}

		// Restore the io.ReadCloser to its original state
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Log Request Info
		log.Printf("Request URI: %s", c.Request.URL.Path)
		log.Printf("Request Method: %s", c.Request.Method)

		// Log Query Parameters
		params := c.Request.URL.Query()
		if len(params) > 0 {
			log.Printf("Query Parameters:")
			for key, value := range params {
				log.Printf("  %s: %v", key, value)
			}
		}

		// Log Request Headers
		log.Printf("Request Headers:")
		accountIDStr := c.GetHeader("X-Account-ID")
		workspaceIDStr := c.GetHeader("X-Workspace-ID")
		userIDStr := c.GetHeader("X-User-ID")
		apiKey := c.GetHeader("X-API-Key")
		jwtToken := c.GetHeader("Authorization")

		if accountIDStr != "" {
			log.Printf("  X-Account-ID: %s", accountIDStr)
		}
		if workspaceIDStr != "" {
			log.Printf("  X-Workspace-ID: %s", workspaceIDStr)
		}
		if userIDStr != "" {
			log.Printf("  X-User-ID: %s", userIDStr)
		}
		if apiKey != "" {
			log.Printf("  X-API-Key: %s", apiKey)
		}
		if jwtToken != "" {
			log.Printf("  Authorization: %s", jwtToken[:50])
		}

		// Log Request Body
		if len(bodyBytes) > 0 {
			log.Printf("Request Body: %s", string(bodyBytes))
		}

		// Create a custom response writer to capture the response body
		responseBody := &bytes.Buffer{}
		writer := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           responseBody,
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Log Response Info
		log.Printf("Response Status: %d", c.Writer.Status())

		// Log Response Body
		if responseBody.Len() > 0 {
			log.Printf("Response Body: %s", responseBody.String())
		}
	}
}
