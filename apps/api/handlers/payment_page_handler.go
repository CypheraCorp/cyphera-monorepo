package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentPageHandler handles public payment page requests
type PaymentPageHandler struct {
	common             *CommonServices
	paymentLinkService interfaces.PaymentLinkService
}

// NewPaymentPageHandler creates a handler with interface dependencies
func NewPaymentPageHandler(
	common *CommonServices,
	paymentLinkService interfaces.PaymentLinkService,
) *PaymentPageHandler {
	return &PaymentPageHandler{
		common:             common,
		paymentLinkService: paymentLinkService,
	}
}

// Use types from the centralized packages
type PaymentPageDataResponse = responses.PaymentPageDataResponse
type PaymentLinkData = responses.PaymentLinkData
type ProductData = responses.ProductData
type PriceData = responses.PriceData
type WorkspaceData = responses.WorkspaceData
type NetworkData = responses.NetworkData
type AcceptedTokenData = responses.AcceptedTokenData
type GasSponsorshipData = responses.GasSponsorshipData
type PaymentIntentRequest = requests.PaymentIntentRequest
type ShippingAddressInput = requests.ShippingAddressInput
type PaymentIntentResponse = responses.PaymentIntentResponse

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
		ExpiresAt:        &[]time.Time{time.Now().Add(15 * time.Minute)}[0], // 15 minute expiration
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
