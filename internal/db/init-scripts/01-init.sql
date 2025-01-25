-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum for API key access levels
CREATE TYPE api_key_level AS ENUM ('read', 'write', 'admin');

-- Enum for user roles
CREATE TYPE user_role AS ENUM ('admin', 'account');

-- Users table (for admins and accounts)
CREATE TABLE IF NOT EXISTS users (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    auth0_id VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    role user_role NOT NULL,
    name VARCHAR(255),
    picture_url TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Accounts table
CREATE TABLE IF NOT EXISTS accounts (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    business_name VARCHAR(255),
    business_type VARCHAR(100),
    website_url TEXT,
    support_email VARCHAR(255),
    support_phone VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    livemode BOOLEAN DEFAULT false
);

-- Customers table (subscribers/payers)
CREATE TABLE IF NOT EXISTS customers (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id),
    email VARCHAR(255) NOT NULL,
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
    livemode BOOLEAN DEFAULT false,
    UNIQUE(account_id, email)  -- Ensure email is unique per account
);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    account_id UUID REFERENCES accounts(id),
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,  -- Store hashed key, not plain text
    level api_key_level NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB,
    livemode BOOLEAN DEFAULT false
);
-- Create indexes
CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_customers_account_id ON customers(account_id);
CREATE INDEX idx_api_keys_account_id ON api_keys(account_id);

-- Insert test data for development
INSERT INTO users (auth0_id, email, role, name)
VALUES 
    ('auth0|admin123', 'admin@cyphera.com', 'admin', 'Admin User'),
    ('auth0|account123', 'merchant@example.com', 'account', 'Test Merchant')
ON CONFLICT (email) DO NOTHING;

INSERT INTO accounts (user_id, name, description, business_name)
VALUES 
    (
        (SELECT id FROM users WHERE email = 'merchant@example.com'),
        'Test Account',
        'Test merchant account for development',
        'Test Business LLC'
    ),
    (
        (SELECT id FROM users WHERE email = 'admin@cyphera.com'),
        'Admin Account',
        'Admin account for development',
        'Cyphera Admin'
    )
ON CONFLICT DO NOTHING;

-- Insert test API keys for the merchant and admin
INSERT INTO api_keys (account_id, name, key_hash, level, expires_at, is_active, metadata)
VALUES 
    (
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        'Test Valid API Key',
        'test_valid_key_hash',  -- In production, this would be a proper hash
        'write',
        NULL,  -- No expiration
        true,
        '{"test": true}'::jsonb
    ),
    (
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        'Test Expired API Key',
        'test_expired_key_hash',  -- In production, this would be a proper hash
        'write',
        CURRENT_TIMESTAMP - INTERVAL '1 day',  -- Expired yesterday
        true,
        '{"test": true, "expired": true}'::jsonb
    ),
    (
        (SELECT id FROM accounts WHERE name = 'Admin Account'),
        'Admin API Key',
        'admin_valid_key_hash',  -- In production, this would be a proper hash
        'admin',  -- Admin level access
        NULL,  -- No expiration
        true,
        '{"test": true, "admin": true}'::jsonb
    )
ON CONFLICT DO NOTHING;

INSERT INTO customers (account_id, email, name, description, balance, currency, metadata)
VALUES 
    (
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        'customer1@example.com',
        'Test Customer One',
        'First test customer for development',
        1000,  -- $10.00 balance
        'USD',
        '{"test": true, "customer_number": 1}'::jsonb
    ),
    (
        (SELECT id FROM accounts WHERE name = 'Test Account'),
        'customer2@example.com',
        'Test Customer Two',
        'Second test customer for development',
        5000,  -- $50.00 balance
        'USD',
        '{"test": true, "customer_number": 2}'::jsonb
    )
ON CONFLICT DO NOTHING;