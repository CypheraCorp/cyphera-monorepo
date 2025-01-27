package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
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
	Balance           int32                  `json:"balance"`
	Currency          string                 `json:"currency"`
	DefaultSourceID   string                 `json:"default_source,omitempty"`
	InvoicePrefix     string                 `json:"invoice_prefix,omitempty"`
	NextInvoiceNumber int32                  `json:"next_invoice_number"`
	TaxExempt         bool                   `json:"tax_exempt"`
	TaxIDs            map[string]interface{} `json:"tax_ids,omitempty"`
	Livemode          bool                   `json:"livemode"`
	Created           int64                  `json:"created"`
	WorkspaceName     string                 `json:"workspace_name,omitempty"`
	BusinessName      string                 `json:"business_name,omitempty"`
}

// CreateCustomerRequest represents the request body for creating a customer
type CreateCustomerRequest struct {
	ExternalID  string                 `json:"external_id,omitempty"`
	Email       string                 `json:"email" binding:"required,email"`
	Name        string                 `json:"name,omitempty"`
	Phone       string                 `json:"phone,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Currency    string                 `json:"currency,omitempty"`
	TaxExempt   bool                   `json:"tax_exempt,omitempty"`
	TaxIDs      map[string]interface{} `json:"tax_ids,omitempty"`
}

// UpdateCustomerRequest represents the request body for updating a customer
type UpdateCustomerRequest struct {
	ExternalID  string                 `json:"external_id,omitempty"`
	Email       string                 `json:"email,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Phone       string                 `json:"phone,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Currency    string                 `json:"currency,omitempty"`
	TaxExempt   bool                   `json:"tax_exempt,omitempty"`
	TaxIDs      map[string]interface{} `json:"tax_ids,omitempty"`
}

// GetCustomer godoc
// @Summary Get a customer
// @Description Retrieves a specific customer by its ID
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{id} [get]
func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid customer ID format"})
		return
	}

	customer, err := h.common.db.GetCustomer(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(customer))
}

// ListCustomers godoc
// @Summary List customers
// @Description Retrieves all customers for the current workspace
// @Tags customers
// @Accept json
// @Produce json
// @Success 200 {array} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	workspaceID := c.GetString("workspaceID")
	parsedUUID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID format"})
		return
	}

	customers, err := h.common.db.ListCustomers(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve customers"})
		return
	}

	response := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		response[i] = toCustomerResponse(customer)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// CreateCustomer godoc
// @Summary Create customer
// @Description Creates a new customer in the current workspace
// @Tags customers
// @Accept json
// @Produce json
// @Param customer body CreateCustomerRequest true "Customer creation data"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	workspaceID := c.GetString("workspaceID")
	parsedWorkspaceID, err := uuid.Parse(workspaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID format"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	taxIDs, err := json.Marshal(req.TaxIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid tax IDs format"})
		return
	}

	customer, err := h.common.db.CreateCustomer(c.Request.Context(), db.CreateCustomerParams{
		WorkspaceID: parsedWorkspaceID,
		Email:       pgtype.Text{String: req.Email, Valid: req.Email != ""},
		Name:        pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Phone:       pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Currency:    pgtype.Text{String: req.Currency, Valid: req.Currency != ""},
		TaxExempt:   pgtype.Bool{Bool: req.TaxExempt, Valid: true},
		TaxIds:      taxIDs,
		Metadata:    metadata,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create customer"})
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(customer))
}

// UpdateCustomer godoc
// @Summary Update customer
// @Description Updates an existing customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param customer body UpdateCustomerRequest true "Customer update data"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid customer ID format"})
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid metadata format"})
		return
	}

	taxIDs, err := json.Marshal(req.TaxIDs)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid tax IDs format"})
		return
	}

	customer, err := h.common.db.UpdateCustomer(c.Request.Context(), db.UpdateCustomerParams{
		ID:          parsedUUID,
		Email:       pgtype.Text{String: req.Email, Valid: req.Email != ""},
		Name:        pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Phone:       pgtype.Text{String: req.Phone, Valid: req.Phone != ""},
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Currency:    pgtype.Text{String: req.Currency, Valid: req.Currency != ""},
		TaxExempt:   pgtype.Bool{Bool: req.TaxExempt, Valid: true},
		TaxIds:      taxIDs,
		Metadata:    metadata,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(customer))
}

// DeleteCustomer godoc
// @Summary Delete customer
// @Description Soft deletes a customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	id := c.Param("id")
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid customer ID format"})
		return
	}

	err = h.common.db.DeleteCustomer(c.Request.Context(), parsedUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Customer not found"})
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
	if err := json.Unmarshal(c.TaxIds, &taxIDs); err != nil {
		log.Printf("Error unmarshaling customer tax IDs: %v", err)
		taxIDs = make(map[string]interface{}) // Use empty map if unmarshal fails
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
		Balance:           c.Balance.Int32,
		Currency:          c.Currency.String,
		DefaultSourceID:   defaultSourceID,
		InvoicePrefix:     c.InvoicePrefix.String,
		NextInvoiceNumber: c.NextInvoiceSequence.Int32,
		TaxExempt:         c.TaxExempt.Bool,
		TaxIDs:            taxIDs,
		Livemode:          c.Livemode.Bool,
		Created:           c.CreatedAt.Time.Unix(),
	}
}
