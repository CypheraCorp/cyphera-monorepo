# Net Terms Invoice Enhancement - Addendum to Part 1

## Overview

This addendum enhances the original platform implementation plan with comprehensive net terms invoice functionality, supporting two distinct one-off payment types:

1. **Standard Checkout** - Immediate payment collection (supports both EOA and smart accounts)
2. **Net Terms Invoices** - B2B invoices with payment terms (Net 30/60/90) and dynamic pricing

## Enhanced Database Schema

### 1. Enhanced invoices table
```sql
-- Add net terms fields to existing invoices table
ALTER TABLE invoices 
ADD COLUMN net_terms INTEGER DEFAULT 0, -- Number of days (0, 30, 60, 90, custom)
ADD COLUMN net_terms_type VARCHAR(50), -- 'immediate', 'net_30', 'net_60', 'net_90', 'custom'
ADD COLUMN issue_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- When invoice was issued
ADD COLUMN assigned_to_customer_id UUID REFERENCES customers(id), -- For pre-assignment
ADD COLUMN payment_schedule_id UUID REFERENCES payment_schedules(id), -- For scheduled payments

-- Dynamic pricing fields
ADD COLUMN pricing_type VARCHAR(50) DEFAULT 'fixed', -- 'fixed', 'dynamic', 'early_discount'
ADD COLUMN base_amount_in_cents BIGINT, -- Original amount before any adjustments
ADD COLUMN dynamic_pricing_rules JSONB DEFAULT '{}', -- Rules for price changes

-- Payment tracking
ADD COLUMN payment_commitment_date TIMESTAMP, -- When customer commits to pay
ADD COLUMN scheduled_payment_date TIMESTAMP, -- For smart account scheduled payments
ADD COLUMN late_fee_amount_cents BIGINT DEFAULT 0,
ADD COLUMN discount_amount_cents BIGINT DEFAULT 0,

-- B2B specific fields
ADD COLUMN purchase_order_number VARCHAR(255),
ADD COLUMN invoice_memo TEXT,
ADD COLUMN payment_instructions TEXT,

-- Reminders and notifications
ADD COLUMN reminder_sent_count INTEGER DEFAULT 0,
ADD COLUMN last_reminder_sent_at TIMESTAMP,
ADD COLUMN next_reminder_date TIMESTAMP;

-- Add check constraint for net terms
ALTER TABLE invoices
ADD CONSTRAINT chk_net_terms CHECK (
    (net_terms_type = 'immediate' AND net_terms = 0) OR
    (net_terms_type = 'net_30' AND net_terms = 30) OR
    (net_terms_type = 'net_60' AND net_terms = 60) OR
    (net_terms_type = 'net_90' AND net_terms = 90) OR
    (net_terms_type = 'custom' AND net_terms > 0)
);

-- Add index for net terms queries
CREATE INDEX idx_invoices_net_terms ON invoices(net_terms_type, due_date) WHERE status = 'open';
CREATE INDEX idx_invoices_assigned_customer ON invoices(assigned_to_customer_id) WHERE assigned_to_customer_id IS NOT NULL;
```

### 2. Dynamic pricing rules table
```sql
CREATE TABLE dynamic_pricing_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Rule configuration
    name VARCHAR(255) NOT NULL,
    rule_type VARCHAR(50) NOT NULL, -- 'early_payment_discount', 'late_payment_fee', 'volume_discount'
    is_active BOOLEAN DEFAULT true,
    
    -- Conditions
    condition_type VARCHAR(50), -- 'days_before_due', 'days_after_due', 'payment_count', 'total_amount'
    condition_operator VARCHAR(10), -- '>', '<', '>=', '<=', '='
    condition_value DECIMAL(10,2),
    
    -- Actions
    action_type VARCHAR(50), -- 'percentage_discount', 'fixed_discount', 'percentage_fee', 'fixed_fee'
    action_value DECIMAL(10,2),
    max_discount_amount_cents BIGINT, -- Cap for percentage discounts
    min_fee_amount_cents BIGINT, -- Minimum fee for percentage fees
    
    -- Application rules
    applies_to VARCHAR(50) DEFAULT 'all', -- 'all', 'specific_products', 'specific_customers'
    applicable_product_ids UUID[], -- Array of product IDs
    applicable_customer_ids UUID[], -- Array of customer IDs
    
    -- Usage tracking
    times_used INTEGER DEFAULT 0,
    total_discount_given_cents BIGINT DEFAULT 0,
    total_fees_collected_cents BIGINT DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pricing_rules_workspace ON dynamic_pricing_rules(workspace_id, is_active);
CREATE INDEX idx_pricing_rules_type ON dynamic_pricing_rules(rule_type, is_active);
```

### 3. Payment schedules table (for smart account automation)
```sql
CREATE TABLE payment_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- References
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    smart_account_address VARCHAR(255) NOT NULL,
    delegation_id UUID REFERENCES delegations(id), -- For authorization
    
    -- Schedule details
    scheduled_date TIMESTAMP NOT NULL,
    amount_in_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    
    -- Crypto details
    token_id UUID REFERENCES tokens(id),
    token_amount DECIMAL(36,18), -- Calculated at schedule time
    max_slippage_percent DECIMAL(5,2) DEFAULT 2.00, -- Max price change tolerance
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed', 'cancelled'
    processing_started_at TIMESTAMP,
    completed_at TIMESTAMP,
    failed_at TIMESTAMP,
    failure_reason TEXT,
    transaction_hash VARCHAR(255),
    
    -- Retry logic
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at TIMESTAMP,
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payment_schedules_pending ON payment_schedules(scheduled_date) 
WHERE status = 'pending' AND scheduled_date <= CURRENT_TIMESTAMP;
CREATE INDEX idx_payment_schedules_invoice ON payment_schedules(invoice_id);
CREATE INDEX idx_payment_schedules_customer ON payment_schedules(customer_id);
```

### 4. Invoice reminders table
```sql
CREATE TABLE invoice_reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    
    -- Reminder configuration
    reminder_type VARCHAR(50) NOT NULL, -- 'before_due', 'on_due', 'after_due', 'final_notice'
    days_offset INTEGER NOT NULL, -- Negative for before, positive for after due date
    
    -- Delivery
    delivery_method VARCHAR(50) DEFAULT 'email', -- 'email', 'sms', 'webhook'
    sent_at TIMESTAMP,
    delivery_status VARCHAR(50), -- 'pending', 'sent', 'delivered', 'failed'
    
    -- Content
    subject VARCHAR(500),
    message TEXT,
    
    -- Response tracking
    opened_at TIMESTAMP,
    clicked_at TIMESTAMP,
    
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_reminders_invoice ON invoice_reminders(invoice_id);
CREATE INDEX idx_reminders_pending ON invoice_reminders(invoice_id, sent_at) WHERE sent_at IS NULL;
```

## Payment Flow Implementations

### 1. Standard Checkout Flow (Immediate Payment)

```typescript
// For payment links or direct checkout
interface StandardCheckoutParams {
  productId: string;
  priceId?: string;
  amount?: number; // For custom amounts
  currency: string;
  paymentMethod: 'eoa_wallet' | 'smart_account';
  customerEmail?: string;
}

// Frontend implementation
export async function initiateStandardCheckout(params: StandardCheckoutParams) {
  // Create invoice with immediate payment
  const invoice = await api.invoices.create({
    ...params,
    net_terms_type: 'immediate',
    net_terms: 0,
    collection_method: 'charge_automatically',
    status: 'open'
  });
  
  if (params.paymentMethod === 'eoa_wallet') {
    // Direct transfer from EOA wallet
    return initiateEOAPayment(invoice);
  } else {
    // Use smart account with delegation
    return initiateSmartAccountPayment(invoice);
  }
}
```

### 2. Net Terms Invoice Flow (B2B)

```typescript
// For B2B invoices with payment terms
interface NetTermsInvoiceParams {
  customerId: string;
  lineItems: LineItem[];
  netTermsType: 'net_30' | 'net_60' | 'net_90' | 'custom';
  customNetTerms?: number;
  dynamicPricingRules?: string[]; // IDs of pricing rules to apply
  purchaseOrderNumber?: string;
  memo?: string;
}

// Backend implementation
func (h *InvoiceHandler) CreateNetTermsInvoice(c *gin.Context) {
    var req NetTermsInvoiceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Calculate due date based on net terms
    issueDate := time.Now()
    dueDate := calculateDueDate(issueDate, req.NetTermsType, req.CustomNetTerms)
    
    // Apply dynamic pricing rules
    pricingAdjustments := h.calculateDynamicPricing(req)
    
    // Create invoice
    invoice := &Invoice{
        WorkspaceID:      c.GetString("workspace_id"),
        CustomerID:       req.CustomerID,
        Status:          "open",
        NetTermsType:    req.NetTermsType,
        NetTerms:        getNetTermsDays(req.NetTermsType),
        IssueDate:       issueDate,
        DueDate:         dueDate,
        BaseAmount:      req.TotalAmount,
        AmountDue:       req.TotalAmount + pricingAdjustments,
        CollectionMethod: "send_invoice",
    }
    
    // Set up reminders
    h.scheduleInvoiceReminders(invoice)
    
    c.JSON(201, invoice)
}
```

### 3. Dynamic Pricing Implementation

```go
// Early payment discount example
type EarlyPaymentDiscount struct {
    DaysBeforeDue int
    DiscountPercent float64
    MaxDiscountCents int64
}

func calculateEarlyPaymentDiscount(invoice *Invoice, paymentDate time.Time) int64 {
    daysUntilDue := invoice.DueDate.Sub(paymentDate).Hours() / 24
    
    for _, rule := range invoice.DynamicPricingRules {
        if rule.Type == "early_payment_discount" && 
           daysUntilDue >= float64(rule.DaysBeforeDue) {
            discount := int64(float64(invoice.BaseAmount) * rule.DiscountPercent / 100)
            if rule.MaxDiscountCents > 0 && discount > rule.MaxDiscountCents {
                discount = rule.MaxDiscountCents
            }
            return discount
        }
    }
    
    return 0
}

// Late payment fee example
func calculateLatePaymentFee(invoice *Invoice, paymentDate time.Time) int64 {
    if paymentDate.Before(invoice.DueDate) {
        return 0
    }
    
    daysLate := paymentDate.Sub(invoice.DueDate).Hours() / 24
    
    for _, rule := range invoice.DynamicPricingRules {
        if rule.Type == "late_payment_fee" {
            if rule.ActionType == "percentage_fee" {
                fee := int64(float64(invoice.BaseAmount) * rule.ActionValue / 100)
                if rule.MinFeeCents > 0 && fee < rule.MinFeeCents {
                    fee = rule.MinFeeCents
                }
                return fee
            } else if rule.ActionType == "fixed_fee" {
                return int64(rule.ActionValue)
            }
        }
    }
    
    return 0
}
```

### 4. Smart Account Scheduled Payment

```typescript
// Frontend - Schedule payment
export async function scheduleInvoicePayment(params: {
  invoiceId: string;
  paymentDate: Date;
  smartAccountAddress: string;
  delegation: Delegation;
}) {
  // Calculate token amount based on scheduled date
  const tokenQuote = await getTokenQuote({
    fiatAmount: invoice.amountDue,
    fiatCurrency: invoice.currency,
    tokenSymbol: selectedToken.symbol,
    executionDate: params.paymentDate
  });
  
  // Create payment schedule
  const schedule = await api.paymentSchedules.create({
    invoice_id: params.invoiceId,
    scheduled_date: params.paymentDate,
    amount_in_cents: invoice.amountDue,
    token_amount: tokenQuote.tokenAmount,
    smart_account_address: params.smartAccountAddress,
    delegation_id: params.delegation.id,
    max_slippage_percent: 2.0
  });
  
  return schedule;
}

// Backend - Process scheduled payments
func (w *PaymentScheduleWorker) ProcessScheduledPayments() error {
    // Get all pending payments due now
    schedules, err := w.db.GetPendingScheduledPayments(time.Now())
    if err != nil {
        return err
    }
    
    for _, schedule := range schedules {
        // Mark as processing
        w.db.UpdateScheduleStatus(schedule.ID, "processing")
        
        // Get current token price
        currentPrice, err := w.getTokenPrice(schedule.TokenID)
        if err != nil {
            w.handleScheduleError(schedule, err)
            continue
        }
        
        // Check slippage
        if !w.isWithinSlippage(schedule, currentPrice) {
            w.handleSlippageExceeded(schedule)
            continue
        }
        
        // Execute payment via delegation
        txHash, err := w.executeScheduledPayment(schedule)
        if err != nil {
            w.handleScheduleError(schedule, err)
            continue
        }
        
        // Update schedule as completed
        w.db.CompleteSchedule(schedule.ID, txHash)
        
        // Update invoice
        w.updateInvoicePayment(schedule.InvoiceID, schedule.AmountInCents)
    }
    
    return nil
}
```

## UI Components

### 1. Invoice Creation Form Enhancement
```typescript
// Enhanced invoice form with net terms
export function CreateInvoiceForm() {
  const [invoiceType, setInvoiceType] = useState<'immediate' | 'net_terms'>('immediate');
  const [netTerms, setNetTerms] = useState<'net_30' | 'net_60' | 'net_90' | 'custom'>('net_30');
  const [dynamicPricing, setDynamicPricing] = useState({
    earlyPaymentDiscount: false,
    latePaymentFee: false,
    discountPercent: 2,
    feePercent: 1.5
  });
  
  return (
    <Form onSubmit={handleSubmit}>
      <RadioGroup value={invoiceType} onValueChange={setInvoiceType}>
        <RadioItem value="immediate">
          <Label>Immediate Payment</Label>
          <Description>Customer pays right away</Description>
        </RadioItem>
        <RadioItem value="net_terms">
          <Label>Net Terms Invoice</Label>
          <Description>B2B invoice with payment terms</Description>
        </RadioItem>
      </RadioGroup>
      
      {invoiceType === 'net_terms' && (
        <>
          <Select value={netTerms} onValueChange={setNetTerms}>
            <SelectItem value="net_30">Net 30 Days</SelectItem>
            <SelectItem value="net_60">Net 60 Days</SelectItem>
            <SelectItem value="net_90">Net 90 Days</SelectItem>
            <SelectItem value="custom">Custom Terms</SelectItem>
          </Select>
          
          <DynamicPricingConfig 
            value={dynamicPricing}
            onChange={setDynamicPricing}
          />
        </>
      )}
    </Form>
  );
}
```

### 2. Customer Invoice Portal
```typescript
// Customer-facing invoice view with payment options
export function CustomerInvoiceView({ invoiceId }: { invoiceId: string }) {
  const invoice = useInvoice(invoiceId);
  const { hasSmartAccount } = useWallet();
  
  const earlyPaymentDiscount = calculateEarlyPaymentDiscount(invoice);
  const currentAmount = invoice.baseAmount - earlyPaymentDiscount;
  
  return (
    <InvoiceCard>
      <InvoiceHeader>
        <h1>Invoice #{invoice.number}</h1>
        <NetTermsBadge type={invoice.netTermsType} />
      </InvoiceHeader>
      
      <InvoiceDetails>
        <AmountDisplay 
          baseAmount={invoice.baseAmount}
          currentAmount={currentAmount}
          discount={earlyPaymentDiscount}
        />
        
        <DueDateInfo 
          dueDate={invoice.dueDate}
          daysRemaining={getDaysUntilDue(invoice)}
        />
        
        {earlyPaymentDiscount > 0 && (
          <Alert variant="success">
            Pay by {formatDate(getEarlyPaymentDeadline(invoice))} 
            to save {formatCurrency(earlyPaymentDiscount)}
          </Alert>
        )}
      </InvoiceDetails>
      
      <PaymentOptions>
        <Button onClick={() => payWithEOA(invoice)}>
          Pay Now with Wallet
        </Button>
        
        {hasSmartAccount && invoice.netTermsType !== 'immediate' && (
          <Button onClick={() => schedulePayment(invoice)}>
            Schedule Payment
          </Button>
        )}
      </PaymentOptions>
    </InvoiceCard>
  );
}
```

## API Endpoints

### Invoice Endpoints
```yaml
# Create standard checkout invoice
POST /api/v1/invoices/checkout
{
  "product_id": "uuid",
  "amount": 10000,
  "currency": "USD",
  "payment_method": "eoa_wallet"
}

# Create net terms invoice
POST /api/v1/invoices/net-terms
{
  "customer_id": "uuid",
  "line_items": [...],
  "net_terms_type": "net_30",
  "dynamic_pricing_rules": ["rule_id_1", "rule_id_2"],
  "purchase_order": "PO-12345"
}

# Get invoice with current pricing
GET /api/v1/invoices/:id/current-pricing
Response: {
  "base_amount": 10000,
  "current_amount": 9800,
  "discount": 200,
  "discount_reason": "2% early payment discount",
  "valid_until": "2024-01-25T00:00:00Z"
}

# Schedule payment (customer endpoint)
POST /api/v1/invoices/:id/schedule-payment
{
  "payment_date": "2024-02-15T10:00:00Z",
  "smart_account_address": "0x...",
  "delegation": {...},
  "token_id": "uuid"
}
```

## Integration Points

### 1. With Existing Systems
- Invoices link to products and prices tables
- Integration with payment_events for tracking
- Dashboard metrics updated on payment
- Subscription invoices can have net terms

### 2. With Smart Accounts
- Scheduled payments use delegation system
- Gas estimation for future payments
- Slippage protection for volatile tokens
- Automatic retry on failure

### 3. With Notification System
- Email reminders for upcoming due dates
- SMS alerts for large invoices
- Webhook notifications for B2B integrations
- In-app notifications for payment received

## Benefits

1. **Flexibility**: Support both immediate and deferred payment models
2. **B2B Ready**: Professional net terms invoicing for enterprise customers
3. **Cash Flow Optimization**: Dynamic discounting encourages early payment
4. **Automation**: Smart account scheduled payments reduce manual work
5. **Compatibility**: Works with both EOA wallets and smart accounts

This enhancement positions Cyphera as a comprehensive B2B crypto payment platform, supporting the full spectrum from simple checkouts to complex enterprise invoicing with flexible payment terms.