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
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	WorkspaceID       string                 `json:"workspace_id"`
	ExternalID        string                 `json:"external_id,omitempty"`
	Email             string                 `json:"email"`
	Name              string                 `json:"name,omitempty"`
	Phone             string                 `json:"phone,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	BalanceInPennies  int32                  `json:"balance_in_pennies"`
	Currency          string                 `json:"currency"`
	DefaultSourceID   string                 `json:"default_source,omitempty"`
	InvoicePrefix     string                 `json:"invoice_prefix,omitempty"`
	NextInvoiceNumber int32                  `json:"next_invoice_number"`
	TaxExempt         bool                   `json:"tax_exempt"`
	TaxIDs            map[string]interface{} `json:"tax_ids,omitempty"`
	Livemode          bool                   `json:"livemode"`
	CreatedAt         int64                  `json:"created_at"`
	UpdatedAt         int64                  `json:"updated_at"`
	WorkspaceName     string                 `json:"workspace_name,omitempty"`
	BusinessName      string                 `json:"business_name,omitempty"`
}

// CreateCustomerRequest represents the request body for creating a customer
type CreateCustomerRequest struct {
	ExternalID          string                 `json:"external_id,omitempty"`
	Email               string                 `json:"email" binding:"required,email"`
	Name                string                 `json:"name,omitempty"`
	Phone               string                 `json:"phone,omitempty"`
	Description         string                 `json:"description,omitempty"`
	BalanceInPennies    int32                  `json:"balance_in_pennies,omitempty"`
	Currency            string                 `json:"currency,omitempty" binding:"required_with=BalanceInPennies"`
	DefaultSourceID     string                 `json:"default_source_id,omitempty" binding:"omitempty,uuid4"`
	InvoicePrefix       string                 `json:"invoice_prefix,omitempty"`
	NextInvoiceSequence *int32                 `json:"next_invoice_sequence,omitempty"`
	TaxExempt           *bool                  `json:"tax_exempt,omitempty"`
	TaxIDs              map[string]interface{} `json:"tax_ids,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Livemode            *bool                  `json:"livemode,omitempty"`
}

// UpdateCustomerRequest represents the request body for updating a customer
type UpdateCustomerRequest struct {
	ExternalID          *string                `json:"external_id,omitempty"`
	Email               *string                `json:"email,omitempty" binding:"omitempty,email"`
	Name                *string                `json:"name,omitempty"`
	Phone               *string                `json:"phone,omitempty"`
	Description         *string                `json:"description,omitempty"`
	BalanceInPennies    *int32                 `json:"balance_in_pennies,omitempty"`
	Currency            *string                `json:"currency,omitempty" binding:"omitempty,required_with=BalanceInPennies"`
	DefaultSourceID     *string                `json:"default_source_id,omitempty" binding:"omitempty,uuid4"`
	InvoicePrefix       *string                `json:"invoice_prefix,omitempty"`
	NextInvoiceSequence *int32                 `json:"next_invoice_sequence,omitempty"`
	TaxExempt           *bool                  `json:"tax_exempt,omitempty"`
	TaxIDs              map[string]interface{} `json:"tax_ids,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Livemode            *bool                  `json:"livemode,omitempty"`
}

// ListCustomersResponse represents the paginated response for customer list operations
type ListCustomersResponse struct {
	Object  string             `json:"object"`
	Data    []CustomerResponse `json:"data"`
	HasMore bool               `json:"has_more"`
	Total   int64              `json:"total"`
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
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	customerId := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	customer, err := h.common.db.GetCustomer(c.Request.Context(), db.GetCustomerParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
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
	if req.InvoicePrefix != nil {
		params.InvoicePrefix = pgtype.Text{String: *req.InvoicePrefix, Valid: true}
	}
	if req.Currency != nil {
		params.Currency = pgtype.Text{String: *req.Currency, Valid: true}
	}
}

// updateCustomerNumericFields updates numeric fields of the customer
func (h *CustomerHandler) updateCustomerNumericFields(params *db.UpdateCustomerParams, req UpdateCustomerRequest) {
	if req.BalanceInPennies != nil {
		params.BalanceInPennies = pgtype.Int4{Int32: *req.BalanceInPennies, Valid: true}
	}
	if req.NextInvoiceSequence != nil {
		params.NextInvoiceSequence = pgtype.Int4{Int32: *req.NextInvoiceSequence, Valid: true}
	}
}

// updateCustomerJSONFields updates JSON fields of the customer
func (h *CustomerHandler) updateCustomerJSONFields(params *db.UpdateCustomerParams, req UpdateCustomerRequest) error {
	if req.TaxIDs != nil {
		taxIDs, err := json.Marshal(req.TaxIDs)
		if err != nil {
			return fmt.Errorf("invalid tax IDs format: %w", err)
		}
		params.TaxIds = taxIDs
	}
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

	// Update numeric fields
	h.updateCustomerNumericFields(&params, req)

	// Update boolean fields
	if req.TaxExempt != nil {
		params.TaxExempt = pgtype.Bool{Bool: *req.TaxExempt, Valid: true}
	}

	// Update UUID fields
	if req.DefaultSourceID != nil {
		parsedSourceID, err := uuid.Parse(*req.DefaultSourceID)
		if err != nil {
			return params, fmt.Errorf("invalid default source ID: %w", err)
		}
		params.DefaultSourceID = pgtype.UUID{Bytes: parsedSourceID, Valid: true}
	}

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
// @Success 200 {object} ListCustomersResponse
// @Failure 400 {object} ErrorResponse "Invalid workspace ID format or pagination parameters"
// @Failure 401 {object} ErrorResponse "Unauthorized access to workspace"
// @Failure 500 {object} ErrorResponse "Server error"
// @Security ApiKeyAuth
// @Router /customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	// Get pagination parameters
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
		WorkspaceID: parsedWorkspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		handleDBError(c, err, "Failed to retrieve customers")
		return
	}

	listCustomersResponse := ListCustomersResponse{
		Object:  "list",
		Data:    make([]CustomerResponse, len(customers)),
		HasMore: false,
		Total:   int64(len(customers)),
	}

	for i, customer := range customers {
		listCustomersResponse.Data[i] = toCustomerResponse(customer)
	}

	sendSuccess(c, http.StatusOK, listCustomersResponse)
}

// CreateCustomer godoc
// @Summary Create customer
// @Description Creates a new customer
// @Tags customers
// @Accept json
// @Produce json
// @Param customer body CreateCustomerRequest true "Customer creation data"
// @Success 201 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid metadata format", err)
		return
	}

	customer, err := h.common.db.CreateCustomer(c.Request.Context(), db.CreateCustomerParams{
		WorkspaceID: parsedWorkspaceID,
		ExternalID:  pgtype.Text{String: req.ExternalID, Valid: req.ExternalID != ""},
		Email:       pgtype.Text{String: req.Email, Valid: req.Email != ""},
		Name:        pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Phone:       pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		BalanceInPennies: pgtype.Int4{
			Int32: req.BalanceInPennies,
			Valid: req.BalanceInPennies != 0,
		},
		Metadata: metadata,
	})
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create customer", err)
		return
	}

	sendSuccess(c, http.StatusCreated, toCustomerResponse(customer))
}

// validateCustomerParams validates customer parameters
func validateCustomerParams(req *UpdateCustomerRequest) error {
	if req.BalanceInPennies != nil && *req.BalanceInPennies < 0 {
		return fmt.Errorf("balance_in_pennies cannot be negative")
	}
	if req.NextInvoiceSequence != nil && *req.NextInvoiceSequence < 0 {
		return fmt.Errorf("next_invoice_sequence cannot be negative")
	}
	return nil
}

// UpdateCustomer godoc
// @Summary Update customer
// @Description Updates an existing customer
// @Tags customers
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Param customer body UpdateCustomerRequest true "Customer update data"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{customer_id} [put]
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

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

	// Validate integer parameters
	if err := validateCustomerParams(&req); err != nil {
		sendError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	params, err := h.updateCustomerParams(parsedUUID, req)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid update parameters", err)
		return
	}

	// Update the workspace ID
	params.WorkspaceID = parsedWorkspaceID

	customer, err := h.common.db.UpdateCustomer(c.Request.Context(), params)
	if err != nil {
		handleDBError(c, err, "Failed to update customer")
		return
	}

	sendSuccess(c, http.StatusOK, toCustomerResponse(customer))
}

// DeleteCustomer godoc
// @Summary Delete customer
// @Description Deletes a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{customer_id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid workspace ID format", err)
		return
	}

	customerId := c.Param("customer_id")
	parsedUUID, err := uuid.Parse(customerId)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid customer ID format", err)
		return
	}

	err = h.common.db.DeleteCustomer(c.Request.Context(), db.DeleteCustomerParams{
		ID:          parsedUUID,
		WorkspaceID: parsedWorkspaceID,
	})
	if err != nil {
		handleDBError(c, err, "Failed to delete customer")
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper function to convert database model to API response
func toCustomerResponse(c db.Customer) CustomerResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(c.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling customer metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	var taxIDs map[string]interface{}
	if len(c.TaxIds) > 0 {
		if err := json.Unmarshal(c.TaxIds, &taxIDs); err != nil {
			log.Printf("Error unmarshaling customer tax IDs: %v", err)
			taxIDs = make(map[string]interface{}) // Use empty map if unmarshal fails
		}
	} else {
		taxIDs = make(map[string]interface{}) // Initialize empty map for nil or empty tax IDs
	}

	defaultSourceID := ""
	if c.DefaultSourceID.Valid {
		defaultSourceID = uuid.UUID(c.DefaultSourceID.Bytes).String()
	}

	return CustomerResponse{
		ID:                c.ID.String(),
		Object:            "customer",
		WorkspaceID:       c.WorkspaceID.String(),
		ExternalID:        c.ExternalID.String,
		Email:             c.Email.String,
		Name:              c.Name.String,
		Phone:             c.Phone.String,
		Description:       c.Description.String,
		Metadata:          metadata,
		BalanceInPennies:  c.BalanceInPennies.Int32,
		Currency:          c.Currency.String,
		DefaultSourceID:   defaultSourceID,
		InvoicePrefix:     c.InvoicePrefix.String,
		NextInvoiceNumber: c.NextInvoiceSequence.Int32,
		TaxExempt:         c.TaxExempt.Bool,
		TaxIDs:            taxIDs,
		Livemode:          c.Livemode.Bool,
		CreatedAt:         c.CreatedAt.Time.Unix(),
		UpdatedAt:         c.UpdatedAt.Time.Unix(),
	}
}
