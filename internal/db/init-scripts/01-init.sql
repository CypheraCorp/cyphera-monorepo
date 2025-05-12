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
CREATE TYPE wallet_type AS ENUM ('wallet', 'circle_wallet');
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

-- Customers table (depends on workspaces)
CREATE TABLE IF NOT EXISTS customers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    external_id VARCHAR(255),
    email VARCHAR(255),
    name VARCHAR(255),
    phone VARCHAR(255),
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(workspace_id, external_id)
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
    wallet_type TEXT NOT NULL,                -- 'wallet' or 'circle_wallet'
    wallet_address TEXT NOT NULL,
    network_type network_type NOT NULL,
    network_id UUID REFERENCES networks(id),
    nickname TEXT,
    ens TEXT,
    is_primary BOOLEAN DEFAULT false,
    verified BOOLEAN DEFAULT false,
    last_used_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT check_wallet_type CHECK (wallet_type IN ('wallet', 'circle_wallet'))
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

-- Products table (depends on workspaces, wallets)
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    url TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Prices table (depends on products)
CREATE TABLE prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    active BOOLEAN NOT NULL DEFAULT true,
    type price_type NOT NULL, -- 'recurring' or 'one_off'
    nickname TEXT,
    currency currency NOT NULL, -- 'USD', 'EUR'
    unit_amount_in_pennies INTEGER NOT NULL,
    interval_type interval_type, -- Nullable, for 'recurring' type
    term_length INTEGER, -- Nullable, for 'recurring' type, e.g., 12 for 12 months
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT prices_recurring_fields_check CHECK (
        (type = 'recurring' AND interval_type IS NOT NULL AND term_length IS NOT NULL AND term_length > 0) OR
        (type = 'one_off' AND interval_type IS NULL AND term_length IS NULL)
    )
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

-- Subscriptions table (depends on customers, products, products_tokens, delegation_data, customer_wallets)
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    product_id UUID NOT NULL REFERENCES products(id),
    price_id UUID NOT NULL REFERENCES prices(id),
    product_token_id UUID NOT NULL REFERENCES products_tokens(id),
    token_amount NUMERIC NOT NULL,
    delegation_id UUID NOT NULL REFERENCES delegation_data(id),
    customer_wallet_id UUID REFERENCES customer_wallets(id),
    status subscription_status NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    next_redemption_date TIMESTAMP WITH TIME ZONE,
    total_redemptions INT NOT NULL DEFAULT 0,
    total_amount_in_cents INT NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
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
-- (Order doesn't matter as much, but group by table for readability)

-- accounts
-- (Primary key index created automatically)

-- users
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_supabase_id ON users(supabase_id);
CREATE INDEX idx_users_account_id ON users(account_id);

-- workspaces
CREATE INDEX idx_workspaces_account_id ON workspaces(account_id);

-- circle_users
CREATE INDEX idx_circle_users_workspace_id ON circle_users(workspace_id);

-- customers
CREATE INDEX idx_customers_workspace_id ON customers(workspace_id);

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

-- prices
CREATE INDEX idx_prices_product_id ON prices(product_id);
CREATE INDEX idx_prices_active ON prices(active) WHERE deleted_at IS NULL;
CREATE INDEX idx_prices_type ON prices(type);
CREATE INDEX idx_prices_currency ON prices(currency);

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

INSERT INTO users (supabase_id, email, first_name, last_name, display_name, account_id, role, is_account_owner)
VALUES 
    ('supabase|admin', 'admin@cyphera.com', 'Admin', 'User', 'Admin User',
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
        'admin_valid_key_hash',
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
    ('Base Mainnet', 'Mainnet', 'evm', 'BASE', 8453, false, false, 'https://basescan.org'),
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

