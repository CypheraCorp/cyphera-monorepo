# Workspace Handler Refactoring Migration Guide

## Overview

This guide explains how to migrate from the direct handler → database pattern to the new handler → service → database pattern for the workspace handlers.

## Architecture Changes

### Before (Direct Pattern)
```
HTTP Request → Handler → Database Queries → Response
```

### After (Service Layer Pattern)
```
HTTP Request → Handler → Service → Database Queries → Response
                 ↓          ↓
              Helpers   Business Logic
```

## Benefits of the Refactored Architecture

1. **Separation of Concerns**
   - Handlers focus on HTTP request/response handling
   - Services contain business logic
   - Helpers provide reusable utilities

2. **Better Testability**
   - Services can be unit tested without HTTP context
   - Mock services easily for handler tests
   - Isolated business logic testing

3. **Code Reusability**
   - Services can be used by multiple handlers
   - Business logic can be shared across different interfaces (REST, gRPC, CLI)
   - Common validation and utilities in helpers

4. **Improved Maintainability**
   - Smaller, focused components
   - Clear responsibilities
   - Easier to locate and fix bugs

## Key Components

### 1. WorkspaceService (`/libs/go/services/workspace_service.go`)
- **Purpose**: Business logic for workspace operations
- **Responsibilities**:
  - Validation of business rules
  - Orchestration of database operations
  - Logging and error handling
  - Transaction management (when needed)

### 2. WorkspaceHandlerV2 (`/apps/api/handlers/workspace_handlers_refactored.go`)
- **Purpose**: HTTP request/response handling
- **Responsibilities**:
  - Request parsing and validation
  - Calling appropriate service methods
  - Response formatting
  - HTTP status code management

### 3. WorkspaceHelper (`/libs/go/helpers/workspace_helper.go`)
- **Purpose**: Reusable utility functions
- **Responsibilities**:
  - Input validation
  - Data formatting
  - Common transformations
  - Business rule helpers

## Migration Steps

### Step 1: Update Server Initialization

Replace the old handler with the new one in your server setup:

```go
// Before
workspaceHandler = handlers.NewWorkspaceHandler(commonServices)

// After
workspaceHandler = handlers.NewWorkspaceHandlerV2(commonServices)
```

### Step 2: Update Route Definitions

The routes remain the same, just the handler instance changes:

```go
// Routes stay the same
workspaces := protected.Group("/workspaces")
{
    workspaces.GET("", workspaceHandler.ListWorkspaces)
    workspaces.POST("", workspaceHandler.CreateWorkspace)
    workspaces.GET("/all", workspaceHandler.GetAllWorkspaces)
    workspaces.GET("/:workspace_id", workspaceHandler.GetWorkspace)
    workspaces.PUT("/:workspace_id", workspaceHandler.UpdateWorkspace)
    workspaces.DELETE("/:workspace_id", workspaceHandler.DeleteWorkspace)
    workspaces.GET("/:workspace_id/stats", workspaceHandler.GetWorkspaceStats) // New endpoint
}
```

### Step 3: Testing the Migration

1. **Unit Tests for Service**
```go
func TestWorkspaceService_CreateWorkspace(t *testing.T) {
    // Mock the database queries
    mockQueries := &MockQuerier{}
    service := services.NewWorkspaceService(mockQueries)
    
    // Test business logic
    workspace, err := service.CreateWorkspace(ctx, params)
    assert.NoError(t, err)
    assert.NotNil(t, workspace)
}
```

2. **Integration Tests for Handler**
```go
func TestWorkspaceHandler_CreateWorkspace(t *testing.T) {
    // Set up test server with real services
    handler := handlers.NewWorkspaceHandlerV2(commonServices)
    
    // Test HTTP endpoint
    w := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/workspaces", body)
    handler.CreateWorkspace(ginContext)
    
    assert.Equal(t, http.StatusCreated, w.Code)
}
```

## New Features Added

1. **Workspace Statistics Endpoint**
   - `GET /workspaces/:workspace_id/stats`
   - Returns customer, product, and subscription counts

2. **Enhanced Validation**
   - Workspace name validation
   - Business email validation
   - Website URL validation
   - Phone number formatting

3. **Metadata Management**
   - Structured metadata parsing
   - Type-safe metadata handling
   - Default values for new workspaces

4. **Workspace Features**
   - Feature flags based on workspace type
   - Mode-specific features (test vs live)

## Code Comparison Examples

### Example 1: GetWorkspace

**Before:**
```go
func (h *WorkspaceHandler) GetWorkspace(c *gin.Context) {
    workspaceId := c.Param("workspace_id")
    parsedUUID, err := uuid.Parse(workspaceId)
    if err != nil {
        sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
        return
    }

    workspace, err := h.common.db.GetWorkspace(c.Request.Context(), parsedUUID)
    if err != nil {
        handleDBError(c, err, "Workspace not found")
        return
    }

    sendSuccess(c, http.StatusOK, toWorkspaceResponse(workspace))
}
```

**After:**
```go
func (h *WorkspaceHandlerV2) GetWorkspace(c *gin.Context) {
    // Parse and validate workspace ID
    workspaceIDStr := c.Param("workspace_id")
    workspaceID, err := uuid.Parse(workspaceIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error: "Invalid workspace ID format",
        })
        return
    }

    // Use service to get workspace
    workspace, err := h.workspaceService.GetWorkspace(c.Request.Context(), workspaceID)
    if err != nil {
        if err.Error() == "workspace not found" {
            c.JSON(http.StatusNotFound, ErrorResponse{
                Error: "Workspace not found",
            })
            return
        }
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "Failed to retrieve workspace",
        })
        return
    }

    // Convert to response format
    response := h.toWorkspaceResponse(*workspace)
    c.JSON(http.StatusOK, response)
}
```

### Example 2: Using Helpers

```go
// Validate workspace name before creation
if err := workspaceHelper.ValidateWorkspaceName(req.Name); err != nil {
    c.JSON(http.StatusBadRequest, ErrorResponse{
        Error: err.Error(),
    })
    return
}

// Generate a URL-safe slug
slug := workspaceHelper.GenerateWorkspaceSlug(req.Name)

// Check uniqueness
isUnique, err := workspaceHelper.CheckWorkspaceNameUniqueness(ctx, accountID, req.Name)
if !isUnique {
    c.JSON(http.StatusConflict, ErrorResponse{
        Error: "Workspace name already exists",
    })
    return
}
```

## Best Practices

1. **Keep Handlers Thin**
   - Only handle HTTP concerns
   - Delegate business logic to services

2. **Services Should Be Framework-Agnostic**
   - Don't import Gin or HTTP packages
   - Return domain errors, not HTTP errors

3. **Use Helpers for Reusable Logic**
   - Validation functions
   - Data transformations
   - Common calculations

4. **Error Handling**
   - Services return descriptive errors
   - Handlers map errors to HTTP status codes
   - Log errors at appropriate levels

5. **Testing Strategy**
   - Unit test services with mocked dependencies
   - Integration test handlers with real services
   - Test helpers independently

## Rollback Plan

If you need to rollback:

1. Change handler initialization back to original
2. Routes remain unchanged
3. No database changes required
4. Can run both versions side-by-side during transition

## Next Steps

1. Apply similar refactoring to other handlers:
   - Customer handlers
   - Product handlers
   - Subscription handlers
   - Payment handlers

2. Create shared service interfaces for common patterns

3. Implement middleware that uses services for:
   - Authentication
   - Authorization
   - Rate limiting

4. Add comprehensive logging and metrics to services