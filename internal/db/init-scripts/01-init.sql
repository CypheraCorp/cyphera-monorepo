-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum for API key access levels
CREATE TYPE api_key_level AS ENUM ('read', 'write', 'admin');

-- Enum for account types
CREATE TYPE account_type AS ENUM ('admin', 'merchant');

-- Enum for user roles within an account
CREATE TYPE user_role AS ENUM ('owner', 'admin', 'support', 'developer');

-- Accounts table (top level organization)
CREATE TABLE IF NOT EXISTS accounts (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    account_type account_type NOT NULL,
    business_name VARCHAR(255),
    business_type VARCHAR(255),
    website_url TEXT,
    support_email VARCHAR(255),
    support_phone VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    auth0_id VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    display_name VARCHAR(255),
    picture_url TEXT,
    phone VARCHAR(255),
    timezone VARCHAR(50),
    locale VARCHAR(10) DEFAULT 'en',
    last_login_at TIMESTAMP WITH TIME ZONE,
    email_verified BOOLEAN DEFAULT false,
    two_factor_enabled BOOLEAN DEFAULT false,
    status VARCHAR(50) DEFAULT 'active',
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- User Account relationships table
CREATE TABLE IF NOT EXISTS user_accounts (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    account_id UUID NOT NULL REFERENCES accounts(id),
    role user_role NOT NULL,
    is_owner BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, account_id),
    CONSTRAINT one_owner_per_account EXCLUDE (account_id WITH =) WHERE (is_owner = true AND deleted_at IS NULL)
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
    balance INTEGER DEFAULT 0,
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

-- Create indexes
CREATE INDEX idx_workspaces_account_id ON workspaces(account_id);
CREATE INDEX idx_customers_workspace_id ON customers(workspace_id);
CREATE INDEX idx_api_keys_workspace_id ON api_keys(workspace_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_auth0_id ON users(auth0_id);
CREATE INDEX idx_user_accounts_user_id ON user_accounts(user_id);
CREATE INDEX idx_user_accounts_account_id ON user_accounts(account_id);
CREATE INDEX idx_user_accounts_role ON user_accounts(role);

-- Insert test data for development
INSERT INTO users (auth0_id, email, first_name, last_name, display_name)
VALUES 
    ('auth0|admin', 'admin@cyphera.com', 'Admin', 'User', 'Admin User'),
    ('auth0|merchant', 'merchant@example.com', 'Test', 'Merchant', 'Test Merchant')
ON CONFLICT DO NOTHING;

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

INSERT INTO user_accounts (user_id, account_id, role, is_owner)
VALUES 
    (
        (SELECT id FROM users WHERE email = 'merchant@example.com'),
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        'owner',
        true
    ),
    (
        (SELECT id FROM users WHERE email = 'admin@cyphera.com'),
        (SELECT id FROM accounts WHERE name = 'Admin Account'),
        'owner',
        true
    )
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
INSERT INTO customers (workspace_id, external_id, email, name, description, balance)
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