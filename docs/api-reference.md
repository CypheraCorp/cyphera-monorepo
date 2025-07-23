# API Reference

> **Navigation:** [← Quick Start](quick-start.md) | [↑ README](../README.md) | [Architecture →](architecture.md)

Complete reference for the Cyphera Platform API.

## Table of Contents

- [Authentication](#authentication)
- [Request Format](#request-format)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [API Resources](#api-resources)
- [Rate Limits](#rate-limits)

## Authentication

The Cyphera API supports two authentication methods:

### JWT Token Authentication (Primary)
For user-facing operations, use Web3Auth JWT tokens:

```http
Authorization: Bearer <jwt_token>
X-Workspace-ID: <workspace_uuid>
```

### API Key Authentication
For service-to-service communication:

```http
Authorization: <api_key>
X-Workspace-ID: <workspace_uuid>
```

#### API Key Access Levels
- **Read:** Get operations only
- **Write:** Create, update, and delete operations
- **Admin:** Full access including user management

## Request Format

### Base URL
- **Development:** `http://localhost:8080/api/v1`
- **Production:** `https://api.cyphera.com/api/v1`

### Required Headers
```http
Content-Type: application/json
Authorization: Bearer <token> | <api_key>
X-Workspace-ID: <workspace_uuid>
```

### Request Body
All POST and PUT requests should include a JSON body:

```json
{
  "field1": "value1",
  "field2": "value2"
}
```

## Response Format

### Success Response
All successful responses follow this structure:

```json
{
  "id": "uuid",
  "object": "resource_type",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  // ... resource-specific fields
}
```

### List Response
Paginated list responses:

```json
{
  "object": "list",
  "data": [
    // ... array of resources
  ],
  "has_more": true,
  "pagination": {
    "limit": 50,
    "offset": 0,
    "total": 150
  }
}
```

## Error Handling

### Error Response Format
```json
{
  "error": {
    "type": "invalid_request_error",
    "code": "validation_error",
    "message": "The provided email is invalid",
    "param": "email"
  }
}
```

### HTTP Status Codes
- **200** - Success
- **201** - Created successfully
- **400** - Bad Request (validation error)
- **401** - Unauthorized (invalid credentials)
- **403** - Forbidden (insufficient permissions)
- **404** - Not Found
- **409** - Conflict (resource already exists)
- **429** - Rate Limited
- **500** - Internal Server Error

## API Resources

### Authentication & Accounts

#### Sign In
Create or sign in to an account:

```http
POST /admin/accounts/signin
```

**Request Body:**
```json
{
  "email": "merchant@example.com",
  "name": "John Doe",
  "role": "admin"
}
```

**Response:**
```json
{
  "id": "acc_1234567890",
  "object": "account",
  "email": "merchant@example.com",
  "name": "John Doe",
  "role": "admin",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### Get Account
Retrieve account information:

```http
GET /accounts/{account_id}
```

### Workspaces

#### List Workspaces
Get all workspaces for authenticated user:

```http
GET /admin/workspaces
```

#### Create Workspace
Create a new workspace:

```http
POST /admin/workspaces
```

**Request Body:**
```json
{
  "name": "My Business",
  "description": "E-commerce subscription platform"
}
```

### Products & Pricing

#### List Products
Get products for a workspace:

```http
GET /products
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "id": "prod_1234567890",
      "object": "product",
      "name": "Premium Subscription",
      "description": "Access to premium features",
      "prices": [
        {
          "id": "price_1234567890",
          "amount": 2999,
          "currency": "USD",
          "interval": "month"
        }
      ],
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Create Product
Create a new product with pricing:

```http
POST /products
```

**Request Body:**
```json
{
  "name": "Premium Subscription",
  "description": "Access to premium features",
  "prices": [
    {
      "amount": 2999,
      "currency": "USD",
      "interval": "month",
      "interval_count": 1
    }
  ],
  "tokens": [
    {
      "network_id": "eth_mainnet",
      "token_address": "0xa0b86a33e6776...",
      "amount": "29.99"
    }
  ]
}
```

### Customers

#### List Customers
Get customers for a workspace:

```http
GET /customers
```

#### Create Customer
Register a new customer:

```http
POST /customers
```

**Request Body:**
```json
{
  "email": "customer@example.com",
  "name": "Jane Smith",
  "web3_auth_id": "web3auth_user_id"
}
```

#### Get Customer
Retrieve customer details:

```http
GET /customers/{customer_id}
```

### Subscriptions

#### List Subscriptions
Get subscriptions for a workspace:

```http
GET /subscriptions
```

#### Subscribe to Product
Create a new subscription:

```http
POST /admin/prices/{price_id}/subscribe
```

**Request Body:**
```json
{
  "customer_id": "cust_1234567890",
  "wallet_id": "wallet_1234567890",
  "delegation_data": {
    "signature": "0x...",
    "permissions": {...}
  }
}
```

### Wallets

#### List Wallets
Get wallets for a workspace:

```http
GET /wallets
```

#### Create Wallet
Create a new wallet for a customer:

```http
POST /wallets
```

**Request Body:**
```json
{
  "customer_id": "cust_1234567890",
  "type": "circle_wallet",
  "network_id": "eth_mainnet",
  "address": "0x742d35Cc6588b..."  
}
```

#### Get Wallet
Retrieve wallet details:

```http
GET /wallets/{wallet_id}
```

### Networks

#### List Networks
Get available blockchain networks:

```http
GET /networks
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "id": "eth_mainnet",
      "object": "network",
      "name": "Ethereum Mainnet",
      "chain_id": 1,
      "rpc_url": "https://mainnet.infura.io/v3/...",
      "block_explorer": "https://etherscan.io",
      "native_currency": {
        "name": "Ether",
        "symbol": "ETH",
        "decimals": 18
      }
    }
  ]
}
```

### Circle Integration

#### Create Circle User
Initialize a Circle user for wallet management:

```http
POST /admin/circle/users/{workspace_id}
```

#### Create Circle Wallet
Create a programmable wallet via Circle:

```http
POST /admin/circle/wallets/{workspace_id}
```

**Request Body:**
```json
{
  "blockchain": "ETH",
  "account_type": "SCA"
}
```

#### Get Wallet Balance
Check wallet balance across tokens:

```http
GET /admin/circle/wallets/balances/{wallet_id}
```

### API Keys

#### List API Keys
Get API keys for a workspace:

```http
GET /api-keys
```

#### Create API Key
Generate a new API key:

```http
POST /api-keys
```

**Request Body:**
```json
{
  "name": "Integration Key",
  "access_level": "write",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

## Rate Limits

### Standard Limits
- **Authenticated requests:** 1000 requests per minute
- **Unauthenticated requests:** 100 requests per minute
- **Admin operations:** 500 requests per minute

### Rate Limit Headers
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

### Exceeding Limits
When rate limited, you'll receive:

```http
HTTP/1.1 429 Too Many Requests

{
  "error": {
    "type": "rate_limit_error",
    "message": "Too many requests. Try again in 60 seconds."
  }
}
```

## Webhooks

### Webhook Events
Subscribe to real-time events:

- `subscription.created`
- `subscription.updated`
- `subscription.cancelled`
- `payment.succeeded`
- `payment.failed`
- `customer.created`
- `wallet.created`

### Webhook Payload
```json
{
  "id": "evt_1234567890",
  "object": "event",
  "type": "subscription.created",
  "data": {
    "object": {
      // ... subscription object
    }
  },
  "created": 1640995200
}
```

## SDKs & Libraries

### Official SDKs
- **JavaScript/TypeScript:** `@cyphera/sdk-js`
- **Go:** `github.com/cyphera/go-sdk`
- **Python:** `cyphera-python` (coming soon)

### Community Libraries
- **React Hooks:** `@cyphera/react-hooks`
- **Vue.js:** `@cyphera/vue-components`

## Testing

### Test API Keys
Use test API keys in development:
```
sk_test_1234567890abcdef
```

### Test Mode
All operations in test mode use:
- Test blockchain networks (Sepolia, Mumbai)
- Mock payment processors
- Sandbox Circle API

---

## Need Help?

- **[Quick Start Guide](quick-start.md)** - Get started quickly
- **[Architecture Guide](architecture.md)** - Understand the system
- **[Troubleshooting](troubleshooting.md)** - Common issues
- **Support:** [GitHub Issues](https://github.com/your-org/cyphera-api/issues)

---

*Last updated: $(date '+%Y-%m-%d')*
*API Version: v1 | SDK Version: 2.0.0*