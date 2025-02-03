package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// Domain-specific handlers
type UserHandler struct {
	common *CommonServices
}

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(common *CommonServices) *UserHandler {
	return &UserHandler{common: common}
}

// UserResponse represents the standardized API response for user operations
type UserResponse struct {
	ID               string                 `json:"id"`
	Object           string                 `json:"object"`
	Auth0ID          string                 `json:"auth0_id"`
	Email            string                 `json:"email"`
	FirstName        string                 `json:"first_name,omitempty"`
	LastName         string                 `json:"last_name,omitempty"`
	DisplayName      string                 `json:"display_name,omitempty"`
	PictureURL       string                 `json:"picture_url,omitempty"`
	Phone            string                 `json:"phone,omitempty"`
	Timezone         string                 `json:"timezone,omitempty"`
	Locale           string                 `json:"locale,omitempty"`
	EmailVerified    bool                   `json:"email_verified"`
	TwoFactorEnabled bool                   `json:"two_factor_enabled"`
	Status           string                 `json:"status"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Created          int64                  `json:"created"`
	Updated          int64                  `json:"updated"`
}

// UserAccountResponse represents a user's relationship with an account
type UserAccountResponse struct {
	UserResponse
	AccountName string `json:"account_name"`
	Role        string `json:"role"`
	IsOwner     bool   `json:"is_owner"`
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Auth0ID        string                 `json:"auth0_id" binding:"required"`
	Email          string                 `json:"email" binding:"required,email"`
	AccountID      uuid.UUID              `json:"account_id" binding:"required"`
	Role           string                 `json:"role" binding:"required,oneof=admin support developer"`
	IsAccountOwner bool                   `json:"is_account_owner"`
	FirstName      string                 `json:"first_name,omitempty"`
	LastName       string                 `json:"last_name,omitempty"`
	DisplayName    string                 `json:"display_name,omitempty"`
	PictureURL     string                 `json:"picture_url,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	Timezone       string                 `json:"timezone,omitempty"`
	Locale         string                 `json:"locale,omitempty"`
	EmailVerified  bool                   `json:"email_verified"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email            string                 `json:"email,omitempty"`
	FirstName        string                 `json:"first_name,omitempty"`
	LastName         string                 `json:"last_name,omitempty"`
	DisplayName      string                 `json:"display_name,omitempty"`
	PictureURL       string                 `json:"picture_url,omitempty"`
	Phone            string                 `json:"phone,omitempty"`
	Timezone         string                 `json:"timezone,omitempty"`
	Locale           string                 `json:"locale,omitempty"`
	EmailVerified    *bool                  `json:"email_verified,omitempty"`
	TwoFactorEnabled *bool                  `json:"two_factor_enabled,omitempty"`
	Status           string                 `json:"status,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// AddUserToAccountRequest represents the request to add a user to an account
type AddUserToAccountRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin support developer"`
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
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	auth0ID := c.GetString("auth0ID")
	if auth0ID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Not authenticated"})
		return
	}

	user, err := h.common.db.GetUserByAuth0ID(c.Request.Context(), auth0ID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// GetUser godoc
// @Summary Get a user
// @Description Retrieves the details of an existing user
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{user_id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	user, err := h.common.db.GetUserByID(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// CreateUser godoc
// @Summary Create a user
// @Description Creates a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User creation data"
// @Success 201 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "User already exists"
// @Security ApiKeyAuth
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
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

	user, err := h.common.db.CreateUser(c.Request.Context(), db.CreateUserParams{
		Auth0ID:        req.Auth0ID,
		Email:          req.Email,
		AccountID:      req.AccountID,
		Role:           db.UserRole(req.Role),
		IsAccountOwner: pgtype.Bool{Bool: req.IsAccountOwner, Valid: true},
		FirstName:      pgtype.Text{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:       pgtype.Text{String: req.LastName, Valid: req.LastName != ""},
		DisplayName:    pgtype.Text{String: req.DisplayName, Valid: req.DisplayName != ""},
		PictureUrl:     pgtype.Text{String: req.PictureURL, Valid: req.PictureURL != ""},
		Phone:          pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Timezone:       pgtype.Text{String: req.Timezone, Valid: req.Timezone != ""},
		Locale:         pgtype.Text{String: req.Locale, Valid: req.Locale != ""},
		EmailVerified:  pgtype.Bool{Bool: req.EmailVerified, Valid: true},
		Metadata:       metadata,
	})
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "User already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
}

// UpdateUser godoc
// @Summary Update a user
// @Description Updates an existing user's information
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param user body UpdateUserRequest true "User update data"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{user_id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
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

	user, err := h.common.db.UpdateUser(c.Request.Context(), db.UpdateUserParams{
		ID:               parsedUUID,
		Email:            req.Email,
		FirstName:        pgtype.Text{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:         pgtype.Text{String: req.LastName, Valid: req.LastName != ""},
		DisplayName:      pgtype.Text{String: req.DisplayName, Valid: req.DisplayName != ""},
		PictureUrl:       pgtype.Text{String: req.PictureURL, Valid: req.PictureURL != ""},
		Phone:            pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Timezone:         pgtype.Text{String: req.Timezone, Valid: req.Timezone != ""},
		Locale:           pgtype.Text{String: req.Locale, Valid: req.Locale != ""},
		EmailVerified:    pgtype.Bool{Bool: *req.EmailVerified, Valid: req.EmailVerified != nil},
		TwoFactorEnabled: pgtype.Bool{Bool: *req.TwoFactorEnabled, Valid: req.TwoFactorEnabled != nil},
		Status:           db.NullUserStatus{UserStatus: db.UserStatus(req.Status), Valid: req.Status != ""},
		Metadata:         metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Soft deletes a user from the system
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	err = h.common.db.DeleteUser(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetUserAccount godoc
// @Summary Get user's account
// @Description Gets the account details associated with a user
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} UserAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{user_id}/account [get]
func (h *UserHandler) GetUserAccount(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	userAccount, err := h.common.db.GetUserAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	response := toUserAccountResponse(userAccount)
	c.JSON(http.StatusOK, response)
}

// GetUserByAuth0ID godoc
// @Summary Get user by Auth0 ID
// @Description Retrieves a user's details using their Auth0 ID
// @Tags users
// @Accept json
// @Produce json
// @Param auth0_id path string true "Auth0 ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/auth0/{auth0_id} [get]
func (h *UserHandler) GetUserByAuth0ID(c *gin.Context) {
	auth0ID := c.Param("auth0_id")
	if auth0ID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "auth0_id is required"})
		return
	}

	user, err := h.common.db.GetUserByAuth0ID(c.Request.Context(), auth0ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to get user"})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// Helper function to convert database model to API response
func toUserResponse(u db.User) UserResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(u.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling user metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return UserResponse{
		ID:               u.ID.String(),
		Object:           "user",
		Auth0ID:          u.Auth0ID,
		Email:            u.Email,
		FirstName:        u.FirstName.String,
		LastName:         u.LastName.String,
		DisplayName:      u.DisplayName.String,
		PictureURL:       u.PictureUrl.String,
		Phone:            u.Phone.String,
		Timezone:         u.Timezone.String,
		Locale:           u.Locale.String,
		EmailVerified:    u.EmailVerified.Bool,
		TwoFactorEnabled: u.TwoFactorEnabled.Bool,
		Status:           string(u.Status.UserStatus),
		Metadata:         metadata,
		Created:          u.CreatedAt.Time.Unix(),
		Updated:          u.UpdatedAt.Time.Unix(),
	}
}

// Helper function to convert GetUserAccountRow to API response
func toUserAccountResponse(u db.GetUserAccountRow) UserAccountResponse {
	userResponse := UserResponse{
		ID:               u.ID.String(),
		Object:           "user",
		Auth0ID:          u.Auth0ID,
		Email:            u.Email,
		FirstName:        u.FirstName.String,
		LastName:         u.LastName.String,
		DisplayName:      u.DisplayName.String,
		PictureURL:       u.PictureUrl.String,
		Phone:            u.Phone.String,
		Timezone:         u.Timezone.String,
		Locale:           u.Locale.String,
		EmailVerified:    u.EmailVerified.Bool,
		TwoFactorEnabled: u.TwoFactorEnabled.Bool,
		Status:           string(u.Status.UserStatus),
		Created:          u.CreatedAt.Time.Unix(),
		Updated:          u.UpdatedAt.Time.Unix(),
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(u.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling user metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}
	userResponse.Metadata = metadata

	return UserAccountResponse{
		UserResponse: userResponse,
		AccountName:  u.AccountName,
		Role:         string(u.Role),
		IsOwner:      u.IsAccountOwner.Bool,
	}
}
