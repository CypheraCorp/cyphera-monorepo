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
	AddressLine1     string                 `json:"address_line_1,omitempty"`
	AddressLine2     string                 `json:"address_line_2,omitempty"`
	City             string                 `json:"city,omitempty"`
	StateRegion      string                 `json:"state_region,omitempty"`
	PostalCode       string                 `json:"postal_code,omitempty"`
	Country          string                 `json:"country,omitempty"`
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
	AddressLine1   string                 `json:"address_line_1,omitempty"`
	AddressLine2   string                 `json:"address_line_2,omitempty"`
	City           string                 `json:"city,omitempty"`
	StateRegion    string                 `json:"state_region,omitempty"`
	PostalCode     string                 `json:"postal_code,omitempty"`
	Country        string                 `json:"country,omitempty"`
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
	AddressLine1     string                 `json:"address_line_1,omitempty"`
	AddressLine2     string                 `json:"address_line_2,omitempty"`
	City             string                 `json:"city,omitempty"`
	StateRegion      string                 `json:"state_region,omitempty"`
	PostalCode       string                 `json:"postal_code,omitempty"`
	Country          string                 `json:"country,omitempty"`
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
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID == "" {
		sendError(c, http.StatusBadRequest, "Workspace ID is required", nil)
		return
	}

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

	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		handleDBError(c, err, "Workspace not found")
		return
	}

	if user.AccountID.String() != workspace.AccountID.String() {
		sendError(c, http.StatusForbidden, "User does not have access to this workspace", nil)
		return
	}

	sendSuccess(c, http.StatusOK, toUserResponse(user))
}

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
		AddressLine1:   pgtype.Text{String: req.AddressLine1, Valid: req.AddressLine1 != ""},
		AddressLine2:   pgtype.Text{String: req.AddressLine2, Valid: req.AddressLine2 != ""},
		City:           pgtype.Text{String: req.City, Valid: req.City != ""},
		StateRegion:    pgtype.Text{String: req.StateRegion, Valid: req.StateRegion != ""},
		PostalCode:     pgtype.Text{String: req.PostalCode, Valid: req.PostalCode != ""},
		Country:        pgtype.Text{String: req.Country, Valid: req.Country != ""},
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
		AddressLine1:     pgtype.Text{String: req.AddressLine1, Valid: req.AddressLine1 != ""},
		AddressLine2:     pgtype.Text{String: req.AddressLine2, Valid: req.AddressLine2 != ""},
		City:             pgtype.Text{String: req.City, Valid: req.City != ""},
		StateRegion:      pgtype.Text{String: req.StateRegion, Valid: req.StateRegion != ""},
		PostalCode:       pgtype.Text{String: req.PostalCode, Valid: req.PostalCode != ""},
		Country:          pgtype.Text{String: req.Country, Valid: req.Country != ""},
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
		AddressLine1:     u.AddressLine1.String,
		AddressLine2:     u.AddressLine2.String,
		City:             u.City.String,
		StateRegion:      u.StateRegion.String,
		PostalCode:       u.PostalCode.String,
		Country:          u.Country.String,
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
		AddressLine1:     u.AddressLine1.String,
		AddressLine2:     u.AddressLine2.String,
		City:             u.City.String,
		StateRegion:      u.StateRegion.String,
		PostalCode:       u.PostalCode.String,
		Country:          u.Country.String,
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
