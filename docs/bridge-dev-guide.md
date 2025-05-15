# Bridge Developer Guide

## Introduction

Bridge is a platform that offers stablecoin orchestration and issuance as a service. It provides a Stripe-like experience enabling developers to easily convert between different dollar formats (fiat, USDC, USDP, etc.). The issuance service gives developers the ability to convert any of these dollars into a stablecoin they can program and benefit from.

## Table of Contents

1. [Overview](#overview)
2. [Core API Concepts](#core-api-concepts)
3. [Getting Started](#getting-started)
4. [Customers](#customers)
5. [External Accounts](#external-accounts)
6. [Transfers](#transfers)
7. [Liquidation Address](#liquidation-address)
8. [Virtual Accounts](#virtual-accounts)
9. [Static Memos](#static-memos)
10. [Stablecoin Issuance](#stablecoin-issuance)
11. [Webhooks](#webhooks)
12. [Cards](#cards)
13. [Wallets](#wallets)
14. [Supported Geographies and Payment Methods](#supported-geographies-and-payment-methods)
15. [Compliance Requirements](#compliance-requirements)
16. [Pricing and Fees](#pricing-and-fees)

## Overview

Bridge provides several key services:

- **Stablecoin Orchestration**: Convert between fiat and various stablecoins
- **Stablecoin Issuance**: Create your own fiat-backed stablecoin
- **Custodial Wallets**: Hold crypto securely
- **Card Issuance**: Offer Visa cards to customers that spend directly from stablecoin balances

The documentation is organized into:
- Quick Start guide
- Core API Concepts
- Products
- API Reference
- API Recipes (Coming soon)

## Core API Concepts

Before starting with the API, you should understand these core concepts:

- **Authentication**: How Bridge identifies who you are
- **Idempotence**: How Bridge prevents duplicate requests
- **Customers**: Bridge's understanding of your customers using your products
- **External Accounts**: Your accounts that Bridge can send or receive funds from
- **Developer Fees**: Fees that can be set to charge your customers
- **Receipts**: Details about how outbound funds were delivered with important disclosures
- **Transfers**: How to initiate money movement
- **Liquidation Address**: A permanent wallet address which sends deposited funds to a pre-configured destination
- **Virtual Accounts**: A permanent bank account number which sends deposited funds to a pre-configured crypto address
- **SEPA**: Single Euro Payments Area for processing bank transfers in euros
- **SWIFT**: Global network that facilitates secure financial transactions between banks
- **Cards**: Bridge card issuing for offering Visa cards to customers globally

## Getting Started

### Create a Bridge Account

1. Go to dashboard.bridge.xyz
2. Create a free developer account (uses passwordless sign-in via email)

### Create API Keys

1. Log in to the dashboard
2. Click on "API Keys" tab on the top menu bar
3. Generate a new API key
4. Save the key immediately in a secure location (it's only shown once)

### Onboard Your First Customer

#### 1. Request a Terms of Service link

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/customers/tos_links' \
--header 'Idempotency-Key: <A unique idempotency key>' \
--header 'Api-Key: <API Key>'
```

Response:
```json
{
  "url": "https://dashboard.bridge.xyz/accept-terms-of-service?session_token=4..."
}
```

Direct your customer to this URL to accept the terms of service. You can embed this in an iframe or open in a new browser window. You can pass a `redirect_uri` parameter to have the user redirected back to your application with a `signed_agreement_id`.

#### 2. Create a customer

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/customers' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <generate a uuid>' \
--data-raw '{
  "type": "individual",
  "first_name": "John",
  "last_name": "Doe",
  "email": "johndoe@johndoe.com",
  "phone": "+12223334444",
  "address": {
    "street_line_1": "1234 Lombard Street",
    "street_line_2": "Apt 2F",
    "city": "San Francisco",
    "state": "CA",
    "postal_code": "94109",
    "country": "USA"
  },
  "signed_agreement_id": "<signed_agreement_id from above>",
  "birth_date": "1989-09-09",
  "tax_identification_number": "111-11-1111"
}'
```

Bridge will handle KYC for the customer and return a status indicating whether they've been approved.

#### 3. Register the customer's bank account

Follow the steps for registering a customer's bank account with Plaid or manually register a bank account.

#### 4. Move your first dollar

Once you have a customer with an active KYC status and a registered bank account, you can use Bridge's Transfers or Liquidation Address APIs:

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/transfers' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <generate a idempotency-key>' \
--data-raw '{
  "amount": "100.00",
  "on_behalf_of": "customer_123",
  "source": {
    "payment_rail": "polygon",
    "currency": "usdc",
    "from_address": "0xdeadbeef"
  },
  "destination": {
    "payment_rail": "wire",
    "currency": "usd",
    "external_account_id": "external_account_123"
  }
}'
```

## Authentication

Bridge authenticates API requests using API keys generated from the dashboard. All authentication is performed via HTTP Basic Auth through the `Api-Key` header. No additional information or password is needed.

```shell
curl --location --request GET 'https://api.bridge.xyz/v1/customers' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <API Key>'
```

If a key is not included or an incorrect key is used, Bridge will return a 401 - Unauthorized HTTP status code. HTTPS is required for all API requests.

**Security Note:** API keys provide full access to the APIs, so keep them secure. They should never be exposed in public forums or broadcast internally within your organization.

## Idempotence

All critical Bridge POST APIs require idempotency to safely allow identical requests multiple times in case of network failures or timeouts. Include an `Idempotency-Key` header with a unique value identifying the request.

Read, update, and delete requests (GET, PUT, PATCH, DELETE) should not include an Idempotency-Key as they are naturally idempotent.

```shell
curl --location --request POST 'https://api.bridge.xyz/v1/customers' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <Api-Key>' \
--header 'Idempotency-Key: <generate a uuid>' \
--data-raw '{ ... }'
```

For 24 hours following the initial request, Bridge guarantees that you can safely make the same request multiple times without side effects as long as the same Idempotency-Key is used.

## Customers

Customers represent users of your business. By registering them with Bridge through the Customers API or KYC Links API, you enable seamless transfers of stablecoins or fiat currencies from or to your customer's wallets or bank accounts.

Bridge handles all KYC and KYB checks, allowing you to safely move funds knowing that Bridge has properly vetted your users in compliance with legal requirements.

### Creating Customers via API

The Customers API allows you to directly pass KYC information to Bridge. This gives you control over the UI and lets you handle all communication with customers.

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/customers' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <generate a uuid>' \
--data-raw '{
  "type": "individual",
  "first_name": "John",
  "last_name": "Doe",
  "email": "email@example.com",
  "address": {
    "street_line_1": "123 Main St",
    "city": "New York City",
    "subdivision": "New York",
    "postal_code": "10001",
    "country": "USA"
  },
  "birth_date": "2007-01-01",
  "signed_agreement_id": "d536a227-06d3-4de1-acd3-8b5131730480",
  "identifying_information": [
    {
      "type": "ssn",
      "issuing_country": "usa",
      "number": "xxx-xx-xxxx"
    },
    {
      "type": "drivers_license",
      "issuing_country": "usa",
      "number": "xxxxxxxxxxxxx",
      "image_front": "data:image/jpg;base64,...",
      "image_back": "data:image/jpg;base64,..."
    }
  ]
}'
```

Bridge will review the customer information and return statuses for both overall KYC (`kyc_status`) and each requested endorsement.

### Creating Customers via KYC Links

As an alternative to the Customers API, you can use KYC Links. This approach supports individual and business customers worldwide.

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/kyc_links' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <generate a uuid>' \
--data-raw '{
  "full_name": "John Doe",
  "email": "johndoe@johndoe.com",
  "type": "individual" // or "business"
}'
```

The response will contain:
- `tos_link`: Bridge's Terms of Service link that needs to be accepted
- `kyc_link`: A hosted KYC/B flow link where customers submit information to Bridge

### Endorsements

Endorsements represent approval of a customer to onboard and transact with Bridge. Different endorsements are available depending on the region and intended payment rails:

1. **Base Endorsement**: Approval to use all payment rails except SEPA/Euro
2. **SEPA Endorsement**: Approval to use SEPA/Euro services
3. **SPEI Endorsement**: Approval to use SPEI/MXN peso services

Customers need to meet specific KYC requirements and accept the appropriate Terms of Service for each endorsement type.

## External Accounts

External Accounts represent your user's financial accounts (bank accounts, debit cards, etc.) that can be used to withdraw funds. Bridge validates all external accounts and performs KYC/KYB so you don't have to.

### Adding External Accounts via Bridge API

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/customers/{customer_id}/external_accounts' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <Api-Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "type": "raw",
  "bank_name": "Chase",
  "account_number": "12345678901",
  "routing_number": "123456789",
  "account_name": "Checking",
  "account_owner_name": "John Doe",
  "active": true,
  "address": {
    "street_line_1": "123 Washington St",
    "street_line_2": "Apt 2F",
    "city": "New York",
    "state": "NY",
    "postal_code": "10001",
    "country": "USA"
  }
}'
```

### Adding External Accounts via Plaid

1. Request a Plaid Link Token from Bridge
2. Start the Plaid Link SDK with the Token
3. Send the Plaid Public Token back to Bridge
4. Bridge retrieves linked accounts and creates External Accounts
5. Fetch the Customer's External Accounts

## Transfers

Bridge's Transfer API allows you to seamlessly convert between fiat and crypto or between different cryptocurrencies. Transfers require a source and a destination, which can be fiat sources (bank accounts, debit cards, etc.) or crypto sources (wallets on chains).

### Transfer States

- `awaiting_funds`: Waiting to receive funds from the customer
- `in_review`: Temporary state when a transaction is under review
- `funds_received`: Acknowledged receipt of funds, processing the movement
- `payment_submitted`: Payment sent and awaiting verification
- `payment_processed`: Transfer completed
- `undeliverable`: Unable to send funds to the specified destination
- `returned`: Payment wasn't successful and funds are being returned
- `refunded`: Funds have been sent back to the original sender
- `canceled`: Transfer has been canceled
- `error`: Problem preventing processing, may require manual intervention

### Fiat to Stablecoin (On-Ramp)

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/transfers' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "amount": "10.0",
  "on_behalf_of": "cust_alice",
  "developer_fee": "0.5",
  "source": {
    "payment_rail": "wire",
    "currency": "usd"
  },
  "destination": {
    "payment_rail": "ethereum",
    "currency": "usdc",
    "to_address": "0xdeadbeef"
  }
}'
```

### Stablecoin to Fiat (Off-Ramp)

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/transfers' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "amount": "10.0",
  "on_behalf_of": "cust_alice",
  "developer_fee": "0.5",
  "source": {
    "payment_rail": "solana",
    "currency": "xUSD",
    "from_address": "0xfromdeadbeef"
  },
  "destination": {
    "payment_rail": "ach",
    "currency": "usd",
    "external_account_id": "840ac7f3-555d-49ff-8128-28709afff2a6"
  }
}'
```

### Stablecoin to Stablecoin

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/transfers' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "amount": "10.0",
  "developer_fee": "0.5",
  "on_behalf_of": "cust_alice",
  "source": {
    "payment_rail": "ethereum",
    "currency": "usdc",
    "from_address": "0xdeadbeef"
  },
  "destination": {
    "payment_rail": "polygon",
    "currency": "usdc",
    "to_address": "0xdeadbeef"
  }
}'
```

### Transfer Features

Bridge offers flexible configuration options:

1. **Flexible Amount**: Create transfers that match any funds sent regardless of amount
2. **Static Templates**: Create a template with standing deposit instructions that can be used multiple times
3. **Allow Any From Address**: For offramps where the sending address isn't known in advance

## Liquidation Address

A "Liquidation Address" is a permanent payment route that ties a blockchain address to either a bank account or another blockchain address. When customers send crypto to their liquidation address, the funds are converted and sent to the preconfigured destination.

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/customers/cust_alice/liquidation_addresses' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <Api-Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "chain": "ethereum",
  "currency": "usdc",
  "external_account_id": "ea_alice_bofa",
  "destination_wire_message": "alice_wire_123",
  "destination_payment_rail": "wire",
  "destination_currency": "usd"
}'
```

## Virtual Accounts

Virtual accounts provide a unique routing and account number that accepts wire transfers and ACH deposits. When funds are deposited, they are automatically converted to the specified cryptocurrency and sent to the configured destination address.

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/customers/cust_alice/virtual_accounts' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <Api-Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "source": {
    "currency": "usd"
  },
  "destination": {
    "payment_rail": "ethereum",
    "currency": "usdc",
    "address": "0xDEADBEEF"
  },
  "developer_fee_percent": "1.0"
}'
```

## Static Memos

Static Memos are long-lived deposit instructions that accept fiat funds through wire and ACH push and send funds to the configured destination wallet. The same Static Memo can receive both wires and ACH pushes interchangeably.

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/customers/cust_alice/static_memos' \
--header 'Content-Type: application/json' \
--header 'Api-Key: <Api-Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "source": {
    "currency": "usd",
    "payment_rail": "wire"
  },
  "destination": {
    "payment_rail": "ethereum",
    "currency": "usdc",
    "address": "0xDEADBEEF"
  },
  "developer_fee_percent": "1.0"
}'
```

## Webhooks

Bridge's Webhooks API lets developers manage webhook endpoints subscribed to events for real-time updates. Webhook endpoints require a URL with HTTPS scheme and valid X.509 certificate.

### Event Structure

Each webhook event contains:
- `api_version`: Current version, currently "v0"
- `event_id`: Globally unique identifier for the event
- `event_category`: Category of the event (e.g., "customer", "transfer")
- `event_type`: Type of event within the category (e.g., "created", "updated")
- `event_object_id`: ID of the object related to the event
- `event_object_status`: Current status of the object (if applicable)
- `event_object`: The full payload of the API object
- `event_object_changes`: Diffs from the previous webhook event
- `event_created_at`: Event creation time in ISO 8601 format

### Supported Event Categories

- `customer`: Customer-related events
- `kyc_link`: KYC Link-related events
- `liquidation_address.drain`: Liquidation Address drain events
- `static_memo.activity`: Static Memo activity events
- `transfer`: Transfer-related events
- `virtual_account.activity`: Virtual Account activity events
- `card_account`: Card account-related events
- `card_transaction`: Card transaction events
- `posted_card_account_transaction`: Finalized card transactions
- `card_dispute`: Card dispute events
- `card_withdrawal`: Card withdrawal status changes

## Stablecoin Issuance

Bridge allows you to access a Bridge stablecoin ("USDB") or create your own custom stablecoin ("xUSD"). These stablecoins are always backed 1:1 by the equivalent value of US dollars.

### USDB

USDB is a stablecoin issued and managed by Bridge that can be exchanged to and from most stablecoins or fiat using Bridge Orchestration APIs.

```shell
curl --location --request POST 'https://api.bridge.xyz/v0/transfers' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--data-raw '{
  "amount": "10.0",
  "on_behalf_of": "cust_alice",
  "developer_fee": "0.5",
  "source": {
    "payment_rail": "ach",
    "currency": "usd",
    "external_account_id": "840ac7f3-555d-49ff-8128-28709afff2a6"
  },
  "destination": {
    "payment_rail": "solana",
    "currency": "usdb",
    "to_address": "0xtodeadbeef"
  }
}'
```

### Custom Stablecoin (xUSD)

Custom stablecoins (xUSD) allow you to have your own branded stablecoin issued and managed by Bridge.

To set up a custom stablecoin, you need to specify:
- Chain: Which blockchain to deploy on
- Token Name: Name of the token
- Token ID: Ticker symbol of the token
- Token Logo: Image for display in apps and tracking websites
- Reserves Strategy: Percentage of assets in cash vs. investments
- Refundable Deposits: Dollar value of xUSD to hold in "inventory"

## Wallets

Bridge provides custodial wallets to hold cryptocurrencies. The key functionality includes:
- Creating individual wallets for users
- Creating company/treasury wallets
- Transferring in and out of Bridge wallets
- Transferring between different Bridge wallets
- Querying balances
- Tagging wallets and setting policies

```shell
curl --request POST \
--url https://api.bridge.xyz/v0/customers/customerId/wallets \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <Unique Idempotency Key>' \
--header 'accept: application/json' \
--header 'content-type: application/json' \
--data '
{
  "chain": "solana"
}
```

## Cards

Bridge card issuing allows you to offer a Visa card to customers in multiple markets globally with a single integration. End customers can spend stablecoin balances anywhere Visa is accepted with a virtual card, physical card, or mobile wallets.

```shell
curl -X POST 'https://api.bridge.xyz/v0/customers/<customerID>/card_accounts'\
--header 'Content-Type: application/json' \
--header 'Api-Key: <API Key>' \
--header 'Idempotency-Key: <generate a uuid>' \
-d client_reference_id="test-card-reference-id1" \
-d currency="usdc" \
-d chain="solana" \
```

## Supported Geographies and Payment Methods

### Geographies

Bridge can facilitate payments for customers in most countries not on the OFAC sanctions list. Within the United States, Bridge supports customers in most states, excluding New York, Florida, Alaska, and Louisiana.

### Payment Methods

- **US**: ACH, Same Day ACH, Wire
- **Euro**: SEPA
- **Mexico**: SPEI using CLABE

### Stablecoins and Blockchains

Bridge supports various fiat-backed stablecoins on different blockchains:

| | ETH | Polygon | Base | Arbitrum | Avalanche | Optimism | Solana |
|---|---|---|---|---|---|---|---|
| USDC | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| USDT | ✅ | - | - | - | - | - | - |
| DAI | ✅ | - | - | - | - | - | - |
| USDP | ✅ | - | - | - | - | - | - |
| PYUSD | ✅ | ✅ | - | - | - | - | - |
| EURC | ✅ | - | - | - | - | - | - |

## Compliance Requirements

### Individual Requirements

- First and last name
- Country
- Street address, city, postal code, province/state, country
- Date of birth
- Email
- National identity number (for non-USA residents)
- Social Security number (for USA residents)
- ID verification (optional for US residents with minimal activity)
- Proof of address (for SEPA access or EEA+ residency)

### Business Requirements

- Legal Entity Name
- Registered Address
- Principal Operating Address
- EIN/TIN (or equivalent)
- Business Entity Type
- Business Formation Documents
- Business Ownership Documents
- KYC on all Beneficial Owners and/or Control Owners
- Proof of Address Documents (if applicable)
- Business Description
- Business Website

## Pricing and Fees

### Transaction Costs

Bridge passes through payment-method transaction fees at cost:

- **USD payments**
  - ACH: $0.50
  - Same Day ACH: $1
  - Wire: $20
  - ACH and Wire Returns: Varies

- **EUR payments**
  - SEPA: $1

- **Crypto withdrawals**
  - Gas fees (vary by blockchain)

### Developer Fees

Bridge collects configurable developer fees through customer transactions. These are set aside in a special account and paid out on the 5th of each month to the developer's external account.

## Settlement

- **Settlement Hours**: Bridge moves funds on days when US banks are open
- **ACH**: Transactions received after 1pm ET are processed the next business day
- **Wires**: Transactions received after 5pm ET are processed the next business day
- **Stablecoin to fiat**: 
  - ACH: Sent in batches at 1:00pm ET, takes 1-2 business days to land
  - Wire: Generally sent within minutes, up to 90 minutes to complete
- **Fiat to stablecoin**: Generally initiated within minutes of fiat landing, up to 90 minutes

## Support

Bridge sets up shared Slack or Discord channels with development teams. Customers can also reach out directly to Bridge at support@bridge.xyz.
