package handlers

import (
	"bytes"
	"cyphera-api/internal/logger"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

// formatJSONBody attempts to format a JSON byte slice for pretty printing
func formatJSONBody(body []byte, contentType string) interface{} {
	// If empty or not JSON content type, return as is
	if len(body) == 0 || !strings.Contains(strings.ToLower(contentType), "application/json") {
		return string(body)
	}

	// Try to unmarshal JSON into interface{}
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		// If we can't parse as JSON, return original string
		return string(body)
	}

	// Return the unmarshaled data directly
	return jsonData
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

		// Create fields for structured logging
		fields := []zap.Field{
			zap.String("uri", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		}

		// Add query parameters if present
		params := c.Request.URL.Query()
		if len(params) > 0 {
			fields = append(fields, zap.Any("query_params", params))
		}

		// Add relevant headers
		headers := map[string]string{}
		if accountID := c.GetHeader("X-Account-ID"); accountID != "" {
			headers["X-Account-ID"] = accountID
		}
		if workspaceID := c.GetHeader("X-Workspace-ID"); workspaceID != "" {
			headers["X-Workspace-ID"] = workspaceID
		}
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			headers["X-User-ID"] = userID
		}
		if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
			headers["X-API-Key"] = apiKey[:8] + "..." // Only log first 8 chars for security
		}
		if jwt := c.GetHeader("Authorization"); jwt != "" {
			headers["Authorization"] = jwt[:8] + "..." // Only log first 8 chars for security
		}

		if len(headers) > 0 {
			fields = append(fields, zap.Any("headers", headers))
		}

		// Add request body if present
		if len(bodyBytes) > 0 {
			fields = append(fields, zap.Any("request_body", formatJSONBody(bodyBytes, c.GetHeader("Content-Type"))))
		}

		// Log the request
		logger.Debug("Incoming request", fields...)

		// Create a custom response writer to capture the response body
		responseBody := &bytes.Buffer{}
		writer := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           responseBody,
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Log the response
		responseFields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
		}

		if responseBody.Len() > 0 {
			responseFields = append(responseFields, zap.Any("response_body",
				formatJSONBody(responseBody.Bytes(), c.Writer.Header().Get("Content-Type"))))
		}

		logger.Debug("Response sent", responseFields...)
	}
}
