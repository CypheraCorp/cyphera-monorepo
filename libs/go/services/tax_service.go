package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TaxService handles comprehensive tax calculation and compliance
type TaxService struct {
	queries db.Querier
	logger  *zap.Logger
}

// NewTaxService creates a new tax service
func NewTaxService(queries db.Querier) *TaxService {
	return &TaxService{
		queries: queries,
		logger:  logger.Log,
	}
}

// TaxCalculationParams contains parameters for tax calculation
type TaxCalculationParams struct {
	WorkspaceID       uuid.UUID
	CustomerID        uuid.UUID
	ProductID         *uuid.UUID
	SubscriptionID    *uuid.UUID
	AmountCents       int64
	Currency          string
	CustomerAddress   *Address
	BusinessAddress   *Address
	TransactionType   string // "subscription", "one_time", "refund"
	ProductType       string // "digital", "physical", "service"
	IsB2B             bool
	CustomerVATNumber *string
	TaxExempt         bool
}

// Address represents a tax-relevant address
type Address struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// TaxCalculationResult contains the calculated tax information
type TaxCalculationResult struct {
	SubtotalCents      int64           `json:"subtotal_cents"`
	TotalTaxCents      int64           `json:"total_tax_cents"`
	TotalAmountCents   int64           `json:"total_amount_cents"`
	TaxBreakdown       []TaxLineItem   `json:"tax_breakdown"`
	AppliedJurisdictions []string      `json:"applied_jurisdictions"`
	TaxExemptReason    *string         `json:"tax_exempt_reason,omitempty"`
	CalculatedAt       time.Time       `json:"calculated_at"`
	Confidence         float64         `json:"confidence"` // 0.0 to 1.0
	AuditTrail         TaxAuditTrail   `json:"audit_trail"`
}

// TaxLineItem represents a single tax component
type TaxLineItem struct {
	TaxType         string  `json:"tax_type"`     // "sales", "vat", "gst", "excise"
	Jurisdiction    string  `json:"jurisdiction"` // "US-CA", "EU-DE", "CA-ON"
	Rate            float64 `json:"rate"`         // e.g., 0.0825 for 8.25%
	TaxableAmount   int64   `json:"taxable_amount_cents"`
	TaxAmountCents  int64   `json:"tax_amount_cents"`
	Description     string  `json:"description"`
	IsReversCharge  bool    `json:"is_reverse_charge"`
}

// TaxAuditTrail contains audit information for tax calculations
type TaxAuditTrail struct {
	RulesVersion     string            `json:"rules_version"`
	DetectedLocation *Address          `json:"detected_location,omitempty"`
	AppliedRules     []string          `json:"applied_rules"`
	Overrides        []TaxOverride     `json:"overrides,omitempty"`
	Notes            []string          `json:"notes,omitempty"`
}

// TaxOverride represents a manual tax override
type TaxOverride struct {
	Reason      string    `json:"reason"`
	OriginalTax int64     `json:"original_tax_cents"`
	NewTax      int64     `json:"new_tax_cents"`
	AppliedBy   uuid.UUID `json:"applied_by"`
	AppliedAt   time.Time `json:"applied_at"`
}

// TaxJurisdiction represents a tax jurisdiction with its rules
type TaxJurisdiction struct {
	Code        string            `json:"code"`
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "country", "state", "province", "city"
	TaxRates    map[string]float64 `json:"tax_rates"` // product_type -> rate
	Thresholds  map[string]int64  `json:"thresholds"` // minimum amounts
	IsActive    bool              `json:"is_active"`
	EffectiveDate time.Time       `json:"effective_date"`
}

// CalculateTax performs comprehensive tax calculation
func (s *TaxService) CalculateTax(ctx context.Context, params TaxCalculationParams) (*TaxCalculationResult, error) {
	s.logger.Info("Calculating tax",
		zap.String("workspace_id", params.WorkspaceID.String()),
		zap.String("customer_id", params.CustomerID.String()),
		zap.Int64("amount_cents", params.AmountCents),
		zap.String("currency", params.Currency),
		zap.Bool("is_b2b", params.IsB2B))

	result := &TaxCalculationResult{
		SubtotalCents:    params.AmountCents,
		TaxBreakdown:     []TaxLineItem{},
		CalculatedAt:     time.Now(),
		Confidence:       1.0,
		AuditTrail: TaxAuditTrail{
			RulesVersion: "v2024.1",
			AppliedRules: []string{},
			Notes:        []string{},
		},
	}

	// Check for tax exemption
	if params.TaxExempt {
		exemptReason := "Customer marked as tax exempt"
		result.TaxExemptReason = &exemptReason
		result.TotalTaxCents = 0
		result.TotalAmountCents = params.AmountCents
		result.AuditTrail.Notes = append(result.AuditTrail.Notes, exemptReason)
		return result, nil
	}

	// Determine tax jurisdiction based on addresses
	jurisdiction, err := s.determineJurisdiction(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to determine jurisdiction: %w", err)
	}

	result.AppliedJurisdictions = []string{jurisdiction.Code}
	result.AuditTrail.DetectedLocation = params.CustomerAddress

	// Handle B2B transactions with reverse charge
	if params.IsB2B && s.shouldApplyReverseCharge(jurisdiction, params) {
		return s.calculateReverseCharge(ctx, params, jurisdiction, result)
	}

	// Calculate standard tax
	return s.calculateStandardTax(ctx, params, jurisdiction, result)
}

// determineJurisdiction determines the applicable tax jurisdiction
func (s *TaxService) determineJurisdiction(ctx context.Context, params TaxCalculationParams) (*TaxJurisdiction, error) {
	// Priority order: Customer address > Business address > Workspace default
	address := params.CustomerAddress
	if address == nil {
		address = params.BusinessAddress
	}

	if address == nil {
		// Use workspace default jurisdiction
		workspace, err := s.queries.GetWorkspace(ctx, params.WorkspaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get workspace: %w", err)
		}
		
		// Parse default tax jurisdiction from workspace
		return s.getDefaultJurisdiction(workspace), nil
	}

	// Determine jurisdiction from address
	return s.getJurisdictionFromAddress(address), nil
}

// shouldApplyReverseCharge determines if reverse charge should be applied for B2B
func (s *TaxService) shouldApplyReverseCharge(jurisdiction *TaxJurisdiction, params TaxCalculationParams) bool {
	// EU reverse charge logic
	if strings.HasPrefix(jurisdiction.Code, "EU-") && params.CustomerVATNumber != nil {
		return s.isValidEUVATNumber(*params.CustomerVATNumber) && 
			   s.isDigitalService(params.ProductType)
	}
	
	// Other jurisdictions with reverse charge rules
	reverseChargeJurisdictions := map[string]bool{
		"UK":  true,
		"AU":  true,
		"CA":  true,
	}
	
	return reverseChargeJurisdictions[jurisdiction.Code] && params.CustomerVATNumber != nil
}

// calculateReverseCharge handles B2B reverse charge scenarios
func (s *TaxService) calculateReverseCharge(ctx context.Context, params TaxCalculationParams, jurisdiction *TaxJurisdiction, result *TaxCalculationResult) (*TaxCalculationResult, error) {
	// In reverse charge, customer pays the tax in their jurisdiction
	taxLineItem := TaxLineItem{
		TaxType:         "vat",
		Jurisdiction:    jurisdiction.Code,
		Rate:            0.0, // Merchant doesn't charge tax
		TaxableAmount:   params.AmountCents,
		TaxAmountCents:  0,
		Description:     "Reverse charge - customer responsible for VAT",
		IsReversCharge:  true,
	}

	result.TaxBreakdown = append(result.TaxBreakdown, taxLineItem)
	result.TotalTaxCents = 0
	result.TotalAmountCents = params.AmountCents
	result.AuditTrail.AppliedRules = append(result.AuditTrail.AppliedRules, "B2B_REVERSE_CHARGE")
	result.AuditTrail.Notes = append(result.AuditTrail.Notes, 
		fmt.Sprintf("Reverse charge applied for VAT number: %s", *params.CustomerVATNumber))

	return result, nil
}

// calculateStandardTax calculates standard tax rates
func (s *TaxService) calculateStandardTax(ctx context.Context, params TaxCalculationParams, jurisdiction *TaxJurisdiction, result *TaxCalculationResult) (*TaxCalculationResult, error) {
	// Get applicable tax rate for product type
	taxRate, exists := jurisdiction.TaxRates[params.ProductType]
	if !exists {
		taxRate = jurisdiction.TaxRates["default"]
	}

	// Check if amount meets minimum threshold
	threshold, hasThreshold := jurisdiction.Thresholds[params.ProductType]
	if hasThreshold && params.AmountCents < threshold {
		taxRate = 0.0
		result.AuditTrail.Notes = append(result.AuditTrail.Notes, 
			fmt.Sprintf("Amount below tax threshold: %d < %d", params.AmountCents, threshold))
	}

	// Calculate tax amount
	taxAmountCents := int64(float64(params.AmountCents) * taxRate)

	// Create tax line item
	taxLineItem := TaxLineItem{
		TaxType:        s.getTaxTypeForJurisdiction(jurisdiction.Code),
		Jurisdiction:   jurisdiction.Code,
		Rate:           taxRate,
		TaxableAmount:  params.AmountCents,
		TaxAmountCents: taxAmountCents,
		Description:    fmt.Sprintf("%s - %s", jurisdiction.Name, params.ProductType),
		IsReversCharge: false,
	}

	result.TaxBreakdown = append(result.TaxBreakdown, taxLineItem)
	result.TotalTaxCents = taxAmountCents
	result.TotalAmountCents = params.AmountCents + taxAmountCents
	result.AuditTrail.AppliedRules = append(result.AuditTrail.AppliedRules, 
		fmt.Sprintf("STANDARD_TAX_%s", jurisdiction.Code))

	return result, nil
}

// StoreTaxCalculation stores tax calculation for audit purposes
func (s *TaxService) StoreTaxCalculation(ctx context.Context, paymentID uuid.UUID, calculation *TaxCalculationResult) error {
	// TODO: Implement once tax_calculations table is created
	s.logger.Info("Tax calculation storage not yet implemented",
		zap.String("payment_id", paymentID.String()),
		zap.Int64("tax_cents", calculation.TotalTaxCents))
	
	return nil
	
	/* Original implementation for when DB is ready:
	auditTrailJSON, err := json.Marshal(calculation.AuditTrail)
	if err != nil {
		return fmt.Errorf("failed to marshal audit trail: %w", err)
	}

	breakdownJSON, err := json.Marshal(calculation.TaxBreakdown)
	if err != nil {
		return fmt.Errorf("failed to marshal tax breakdown: %w", err)
	}

	params := db.CreateTaxCalculationParams{
		PaymentID:         paymentID,
		SubtotalCents:     calculation.SubtotalCents,
		TotalTaxCents:     calculation.TotalTaxCents,
		TotalAmountCents:  calculation.TotalAmountCents,
		TaxBreakdown:      breakdownJSON,
		AuditTrail:        auditTrailJSON,
		CalculatedAt:      pgtype.Timestamptz{Time: calculation.CalculatedAt, Valid: true},
		Confidence:        pgtype.Numeric{}, // Would need proper decimal conversion
	}

	_, err = s.queries.CreateTaxCalculation(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to store tax calculation: %w", err)
	}

	s.logger.Info("Stored tax calculation",
		zap.String("payment_id", paymentID.String()),
		zap.Int64("tax_cents", calculation.TotalTaxCents))

	return nil
	*/
}

// GetTaxRatesForJurisdiction retrieves current tax rates for a jurisdiction
func (s *TaxService) GetTaxRatesForJurisdiction(ctx context.Context, jurisdictionCode string) (*TaxJurisdiction, error) {
	// This would normally query a tax rates database or external service
	return s.getJurisdictionByCode(jurisdictionCode), nil
}

// ValidateVATNumber validates EU VAT numbers
func (s *TaxService) ValidateVATNumber(ctx context.Context, vatNumber, countryCode string) (bool, error) {
	// This is a simplified validation - in production you'd use VIES or similar service
	return s.isValidEUVATNumber(vatNumber), nil
}

// Helper functions

// getDefaultJurisdiction returns default tax jurisdiction for workspace
func (s *TaxService) getDefaultJurisdiction(workspace db.Workspace) *TaxJurisdiction {
	// Parse from workspace configuration or use US as default
	return &TaxJurisdiction{
		Code: "US",
		Name: "United States",
		Type: "country",
		TaxRates: map[string]float64{
			"digital":  0.0,    // No federal tax on digital goods
			"physical": 0.0,    // Varies by state
			"service":  0.0,    // Varies by state  
			"default":  0.0,
		},
		Thresholds: map[string]int64{
			"default": 0, // No minimum threshold
		},
		IsActive:      true,
		EffectiveDate: time.Now(),
	}
}

// getJurisdictionFromAddress determines jurisdiction from address
func (s *TaxService) getJurisdictionFromAddress(address *Address) *TaxJurisdiction {
	// Simplified jurisdiction mapping - in production this would be more comprehensive
	jurisdictions := map[string]*TaxJurisdiction{
		"US": {
			Code: "US-" + address.State,
			Name: fmt.Sprintf("United States - %s", address.State),
			Type: "state",
			TaxRates: s.getUSStateTaxRates(address.State),
			Thresholds: map[string]int64{"default": 0},
			IsActive: true,
			EffectiveDate: time.Now(),
		},
		"CA": {
			Code: "CA-" + address.State,
			Name: fmt.Sprintf("Canada - %s", address.State),
			Type: "province", 
			TaxRates: s.getCanadaTaxRates(address.State),
			Thresholds: map[string]int64{"default": 0},
			IsActive: true,
			EffectiveDate: time.Now(),
		},
		"GB": {
			Code: "UK",
			Name: "United Kingdom",
			Type: "country",
			TaxRates: map[string]float64{
				"digital": 0.20,
				"physical": 0.20,
				"service": 0.20,
				"default": 0.20,
			},
			Thresholds: map[string]int64{"default": 0},
			IsActive: true,
			EffectiveDate: time.Now(),
		},
	}

	if jurisdiction, exists := jurisdictions[address.Country]; exists {
		return jurisdiction
	}

	// Default to no tax for unknown jurisdictions
	return &TaxJurisdiction{
		Code: address.Country,
		Name: address.Country,
		Type: "country",
		TaxRates: map[string]float64{"default": 0.0},
		Thresholds: map[string]int64{"default": 0},
		IsActive: true,
		EffectiveDate: time.Now(),
	}
}

// getUSStateTaxRates returns tax rates for US states
func (s *TaxService) getUSStateTaxRates(state string) map[string]float64 {
	// Simplified state tax rates - in production this would be comprehensive and up-to-date
	stateTaxRates := map[string]map[string]float64{
		"CA": {"digital": 0.0725, "physical": 0.0725, "service": 0.0725, "default": 0.0725},
		"NY": {"digital": 0.08, "physical": 0.08, "service": 0.08, "default": 0.08},
		"TX": {"digital": 0.0625, "physical": 0.0625, "service": 0.0625, "default": 0.0625},
		"FL": {"digital": 0.06, "physical": 0.06, "service": 0.06, "default": 0.06},
	}

	if rates, exists := stateTaxRates[state]; exists {
		return rates
	}

	return map[string]float64{"default": 0.0} // No tax for unknown states
}

// getCanadaTaxRates returns tax rates for Canadian provinces
func (s *TaxService) getCanadaTaxRates(province string) map[string]float64 {
	// Simplified Canadian tax rates (GST + PST/HST)
	provinceTaxRates := map[string]map[string]float64{
		"ON": {"digital": 0.13, "physical": 0.13, "service": 0.13, "default": 0.13}, // HST
		"BC": {"digital": 0.12, "physical": 0.12, "service": 0.12, "default": 0.12}, // GST + PST
		"AB": {"digital": 0.05, "physical": 0.05, "service": 0.05, "default": 0.05}, // GST only
		"QC": {"digital": 0.14975, "physical": 0.14975, "service": 0.14975, "default": 0.14975}, // GST + QST
	}

	if rates, exists := provinceTaxRates[province]; exists {
		return rates
	}

	return map[string]float64{"default": 0.05} // Default to GST only
}

// getJurisdictionByCode retrieves jurisdiction configuration by code
func (s *TaxService) getJurisdictionByCode(code string) *TaxJurisdiction {
	// This would normally query a database or external service
	// For now, return a default based on common codes
	if strings.HasPrefix(code, "US-") {
		state := strings.TrimPrefix(code, "US-")
		return &TaxJurisdiction{
			Code: code,
			Name: fmt.Sprintf("United States - %s", state),
			Type: "state",
			TaxRates: s.getUSStateTaxRates(state),
			Thresholds: map[string]int64{"default": 0},
			IsActive: true,
			EffectiveDate: time.Now(),
		}
	}

	return s.getDefaultJurisdiction(db.Workspace{}) // Return default
}

// getTaxTypeForJurisdiction returns the appropriate tax type for a jurisdiction
func (s *TaxService) getTaxTypeForJurisdiction(code string) string {
	if strings.HasPrefix(code, "US-") {
		return "sales"
	}
	if strings.HasPrefix(code, "CA-") {
		return "gst"
	}
	if strings.HasPrefix(code, "EU-") || code == "UK" {
		return "vat"
	}
	return "tax"
}

// isValidEUVATNumber validates EU VAT number format
func (s *TaxService) isValidEUVATNumber(vatNumber string) bool {
	// Simplified validation - in production use VIES service
	return len(vatNumber) >= 8 && len(vatNumber) <= 12
}

// isDigitalService determines if a product type is a digital service
func (s *TaxService) isDigitalService(productType string) bool {
	digitalTypes := map[string]bool{
		"digital":     true,
		"software":    true,
		"subscription": true,
		"saas":        true,
	}
	return digitalTypes[productType]
}