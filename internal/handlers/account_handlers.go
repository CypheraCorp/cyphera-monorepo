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

type AccountDetailsResponse struct {
	AccountResponse AccountResponse  `json:"account"`
	User            UserResponse     `json:"user,omitempty"`
	Wallets         []WalletResponse `json:"wallets,omitempty"`
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
	Workspaces         []WorkspaceResponse    `json:"workspaces,omitempty"`
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
	// Web3Auth embedded wallet data to be created during registration
	WalletData *EmbeddedWalletRequest `json:"wallet_data,omitempty"`
}

// EmbeddedWalletRequest represents the request body for creating an embedded wallet during account creation
// This is different from CreateWalletRequest as it doesn't require network_id (we create for all active networks)
type EmbeddedWalletRequest struct {
	WalletType    string                 `json:"wallet_type" binding:"required"` // 'web3auth', 'wallet', or 'circle_wallet'
	WalletAddress string                 `json:"wallet_address" binding:"required"`
	NetworkType   string                 `json:"network_type" binding:"required"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	// Circle wallet specific fields
	CircleUserID   string `json:"circle_user_id,omitempty"`   // Only for circle wallets
	CircleWalletID string `json:"circle_wallet_id,omitempty"` // Only for circle wallets
	ChainID        int32  `json:"chain_id,omitempty"`         // Only for circle wallets
	State          string `json:"state,omitempty"`            // Only for circle wallets
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
	AddressLine1  string `json:"address_line1"` // Frontend sends address_line1 (no underscore)
	AddressLine2  string `json:"address_line2"` // Frontend sends address_line2 (no underscore)
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	WalletAddress string `json:"wallet_address"`
}

func (h *AccountHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.common.db.ListAccounts(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve accounts", err)
		return
	}

	response := make([]AccountResponse, len(accounts))
	for i, account := range accounts {
		response[i] = toAccountResponse(account, []db.Workspace{})
	}

	sendList(c, response)
}

// GetAccount godoc
// @Summary Get account
// @Description Retrieves the details of the user's account
// @Tags accounts
// @Accept json
// @Produce json
// @Success 200 {object} AccountDetailsResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /accounts/{account_id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	accountID := c.Param("account_id")
	if accountID == "" {
		sendError(c, http.StatusBadRequest, "Account ID is required", nil)
		return
	}

	// Get and parse account ID from context
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Get the workspace
	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusNotFound, "Workspace not found", err)
		return
	}

	if workspace.AccountID.String() != accountID {
		sendError(c, http.StatusForbidden, "You are not authorized to access this account", nil)
		return
	}

	// Get the account
	account, err := h.common.db.GetAccount(c.Request.Context(), workspace.AccountID)
	if err != nil {
		sendError(c, http.StatusNotFound, "Account not found", err)
		return
	}

	// Get Workspaces Responses
	workspaces, err := h.common.db.ListWorkspacesByAccountID(c.Request.Context(), account.ID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve workspaces", err)
		return
	}

	response := toAccountResponse(account, workspaces)

	sendSuccess(c, http.StatusOK, response)
}

//func (h *AccountHandler) UpdateAccount(c *gin.Context) {
// Check account access
// TODO: Update because this is an admin only function
// access, err := h.GetAccount(c)
// if HandleAccountAccessError(c, err) {
// 	sendError(c, http.StatusBadRequest, "Invalid request body", err)
// 	return
// }

// var req UpdateAccountRequest
// if err := c.ShouldBindJSON(&req); err != nil {
// 	sendError(c, http.StatusBadRequest, "Invalid request body", err)
// 	return
// }

// // Start with base params containing only the ID
// params := db.UpdateAccountParams{
// 	ID:                 access.Account.ID,
// 	Name:               access.Account.Name,
// 	AccountType:        access.Account.AccountType,
// 	BusinessName:       access.Account.BusinessName,
// 	BusinessType:       access.Account.BusinessType,
// 	WebsiteUrl:         access.Account.WebsiteUrl,
// 	SupportEmail:       access.Account.SupportEmail,
// 	SupportPhone:       access.Account.SupportPhone,
// 	FinishedOnboarding: access.Account.FinishedOnboarding,
// 	Metadata:           access.Account.Metadata,
// }

// // Only update fields that are provided in the request
// if req.Name != "" {
// 	params.Name = req.Name
// }
// if req.AccountType != "" {
// 	params.AccountType = db.AccountType(req.AccountType)
// }
// if req.BusinessName != "" {
// 	params.BusinessName = pgtype.Text{String: req.BusinessName, Valid: true}
// }
// if req.BusinessType != "" {
// 	params.BusinessType = pgtype.Text{String: req.BusinessType, Valid: true}
// }
// if req.WebsiteURL != "" {
// 	params.WebsiteUrl = pgtype.Text{String: req.WebsiteURL, Valid: true}
// }
// if req.SupportEmail != "" {
// 	params.SupportEmail = pgtype.Text{String: req.SupportEmail, Valid: true}
// }
// if req.SupportPhone != "" {
// 	params.SupportPhone = pgtype.Text{String: req.SupportPhone, Valid: true}
// }

// // For boolean fields, we need to check if they were explicitly set in the request
// params.FinishedOnboarding = pgtype.Bool{Bool: req.FinishedOnboarding, Valid: true}

// // Only update metadata if it's provided
// if req.Metadata != nil {
// 	metadata, err := json.Marshal(req.Metadata)
// 	if err != nil {
// 		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
// 		return
// 	}
// 	params.Metadata = metadata
// }

// // Handle finished_onboarding separately since it's a boolean
// if !access.Account.FinishedOnboarding.Bool {
// 	params.FinishedOnboarding = pgtype.Bool{Bool: true, Valid: true}
// }

// _, err = h.common.db.UpdateAccount(c.Request.Context(), params)
// if err != nil {
// 	sendError(c, http.StatusInternalServerError, "Failed to update account", err)
// 	return
// }

// TODO: Update because this is an admin only function

// fullAccountResponse, err := h.GetAccount(c)
// if err != nil {
// 	sendError(c, http.StatusInternalServerError, "Failed to retrieve account details", err)
// 	return
// }

// sendSuccess(c, http.StatusOK, toFullAccountResponse(fullAccountResponse))
// }

// CreateAccount godoc
// @Summary Create a new account
// @Description Creates a new account with the specified name and account type
// @Tags accounts
// @Accept json
// @Tags exclude
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

	sendSuccess(c, http.StatusCreated, toAccountResponse(account, []db.Workspace{}))
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

	// Check for Web3Auth metadata format
	if web3authId, ok := metaDataMap["ownerWeb3AuthId"].(string); ok {
		email, ok := metaDataMap["email"].(string)
		if !ok || email == "" {
			return "", "", nil, errors.New("email is required")
		}
		return web3authId, email, metadata, nil
	}

	return "", "", nil, errors.New("ownerWeb3AuthId is required")
}

// createWalletsForActiveNetworks creates wallet entries for all active networks
func (h *AccountHandler) createWalletsForActiveNetworks(ctx *gin.Context, workspaceID uuid.UUID, walletData *EmbeddedWalletRequest) error {
	// Get all active networks
	networks, err := h.common.db.ListNetworks(ctx.Request.Context(), db.ListNetworksParams{
		IsActive: pgtype.Bool{Bool: true, Valid: true},
	})
	if err != nil {
		return errors.Wrap(err, "failed to fetch active networks")
	}

	if len(networks) == 0 {
		return errors.New("no active networks found")
	}

	// Create wallet entry for each active network
	for i, network := range networks {
		// Prepare metadata for the wallet
		var walletMetadata []byte
		if walletData.Metadata != nil {
			walletMetadata, err = json.Marshal(walletData.Metadata)
			if err != nil {
				return errors.Wrap(err, "failed to marshal wallet metadata")
			}
		} else {
			walletMetadata = []byte("{}")
		}

		// Set the first wallet as primary, others as non-primary
		isPrimary := i == 0 && walletData.IsPrimary

		// Determine the nickname to use - default to "Cyphera Wallet" if none provided
		nickname := walletData.Nickname
		if nickname == "" {
			nickname = "Cyphera Wallet"
		}

		// Create wallet for this network
		_, err = h.common.db.CreateWallet(ctx.Request.Context(), db.CreateWalletParams{
			WorkspaceID:   workspaceID,
			WalletType:    walletData.WalletType,
			WalletAddress: walletData.WalletAddress,
			NetworkType:   network.NetworkType,
			NetworkID:     pgtype.UUID{Bytes: network.ID, Valid: true},
			Nickname:      pgtype.Text{String: nickname, Valid: true},
			Ens:           pgtype.Text{String: walletData.ENS, Valid: walletData.ENS != ""},
			IsPrimary:     pgtype.Bool{Bool: isPrimary, Valid: true},
			Verified:      pgtype.Bool{Bool: walletData.Verified, Valid: true},
			Metadata:      walletMetadata,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to create wallet for network %s (chain_id: %d)", network.Name, network.ChainID)
		}
	}

	return nil
}

func (h *AccountHandler) createNewAccountWithUser(ctx *gin.Context, req CreateAccountRequest, web3authId string, email string, metadata []byte) (*AccountDetailsResponse, error) {
	// Extract verifier and verifierId from metadata
	var metadataMap map[string]interface{}
	var verifier, verifierId string

	if err := json.Unmarshal(metadata, &metadataMap); err == nil {
		if v, ok := metadataMap["verifier"].(string); ok {
			verifier = v
		}
		if v, ok := metadataMap["verifierId"].(string); ok {
			verifierId = v
		}
	}

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
		Email:          email,
		AccountID:      account.ID,
		Role:           db.UserRoleAdmin,
		IsAccountOwner: pgtype.Bool{Bool: true, Valid: true},
		FirstName:      pgtype.Text{Valid: false},
		LastName:       pgtype.Text{Valid: false},
		AddressLine1:   pgtype.Text{Valid: false},
		AddressLine2:   pgtype.Text{Valid: false},
		City:           pgtype.Text{Valid: false},
		StateRegion:    pgtype.Text{Valid: false},
		PostalCode:     pgtype.Text{Valid: false},
		Country:        pgtype.Text{Valid: false},
		DisplayName:    pgtype.Text{Valid: false},
		PictureUrl:     pgtype.Text{Valid: false},
		Phone:          pgtype.Text{Valid: false},
		Timezone:       pgtype.Text{Valid: false},
		Locale:         pgtype.Text{Valid: false},
		EmailVerified:  pgtype.Bool{Bool: true, Valid: true}, // Assume Web3Auth emails are verified
		Metadata:       []byte("{}"),
		Web3authID:     pgtype.Text{String: web3authId, Valid: web3authId != ""},
		Verifier:       pgtype.Text{String: verifier, Valid: verifier != ""},
		VerifierID:     pgtype.Text{String: verifierId, Valid: verifierId != ""},
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

	// Create wallets for all active networks if wallet data is provided
	if req.WalletData != nil {
		err = h.createWalletsForActiveNetworks(ctx, workspace.ID, req.WalletData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create wallets for active networks")
		}
	}

	return &AccountDetailsResponse{
		AccountResponse: toAccountResponse(account, []db.Workspace{workspace}),
		User:            toUserResponse(user),
	}, nil
}

// SignInRegisterAccount godoc
// @Summary Sign in to an account
// @Description Signs in to an account with the specified email and password
// @Tags accounts
// @Accept json
// @Tags exclude
func (h *AccountHandler) SignInRegisterAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	web3authId, email, metadata, err := h.validateSignInRequest(req)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	// Check if user already exists by Web3Auth ID
	user, err := h.common.db.GetUserByWeb3AuthID(c.Request.Context(), pgtype.Text{String: web3authId, Valid: web3authId != ""})
	if err != nil {
		if err != pgx.ErrNoRows {
			sendError(c, http.StatusInternalServerError, "Failed to check existing user", err)
			return
		}
	}

	var response *AccountDetailsResponse
	if errors.Is(err, pgx.ErrNoRows) {
		// User doesn't exist, create new account and user
		response, err = h.createNewAccountWithUser(c, req, web3authId, email, metadata)
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

		sendSuccess(c, http.StatusOK, toAccountDetailsResponse(&AccountAccessResponse{
			Account:   account,
			User:      user,
			Workspace: workspaces,
		}))
	}
}

func toAccountDetailsResponse(acc *AccountAccessResponse) AccountDetailsResponse {
	return AccountDetailsResponse{
		AccountResponse: toAccountResponse(acc.Account, acc.Workspace),
		User:            toUserResponse(acc.User),
	}
}

// OnboardAccount godoc
// @Summary Onboard an account
// @Description Onboards an account with the specified user details
// @Tags accounts
// @Accept json
// @Tags exclude
func (h *AccountHandler) OnboardAccount(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	userId := c.GetHeader("X-User-ID")
	parsedUserID, err := uuid.Parse(userId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	user, err := h.common.db.GetUserByID(c.Request.Context(), parsedUserID)
	if err != nil {
		sendError(c, http.StatusNotFound, "User not found", err)
		return
	}

	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusNotFound, "Workspace not found", err)
		return
	}

	account, err := h.common.db.GetAccount(c.Request.Context(), workspace.AccountID)
	if err != nil {
		sendError(c, http.StatusNotFound, "Account not found", err)
		return
	}

	var req OnboardAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Start with base params containing only the ID
	accountParams := db.UpdateAccountParams{
		ID:                 account.ID,
		Name:               account.Name,
		AccountType:        account.AccountType,
		BusinessName:       account.BusinessName,
		BusinessType:       account.BusinessType,
		WebsiteUrl:         account.WebsiteUrl,
		SupportEmail:       account.SupportEmail,
		SupportPhone:       account.SupportPhone,
		FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true},
		Metadata:           account.Metadata,
		OwnerID:            pgtype.UUID{Bytes: user.ID, Valid: true},
	}

	_, err = h.common.db.UpdateAccount(c.Request.Context(), accountParams)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to onboard account", err)
		return
	}

	userParams := db.UpdateUserParams{
		ID:                 user.ID,
		Email:              user.Email,
		FirstName:          pgtype.Text{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:           pgtype.Text{String: req.LastName, Valid: req.LastName != ""},
		AddressLine1:       pgtype.Text{String: req.AddressLine1, Valid: req.AddressLine1 != ""},
		AddressLine2:       pgtype.Text{String: req.AddressLine2, Valid: req.AddressLine2 != ""},
		City:               pgtype.Text{String: req.City, Valid: req.City != ""},
		StateRegion:        pgtype.Text{String: req.State, Valid: req.State != ""},
		PostalCode:         pgtype.Text{String: req.PostalCode, Valid: req.PostalCode != ""},
		Country:            pgtype.Text{String: req.Country, Valid: req.Country != ""},
		DisplayName:        user.DisplayName,
		PictureUrl:         user.PictureUrl,
		Phone:              user.Phone,
		Timezone:           user.Timezone,
		Locale:             user.Locale,
		EmailVerified:      pgtype.Bool{Bool: true, Valid: true},
		TwoFactorEnabled:   user.TwoFactorEnabled,
		FinishedOnboarding: pgtype.Bool{Bool: true, Valid: true}, // Automatically set to true when onboarding
		Status:             user.Status,
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

// DeleteAccount godoc
// @Summary Delete an account
// @Description Deletes an account with the specified ID
// @Tags accounts
// @Accept json
// @Tags exclude
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
func toAccountResponse(a db.Account, workspaces []db.Workspace) AccountResponse {

	workspacesResponses := make([]WorkspaceResponse, 0)
	for _, workspace := range workspaces {
		workspacesResponses = append(workspacesResponses, toWorkspaceResponse(workspace))
	}

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
		Workspaces:         workspacesResponses,
	}
}
