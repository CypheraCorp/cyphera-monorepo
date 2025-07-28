package handlers

import (
	"fmt"

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
