-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum definitions (order doesn't matter here)
CREATE TYPE api_key_level AS ENUM ('read', 'write', 'admin');
CREATE TYPE account_type AS ENUM ('admin', 'merchant');
CREATE TYPE user_role AS ENUM ('admin', 'support', 'developer');
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'pending');
CREATE TYPE price_type AS ENUM ('recurring', 'one_off');
CREATE TYPE interval_type AS ENUM ('1min', '5mins', 'daily', 'week', 'month', 'year');
CREATE TYPE network_type AS ENUM ('evm', 'solana', 'cosmos', 'bitcoin', 'polkadot');
CREATE TYPE currency AS ENUM ('USD', 'EUR');
CREATE TYPE wallet_type AS ENUM ('wallet', 'circle_wallet', 'web3auth');
CREATE TYPE circle_network_type AS ENUM ('ARB', 'ARB-SEPOLIA', 'ETH', 'ETH-SEPOLIA', 'MATIC', 'MATIC-AMOY', 'OP', 'OP-SEPOLIA', 'BASE', 'BASE-SEPOLIA', 'UNI', 'UNI-SEPOLIA');
CREATE TYPE subscription_status AS ENUM ('active', 'canceled', 'expired', 'overdue', 'suspended', 'failed', 'completed');
CREATE TYPE subscription_event_type AS ENUM (
    'created', 
    'redeemed', 
    'renewed', 
    'canceled', 
    'expired',
    'completed',
    'failed',
    'failed_validation',
    'failed_customer_creation',
    'failed_wallet_creation',
    'failed_delegation_storage',
    'failed_subscription_db',
    'failed_redemption',
    'failed_transaction',
    'failed_duplicate'
);

-- Create Tables in dependency order

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
    block_explorer_url TEXT,
    chain_id INTEGER NOT NULL UNIQUE,
    is_testnet BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
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

-- Products table (depends on workspaces, wallets) - WITH PAYMENT SYNC COLUMNS
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    external_id VARCHAR(255), -- Provider's ID (Stripe ID, etc.)
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    url TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
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

-- Prices table (depends on products) - WITH PAYMENT SYNC COLUMNS
CREATE TABLE prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    external_id VARCHAR(255), -- Provider's ID (Stripe ID, etc.)
    active BOOLEAN NOT NULL DEFAULT true,
    type price_type NOT NULL, -- 'recurring' or 'one_off'
    nickname TEXT,
    currency currency NOT NULL, -- 'USD', 'EUR'
    unit_amount_in_pennies INTEGER NOT NULL,
    interval_type interval_type NOT NULL,
    term_length INTEGER NOT NULL, -- Nullable, for 'recurring' type, e.g., 12 for 12 months
    metadata JSONB DEFAULT '{}'::jsonb,
    -- Payment sync tracking columns
    payment_sync_status VARCHAR(20) DEFAULT 'pending',
    payment_synced_at TIMESTAMP WITH TIME ZONE, 
    payment_sync_version INTEGER DEFAULT 1,
    payment_provider VARCHAR(50), -- 'stripe', 'chargebee', etc.
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT prices_recurring_fields_check CHECK (
        (type = 'recurring' AND interval_type IS NOT NULL AND term_length IS NOT NULL AND term_length > 0) OR
        (type = 'one_off' AND interval_type IS NULL AND term_length IS NULL)
    ),
    -- Add unique constraint for external_id per provider
    UNIQUE(external_id, payment_provider)
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
    customer_id UUID NOT NULL REFERENCES customers(id),
    product_id UUID NOT NULL REFERENCES products(id),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    price_id UUID NOT NULL REFERENCES prices(id),
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

-- prices
CREATE INDEX idx_prices_product_id ON prices(product_id);
CREATE INDEX idx_prices_active ON prices(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_prices_type ON prices(type);
CREATE INDEX idx_prices_currency ON prices(currency);
CREATE INDEX idx_prices_payment_provider ON prices(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_prices_payment_sync_status ON prices(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_prices_external_id ON prices(external_id) WHERE deleted_at IS NULL;

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

-- delegation_data
CREATE INDEX idx_delegation_data_delegator ON delegation_data(delegator);
CREATE INDEX idx_delegation_data_delegate ON delegation_data(delegate);

-- subscriptions
CREATE INDEX idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX idx_subscriptions_product_id ON subscriptions(product_id);
CREATE INDEX idx_subscriptions_product_token_id ON subscriptions(product_token_id);
CREATE INDEX idx_subscriptions_delegation_id ON subscriptions(delegation_id);
CREATE INDEX idx_subscriptions_customer_wallet_id ON subscriptions(customer_wallet_id);
CREATE INDEX idx_subscriptions_price_id ON subscriptions(price_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_next_redemption_date ON subscriptions(next_redemption_date) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX idx_subscriptions_payment_provider ON subscriptions(payment_provider) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_payment_sync_status ON subscriptions(payment_sync_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_subscriptions_external_id ON subscriptions(external_id) WHERE deleted_at IS NULL;

-- subscription_events
CREATE INDEX idx_subscription_events_subscription_id ON subscription_events(subscription_id);
CREATE INDEX idx_subscription_events_event_type ON subscription_events(event_type);
CREATE INDEX idx_subscription_events_transaction_hash ON subscription_events(transaction_hash);
CREATE INDEX idx_subscription_events_occurred_at ON subscription_events(occurred_at);

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

INSERT INTO api_keys (workspace_id, name, key_hash, access_level)
VALUES 
    (
        (SELECT id FROM workspaces WHERE name = 'Admin Workspace'),
        'Admin API Key',
        'admin_valid_key',
        'admin'
    )
ON CONFLICT DO NOTHING;

INSERT INTO networks (name, type, network_type, circle_network_type, chain_id, is_testnet, active, block_explorer_url)
VALUES 
    ('Ethereum Sepolia', 'Sepolia', 'evm', 'ETH-SEPOLIA', 11155111, true, true, 'https://sepolia.etherscan.io'),
    ('Ethereum Mainnet', 'Mainnet', 'evm', 'ETH', 1, false, false, 'https://etherscan.io'),
    ('Polygon Amoy', 'Amoy', 'evm', 'MATIC-AMOY', 80002, true, false, 'https://www.oklink.com/amoy'), 
    ('Polygon Mainnet', 'Mainnet', 'evm', 'MATIC', 137, false, false, 'https://polygonscan.com'),
    ('Arbitrum Sepolia', 'Sepolia', 'evm', 'ARB-SEPOLIA', 421614, true, false, 'https://sepolia.arbiscan.io'),
    ('Arbitrum One', 'Mainnet', 'evm', 'ARB', 42161, false, false, 'https://arbiscan.io'),
    ('Base Sepolia', 'Sepolia', 'evm', 'BASE-SEPOLIA', 84532, true, true, 'https://sepolia.basescan.org'),
    ('Base Mainnet', 'Mainnet', 'evm', 'BASE', 8453, false, true, 'https://basescan.org'),
    ('Optimism Sepolia', 'Sepolia', 'evm', 'OP-SEPOLIA', 11155420, true, false, 'https://sepolia.optimism.io'),
    ('Optimism Mainnet', 'Mainnet', 'evm', 'OP', 10, false, false, 'https://optimistic.etherscan.io'),
    ('Unichain Sepolia', 'Sepolia', 'evm', 'UNI-SEPOLIA', 1301, true, false, 'https://sepolia.unichain.io'),
    ('Unichain Mainnet', 'Mainnet', 'evm', 'UNI', 130, false, false, 'https://unichain.io')
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

CREATE TRIGGER set_prices_updated_at
    BEFORE UPDATE ON prices
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Add triggers for payment sync tables
CREATE TRIGGER set_payment_sync_sessions_updated_at
    BEFORE UPDATE ON payment_sync_sessions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

