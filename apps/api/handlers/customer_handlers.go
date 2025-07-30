package handlers

import (
	"encoding/json"
	"fmt"
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
)

// CustomerHandler handles customer-related operations
type CustomerHandler struct {
	common          *CommonServices
	customerService interfaces.CustomerService
}

// Use types from the centralized packages
type CreateCustomerRequest = requests.CreateCustomerRequest
type UpdateCustomerRequest = requests.UpdateCustomerRequest
type SignInRegisterCustomerRequest = requests.SignInRegisterCustomerRequest
type CustomerWalletRequest = requests.CustomerWalletRequest
type UpdateCustomerOnboardingStatusRequest = requests.UpdateCustomerOnboardingStatusRequest

type CustomerResponse = responses.CustomerResponse
type CustomerWalletResponse = responses.CustomerWalletResponse
type CustomerDetailsResponse = responses.CustomerDetailsResponse

// NewCustomerHandler creates a handler with interface dependencies
func NewCustomerHandler(
	common *CommonServices,
	customerService interfaces.CustomerService,
) *CustomerHandler {
	return &CustomerHandler{
		common:          common,
		customerService: customerService,
	}
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

	customer, err := h.customerService.GetCustomer(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == constants.CustomerNotFound {
			sendError(c, http.StatusNotFound, "Customer not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToCustomerResponse(*customer))
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
	pageParams, err := helpers.ParsePaginationParams(c)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid pagination parameters", err)
		return
	}
	limit, page := pageParams.Limit, pageParams.Page

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	result, err := h.customerService.ListCustomers(c.Request.Context(), params.ListCustomersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// No conversion needed - result.Customers is already []responses.CustomerResponse
	customerResponses := result.Customers

	response := sendPaginatedSuccess(c, http.StatusOK, customerResponses, int(page), int(limit), int(result.Total))
	c.JSON(http.StatusOK, response)
}

func (h *CustomerHandler) listWorkspaceCustomers(c *gin.Context, workspaceID string) {
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	pageParams, err := helpers.ParsePaginationParams(c)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid pagination parameters", err)
		return
	}
	limit, page := pageParams.Limit, pageParams.Page

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	result, err := h.customerService.ListWorkspaceCustomers(c.Request.Context(), params.ListWorkspaceCustomersParams{
		WorkspaceID: parsedWorkspaceID,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// No conversion needed - result.Customers is already []responses.CustomerResponse
	customerResponses := result.Customers

	response := sendPaginatedSuccess(c, http.StatusOK, customerResponses, int(page), int(limit), int(result.Total))
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

	finishedOnboarding := false
	if req.FinishedOnboarding != nil {
		finishedOnboarding = *req.FinishedOnboarding
	}

	customer, err := h.customerService.CreateCustomer(c.Request.Context(), params.CreateCustomerParams{
		Email:              req.Email,
		Name:               &req.Name,
		Phone:              &req.Phone,
		Description:        &req.Description,
		FinishedOnboarding: finishedOnboarding,
		Metadata:           req.Metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// If workspace ID is provided, associate the customer with the workspace
	workspaceID := c.GetHeader("X-Workspace-ID")
	if workspaceID != "" {
		parsedWorkspaceID, err := uuid.Parse(workspaceID)
		if err == nil {
			err = h.customerService.AddCustomerToWorkspace(c.Request.Context(), parsedWorkspaceID, customer.ID)
			if err != nil {
				log.Printf("Failed to associate customer with workspace: %v", err)
				// Don't fail the request, just log the error
			}
		}
	}

	sendSuccess(c, http.StatusCreated, helpers.ToCustomerResponse(*customer))
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

	customer, err := h.customerService.UpdateCustomer(c.Request.Context(), params.UpdateCustomerParams{
		ID:                 parsedUUID,
		Email:              req.Email,
		Name:               req.Name,
		Phone:              req.Phone,
		Description:        req.Description,
		FinishedOnboarding: req.FinishedOnboarding,
		Metadata:           req.Metadata,
	})
	if err != nil {
		if err.Error() == constants.CustomerNotFound {
			sendError(c, http.StatusNotFound, "Customer not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToCustomerResponse(*customer))
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

	err = h.customerService.DeleteCustomer(c.Request.Context(), parsedUUID)
	if err != nil {
		if err.Error() == constants.CustomerNotFound {
			sendError(c, http.StatusNotFound, "Customer not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	c.Status(http.StatusNoContent)
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

	var req UpdateCustomerOnboardingStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	customer, err := h.customerService.UpdateCustomerOnboardingStatus(c.Request.Context(), parsedUUID, req.FinishedOnboarding)
	if err != nil {
		if err.Error() == constants.CustomerNotFound {
			sendError(c, http.StatusNotFound, "Customer not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	sendSuccess(c, http.StatusOK, helpers.ToCustomerResponse(*customer))
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
	customer, err := h.customerService.GetCustomerByWeb3AuthID(c.Request.Context(), web3authId)
	if err != nil {
		if err.Error() != "customer not found" {
			sendError(c, http.StatusInternalServerError, "Failed to check existing customer", err)
			return
		}
	}

	var response *CustomerDetailsResponse
	if err != nil && err.Error() == constants.CustomerNotFound {
		// Customer doesn't exist, create new customer and wallet
		response, err = h.createNewCustomerWithWallet(c, req, web3authId, email, metadata)
		if err != nil {
			sendError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
		sendSuccess(c, http.StatusCreated, response)
	} else {
		// Customer exists, get existing customer and wallet details
		wallets, err := h.customerService.ListCustomerWallets(c.Request.Context(), customer.ID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to retrieve customer wallets", err)
			return
		}

		var primaryWallet *db.CustomerWallet
		if len(wallets) > 0 {
			// Find primary wallet or use first wallet
			for _, wallet := range wallets {
				if wallet.IsPrimary.Bool {
					primaryWallet = &wallet
					break
				}
			}
			if primaryWallet == nil {
				primaryWallet = &wallets[0]
			}
		}

		response = &CustomerDetailsResponse{
			Customer: helpers.ToResponsesCustomerResponse(helpers.ToCustomerResponse(*customer)),
		}
		if primaryWallet != nil {
			response.Wallet = helpers.ToResponsesCustomerWalletResponse(helpers.ToCustomerWalletResponse(*primaryWallet))
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
	var metadataMap map[string]interface{}
	if err := json.Unmarshal(metadata, &metadataMap); err != nil {
		return nil, fmt.Errorf("invalid metadata format: %w", err)
	}

	// Create the customer (now workspace-independent)
	customer, err := h.customerService.CreateCustomerWithWeb3Auth(ctx.Request.Context(), params.CreateCustomerWithWeb3AuthParams{
		Web3AuthID: web3authId,
		Email:      email,
		Name:       &req.Name,
		Metadata:   metadataMap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	response := &CustomerDetailsResponse{
		Customer: helpers.ToResponsesCustomerResponse(helpers.ToCustomerResponse(*customer)),
	}

	// Create wallet if wallet data is provided
	if req.WalletData != nil {
		wallet, err := h.customerService.CreateCustomerWallet(ctx.Request.Context(), params.CreateCustomerWalletParams{
			CustomerID:    customer.ID,
			WalletAddress: req.WalletData.WalletAddress,
			NetworkType:   req.WalletData.NetworkType,
			IsPrimary:     req.WalletData.IsPrimary,
			Verified:      req.WalletData.Verified,
			Metadata:      req.WalletData.Metadata,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create customer wallet: %w", err)
		}

		response.Wallet = helpers.ToResponsesCustomerWalletResponse(helpers.ToCustomerWalletResponse(*wallet))
	}

	return response, nil
}
