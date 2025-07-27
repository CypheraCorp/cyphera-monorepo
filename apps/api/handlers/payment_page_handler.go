package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentPageHandler handles public payment page requests
type PaymentPageHandler struct {
	common             *CommonServices
	paymentLinkService *services.PaymentLinkService
}

// NewPaymentPageHandler creates a new payment page handler
func NewPaymentPageHandler(common *CommonServices) *PaymentPageHandler {
	baseURL := "https://pay.cyphera.com" // TODO: Get from environment
	paymentLinkService := services.NewPaymentLinkService(
		common.db,
		common.logger,
		baseURL,
	)

	return &PaymentPageHandler{
		common:             common,
		paymentLinkService: paymentLinkService,
	}
}

// PaymentPageDataResponse represents the data needed to render a payment page
type PaymentPageDataResponse struct {
	PaymentLink       PaymentLinkData       `json:"payment_link"`
	Product           *ProductData          `json:"product,omitempty"`
	Price             *PriceData            `json:"price,omitempty"`
	Workspace         WorkspaceData         `json:"workspace"`
	SupportedNetworks []NetworkData         `json:"supported_networks"`
	AcceptedTokens    []AcceptedTokenData   `json:"accepted_tokens"`
	GasSponsorship    *GasSponsorshipData   `json:"gas_sponsorship,omitempty"`
}

// PaymentLinkData represents payment link information for the payment page
type PaymentLinkData struct {
	ID              string     `json:"id"`
	Slug            string     `json:"slug"`
	Status          string     `json:"status"`
	AmountCents     *int64     `json:"amount_cents,omitempty"`
	Currency        string     `json:"currency"`
	PaymentType     string     `json:"payment_type"`
	CollectEmail    bool       `json:"collect_email"`
	CollectShipping bool       `json:"collect_shipping"`
	CollectName     bool       `json:"collect_name"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	MaxUses         *int32     `json:"max_uses,omitempty"`
	UsedCount       int32      `json:"used_count"`
	QRCodeData      *string    `json:"qr_code_data,omitempty"`
}

// ProductData represents product information for the payment page
type ProductData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

// PriceData represents price information for the payment page
type PriceData struct {
	ID                  string `json:"id"`
	UnitAmountInPennies int32  `json:"unit_amount_in_pennies"`
	Currency            string `json:"currency"`
	Type                string `json:"type"`
	IntervalType        string `json:"interval_type,omitempty"`
	IntervalCount       int32  `json:"interval_count,omitempty"`
}

// WorkspaceData represents workspace information for the payment page
type WorkspaceData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	LogoURL     string `json:"logo_url,omitempty"`
	BrandColor  string `json:"brand_color,omitempty"`
}

// NetworkData represents supported network information
type NetworkData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ChainID   string `json:"chain_id"`
	Type      string `json:"type"`
	IsTestnet bool   `json:"is_testnet"`
}

// AcceptedTokenData represents accepted token information
type AcceptedTokenData struct {
	ID            string `json:"id"`
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	NetworkID     string `json:"network_id"`
	NetworkName   string `json:"network_name"`
	ContractAddress string `json:"contract_address"`
	Decimals      int32  `json:"decimals"`
}

// GasSponsorshipData represents gas sponsorship information
type GasSponsorshipData struct {
	IsSponsored   bool   `json:"is_sponsored"`
	SponsorType   string `json:"sponsor_type,omitempty"`
	CoverageType  string `json:"coverage_type,omitempty"`
}

// PaymentIntentRequest represents a request to create a payment intent
type PaymentIntentRequest struct {
	CustomerEmail   string                 `json:"customer_email" binding:"required,email"`
	CustomerName    string                 `json:"customer_name"`
	WalletAddress   string                 `json:"wallet_address" binding:"required"`
	NetworkID       string                 `json:"network_id" binding:"required"`
	TokenID         string                 `json:"token_id" binding:"required"`
	ShippingAddress *ShippingAddressInput  `json:"shipping_address,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ShippingAddressInput represents shipping address input
type ShippingAddressInput struct {
	Line1      string `json:"line1" binding:"required"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	Country    string `json:"country" binding:"required"`
}

// PaymentIntentResponse represents the response for a payment intent
type PaymentIntentResponse struct {
	IntentID          string                 `json:"intent_id"`
	Status            string                 `json:"status"`
	AmountCents       int64                  `json:"amount_cents"`
	Currency          string                 `json:"currency"`
	NetworkID         string                 `json:"network_id"`
	TokenID           string                 `json:"token_id"`
	RecipientAddress  string                 `json:"recipient_address"`
	ExpiresAt         time.Time              `json:"expires_at"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// GetPaymentPageData retrieves all data needed to render a payment page
// @Summary Get payment page data
// @Description Get all data needed to render a payment page for a payment link
// @Tags payment-pages
// @Accept json
// @Produce json
// @Param slug path string true "Payment Link Slug"
// @Success 200 {object} PaymentPageDataResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-pages/{slug} [get]
func (h *PaymentPageHandler) GetPaymentPageData(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	// Get payment link by slug
	link, err := h.paymentLinkService.GetPaymentLinkBySlug(ctx, slug)
	if err != nil {
		h.common.HandleError(c, err, "Payment link not found or inactive", http.StatusNotFound, h.common.logger)
		return
	}

	// Get workspace details
	workspace, err := h.common.db.GetWorkspace(ctx, link.WorkspaceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get workspace details", http.StatusInternalServerError, h.common.logger)
		return
	}

	response := PaymentPageDataResponse{
		PaymentLink: PaymentLinkData{
			ID:              link.ID.String(),
			Slug:            link.Slug,
			Status:          link.Status,
			AmountCents:     link.AmountCents,
			Currency:        link.Currency,
			PaymentType:     link.PaymentType,
			CollectEmail:    link.CollectEmail,
			CollectShipping: link.CollectShipping,
			CollectName:     link.CollectName,
			ExpiresAt:       link.ExpiresAt,
			MaxUses:         link.MaxUses,
			UsedCount:       link.UsedCount,
			QRCodeData:      link.QRCodeData,
		},
		Workspace: WorkspaceData{
			ID:         workspace.ID.String(),
			Name:       workspace.Name,
			LogoURL:    "", // TODO: Add logo URL to workspace table
			BrandColor: "", // TODO: Add brand color to workspace settings
		},
	}

	// Get product details if available
	if link.ProductID != nil {
		// TODO: Implement product retrieval
		// For now, provide placeholder data
		response.Product = &ProductData{
			ID:          link.ProductID.String(),
			Name:        "Product",
			Description: "Product description",
			ImageURL:    "",
		}
	}

	// Get price details if available
	if link.PriceID != nil && link.AmountCents != nil {
		// TODO: Implement price retrieval
		// For now, use data from payment link
		response.Price = &PriceData{
			ID:                  link.PriceID.String(),
			UnitAmountInPennies: int32(*link.AmountCents),
			Currency:            link.Currency,
			Type:                link.PaymentType,
			IntervalType:        "",
			IntervalCount:       1,
		}
	}

	// Get supported networks and tokens
	networks, err := h.getSupportedNetworks(ctx, link.WorkspaceID)
	if err == nil {
		response.SupportedNetworks = networks
	}

	tokens, err := h.getAcceptedTokens(ctx, link.WorkspaceID, link.ProductID)
	if err == nil {
		response.AcceptedTokens = tokens
	}

	// Check gas sponsorship
	sponsorship := h.checkGasSponsorship(ctx, link.WorkspaceID)
	if sponsorship != nil {
		response.GasSponsorship = sponsorship
	}

	c.JSON(http.StatusOK, response)
}

// CreatePaymentIntent creates a payment intent for processing
// @Summary Create payment intent
// @Description Create a payment intent to initiate payment processing
// @Tags payment-pages
// @Accept json
// @Produce json
// @Param slug path string true "Payment Link Slug"
// @Param request body PaymentIntentRequest true "Payment intent request"
// @Success 201 {object} PaymentIntentResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payment-pages/{slug}/intent [post]
func (h *PaymentPageHandler) CreatePaymentIntent(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	var req PaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.common.HandleError(c, err, "Invalid request body", http.StatusBadRequest, h.common.logger)
		return
	}

	// Get payment link
	link, err := h.paymentLinkService.GetPaymentLinkBySlug(ctx, slug)
	if err != nil {
		h.common.HandleError(c, err, "Payment link not found or inactive", http.StatusNotFound, h.common.logger)
		return
	}

	// Validate network and token
	_, _ = uuid.Parse(req.NetworkID)
	_, _ = uuid.Parse(req.TokenID)

	// Get workspace wallet address (simplified for now)
	// TODO: Implement proper wallet selection based on network
	_, err = h.common.db.GetWorkspace(ctx, link.WorkspaceID)
	if err != nil {
		h.common.HandleError(c, err, "Failed to get workspace", http.StatusInternalServerError, h.common.logger)
		return
	}
	
	// For now, use a placeholder address
	recipientAddress := "0x0000000000000000000000000000000000000000" // TODO: Get actual wallet address

	// Create payment intent ID
	intentID := uuid.New()

	// TODO: Store payment intent in database/cache for later processing
	// For now, we'll return the intent data directly

	response := PaymentIntentResponse{
		IntentID:         intentID.String(),
		Status:           "pending",
		AmountCents:      *link.AmountCents,
		Currency:         link.Currency,
		NetworkID:        req.NetworkID,
		TokenID:          req.TokenID,
		RecipientAddress: recipientAddress,
		ExpiresAt:        time.Now().Add(15 * time.Minute), // 15 minute expiration
		Metadata:         req.Metadata,
	}

	c.JSON(http.StatusCreated, response)
}

// Helper functions

func (h *PaymentPageHandler) getSupportedNetworks(ctx context.Context, workspaceID uuid.UUID) ([]NetworkData, error) {
	// TODO: Get networks that have wallets configured for this workspace
	// For now, return placeholder data
	return []NetworkData{
		{
			ID:        "00000000-0000-0000-0000-000000000001",
			Name:      "Ethereum",
			ChainID:   "1",
			Type:      "evm",
			IsTestnet: false,
		},
		{
			ID:        "00000000-0000-0000-0000-000000000002",
			Name:      "Polygon",
			ChainID:   "137",
			Type:      "evm",
			IsTestnet: false,
		},
	}, nil
}

func (h *PaymentPageHandler) getAcceptedTokens(ctx context.Context, workspaceID uuid.UUID, productID *uuid.UUID) ([]AcceptedTokenData, error) {
	// TODO: Get tokens configured for this product/workspace
	// For now, return placeholder data
	return []AcceptedTokenData{
		{
			ID:              "00000000-0000-0000-0000-000000000001",
			Symbol:          "USDC",
			Name:            "USD Coin",
			NetworkID:       "00000000-0000-0000-0000-000000000001",
			NetworkName:     "Ethereum",
			ContractAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Decimals:        6,
		},
		{
			ID:              "00000000-0000-0000-0000-000000000002",
			Symbol:          "USDC",
			Name:            "USD Coin",
			NetworkID:       "00000000-0000-0000-0000-000000000002",
			NetworkName:     "Polygon",
			ContractAddress: "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
			Decimals:        6,
		},
	}, nil
}

func (h *PaymentPageHandler) checkGasSponsorship(ctx context.Context, workspaceID uuid.UUID) *GasSponsorshipData {
	// Check if workspace has active gas sponsorship
	// For now, we'll check for workspace-level sponsorship
	_, err := h.common.db.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return nil
	}

	// Check workspace settings for gas sponsorship (simplified for now)
	// TODO: Implement proper gas sponsorship checking
	// For now, always return sponsored for testing
	return &GasSponsorshipData{
		IsSponsored:  true,
		SponsorType:  "workspace",
		CoverageType: "full",
	}
}