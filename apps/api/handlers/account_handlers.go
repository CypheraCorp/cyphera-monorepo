package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/cyphera/cyphera-api/apps/api/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pkg/errors"
)

// AccountHandler handles account-related operations
type AccountHandler struct {
	common         *CommonServices
	accountService interfaces.AccountService
	walletService  interfaces.WalletService
}

// Use types from the centralized packages
type CreateAccountRequest = requests.CreateAccountRequest
type EmbeddedWalletRequest = requests.EmbeddedWalletRequest
type UpdateAccountRequest = requests.UpdateAccountRequest
type OnboardAccountRequest = requests.OnboardAccountRequest

type AccountResponse = responses.AccountResponse
type AccountDetailsResponse = responses.AccountDetailsResponse
type AccountAccessResponse = responses.AccountAccessResponse

// NewAccountHandler creates a handler with interface dependencies
func NewAccountHandler(
	common *CommonServices,
	accountService interfaces.AccountService,
	walletService interfaces.WalletService,
) *AccountHandler {
	return &AccountHandler{
		common:         common,
		accountService: accountService,
		walletService:  walletService,
	}
}

func (h *AccountHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.accountService.ListAccounts(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve accounts", err)
		return
	}

	response := make([]helpers.AccountResponse, len(accounts))
	for i, account := range accounts {
		response[i] = helpers.ToAccountResponse(account, []db.Workspace{})
	}

	sendList(c, response)
}

// GetAccount godoc
// @Summary Get account
// @Description Retrieves the details of the user's account
// @Tags accounts
// @Accept json
// @Produce json
// @Param account_id path string true "Account ID"
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

	parsedAccountID, err := uuid.Parse(accountID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid account ID format", err)
		return
	}

	// Get and parse workspace ID from context
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Validate account access through workspace
	err = h.accountService.ValidateAccountAccess(c.Request.Context(), parsedAccountID, parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusForbidden, err.Error(), err)
		return
	}

	// Get the account
	account, err := h.accountService.GetAccount(c.Request.Context(), parsedAccountID)
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

	response := helpers.ToAccountResponse(*account, workspaces)

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

	// Create account using service
	account, err := h.accountService.CreateAccount(c.Request.Context(), params.CreateAccountParams{
		Name:               req.Name,
		AccountType:        req.AccountType,
		BusinessName:       req.BusinessName,
		BusinessType:       req.BusinessType,
		WebsiteURL:         req.WebsiteURL,
		SupportEmail:       req.SupportEmail,
		SupportPhone:       req.SupportPhone,
		FinishedOnboarding: req.FinishedOnboarding,
		Metadata:           req.Metadata,
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

	sendSuccess(c, http.StatusCreated, helpers.ToAccountResponse(*account, []db.Workspace{}))
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

// Commented out: unused function
/*
func (h *AccountHandler) createNewAccountWithUser(ctx *gin.Context, req CreateAccountRequest, web3authId string, email string, metadata []byte) (*helpers.AccountDetailsResponse, error) {
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

	return &helpers.AccountDetailsResponse{
		AccountResponse: helpers.ToAccountResponse(account, []db.Workspace{workspace}),
		User:            helpers.ToUserResponse(user),
	}, nil
}
*/

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

	// Validate sign-in request
	web3authId, email, err := h.accountService.ValidateSignInRequest(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// Handle sign-in or registration
	result, err := h.accountService.SignInOrRegisterAccount(c.Request.Context(), params.CreateAccountParams{
		Name:               req.Name,
		AccountType:        req.AccountType,
		BusinessName:       req.BusinessName,
		BusinessType:       req.BusinessType,
		WebsiteURL:         req.WebsiteURL,
		SupportEmail:       req.SupportEmail,
		SupportPhone:       req.SupportPhone,
		FinishedOnboarding: req.FinishedOnboarding,
		Metadata:           req.Metadata,
	}, web3authId, email)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Create wallets for all active networks if wallet data is provided and it's a new user
	if result.IsNewUser && req.WalletData != nil {
		err = h.createWalletsForActiveNetworks(c, result.Workspaces[0].ID, req.WalletData)
		if err != nil {
			log.Printf("Warning: Failed to create wallets for active networks: %v", err)
		}
	}

	response := helpers.AccountDetailsResponse{
		AccountResponse: helpers.ToAccountResponse(*result.Account, result.Workspaces),
		User:            helpers.ToUserResponse(*result.User),
	}

	if result.IsNewUser {
		sendSuccess(c, http.StatusCreated, response)
	} else {
		sendSuccess(c, http.StatusOK, response)
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

	// Get workspace to get account ID
	workspace, err := h.common.db.GetWorkspace(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusNotFound, "Workspace not found", err)
		return
	}

	var req OnboardAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Onboard account using service
	err = h.accountService.OnboardAccount(c.Request.Context(), params.OnboardAccountParams{
		AccountID:    workspace.AccountID,
		UserID:       parsedUserID,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
	})
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
	case "account not found", constants.UserNotFound:
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

	err = h.accountService.DeleteAccount(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Account not found")
		return
	}

	sendSuccessMessage(c, http.StatusNoContent, "Account successfully deleted")
}
