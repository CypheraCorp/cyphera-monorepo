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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Domain-specific handlers
type AccountHandler struct {
	common *CommonServices
}

func NewAccountHandler(common *CommonServices) *AccountHandler {
	return &AccountHandler{common: common}
}

type FullAccountResponse struct {
	AccountResponse AccountResponse     `json:"account"`
	User            UserResponse        `json:"user"`
	Workspaces      []WorkspaceResponse `json:"workspaces"`
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

// AccountAccessResponse represents the response from checking account access
type AccountAccessResponse struct {
	User      db.User
	Account   db.Account
	Workspace []db.Workspace
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

// GetCurrentAccountDetails godoc
// @Summary Get current account
// @Description Retrieves the details of the currently authenticated user's account
// @Tags accounts
// @Accept json
// @Produce json
// @Success 200 {object} FullAccountResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/me/details [get]
func (h *AccountHandler) GetCurrentAccountDetails(c *gin.Context) {
	// Check account access
	access, err := h.CheckAccountAccess(c)
	if HandleAccountAccessError(c, err) {
		return
	}

	// If no owned account found, return the first associated account
	c.JSON(http.StatusOK, toFullAccountResponse(access))
}

// GetAccountDetails retrieves and validates account, user, and workspace information from context
func (h *AccountHandler) getAccountDetails(c *gin.Context) (*AccountAccessResponse, error) {
	// Get and parse account ID from context
	accountID := c.GetString("accountID")
	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID format")
	}

	// Get the account
	account, err := h.common.db.GetAccount(c.Request.Context(), parsedAccountID)
	if err != nil {
		return nil, fmt.Errorf("account not found")
	}

	// Get and parse user ID from context
	userID := c.GetString("userID")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format")
	}

	// Get the user
	user, err := h.common.db.GetUserByID(c.Request.Context(), parsedUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// get workspace by account id
	workspaces, err := h.common.db.ListWorkspacesByAccountID(c.Request.Context(), account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workspace")
	}

	return &AccountAccessResponse{
		User:      user,
		Account:   account,
		Workspace: workspaces,
	}, nil
}

// UpdateCurrentAccount godoc
// @Summary Update current account
// @Description Updates the currently authenticated user's account details
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body UpdateAccountRequest true "Account update data"
// @Success 200 {object} FullAccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/me [put]
func (h *AccountHandler) UpdateCurrentAccount(c *gin.Context) {
	// Check account access
	access, err := h.CheckAccountAccess(c)
	if HandleAccountAccessError(c, err) {
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Prepare update parameters starting with existing values
	params := db.UpdateAccountParams{
		ID:                 access.Account.ID,
		Name:               access.Account.Name,
		AccountType:        access.Account.AccountType,
		BusinessName:       access.Account.BusinessName,
		BusinessType:       access.Account.BusinessType,
		WebsiteUrl:         access.Account.WebsiteUrl,
		SupportEmail:       access.Account.SupportEmail,
		SupportPhone:       access.Account.SupportPhone,
		FinishedOnboarding: access.Account.FinishedOnboarding,
		Metadata:           access.Account.Metadata,
	}

	// Only update fields that are provided in the request
	if req.Name != "" {
		params.Name = req.Name
	}
	if req.AccountType != "" {
		params.AccountType = db.AccountType(req.AccountType)
	}
	if req.BusinessName != "" {
		params.BusinessName = pgtype.Text{String: req.BusinessName, Valid: true}
	}
	if req.BusinessType != "" {
		params.BusinessType = pgtype.Text{String: req.BusinessType, Valid: true}
	}
	if req.WebsiteURL != "" {
		params.WebsiteUrl = pgtype.Text{String: req.WebsiteURL, Valid: true}
	}
	if req.SupportEmail != "" {
		params.SupportEmail = pgtype.Text{String: req.SupportEmail, Valid: true}
	}
	if req.SupportPhone != "" {
		params.SupportPhone = pgtype.Text{String: req.SupportPhone, Valid: true}
	}

	// Handle metadata separately - only update if provided
	if req.Metadata != nil {
		metadata, err := json.Marshal(req.Metadata)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
			return
		}
		params.Metadata = metadata
	}

	// Handle finished_onboarding separately since it's a boolean
	if !access.Account.FinishedOnboarding.Bool {
		params.FinishedOnboarding = pgtype.Bool{Bool: true, Valid: true}
	}

	_, err = h.common.db.UpdateAccount(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update account"})
		return
	}

	fullAccountResponse, err := h.getAccountDetails(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve account details"})
		return
	}

	c.JSON(http.StatusOK, toFullAccountResponse(fullAccountResponse))
}

// CreateAccount godoc
// @Summary Create an account
// @Description Creates a new account with a default workspace. Only accessible by admins.
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Account creation data"
// @Success 201 {object} AccountResponse
// @Failure 400 {object} ErrorResponse "Invalid request body or metadata format"
// @Failure 401 {object} ErrorResponse "Not authenticated"
// @Failure 403 {object} ErrorResponse "Not authorized"
// @Failure 500 {object} ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	// Only admins can create accounts
	if c.GetString("accountType") != constants.AccountTypeAdmin {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Only admin accounts can create accounts"})
		return
	}

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

	// Create default workspace for the account
	_, err = h.common.db.CreateWorkspace(c.Request.Context(), db.CreateWorkspaceParams{
		Name:      "my_workspace",
		AccountID: account.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create workspace"})
		return
	}

	c.JSON(http.StatusCreated, toAccountResponse(account))
}

// SignInAccount godoc
// @Summary Register or sign in to an account
// @Description Creates a new account with user and workspace, or returns existing account details
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body CreateAccountRequest true "Account creation data"
// @Success 200 {object} FullAccountResponse "Existing account details"
// @Success 201 {object} FullAccountResponse "Newly created account"
// @Failure 400 {object} ErrorResponse "Invalid request body, metadata format, or missing required fields"
// @Failure 500 {object} ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /accounts/signin [post]
func (h *AccountHandler) SignInAccount(c *gin.Context) {
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

	ownerAuth0Id, ok := metaDataMap["ownerAuth0Id"].(string)
	if !ok || ownerAuth0Id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Owner Auth0 ID is required"})
		return
	}

	email, ok := metaDataMap["email"].(string)
	if !ok || email == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Email is required"})
		return
	}

	// Check if user already exists
	user, err := h.common.db.GetUserByAuth0ID(c.Request.Context(), ownerAuth0Id)
	if err != nil {
		if err != pgx.ErrNoRows {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to check existing user"})
			return
		}

		// User doesn't exist, create new account and user
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

		// Create user with account association
		user, err = h.common.db.CreateUser(c.Request.Context(), db.CreateUserParams{
			Auth0ID:        ownerAuth0Id,
			Email:          email,
			AccountID:      account.ID,
			Role:           db.UserRoleAdmin,
			IsAccountOwner: pgtype.Bool{Bool: true, Valid: true},
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

		fullAccountResponse := FullAccountResponse{
			AccountResponse: toAccountResponse(account),
			User:            toUserResponse(user),
			Workspaces:      []WorkspaceResponse{toWorkspaceResponse(workspace)},
		}

		c.JSON(http.StatusCreated, fullAccountResponse)
		return
	}

	// User exists, get their account
	account, err := h.common.db.GetAccountByID(c.Request.Context(), user.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve account"})
		return
	}

	// Get workspaces for the account
	workspaces, err := h.common.db.ListWorkspacesByAccountID(c.Request.Context(), account.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve workspaces"})
		return
	}

	workspaceResponses := make([]WorkspaceResponse, len(workspaces))
	for i, workspace := range workspaces {
		workspaceResponses[i] = toWorkspaceResponse(workspace)
	}

	getAccountResponse := FullAccountResponse{
		AccountResponse: toAccountResponse(account),
		User:            toUserResponse(user),
		Workspaces:      workspaceResponses,
	}

	c.JSON(http.StatusOK, getAccountResponse)
}

// HandleAccountAccessError is a helper function to handle account access errors consistently
func HandleAccountAccessError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch err.Error() {
	case "invalid account ID format", "invalid user ID format":
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	case "account not found", "user not found":
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
	case "user does not have access to this account":
		c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to verify account access"})
	}
	return true
}

// UpdateAccount godoc
// @Summary Update an account
// @Description Updates the specified account by setting the values of the parameters passed
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body UpdateAccountRequest true "Account update data"
// @Success 200 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/:id [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid account ID format"})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Prepare update parameters
	params := db.UpdateAccountParams{
		ID:                 parsedUUID,
		Name:               req.Name,
		AccountType:        db.AccountType(req.AccountType),
		BusinessName:       pgtype.Text{String: req.BusinessName, Valid: req.BusinessName != ""},
		BusinessType:       pgtype.Text{String: req.BusinessType, Valid: req.BusinessType != ""},
		WebsiteUrl:         pgtype.Text{String: req.WebsiteURL, Valid: req.WebsiteURL != ""},
		SupportEmail:       pgtype.Text{String: req.SupportEmail, Valid: req.SupportEmail != ""},
		SupportPhone:       pgtype.Text{String: req.SupportPhone, Valid: req.SupportPhone != ""},
		FinishedOnboarding: pgtype.Bool{Bool: req.FinishedOnboarding, Valid: true},
	}

	// Handle metadata separately
	if req.Metadata != nil {
		metadata, err := json.Marshal(req.Metadata)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
			return
		}
		params.Metadata = metadata
	}

	account, err := h.common.db.UpdateAccount(c.Request.Context(), params)
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

// Helper function to convert AccountAccessResponse to FullAccountResponse
func toFullAccountResponse(acc *AccountAccessResponse) FullAccountResponse {
	// Convert account data
	accountResponse := toAccountResponse(acc.Account)

	// Convert user data
	userResponse := toUserResponse(acc.User)

	// Convert workspace data
	workspaceResponses := make([]WorkspaceResponse, len(acc.Workspace))
	for i, workspace := range acc.Workspace {
		workspaceResponses[i] = toWorkspaceResponse(workspace)
	}

	return FullAccountResponse{
		AccountResponse: accountResponse,
		User:            userResponse,
		Workspaces:      workspaceResponses,
	}
}

// CheckAccountAccess verifies if a user has access to an account and returns both objects if they do
func (h *AccountHandler) CheckAccountAccess(c *gin.Context) (*AccountAccessResponse, error) {
	accountDetails, err := h.getAccountDetails(c)
	if err != nil {
		return nil, err
	}

	// Check if user has access to this account (user.AccountID should match the account.ID)
	if accountDetails.User.AccountID != accountDetails.Account.ID {
		return nil, fmt.Errorf("user does not have access to this account")
	}

	return accountDetails, nil
}
