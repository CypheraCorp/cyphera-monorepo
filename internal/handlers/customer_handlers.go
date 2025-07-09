package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// CustomerHandler handles customer related operations
type CustomerHandler struct {
	common *CommonServices
}

// NewCustomerHandler creates a new instance of CustomerHandler
func NewCustomerHandler(common *CommonServices) *CustomerHandler {
	return &CustomerHandler{common: common}
}

// CustomerResponse represents the standardized API response for customer operations
type CustomerResponse struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	ExternalID         string                 `json:"external_id,omitempty"`
	Email              string                 `json:"email"`
	Name               string                 `json:"name,omitempty"`
	Phone              string                 `json:"phone,omitempty"`
	Description        string                 `json:"description,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
}

// CreateCustomerRequest represents the request body for creating a customer
type CreateCustomerRequest struct {
	ExternalID         string                 `json:"external_id,omitempty"`
	Email              string                 `json:"email" binding:"required,email"`
	Name               string                 `json:"name,omitempty"`
	Phone              string                 `json:"phone,omitempty"`
	Description        string                 `json:"description,omitempty"`
	FinishedOnboarding *bool                  `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateCustomerRequest represents the request body for updating a customer
type UpdateCustomerRequest struct {
	ExternalID         *string                `json:"external_id,omitempty"`
	Email              *string                `json:"email,omitempty" binding:"omitempty,email"`
	Name               *string                `json:"name,omitempty"`
	Phone              *string                `json:"phone,omitempty"`
	Description        *string                `json:"description,omitempty"`
	FinishedOnboarding *bool                  `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// ListCustomersResponse represents the paginated response for customer list operations
type ListCustomersResponse struct {
	Object  string             `json:"object"`
	Data    []CustomerResponse `json:"data"`
	HasMore bool               `json:"has_more"`
	Total   int64              `json:"total"`
}

// Customer wallet response for the sign-in/register API
type CustomerWalletResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	CustomerID    string                 `json:"customer_id"`
	WalletAddress string                 `json:"wallet_address"`
	NetworkType   string                 `json:"network_type"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// SignInRegisterCustomerRequest represents the request body for customer sign-in/register
type SignInRegisterCustomerRequest struct {
	Email    string                 `json:"email" binding:"required,email"`
	Name     string                 `json:"name,omitempty"`
	Phone    string                 `json:"phone,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Web3Auth wallet data to be created during registration
	WalletData *CustomerWalletRequest `json:"wallet_data,omitempty"`
}

// CustomerWalletRequest represents wallet data for customer registration
type CustomerWalletRequest struct {
	WalletAddress string                 `json:"wallet_address" binding:"required"`
	NetworkType   string                 `json:"network_type" binding:"required"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CustomerDetailsResponse represents the response for customer sign-in/register
type CustomerDetailsResponse struct {
	Customer CustomerResponse       `json:"customer"`
	Wallet   CustomerWalletResponse `json:"wallet,omitempty"`
}

// GetCustomer godoc
// @Summary Get customer by ID
// @Description Get customer details by customer ID
// @Tags customers
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{customer_id} [get]
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	customerId := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	customer, err := h.common.db.GetCustomer(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Customer not found")
		return
	}

	sendSuccess(c, http.StatusOK, toCustomerResponse(customer))
}

// updateBasicCustomerFields updates basic text fields of the customer
func (h *CustomerHandler) updateBasicCustomerFields(params *db.UpdateCustomerParams, req UpdateCustomerRequest) {
	if req.Email != nil {
		params.Email = pgtype.Text{String: *req.Email, Valid: true}
	}
	if req.Name != nil {
		params.Name = pgtype.Text{String: *req.Name, Valid: true}
	}
	if req.Phone != nil {
		params.Phone = pgtype.Text{String: *req.Phone, Valid: true}
	}
	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.FinishedOnboarding != nil {
		params.FinishedOnboarding = pgtype.Bool{Bool: *req.FinishedOnboarding, Valid: true}
	}
}

// updateCustomerJSONFields updates JSON fields of the customer
func (h *CustomerHandler) updateCustomerJSONFields(params *db.UpdateCustomerParams, req UpdateCustomerRequest) error {
	if req.Metadata != nil {
		metadata, err := json.Marshal(req.Metadata)
		if err != nil {
			return fmt.Errorf("invalid metadata format: %w", err)
		}
		params.Metadata = metadata
	}
	return nil
}

// updateCustomerParams creates the update parameters for a customer
func (h *CustomerHandler) updateCustomerParams(id uuid.UUID, req UpdateCustomerRequest) (db.UpdateCustomerParams, error) {
	params := db.UpdateCustomerParams{
		ID: id,
	}

	// Update basic text fields
	h.updateBasicCustomerFields(&params, req)

	// Update JSON fields
	if err := h.updateCustomerJSONFields(&params, req); err != nil {
		return params, err
	}

	return params, nil
}

// ListCustomers godoc
// @Summary List customers
// @Description Retrieves paginated customers for the current workspace
// @Tags customers
// @Accept json
// @Produce json
// @Param limit query int false "Number of customers to return (default 10, max 100)"
// @Param offset query int false "Number of customers to skip (default 0)"
// @Success 200 {object} PaginatedResponse{data=[]CustomerResponse}
// @Failure 400 {object} ErrorResponse "Invalid workspace ID format or pagination parameters"
// @Failure 401 {object} ErrorResponse "Unauthorized access to workspace"
// @Failure 500 {object} ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID != "" {
		// If workspace ID is provided, list customers for that workspace
		h.listWorkspaceCustomers(c, workspaceID)
		return
	}

	// Otherwise, list all customers (global)
	limit, page, err := validatePaginationParams(c)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid pagination parameters", err)
		return
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	customers, err := h.common.db.ListCustomersWithPagination(c.Request.Context(), db.ListCustomersWithPaginationParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		handleDBError(c, err, "Failed to retrieve customers")
		return
	}

	totalCount, err := h.common.db.CountCustomers(c.Request.Context())
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to count customers", err)
		return
	}

	customerResponses := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		customerResponses[i] = toCustomerResponse(customer)
	}

	response := sendPaginatedSuccess(c, http.StatusOK, customerResponses, int(page), int(limit), int(totalCount))
	c.JSON(http.StatusOK, response)
}

func (h *CustomerHandler) listWorkspaceCustomers(c *gin.Context, workspaceID string) {
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	limit, page, err := validatePaginationParams(c)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid pagination parameters", err)
		return
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	customers, err := h.common.db.ListWorkspaceCustomersWithPagination(c.Request.Context(), db.ListWorkspaceCustomersWithPaginationParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		handleDBError(c, err, "Failed to retrieve workspace customers")
		return
	}

	totalCount, err := h.common.db.CountWorkspaceCustomers(c.Request.Context(), parsedWorkspaceID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to count workspace customers", err)
		return
	}

	customerResponses := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		customerResponses[i] = toCustomerResponse(customer)
	}

	response := sendPaginatedSuccess(c, http.StatusOK, customerResponses, int(page), int(limit), int(totalCount))
	c.JSON(http.StatusOK, response)
}

// CreateCustomer godoc
// @Summary Create a new customer
// @Description Creates a new customer with the specified details
// @Tags customers
// @Accept json
// @Produce json
// @Tags exclude
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	metadataBytes, err := json.Marshal(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	finishedOnboarding := false
	if req.FinishedOnboarding != nil {
		finishedOnboarding = *req.FinishedOnboarding
	}

	customer, err := h.common.db.CreateCustomer(c.Request.Context(), db.CreateCustomerParams{
		ExternalID:         pgtype.Text{String: req.ExternalID, Valid: req.ExternalID != ""},
		Email:              pgtype.Text{String: req.Email, Valid: req.Email != ""},
		Name:               pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Description:        pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Phone:              pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Metadata:           metadataBytes,
		FinishedOnboarding: finishedOnboarding,
		PaymentSyncStatus:  "pending",
		PaymentProvider:    pgtype.Text{},
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create customer", err)
		return
	}

	// If workspace ID is provided, associate the customer with the workspace
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID != "" {
		parsedWorkspaceID, err := uuid.Parse(workspaceID)
		if err == nil {
			_, err = h.common.db.AddCustomerToWorkspace(c.Request.Context(), db.AddCustomerToWorkspaceParams{
				WorkspaceID: parsedWorkspaceID,
				CustomerID:  customer.ID,
			})
			if err != nil {
				log.Printf("Failed to associate customer with workspace: %v", err)
				// Don't fail the request, just log the error
			}
		}
	}

	sendSuccess(c, http.StatusCreated, toCustomerResponse(customer))
}

// UpdateCustomer godoc
// @Summary Update a customer
// @Description Updates an existing customer with the specified details
// @Tags customers
// @Accept json
// @Produce json
// @Tags exclude
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	customerId := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	params, err := h.updateCustomerParams(parsedUUID, req)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid update parameters", err)
		return
	}

	customer, err := h.common.db.UpdateCustomer(c.Request.Context(), params)
	if err != nil {
		handleDBError(c, err, "Failed to update customer")
		return
	}

	sendSuccess(c, http.StatusOK, toCustomerResponse(customer))
}

// DeleteCustomer godoc
// @Summary Delete a customer
// @Description Deletes a customer with the specified ID
// @Tags customers
// @Accept json
// @Produce json
// @Tags exclude
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	customerId := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	err = h.common.db.DeleteCustomer(c.Request.Context(), parsedUUID)
	if err != nil {
		handleDBError(c, err, "Failed to delete customer")
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateCustomerOnboardingStatusRequest represents the request body for updating customer onboarding status
type UpdateCustomerOnboardingStatusRequest struct {
	FinishedOnboarding bool `json:"finished_onboarding" binding:"required"`
}

// UpdateCustomerOnboardingStatus godoc
// @Summary Update customer onboarding status
// @Description Updates the finished_onboarding status for a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Param finished_onboarding body bool true "Finished onboarding status"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{customer_id}/onboarding [put]
func (h *CustomerHandler) UpdateCustomerOnboardingStatus(c *gin.Context) {
	customerId := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	var req struct {
		FinishedOnboarding bool `json:"finished_onboarding" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	customer, err := h.common.db.UpdateCustomerOnboardingStatus(c.Request.Context(), db.UpdateCustomerOnboardingStatusParams{
		ID:                 parsedUUID,
		FinishedOnboarding: pgtype.Bool{Bool: req.FinishedOnboarding, Valid: true},
	})
	if err != nil {
		handleDBError(c, err, "Failed to update customer onboarding status")
		return
	}

	sendSuccess(c, http.StatusOK, toCustomerResponse(customer))
}

// Helper function to convert database model to API response
func toCustomerResponse(c db.Customer) CustomerResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(c.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling customer metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return CustomerResponse{
		ID:                 c.ID.String(),
		Object:             "customer",
		ExternalID:         c.ExternalID.String,
		Email:              c.Email.String,
		Name:               c.Name.String,
		Phone:              c.Phone.String,
		Description:        c.Description.String,
		FinishedOnboarding: c.FinishedOnboarding.Bool,
		Metadata:           metadata,
		CreatedAt:          c.CreatedAt.Time.Unix(),
		UpdatedAt:          c.UpdatedAt.Time.Unix(),
	}
}

// SignInRegisterCustomer godoc
// @Summary Sign in or register a customer
// @Description Signs in to an existing customer account or creates a new customer with Web3Auth ID
// @Tags customers
// @Accept json
// @Produce json
// @Tags exclude
func (h *CustomerHandler) SignInRegisterCustomer(c *gin.Context) {
	var req SignInRegisterCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	web3authId, email, metadata, err := h.validateCustomerSignInRequest(req)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	// Check if customer already exists by Web3Auth ID
	customer, err := h.common.db.GetCustomerByWeb3AuthID(c.Request.Context(), pgtype.Text{String: web3authId, Valid: web3authId != ""})
	if err != nil {
		if err.Error() != "no rows in result set" {
			sendError(c, http.StatusInternalServerError, "Failed to check existing customer", err)
			return
		}
	}

	var response *CustomerDetailsResponse
	if err != nil && err.Error() == "no rows in result set" {
		// Customer doesn't exist, create new customer and wallet
		response, err = h.createNewCustomerWithWallet(c, req, web3authId, email, metadata)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
		sendSuccess(c, http.StatusCreated, response)
	} else {
		// Customer exists, get existing customer and wallet details
		wallets, err := h.common.db.ListCustomerWallets(c.Request.Context(), customer.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve customer wallets", err)
			return
		}

		var walletResponse CustomerWalletResponse
		if len(wallets) > 0 {
			// Find primary wallet or use first wallet
			var primaryWallet *db.CustomerWallet
			for _, wallet := range wallets {
				if wallet.IsPrimary.Bool {
					primaryWallet = &wallet
					break
				}
			}
			if primaryWallet == nil {
				primaryWallet = &wallets[0]
			}
			walletResponse = toCustomerWalletResponse(*primaryWallet)
		}

		response = &CustomerDetailsResponse{
			Customer: toCustomerResponse(customer),
			Wallet:   walletResponse,
		}
		sendSuccess(c, http.StatusOK, response)
	}
}

// validateCustomerSignInRequest validates the sign-in request and extracts metadata
func (h *CustomerHandler) validateCustomerSignInRequest(req SignInRegisterCustomerRequest) (string, string, []byte, error) {
	// Extract Web3Auth ID from metadata if present
	web3authId := ""
	if req.Metadata != nil {
		if id, exists := req.Metadata["web3auth_id"]; exists {
			if idStr, ok := id.(string); ok {
				web3authId = idStr
			}
		}
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	return web3authId, req.Email, metadata, nil
}

// createNewCustomerWithWallet creates a new customer and associated wallet
func (h *CustomerHandler) createNewCustomerWithWallet(ctx *gin.Context, req SignInRegisterCustomerRequest, web3authId string, email string, metadata []byte) (*CustomerDetailsResponse, error) {
	// Create the customer (now workspace-independent)
	customer, err := h.common.db.CreateCustomerWithWeb3Auth(ctx.Request.Context(), db.CreateCustomerWithWeb3AuthParams{
		Web3authID:         pgtype.Text{String: web3authId, Valid: web3authId != ""},
		Email:              pgtype.Text{String: email, Valid: email != ""},
		Name:               pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Phone:              pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Description:        pgtype.Text{},
		Metadata:           metadata,
		FinishedOnboarding: false, // finished_onboarding defaults to false
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	var walletResponse CustomerWalletResponse

	// Create wallet if wallet data is provided
	if req.WalletData != nil {
		walletMetadata, err := json.Marshal(req.WalletData.Metadata)
		if err != nil {
			return nil, fmt.Errorf("invalid wallet metadata format: %w", err)
		}

		// Parse network type
		networkType, err := h.parseNetworkType(req.WalletData.NetworkType)
		if err != nil {
			return nil, fmt.Errorf("invalid network type: %w", err)
		}

		wallet, err := h.common.db.CreateCustomerWallet(ctx.Request.Context(), db.CreateCustomerWalletParams{
			CustomerID:    customer.ID,
			WalletAddress: req.WalletData.WalletAddress,
			NetworkType:   networkType,
			Nickname:      pgtype.Text{String: req.WalletData.Nickname, Valid: req.WalletData.Nickname != ""},
			Ens:           pgtype.Text{String: req.WalletData.ENS, Valid: req.WalletData.ENS != ""},
			IsPrimary:     pgtype.Bool{Bool: req.WalletData.IsPrimary, Valid: true},
			Verified:      pgtype.Bool{Bool: req.WalletData.Verified, Valid: true},
			Metadata:      walletMetadata,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create customer wallet: %w", err)
		}

		walletResponse = toCustomerWalletResponse(wallet)
	}

	return &CustomerDetailsResponse{
		Customer: toCustomerResponse(customer),
		Wallet:   walletResponse,
	}, nil
}

// parseNetworkType converts string to NetworkType
func (h *CustomerHandler) parseNetworkType(networkType string) (db.NetworkType, error) {
	switch networkType {
	case "evm":
		return db.NetworkTypeEvm, nil
	case "solana":
		return db.NetworkTypeSolana, nil
	case "cosmos":
		return db.NetworkTypeCosmos, nil
	case "bitcoin":
		return db.NetworkTypeBitcoin, nil
	case "polkadot":
		return db.NetworkTypePolkadot, nil
	default:
		return "", fmt.Errorf("unsupported network type: %s", networkType)
	}
}

// toCustomerWalletResponse converts database CustomerWallet to API response
func toCustomerWalletResponse(w db.CustomerWallet) CustomerWalletResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling wallet metadata: %v", err)
		metadata = make(map[string]interface{})
	}

	return CustomerWalletResponse{
		ID:            w.ID.String(),
		Object:        "customer_wallet",
		CustomerID:    w.CustomerID.String(),
		WalletAddress: w.WalletAddress,
		NetworkType:   string(w.NetworkType),
		Nickname:      w.Nickname.String,
		ENS:           w.Ens.String,
		IsPrimary:     w.IsPrimary.Bool,
		Verified:      w.Verified.Bool,
		Metadata:      metadata,
		CreatedAt:     w.CreatedAt.Time.Unix(),
		UpdatedAt:     w.UpdatedAt.Time.Unix(),
	}
}
