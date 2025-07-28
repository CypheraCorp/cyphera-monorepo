package handlers

import (
	"net/http"

	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler handles user-related operations
type UserHandler struct {
	common      *CommonServices
	userService interfaces.UserService
}

// Use types from the centralized packages
type CreateUserRequest = requests.CreateUserRequest
type UpdateUserRequest = requests.UpdateUserRequest
type AddUserToAccountRequest = requests.AddUserToAccountRequest

type UserResponse = responses.UserResponse
type UserAccountResponse = responses.UserAccountResponse

// NewUserHandler creates a handler with interface dependencies
func NewUserHandler(
	common *CommonServices,
	userService interfaces.UserService,
) *UserHandler {
	return &UserHandler{
		common:      common,
		userService: userService,
	}
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
	user, err := h.userService.CreateUser(c.Request.Context(), params.CreateUserParams{
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
	user, err := h.userService.UpdateUser(c.Request.Context(), params.UpdateUserParams{
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
func (h *UserHandler) GetUserByID(c *gin.Context) {
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
