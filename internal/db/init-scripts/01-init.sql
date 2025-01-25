-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum for API key access levels
CREATE TYPE api_key_level AS ENUM ('read', 'write', 'admin');

-- Enum for subscription status
CREATE TYPE subscription_status AS ENUM ('active', 'canceled', 'past_due', 'incomplete', 'incomplete_expired', 'trialing', 'unpaid');

-- Customers table
CREATE TABLE IF NOT EXISTS customers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255),
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    balance INTEGER DEFAULT 0,  -- Stored in cents
    currency VARCHAR(3) DEFAULT 'USD',
    default_source_id UUID,  -- Reference to default payment method
    invoice_prefix VARCHAR(12),
    next_invoice_sequence INTEGER DEFAULT 1,
    tax_exempt VARCHAR(20) DEFAULT 'none',
    tax_ids JSONB,
    livemode BOOLEAN DEFAULT false
);
-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    customer_id UUID REFERENCES customers(id),
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,  -- Store hashed key, not plain text
    level api_key_level NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB,
    livemode BOOLEAN DEFAULT false
);

-- Products table
CREATE TABLE IF NOT EXISTS products (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN DEFAULT true,
    default_price_id UUID,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    livemode BOOLEAN DEFAULT false
);

-- Prices table
CREATE TABLE IF NOT EXISTS prices (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    product_id UUID REFERENCES products(id),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    unit_amount INTEGER,  -- Stored in cents
    recurring_interval VARCHAR(10),  -- 'month', 'year', etc.
    recurring_interval_count INTEGER,
    usage_type VARCHAR(20),  -- 'licensed' or 'metered'
    billing_scheme VARCHAR(20),  -- 'per_unit' or 'tiered'
    active BOOLEAN DEFAULT true,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    livemode BOOLEAN DEFAULT false
);


-- Subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    customer_id UUID REFERENCES customers(id),
    status subscription_status NOT NULL,
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    cancel_at TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    livemode BOOLEAN DEFAULT false
);

-- Subscription Items table
CREATE TABLE IF NOT EXISTS subscription_items (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    subscription_id UUID REFERENCES subscriptions(id),
    price_id UUID REFERENCES prices(id),
    quantity INTEGER DEFAULT 1,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Payment Methods table
CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    customer_id UUID REFERENCES customers(id),
    type VARCHAR(20) NOT NULL,  -- 'card', 'bank_account', etc.
    details JSONB NOT NULL,  -- Encrypted payment details
    is_default BOOLEAN DEFAULT false,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    livemode BOOLEAN DEFAULT false
);

-- Invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    customer_id UUID REFERENCES customers(id),
    subscription_id UUID REFERENCES subscriptions(id),
    status VARCHAR(20) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    amount_due INTEGER NOT NULL,  -- Stored in cents
    amount_paid INTEGER DEFAULT 0,  -- Stored in cents
    amount_remaining INTEGER,  -- Stored in cents
    paid BOOLEAN DEFAULT false,
    attempt_count INTEGER DEFAULT 0,
    next_payment_attempt TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    livemode BOOLEAN DEFAULT false
);

-- Create indexes
CREATE INDEX idx_api_keys_customer_id ON api_keys(customer_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX idx_payment_methods_customer_id ON payment_methods(customer_id);
CREATE INDEX idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX idx_subscription_items_subscription_id ON subscription_items(subscription_id);

-- Insert test data for development
INSERT INTO customers (email, name, description)
VALUES ('test@example.com', 'Test Customer', 'Test customer for development')
ON CONFLICT (email) DO NOTHING;

-- Insert a test API key (hash of 'test-key-123')
INSERT INTO api_keys (customer_id, name, key_hash, level)
VALUES (
    (SELECT id FROM customers WHERE email = 'test@example.com'),
    'Test API Key',
    'test-key-hash-123',  -- In production, use proper hashing
    'admin'
)
ON CONFLICT (key_hash) DO NOTHING;