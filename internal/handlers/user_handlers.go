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

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(common *CommonServices) *UserHandler {
	return &UserHandler{common: common}
}

// UserResponse represents the standardized API response for user operations
type UserResponse struct {
	ID               string                 `json:"id"`
	Object           string                 `json:"object"`
	SupabaseID       string                 `json:"supabase_id"`
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
	CreatedAt        int64                  `json:"created_at"`
	UpdatedAt        int64                  `json:"updated_at"`
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
	SupabaseID     string                 `json:"supabase_id" binding:"required"`
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
	userID := c.GetString("userID")
	parsedUUID, err := uuid.Parse(userID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	user, err := h.common.db.GetUserByID(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "User not found")
		return
	}

	sendSuccess(c, http.StatusOK, toUserResponse(user))
}

// GetUser godoc
// @Summary Get user by ID
// @Description Gets a user by their ID
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
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	user, err := h.common.db.GetUserByID(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "User not found")
		return
	}

	sendSuccess(c, http.StatusOK, toUserResponse(user))
}

// CreateUser godoc
// @Summary Create user
// @Description Creates a new user
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User creation data"
// @Success 201 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	accountID := c.GetString("accountID")
	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID format", err)
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	user, err := h.common.db.CreateUser(c.Request.Context(), db.CreateUserParams{
		SupabaseID:     req.SupabaseID,
		Email:          req.Email,
		AccountID:      parsedAccountID,
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
		sendError(c, http.StatusInternalServerError, "Failed to CreateUser", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toUserResponse(user))
}

// UpdateUser godoc
// @Summary Update user
// @Description Updates a user's information
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param user body UpdateUserRequest true "User update data"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{user_id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	var emailVerified, twoFactorEnabled pgtype.Bool
	if req.EmailVerified != nil {
		emailVerified = pgtype.Bool{Bool: *req.EmailVerified, Valid: true}
	}
	if req.TwoFactorEnabled != nil {
		twoFactorEnabled = pgtype.Bool{Bool: *req.TwoFactorEnabled, Valid: true}
	}

	var status db.NullUserStatus
	if req.Status != "" {
		status = db.NullUserStatus{
			UserStatus: db.UserStatus(req.Status),
			Valid:      true,
		}
	}

	user, err := h.common.db.UpdateUser(c.Request.Context(), db.UpdateUserParams{
		Email:            parsedUUID.String(), // First parameter is used for both ID and email in the query
		FirstName:        pgtype.Text{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:         pgtype.Text{String: req.LastName, Valid: req.LastName != ""},
		DisplayName:      pgtype.Text{String: req.DisplayName, Valid: req.DisplayName != ""},
		PictureUrl:       pgtype.Text{String: req.PictureURL, Valid: req.PictureURL != ""},
		Phone:            pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Timezone:         pgtype.Text{String: req.Timezone, Valid: req.Timezone != ""},
		Locale:           pgtype.Text{String: req.Locale, Valid: req.Locale != ""},
		EmailVerified:    emailVerified,
		TwoFactorEnabled: twoFactorEnabled,
		Status:           status,
		Metadata:         metadata,
	})
	if err != nil {
		handleDBError(c, err, "Failed to update user")
		return
	}

	sendSuccess(c, http.StatusOK, toUserResponse(user))
}

// DeleteUser godoc
// @Summary Delete user
// @Description Deletes a user
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/{user_id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	err = h.common.db.DeleteUser(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to delete user")
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
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
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	userAccount, err := h.common.db.GetUserAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "User not found")
		return
	}

	response := toUserAccountResponse(userAccount)
	sendSuccess(c, http.StatusOK, response)
}

// GetUserBySupabaseID godoc
// @Summary Get user by Supabase ID
// @Description Gets a user by their Supabase ID
// @Tags users
// @Accept json
// @Produce json
// @Param supabase_id query string true "Supabase ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /users/supabase [get]
func (h *UserHandler) GetUserBySupabaseID(c *gin.Context) {
	supabaseID := c.Query("supabase_id")
	if supabaseID == "" {
		sendError(c, http.StatusBadRequest, "Supabase ID is required", nil)
		return
	}

	user, err := h.common.db.GetUserBySupabaseID(c.Request.Context(), supabaseID)
	if err != nil {
		handleDBError(c, err, "User not found")
		return
	}

	sendSuccess(c, http.StatusOK, toUserResponse(user))
}

// Helper function to convert database model to API response
func toUserResponse(u db.User) UserResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(u.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling user metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	var status string
	if u.Status.Valid {
		status = string(u.Status.UserStatus)
	}

	return UserResponse{
		ID:               u.ID.String(),
		Object:           "user",
		SupabaseID:       u.SupabaseID,
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
		Status:           status,
		Metadata:         metadata,
		CreatedAt:        u.CreatedAt.Time.Unix(),
		UpdatedAt:        u.UpdatedAt.Time.Unix(),
	}
}

// Helper function to convert GetUserAccountRow to API response
func toUserAccountResponse(u db.GetUserAccountRow) UserAccountResponse {
	userResponse := UserResponse{
		ID:               u.ID.String(),
		Object:           "user",
		SupabaseID:       u.SupabaseID,
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
		CreatedAt:        u.CreatedAt.Time.Unix(),
		UpdatedAt:        u.UpdatedAt.Time.Unix(),
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
