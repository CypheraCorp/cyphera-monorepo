# Subscription Events API Documentation

## Overview

The Subscription Events API provides endpoints for retrieving and managing subscription-related events in the Cyphera platform.

## Endpoints

### List Subscription Event Transactions

Retrieves a paginated list of subscription events for a workspace with full details including network and customer information.

**Endpoint:** `GET /api/v1/subscription-events/transactions`

**Authentication:** Required (API Key)

**Headers:**
- `X-Api-Key`: Your API key (required)
- `X-Workspace-ID`: Workspace UUID (required for non-admin keys)

**Query Parameters:**
- `page` (integer, optional): Page number (default: 1)
- `limit` (integer, optional): Number of events to return (default: 20, max: 100)

**Response Structure:**
```json
{
  "data": [
    {
      "id": "string (uuid)",
      "subscription_id": "string (uuid)",
      "event_type": "string (e.g., 'redeemed', 'failed', 'cancelled')",
      "transaction_hash": "string",
      "event_amount_in_cents": "number",
      "event_occurred_at": "string (ISO 8601 timestamp)",
      "event_metadata": "string (base64 encoded JSON)",
      "event_created_at": "string (ISO 8601 timestamp)",
      "customer_id": "string (uuid)",
      "subscription_status": "string",
      "product_id": "string (uuid)",
      "product_name": "string",
      "customer": {
        "id": "string (uuid)",
        "object": "string",
        "email": "string",
        "name": "string",
        "finished_onboarding": "boolean",
        "created_at": "number (unix timestamp)",
        "updated_at": "number (unix timestamp)"
      },
      "product": {
        "id": "string (uuid)",
        "object": "string",
        "workspace_id": "string (uuid)",
        "wallet_id": "string (uuid)",
        "name": "string",
        "active": "boolean",
        "created_at": "number (unix timestamp)",
        "updated_at": "number (unix timestamp)"
      },
      "price_info": {
        "id": "string (uuid)",
        "type": "string",
        "currency": "string",
        "unit_amount_in_pennies": "number",
        "interval_type": "string",
        "term_length": "number",
        "created_at": "number (unix timestamp)",
        "updated_at": "number (unix timestamp)"
      },
      "product_token": {
        "id": "string (uuid)",
        "object": "string",
        "product_id": "string (uuid)",
        "product_token_id": "string (uuid)",
        "network_id": "string (uuid)",
        "token_id": "string (uuid)",
        "token_symbol": "string",
        "active": "boolean",
        "created_at": "number (unix timestamp)",
        "updated_at": "number (unix timestamp)"
      },
      "network": {
        "id": "string (uuid)",
        "object": "string",
        "name": "string (e.g., 'Base Sepolia')",
        "type": "string",
        "chain_id": "number (e.g., 84532)",
        "network_type": "string",
        "circle_network_type": "string",
        "is_testnet": "boolean",
        "active": "boolean",
        "created_at": "number (unix timestamp)",
        "updated_at": "number (unix timestamp)"
      }
    }
  ],
  "object": "list",
  "has_more": "boolean",
  "pagination": {
    "current_page": "number",
    "per_page": "number",
    "total_items": "number",
    "total_pages": "number"
  }
}
```

**Status Codes:**
- `200 OK`: Successfully retrieved subscription events
- `400 Bad Request`: Invalid workspace ID format or pagination parameters
- `401 Unauthorized`: Missing or invalid API key
- `500 Internal Server Error`: Failed to retrieve subscription events

**Example Request:**
```bash
curl -H "X-Api-Key: your-api-key" \
     -H "X-Workspace-ID: 780dba16-e956-416a-aaa3-9eb9b2565c5e" \
     "https://api.cyphera.com/api/v1/subscription-events/transactions?page=1&limit=10"
```

### Get Subscription Event by ID

Retrieves details of a specific subscription event.

**Endpoint:** `GET /api/v1/subscription-events/:event_id`

**Authentication:** Required (API Key)

**Headers:**
- `X-Api-Key`: Your API key (required)
- `X-Workspace-ID`: Workspace UUID (required for non-admin keys)

**Path Parameters:**
- `event_id`: The UUID of the subscription event

**Response:** Returns a single `SubscriptionEventResponse` object

### List Events for a Subscription

Get a list of all events for a specific subscription.

**Endpoint:** `GET /api/v1/subscriptions/:subscription_id/events`

**Authentication:** Required (API Key)

**Headers:**
- `X-Api-Key`: Your API key (required)
- `X-Workspace-ID`: Workspace UUID (required for non-admin keys)

**Path Parameters:**
- `subscription_id`: The UUID of the subscription

**Response:** Returns an array of `SubscriptionEventResponse` objects

## Data Types

### SubscriptionEventFullResponse

The `SubscriptionEventFullResponse` type includes complete information about a subscription event, including:

- Event details (ID, type, transaction hash, amounts, timestamps)
- Customer information
- Product information
- Price details
- Product token details
- **Network information (including chain_id)**

This response type is returned by the `/subscription-events/transactions` endpoint to provide all necessary information for displaying transaction history in frontend applications.

## Important Notes

1. **Network Information**: The `network` object in the response includes the `chain_id` field, which is essential for blockchain operations and transaction verification.

2. **Admin API Keys**: Admin API keys can access subscription events across all workspaces without specifying a workspace ID.

3. **Pagination**: Large result sets are automatically paginated. Use the `page` and `limit` query parameters to navigate through results.

4. **Event Metadata**: The `event_metadata` field contains base64-encoded JSON with additional event-specific information.

## Changes and Updates

### Recent Updates (July 2025)

- **Fixed**: The `/subscription-events/transactions` endpoint now returns `SubscriptionEventFullResponse` objects with complete network information including `chain_id`.
- **Added**: Full customer and product details are now included in the response.
- **Improved**: Response structure now matches frontend TypeScript types for better type safety.