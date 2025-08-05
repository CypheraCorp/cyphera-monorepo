-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum definitions (order doesn't matter here)
CREATE TYPE api_key_level AS ENUM ('read', 'write', 'admin');
CREATE TYPE account_type AS ENUM ('admin', 'merchant');
CREATE TYPE user_role AS ENUM ('admin', 'support', 'developer');
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'pending');
CREATE TYPE price_type AS ENUM ('recurring', 'one_time');
CREATE TYPE interval_type AS ENUM ('1min', '5mins', 'daily', 'week', 'month', 'year');
CREATE TYPE network_type AS ENUM ('evm', 'solana', 'cosmos', 'bitcoin', 'polkadot');
-- Currency enum removed - using fiat_currencies table instead
CREATE TYPE wallet_type AS ENUM ('wallet', 'circle_wallet', 'web3auth');
CREATE TYPE circle_network_type AS ENUM ('ARB', 'ARB-SEPOLIA', 'ETH', 'ETH-SEPOLIA', 'MATIC', 'MATIC-AMOY', 'OP', 'OP-SEPOLIA', 'BASE', 'BASE-SEPOLIA', 'UNI', 'UNI-SEPOLIA');
CREATE TYPE subscription_status AS ENUM ('active', 'canceled', 'expired', 'overdue', 'suspended', 'failed', 'completed', 'trial');
CREATE TYPE subscription_event_type AS ENUM (
    'create', 
    'redeem', 
    'renew', 
    'cancel', 
    'expire',
    'upgrade',
    'downgrade',
    'pause',
    'resume',
    'reactivate',
    'complete',
    'fail',
    'fail_validation',
    'fail_customer_creation',
    'fail_wallet_creation',
    'fail_delegation_storage',
    'fail_subscription_db',
    'fail_redemption',
    'fail_transaction',
    'fail_duplicate'
);

CREATE TYPE subscription_change_type AS ENUM ('upgrade', 'downgrade', 'cancel', 'pause', 'resume', 'modify_items', 'reactivate');

-- Create Tables in dependency order

-- Fiat Currencies table (no dependencies - replaces currency enum)
CREATE TABLE IF NOT EXISTS fiat_currencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(3) NOT NULL UNIQUE, -- ISO 4217 currency code (USD, EUR, GBP, etc.)
    name VARCHAR(100) NOT NULL, -- Full currency name
    symbol VARCHAR(10) NOT NULL, -- Currency symbol ($, €, £, etc.)
    decimal_places INTEGER NOT NULL DEFAULT 2, -- Number of decimal places
    is_active BOOLEAN DEFAULT true,
    
    -- Display settings
    symbol_position VARCHAR(10) DEFAULT 'before', -- before or after the amount
    thousand_separator VARCHAR(2) DEFAULT ',',
    decimal_separator VARCHAR(2) DEFAULT '.',
    
    -- Regional info
    countries JSONB DEFAULT '[]', -- Array of country codes using this currency
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert common currencies (only USD and EUR are active initially)
INSERT INTO fiat_currencies (code, name, symbol, decimal_places, is_active, countries) VALUES
    ('USD', 'US Dollar', '$', 2, true, '["US"]'),
    ('EUR', 'Euro', '€', 2, true, '["DE", "FR", "IT", "ES", "NL", "BE", "AT", "IE", "FI", "PT", "GR", "LU"]'),
    ('GBP', 'British Pound', '£', 2, false, '["GB"]'),
    ('JPY', 'Japanese Yen', '¥', 0, false, '["JP"]'),
    ('CAD', 'Canadian Dollar', 'C$', 2, false, '["CA"]'),
    ('AUD', 'Australian Dollar', 'A$', 2, false, '["AU"]'),
    ('CHF', 'Swiss Franc', 'Fr', 2, false, '["CH"]'),
    ('CNY', 'Chinese Yuan', '¥', 2, false, '["CN"]'),
    ('INR', 'Indian Rupee', '₹', 2, false, '["IN"]'),
    ('SGD', 'Singapore Dollar', 'S$', 2, false, '["SG"]')
ON CONFLICT (code) DO NOTHING;

-- Accounts table (top level organization)
CREATE TABLE IF NOT EXISTS accounts (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    account_type account_type NOT NULL,
    owner_id UUID,
    business_name VARCHAR(255),
    business_type VARCHAR(255),
    website_url TEXT,
    support_email VARCHAR(255),
    support_phone VARCHAR(255),
    metadata JSONB,
    finished_onboarding BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Users table (depends on accounts)
CREATE TABLE IF NOT EXISTS users (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    web3auth_id VARCHAR(255) UNIQUE,
    verifier VARCHAR(100),
    verifier_id VARCHAR(255),
    email VARCHAR(255) NOT NULL UNIQUE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    role user_role NOT NULL,
    is_account_owner BOOLEAN DEFAULT false,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    address_line_1 VARCHAR(255),
    address_line_2 VARCHAR(255),
    city VARCHAR(255),
    state_region VARCHAR(255),
    postal_code VARCHAR(255),
    country VARCHAR(255),
    display_name VARCHAR(255),
    picture_url TEXT,
    phone VARCHAR(255),
    timezone VARCHAR(50),
    locale VARCHAR(10) DEFAULT 'en',
    last_login_at TIMESTAMP WITH TIME ZONE,
    email_verified BOOLEAN DEFAULT false,
    two_factor_enabled BOOLEAN DEFAULT false,
    finished_onboarding BOOLEAN DEFAULT false,
    status user_status DEFAULT 'active',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT one_owner_per_account EXCLUDE (account_id WITH =) WHERE (is_account_owner = true AND deleted_at IS NULL)
);

-- Workspaces table (depends on accounts)
CREATE TABLE IF NOT EXISTS workspaces (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    business_name VARCHAR(255),
    business_type VARCHAR(255),
    website_url TEXT,
    support_email VARCHAR(255),
    support_phone VARCHAR(255),
    metadata JSONB,
    livemode BOOLEAN DEFAULT false,
    default_currency VARCHAR(3) DEFAULT 'USD' REFERENCES fiat_currencies(code),
    supported_currencies JSONB DEFAULT '["USD"]'::jsonb, -- Array of supported currency codes
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Circle Users Table (depends on workspaces)
CREATE TABLE IF NOT EXISTS circle_users (
    id UUID PRIMARY KEY,                      -- Circle User ID
    workspace_id UUID NOT NULL REFERENCES workspaces(id), -- Our Workspace ID
    circle_create_date TIMESTAMP WITH TIME ZONE NOT NULL,
    pin_status TEXT NOT NULL DEFAULT 'UNSET',
    status TEXT NOT NULL DEFAULT 'ENABLED',
    security_question_status TEXT NOT NULL DEFAULT 'UNSET',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Index for fast lookups by workspace_id
    UNIQUE(workspace_id),
    
    -- Constraints for enum-like fields
    CONSTRAINT valid_pin_status CHECK (pin_status IN ('ENABLED', 'UNSET', 'LOCKED')),
    CONSTRAINT valid_status CHECK (status IN ('ENABLED', 'DISABLED')),
    CONSTRAINT valid_security_question_status CHECK (security_question_status IN ('ENABLED', 'UNSET', 'LOCKED'))
);

-- Customers table (depends on workspaces) - WITH PAYMENT SYNC COLUMNS
CREATE TABLE IF NOT EXISTS customers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    num_id BIGSERIAL UNIQUE NOT NULL,
    web3auth_id VARCHAR(255) UNIQUE,
    external_id VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    phone VARCHAR(255),
    description TEXT,
    metadata JSONB,
    finished_onboarding BOOLEAN DEFAULT false,
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- API Keys table (depends on workspaces)
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(20), -- First part of key for identification
    access_level api_key_level NOT NULL DEFAULT 'read',
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Networks table (no dependencies)
CREATE TABLE networks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    network_type network_type NOT NULL,
    circle_network_type circle_network_type NOT NULL,
    rpc_id TEXT NOT NULL,
    block_explorer_url TEXT,
    chain_id INTEGER NOT NULL UNIQUE,
    is_testnet BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    -- Display information
    logo_url TEXT,
    display_name TEXT,
    chain_namespace TEXT DEFAULT 'eip155',
    -- Gas configuration
    base_fee_multiplier DECIMAL(4,2) DEFAULT 1.2,
    priority_fee_multiplier DECIMAL(4,2) DEFAULT 1.1,
    deployment_gas_limit TEXT DEFAULT '500000',
    token_transfer_gas_limit TEXT DEFAULT '100000',
    supports_eip1559 BOOLEAN DEFAULT true,
    gas_oracle_url TEXT,
    gas_refresh_interval_ms INTEGER DEFAULT 30000,
    gas_priority_levels JSONB DEFAULT '{"slow":{"max_fee_per_gas":"1000000000","max_priority_fee_per_gas":"100000000"},"standard":{"max_fee_per_gas":"2000000000","max_priority_fee_per_gas":"200000000"},"fast":{"max_fee_per_gas":"5000000000","max_priority_fee_per_gas":"500000000"}}'::jsonb,
    -- Network performance
    average_block_time_ms INTEGER DEFAULT 2000,
    peak_hours_multiplier DECIMAL(4,2) DEFAULT 1.5,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Wallets table (depends on workspaces, networks)
CREATE TABLE IF NOT EXISTS wallets (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id), -- Changed from account_id
    wallet_type TEXT NOT NULL,                -- 'wallet', 'circle_wallet', or 'web3auth'
    wallet_address TEXT NOT NULL,
    network_type network_type NOT NULL,
    network_id UUID REFERENCES networks(id),
    nickname TEXT,
    ens TEXT,
    is_primary BOOLEAN DEFAULT false,
    verified BOOLEAN DEFAULT false,
    last_used_at TIMESTAMP WITH TIME ZONE,
    web3auth_user_id VARCHAR(255),
    smart_account_type VARCHAR(50) CHECK (smart_account_type IN ('web3auth_eoa', 'web3auth_smart_account')),
    deployment_status VARCHAR(50) CHECK (deployment_status IN ('pending', 'deployed', 'failed')),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT check_wallet_type CHECK (wallet_type IN ('wallet', 'circle_wallet', 'web3auth'))
);

-- Circle Wallets Table (depends on wallets, circle_users)
CREATE TABLE IF NOT EXISTS circle_wallets (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    circle_user_id UUID NOT NULL REFERENCES circle_users(id),
    circle_wallet_id TEXT NOT NULL,
    chain_id INTEGER NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    UNIQUE(wallet_id),
    UNIQUE(circle_wallet_id)
);

-- Customer Wallets table (depends on customers)
CREATE TABLE customer_wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    wallet_address TEXT NOT NULL,
    network_type network_type NOT NULL,
    nickname TEXT,
    ens TEXT,
    is_primary BOOLEAN DEFAULT false,
    verified BOOLEAN DEFAULT false,
    last_used_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Products table (depends on workspaces, wallets) - WITH MERGED PRICE DATA
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    external_id VARCHAR(255), -- Provider's ID (Stripe ID, etc.)
    
    -- Product details
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    url TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    
    -- Product categorization
    product_type VARCHAR(50) DEFAULT 'base', -- 'base' or 'addon'
    product_group VARCHAR(100), -- Groups related products (e.g., "pro_plan" for monthly/yearly variants)
    
    -- Merged price fields (previously in prices table)
    price_type price_type NOT NULL DEFAULT 'recurring', -- 'recurring' or 'one_time'
    currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code), -- ISO 4217 currency code
    unit_amount_in_pennies INTEGER NOT NULL,
    interval_type interval_type, -- 'daily', 'weekly', 'monthly', 'yearly' (NULL for one_time)
    term_length INTEGER, -- Number of intervals, e.g., 12 for 12 months (NULL for one_time)
    price_nickname TEXT, -- Optional friendly name for the price
    price_external_id VARCHAR(255), -- Provider's price ID (e.g., Stripe price ID)
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending', 
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT products_price_type_check CHECK (
        (price_type = 'recurring' AND interval_type IS NOT NULL AND term_length IS NOT NULL AND term_length > 0) OR
        (price_type = 'one_time' AND interval_type IS NULL AND term_length IS NULL)
    ),
    
    -- Add unique constraint for external_id per workspace and provider
    UNIQUE(workspace_id, external_id, payment_provider)
);

-- Tokens table (depends on networks)
CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID NOT NULL REFERENCES networks(id),
    gas_token BOOLEAN NOT NULL DEFAULT false,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    contract_address TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT false,
    decimals INTEGER NOT NULL DEFAULT 2,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(network_id, contract_address)
);

-- Products Tokens table (depends on products, networks, tokens)
CREATE TABLE products_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    network_id UUID NOT NULL REFERENCES networks(id),
    token_id UUID NOT NULL REFERENCES tokens(id),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(product_id, network_id, token_id)
);

-- Product Addon Relationships table
-- Manages the relationships between base products and their available addons
CREATE TABLE product_addon_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    base_product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    addon_product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    
    -- Relationship details
    is_required BOOLEAN DEFAULT FALSE, -- Whether this addon is required with the base product
    max_quantity INTEGER, -- Maximum quantity allowed (NULL = unlimited)
    min_quantity INTEGER DEFAULT 0, -- Minimum quantity required
    
    -- Display order
    display_order INTEGER DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique relationships
    UNIQUE(base_product_id, addon_product_id),
    -- Ensure base products can't be addons to themselves
    CONSTRAINT different_products CHECK (base_product_id != addon_product_id),
    -- Ensure min/max quantity constraints are valid
    CONSTRAINT valid_quantity_range CHECK (
        (max_quantity IS NULL OR max_quantity > 0) AND
        min_quantity >= 0 AND
        (max_quantity IS NULL OR min_quantity <= max_quantity)
    )
);

-- Delegation Data table (no dependencies)
CREATE TABLE delegation_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delegate TEXT NOT NULL,
    delegator TEXT NOT NULL,
    authority TEXT NOT NULL,
    caveats JSONB NOT NULL DEFAULT '[]'::jsonb,
    salt TEXT NOT NULL,
    signature TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Subscriptions table (depends on customers, products, products_tokens, delegation_data, customer_wallets) - WITH PAYMENT SYNC COLUMNS
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    num_id BIGSERIAL UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id),
    product_id UUID NOT NULL REFERENCES products(id),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    -- price_id removed - pricing is now in products table
    product_token_id UUID NOT NULL REFERENCES products_tokens(id),
    external_id VARCHAR(255), -- Provider's ID (Stripe ID, etc.)
    token_amount INTEGER NOT NULL,
    delegation_id UUID NOT NULL REFERENCES delegation_data(id),
    customer_wallet_id UUID REFERENCES customer_wallets(id),
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    next_redemption_date TIMESTAMP WITH TIME ZONE,
    total_redemptions INT NOT NULL DEFAULT 0,
    total_amount_in_cents INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    -- Add unique constraint for external_id per workspace and provider
    UNIQUE(workspace_id, external_id, payment_provider)
);

-- Subscription Events table (depends on subscriptions)
CREATE TABLE subscription_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    event_type subscription_event_type NOT NULL,
    transaction_hash TEXT,
    amount_in_cents INT NOT NULL,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Subscription Line Items table
-- Tracks individual line items within a subscription (base product + addons)
CREATE TABLE subscription_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    
    -- Line item details
    line_item_type VARCHAR(50) NOT NULL DEFAULT 'base', -- 'base' or 'addon'
    quantity INTEGER NOT NULL DEFAULT 1,
    
    -- Pricing at time of subscription (snapshot)
    unit_amount_in_pennies INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code),
    price_type price_type NOT NULL,
    interval_type interval_type,
    
    -- Total calculation
    total_amount_in_pennies INTEGER NOT NULL, -- quantity * unit_amount
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure line item type matches product type
    CONSTRAINT valid_line_item_type CHECK (line_item_type IN ('base', 'addon'))
);

-- Invoices table (NEW - for storing payment invoices from providers)
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    customer_id UUID REFERENCES customers(id), -- Link to local customer if available
    subscription_id UUID REFERENCES subscriptions(id), -- Link to subscription if applicable
    
    -- External provider fields
    external_id TEXT NOT NULL, -- Provider's invoice ID (Stripe inv_xxx, Chargebee invoice_id, etc.)
    external_customer_id TEXT, -- Provider's customer ID
    external_subscription_id TEXT, -- Provider's subscription ID
    
    -- Invoice details
    status TEXT NOT NULL, -- 'draft', 'open', 'paid', 'void', 'uncollectible'
    collection_method TEXT, -- 'charge_automatically', 'send_invoice'
    amount_due INTEGER NOT NULL, -- Amount due in smallest currency unit (cents)
    amount_paid INTEGER NOT NULL DEFAULT 0, -- Amount paid in smallest currency unit
    amount_remaining INTEGER NOT NULL, -- Remaining amount due
    currency TEXT NOT NULL, -- ISO currency code ('usd', 'eur', etc.)
    
    -- Important dates
    due_date TIMESTAMP WITH TIME ZONE, -- When payment is due
    paid_at TIMESTAMP WITH TIME ZONE, -- When invoice was paid
    created_date TIMESTAMP WITH TIME ZONE NOT NULL, -- When invoice was created in provider
    
    -- Provider URLs and references
    invoice_pdf TEXT, -- URL to invoice PDF
    hosted_invoice_url TEXT, -- URL to hosted invoice page
    charge_id TEXT, -- External charge ID if paid
    payment_intent_id TEXT, -- External payment intent ID
    
    -- Invoice line items (simplified, could be separate table if needed)
    line_items JSONB DEFAULT '[]'::jsonb, -- Array of line items
    
    -- Tax and billing information
    tax_amount INTEGER DEFAULT 0, -- Total tax amount
    total_tax_amounts JSONB DEFAULT '[]'::jsonb, -- Detailed tax breakdown
    billing_reason TEXT, -- 'subscription_cycle', 'manual', etc.
    paid_out_of_band BOOLEAN DEFAULT false, -- Whether paid outside the provider
    
    -- Payment sync fields
    payment_provider TEXT, -- 'stripe', 'chargebee', etc.
    payment_sync_status TEXT DEFAULT 'pending', -- 'pending', 'synced', 'failed'
    payment_synced_at TIMESTAMP WITH TIME ZONE,
    
    -- Retry and processing
    attempt_count INTEGER DEFAULT 0, -- Number of payment attempts
    next_payment_attempt TIMESTAMP WITH TIME ZONE, -- Next retry attempt
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    UNIQUE(workspace_id, external_id, payment_provider)
);

-- Failed Subscription Attempts table (depends on customers, products, products_tokens, customer_wallets)
CREATE TABLE failed_subscription_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID REFERENCES customers(id),
    product_id UUID NOT NULL REFERENCES products(id),
    product_token_id UUID NOT NULL REFERENCES products_tokens(id),
    customer_wallet_id UUID REFERENCES customer_wallets(id),
    wallet_address TEXT NOT NULL,
    error_type subscription_event_type NOT NULL,
    error_message TEXT NOT NULL,
    error_details JSONB DEFAULT '{}'::jsonb,
    delegation_signature TEXT,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Payment Sync Sessions table (NEW - for tracking sync jobs)
CREATE TABLE IF NOT EXISTS payment_sync_sessions (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', 'recurly', etc.
    session_type VARCHAR(50) NOT NULL, -- 'initial_sync', 'partial_sync', 'delta_sync'
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed', 'cancelled'
    entity_types TEXT[] NOT NULL, -- ['customers', 'products', 'prices', 'subscriptions', 'invoices']
    config JSONB DEFAULT '{}',
    progress JSONB DEFAULT '{}',
    error_summary JSONB,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Payment Sync Events table (NEW - for detailed sync tracking)
CREATE TABLE IF NOT EXISTS payment_sync_events (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES payment_sync_sessions(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', etc.
    entity_type VARCHAR(50) NOT NULL, -- 'customer', 'product', 'price', 'subscription', 'invoice'
    entity_id UUID, -- Reference to the actual entity (customer_id, product_id, etc.)
    external_id VARCHAR(255), -- Provider's ID (Stripe ID, Chargebee ID, etc.)
    event_type VARCHAR(50) NOT NULL, -- 'sync_started', 'sync_completed', 'sync_failed', 'sync_skipped'
    event_message TEXT,
    event_details JSONB,
    -- NEW: Webhook-specific fields for multi-workspace webhook processing
    webhook_event_id VARCHAR(255), -- Stripe event ID (evt_xxx), Chargebee event ID, etc.
    provider_account_id VARCHAR(255), -- Provider Account ID for workspace routing (Stripe acct_xxx, Chargebee site_id, etc.)
    idempotency_key VARCHAR(255), -- For preventing duplicate processing (workspace_id + event_id)
    processing_attempts INTEGER DEFAULT 0, -- Number of processing attempts for retry logic
    signature_valid BOOLEAN, -- Whether webhook signature was validated
    occurred_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Workspace Payment Configurations Table
CREATE TABLE IF NOT EXISTS workspace_payment_configurations (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', etc.
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_test_mode BOOLEAN NOT NULL DEFAULT true, -- Whether using test/sandbox credentials
    configuration JSONB NOT NULL DEFAULT '{}', -- Encrypted configuration data (API keys, etc.)
    webhook_endpoint_url TEXT, -- The webhook URL for this workspace+provider
    webhook_secret_key TEXT, -- Webhook signing secret
    connected_account_id TEXT, -- External account ID (e.g., Stripe account ID)
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_webhook_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Ensure one active configuration per workspace per provider
    UNIQUE(workspace_id, provider_name, is_active) DEFERRABLE INITIALLY DEFERRED
);

-- Workspace Provider Account Mapping Table (NEW - for multi-workspace webhook support)
-- Generic table to map provider account IDs to workspaces (Stripe, Chargebee, PayPal, etc.)
CREATE TABLE IF NOT EXISTS workspace_provider_accounts (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', 'paypal', etc.
    provider_account_id VARCHAR(255) NOT NULL, -- Stripe Account ID (acct_xxx), Chargebee site_id, PayPal merchant_id, etc.
    account_type VARCHAR(50) NOT NULL, -- Provider-specific: 'standard', 'express', 'custom', 'platform' for Stripe
    is_active BOOLEAN NOT NULL DEFAULT true,
    environment VARCHAR(20) NOT NULL DEFAULT 'live', -- 'live', 'test', 'sandbox'
    display_name VARCHAR(255), -- Human-readable name for this provider account
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Ensure unique provider account per environment per provider
    UNIQUE(provider_name, provider_account_id, environment),
    -- Ensure workspace can have multiple provider accounts but track them uniquely
    UNIQUE(workspace_id, provider_name, provider_account_id, environment)
);

-- Indexes for workspace payment configurations
CREATE INDEX idx_workspace_payment_configurations_workspace_id ON workspace_payment_configurations(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_payment_configurations_provider ON workspace_payment_configurations(provider_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_payment_configurations_active ON workspace_payment_configurations(workspace_id, provider_name, is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_payment_configurations_last_sync ON workspace_payment_configurations(last_sync_at) WHERE deleted_at IS NULL;

-- Indexes for workspace provider accounts (NEW)
CREATE INDEX idx_workspace_provider_accounts_workspace_id ON workspace_provider_accounts(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_provider_accounts_provider ON workspace_provider_accounts(provider_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_provider_accounts_active ON workspace_provider_accounts(is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_provider_accounts_environment ON workspace_provider_accounts(environment) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspace_provider_accounts_lookup ON workspace_provider_accounts(provider_name, provider_account_id, environment, is_active) WHERE deleted_at IS NULL;

-- Add updated_at trigger for workspace_payment_configurations
CREATE TRIGGER set_workspace_payment_configurations_updated_at
    BEFORE UPDATE ON workspace_payment_configurations
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Add updated_at trigger for workspace_provider_accounts (NEW)
CREATE TRIGGER set_workspace_provider_accounts_updated_at
    BEFORE UPDATE ON workspace_provider_accounts
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Create trigger function to validate token network relationship
CREATE OR REPLACE FUNCTION validate_token_network()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM tokens 
        WHERE id = NEW.token_id 
        AND network_id = NEW.network_id
    ) THEN
        RAISE EXCEPTION 'Token % does not belong to network %', NEW.token_id, NEW.network_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for token network validation
CREATE TRIGGER validate_token_network_trigger
    BEFORE INSERT OR UPDATE ON products_tokens
    FOR EACH ROW
    EXECUTE FUNCTION validate_token_network();

-- SQL queries for Web3Auth functionality would be added to sqlc queries directory

-- Create indexes
-- (Order doesn't matter as much, but group by table for readability)

-- accounts
-- (Primary key index created automatically)

-- users
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_web3auth_id ON users(web3auth_id);
CREATE INDEX idx_users_verifier ON users(verifier);
CREATE INDEX idx_users_verifier_id ON users(verifier_id);
CREATE INDEX idx_users_account_id ON users(account_id);

-- workspaces
CREATE INDEX idx_workspaces_account_id ON workspaces(account_id);

-- circle_users
CREATE INDEX idx_circle_users_workspace_id ON circle_users(workspace_id);

-- customers
CREATE INDEX idx_customers_web3auth_id ON customers(web3auth_id);
CREATE INDEX idx_customers_num_id ON customers(num_id);

-- Workspace-Customer Association Table (Many-to-Many)
CREATE TABLE IF NOT EXISTS workspace_customers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(workspace_id, customer_id)
);

CREATE INDEX idx_workspace_customers_workspace_id ON workspace_customers(workspace_id);
CREATE INDEX idx_workspace_customers_customer_id ON workspace_customers(customer_id);
CREATE INDEX idx_workspace_customers_deleted_at ON workspace_customers(deleted_at);
CREATE INDEX idx_customers_payment_provider ON customers(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_customers_payment_sync_status ON customers(payment_sync_status) WHERE deleted_at IS NULL;

-- api_keys
CREATE INDEX idx_api_keys_workspace_id ON api_keys(workspace_id);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix);

-- networks
CREATE INDEX idx_networks_chain_id ON networks(chain_id);
CREATE INDEX idx_networks_active ON networks(active) WHERE deleted_at IS NULL;

-- wallets
CREATE INDEX idx_wallets_workspace_id ON wallets(workspace_id);
CREATE INDEX idx_wallets_address ON wallets(wallet_address);
CREATE INDEX idx_wallets_network_type ON wallets(network_type);
CREATE INDEX idx_wallets_is_primary ON wallets(is_primary) WHERE deleted_at IS NULL;
CREATE INDEX idx_wallets_network_id ON wallets(network_id);
CREATE INDEX idx_wallets_wallet_type ON wallets(wallet_type);
CREATE INDEX idx_wallets_web3auth_user_id ON wallets(web3auth_user_id);
CREATE INDEX idx_wallets_smart_account_type ON wallets(smart_account_type);
CREATE INDEX idx_wallets_deployment_status ON wallets(deployment_status);

-- circle_wallets
CREATE INDEX idx_circle_wallets_wallet_id ON circle_wallets(wallet_id);
CREATE INDEX idx_circle_wallets_circle_user_id ON circle_wallets(circle_user_id);
CREATE INDEX idx_circle_wallets_circle_wallet_id ON circle_wallets(circle_wallet_id);
CREATE INDEX idx_circle_wallets_state ON circle_wallets(state);

-- customer_wallets
CREATE INDEX idx_customer_wallets_customer_id ON customer_wallets(customer_id);
CREATE INDEX idx_customer_wallets_wallet_address ON customer_wallets(wallet_address);
CREATE INDEX idx_customer_wallets_network_type ON customer_wallets(network_type);
CREATE INDEX idx_customer_wallets_is_primary ON customer_wallets(is_primary) WHERE deleted_at IS NULL;
CREATE INDEX idx_customer_wallets_verified ON customer_wallets(verified) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX customer_wallets_customer_address_network_unique_idx ON customer_wallets(customer_id, wallet_address, network_type) WHERE deleted_at IS NULL;

-- products
CREATE INDEX idx_products_workspace_id ON products(workspace_id);
CREATE INDEX idx_products_wallet_id ON products(wallet_id);
CREATE INDEX idx_products_active ON products(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_created_at ON products(created_at);
CREATE INDEX idx_products_payment_provider ON products(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_payment_sync_status ON products(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_external_id ON products(external_id) WHERE deleted_at IS NULL;
-- New indexes for merged price fields
CREATE INDEX idx_products_price_type ON products(price_type);
CREATE INDEX idx_products_currency ON products(currency);
CREATE INDEX idx_products_product_type ON products(product_type);
CREATE INDEX idx_products_product_group ON products(product_group) WHERE product_group IS NOT NULL;

-- prices indexes removed - pricing is now in products table
-- CREATE INDEX idx_prices_product_id ON prices(product_id);
-- CREATE INDEX idx_prices_active ON prices(active) WHERE deleted_at IS NULL;
-- CREATE INDEX idx_prices_type ON prices(type);
-- CREATE INDEX idx_prices_currency ON prices(currency);
CREATE INDEX idx_fiat_currencies_active ON fiat_currencies(is_active, code);
-- CREATE INDEX idx_prices_payment_provider ON prices(payment_provider) WHERE deleted_at IS NULL;
-- CREATE INDEX idx_prices_payment_sync_status ON prices(payment_sync_status) WHERE deleted_at IS NULL;
-- CREATE INDEX idx_prices_external_id ON prices(external_id) WHERE deleted_at IS NULL;

-- tokens
CREATE INDEX idx_tokens_network_id ON tokens(network_id);
CREATE INDEX idx_tokens_active ON tokens(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_tokens_contract_address ON tokens(contract_address);

-- products_tokens
CREATE INDEX idx_products_tokens_product_id ON products_tokens(product_id);
CREATE INDEX idx_products_tokens_network_id ON products_tokens(network_id);
CREATE INDEX idx_products_tokens_token_id ON products_tokens(token_id);
CREATE INDEX idx_products_tokens_active ON products_tokens(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_tokens_composite ON products_tokens(product_id, network_id, active) WHERE deleted_at IS NULL;

-- product_addon_relationships
CREATE INDEX idx_product_addon_base ON product_addon_relationships(base_product_id);
CREATE INDEX idx_product_addon_addon ON product_addon_relationships(addon_product_id);
CREATE INDEX idx_product_addon_required ON product_addon_relationships(is_required) WHERE is_required = TRUE;

-- delegation_data
CREATE INDEX idx_delegation_data_delegator ON delegation_data(delegator);
CREATE INDEX idx_delegation_data_delegate ON delegation_data(delegate);

-- subscriptions
CREATE INDEX idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX idx_subscriptions_product_id ON subscriptions(product_id);
CREATE INDEX idx_subscriptions_product_token_id ON subscriptions(product_token_id);
CREATE INDEX idx_subscriptions_delegation_id ON subscriptions(delegation_id);
CREATE INDEX idx_subscriptions_customer_wallet_id ON subscriptions(customer_wallet_id);
-- CREATE INDEX idx_subscriptions_price_id ON subscriptions(price_id); -- Removed: price_id is now in products table
CREATE INDEX idx_subscriptions_status ON subscriptions(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_next_redemption_date ON subscriptions(next_redemption_date) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX idx_subscriptions_payment_provider ON subscriptions(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_payment_sync_status ON subscriptions(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_external_id ON subscriptions(external_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_num_id ON subscriptions(num_id);

-- subscription_events
CREATE INDEX idx_subscription_events_subscription_id ON subscription_events(subscription_id);
CREATE INDEX idx_subscription_events_event_type ON subscription_events(event_type);
CREATE INDEX idx_subscription_events_transaction_hash ON subscription_events(transaction_hash);
CREATE INDEX idx_subscription_events_occurred_at ON subscription_events(occurred_at);

-- subscription_line_items
CREATE INDEX idx_subscription_line_items_subscription ON subscription_line_items(subscription_id);
CREATE INDEX idx_subscription_line_items_product ON subscription_line_items(product_id);
CREATE INDEX idx_subscription_line_items_type ON subscription_line_items(line_item_type);
CREATE INDEX idx_subscription_line_items_active ON subscription_line_items(is_active) WHERE is_active = TRUE;

-- failed_subscription_attempts
CREATE INDEX idx_failed_subscription_attempts_customer_id ON failed_subscription_attempts(customer_id);
CREATE INDEX idx_failed_subscription_attempts_product_id ON failed_subscription_attempts(product_id);
CREATE INDEX idx_failed_subscription_attempts_error_type ON failed_subscription_attempts(error_type);
CREATE INDEX idx_failed_subscription_attempts_wallet_address ON failed_subscription_attempts(wallet_address);
CREATE INDEX idx_failed_subscription_attempts_occurred_at ON failed_subscription_attempts(occurred_at);

-- payment_sync_sessions (NEW INDEXES)
CREATE INDEX idx_payment_sync_sessions_workspace_id ON payment_sync_sessions(workspace_id);
CREATE INDEX idx_payment_sync_sessions_provider ON payment_sync_sessions(provider_name);
CREATE INDEX idx_payment_sync_sessions_status ON payment_sync_sessions(status) WHERE deleted_at IS NULL;

-- payment_sync_events (NEW INDEXES)
CREATE INDEX idx_payment_sync_events_session_id ON payment_sync_events(session_id);
CREATE INDEX idx_payment_sync_events_provider ON payment_sync_events(provider_name);
CREATE INDEX idx_payment_sync_events_entity_type ON payment_sync_events(entity_type);
CREATE INDEX idx_payment_sync_events_external_id ON payment_sync_events(external_id);
-- NEW: Webhook-specific indexes for multi-workspace processing
CREATE INDEX idx_payment_sync_events_webhook_id ON payment_sync_events(webhook_event_id) WHERE webhook_event_id IS NOT NULL;
CREATE INDEX idx_payment_sync_events_provider_account ON payment_sync_events(provider_account_id) WHERE provider_account_id IS NOT NULL;
CREATE INDEX idx_payment_sync_events_idempotency ON payment_sync_events(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_payment_sync_events_processing_attempts ON payment_sync_events(processing_attempts) WHERE processing_attempts > 0;

-- invoices
CREATE INDEX idx_invoices_workspace_id ON invoices(workspace_id);
CREATE INDEX idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX idx_invoices_subscription_id ON invoices(subscription_id);
CREATE INDEX idx_invoices_status ON invoices(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_payment_provider ON invoices(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_payment_sync_status ON invoices(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_external_id ON invoices(external_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_external_customer_id ON invoices(external_customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_external_subscription_id ON invoices(external_subscription_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_due_date ON invoices(due_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_paid_at ON invoices(paid_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_created_date ON invoices(created_date) WHERE deleted_at IS NULL;

-- Insert test data for development (Order matters!)

INSERT INTO accounts (name, account_type, business_name, business_type)
VALUES 
    (
        'Admin Account',
        'admin',
        'Cyphera Admin',
        'Corporation'
    )
ON CONFLICT DO NOTHING;

INSERT INTO users (web3auth_id, verifier, verifier_id, email, first_name, last_name, display_name, account_id, role, is_account_owner)
VALUES 
    ('web3auth|admin', 'custom', 'admin@cyphera.com', 'admin@cyphera.com', 'Admin', 'User', 'Admin User',
     (SELECT id FROM accounts WHERE name = 'Admin Account'), 'admin', true)
ON CONFLICT DO NOTHING;

INSERT INTO workspaces (account_id, name, description, business_name)
VALUES 
    (
        (SELECT id FROM accounts WHERE name = 'Admin Account'),
        'Admin Workspace',
        'Admin workspace for development',
        'Cyphera Admin'
    )
ON CONFLICT DO NOTHING;

-- Insert admin API key with proper bcrypt hash
-- The actual key is: cyk_admin_test_key_do_not_use_in_production
INSERT INTO api_keys (workspace_id, name, key_hash, key_prefix, access_level)
VALUES 
    (
        (SELECT id FROM workspaces WHERE name = 'Admin Workspace'),
        'Admin API Key',
        '$2a$12$vHX/kQ9dEUoxkt1CIeDQ8O5/vGr6ZdAQAyZ40b.sc6HR.o6nqs6li',
        'cyk_admin***',
        'admin'
    )
ON CONFLICT DO NOTHING;

INSERT INTO networks (name, type, network_type, rpc_id, circle_network_type, chain_id, is_testnet, active, block_explorer_url, logo_url, display_name, chain_namespace, base_fee_multiplier, priority_fee_multiplier, deployment_gas_limit, token_transfer_gas_limit, supports_eip1559, average_block_time_ms, gas_priority_levels)
VALUES 
    ('Ethereum Sepolia', 'Sepolia', 'evm', 'sepolia', 'ETH-SEPOLIA', 11155111, true, false, 'https://sepolia.etherscan.io', 'https://cryptologos.cc/logos/ethereum-eth-logo.png', 'Ethereum Sepolia', 'eip155', 1.2, 1.1, '500000', '100000', true, 12000, '{"slow":{"max_fee_per_gas":"1000000000","max_priority_fee_per_gas":"100000000"},"standard":{"max_fee_per_gas":"2000000000","max_priority_fee_per_gas":"200000000"},"fast":{"max_fee_per_gas":"5000000000","max_priority_fee_per_gas":"500000000"}}'),
    ('Ethereum Mainnet', 'Mainnet', 'evm', 'eth', 'ETH', 1, false, false, 'https://etherscan.io', 'https://cryptologos.cc/logos/ethereum-eth-logo.png', 'Ethereum', 'eip155', 1.2, 1.1, '500000', '100000', true, 12000, '{"slow":{"max_fee_per_gas":"20000000000","max_priority_fee_per_gas":"1000000000"},"standard":{"max_fee_per_gas":"30000000000","max_priority_fee_per_gas":"2000000000"},"fast":{"max_fee_per_gas":"50000000000","max_priority_fee_per_gas":"3000000000"}}'),
    ('Polygon Amoy', 'Amoy', 'evm', 'amoy', 'MATIC-AMOY', 80002, true, false, 'https://www.oklink.com/amoy', 'https://cryptologos.cc/logos/polygon-matic-logo.png', 'Polygon Amoy', 'eip155', 1.3, 1.2, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"30000000000","max_priority_fee_per_gas":"30000000000"},"standard":{"max_fee_per_gas":"35000000000","max_priority_fee_per_gas":"35000000000"},"fast":{"max_fee_per_gas":"40000000000","max_priority_fee_per_gas":"40000000000"}}'), 
    ('Polygon Mainnet', 'Mainnet', 'evm', 'polygon', 'MATIC', 137, false, false, 'https://polygonscan.com', 'https://cryptologos.cc/logos/polygon-matic-logo.png', 'Polygon', 'eip155', 1.3, 1.2, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"30000000000","max_priority_fee_per_gas":"30000000000"},"standard":{"max_fee_per_gas":"35000000000","max_priority_fee_per_gas":"35000000000"},"fast":{"max_fee_per_gas":"40000000000","max_priority_fee_per_gas":"40000000000"}}'),
    ('Arbitrum Sepolia', 'Sepolia', 'evm', 'arbitrum-sepolia', 'ARB-SEPOLIA', 421614, true, false, 'https://sepolia.arbiscan.io', 'https://cryptologos.cc/logos/arbitrum-arb-logo.png', 'Arbitrum Sepolia', 'eip155', 1.1, 1.1, '1000000', '150000', true, 250, '{"slow":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"0"},"standard":{"max_fee_per_gas":"150000000","max_priority_fee_per_gas":"0"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"0"}}'),
    ('Arbitrum One', 'Mainnet', 'evm', 'arbitrum-mainnet', 'ARB', 42161, false, false, 'https://arbiscan.io', 'https://cryptologos.cc/logos/arbitrum-arb-logo.png', 'Arbitrum', 'eip155', 1.1, 1.1, '1000000', '150000', true, 250, '{"slow":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"0"},"standard":{"max_fee_per_gas":"150000000","max_priority_fee_per_gas":"0"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"0"}}'),
    ('Base Sepolia', 'Sepolia', 'evm', 'base-sepolia', 'BASE-SEPOLIA', 84532, true, true, 'https://sepolia.basescan.org', 'https://basescan.org/images/svg/logos/chain-light.svg', 'Base Sepolia', 'eip155', 1.2, 1.1, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"50000000","max_priority_fee_per_gas":"50000000"},"standard":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"100000000"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"150000000"}}'),
    ('Base Mainnet', 'Mainnet', 'evm', 'base-mainnet', 'BASE', 8453, false, false, 'https://basescan.org', 'https://basescan.org/images/svg/logos/chain-light.svg', 'Base', 'eip155', 1.2, 1.1, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"50000000","max_priority_fee_per_gas":"50000000"},"standard":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"100000000"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"150000000"}}'),
    ('Optimism Sepolia', 'Sepolia', 'evm', 'optimism-sepolia', 'OP-SEPOLIA', 11155420, true, false, 'https://sepolia.optimism.io', 'https://cryptologos.cc/logos/optimism-ethereum-op-logo.png', 'Optimism Sepolia', 'eip155', 1.2, 1.1, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"50000000","max_priority_fee_per_gas":"50000000"},"standard":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"100000000"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"150000000"}}'),
    ('Optimism Mainnet', 'Mainnet', 'evm', 'optimism-mainnet', 'OP', 10, false, false, 'https://optimistic.etherscan.io', 'https://cryptologos.cc/logos/optimism-ethereum-op-logo.png', 'Optimism', 'eip155', 1.2, 1.1, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"50000000","max_priority_fee_per_gas":"50000000"},"standard":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"100000000"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"150000000"}}'),
    ('Unichain Sepolia', 'Sepolia', 'evm', 'unichain-sepolia', 'UNI-SEPOLIA', 1301, true, false, 'https://sepolia.unichain.io', 'https://cryptologos.cc/logos/uniswap-uni-logo.png', 'Unichain Sepolia', 'eip155', 1.2, 1.1, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"50000000","max_priority_fee_per_gas":"50000000"},"standard":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"100000000"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"150000000"}}'),
    ('Unichain Mainnet', 'Mainnet', 'evm', 'unichain-mainnet', 'UNI', 130, false, false, 'https://unichain.io', 'https://cryptologos.cc/logos/uniswap-uni-logo.png', 'Unichain', 'eip155', 1.2, 1.1, '500000', '100000', true, 2000, '{"slow":{"max_fee_per_gas":"50000000","max_priority_fee_per_gas":"50000000"},"standard":{"max_fee_per_gas":"100000000","max_priority_fee_per_gas":"100000000"},"fast":{"max_fee_per_gas":"200000000","max_priority_fee_per_gas":"150000000"}}')
ON CONFLICT DO NOTHING;

INSERT INTO tokens (network_id, name, symbol, contract_address, gas_token, active, decimals)
VALUES 
    -- Ethereum Sepolia tokens
    ((SELECT id FROM networks WHERE chain_id = 11155111 AND deleted_at IS NULL), 'USD Coin', 'USDC', '0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238', false, true, 6),
    ((SELECT id FROM networks WHERE chain_id = 11155111 AND deleted_at IS NULL), 'Ethereum', 'ETH', '0xd38E5c25935291fFD51C9d66C3B7384494bb099A', true, true, 18),
    ((SELECT id FROM networks WHERE chain_id = 84532 AND deleted_at IS NULL), 'USD Coin', 'USDC', '0x036CbD53842c5426634e7929541eC2318f3dCF7e', false, true, 6),
    ((SELECT id FROM networks WHERE chain_id = 84532 AND deleted_at IS NULL), 'Ethereum', 'ETH', '0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee', true, true, 18)
ON CONFLICT DO NOTHING;


-- Create function for updating updated_at timestamp
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add triggers for updated_at
CREATE TRIGGER set_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_wallets_updated_at
    BEFORE UPDATE ON wallets
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_workspaces_updated_at
    BEFORE UPDATE ON workspaces
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_api_keys_updated_at
    BEFORE UPDATE ON api_keys
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_products_updated_at
    BEFORE UPDATE ON products
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_networks_updated_at
    BEFORE UPDATE ON networks
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_tokens_updated_at
    BEFORE UPDATE ON tokens
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_products_tokens_updated_at
    BEFORE UPDATE ON products_tokens
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_product_addon_relationships_updated_at
    BEFORE UPDATE ON product_addon_relationships
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_delegation_data_updated_at
    BEFORE UPDATE ON delegation_data
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_subscription_events_updated_at
    BEFORE UPDATE ON subscription_events
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_subscription_line_items_updated_at
    BEFORE UPDATE ON subscription_line_items
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_customer_wallets_updated_at
    BEFORE UPDATE ON customer_wallets
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_failed_subscription_attempts_updated_at
    BEFORE UPDATE ON failed_subscription_attempts
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_circle_users_updated_at
    BEFORE UPDATE ON circle_users
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_circle_wallets_updated_at
    BEFORE UPDATE ON circle_wallets
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- CREATE TRIGGER set_prices_updated_at
--     BEFORE UPDATE ON prices
--     FOR EACH ROW
--     EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Add triggers for payment sync tables
CREATE TRIGGER set_payment_sync_sessions_updated_at
    BEFORE UPDATE ON payment_sync_sessions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================
-- PLATFORM ENHANCEMENT TABLES
-- ============================================

-- Payments table - Core payment tracking
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    invoice_id UUID REFERENCES invoices(id),
    subscription_id UUID REFERENCES subscriptions(id),
    subscription_event UUID REFERENCES subscription_events(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    
    -- Payment details
    amount_in_cents BIGINT NOT NULL, -- Total amount including gas if customer pays
    currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code),
    status VARCHAR(50) NOT NULL, -- pending, processing, completed, failed, refunded
    payment_method VARCHAR(50) NOT NULL, -- crypto, card, bank_transfer
    
    -- Crypto specific
    transaction_hash VARCHAR(255),
    network_id UUID REFERENCES networks(id),
    token_id UUID REFERENCES tokens(id),
    crypto_amount DECIMAL(36,18), -- Actual token amount transferred
    exchange_rate DECIMAL(20,8), -- Fiat to crypto rate at time of payment
    
    -- Gas fee reference (details in gas_fee_payments table)
    has_gas_fee BOOLEAN DEFAULT FALSE,
    gas_fee_usd_cents BIGINT, -- Quick reference for total gas cost
    gas_sponsored BOOLEAN DEFAULT FALSE,
    
    -- External references
    external_payment_id VARCHAR(255),
    payment_provider VARCHAR(50), -- circle, stripe, internal
    
    -- Financial breakdown
    product_amount_cents BIGINT NOT NULL, -- Amount for product/service
    tax_amount_cents BIGINT DEFAULT 0,
    gas_amount_cents BIGINT DEFAULT 0, -- Gas amount if customer pays
    discount_amount_cents BIGINT DEFAULT 0,
    
    -- Timestamps
    initiated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    failed_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_transaction_hash UNIQUE(transaction_hash),
    CONSTRAINT unique_external_payment UNIQUE(workspace_id, external_payment_id, payment_provider),
    CONSTRAINT check_amount_breakdown CHECK (
        amount_in_cents = product_amount_cents + tax_amount_cents + gas_amount_cents - discount_amount_cents
    )
);

-- Indexes for performance
CREATE INDEX idx_payments_workspace_status ON payments(workspace_id, status);
CREATE INDEX idx_payments_customer ON payments(customer_id);
CREATE INDEX idx_payments_completed_at ON payments(workspace_id, completed_at);
CREATE INDEX idx_payments_transaction_hash ON payments(transaction_hash) WHERE transaction_hash IS NOT NULL;

-- Invoice Line Items table
CREATE TABLE invoice_line_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    
    -- Line item details
    description TEXT NOT NULL,
    quantity DECIMAL(10,4) NOT NULL DEFAULT 1,
    unit_amount_in_cents BIGINT NOT NULL,
    amount_in_cents BIGINT NOT NULL, -- quantity * unit_amount
    
    -- Currency details
    fiat_currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code), -- ISO 4217 currency code
    
    -- References
    subscription_id UUID REFERENCES subscriptions(id),
    product_id UUID REFERENCES products(id),
    -- price_id removed - pricing is now in products table
    
    -- Crypto payment details
    network_id UUID REFERENCES networks(id),
    token_id UUID REFERENCES tokens(id),
    crypto_amount DECIMAL(36,18), -- Actual token amount (supports up to 18 decimals)
    exchange_rate DECIMAL(20,8), -- Fiat to crypto rate at time of invoice
    
    -- Tax
    tax_rate DECIMAL(5,4) DEFAULT 0, -- 0.0000 to 0.9999 (0% to 99.99%)
    tax_amount_in_cents BIGINT DEFAULT 0,
    tax_crypto_amount DECIMAL(36,18), -- Tax amount in crypto
    
    -- Period for subscription items
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,
    
    -- Gas fee tracking
    line_item_type VARCHAR(50) DEFAULT 'product', -- 'product', 'gas_fee', 'tax', 'discount'
    gas_fee_payment_id UUID, -- Will reference gas_fee_payments(id) when created
    is_gas_sponsored BOOLEAN DEFAULT FALSE,
    gas_sponsor_type VARCHAR(50),
    gas_sponsor_name VARCHAR(255), -- Human readable sponsor name
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_line_items_invoice ON invoice_line_items(invoice_id);
CREATE INDEX idx_line_items_token ON invoice_line_items(token_id) WHERE token_id IS NOT NULL;
CREATE INDEX idx_line_items_currency ON invoice_line_items(fiat_currency, invoice_id);
CREATE INDEX idx_line_items_type ON invoice_line_items(line_item_type) WHERE line_item_type != 'product';

-- Dashboard Metrics table
CREATE TABLE dashboard_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Time period
    metric_date DATE NOT NULL,
    metric_type VARCHAR(50) NOT NULL, -- hourly, daily, weekly, monthly, yearly
    metric_hour INTEGER, -- 0-23 for hourly metrics
    
    -- Currency
    fiat_currency VARCHAR(3) NOT NULL REFERENCES fiat_currencies(code),
    
    -- Revenue metrics (stored in cents)
    mrr_cents BIGINT DEFAULT 0,
    arr_cents BIGINT DEFAULT 0,
    total_revenue_cents BIGINT DEFAULT 0,
    new_revenue_cents BIGINT DEFAULT 0,
    expansion_revenue_cents BIGINT DEFAULT 0, -- Upsells/upgrades
    contraction_revenue_cents BIGINT DEFAULT 0, -- Downgrades
    
    -- Customer metrics
    total_customers INTEGER DEFAULT 0,
    new_customers INTEGER DEFAULT 0,
    churned_customers INTEGER DEFAULT 0,
    reactivated_customers INTEGER DEFAULT 0,
    
    -- Subscription metrics
    active_subscriptions INTEGER DEFAULT 0,
    new_subscriptions INTEGER DEFAULT 0,
    cancelled_subscriptions INTEGER DEFAULT 0,
    paused_subscriptions INTEGER DEFAULT 0,
    trial_subscriptions INTEGER DEFAULT 0,
    
    -- Calculated rates
    churn_rate DECIMAL(5,4) DEFAULT 0, -- 0.0000 to 0.9999
    growth_rate DECIMAL(5,4) DEFAULT 0,
    ltv_avg_cents BIGINT DEFAULT 0, -- Average customer lifetime value
    
    -- Payment metrics
    successful_payments INTEGER DEFAULT 0,
    failed_payments INTEGER DEFAULT 0,
    pending_payments INTEGER DEFAULT 0,
    total_payment_volume_cents BIGINT DEFAULT 0,
    avg_payment_size_cents BIGINT DEFAULT 0,
    
    -- Crypto-specific metrics
    total_gas_fees_cents BIGINT DEFAULT 0,
    sponsored_gas_fees_cents BIGINT DEFAULT 0, -- Gas fees paid by merchant
    customer_gas_fees_cents BIGINT DEFAULT 0, -- Gas fees paid by customer
    avg_gas_fee_cents BIGINT DEFAULT 0,
    gas_sponsorship_rate DECIMAL(5,2), -- Percentage of gas sponsored
    unique_wallet_addresses INTEGER DEFAULT 0,
    new_wallet_addresses INTEGER DEFAULT 0,
    
    -- Network breakdown (JSONB for flexibility)
    network_metrics JSONB DEFAULT '{}', -- {ethereum: {payments: 10, volume_cents: 1000}, polygon: {...}}
    token_metrics JSONB DEFAULT '{}', -- {USDC: {payments: 10, volume_cents: 1000}, USDT: {...}}
    
    -- Performance metrics
    avg_payment_confirmation_time_seconds INTEGER,
    payment_success_rate DECIMAL(5,4) DEFAULT 0, -- 0.0000 to 1.0000
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_workspace_metric UNIQUE(workspace_id, metric_date, metric_type, metric_hour, fiat_currency)
);

CREATE INDEX idx_metrics_workspace_date ON dashboard_metrics(workspace_id, metric_date DESC, metric_type);
CREATE INDEX idx_metrics_hourly ON dashboard_metrics(workspace_id, metric_date, metric_hour) WHERE metric_type = 'hourly';
CREATE INDEX idx_metrics_currency ON dashboard_metrics(workspace_id, fiat_currency, metric_date DESC);

-- Payment Links table
CREATE TABLE payment_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Link details
    slug VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, inactive, expired
    
    -- Payment configuration
    product_id UUID REFERENCES products(id),
    -- price_id removed - pricing is now in products table
    amount_in_cents BIGINT, -- For one-time custom amounts
    currency VARCHAR(3) REFERENCES fiat_currencies(code),
    payment_type VARCHAR(50) DEFAULT 'one_time', -- one_time, recurring
    
    -- Customer collection
    collect_email BOOLEAN DEFAULT true,
    collect_shipping BOOLEAN DEFAULT false,
    collect_name BOOLEAN DEFAULT true,
    
    -- Expiration
    expires_at TIMESTAMP WITH TIME ZONE,
    max_uses INTEGER,
    used_count INTEGER DEFAULT 0,
    
    -- Success behavior
    redirect_url TEXT, -- Where to redirect after successful payment
    
    -- QR Code
    qr_code_url TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_payment_links_slug ON payment_links(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_links_workspace ON payment_links(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_links_status ON payment_links(status, expires_at) WHERE deleted_at IS NULL;

-- Gas Fee Payments table
-- Stores detailed gas fee information for crypto payments
-- One-to-one relationship with payments table when has_gas_fee = true
CREATE TABLE gas_fee_payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payment_id UUID NOT NULL REFERENCES payments(id) UNIQUE, -- One-to-one with payments
    
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
    
    -- Network specifics
    network_id UUID NOT NULL REFERENCES networks(id),
    block_number BIGINT,
    block_timestamp TIMESTAMP WITH TIME ZONE,
    
    -- Conversion rates at time of payment
    eth_usd_price DECIMAL(10, 2), -- ETH price in USD
    token_usd_price DECIMAL(10, 2), -- Gas token price in USD
    gas_fee_usd_cents BIGINT, -- Calculated USD value
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_payment_gas FOREIGN KEY (payment_id) 
        REFERENCES payments(id) ON DELETE CASCADE
);

CREATE INDEX idx_gas_fee_payments_sponsor ON gas_fee_payments(sponsor_type, sponsor_id);
CREATE INDEX idx_gas_fee_payments_created ON gas_fee_payments(created_at);

-- Gas Sponsorship Configs table
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
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(workspace_id)
);

-- Update invoice_line_items to add foreign key for gas_fee_payments
ALTER TABLE invoice_line_items 
ADD CONSTRAINT fk_gas_fee_payment 
FOREIGN KEY (gas_fee_payment_id) 
REFERENCES gas_fee_payments(id);

-- Add constraint to ensure gas line items have proper references
ALTER TABLE invoice_line_items 
ADD CONSTRAINT chk_gas_line_item 
CHECK (
    (line_item_type != 'gas_fee') OR 
    (line_item_type = 'gas_fee' AND gas_fee_payment_id IS NOT NULL)
);

-- Update invoices table with new columns
ALTER TABLE invoices 
ADD COLUMN IF NOT EXISTS invoice_number VARCHAR(255),
ADD COLUMN IF NOT EXISTS subtotal_cents BIGINT,
ADD COLUMN IF NOT EXISTS discount_cents BIGINT DEFAULT 0,
ADD COLUMN IF NOT EXISTS payment_link_id UUID REFERENCES payment_links(id),
ADD COLUMN IF NOT EXISTS delegation_address VARCHAR(255),
ADD COLUMN IF NOT EXISTS qr_code_data TEXT,
-- Tax fields
ADD COLUMN IF NOT EXISTS tax_amount_cents BIGINT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS tax_details JSONB DEFAULT '[]'::jsonb, -- Array of tax calculations
ADD COLUMN IF NOT EXISTS customer_tax_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS customer_jurisdiction_id UUID, -- Will reference tax_jurisdictions(id) when created
ADD COLUMN IF NOT EXISTS reverse_charge_applies BOOLEAN DEFAULT FALSE;

-- Add unique constraint for invoice numbers per workspace
ALTER TABLE invoices 
ADD CONSTRAINT unique_workspace_invoice_number 
UNIQUE(workspace_id, invoice_number);

-- Add currency to subscriptions table
ALTER TABLE subscriptions
ADD COLUMN IF NOT EXISTS currency VARCHAR(3) REFERENCES fiat_currencies(code);

-- Update customers table for tax support
ALTER TABLE customers 
ADD COLUMN IF NOT EXISTS tax_jurisdiction_id UUID, -- Will reference tax_jurisdictions(id) when created
ADD COLUMN IF NOT EXISTS tax_id VARCHAR(255), -- VAT number, EIN, etc.
ADD COLUMN IF NOT EXISTS tax_id_type VARCHAR(50), -- 'vat', 'ein', 'gst', etc.
ADD COLUMN IF NOT EXISTS tax_id_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS tax_id_verified_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS is_business BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS business_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS billing_country VARCHAR(2), -- ISO country code
ADD COLUMN IF NOT EXISTS billing_state VARCHAR(50),
ADD COLUMN IF NOT EXISTS billing_city VARCHAR(255),
ADD COLUMN IF NOT EXISTS billing_postal_code VARCHAR(20);

-- Create indexes for new customer fields
CREATE INDEX IF NOT EXISTS idx_customers_billing_location ON customers(billing_country, billing_state);

-- Add triggers for new tables
CREATE TRIGGER set_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_invoice_line_items_updated_at
    BEFORE UPDATE ON invoice_line_items
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_dashboard_metrics_updated_at
    BEFORE UPDATE ON dashboard_metrics
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_payment_links_updated_at
    BEFORE UPDATE ON payment_links
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_gas_sponsorship_configs_updated_at
    BEFORE UPDATE ON gas_sponsorship_configs
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================================================
-- DUNNING MANAGEMENT TABLES
-- ============================================================================

-- Dunning configuration per workspace
CREATE TABLE dunning_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Configuration settings
    is_active BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false, -- Default config for workspace
    
    -- Retry settings
    max_retry_attempts INTEGER NOT NULL DEFAULT 4,
    retry_interval_days INTEGER[] NOT NULL DEFAULT ARRAY[3, 7, 7, 7], -- Days between each retry
    
    -- Actions for each attempt
    attempt_actions JSONB NOT NULL DEFAULT '[]'::jsonb, 
    -- Format: [{attempt: 1, actions: ["email", "in_app"], email_template_id: "uuid"}, ...]
    
    -- Final action after all retries fail
    final_action VARCHAR(50) NOT NULL DEFAULT 'cancel', -- cancel, pause, downgrade
    final_action_config JSONB DEFAULT '{}'::jsonb, -- Config for final action (e.g., downgrade_to_plan_id)
    
    -- Email settings
    send_pre_dunning_reminder BOOLEAN DEFAULT true,
    pre_dunning_days INTEGER DEFAULT 3, -- Days before payment due date
    
    -- Customer communication
    allow_customer_retry BOOLEAN DEFAULT true, -- Allow customers to manually retry payment
    grace_period_hours INTEGER DEFAULT 24, -- Hours after failure before first retry
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Dunning campaigns for individual subscriptions/payments
CREATE TABLE dunning_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    configuration_id UUID NOT NULL REFERENCES dunning_configurations(id),
    
    -- Target of dunning
    subscription_id UUID REFERENCES subscriptions(id),
    payment_id UUID REFERENCES payments(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    
    -- Campaign details
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, paused, completed, cancelled
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Tracking
    current_attempt INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP WITH TIME ZONE,
    last_retry_at TIMESTAMP WITH TIME ZONE,
    
    -- Results
    recovered BOOLEAN DEFAULT false,
    recovered_at TIMESTAMP WITH TIME ZONE,
    recovered_amount_cents BIGINT,
    final_action_taken VARCHAR(50), -- What action was taken if not recovered
    final_action_at TIMESTAMP WITH TIME ZONE,
    
    -- Original failure details
    original_failure_reason TEXT,
    original_amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_dunning_target CHECK (
        (subscription_id IS NOT NULL AND payment_id IS NULL) OR 
        (subscription_id IS NULL AND payment_id IS NOT NULL)
    )
);

-- Dunning attempt history
CREATE TABLE dunning_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES dunning_campaigns(id),
    
    attempt_number INTEGER NOT NULL,
    attempt_type VARCHAR(50) NOT NULL, -- retry_payment, send_email, send_sms, in_app_notification
    
    -- Attempt details
    status VARCHAR(50) NOT NULL, -- pending, processing, success, failed
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Payment retry details (if applicable)
    payment_id UUID REFERENCES payments(id),
    payment_status VARCHAR(50),
    payment_error TEXT,
    
    -- Communication details (if applicable)
    communication_type VARCHAR(50), -- email, sms, in_app
    communication_sent BOOLEAN DEFAULT false,
    communication_error TEXT,
    email_template_id UUID,
    
    -- Response tracking
    customer_response VARCHAR(50), -- clicked, opened, retried_manually
    customer_response_at TIMESTAMP WITH TIME ZONE,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Dunning email templates
CREATE TABLE dunning_email_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    name VARCHAR(255) NOT NULL,
    template_type VARCHAR(50) NOT NULL, -- pre_dunning, attempt_1, attempt_2, final_notice, recovery_success
    
    -- Email content
    subject VARCHAR(500) NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT, -- Plain text version
    
    -- Variables available: {{customer_name}}, {{amount}}, {{retry_date}}, {{product_name}}, etc.
    available_variables JSONB DEFAULT '[]'::jsonb,
    
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Dunning analytics aggregated data
CREATE TABLE dunning_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    
    -- Time period
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    period_type VARCHAR(20) NOT NULL, -- daily, weekly, monthly
    
    -- Campaign metrics
    total_campaigns_started INTEGER DEFAULT 0,
    total_campaigns_recovered INTEGER DEFAULT 0,
    total_campaigns_lost INTEGER DEFAULT 0,
    recovery_rate DECIMAL(5,4) DEFAULT 0, -- 0.0000 to 1.0000
    
    -- Financial metrics
    total_at_risk_cents BIGINT DEFAULT 0,
    total_recovered_cents BIGINT DEFAULT 0,
    total_lost_cents BIGINT DEFAULT 0,
    
    -- Attempt metrics
    total_payment_retries INTEGER DEFAULT 0,
    successful_payment_retries INTEGER DEFAULT 0,
    total_emails_sent INTEGER DEFAULT 0,
    email_open_rate DECIMAL(5,4) DEFAULT 0,
    email_click_rate DECIMAL(5,4) DEFAULT 0,
    
    -- Recovery by attempt number
    recovery_by_attempt JSONB DEFAULT '{}'::jsonb, -- {1: 20, 2: 15, 3: 10, 4: 5}
    
    -- Average time to recovery
    avg_hours_to_recovery DECIMAL(10,2),
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_workspace_analytics_period UNIQUE(workspace_id, period_start, period_end, period_type)
);

-- Indexes for dunning tables
CREATE INDEX idx_dunning_configs_workspace ON dunning_configurations(workspace_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_dunning_configs_default ON dunning_configurations(workspace_id, is_default) WHERE is_default = true AND deleted_at IS NULL;
CREATE INDEX idx_dunning_campaigns_workspace ON dunning_campaigns(workspace_id, status);
CREATE INDEX idx_dunning_campaigns_customer ON dunning_campaigns(customer_id);
CREATE INDEX idx_dunning_campaigns_next_retry ON dunning_campaigns(next_retry_at) WHERE status = 'active';
CREATE INDEX idx_dunning_attempts_campaign ON dunning_attempts(campaign_id);
CREATE INDEX idx_dunning_templates_workspace ON dunning_email_templates(workspace_id, template_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_dunning_analytics_workspace_period ON dunning_analytics(workspace_id, period_type, period_start DESC);

-- Triggers for dunning tables
CREATE TRIGGER set_dunning_configurations_updated_at
    BEFORE UPDATE ON dunning_configurations
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_dunning_campaigns_updated_at
    BEFORE UPDATE ON dunning_campaigns
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_dunning_email_templates_updated_at
    BEFORE UPDATE ON dunning_email_templates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_dunning_analytics_updated_at
    BEFORE UPDATE ON dunning_analytics
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- =====================================================
-- SUBSCRIPTION MANAGEMENT TABLES
-- =====================================================

-- First, update the subscriptions table to add cancellation and pause fields
ALTER TABLE subscriptions 
ADD COLUMN IF NOT EXISTS cancel_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS cancellation_reason TEXT,
ADD COLUMN IF NOT EXISTS paused_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS pause_ends_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS trial_start TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS trial_end TIMESTAMP WITH TIME ZONE;

-- Subscription schedule changes table
CREATE TABLE subscription_schedule_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    
    -- Change details
    change_type subscription_change_type NOT NULL,
    scheduled_for TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- For upgrades/downgrades - store line items as JSONB
    from_line_items JSONB,
    to_line_items JSONB,
    
    -- Proration details
    proration_amount_cents BIGINT,
    proration_calculation JSONB,
    
    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'processing', 'completed', 'cancelled', 'failed')),
    processed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    
    -- Metadata
    reason TEXT,
    initiated_by VARCHAR(50) CHECK (initiated_by IN ('customer', 'admin', 'system', 'dunning')),
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Subscription proration records
CREATE TABLE subscription_prorations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    schedule_change_id UUID REFERENCES subscription_schedule_changes(id),
    
    -- Proration type
    proration_type VARCHAR(50) NOT NULL CHECK (proration_type IN ('upgrade_credit', 'downgrade_adjustment', 'cancellation_credit', 'pause_credit')),
    
    -- Time period for calculation
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    days_total INTEGER NOT NULL,
    days_used INTEGER NOT NULL,
    days_remaining INTEGER NOT NULL,
    
    -- Financial amounts
    original_amount_cents BIGINT NOT NULL,
    used_amount_cents BIGINT NOT NULL,
    credit_amount_cents BIGINT NOT NULL,
    
    -- Where the credit was applied
    applied_to_invoice_id UUID REFERENCES invoices(id),
    applied_to_payment_id UUID REFERENCES payments(id),
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Subscription state history for audit trail
CREATE TABLE subscription_state_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    
    -- State transition
    from_status subscription_status,
    to_status subscription_status NOT NULL,
    
    -- Financial changes
    from_amount_cents BIGINT,
    to_amount_cents BIGINT,
    
    -- Line items snapshot (JSONB for flexibility)
    line_items_snapshot JSONB,
    
    -- Context
    change_reason TEXT,
    schedule_change_id UUID REFERENCES subscription_schedule_changes(id),
    initiated_by VARCHAR(50),
    
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for subscription management tables
CREATE INDEX idx_schedule_changes_subscription ON subscription_schedule_changes(subscription_id, scheduled_for);
CREATE INDEX idx_schedule_changes_status ON subscription_schedule_changes(status, scheduled_for) WHERE status = 'scheduled';
CREATE INDEX idx_schedule_changes_type ON subscription_schedule_changes(change_type);

CREATE INDEX idx_prorations_subscription ON subscription_prorations(subscription_id);
CREATE INDEX idx_prorations_schedule_change ON subscription_prorations(schedule_change_id);

CREATE INDEX idx_state_history_subscription ON subscription_state_history(subscription_id, occurred_at DESC);
CREATE INDEX idx_state_history_schedule_change ON subscription_state_history(schedule_change_id);

-- Subscription management triggers
CREATE TRIGGER set_subscription_schedule_changes_updated_at
    BEFORE UPDATE ON subscription_schedule_changes
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Invoice activities table for audit trail
CREATE TABLE IF NOT EXISTS invoice_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    activity_type VARCHAR(50) NOT NULL, -- 'created', 'status_changed', 'sent', 'viewed', 'paid', 'reminder_sent', etc.
    from_status VARCHAR(50),
    to_status VARCHAR(50),
    performed_by UUID REFERENCES users(id),
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for invoice activities
CREATE INDEX idx_invoice_activities_invoice_id ON invoice_activities(invoice_id);
CREATE INDEX idx_invoice_activities_created_at ON invoice_activities(created_at);

-- Add reminder fields to invoices table
ALTER TABLE invoices
ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS reminder_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS notes TEXT,
ADD COLUMN IF NOT EXISTS terms TEXT,
ADD COLUMN IF NOT EXISTS footer TEXT;

