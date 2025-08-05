# Product Add-ons and Subscription Line Items Architecture

## Overview

This document describes the implementation of a modular subscription system that supports base products with optional add-ons. The system allows customers to build custom subscriptions by selecting a base product and adding optional components, with full transparency in billing through subscription and invoice line items.

## Key Concepts

- **Base Product**: The primary product in a subscription (e.g., "Pro Plan")
- **Add-on Product**: Optional components that enhance the base product (e.g., "Extra Seats", "Premium Support")
- **Subscription Line Items**: Individual components that make up a subscription
- **Invoice Line Items**: Billing breakdown that mirrors subscription line items

## Database Architecture

### 1. Product Structure

Products now include pricing information directly (no separate prices table):

```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    external_id VARCHAR(255),
    
    -- Product details
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    url TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    
    -- Product categorization
    product_type VARCHAR(50) DEFAULT 'base', -- 'base' or 'addon'
    
    -- Pricing information (merged from prices table)
    price_type price_type NOT NULL DEFAULT 'recurring', -- 'recurring' or 'one_time'
    currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code),
    unit_amount_in_pennies INTEGER NOT NULL,
    interval_type interval_type, -- 'daily', 'weekly', 'monthly', 'yearly'
    term_length INTEGER, -- Number of intervals (e.g., 12 for 12 months)
    
    -- Grouping for related products
    product_group VARCHAR(100), -- Groups related products (e.g., "pro_plan" for monthly/yearly variants)
    
    -- Payment sync fields
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50),
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(workspace_id, external_id, payment_provider)
);
```

### 2. Product Add-ons Relationship

Links base products with their available add-ons:

```sql
CREATE TABLE product_addons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    addon_product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    
    -- Display configuration
    display_order INTEGER DEFAULT 0,
    is_required BOOLEAN DEFAULT false,
    is_popular BOOLEAN DEFAULT false,
    
    -- Quantity constraints
    min_quantity INTEGER DEFAULT 0,
    max_quantity INTEGER, -- NULL = unlimited
    default_quantity INTEGER DEFAULT 1,
    
    -- Grouping for mutually exclusive add-ons
    addon_group VARCHAR(100), -- Only one add-on per group can be selected
    
    -- Metadata for UI customization
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(base_product_id, addon_product_id)
);
```

### 3. Subscription Line Items

Tracks individual components of a subscription:

```sql
CREATE TABLE subscription_line_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    
    -- Line item identification
    name TEXT NOT NULL,
    description TEXT,
    
    -- Product reference
    product_id UUID NOT NULL REFERENCES products(id),
    product_token_id UUID REFERENCES products_tokens(id),
    
    -- Quantity and pricing
    quantity DECIMAL(10,4) NOT NULL DEFAULT 1,
    unit_amount_in_cents BIGINT NOT NULL,
    amount_in_cents BIGINT NOT NULL, -- quantity * unit_amount
    
    -- Crypto amounts
    token_amount DECIMAL(36,18) NOT NULL,
    
    -- Line item categorization
    line_item_type VARCHAR(50) NOT NULL DEFAULT 'base', -- 'base', 'addon', 'discount'
    is_primary BOOLEAN DEFAULT false, -- True for the main subscription item
    
    -- Status and ordering
    is_active BOOLEAN DEFAULT true,
    display_order INTEGER DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 4. Updated Subscriptions Table

Subscriptions now reference multiple line items instead of a single product/price:

```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    num_id BIGSERIAL UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Remove direct product/price references (now in line items)
    -- product_id, price_id removed
    
    -- Customer payment details
    customer_wallet_id UUID REFERENCES customer_wallets(id),
    delegation_id UUID NOT NULL REFERENCES delegation_data(id),
    
    -- Status and scheduling
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    next_redemption_date TIMESTAMP WITH TIME ZONE,
    
    -- Totals (calculated from line items)
    total_amount_in_cents INTEGER NOT NULL DEFAULT 0,
    total_token_amount DECIMAL(36,18) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code),
    
    -- Redemption tracking
    total_redemptions INTEGER NOT NULL DEFAULT 0,
    
    -- External references
    external_id VARCHAR(255),
    initial_transaction_hash TEXT,
    
    -- Payment sync
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50),
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(workspace_id, external_id, payment_provider)
);
```

## API Design

### 1. Product Response with Add-ons

```typescript
interface ProductDetailResponse {
    id: string;
    name: string;
    description: string;
    product_type: 'base' | 'addon';
    
    // Pricing information
    price_type: 'recurring' | 'one_time';
    currency: string;
    unit_amount_in_pennies: number;
    interval_type?: string;
    term_length?: number;
    
    // Available add-ons (only for base products)
    available_addons?: ProductAddonResponse[];
}

interface ProductAddonResponse {
    product: ProductDetailResponse;
    display_order: number;
    is_required: boolean;
    is_popular: boolean;
    min_quantity: number;
    max_quantity?: number;
    default_quantity: number;
    addon_group?: string;
}
```

### 2. Create Subscription Request

```typescript
interface CreateSubscriptionRequest {
    base_product_id: string;
    product_token_id: string;
    delegation_data: DelegationData;
    
    // Selected add-ons with quantities
    addons?: Array<{
        product_id: string;
        quantity: number;
    }>;
}
```

### 3. Subscription Response with Line Items

```typescript
interface SubscriptionResponse {
    id: string;
    num_id: number;
    status: string;
    
    // Totals calculated from line items
    total_amount_in_cents: number;
    total_token_amount: string;
    currency: string;
    
    // Line items breakdown
    line_items: SubscriptionLineItemResponse[];
    
    // Other subscription fields...
}

interface SubscriptionLineItemResponse {
    id: string;
    name: string;
    description?: string;
    product_id: string;
    quantity: string;
    unit_amount_in_cents: number;
    amount_in_cents: number;
    token_amount: string;
    line_item_type: 'base' | 'addon' | 'discount';
    is_primary: boolean;
    is_active: boolean;
}
```

## Implementation Flow

### 1. Product Setup

```sql
-- Base product
INSERT INTO products (name, product_type, unit_amount_in_pennies, currency, interval_type, term_length) 
VALUES ('Pro Plan', 'base', 9900, 'USD', 'monthly', 1);

-- Add-on products
INSERT INTO products (name, product_type, unit_amount_in_pennies, currency, interval_type, term_length) 
VALUES ('Extra Seats (5 pack)', 'addon', 5000, 'USD', 'monthly', 1);

INSERT INTO products (name, product_type, unit_amount_in_pennies, currency, interval_type, term_length) 
VALUES ('Premium Support', 'addon', 3000, 'USD', 'monthly', 1);

-- Link add-ons to base product
INSERT INTO product_addons (base_product_id, addon_product_id, display_order, is_popular)
VALUES 
    ((SELECT id FROM products WHERE name = 'Pro Plan'), 
     (SELECT id FROM products WHERE name = 'Extra Seats (5 pack)'), 
     1, false),
    ((SELECT id FROM products WHERE name = 'Pro Plan'), 
     (SELECT id FROM products WHERE name = 'Premium Support'), 
     2, true);
```

### 2. Subscription Creation

When a customer subscribes with add-ons:

1. Create the subscription record with calculated totals
2. Create subscription_line_items for base product and each selected add-on
3. Process the blockchain transaction for the total amount
4. Create subscription events for tracking

### 3. Invoice Generation

When generating an invoice:

```sql
-- Copy subscription line items to invoice line items
INSERT INTO invoice_line_items (
    invoice_id,
    subscription_id,
    product_id,
    description,
    quantity,
    unit_amount_in_cents,
    amount_in_cents,
    fiat_currency,
    line_item_type
)
SELECT 
    @invoice_id,
    @subscription_id,
    product_id,
    name,
    quantity,
    unit_amount_in_cents,
    amount_in_cents,
    @currency,
    line_item_type
FROM subscription_line_items
WHERE subscription_id = @subscription_id
AND is_active = true;
```

## Benefits

1. **Flexibility**: Support any combination of products and add-ons
2. **Transparency**: Clear breakdown of what customers are paying for
3. **Scalability**: Easy to add new add-ons without modifying core structure
4. **Simplicity**: Single product = single price, no complex price arrays
5. **Compatibility**: Direct mapping between subscription and invoice line items

## Example Use Cases

### 1. SaaS with Seat-based Add-ons
- Base: Team Plan ($99/month)
- Add-on: Extra seats at $10/seat/month
- Customer can add 1-50 extra seats

### 2. Service with Support Tiers
- Base: Standard Service ($199/month)
- Add-on Group "support_tier":
  - Email Support (free, default)
  - Priority Support (+$50/month)
  - 24/7 Support (+$150/month)
- Customer can only select one support tier

### 3. Platform with Feature Add-ons
- Base: Basic Plan ($49/month)
- Add-ons:
  - API Access (+$99/month)
  - Advanced Analytics (+$79/month)
  - White-label Options (+$199/month)
- Customer can mix and match features

## Testing Considerations

1. **Validation**:
   - Ensure required add-ons are included
   - Validate quantity constraints
   - Check mutually exclusive groups

2. **Calculations**:
   - Verify total amounts match sum of line items
   - Test crypto token amount calculations
   - Validate invoice generation

3. **Edge Cases**:
   - Subscription with no add-ons
   - Maximum quantity limits
   - Deactivating line items mid-subscription
   - Add-on price changes

## Future Enhancements

1. **Usage-based Add-ons**: Track actual usage for metered components
2. **Proration**: Handle mid-cycle add-on additions/removals
3. **Bundling Discounts**: Apply discounts when certain add-ons are combined
4. **Trial Periods**: Support different trial lengths for base vs add-ons