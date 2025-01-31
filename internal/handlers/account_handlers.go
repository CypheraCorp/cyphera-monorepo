package handlers

import (
	"cyphera-api/internal/constants"
	"cyphera-api/internal/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Domain-specific handlers
type AccountHandler struct {
	common *CommonServices
}

func NewAccountHandler(common *CommonServices) *AccountHandler {
	return &AccountHandler{common: common}
}

// AccountResponse represents the standardized API response for account operations
type AccountResponse struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	Name               string                 `json:"name"`
	AccountType        string                 `json:"account_type"`
	BusinessName       string                 `json:"business_name,omitempty"`
	BusinessType       string                 `json:"business_type,omitempty"`
	WebsiteURL         string                 `json:"website_url,omitempty"`
	SupportEmail       string                 `json:"support_email,omitempty"`
	SupportPhone       string                 `json:"support_phone,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	Created            int64                  `json:"created"`
	Updated            int64                  `json:"updated"`
}

// CreateAccountRequest represents the request body for creating an account
type CreateAccountRequest struct {
	Name               string                 `json:"name" binding:"required"`
	AccountType        string                 `json:"account_type" binding:"required,oneof=admin merchant"`
	Description        string                 `json:"description,omitempty"`
	BusinessName       string                 `json:"business_name,omitempty"`
	BusinessType       string                 `json:"business_type,omitempty"`
	WebsiteURL         string                 `json:"website_url,omitempty"`
	SupportEmail       string                 `json:"support_email,omitempty"`
	SupportPhone       string                 `json:"support_phone,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateAccountRequest represents the request body for updating an account
type UpdateAccountRequest struct {
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description,omitempty"`
	BusinessName       string                 `json:"business_name,omitempty"`
	BusinessType       string                 `json:"business_type,omitempty"`
	WebsiteURL         string                 `json:"website_url,omitempty"`
	SupportEmail       string                 `json:"support_email,omitempty"`
	SupportPhone       string                 `json:"support_phone,omitempty"`
	AccountType        string                 `json:"account_type,omitempty" binding:"omitempty,oneof=admin merchant"`
	FinishedOnboarding bool                   `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// InitializeAccountResponse represents the response body for initializing an account
type InitializeAccountResponse struct {
	AccountResponse AccountResponse   `json:"account"`
	User            UserResponse      `json:"user"`
	Workspace       WorkspaceResponse `json:"workspace"`
}

// ListAccounts godoc
// @Summary List accounts
// @Description Returns a list of accounts. Only accessible by admins.
// @Tags accounts
// @Accept json
// @Produce json
// @Success 200 {object} AccountResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts [get]
func (h *AccountHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.common.db.ListAccounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve accounts"})
		return
	}

	response := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		response[i] = toAccountResponse(account)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// GetAccount godoc
// @Summary Get an account
// @Description Retrieves the details of an existing account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/{id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid account ID format"})
		return
	}

	account, err := h.common.db.GetAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Account not found"})
		return
	}

	c.JSON(http.StatusOK, toAccountResponse(account))
}

// GetCurrentAccount godoc
// @Summary Get current account
// @Description Retrieves the details of the currently authenticated user's account
// @Tags accounts
// @Accept json
// @Produce json
// @Success 200 {object} AccountResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/me [get]
func (h *AccountHandler) GetCurrentAccount(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Not authenticated"})
		return
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	// Get user's accounts
	accounts, err := h.common.db.ListAccountsByUser(c.Request.Context(), parsedUserID)
	if err != nil || len(accounts) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "No account found"})
		return
	}

	// Find the account where the user is the owner
	for _, acc := range accounts {
		if acc.IsOwner.Bool {
			c.JSON(http.StatusOK, toAccountResponseFromUserAccount(acc))
			return
		}
	}

	// If no owned account found, return the first associated account
	c.JSON(http.StatusOK, toAccountResponseFromUserAccount(accounts[0]))
}

// CreateAccount godoc
// @Summary Create an account
// @Description Creates a new account object. Only accessible by admins.
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Account creation data"
// @Success 200 {object} CreateAccountRequest
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	account, err := h.common.db.CreateAccount(c.Request.Context(), db.CreateAccountParams{
		Name:               req.Name,
		AccountType:        db.AccountType(req.AccountType),
		BusinessName:       pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
		BusinessType:       pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
		WebsiteUrl:         pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
		SupportEmail:       pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
		SupportPhone:       pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
		FinishedOnboarding: pgtype.Bool{Bool: req.FinishedOnboarding, Valid: true},
		Metadata:           metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create account"})
		return
	}

	accountResponse := toAccountResponse(account)

	c.JSON(http.StatusOK, accountResponse)
}

// CreateAccount godoc
// @Summary Create an account
// @Description Creates a new account object. Only accessible by admins.
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Account creation data"
// @Success 200 {object} InitializeAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/initialize [post]
func (h *AccountHandler) InitializeAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	metaDataMap := make(map[string]interface{})
	err = json.Unmarshal(metadata, &metaDataMap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to unmarshal metadata"})
		return
	}

	ownerAuth0Id := metaDataMap["ownerAuth0Id"].(string)
	if ownerAuth0Id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Owner Auth0 ID is required"})
		return
	}

	email := metaDataMap["email"].(string)
	if email == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Email is required"})
		return
	}

	// Check if user already exists
	user, err := h.common.db.GetUserByAuth0ID(c.Request.Context(), ownerAuth0Id)
	if err != nil {
		account, err := h.common.db.CreateAccount(c.Request.Context(), db.CreateAccountParams{
			Name:               req.Name,
			AccountType:        db.AccountType(req.AccountType),
			BusinessName:       pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
			BusinessType:       pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
			WebsiteUrl:         pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
			SupportEmail:       pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
			SupportPhone:       pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
			FinishedOnboarding: pgtype.Bool{Bool: req.FinishedOnboarding, Valid: true},
			Metadata:           metadata,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create account"})
			return
		}

		// Create user account
		user, err = h.common.db.CreateUser(c.Request.Context(), db.CreateUserParams{
			Auth0ID: ownerAuth0Id,
			Email:   email,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
			return
		}

		workspace, err := h.common.db.CreateWorkspace(c.Request.Context(), db.CreateWorkspaceParams{
			Name:      strings.ToLower(fmt.Sprintf("%s's Workspace", strings.ReplaceAll(account.Name, " ", "_"))),
			AccountID: account.ID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create workspace"})
			return
		}

		// add user to account
		_, err = h.common.db.AddUserToAccount(c.Request.Context(), db.AddUserToAccountParams{
			UserID:    user.ID,
			AccountID: account.ID,
			Role:      db.UserRoleAdmin,
			IsOwner:   pgtype.Bool{Bool: true, Valid: true},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to add user to account"})
			return
		}

		accountResponse := toAccountResponse(account)
		userResponse := toUserResponse(user)
		workspaceResponse := toWorkspaceResponse(workspace)

		accountResponseWithUser := InitializeAccountResponse{
			AccountResponse: accountResponse,
			User:            userResponse,
			Workspace:       workspaceResponse,
		}

		c.JSON(http.StatusOK, accountResponseWithUser)
	}

	// user and account is already created so we can return the data that is there.
	accounts, err := h.common.db.GetUserAssociatedAccounts(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve account"})
		return
	}

	account, err := h.common.db.GetAccountByID(c.Request.Context(), accounts[0].ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve account"})
		return
	}

	// get workspace by account id
	workspaces, err := h.common.db.ListWorkspacesByAccountID(c.Request.Context(), account.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve workspace"})
		return
	}

	workspace := workspaces[0]

	accountResponse := toAccountResponse(account)
	userResponse := toUserResponse(user)
	workspaceResponse := toWorkspaceResponse(workspace)

	accountResponseWithUser := InitializeAccountResponse{
		AccountResponse: accountResponse,
		User:            userResponse,
		Workspace:       workspaceResponse,
	}

	c.JSON(http.StatusOK, accountResponseWithUser)
}

// UpdateAccount godoc
// @Summary Update an account
// @Description Updates the specified account by setting the values of the parameters passed
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param account body UpdateAccountRequest true "Account update data"
// @Success 200 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/{id} [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid account ID format"})
		return
	}

	// Check if user has access to this account
	userID := c.GetString("userID")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user ID format"})
		return
	}

	accounts, err := h.common.db.ListAccountsByUser(c.Request.Context(), parsedUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to verify account access"})
		return
	}

	var hasAccess bool
	for _, acc := range accounts {
		if acc.ID == parsedUUID {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "You don't have access to this account"})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	account, err := h.common.db.UpdateAccount(c.Request.Context(), db.UpdateAccountParams{
		ID:                 parsedUUID,
		Name:               req.Name,
		AccountType:        db.AccountType(req.AccountType),
		BusinessName:       pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
		BusinessType:       pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
		WebsiteUrl:         pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
		SupportEmail:       pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
		SupportPhone:       pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
		FinishedOnboarding: pgtype.Bool{Bool: req.FinishedOnboarding, Valid: true},
		Metadata:           metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update account"})
		return
	}

	c.JSON(http.StatusOK, toAccountResponse(account))
}

// DeleteAccount godoc
// @Summary Delete an account
// @Description Deletes an account. Only accessible by admins.
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/accounts/{id} [delete]
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	// Only admins can delete accounts
	if c.GetString("accountType") != constants.AccountTypeAdmin {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only admin accounts can delete accounts"})
		return
	}

	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid account ID format"})
		return
	}

	err = h.common.db.DeleteAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Account not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper function to convert database model to API response
func toAccountResponse(a db.Account) AccountResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(a.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling account metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return AccountResponse{
		ID:                 a.ID.String(),
		Object:             "account",
		Name:               a.Name,
		AccountType:        string(a.AccountType),
		BusinessName:       a.BusinessName.String,
		BusinessType:       a.BusinessType.String,
		WebsiteURL:         a.WebsiteUrl.String,
		SupportEmail:       a.SupportEmail.String,
		SupportPhone:       a.SupportPhone.String,
		Metadata:           metadata,
		FinishedOnboarding: a.FinishedOnboarding.Bool,
		Created:            a.CreatedAt.Time.Unix(),
		Updated:            a.UpdatedAt.Time.Unix(),
	}
}

// Helper function to convert ListAccountsByUserRow to API response
func toAccountResponseFromUserAccount(a db.ListAccountsByUserRow) AccountResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(a.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling account metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return AccountResponse{
		ID:                 a.ID.String(),
		Object:             "account",
		Name:               a.Name,
		AccountType:        string(a.AccountType),
		BusinessName:       a.BusinessName.String,
		BusinessType:       a.BusinessType.String,
		WebsiteURL:         a.WebsiteUrl.String,
		SupportEmail:       a.SupportEmail.String,
		SupportPhone:       a.SupportPhone.String,
		Metadata:           metadata,
		FinishedOnboarding: a.FinishedOnboarding.Bool,
		Created:            a.CreatedAt.Time.Unix(),
		Updated:            a.UpdatedAt.Time.Unix(),
	}
}
