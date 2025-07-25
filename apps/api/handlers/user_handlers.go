package handlers

import (
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Domain-specific handlers
type UserHandler struct {
	common      *CommonServices
	userService *services.UserService
}

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(common *CommonServices) *UserHandler {
	return &UserHandler{
		common:      common,
		userService: services.NewUserService(common.db),
	}
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Web3AuthID     string                 `json:"web3auth_id,omitempty"` // Web3Auth user ID
	Verifier       string                 `json:"verifier,omitempty"`    // Login method (google, discord, etc.)
	VerifierID     string                 `json:"verifier_id,omitempty"` // ID from the verifier
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

	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Get user with workspace access validation
	user, err := h.userService.GetUserWithWorkspaceAccess(c.Request.Context(), parsedUUID, parsedWorkspaceID)
	if err != nil {
		if err.Error() == "user not found" {
			sendError(c, http.StatusNotFound, "User not found", nil)
			return
		}
		if err.Error() == "workspace not found" {
			sendError(c, http.StatusNotFound, "Workspace not found", nil)
			return
		}
		if err.Error() == "user does not have access to this workspace" {
			sendError(c, http.StatusForbidden, "User does not have access to this workspace", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToUserResponse(*user))
}

// CreateUser godoc
// @Summary Create a new user
// @Description Creates a new user with the specified details
// @Tags users
// @Accept json
// @Produce json
// @Tags exclude
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

	// Create user using the service
	user, err := h.userService.CreateUser(c.Request.Context(), services.CreateUserParams{
		Web3AuthID:     req.Web3AuthID,
		Verifier:       req.Verifier,
		VerifierID:     req.VerifierID,
		Email:          req.Email,
		AccountID:      parsedAccountID,
		Role:           req.Role,
		IsAccountOwner: req.IsAccountOwner,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		AddressLine1:   req.AddressLine1,
		AddressLine2:   req.AddressLine2,
		City:           req.City,
		StateRegion:    req.StateRegion,
		PostalCode:     req.PostalCode,
		Country:        req.Country,
		DisplayName:    req.DisplayName,
		PictureURL:     req.PictureURL,
		Phone:          req.Phone,
		Timezone:       req.Timezone,
		Locale:         req.Locale,
		EmailVerified:  req.EmailVerified,
		Metadata:       req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusCreated, helpers.ToUserResponse(*user))
}

// UpdateUser godoc
// @Summary Update a user
// @Description Updates an existing user with the specified details
// @Tags users
// @Accept json
// @Produce json
// @Tags exclude
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

	// Update user using the service
	user, err := h.userService.UpdateUser(c.Request.Context(), services.UpdateUserParams{
		ID:               parsedUUID,
		Email:            req.Email,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		AddressLine1:     req.AddressLine1,
		AddressLine2:     req.AddressLine2,
		City:             req.City,
		StateRegion:      req.StateRegion,
		PostalCode:       req.PostalCode,
		Country:          req.Country,
		DisplayName:      req.DisplayName,
		PictureURL:       req.PictureURL,
		Phone:            req.Phone,
		Timezone:         req.Timezone,
		Locale:           req.Locale,
		EmailVerified:    req.EmailVerified,
		TwoFactorEnabled: req.TwoFactorEnabled,
		Status:           req.Status,
		Metadata:         req.Metadata,
	})
	if err != nil {
		if err.Error() == "user not found" {
			sendError(c, http.StatusNotFound, "User not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToUserResponse(*user))
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Deletes a user with the specified ID
// @Tags users
// @Accept json
// @Produce json
// @Tags exclude
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	err = h.userService.DeleteUser(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == "user not found" {
			sendError(c, http.StatusNotFound, "User not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusNoContent, nil)
}

// GetUserAccount godoc
// @Summary Get user account by ID
// @Description Gets a user account by their ID
// @Tags users
// @Accept json
// @Produce json
// @Tags exclude
func (h *UserHandler) GetUserAccount(c *gin.Context) {
	userId := c.Param("user_id")
	parsedUUID, err := uuid.Parse(userId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	userAccount, err := h.userService.GetUserAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == "user not found" {
			sendError(c, http.StatusNotFound, "User not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	response := helpers.ToUserAccountResponse(*userAccount)
	sendSuccess(c, http.StatusOK, response)
}
