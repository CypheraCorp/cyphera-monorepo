# Tax Tracking Enhancement Plan

## Current State Analysis

### Issues Identified
1. **Minimal tax support**: Only basic `tax_amount` field in invoices table
2. **No tax configuration**: No way to configure tax rates by jurisdiction
3. **No tax calculation engine**: Tax must be manually calculated
4. **No crypto tax specifics**: No tracking of cost basis, capital gains, etc.
5. **No tax reporting**: No tax summary reports or exports
6. **No multi-jurisdiction support**: Can't handle different tax rates by location
7. **No tax exemptions**: No way to mark customers as tax-exempt

## Tax Requirements for Crypto Subscriptions

### Key Tax Considerations
1. **Sales Tax / VAT**: On subscription services
2. **Digital Services Tax**: Many jurisdictions tax digital services
3. **Crypto-specific taxes**: Some jurisdictions have specific crypto taxes
4. **B2B vs B2C**: Different tax rules apply
5. **Tax residency**: Customer location determines tax jurisdiction
6. **Reverse charge**: B2B EU transactions may use reverse charge
7. **Nexus rules**: Physical/economic presence determines tax obligations

## Enhanced Schema Design

### 1. Tax Configuration Tables

```sql
-- Tax jurisdictions (countries, states, etc.)
CREATE TABLE tax_jurisdictions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Jurisdiction details
    country_code VARCHAR(2) NOT NULL, -- ISO 3166-1 alpha-2
    state_code VARCHAR(10), -- State/province code
    city VARCHAR(255),
    postal_code_pattern VARCHAR(255), -- Regex for postal code validation
    
    -- Jurisdiction info
    name VARCHAR(255) NOT NULL,
    jurisdiction_type VARCHAR(50) NOT NULL, -- 'country', 'state', 'city', 'special'
    parent_jurisdiction_id UUID REFERENCES tax_jurisdictions(id),
    
    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    requires_tax_id BOOLEAN DEFAULT FALSE,
    tax_id_format VARCHAR(255), -- Regex for tax ID validation
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(country_code, state_code, city)
);

CREATE INDEX idx_tax_jurisdictions_country ON tax_jurisdictions(country_code);
CREATE INDEX idx_tax_jurisdictions_active ON tax_jurisdictions(is_active);
```

```sql
-- Tax rates by jurisdiction and type
CREATE TABLE tax_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID REFERENCES workspaces(id), -- NULL for platform-wide rates
    jurisdiction_id UUID NOT NULL REFERENCES tax_jurisdictions(id),
    
    -- Rate details
    tax_type VARCHAR(50) NOT NULL, -- 'sales_tax', 'vat', 'gst', 'digital_services_tax'
    rate_percent DECIMAL(5,4) NOT NULL, -- 0.0000 to 99.9999%
    
    -- Applicability
    applies_to_digital_goods BOOLEAN DEFAULT TRUE,
    applies_to_services BOOLEAN DEFAULT TRUE,
    applies_to_subscriptions BOOLEAN DEFAULT TRUE,
    applies_to_crypto BOOLEAN DEFAULT TRUE,
    
    -- B2B settings
    b2b_reverse_charge BOOLEAN DEFAULT FALSE,
    b2b_exempt BOOLEAN DEFAULT FALSE,
    
    -- Date range for rate validity
    effective_from DATE NOT NULL,
    effective_to DATE,
    
    -- Priority for rate selection
    priority INTEGER DEFAULT 0,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_rate_validity CHECK (effective_to IS NULL OR effective_to > effective_from)
);

CREATE INDEX idx_tax_rates_jurisdiction ON tax_rates(jurisdiction_id);
CREATE INDEX idx_tax_rates_effective ON tax_rates(effective_from, effective_to);
CREATE INDEX idx_tax_rates_workspace ON tax_rates(workspace_id) WHERE workspace_id IS NOT NULL;
```

```sql
-- Tax exemptions for customers
CREATE TABLE customer_tax_exemptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    
    -- Exemption details
    exemption_type VARCHAR(50) NOT NULL, -- 'reseller', 'nonprofit', 'government', 'other'
    exemption_certificate_number VARCHAR(255),
    exemption_certificate_url TEXT, -- Link to stored certificate
    
    -- Jurisdiction specific
    jurisdiction_id UUID REFERENCES tax_jurisdictions(id), -- NULL for all jurisdictions
    tax_types JSONB DEFAULT '[]'::jsonb, -- Array of exempt tax types
    
    -- Validity
    valid_from DATE NOT NULL,
    valid_until DATE,
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Verification
    verified_at TIMESTAMP,
    verified_by UUID REFERENCES accounts(id),
    verification_notes TEXT,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_exemption_validity CHECK (valid_until IS NULL OR valid_until > valid_from)
);

CREATE INDEX idx_customer_exemptions_customer ON customer_tax_exemptions(customer_id);
CREATE INDEX idx_customer_exemptions_active ON customer_tax_exemptions(is_active, valid_from, valid_until);
```

### 2. Enhanced Customer Table

```sql
-- Add tax fields to customers table
ALTER TABLE customers 
ADD COLUMN tax_jurisdiction_id UUID REFERENCES tax_jurisdictions(id),
ADD COLUMN tax_id VARCHAR(255), -- VAT number, EIN, etc.
ADD COLUMN tax_id_type VARCHAR(50), -- 'vat', 'ein', 'gst', etc.
ADD COLUMN tax_id_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN tax_id_verified_at TIMESTAMP,
ADD COLUMN is_business BOOLEAN DEFAULT FALSE,
ADD COLUMN business_name VARCHAR(255),
ADD COLUMN billing_country VARCHAR(2), -- ISO country code
ADD COLUMN billing_state VARCHAR(50),
ADD COLUMN billing_city VARCHAR(255),
ADD COLUMN billing_postal_code VARCHAR(20);

-- Index for tax lookups
CREATE INDEX idx_customers_tax_jurisdiction ON customers(tax_jurisdiction_id);
CREATE INDEX idx_customers_billing_location ON customers(billing_country, billing_state);
```

### 3. Tax Calculation Records

```sql
-- Store tax calculations for audit trail
CREATE TABLE tax_calculations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- References
    invoice_id UUID REFERENCES invoices(id),
    invoice_line_item_id UUID REFERENCES invoice_line_items(id),
    payment_id UUID REFERENCES payments(id),
    subscription_id UUID REFERENCES subscriptions(id),
    
    -- Customer info at time of calculation
    customer_id UUID NOT NULL REFERENCES customers(id),
    customer_jurisdiction_id UUID REFERENCES tax_jurisdictions(id),
    customer_tax_id VARCHAR(255),
    is_b2b BOOLEAN DEFAULT FALSE,
    
    -- Amounts
    subtotal_cents BIGINT NOT NULL,
    taxable_amount_cents BIGINT NOT NULL,
    
    -- Tax details
    tax_jurisdiction_id UUID NOT NULL REFERENCES tax_jurisdictions(id),
    tax_rate_id UUID NOT NULL REFERENCES tax_rates(id),
    tax_type VARCHAR(50) NOT NULL,
    tax_rate DECIMAL(5,4) NOT NULL,
    tax_amount_cents BIGINT NOT NULL,
    
    -- Exemptions
    exemption_applied BOOLEAN DEFAULT FALSE,
    exemption_id UUID REFERENCES customer_tax_exemptions(id),
    exemption_reason VARCHAR(255),
    
    -- Calculation metadata
    calculation_method VARCHAR(50) NOT NULL, -- 'inclusive', 'exclusive'
    calculation_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    calculation_service VARCHAR(50), -- 'internal', 'stripe_tax', 'taxjar', etc.
    
    -- For crypto transactions
    crypto_amount DECIMAL(36,18),
    crypto_token_id UUID REFERENCES tokens(id),
    exchange_rate_at_time DECIMAL(20,8),
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tax_calculations_invoice ON tax_calculations(invoice_id);
CREATE INDEX idx_tax_calculations_customer ON tax_calculations(customer_id);
CREATE INDEX idx_tax_calculations_timestamp ON tax_calculations(calculation_timestamp);
```

### 4. Tax Summary Tables

```sql
-- Monthly tax summary for reporting
CREATE TABLE tax_summaries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Period
    tax_period_year INTEGER NOT NULL,
    tax_period_month INTEGER NOT NULL,
    
    -- Jurisdiction
    tax_jurisdiction_id UUID NOT NULL REFERENCES tax_jurisdictions(id),
    tax_type VARCHAR(50) NOT NULL,
    
    -- Amounts
    gross_sales_cents BIGINT NOT NULL,
    taxable_sales_cents BIGINT NOT NULL,
    exempt_sales_cents BIGINT NOT NULL,
    tax_collected_cents BIGINT NOT NULL,
    
    -- Counts
    transaction_count INTEGER NOT NULL,
    customer_count INTEGER NOT NULL,
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- 'draft', 'final', 'filed'
    filed_at TIMESTAMP,
    filing_reference VARCHAR(255),
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(workspace_id, tax_period_year, tax_period_month, tax_jurisdiction_id, tax_type)
);

CREATE INDEX idx_tax_summaries_period ON tax_summaries(tax_period_year, tax_period_month);
CREATE INDEX idx_tax_summaries_workspace ON tax_summaries(workspace_id, status);
```

### 5. Update Invoice and Payment Tables

```sql
-- Enhance invoices table
ALTER TABLE invoices
ADD COLUMN subtotal_cents BIGINT NOT NULL DEFAULT 0,
ADD COLUMN tax_amount_cents BIGINT NOT NULL DEFAULT 0,
ADD COLUMN tax_details JSONB DEFAULT '[]'::jsonb, -- Array of tax calculations
ADD COLUMN customer_tax_id VARCHAR(255),
ADD COLUMN customer_jurisdiction_id UUID REFERENCES tax_jurisdictions(id),
ADD COLUMN reverse_charge_applies BOOLEAN DEFAULT FALSE;

-- Enhance payments table
ALTER TABLE payments
ADD COLUMN subtotal_cents BIGINT,
ADD COLUMN tax_amount_cents BIGINT,
ADD COLUMN tax_included_in_amount BOOLEAN DEFAULT TRUE;

-- Enhance invoice_line_items
ALTER TABLE invoice_line_items
ADD COLUMN is_taxable BOOLEAN DEFAULT TRUE,
ADD COLUMN tax_calculation_id UUID REFERENCES tax_calculations(id);
```

## Tax Calculation Service

### Architecture

```go
// services/tax_service.go
type TaxService struct {
    db             *db.Queries
    jurisdictions  *TaxJurisdictionService
    externalAPI    TaxProviderInterface // Stripe Tax, TaxJar, etc.
}

type TaxCalculationRequest struct {
    CustomerID      uuid.UUID
    LineItems       []LineItem
    ShippingAddress *Address
    BillingAddress  *Address
    TaxExemptionID  *uuid.UUID
    Currency        string
}

type TaxCalculationResponse struct {
    Calculations []TaxCalculation
    TotalTax     int64
    Breakdown    map[string]TaxBreakdown
}

func (s *TaxService) CalculateTax(ctx context.Context, req TaxCalculationRequest) (*TaxCalculationResponse, error) {
    // 1. Determine customer jurisdiction
    jurisdiction := s.determineJurisdiction(ctx, req.CustomerID, req.BillingAddress)
    
    // 2. Check exemptions
    exemptions := s.checkExemptions(ctx, req.CustomerID, jurisdiction.ID)
    
    // 3. Get applicable tax rates
    rates := s.getApplicableRates(ctx, jurisdiction.ID, time.Now())
    
    // 4. Calculate tax for each line item
    calculations := s.calculateLineItemTaxes(ctx, req.LineItems, rates, exemptions)
    
    // 5. Store calculations for audit
    s.storeCalculations(ctx, calculations)
    
    return &TaxCalculationResponse{
        Calculations: calculations,
        TotalTax:     s.sumTaxes(calculations),
        Breakdown:    s.createBreakdown(calculations),
    }, nil
}
```

### Tax Jurisdiction Detection

```go
func (s *TaxService) determineJurisdiction(
    ctx context.Context, 
    customerID uuid.UUID,
    address *Address,
) (*TaxJurisdiction, error) {
    // Priority order:
    // 1. Explicit customer tax jurisdiction
    // 2. Billing address
    // 3. IP geolocation (for digital goods)
    // 4. Default workspace jurisdiction
    
    customer, _ := s.db.GetCustomer(ctx, customerID)
    if customer.TaxJurisdictionID != nil {
        return s.db.GetTaxJurisdiction(ctx, *customer.TaxJurisdictionID)
    }
    
    if address != nil {
        return s.jurisdictions.FindByAddress(ctx, address)
    }
    
    // Fallback logic...
}
```

## API Endpoints for Tax Management

### Tax Configuration

```go
// Tax jurisdictions
taxes := protected.Group("/taxes")
{
    // Jurisdictions
    taxes.GET("/jurisdictions", taxHandler.ListJurisdictions)
    taxes.GET("/jurisdictions/search", taxHandler.SearchJurisdictions)
    taxes.GET("/jurisdictions/:id", taxHandler.GetJurisdiction)
    
    // Tax rates
    taxes.GET("/rates", taxHandler.ListTaxRates)
    taxes.POST("/rates", taxHandler.CreateTaxRate)
    taxes.PUT("/rates/:id", taxHandler.UpdateTaxRate)
    taxes.DELETE("/rates/:id", taxHandler.DeleteTaxRate)
    
    // Customer exemptions
    taxes.GET("/exemptions", taxHandler.ListExemptions)
    taxes.POST("/exemptions", taxHandler.CreateExemption)
    taxes.PUT("/exemptions/:id", taxHandler.UpdateExemption)
    taxes.DELETE("/exemptions/:id", taxHandler.DeleteExemption)
    taxes.POST("/exemptions/:id/verify", taxHandler.VerifyExemption)
    
    // Tax calculation
    taxes.POST("/calculate", taxHandler.CalculateTax)
    taxes.POST("/validate-tax-id", taxHandler.ValidateTaxID)
    
    // Reporting
    taxes.GET("/summaries", taxHandler.ListTaxSummaries)
    taxes.GET("/summaries/:id", taxHandler.GetTaxSummary)
    taxes.POST("/summaries/:id/finalize", taxHandler.FinalizeTaxSummary)
    taxes.GET("/export", taxHandler.ExportTaxData)
}
```

## Frontend Components

### Tax Configuration UI

```typescript
// Tax exemption management
function TaxExemptionForm({ customer }) {
  const [exemptionType, setExemptionType] = useState('');
  const [certificateNumber, setCertificateNumber] = useState('');
  const [jurisdictions, setJurisdictions] = useState([]);
  
  const handleSubmit = async () => {
    await fetch(`/api/taxes/exemptions`, {
      method: 'POST',
      body: JSON.stringify({
        customer_id: customer.id,
        exemption_type: exemptionType,
        exemption_certificate_number: certificateNumber,
        jurisdiction_ids: jurisdictions,
      }),
    });
  };
  
  return (
    <Card>
      <CardHeader>
        <CardTitle>Tax Exemption Certificate</CardTitle>
      </CardHeader>
      <CardContent>
        <Select value={exemptionType} onValueChange={setExemptionType}>
          <SelectItem value="reseller">Reseller Certificate</SelectItem>
          <SelectItem value="nonprofit">Non-profit Exemption</SelectItem>
          <SelectItem value="government">Government Entity</SelectItem>
        </Select>
        {/* Additional fields... */}
      </CardContent>
    </Card>
  );
}
```

### Invoice Display with Tax Breakdown

```typescript
function InvoiceTaxBreakdown({ invoice }) {
  const taxBreakdown = invoice.tax_details || [];
  
  return (
    <div className="space-y-2">
      <div className="flex justify-between">
        <span>Subtotal</span>
        <span>{formatCurrency(invoice.subtotal_cents / 100)}</span>
      </div>
      
      {taxBreakdown.map((tax) => (
        <div key={tax.id} className="flex justify-between text-sm">
          <span>{tax.tax_type} ({tax.jurisdiction_name})</span>
          <span>{formatCurrency(tax.amount_cents / 100)}</span>
        </div>
      ))}
      
      {invoice.reverse_charge_applies && (
        <div className="text-sm text-muted-foreground">
          * Reverse charge applies - Customer is responsible for tax
        </div>
      )}
      
      <div className="flex justify-between font-bold">
        <span>Total</span>
        <span>{formatCurrency(invoice.total_cents / 100)}</span>
      </div>
    </div>
  );
}
```

## Integration Strategy

### 1. Third-party Tax Services

```go
// Support for external tax calculation services
type TaxProviderInterface interface {
    CalculateTax(ctx context.Context, req TaxCalculationRequest) (*TaxCalculationResponse, error)
    ValidateTaxID(ctx context.Context, taxID string, country string) (bool, error)
    GetJurisdictions(ctx context.Context, address Address) ([]TaxJurisdiction, error)
}

// Implementations
type StripeTaxProvider struct { /* ... */ }
type TaxJarProvider struct { /* ... */ }
type InternalTaxProvider struct { /* ... */ } // Fallback
```

### 2. Automated Tax Updates

```sql
-- Scheduled job to update tax rates
CREATE OR REPLACE FUNCTION update_tax_rates_from_provider() RETURNS void AS $$
BEGIN
    -- Call external API to get latest rates
    -- Update tax_rates table
    -- Log changes for audit
END;
$$ LANGUAGE plpgsql;

-- Run monthly
SELECT cron.schedule('update-tax-rates', '0 0 1 * *', 'SELECT update_tax_rates_from_provider()');
```

## Compliance Features

### 1. Tax Reporting

```sql
-- View for tax reporting
CREATE VIEW tax_report_summary AS
SELECT 
    ts.workspace_id,
    ts.tax_period_year,
    ts.tax_period_month,
    tj.country_code,
    tj.state_code,
    tj.name as jurisdiction_name,
    ts.tax_type,
    ts.gross_sales_cents,
    ts.taxable_sales_cents,
    ts.exempt_sales_cents,
    ts.tax_collected_cents,
    ts.transaction_count,
    ts.status,
    ts.filed_at
FROM tax_summaries ts
JOIN tax_jurisdictions tj ON tj.id = ts.tax_jurisdiction_id
ORDER BY ts.tax_period_year DESC, ts.tax_period_month DESC;
```

### 2. Audit Trail

```sql
-- Tax audit log
CREATE TABLE tax_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Event details
    event_type VARCHAR(100) NOT NULL, -- 'rate_change', 'exemption_added', 'calculation_override'
    entity_type VARCHAR(50) NOT NULL, -- 'tax_rate', 'exemption', 'calculation'
    entity_id UUID NOT NULL,
    
    -- Changes
    old_values JSONB,
    new_values JSONB,
    
    -- Actor
    performed_by UUID REFERENCES accounts(id),
    performed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Context
    reason TEXT,
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_tax_audit_workspace ON tax_audit_log(workspace_id, performed_at DESC);
```

## Migration Strategy

1. **Phase 1: Basic Tax Support**
   - Add tax configuration tables
   - Implement basic tax calculation
   - Update invoice generation

2. **Phase 2: Advanced Features**
   - Customer exemptions
   - Multi-jurisdiction support
   - B2B reverse charge

3. **Phase 3: Compliance & Reporting**
   - Tax summaries
   - Export functionality
   - Audit trails

4. **Phase 4: External Integrations**
   - Stripe Tax integration
   - TaxJar integration
   - Automated rate updates

## Benefits

1. **Compliance**: Meet tax obligations in multiple jurisdictions
2. **Accuracy**: Automated tax calculations reduce errors
3. **Flexibility**: Support for various tax scenarios
4. **Auditability**: Complete tax calculation history
5. **Customer Experience**: Clear tax breakdowns on invoices
6. **B2B Support**: Handle reverse charge and exemptions
7. **Reporting**: Built-in tax reports for filing