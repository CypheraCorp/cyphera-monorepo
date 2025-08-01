# Price Simplification Plan - Merging Prices into Products

## Overview

This document outlines the plan to simplify the product-price relationship by merging the separate `prices` table directly into the `products` table. This change establishes a 1:1 relationship between products and their pricing, making the system simpler and more intuitive.

## Rationale

### Current Problems
1. **Complexity**: One product can have multiple prices, complicating UI/UX
2. **Add-on Conflicts**: Multiple price tiers conflict with the add-on model
3. **Mental Model**: Developers and users struggle with the separation
4. **Query Overhead**: Always need to join products with prices

### Benefits of Simplification
1. **Clarity**: One product = one price point
2. **Add-on Support**: Clean base + add-on pricing model
3. **Performance**: No joins needed for basic product info
4. **Simplicity**: Easier to understand and maintain

## Implementation Strategy

### 1. Database Schema Changes

#### Remove Prices Table
```sql
-- The entire prices table will be removed
DROP TABLE IF EXISTS prices CASCADE;
```

#### Update Products Table
```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    
    -- Existing product fields
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    url TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    external_id VARCHAR(255),
    
    -- NEW: Product categorization
    product_type VARCHAR(50) DEFAULT 'base', -- 'base' or 'addon'
    product_group VARCHAR(100), -- Groups related products (e.g., "pro_plan")
    
    -- NEW: Merged price fields
    price_type price_type NOT NULL DEFAULT 'recurring',
    currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code),
    unit_amount_in_pennies INTEGER NOT NULL,
    interval_type interval_type, -- NULL for one_time
    term_length INTEGER, -- NULL for one_time
    price_nickname TEXT, -- Optional friendly name
    price_external_id VARCHAR(255), -- Provider's price ID
    
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
    
    -- Constraints
    CONSTRAINT products_price_type_check CHECK (
        (price_type = 'recurring' AND interval_type IS NOT NULL AND term_length IS NOT NULL AND term_length > 0) OR
        (price_type = 'one_time' AND interval_type IS NULL AND term_length IS NULL)
    ),
    
    UNIQUE(workspace_id, external_id, payment_provider)
);
```

### 2. Product Modeling Examples

#### Before (Multiple Prices)
```
Product: "Pro Plan"
├── Price 1: $99/month
├── Price 2: $990/year
└── Price 3: $199 one-time setup
```

#### After (Separate Products)
```
Product 1: "Pro Plan Monthly"
├── price_type: recurring
├── unit_amount: 9900
├── interval_type: monthly
└── product_group: "pro_plan"

Product 2: "Pro Plan Yearly"
├── price_type: recurring
├── unit_amount: 99000
├── interval_type: yearly
└── product_group: "pro_plan"

Product 3: "Pro Plan Setup Fee"
├── price_type: one_time
├── unit_amount: 19900
└── product_group: "pro_plan"
```

### 3. Foreign Key Updates

All references to `price_id` need to be updated:

#### Subscriptions Table
```sql
-- Remove price_id, keep only product_id
ALTER TABLE subscriptions DROP COLUMN price_id;

-- Product_id now contains all pricing info
-- subscription_line_items will reference products directly
```

#### Invoice Line Items
```sql
-- Update to reference product_id instead of price_id
ALTER TABLE invoice_line_items DROP COLUMN price_id;
-- product_id already exists and contains pricing
```

### 4. Query Updates

#### Create Product (Before)
```sql
-- Had to create product, then create price(s)
INSERT INTO products (...) VALUES (...);
INSERT INTO prices (product_id, ...) VALUES (...);
```

#### Create Product (After)
```sql
-- Single insert with all data
INSERT INTO products (
    name, description, product_type,
    price_type, unit_amount_in_pennies, 
    currency, interval_type, term_length
) VALUES (
    'Pro Plan Monthly', 'Our pro tier', 'base',
    'recurring', 9900, 
    'USD', 'monthly', 1
);
```

#### Get Product with Price (Before)
```sql
SELECT p.*, pr.*
FROM products p
JOIN prices pr ON pr.product_id = p.id
WHERE p.id = $1;
```

#### Get Product with Price (After)
```sql
SELECT * FROM products WHERE id = $1;
```

### 5. API Changes

#### Product Response (Before)
```json
{
    "id": "prod_123",
    "name": "Pro Plan",
    "prices": [
        {
            "id": "price_123",
            "unit_amount": 9900,
            "interval": "month"
        },
        {
            "id": "price_456",
            "unit_amount": 99000,
            "interval": "year"
        }
    ]
}
```

#### Product Response (After)
```json
{
    "id": "prod_123",
    "name": "Pro Plan Monthly",
    "product_group": "pro_plan",
    "price_type": "recurring",
    "unit_amount_in_pennies": 9900,
    "currency": "USD",
    "interval_type": "monthly",
    "term_length": 1
}
```

### 6. Subscription Creation

#### Before
```json
{
    "price_id": "price_123",
    "product_token_id": "token_456"
}
```

#### After
```json
{
    "product_id": "prod_123",  // Contains all pricing info
    "product_token_id": "token_456",
    "addons": [
        {
            "product_id": "prod_addon_1",
            "quantity": 2
        }
    ]
}
```

## Impact Analysis

### 1. Database Queries to Update

- **products.sql**: Remove price-related joins
- **subscriptions.sql**: Update to use product pricing directly
- **invoices.sql**: Remove price_id references
- Remove entire **prices.sql** file

### 2. Backend Services

- **ProductService**: Simplify to handle embedded pricing
- **SubscriptionService**: Update to calculate from product prices
- **PaymentSyncService**: Update to sync product prices directly

### 3. Frontend Changes

- **Product Selection**: Show products directly (no price dropdown)
- **Product Grouping**: Group related products by `product_group`
- **Subscription Flow**: Simplified product + add-ons selection

### 4. Data Implications

For existing systems doing a cutover:
- Each current price becomes a separate product
- Products with multiple prices split into multiple products
- Use `product_group` to maintain relationships

## Best Practices

### 1. Naming Conventions
```
Base Products:
- "Pro Plan Monthly"
- "Pro Plan Yearly"
- "Enterprise Plan Quarterly"

Add-on Products:
- "Extra Seats (5 pack)"
- "Premium Support"
- "Additional Storage (100GB)"
```

### 2. Product Grouping
Use `product_group` to link related products:
```sql
-- All Pro Plan variants
SELECT * FROM products 
WHERE product_group = 'pro_plan' 
AND deleted_at IS NULL;
```

### 3. Add-on Compatibility
Use `product_type` to distinguish:
```sql
-- Get all add-ons for a base product
SELECT p.* FROM products p
JOIN product_addons pa ON p.id = pa.addon_product_id
WHERE pa.base_product_id = $1
AND p.product_type = 'addon';
```

## Rollout Steps

1. **Update Schema**: Modify products table in 01-init.sql
2. **Remove Prices**: Delete prices table and related code
3. **Update Queries**: Modify all SQL queries
4. **Regenerate SQLC**: Run code generation
5. **Update Services**: Modify business logic
6. **Update APIs**: Adjust request/response contracts
7. **Frontend Updates**: Modify product selection UI
8. **Testing**: Comprehensive testing of new flow

## Example Scenarios

### 1. SaaS Product Line
```sql
-- Base products
INSERT INTO products (name, product_type, product_group, unit_amount_in_pennies, interval_type) VALUES
('Starter Monthly', 'base', 'starter', 2900, 'monthly'),
('Starter Yearly', 'base', 'starter', 29000, 'yearly'),
('Pro Monthly', 'base', 'pro', 9900, 'monthly'),
('Pro Yearly', 'base', 'pro', 99000, 'yearly');

-- Add-ons
INSERT INTO products (name, product_type, unit_amount_in_pennies, interval_type) VALUES
('Extra User Seat', 'addon', 1000, 'monthly'),
('Priority Support', 'addon', 5000, 'monthly');
```

### 2. One-time Purchase with Subscription
```sql
-- One-time setup
INSERT INTO products (name, price_type, unit_amount_in_pennies) VALUES
('Implementation Package', 'one_time', 50000);

-- Recurring service
INSERT INTO products (name, price_type, unit_amount_in_pennies, interval_type) VALUES
('Managed Service', 'recurring', 20000, 'monthly');
```

## Benefits Summary

1. **Simplicity**: Direct 1:1 product-price relationship
2. **Performance**: No joins for basic product queries
3. **Clarity**: Clear mental model for developers
4. **Flexibility**: Easy to implement add-ons
5. **Maintainability**: Less complex codebase

## Risks and Mitigations

1. **Risk**: More products in the database
   - **Mitigation**: Use indexing and product_group for organization

2. **Risk**: UI complexity for related products
   - **Mitigation**: Smart grouping and filtering in frontend

3. **Risk**: External system integration
   - **Mitigation**: Map external price IDs to products

## Conclusion

This simplification makes the system more intuitive while enabling powerful features like add-ons. The trade-off of having more product records is worth the gained simplicity and flexibility.