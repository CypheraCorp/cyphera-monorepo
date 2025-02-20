-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum for API key access levels
CREATE TYPE api_key_level AS ENUM ('read', 'write', 'admin');

-- Enum for account types
CREATE TYPE account_type AS ENUM ('admin', 'merchant');

-- Enum for user roles within an account
CREATE TYPE user_role AS ENUM ('admin', 'support', 'developer');

-- Enum for user status
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended', 'pending');

-- Create product type enum
CREATE TYPE product_type AS ENUM ('recurring', 'one_off');

-- Create interval type enum
CREATE TYPE interval_type AS ENUM ('5mins', 'daily', 'week', 'month', 'year');

-- Create network type enum
CREATE TYPE network_type AS ENUM ('evm', 'solana', 'cosmos', 'bitcoin', 'polkadot');

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

-- Wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id),
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
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(account_id, wallet_address, network_type)
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    supabase_id VARCHAR(255) NOT NULL UNIQUE,
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
    status user_status DEFAULT 'active',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT one_owner_per_account EXCLUDE (account_id WITH =) WHERE (is_account_owner = true AND deleted_at IS NULL)
);

-- Workspaces table
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

-- Customers table
CREATE TABLE IF NOT EXISTS customers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    external_id VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    phone VARCHAR(255),
    description TEXT,
    balance_in_pennies INTEGER DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    default_source_id UUID,
    invoice_prefix VARCHAR(255),
    next_invoice_sequence INTEGER DEFAULT 1,
    tax_exempt BOOLEAN DEFAULT false,
    tax_ids JSONB,
    metadata JSONB,
    livemode BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(workspace_id, external_id)
);

-- API Keys table
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

-- Create products table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    name TEXT NOT NULL,
    description TEXT,
    product_type product_type NOT NULL,
    interval_type interval_type NOT NULL,
    term_length INTEGER, -- number of intervals
    price_in_pennies INTEGER NOT NULL,
    image_url TEXT,
    url TEXT,
    merchant_paid_gas BOOLEAN NOT NULL DEFAULT false,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT products_term_length_check CHECK (
        (product_type = 'recurring' AND term_length IS NOT NULL AND term_length > 0) OR
        (product_type = 'one_off' AND term_length IS NULL)
    ),
    CONSTRAINT products_interval_type_check CHECK (
        (product_type = 'recurring' AND interval_type IS NOT NULL) OR
        (product_type = 'one_off' AND interval_type IS NULL)
    )
);

-- Create networks table
CREATE TABLE networks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    chain_id INTEGER NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create tokens table
CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_id UUID NOT NULL REFERENCES networks(id),
    gas_token BOOLEAN NOT NULL DEFAULT false,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    contract_address TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(network_id, contract_address)
);

-- Create products_tokens table
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

-- Create Actalink Product table
CREATE TABLE actalink_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    product_token_id UUID NOT NULL REFERENCES products_tokens(id),
    actalink_payment_link_id TEXT NOT NULL,
    actalink_subscription_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(product_id, product_token_id)
);

-- Create indexes for actalink_products table
CREATE INDEX idx_actalink_products_product_id ON actalink_products(product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_actalink_products_product_token_id ON actalink_products(product_token_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_actalink_products_payment_link_id ON actalink_products(actalink_payment_link_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_actalink_products_subscription_id ON actalink_products(actalink_subscription_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_actalink_products_created_at ON actalink_products(created_at);

-- Add updated_at trigger for actalink_products
CREATE TRIGGER set_actalink_products_updated_at
    BEFORE UPDATE ON actalink_products
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

-- Create indexes
CREATE INDEX idx_workspaces_account_id ON workspaces(account_id);

CREATE INDEX idx_customers_workspace_id ON customers(workspace_id);

CREATE INDEX idx_api_keys_workspace_id ON api_keys(workspace_id);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_supabase_id ON users(supabase_id);
CREATE INDEX idx_users_account_id ON users(account_id);

CREATE INDEX idx_wallets_account_id ON wallets(account_id);
CREATE INDEX idx_wallets_address ON wallets(wallet_address);
CREATE INDEX idx_wallets_network_type ON wallets(network_type);
CREATE INDEX idx_wallets_is_primary ON wallets(is_primary) WHERE deleted_at IS NULL;

CREATE INDEX idx_products_workspace_id ON products(workspace_id);
CREATE INDEX idx_products_wallet_id ON products(wallet_id);
CREATE INDEX idx_products_active ON products(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_created_at ON products(created_at);

CREATE INDEX idx_networks_chain_id ON networks(chain_id);
CREATE INDEX idx_networks_active ON networks(active) WHERE deleted_at IS NULL;

CREATE INDEX idx_tokens_network_id ON tokens(network_id);
CREATE INDEX idx_tokens_active ON tokens(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_tokens_contract_address ON tokens(contract_address);

CREATE INDEX idx_products_tokens_product_id ON products_tokens(product_id);
CREATE INDEX idx_products_tokens_network_id ON products_tokens(network_id);
CREATE INDEX idx_products_tokens_token_id ON products_tokens(token_id);
CREATE INDEX idx_products_tokens_active ON products_tokens(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_tokens_composite ON products_tokens(product_id, network_id, active) WHERE deleted_at IS NULL;

-- Insert test data for development
INSERT INTO accounts (name, account_type, business_name, business_type)
VALUES 
    (
        'Test Account',
        'merchant',
        'Test Business LLC',
        'LLC'
    ),
    (
        'Admin Account',
        'admin',
        'Cyphera Admin',
        'Corporation'
    )
ON CONFLICT DO NOTHING;

-- Insert test wallets
INSERT INTO wallets (
    account_id,
    wallet_address,
    network_type,
    nickname,
    ens,
    is_primary,
    verified,
    metadata
)
VALUES 
    (
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        '0x742d35Cc6634C0532925a3b844Bc454e4438f44e',
        'evm',
        'Main Payment Wallet',
        'test-merchant.eth',
        true,
        true,
        '{"chain": "ethereum", "tags": ["payments"]}'::jsonb
    ),
    (
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        '5FHneW46xGXgs5mUiveU4sbTyGBzmstUspZC92UhjJM694ty',
        'polkadot',
        'DOT Wallet',
        null,
        false,
        true,
        '{"chain": "polkadot", "tags": ["staking"]}'::jsonb
    ),
    (
        (SELECT id FROM accounts WHERE name = 'Admin Account'),
        '0x8894e0a0c962cb723c1976a4421c95949be2d4e3',
        'evm',
        'Admin ETH Wallet',
        'cyphera-admin.eth',
        true,
        true,
        '{"chain": "ethereum", "tags": ["admin", "treasury"]}'::jsonb
    ),
    (
        (SELECT id FROM accounts WHERE name = 'Admin Account'),
        '6fYU3gaCuDGK7HLqsMW2nwpBJqecxLmLDBo5Hc3YzKf5',
        'solana',
        'Admin SOL Wallet',
        null,
        false,
        true,
        '{"chain": "solana", "tags": ["admin"]}'::jsonb
    )
ON CONFLICT DO NOTHING;

INSERT INTO users (supabase_id, email, first_name, last_name, display_name, account_id, role, is_account_owner)
VALUES 
    ('supabase|admin', 'admin@cyphera.com', 'Admin', 'User', 'Admin User',
     (SELECT id FROM accounts WHERE name = 'Admin Account'), 'admin', true),
    ('supabase|merchant', 'merchant@example.com', 'Test', 'Merchant', 'Test Merchant',
     (SELECT id FROM accounts WHERE name = 'Test Account'), 'admin', true)
ON CONFLICT DO NOTHING;

-- Insert test workspaces
INSERT INTO workspaces (account_id, name, description, business_name)
VALUES 
    (
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        'Test Workspace',
        'Test merchant workspace for development',
        'Test Business LLC'
    ),
    (
        (SELECT id FROM accounts WHERE name = 'Admin Account'),
        'Admin Workspace',
        'Admin workspace for development',
        'Cyphera Admin'
    )
ON CONFLICT DO NOTHING;

-- Insert test API keys
INSERT INTO api_keys (workspace_id, name, key_hash, access_level)
VALUES 
    (
        (SELECT id FROM workspaces WHERE name = 'Test Workspace'),
        'Test Valid API Key',
        'test_valid_key_hash',
        'write'
    ),
    (
        (SELECT id FROM workspaces WHERE name = 'Admin Workspace'),
        'Admin API Key',
        'admin_valid_key_hash',
        'admin'
    )
ON CONFLICT DO NOTHING;

-- Insert test customers
INSERT INTO customers (workspace_id, external_id, email, name, description, balance_in_pennies)
VALUES 
    (
        (SELECT id FROM workspaces WHERE name = 'Test Workspace'),
        'CUST_001',
        'customer1@example.com',
        'Test Customer One',
        'First test customer',
        1000
    ),
    (
        (SELECT id FROM workspaces WHERE name = 'Test Workspace'),
        'CUST_002',
        'customer2@example.com',
        'Test Customer Two',
        'Second test customer',
        5000
    )
ON CONFLICT DO NOTHING;

-- Insert some test networks
INSERT INTO networks (name, type, chain_id, active)
VALUES 
    ('Polygon', 'mainnet', 137, true)
ON CONFLICT DO NOTHING;

-- Insert some test tokens
INSERT INTO tokens (network_id, name, symbol, contract_address, gas_token)
VALUES 
    ((SELECT id FROM networks WHERE chain_id = 137 AND deleted_at IS NULL), 'USDC', 'USDC', '0x3c499c542cef5e3811e1192ce70d8cc03d5c3359', false)
ON CONFLICT DO NOTHING;

-- Insert test products
INSERT INTO products (
    workspace_id,
    wallet_id,
    name,
    description,
    product_type,
    interval_type,
    term_length,
    price_in_pennies,
    image_url,
    url,
    merchant_paid_gas,
    active,
    metadata
) VALUES 
    (
        (SELECT id FROM workspaces WHERE name = 'Test Workspace'),
        (SELECT id FROM wallets WHERE nickname = 'Main Payment Wallet' AND deleted_at IS NULL LIMIT 1),
        'Basic Subscription',
        'Monthly subscription plan with basic features, ends in 12 months',
        'recurring',
        'month',
        12,
        1000,
        'https://example.com/basic.png',
        'https://example.com/basic',
        true,
        true,
        '{"features": ["basic_support", "standard_limits"]}'::jsonb
    ),
    (
        (SELECT id FROM workspaces WHERE name = 'Test Workspace'),
        (SELECT id FROM wallets WHERE nickname = 'Main Payment Wallet' AND deleted_at IS NULL LIMIT 1),
        'Annual Pro Plan',
        'Annual subscription with all features, ends in 3 years',
        'recurring',
        'year',
        2,
        10000,
        'https://example.com/pro.png',
        'https://example.com/pro',
        true,
        true,
        '{"features": ["priority_support", "advanced_features"]}'::jsonb
    )
ON CONFLICT DO NOTHING;

-- Insert test product tokens (enabling specific tokens for products)
INSERT INTO products_tokens (
    product_id,
    network_id,
    token_id,
    active
) 
-- Enable USDC on Polygon for Basic Subscription
SELECT 
    p.id as product_id,
    t.network_id,
    t.id as token_id,
    true as active
FROM products p
CROSS JOIN tokens t
WHERE p.name = 'Basic Subscription'
AND t.network_id = (SELECT id FROM networks WHERE chain_id = 137)
AND t.symbol IN ('USDC')

UNION ALL

-- Enable only USDC on Polygon for Annual Pro Plan
SELECT 
    p.id as product_id,
    t.network_id,
    t.id as token_id,
    true as active
FROM products p
CROSS JOIN tokens t
WHERE p.name = 'Annual Pro Plan'
AND t.network_id = (SELECT id FROM networks WHERE chain_id = 137)
AND t.symbol = 'USDC'

UNION ALL

-- Enable only USDC on Polygon for Annual Pro Plan
SELECT 
    p.id as product_id,
    t.network_id,
    t.id as token_id,
    true as active
FROM products p
CROSS JOIN tokens t
WHERE p.name = 'Annual Pro Plan'
AND t.network_id = (SELECT id FROM networks WHERE chain_id = 137)
AND t.symbol = 'USDC'
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

-- Add triggers for updated_at
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

