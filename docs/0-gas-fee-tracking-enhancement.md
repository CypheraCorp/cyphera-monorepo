# Gas Fee Tracking Enhancement Plan

## Current State Analysis

### Issues Identified
1. **subscription_events table**: No gas fee tracking
2. **payments table**: Has `gas_fee_in_cents` but no sponsorship tracking
3. **invoice_line_items**: No separation of product amount vs gas fees
4. **No clear distinction between**: Customer-paid gas vs merchant-sponsored gas

## Enhanced Schema Design

### 1. Update payments table

```sql
-- Modify payments table to track gas fees comprehensively
ALTER TABLE payments 
ADD COLUMN gas_fee_wei TEXT, -- Store exact gas in wei for precision
ADD COLUMN gas_fee_token_id UUID REFERENCES tokens(id), -- Which token paid for gas
ADD COLUMN gas_fee_token_amount TEXT, -- Exact token amount for gas
ADD COLUMN gas_sponsor_type VARCHAR(50), -- 'customer', 'merchant', 'platform'
ADD COLUMN gas_sponsor_id UUID, -- Who sponsored (customer_id, workspace_id, etc)
ADD COLUMN gas_price_gwei TEXT, -- Gas price at time of transaction
ADD COLUMN gas_units_used BIGINT, -- Actual gas units consumed
ADD COLUMN max_gas_units BIGINT; -- Max gas units set for transaction

-- Add index for gas analytics
CREATE INDEX idx_payments_gas_sponsor ON payments(gas_sponsor_type, gas_sponsor_id);
CREATE INDEX idx_payments_gas_fee ON payments(gas_fee_in_cents) WHERE gas_fee_in_cents > 0;
```

### 2. Add gas_fee_payments table for detailed tracking

```sql
CREATE TABLE gas_fee_payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    
    -- Gas details
    gas_fee_wei TEXT NOT NULL, -- Exact gas fee in wei
    gas_price_gwei TEXT NOT NULL, -- Gas price at execution
    gas_units_used BIGINT NOT NULL, -- Actual gas consumed
    max_gas_units BIGINT NOT NULL, -- Gas limit set
    base_fee_gwei TEXT, -- EIP-1559 base fee
    priority_fee_gwei TEXT, -- EIP-1559 priority fee
    
    -- Payment details
    payment_token_id UUID REFERENCES tokens(id), -- Token used to pay gas
    payment_token_amount TEXT, -- Amount of token used
    payment_method VARCHAR(50) NOT NULL, -- 'native', 'relay', 'meta_transaction'
    
    -- Sponsorship
    sponsor_type VARCHAR(50) NOT NULL, -- 'customer', 'merchant', 'platform', 'third_party'
    sponsor_id UUID, -- References appropriate entity
    sponsor_workspace_id UUID REFERENCES workspaces(id),
    sponsorship_agreement_id UUID, -- Future: link to sponsorship agreements
    
    -- Network specifics
    network_id UUID NOT NULL REFERENCES networks(id),
    block_number BIGINT,
    block_timestamp TIMESTAMP,
    
    -- Conversion rates at time of payment
    eth_usd_price DECIMAL(10, 2), -- ETH price in USD
    token_usd_price DECIMAL(10, 2), -- Gas token price in USD
    gas_fee_usd_cents BIGINT, -- Calculated USD value
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_payment_gas FOREIGN KEY (payment_id) 
        REFERENCES payments(id) ON DELETE CASCADE
);

CREATE INDEX idx_gas_fee_payments_sponsor ON gas_fee_payments(sponsor_type, sponsor_id);
CREATE INDEX idx_gas_fee_payments_created ON gas_fee_payments(created_at);
```

### 3. Update invoice_line_items for gas separation

```sql
ALTER TABLE invoice_line_items 
ADD COLUMN line_item_type VARCHAR(50) DEFAULT 'product', -- 'product', 'gas_fee', 'tax', 'discount'
ADD COLUMN gas_fee_payment_id UUID REFERENCES gas_fee_payments(id),
ADD COLUMN is_gas_sponsored BOOLEAN DEFAULT FALSE,
ADD COLUMN gas_sponsor_type VARCHAR(50),
ADD COLUMN gas_sponsor_name VARCHAR(255); -- Human readable sponsor name

-- Add constraint to ensure gas line items have proper references
ALTER TABLE invoice_line_items 
ADD CONSTRAINT chk_gas_line_item 
CHECK (
    (line_item_type != 'gas_fee') OR 
    (line_item_type = 'gas_fee' AND gas_fee_payment_id IS NOT NULL)
);
```

### 4. Add gas sponsorship configuration

```sql
CREATE TABLE gas_sponsorship_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Sponsorship settings
    sponsorship_enabled BOOLEAN DEFAULT FALSE,
    sponsor_customer_gas BOOLEAN DEFAULT FALSE, -- Merchant sponsors customer gas
    sponsor_threshold_usd_cents BIGINT, -- Max sponsorship per transaction
    monthly_budget_usd_cents BIGINT, -- Monthly sponsorship budget
    
    -- Rules
    sponsor_for_products JSONB DEFAULT '[]'::jsonb, -- Array of product IDs
    sponsor_for_customers JSONB DEFAULT '[]'::jsonb, -- Array of customer IDs
    sponsor_for_tiers JSONB DEFAULT '[]'::jsonb, -- Customer tiers eligible
    
    -- Tracking
    current_month_spent_cents BIGINT DEFAULT 0,
    last_reset_date DATE,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(workspace_id)
);
```

### 5. Update subscription_events to track gas

```sql
ALTER TABLE subscription_events
ADD COLUMN gas_fee_wei TEXT,
ADD COLUMN gas_fee_usd_cents BIGINT,
ADD COLUMN gas_sponsored BOOLEAN DEFAULT FALSE,
ADD COLUMN gas_sponsor_type VARCHAR(50);
```

## Implementation Details

### 1. Gas Fee Calculation Service

```go
// services/gas_fee_service.go
type GasFeeService struct {
    db          *db.Queries
    priceClient *coinmarketcap.Client
}

type GasFeeSummary struct {
    GasFeeWei         string
    GasPriceGwei      string
    GasUnitsUsed      int64
    GasFeeUSDCents    int64
    PaymentToken      *db.Token
    PaymentAmount     string
    SponsorType       string
    SponsorID         *uuid.UUID
    IsSponsoredByUs   bool
}

func (s *GasFeeService) CalculateGasFee(
    ctx context.Context,
    txHash string,
    networkID uuid.UUID,
) (*GasFeeSummary, error) {
    // 1. Get transaction receipt from blockchain
    // 2. Extract gas used and gas price
    // 3. Calculate total gas fee in wei
    // 4. Get current ETH/token price
    // 5. Convert to USD
    // 6. Determine sponsorship
    return summary, nil
}

func (s *GasFeeService) DetermineSponsor(
    ctx context.Context,
    workspaceID uuid.UUID,
    customerID uuid.UUID,
    productID *uuid.UUID,
    gasFeeUSDCents int64,
) (sponsorType string, sponsorID *uuid.UUID, error) {
    // Check sponsorship configuration
    config, err := s.db.GetGasSponsorshipConfig(ctx, workspaceID)
    if err != nil || !config.SponsorshipEnabled {
        return "customer", &customerID, nil
    }
    
    // Check if under threshold
    if gasFeeUSDCents > config.SponsorThresholdUsdCents {
        return "customer", &customerID, nil
    }
    
    // Check monthly budget
    if config.CurrentMonthSpentCents + gasFeeUSDCents > config.MonthlyBudgetUsdCents {
        return "customer", &customerID, nil
    }
    
    // Check rules...
    
    return "merchant", &workspaceID, nil
}
```

### 2. Invoice Generation with Gas Fees

```go
// When creating invoice line items
func (h *InvoiceHandler) AddGasLineItem(
    ctx context.Context,
    invoiceID uuid.UUID,
    payment *db.Payment,
    gasFeePayment *db.GasFeePayment,
) error {
    // Determine how to display gas fee
    description := "Network Transaction Fee"
    if gasFeePayment.SponsorType == "merchant" {
        description = "Network Transaction Fee (Sponsored)"
    }
    
    lineItem := db.CreateInvoiceLineItemParams{
        InvoiceID:        invoiceID,
        ProductID:        nil, // No product for gas fees
        Description:      description,
        Quantity:         1,
        UnitAmountInCents: gasFeePayment.GasFeeUsdCents,
        LineItemType:     "gas_fee",
        GasFeePaymentID:  &gasFeePayment.ID,
        IsGasSponsored:   gasFeePayment.SponsorType != "customer",
        GasSponsorType:   &gasFeePayment.SponsorType,
    }
    
    // If sponsored, set amount to 0 for customer
    if gasFeePayment.SponsorType != "customer" {
        lineItem.UnitAmountInCents = 0
        lineItem.GasSponsorName = ptr("Paid by " + gasFeePayment.SponsorType)
    }
    
    return h.db.CreateInvoiceLineItem(ctx, lineItem)
}
```

### 3. Payment Processing with Gas Tracking

```go
// In payment handler
func (h *PaymentHandler) ProcessPayment(ctx context.Context, req PaymentRequest) error {
    // ... existing payment logic ...
    
    // After transaction is confirmed
    if txHash != "" {
        gasFee, err := h.gasFeeService.CalculateGasFee(ctx, txHash, payment.NetworkID)
        if err != nil {
            logger.Error("Failed to calculate gas fee", zap.Error(err))
            // Don't fail payment, just log
        } else {
            // Update payment with gas details
            err = h.db.UpdatePaymentGasDetails(ctx, db.UpdatePaymentGasDetailsParams{
                ID:                payment.ID,
                GasFeeWei:        &gasFee.GasFeeWei,
                GasFeeInCents:    &gasFee.GasFeeUSDCents,
                GasSponsorType:   &gasFee.SponsorType,
                GasSponsorID:     gasFee.SponsorID,
            })
            
            // Create detailed gas fee record
            _, err = h.db.CreateGasFeePayment(ctx, db.CreateGasFeePaymentParams{
                PaymentID:         payment.ID,
                GasFeeWei:        gasFee.GasFeeWei,
                GasPriceGwei:     gasFee.GasPriceGwei,
                GasUnitsUsed:     gasFee.GasUnitsUsed,
                // ... other fields
            })
        }
    }
}
```

### 4. Dashboard Metrics for Gas

```sql
-- Add to dashboard_metrics table
ALTER TABLE dashboard_metrics
ADD COLUMN total_gas_fees_cents BIGINT DEFAULT 0,
ADD COLUMN sponsored_gas_fees_cents BIGINT DEFAULT 0,
ADD COLUMN customer_gas_fees_cents BIGINT DEFAULT 0,
ADD COLUMN avg_gas_fee_cents BIGINT DEFAULT 0,
ADD COLUMN gas_sponsorship_rate DECIMAL(5, 2); -- Percentage sponsored

-- View for gas analytics
CREATE VIEW gas_fee_analytics AS
SELECT 
    workspace_id,
    DATE_TRUNC('day', created_at) as date,
    COUNT(*) as transaction_count,
    SUM(gas_fee_usd_cents) as total_gas_cents,
    SUM(CASE WHEN sponsor_type = 'merchant' THEN gas_fee_usd_cents ELSE 0 END) as sponsored_cents,
    SUM(CASE WHEN sponsor_type = 'customer' THEN gas_fee_usd_cents ELSE 0 END) as customer_paid_cents,
    AVG(gas_fee_usd_cents) as avg_gas_cents,
    AVG(gas_units_used) as avg_gas_units,
    COUNT(DISTINCT network_id) as networks_used
FROM gas_fee_payments gfp
JOIN payments p ON p.id = gfp.payment_id
GROUP BY workspace_id, DATE_TRUNC('day', created_at);
```

## Frontend Updates

### 1. Invoice Display with Gas Fees

```typescript
// Enhanced invoice display showing gas fees
interface InvoiceLineItem {
  id: string;
  description: string;
  quantity: number;
  unit_amount_in_cents: number;
  line_item_type: 'product' | 'gas_fee' | 'tax' | 'discount';
  is_gas_sponsored: boolean;
  gas_sponsor_name?: string;
}

function InvoiceLineItems({ items }: { items: InvoiceLineItem[] }) {
  const productItems = items.filter(i => i.line_item_type === 'product');
  const gasItems = items.filter(i => i.line_item_type === 'gas_fee');
  
  return (
    <>
      {/* Product items */}
      {productItems.map(item => (
        <div key={item.id} className="flex justify-between">
          <span>{item.description}</span>
          <span>{formatCurrency(item.unit_amount_in_cents / 100)}</span>
        </div>
      ))}
      
      {/* Gas fees */}
      {gasItems.map(item => (
        <div key={item.id} className="flex justify-between text-sm">
          <span className="text-muted-foreground">
            {item.description}
            {item.is_gas_sponsored && (
              <Badge variant="secondary" className="ml-2">
                {item.gas_sponsor_name}
              </Badge>
            )}
          </span>
          <span className={item.is_gas_sponsored ? 'line-through' : ''}>
            {item.is_gas_sponsored ? 
              <span className="text-green-600">Free</span> : 
              formatCurrency(item.unit_amount_in_cents / 100)
            }
          </span>
        </div>
      ))}
    </>
  );
}
```

### 2. Gas Fee Dashboard Widget

```typescript
function GasFeeDashboard() {
  const { data: gasMetrics } = useQuery({
    queryKey: ['gas-metrics'],
    queryFn: fetchGasMetrics,
  });
  
  return (
    <Card>
      <CardHeader>
        <CardTitle>Gas Fee Analytics</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <MetricRow 
            label="Total Gas Fees"
            value={formatCurrency(gasMetrics.total_gas_fees_cents / 100)}
          />
          <MetricRow 
            label="Sponsored by You"
            value={formatCurrency(gasMetrics.sponsored_gas_fees_cents / 100)}
            subtext={`${gasMetrics.gas_sponsorship_rate}% sponsorship rate`}
          />
          <MetricRow 
            label="Paid by Customers"
            value={formatCurrency(gasMetrics.customer_gas_fees_cents / 100)}
          />
          <MetricRow 
            label="Average Gas Fee"
            value={formatCurrency(gasMetrics.avg_gas_fee_cents / 100)}
          />
        </div>
      </CardContent>
    </Card>
  );
}
```

### 3. Gas Sponsorship Settings

```typescript
function GasSponsorshipSettings() {
  const [config, setConfig] = useState<GasSponsorshipConfig>();
  
  return (
    <Card>
      <CardHeader>
        <CardTitle>Gas Fee Sponsorship</CardTitle>
        <CardDescription>
          Sponsor customer gas fees to improve conversion
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label>Enable Gas Sponsorship</Label>
            <Switch 
              checked={config?.sponsorship_enabled}
              onCheckedChange={(checked) => updateConfig({ sponsorship_enabled: checked })}
            />
          </div>
          
          <div>
            <Label>Monthly Budget</Label>
            <Input 
              type="number"
              value={config?.monthly_budget_usd_cents / 100}
              onChange={(e) => updateConfig({ 
                monthly_budget_usd_cents: parseFloat(e.target.value) * 100 
              })}
              placeholder="100.00"
            />
            <p className="text-sm text-muted-foreground mt-1">
              Spent this month: ${(config?.current_month_spent_cents || 0) / 100}
            </p>
          </div>
          
          <div>
            <Label>Max Per Transaction</Label>
            <Input 
              type="number"
              value={config?.sponsor_threshold_usd_cents / 100}
              onChange={(e) => updateConfig({ 
                sponsor_threshold_usd_cents: parseFloat(e.target.value) * 100 
              })}
              placeholder="5.00"
            />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
```

## Migration Strategy

1. **Add new columns/tables without breaking existing code**
2. **Backfill gas data from blockchain for historical transactions**
3. **Update payment processing to capture gas in real-time**
4. **Add gas line items to new invoices going forward**
5. **Update dashboard to show gas metrics**

## Benefits

1. **Accurate Financial Tracking**: Separate product revenue from gas costs
2. **Sponsorship Flexibility**: Merchants can sponsor gas to improve UX
3. **Customer Transparency**: Clear breakdown of costs on invoices
4. **Analytics**: Understand gas cost impact on business
5. **Tax Compliance**: Proper categorization for tax purposes