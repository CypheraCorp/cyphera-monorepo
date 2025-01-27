package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Domain-specific handlers
type UserHandler struct {
	common *CommonServices
}

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
	Auth0ID       string                 `json:"auth0_id" binding:"required"`
	Email         string                 `json:"email" binding:"required,email"`
	FirstName     string                 `json:"first_name,omitempty"`
	LastName      string                 `json:"last_name,omitempty"`
	DisplayName   string                 `json:"display_name,omitempty"`
	PictureURL    string                 `json:"picture_url,omitempty"`
	Phone         string                 `json:"phone,omitempty"`
	Timezone      string                 `json:"timezone,omitempty"`
	Locale        string                 `json:"locale,omitempty"`
	EmailVerified bool                   `json:"email_verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
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
	UserID  string `json:"user_id" binding:"required"`
	Role    string `json:"role" binding:"required,oneof=admin support developer"`
	IsOwner bool   `json:"is_owner"`
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
// @Param id path string true "User ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
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
		Auth0ID:       req.Auth0ID,
		Email:         req.Email,
		FirstName:     pgtype.Text{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:      pgtype.Text{String: req.LastName, Valid: req.LastName != ""},
		DisplayName:   pgtype.Text{String: req.DisplayName, Valid: req.DisplayName != ""},
		PictureUrl:    pgtype.Text{String: req.PictureURL, Valid: req.PictureURL != ""},
		Phone:         pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Timezone:      pgtype.Text{String: req.Timezone, Valid: req.Timezone != ""},
		Locale:        pgtype.Text{String: req.Locale, Valid: req.Locale != ""},
		EmailVerified: pgtype.Bool{Bool: req.EmailVerified, Valid: true},
		Metadata:      metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(user))
}

// UpdateUser godoc
// @Summary Update a user
// @Description Updates the specified user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body UpdateUserRequest true "User update data"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
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
		Status:           pgtype.Text{String: req.Status, Valid: req.Status != ""},
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
// @Description Soft deletes a user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
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

// ListUserAccounts godoc
// @Summary List user's accounts
// @Description Lists all accounts associated with a user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {array} UserAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{id}/accounts [get]
func (h *UserHandler) ListUserAccounts(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	accounts, err := h.common.db.ListUserAccounts(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	response := make([]UserAccountResponse, len(accounts))
	for i, account := range accounts {
		response[i] = toUserAccountResponse(account)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// AddUserToAccount godoc
// @Summary Add user to account
// @Description Adds a user to an account with specified role
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param user body AddUserToAccountRequest true "User account data"
// @Success 200 {object} UserAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/{id}/users [post]
func (h *UserHandler) AddUserToAccount(c *gin.Context) {
	accountID := c.Param("id")
	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid account ID format"})
		return
	}

	var req AddUserToAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	parsedUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	// Check if user exists
	user, err := h.common.db.GetUserByID(c.Request.Context(), parsedUserID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	// Add user to account
	_, err = h.common.db.AddUserToAccount(c.Request.Context(), db.AddUserToAccountParams{
		UserID:    parsedUserID,
		AccountID: parsedAccountID,
		Role:      db.UserRole(req.Role),
		IsOwner:   pgtype.Bool{Bool: req.IsOwner, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to add user to account"})
		return
	}

	// Get the full user account details
	accounts, err := h.common.db.ListUserAccounts(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get user account details"})
		return
	}

	// Find the newly created relationship
	for _, account := range accounts {
		if account.ID == user.ID {
			c.JSON(http.StatusOK, toUserAccountResponse(account))
			return
		}
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve updated user account"})
}

// RemoveUserFromAccount godoc
// @Summary Remove user from account
// @Description Removes a user from an account
// @Tags users
// @Accept json
// @Produce json
// @Param accountId path string true "Account ID"
// @Param userId path string true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/{accountId}/users/{userId} [delete]
func (h *UserHandler) RemoveUserFromAccount(c *gin.Context) {
	accountID := c.Param("accountId")
	userID := c.Param("userId")

	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid account ID format"})
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	err = h.common.db.RemoveUserFromAccount(c.Request.Context(), db.RemoveUserFromAccountParams{
		UserID:    parsedUserID,
		AccountID: parsedAccountID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to remove user from account"})
		return
	}

	c.Status(http.StatusNoContent)
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
		Status:           u.Status.String,
		Metadata:         metadata,
		Created:          u.CreatedAt.Time.Unix(),
		Updated:          u.UpdatedAt.Time.Unix(),
	}
}

// Helper function to convert ListUserAccountsRow to API response
func toUserAccountResponse(u db.ListUserAccountsRow) UserAccountResponse {
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
		Status:           u.Status.String,
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
		IsOwner:      u.IsOwner.Bool,
	}
}
