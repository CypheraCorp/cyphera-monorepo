package handlers

import (
	"cyphera-api/internal/db"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CustomerHandler struct {
	common *CommonServices
}

func NewCustomerHandler(common *CommonServices) *CustomerHandler {
	return &CustomerHandler{common: common}
}

// CustomerResponse represents the standardized API response for customer operations
type CustomerResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	AccountID         string                 `json:"account_id"`
	Email             string                 `json:"email"`
	Name              string                 `json:"name,omitempty"`
	Description       string                 `json:"description,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Balance           int32                  `json:"balance"`
	Currency          string                 `json:"currency"`
	DefaultSourceID   string                 `json:"default_source,omitempty"`
	InvoicePrefix     string                 `json:"invoice_prefix,omitempty"`
	NextInvoiceNumber int32                  `json:"next_invoice_number"`
	TaxExempt         string                 `json:"tax_exempt"`
	TaxIDs            map[string]interface{} `json:"tax_ids,omitempty"`
	Livemode          bool                   `json:"livemode"`
	Created           int64                  `json:"created"`
	AccountName       string                 `json:"account_name,omitempty"`
	BusinessName      string                 `json:"business_name,omitempty"`
}

// CreateCustomerRequest represents the request body for creating a customer
type CreateCustomerRequest struct {
	Email       string                 `json:"email" binding:"required,email"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Currency    string                 `json:"currency,omitempty"`
	TaxExempt   string                 `json:"tax_exempt,omitempty"`
	TaxIDs      map[string]interface{} `json:"tax_ids,omitempty"`
}

// UpdateCustomerRequest represents the request body for updating a customer
type UpdateCustomerRequest struct {
	Email       string                 `json:"email,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Currency    string                 `json:"currency,omitempty"`
	TaxExempt   string                 `json:"tax_exempt,omitempty"`
	TaxIDs      map[string]interface{} `json:"tax_ids,omitempty"`
}

// GetCustomer godoc
// @Summary Get a customer
// @Description Retrieves the details of an existing customer
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
// @Summary List all customers
// @Description Returns a list of your customers. The customers are returned sorted by creation date, with the most recent customers appearing first.
// @Tags customers
// @Accept json
// @Produce json
// @Success 200 {array} CustomerResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers [get]
func (h *CustomerHandler) ListCustomers(c *gin.Context) {
	// Get user role and account ID from context (set by auth middleware)
	role := c.GetString("userRole")
	accountID := c.GetString("accountID")

	parsedAccountID, _ := uuid.Parse(accountID)

	customers, err := h.common.db.GetCustomersByScope(c.Request.Context(), db.GetCustomersByScopeParams{
		Column1:   role,
		AccountID: parsedAccountID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve customers"})
		return
	}

	response := make([]CustomerResponse, len(customers))
	for i, customer := range customers {
		response[i] = toCustomerScopeResponse(customer)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   response,
	})
}

// CreateCustomer godoc
// @Summary Create a customer
// @Description Creates a new customer object
// @Tags customers
// @Accept json
// @Produce json
// @Param customer body CreateCustomerRequest true "Customer creation data"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Security ApiKeyAuth
// @Router /customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	accountID := c.GetString("accountID")
	parsedAccountID, _ := uuid.Parse(accountID)

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
		AccountID:   parsedAccountID,
		Email:       req.Email,
		Name:        pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Metadata:    metadata,
		Currency:    pgtype.Text{String: req.Currency, Valid: req.Currency != ""},
		TaxExempt:   pgtype.Text{String: req.TaxExempt, Valid: req.TaxExempt != ""},
		TaxIds:      taxIDs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create customer"})
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(customer))
}

// UpdateCustomer godoc
// @Summary Update a customer
// @Description Updates the specified customer by setting the values of the parameters passed
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
		Email:       req.Email,
		Name:        pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Metadata:    metadata,
		Currency:    pgtype.Text{String: req.Currency, Valid: req.Currency != ""},
		TaxExempt:   pgtype.Text{String: req.TaxExempt, Valid: req.TaxExempt != ""},
		TaxIds:      taxIDs,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(customer))
}

// DeleteCustomer godoc
// @Summary Delete a customer
// @Description Deletes a customer
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

// Helper functions to convert database models to API responses
func toCustomerResponse(c db.Customer) CustomerResponse {
	var metadata map[string]interface{}
	json.Unmarshal(c.Metadata, &metadata)

	var taxIDs map[string]interface{}
	json.Unmarshal(c.TaxIds, &taxIDs)

	defaultSourceID := ""
	if c.DefaultSourceID.Valid {
		defaultSourceID = uuid.UUID(c.DefaultSourceID.Bytes).String()
	}

	return CustomerResponse{
		ID:              c.ID.String(),
		Object:          "customer",
		AccountID:       c.AccountID.String(),
		Email:           c.Email,
		Name:            c.Name.String,
		Description:     c.Description.String,
		Metadata:        metadata,
		Balance:         c.Balance.Int32,
		Currency:        c.Currency.String,
		DefaultSourceID: defaultSourceID,
		InvoicePrefix:   c.InvoicePrefix.String,
		TaxExempt:       c.TaxExempt.String,
		TaxIDs:          taxIDs,
		Livemode:        c.Livemode.Bool,
		Created:         c.CreatedAt.Time.Unix(),
	}
}

func toCustomerScopeResponse(c db.GetCustomersByScopeRow) CustomerResponse {
	var metadata map[string]interface{}
	json.Unmarshal(c.Metadata, &metadata)

	var taxIDs map[string]interface{}
	json.Unmarshal(c.TaxIds, &taxIDs)

	defaultSourceID := ""
	if c.DefaultSourceID.Valid {
		defaultSourceID = uuid.UUID(c.DefaultSourceID.Bytes).String()
	}

	return CustomerResponse{
		ID:              c.ID.String(),
		Object:          "customer",
		AccountID:       c.AccountID.String(),
		Email:           c.Email,
		Name:            c.Name.String,
		Description:     c.Description.String,
		Metadata:        metadata,
		Balance:         c.Balance.Int32,
		Currency:        c.Currency.String,
		DefaultSourceID: defaultSourceID,
		InvoicePrefix:   c.InvoicePrefix.String,
		TaxExempt:       c.TaxExempt.String,
		TaxIDs:          taxIDs,
		Livemode:        c.Livemode.Bool,
		Created:         c.CreatedAt.Time.Unix(),
		AccountName:     c.AccountName.String,
		BusinessName:    c.AccountBusinessName.String,
	}
}
