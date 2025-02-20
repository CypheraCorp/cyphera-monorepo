package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pkg/errors"
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
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
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

// OnboardAccountRequest represents the request body for onboarding an account
type OnboardAccountRequest struct {
	AddressLine1  string `json:"address_line_1"`
	AddressLine2  string `json:"address_line_2"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	WalletAddress string `json:"wallet_address"`
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
		sendError(c, http.StatusInternalServerError, "Failed to retrieve accounts", err)
		return
	}

	response := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		response[i] = toAccountResponse(account)
	}

	sendList(c, response)
}

// GetAccount godoc
// @Summary Get account by ID
// @Description Get account details by account ID
// @Tags accounts
// @Accept  json
// @Produce  json
// @Param account_id path string true "Account ID"
// @Success 200 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /accounts/{account_id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountId := c.Param("account_id")
	parsedUUID, err := uuid.Parse(accountId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID format", err)
		return
	}

	account, err := h.common.db.GetAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Account not found")
		return
	}

	sendSuccess(c, http.StatusOK, toAccountResponse(account))
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
	access, err := h.getAccountDetails(c)
	if err != nil {
		handleDBError(c, err, "Failed to retrieve account details")
		return
	}

	sendSuccess(c, http.StatusOK, toFullAccountResponse(access))
}

// GetAccountDetails retrieves and validates account, user, and workspace information from context
func (h *AccountHandler) getAccountDetails(c *gin.Context) (*AccountAccessResponse, error) {
	// Get and parse account ID from context
	accountID := c.GetString("accountID")
	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid account ID format")
	}

	// Get the account
	account, err := h.common.db.GetAccount(c.Request.Context(), parsedAccountID)
	if err != nil {
		return nil, errors.Wrap(err, "account not found")
	}

	// Get and parse user ID from context
	userID := c.GetString("userID")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid user ID format")
	}

	// Get the user
	user, err := h.common.db.GetUserByID(c.Request.Context(), parsedUserID)
	if err != nil {
		return nil, errors.Wrap(err, "user not found")
	}
	// get workspace by account id
	workspaces, err := h.common.db.ListWorkspacesByAccountID(c.Request.Context(), account.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve workspace")
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
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Start with base params containing only the ID
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

	// For boolean fields, we need to check if they were explicitly set in the request
	params.FinishedOnboarding = pgtype.Bool{Bool: req.FinishedOnboarding, Valid: true}

	// Only update metadata if it's provided
	if req.Metadata != nil {
		metadata, err := json.Marshal(req.Metadata)
		if err != nil {
			sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
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
		sendError(c, http.StatusInternalServerError, "Failed to update account", err)
		return
	}

	fullAccountResponse, err := h.getAccountDetails(c)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve account details", err)
		return
	}

	sendSuccess(c, http.StatusOK, toFullAccountResponse(fullAccountResponse))
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
	accountType := c.GetString("accountType")
	if accountType != "admin" {
		sendError(c, http.StatusForbidden, "Only admin accounts can create accounts", nil)
		return
	}

	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
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
		sendError(c, http.StatusInternalServerError, "Failed to create account", err)
		return
	}

	// Create default workspace for the account
	_, err = h.common.db.CreateWorkspace(c.Request.Context(), db.CreateWorkspaceParams{
		AccountID: account.ID,
		Name:      "Default",
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create workspace", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toAccountResponse(account))
}

// validateSignInRequest validates the sign in request and extracts required metadata
func (h *AccountHandler) validateSignInRequest(req CreateAccountRequest) (string, string, []byte, error) {
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return "", "", nil, errors.Wrap(err, "invalid metadata format")
	}

	metaDataMap := make(map[string]interface{})
	err = json.Unmarshal(metadata, &metaDataMap)
	if err != nil {
		return "", "", nil, errors.Wrap(err, "failed to unmarshal metadata")
	}

	// Check for Supabase metadata format
	if supabaseId, ok := metaDataMap["ownerSupabaseId"].(string); ok {
		email, ok := metaDataMap["email"].(string)
		if !ok || email == "" {
			return "", "", nil, errors.New("email is required")
		}
		return supabaseId, email, metadata, nil
	}

	return "", "", nil, errors.New("ownerSupabaseId is required")
}

// createNewAccountWithUser creates a new account with associated user and workspace
func (h *AccountHandler) createNewAccountWithUser(ctx *gin.Context, req CreateAccountRequest, supabaseId string, email string, metadata []byte) (*FullAccountResponse, error) {
	// Create account
	account, err := h.common.db.CreateAccount(ctx.Request.Context(), db.CreateAccountParams{
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
		return nil, errors.Wrap(err, "failed to create account")
	}

	// Create user
	user, err := h.common.db.CreateUser(ctx.Request.Context(), db.CreateUserParams{
		SupabaseID:     supabaseId,
		Email:          email,
		AccountID:      account.ID,
		Role:           db.UserRoleAdmin,
		IsAccountOwner: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to createNewAccountWithUser")
	}

	// Create workspace
	workspace, err := h.common.db.CreateWorkspace(ctx.Request.Context(), db.CreateWorkspaceParams{
		Name:      "Default",
		AccountID: account.ID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace")
	}

	return &FullAccountResponse{
		AccountResponse: toAccountResponse(account),
		User:            toUserResponse(user),
		Workspaces:      []WorkspaceResponse{toWorkspaceResponse(workspace)},
	}, nil
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
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	supabaseId, email, metadata, err := h.validateSignInRequest(req)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	// Check if user already exists by Supabase ID
	user, err := h.common.db.GetUserBySupabaseID(c.Request.Context(), supabaseId)
	if err != nil {
		if err != pgx.ErrNoRows {
			sendError(c, http.StatusInternalServerError, "Failed to check existing user", err)
			return
		}
	}

	var response *FullAccountResponse
	if errors.Is(err, pgx.ErrNoRows) {
		// User doesn't exist, create new account and user
		response, err = h.createNewAccountWithUser(c, req, supabaseId, email, metadata)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
		sendSuccess(c, http.StatusCreated, response)
	} else {
		// User exists, get existing account details
		account, err := h.common.db.GetAccount(c.Request.Context(), user.AccountID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
		workspaces, err := h.common.db.ListWorkspacesByAccountID(c.Request.Context(), account.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		sendSuccess(c, http.StatusOK, toFullAccountResponse(&AccountAccessResponse{
			Account:   account,
			User:      user,
			Workspace: workspaces,
		}))
	}
}

// OnboardAccount godoc
// @Summary Onboard an account
// @Description Onboards an account by setting the finished_onboarding flag to true
// @Tags accounts
// @Accept json
// @Produce json
// @Param account_id path string true "Account ID"
// @Success 200 {object} AccountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /accounts/onboard [post]
func (h *AccountHandler) OnboardAccount(c *gin.Context) {
	// Check account access
	access, err := h.CheckAccountAccess(c)
	if HandleAccountAccessError(c, err) {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var req OnboardAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Start with base params containing only the ID
	accountParams := db.UpdateAccountParams{
		ID:                 access.Account.ID,
		Name:               access.Account.Name,
		AccountType:        access.Account.AccountType,
		BusinessName:       access.Account.BusinessName,
		BusinessType:       access.Account.BusinessType,
		WebsiteUrl:         access.Account.WebsiteUrl,
		SupportEmail:       access.Account.SupportEmail,
		SupportPhone:       access.Account.SupportPhone,
		FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
		Metadata:           access.Account.Metadata,
		OwnerID:            pgtype.UUID{Bytes: access.User.ID, Valid: true},
	}

	_, err = h.common.db.UpdateAccount(c.Request.Context(), accountParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to onboard account", err)
		return
	}

	userParams := db.UpdateUserParams{
		ID:               access.User.ID,
		Email:            access.User.Email,
		FirstName:        pgtype.Text{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:         pgtype.Text{String: req.LastName, Valid: req.LastName != ""},
		AddressLine1:     pgtype.Text{String: req.AddressLine1, Valid: req.AddressLine1 != ""},
		AddressLine2:     pgtype.Text{String: req.AddressLine2, Valid: req.AddressLine2 != ""},
		City:             pgtype.Text{String: req.City, Valid: req.City != ""},
		StateRegion:      pgtype.Text{String: req.State, Valid: req.State != ""},
		PostalCode:       pgtype.Text{String: req.PostalCode, Valid: req.PostalCode != ""},
		Country:          pgtype.Text{String: req.Country, Valid: req.Country != ""},
		DisplayName:      access.User.DisplayName,
		PictureUrl:       access.User.PictureUrl,
		Phone:            access.User.Phone,
		Timezone:         access.User.Timezone,
		Locale:           access.User.Locale,
		EmailVerified:    pgtype.Bool{Bool: true, Valid: true},
		TwoFactorEnabled: access.User.TwoFactorEnabled,
		Status:           access.User.Status,
	}

	_, err = h.common.db.UpdateUser(c.Request.Context(), userParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to onboard account", err)
		return
	}

	sendSuccess(c, http.StatusOK, gin.H{"message": "Account onboarded successfully"})
}

// HandleAccountAccessError is a helper function to handle account access errors consistently
func HandleAccountAccessError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch err.Error() {
	case "invalid account ID format", "invalid user ID format":
		sendError(c, http.StatusBadRequest, err.Error(), err)
	case "account not found", "user not found":
		sendError(c, http.StatusNotFound, err.Error(), err)
	case "user does not have access to this account":
		sendError(c, http.StatusForbidden, err.Error(), err)
	default:
		sendError(c, http.StatusInternalServerError, "Failed to verify account access", err)
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
// @Router /accounts/:account_id [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	accountId := c.Param("account_id")
	parsedUUID, err := uuid.Parse(accountId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID format", err)
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
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
		sendError(c, http.StatusInternalServerError, "Failed to update account", err)
		return
	}

	sendSuccess(c, http.StatusOK, toAccountResponse(account))
}

// DeleteAccount godoc
// @Summary Delete an account
// @Description Deletes an account. Only accessible by admins.
// @Tags accounts
// @Accept json
// @Produce json
// @Param account_id path string true "Account ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /admin/accounts/{account_id} [delete]
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	accountType := c.GetString("accountType")
	if accountType != "admin" {
		sendError(c, http.StatusForbidden, "Only admin accounts can delete accounts", nil)
		return
	}

	accountId := c.Param("account_id")
	parsedUUID, err := uuid.Parse(accountId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID format", err)
		return
	}

	err = h.common.db.DeleteAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Account not found")
		return
	}

	sendSuccessMessage(c, http.StatusNoContent, "Account successfully deleted")
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
		CreatedAt:          a.CreatedAt.Time.Unix(),
		UpdatedAt:          a.UpdatedAt.Time.Unix(),
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
		return nil, errors.Wrap(err, "failed to get account details")
	}

	// Check if user has access to this account (user.AccountID should match the account.ID)
	if accountDetails.User.AccountID != accountDetails.Account.ID {
		return nil, errors.New("user does not have access to this account")
	}

	return accountDetails, nil
}
