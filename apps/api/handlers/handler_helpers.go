package handlers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetWorkspaceID extracts workspace ID from various sources in order of priority:
// 1. X-Workspace-ID header
// 2. URL parameter workspace_id
// 3. Context value set by auth middleware
func GetWorkspaceID(c *gin.Context) (uuid.UUID, error) {
	// First try to get from header
	workspaceIDStr := c.GetHeader("X-Workspace-ID")
	if workspaceIDStr == "" {
		// Try to get from URL param
		workspaceIDStr = c.Param("workspace_id")
	}
	if workspaceIDStr == "" {
		// Try to get from context (set by auth middleware)
		if val, exists := c.Get("workspaceID"); exists {
			if id, ok := val.(uuid.UUID); ok {
				return id, nil
			}
		}
		return uuid.Nil, fmt.Errorf("workspace ID not found")
	}
	
	return uuid.Parse(workspaceIDStr)
}

// GetPaginationParams extracts and validates pagination parameters from query string
// Returns limit and offset with defaults of 20 and 0 respectively
func GetPaginationParams(c *gin.Context) (limit, offset int) {
	limit = 20
	offset = 0
	
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}
	
	if o := c.Query("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}
	
	return limit, offset
}