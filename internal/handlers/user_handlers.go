package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserResponse represents the standardized API response for user operations
type UserResponse struct {
	ID         string                 `json:"id"`
	Object     string                 `json:"object"`
	Auth0ID    string                 `json:"auth0_id"`
	Email      string                 `json:"email"`
	Role       string                 `json:"role"`
	Name       string                 `json:"name,omitempty"`
	PictureURL string                 `json:"picture_url,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Created    int64                  `json:"created"`
	Updated    int64                  `json:"updated"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Auth0ID    string                 `json:"auth0_id" binding:"required"`
	Email      string                 `json:"email" binding:"required,email"`
	Role       string                 `json:"role" binding:"required,oneof=admin account"`
	Name       string                 `json:"name,omitempty"`
	PictureURL string                 `json:"picture_url,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email      string                 `json:"email,omitempty"`
	Role       string                 `json:"role,omitempty" binding:"omitempty,oneof=admin account"`
	Name       string                 `json:"name,omitempty"`
	PictureURL string                 `json:"picture_url,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// GetUser godoc
// @Summary Get a user
// @Description Retrieves the details of an existing user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{id} [get]
func (h *HandlerClient) GetUser(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	user, err := h.db.GetUser(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// GetCurrentUser godoc
// @Summary Get current user
// @Description Retrieves the details of the currently authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} UserResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/me [get]
func (h *HandlerClient) GetCurrentUser(c *gin.Context) {
	auth0ID := c.GetString("auth0ID")
	if auth0ID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Not authenticated"})
		return
	}

	user, err := h.db.GetUserByAuth0ID(c.Request.Context(), auth0ID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// ListUsers godoc
// @Summary List all users
// @Description Returns a list of users. Only accessible by admins.
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} UserResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users [get]
func (h *HandlerClient) ListUsers(c *gin.Context) {
	// Only admins can list all users
	if c.GetString("userRole") != "admin" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only admins can list all users"})
		return
	}

	users, err := h.db.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve users"})
		return
	}

	response := make([]UserResponse, len(users))
	for i, user := range users {
		response[i] = toUserResponse(user)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// CreateUser godoc
// @Summary Create a user
// @Description Creates a new user object. Only accessible by admins.
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User creation data"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users [post]
func (h *HandlerClient) CreateUser(c *gin.Context) {
	// Only admins can create users
	if c.GetString("userRole") != "admin" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only admins can create users"})
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	user, err := h.db.CreateUser(c.Request.Context(), db.CreateUserParams{
		Auth0ID:    req.Auth0ID,
		Email:      req.Email,
		Role:       db.UserRole(req.Role),
		Name:       pgtype.Text{String: req.Name, Valid: req.Name != ""},
		PictureUrl: pgtype.Text{String: req.PictureURL, Valid: req.PictureURL != ""},
		Metadata:   metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// UpdateUser godoc
// @Summary Update a user
// @Description Updates the specified user by setting the values of the parameters passed
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body UpdateUserRequest true "User update data"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{id} [put]
func (h *HandlerClient) UpdateUser(c *gin.Context) {
	// Only admins can update other users
	currentRole := c.GetString("userRole")
	currentAuth0ID := c.GetString("auth0ID")

	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	// Get the user being updated
	targetUser, err := h.db.GetUser(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	// Check permissions
	if currentRole != "admin" && currentAuth0ID != targetUser.Auth0ID {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "You can only update your own user data"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	// Only admins can update roles
	role := targetUser.Role
	if req.Role != "" {
		if currentRole != "admin" {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only admins can update user roles"})
			return
		}
		role = db.UserRole(req.Role)
	}

	user, err := h.db.UpdateUser(c.Request.Context(), db.UpdateUserParams{
		ID:         parsedUUID,
		Email:      req.Email,
		Role:       role,
		Name:       pgtype.Text{String: req.Name, Valid: req.Name != ""},
		PictureUrl: pgtype.Text{String: req.PictureURL, Valid: req.PictureURL != ""},
		Metadata:   metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Deletes a user. Only accessible by admins.
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{id} [delete]
func (h *HandlerClient) DeleteUser(c *gin.Context) {
	// Only admins can delete users
	if c.GetString("userRole") != "admin" {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only admins can delete users"})
		return
	}

	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	err = h.db.DeleteUser(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper function to convert database model to API response
func toUserResponse(u db.User) UserResponse {
	var metadata map[string]interface{}
	json.Unmarshal(u.Metadata, &metadata)

	return UserResponse{
		ID:         u.ID.String(),
		Object:     "user",
		Auth0ID:    u.Auth0ID,
		Email:      u.Email,
		Role:       string(u.Role),
		Name:       u.Name.String,
		PictureURL: u.PictureUrl.String,
		Metadata:   metadata,
		Created:    u.CreatedAt.Time.Unix(),
		Updated:    u.UpdatedAt.Time.Unix(),
	}
}
