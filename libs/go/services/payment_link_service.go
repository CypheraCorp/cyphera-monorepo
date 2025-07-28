package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
)

// PaymentLinkService handles payment link creation and management
type PaymentLinkService struct {
	queries *db.Queries
	logger  *zap.Logger
	baseURL string // Base URL for payment links (e.g., https://pay.cyphera.com)
}

// NewPaymentLinkService creates a new payment link service
func NewPaymentLinkService(queries *db.Queries, logger *zap.Logger, baseURL string) *PaymentLinkService {
	if baseURL == "" {
		baseURL = "https://pay.cyphera.com"
	}
	return &PaymentLinkService{
		queries: queries,
		logger:  logger,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// GetBaseURL returns the base URL for payment links
func (s *PaymentLinkService) GetBaseURL() string {
	return s.baseURL
}



// CreatePaymentLink creates a new payment link
func (s *PaymentLinkService) CreatePaymentLink(ctx context.Context, params params.PaymentLinkCreateParams) (*responses.PaymentLinkResponse, error) {
	// Generate unique slug
	slug, err := s.generateUniqueSlug(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique slug: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create payment link
	link, err := s.queries.CreatePaymentLink(ctx, db.CreatePaymentLinkParams{
		WorkspaceID:      params.WorkspaceID,
		Slug:             slug,
		Status:           "active",
		ProductID:        uuidToPgtypePaymentLink(params.ProductID),
		PriceID:          uuidToPgtypePaymentLink(params.PriceID),
		AmountInCents:    int64ToPgtype(&params.AmountCents),
		Currency:         pgtype.Text{String: params.Currency, Valid: params.Currency != ""},
		PaymentType:      pgtype.Text{String: "one_time", Valid: true}, // Default payment type
		CollectEmail:     pgtype.Bool{Bool: params.RequireCustomerInfo, Valid: true},
		CollectShipping:  pgtype.Bool{Bool: false, Valid: true},
		CollectName:      pgtype.Bool{Bool: params.RequireCustomerInfo, Valid: true},
		ExpiresAt:        stringToPgtypeTimestamp(params.ExpiresAt),
		MaxUses:          int32ToPgtype(params.MaxRedemptions),
		RedirectUrl:      stringToPgtype(params.RedirectURL),
		QrCodeUrl:        pgtype.Text{Valid: false}, // Will be set after QR code generation
		Metadata:         metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment link: %w", err)
	}

	// Generate payment URL
	paymentURL := fmt.Sprintf("%s/pay/%s", s.baseURL, slug)

	// Generate QR code
	qrCodeData, err := s.GenerateQRCode(ctx, paymentURL)
	if err != nil {
		s.logger.Error("Failed to generate QR code", zap.Error(err))
		// Don't fail the whole operation if QR code generation fails
	} else {
		// Update payment link with QR code URL
		link, err = s.queries.UpdatePaymentLinkQRCode(ctx, db.UpdatePaymentLinkQRCodeParams{
			ID:          link.ID,
			WorkspaceID: link.WorkspaceID,
			QrCodeUrl:   pgtype.Text{String: qrCodeData, Valid: true},
		})
		if err != nil {
			s.logger.Error("Failed to update QR code URL", zap.Error(err))
		}
	}

	return s.convertToResponse(link, paymentURL, qrCodeData), nil
}

// GetPaymentLink retrieves a payment link by ID
func (s *PaymentLinkService) GetPaymentLink(ctx context.Context, workspaceID, linkID uuid.UUID) (*responses.PaymentLinkResponse, error) {
	link, err := s.queries.GetPaymentLink(ctx, db.GetPaymentLinkParams{
		ID:          linkID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get payment link: %w", err)
	}

	paymentURL := fmt.Sprintf("%s/pay/%s", s.baseURL, link.Slug)
	return s.convertToResponse(link, paymentURL, link.QrCodeUrl.String), nil
}

// GetPaymentLinkBySlug retrieves a payment link by slug (for payment processing)
func (s *PaymentLinkService) GetPaymentLinkBySlug(ctx context.Context, slug string) (*responses.PaymentLinkResponse, error) {
	link, err := s.queries.GetActivePaymentLinkBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("payment link not found or inactive: %w", err)
	}

	paymentURL := fmt.Sprintf("%s/pay/%s", s.baseURL, link.Slug)
	return s.convertToResponse(link, paymentURL, link.QrCodeUrl.String), nil
}

// UpdatePaymentLink updates a payment link
func (s *PaymentLinkService) UpdatePaymentLink(ctx context.Context, workspaceID, linkID uuid.UUID, updates params.PaymentLinkUpdateParams) (*responses.PaymentLinkResponse, error) {
	// Convert metadata to JSON if provided
	var metadataJSON []byte
	if updates.Metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(updates.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Set status with default
	status := "active"
	if updates.IsActive != nil && !*updates.IsActive {
		status = "inactive"
	}

	link, err := s.queries.UpdatePaymentLink(ctx, db.UpdatePaymentLinkParams{
		ID:          linkID,
		WorkspaceID: workspaceID,
		Status:      status,
		ExpiresAt:   stringToPgtypeTimestamp(updates.ExpiresAt),
		MaxUses:     int32ToPgtype(updates.MaxRedemptions),
		RedirectUrl: stringToPgtype(updates.RedirectURL),
		Metadata:    metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update payment link: %w", err)
	}

	paymentURL := fmt.Sprintf("%s/pay/%s", s.baseURL, link.Slug)
	return s.convertToResponse(link, paymentURL, link.QrCodeUrl.String), nil
}

// PaymentLinkUpdateParams contains parameters for updating a payment link
type PaymentLinkUpdateParams struct {
	Status      *string
	ExpiresAt   *time.Time
	MaxUses     *int32
	RedirectURL *string
	Metadata    map[string]interface{}
}

// DeactivatePaymentLink deactivates a payment link
func (s *PaymentLinkService) DeactivatePaymentLink(ctx context.Context, workspaceID, linkID uuid.UUID) error {
	_, err := s.queries.DeactivatePaymentLink(ctx, db.DeactivatePaymentLinkParams{
		ID:          linkID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to deactivate payment link: %w", err)
	}
	return nil
}

// RecordPaymentLinkUsage increments the usage count for a payment link
func (s *PaymentLinkService) RecordPaymentLinkUsage(ctx context.Context, workspaceID, linkID uuid.UUID) error {
	_, err := s.queries.IncrementPaymentLinkUsage(ctx, db.IncrementPaymentLinkUsageParams{
		ID:          linkID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return fmt.Errorf("failed to record payment link usage: %w", err)
	}
	return nil
}

// ExpireStalePaymentLinks marks expired payment links as expired
func (s *PaymentLinkService) ExpireStalePaymentLinks(ctx context.Context) error {
	err := s.queries.ExpirePaymentLinks(ctx)
	if err != nil {
		return fmt.Errorf("failed to expire payment links: %w", err)
	}
	return nil
}

// GenerateQRCode generates a QR code for a payment link
func (s *PaymentLinkService) GenerateQRCode(ctx context.Context, paymentURL string) (string, error) {
	// Generate QR code as PNG
	qr, err := qrcode.New(paymentURL, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to create QR code: %w", err)
	}

	// Convert to PNG bytes
	pngBytes, err := qr.PNG(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate PNG: %w", err)
	}

	// Encode as base64 data URL
	dataURL := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(pngBytes))
	
	return dataURL, nil
}

// CreatePaymentLinkForInvoice creates a payment link specifically for an invoice
func (s *PaymentLinkService) CreatePaymentLinkForInvoice(ctx context.Context, invoice db.Invoice) (*responses.PaymentLinkResponse, error) {
	// Extract amount and currency from invoice
	amountCents := int64(invoice.AmountDue)
	
	// Create payment link with invoice metadata
	return s.CreatePaymentLink(ctx, params.PaymentLinkCreateParams{
		WorkspaceID:         invoice.WorkspaceID,
		AmountCents:         amountCents,
		Currency:            invoice.Currency,
		RequireCustomerInfo: true,
		InvoiceID:           &invoice.ID,
		Metadata: map[string]interface{}{
			"invoice_id": invoice.ID.String(),
			"type":       "invoice_payment",
		},
	})
}

// Helper functions

func (s *PaymentLinkService) generateUniqueSlug(ctx context.Context) (string, error) {
	const maxAttempts = 10
	
	for i := 0; i < maxAttempts; i++ {
		// Generate random bytes
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}
		
		// Convert to URL-safe base64
		slug := base64.URLEncoding.EncodeToString(b)
		slug = strings.TrimRight(slug, "=") // Remove padding
		slug = strings.ToLower(slug)
		
		// Check if slug exists
		exists, err := s.queries.CheckSlugExists(ctx, slug)
		if err != nil {
			return "", fmt.Errorf("failed to check slug existence: %w", err)
		}
		
		if !exists {
			return slug, nil
		}
	}
	
	return "", fmt.Errorf("failed to generate unique slug after %d attempts", maxAttempts)
}

func (s *PaymentLinkService) convertToResponse(link db.PaymentLink, paymentURL string, qrCodeData string) *responses.PaymentLinkResponse {
	response := &responses.PaymentLinkResponse{
		ID:              link.ID,
		WorkspaceID:     link.WorkspaceID,
		Slug:            link.Slug,
		URL:             paymentURL,
		Status:          link.Status,
		Currency:        link.Currency.String,
		PaymentType:     link.PaymentType.String,
		CollectEmail:    link.CollectEmail.Bool,
		CollectShipping: link.CollectShipping.Bool,
		CollectName:     link.CollectName.Bool,
		UsedCount:       link.UsedCount.Int32,
		CreatedAt:       link.CreatedAt.Time,
		UpdatedAt:       link.UpdatedAt.Time,
	}

	// Set optional fields
	if link.ProductID.Valid {
		id := uuid.UUID(link.ProductID.Bytes)
		response.ProductID = &id
	}
	if link.PriceID.Valid {
		id := uuid.UUID(link.PriceID.Bytes)
		response.PriceID = &id
	}
	if link.AmountInCents.Valid {
		response.AmountCents = &link.AmountInCents.Int64
	}
	if link.ExpiresAt.Valid {
		response.ExpiresAt = &link.ExpiresAt.Time
	}
	if link.MaxUses.Valid {
		response.MaxUses = &link.MaxUses.Int32
	}
	if link.RedirectUrl.Valid {
		response.RedirectURL = &link.RedirectUrl.String
	}
	if link.QrCodeUrl.Valid {
		response.QRCodeURL = &link.QrCodeUrl.String
		response.QRCodeData = &qrCodeData
	}

	// Parse metadata
	if len(link.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(link.Metadata, &metadata); err == nil {
			response.Metadata = metadata
		}
	}

	return response
}

// Utility functions

func uuidToPgtypePaymentLink(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func stringToPgtype(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func int32ToPgtype(i *int32) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *i, Valid: true}
}

func int64ToPgtype(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

func timeToPgtypePaymentLink(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func stringToPgtypeTimestamp(s *string) pgtype.Timestamptz {
	if s == nil || *s == "" {
		return pgtype.Timestamptz{Valid: false}
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}