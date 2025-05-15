# Fern Developer Documentation

## Overview

Fern is a seamless currency conversion and payments API that enables developers to integrate onramps, offramps, FX, crypto exchange, and other features into their applications. Fern works with licensed liquidity providers and financial institutions to offer global currency support in one standardized API.

## Core Features

- **Multiplex API** — Any-to-any currency conversion across all supported crypto and fiat currencies
- **Wallets API** — Provision non-custodial wallets for users, with Fireblocks and the user as wallet signers
- **Multi-currency virtual accounts API** — Named virtual accounts for business and individual customers, enabling fiat acceptance in 15+ currencies
- **Fern compliance program** — Easily onboard businesses and individuals globally with Fern's KYC/B forms branded with your company name and logo
- **Developer experience** — High-touch customer support, scalable API design, and a focus on reliability and responsiveness

## Getting Started

The Fern API is currently in private beta. To begin integration:

1. **Undergo business verification (KYB)**
   - Create an account at [app.fernhq.com](https://app.fernhq.com), indicating your business use case
   - Complete KYB (takes 10-15 minutes)
   - For details on required information, refer to Requirements - Businesses

2. **Get full access to Fern Business and the Developer dashboard**
   - Fern will set up a dedicated Slack or Telegram channel to connect you directly with their team
   - Once verified and approved, you'll have full access to Fern

3. **Commercials & API keys**
   - After finalizing commercials with the Fern team, they will share your API keys securely
   - Store the key safely, as it won't be visible in the Developer dashboard afterwards

4. **Start building**
   - Create your first customers and transactions
   - Reach out to the Fern team through Slack, Telegram, or email for support

## Earning Revenue with Fern

Fern creates revenue generation opportunities for developer partners:

- **Currency conversion**: For on/offramps, FX, and crypto swaps, set a developer fee that Fern deducts from transactions. Revenue is settled monthly in an agreed-upon currency.
- **Multi-currency accounts and wallets**: Charge end users for creating multi-currency accounts and wallets.
- **Yield**: Some Fern products enable users to earn yield on funds at rest. You can specify a percentage to earn on generated yield, as well as the amount for your users.
- **Cards**: Fern shares interchange fees with partners integrating their cards solution.

## Developer Dashboard

Fern's developer dashboard is accessible at [app.fernhq.com](https://app.fernhq.com) and offers:

- Customers table displaying your customers and their status
- Transactions table showing all transactions initiated by your customers
- Quick link to the documentation
- More features coming soon

## Help & Support

Fern provides:
- Dedicated Slack/Telegram channel with their team
- Email support at [support@fernhq.com](mailto:support@fernhq.com)

Their business hours are 9AM to 6PM Pacific Time, Monday to Friday, with after-hours and 24/7 support for critical issues.

## Coverage

### Customer Types

Fern supports business and individual customers across 150+ countries:

**Businesses**:
- DAOs
- Corporations
- Partnerships
- Trusts
- LLCs
- Non-profits
- Other

**Individuals**

### Restricted Customer Geographies

Fern cannot onboard customers from:
- Belarus
- Cuba
- Iran
- North Korea
- Russia
- Syria
- Ukraine: Crimea, Donetsk, Luhansk
- US: Alaska, Hawaii, Louisiana, New York

Additionally, multi-currency virtual accounts are not available for customers from numerous regions including Afghanistan, Albania, China, and others. For a full list, see the documentation.

### Fiat Currency Support

Fern supports several fiat currencies via local payment rails across Africa, APAC, Europe, Middle East, North America, and South America. They also support SWIFT transfers for international wires to all supported regions.

### Cryptocurrency Support

Fern supports stablecoins and other cryptocurrencies on these networks:

| Network | Chain ID | Native Currency | Supported Cryptocurrencies |
|---------|----------|-----------------|----------------------------|
| Arbitrum One | 42161 (0xa4b1) | ETH | USDC, USDT, ETH, ERC-20s |
| Base | 8453 (0x2105) | ETH | USDC, USDT, ETH, ERC-20s |
| Ethereum Mainnet | 1 (0x1) | ETH | USDC, USDT, ETH, ERC-20s |
| OP Mainnet | 10 (0xa) | ETH | USDC, USDT, ETH, ERC-20s |
| Polygon Mainnet | 137 (0x89) | POL | USDC, USDT, ETH, ERC-20s |
| Solana | N/A | SOL | USDC, USDT, SOL |
| Tron | N/A | TRX | USDT |
| Bitcoin | N/A | BTC | BTC |

## Core API Concepts

Fern's core API concepts include:

- **Customers** — Create your individual and business users as customers, uniquely identified by email address
- **Payment accounts** — Create accounts for customers and their contacts
- **Quotes** — Specify transaction details and receive exchange rates and other information
- **Transactions** — Confirm quote details, including instructions for fund transfers

## Create and Verify Customers

Before customers can transact with Fern, they must complete verification (KYC/KYB):

1. **Create a customer**
   ```json
   {
     "customerType": "INDIVIDUAL",
     "email": "bob@builder.com",
     "firstName": "Bob",
     "lastName": "The Builder"
   }
   ```
   Sample response includes a hosted KYC link:
   ```json
   {
     "customerId": "5be4866c-4d85-4861-903d-a6295dfdf1e1",
     "customerStatus": "created",
     "email": "bob@builder.com",
     "customerType": "INDIVIDUAL",
     "name": "Bob The Builder",
     "kycLink": "https://app.fernhq.com/verify-customer/5be4866c-4d...",
     "updatedAt": "2025-04-29T13:31:25.954Z",
     "organizationId": "73fc5722-ae38-4530-a45f-db2df8f69810"
   }
   ```

2. **Customer completes verification via KYC link**
   - Share the hosted KYC form link directly with your end user or complete it on their behalf
   - For individuals, completion takes about 3 minutes
   - For businesses, completion takes 10-15 minutes
   - After completion, status changes from `CREATED` to `PENDING`

3. **Monitor customer verification status**
   - Use the Customers API GET endpoint to check for status updates
   - Status updates also arrive via webhooks
   - When fully approved, status changes to `ACTIVE`

### Requirements for Individuals

Information collected for KYC:
- First name
- Last name
- Address
- Date of birth
- Government-issued ID number (SSN for US residents)
- Government-issued identity document
- Proof of address (optional for some users, mandatory for certain regions)

Typical completion time: <5 minutes  
Typical time to approval: <5 minutes to 1 business day

### Requirements for Businesses

Information collected for KYB:
- Business details (name, type, address, registration number, website, description)
- Source of funds
- Beneficial owner information
- Controller information
- Business documents (formation documents, ownership documents, proof of address)

Typical completion time: 10-15 minutes  
Typical time to approval: 1-3 business days

### Customer Statuses

| Status | Definition |
|--------|------------|
| CREATED | Customer object has been created |
| PENDING | Customer is pending verification |
| ACTIVE | Customer is approved and ready to transact |
| REJECTED | Customer is rejected |
| DEACTIVATED | Customer has been deactivated |

## Create Fern Wallets

Fern enables you to create wallets for your customers on all supported chains:

1. **Create your customer**
   - Use the Customers API to create a customer and obtain a `customerId`
   - Customers don't need to be `ACTIVE` to have an active Fern wallet

2. **Create Fern wallet for the customer**
   - Use the Payment accounts API
   - Specify `paymentAccountType` as `FERN_CRYPTO_WALLET`
   - Include the `customerId`
   - For EVM wallets, include `fernCryptoWallet` object with `cryptoWalletType`: `EVM`

```json
{
  "paymentAccountType": "FERN_CRYPTO_WALLET",
  "customerId": "4d0b92ee-b48f-471f-8825-61761c0c6eb7",
  "fernCryptoWallet": {
    "cryptoWalletType": "EVM"
  }
}
```

Sample response:
```json
{
  "paymentAccountType": "FERN_CRYPTO_WALLET",
  "paymentAccountId": "238e98ce-2305-57e0-83f1-c3286e7db76c",
  "createdAt": "2025-05-07T13:55:06.697Z",
  "fernCryptoWallet": {
    "cryptoWalletType": "EVM",
    "address": "0xb9a71467bc058c744b07285ec12533ac88095270"
  },
  "isThirdParty": false
}
```

This wallet can now be used as the destination for onramps or a source for offramps.

## First-party Onramps

Fern's first-party onramps enable customers to convert fiat into crypto:

1. **Create a bank account for a verified customer**
   - Create a bank account for the customer using the Bank Accounts endpoint
   - Ensure the beneficiary name matches the customer's name

2. **Generate quote**
   - Use the Quotes endpoint to fetch a proposed price for currency conversion
   - This guarantees the price for 5 minutes and provides transparent details
   
   Sample request:
   ```bash
   curl --location 'https://api.fernhq.com/quotes/' \
   --header 'Content-Type: application/json' \
   --header 'Authorization: Bearer <API_TOKEN>' \
   --data '{
     "customerId": "b8643416-7a36-4437-95af-c32ba3b44257",
     "source": {
       "sourcePaymentAccountId": "550f6386-9ae0-4aa8-87f1-74f628e...",
       "sourceCurrency": "USD",
       "sourcePaymentMethod": "ACH",
       "sourceAmount": "11"
     },
     "destination": {
       "destinationPaymentAccountId": "9e67ed18-4b77-41e6-8839-5d...",
       "destinationPaymentMethod": "BASE",
       "destinationCurrency": "USDC"
     }
   }'
   ```
   
   Sample response:
   ```json
   {
     "quoteId": "672cec1f-8224-41ca-b77a-46307ae0f213",
     "expiresAt": "2025-04-08T20:36:09.129Z",
     "estimatedExchangeRate": "1",
     "destinationAmount": "10.478",
     "minGuaranteedDestinationAmount": "10.478",
     "fees": {
       "feeCurrency": {
         "label": "USD"
       },
       "fernFee": {
         "feeAmount": "0.522",
         "feeUSDAmount": "0.522"
       },
       "developerFee": {
         "feeAmount": "0",
         "feeUSDAmount": "0"
       }
     }
   }
   ```

3. **Submit transaction**
   - Use the quote ID to generate an onramp transaction
   - Once created, you'll receive transfer instructions to share with your customer
   
   Sample request:
   ```bash
   curl --location 'https://api.fernhq.com/transactions/' \
   --header 'Content-Type: application/json' \
   --header 'Authorization: Bearer <API_TOKEN>' \
   --data '{
     "quoteId": "672cec1f-8224-41ca-b77a-46307ae0f213"
   }'
   ```
   
   Sample response:
   ```json
   {
     "transactionId": "477f21e2-1b67-5828-a43d-dab19316a711",
     "transactionStatus": "AWAITING_TRANSFER",
     "source": {
       "sourceCurrency": {
         "label": "USD"
       },
       "sourcePaymentMethod": "ACH",
       "sourceAmount": "11",
       "sourcePaymentAccountId": "550f6386-9ae0-4aa8-87f1-74f628e328..."
     },
     "destination": {
       "destinationPaymentAccountId": "9e67ed18-4b77-41e6-8839-5d00e...",
       "destinationPaymentMethod": "BASE",
       "destinationCurrency": {
         "label": "USDC"
       },
       "exchangeRate": "1",
       "destinationAmount": "10.478",
       "minGuaranteedDestinationAmount": "10.478"
     },
     "fees": {
       "feeCurrency": {
         "label": "USD"
       },
       "fernFee": {
         "feeAmount": "0.522",
         "feeUSDAmount": "0.522"
       },
       "developerFee": {
         "feeAmount": "0",
         "feeUSDAmount": "0"
       }
     },
     "createdAt": "2025-04-08T20:30:09.129Z",
     "correlationId": "",
     "transferInstructions": {
       "type": "fiat",
       "transferPaymentMethod": "ACH",
       "transferMessage": "ABCXXXXXXXXXXXX",
       "transferBankName": "Test Bank",
       "transferBankAddress": "500 Main St., Some City, SC 99999",
       "transferBankAccountNumber": "123456123456",
       "transferBankBeneficiaryName": "Bank Name",
       "transferACHRoutingNumber": "123456789"
     }
   }
   ```

4. **Monitor transaction status**
   - Track progress by calling the Transactions endpoint
   - Check the Developer dashboard
   - Subscribe to webhooks for real-time notifications (coming soon)

## First-party Offramps

Fern's first-party offramps enable customers to convert crypto into fiat:

1. **Create a customer**
   - Use the Customer API to create your end customer in the Fern system

2. **Create a payment account for your customer**
   - Create an external bank account either via API or using a custom-branded hosted bank account form

3. **Generate quote for the offramp**
   - Create a quote using the POST endpoint
   - For an offramp of USDC, set `sourceCurrency` as "USDC" and `sourcePaymentMethod` as the source chain (e.g., "BASE")
   - Include destination payment account details

4. **Submit transaction**
   - Use the generated `quoteId` to submit a transaction
   - Transfer instructions will include a crypto wallet address for sending funds

5. **Monitor transaction status**
   - Call the Transactions API endpoint
   - Subscribe to webhooks for real-time notifications
   - Check the Developer dashboard

## Webhooks

Webhooks allow your application to receive real-time notifications from Fern for:

- Customer events (`customer.created`, `customer.updated`)
- Payment Account events (`payment_account.created`, `payment_account.deleted`)
- Transaction events (`transaction.created`, `transaction.updated`)

To implement webhooks:

1. Register a webhook subscription (via dashboard or API)
2. Provide an HTTPS endpoint URL
3. Store the unique secret securely for request verification
4. Respond with a 200 OK status upon receipt
5. Implement idempotent handlers (same event could be sent multiple times)

### Verification

For security, verify webhook signatures:

1. Get signature and timestamp from headers (`x-api-signature`, `x-api-timestamp`)
2. Compute expected signature using your secret: 
   ```
   stringToSign = "<timestamp>.<raw_body>"
   signature = HMAC_SHA256(secret, stringToSign)
   ```
3. Compare signatures using constant-time comparison
4. Verify timestamp freshness (within a few minutes)
5. Process the webhook only if verification succeeds

### Retries

Fern automatically retries failed webhook deliveries up to 4 times:
- 1st: Immediate
- 2nd: 5 seconds delay
- 3rd: 30 seconds delay
- 4th: 1 minute delay

## Transaction Sizes and Statuses

- Minimum transaction value: $10 (based on sending amount)
- No upper limit on transaction size

### Transaction Statuses

**Onramps**:
- `AWAITING_TRANSFER`: Validated and created successfully
- `PROCESSING`: Processing ongoing, funds received
- `COMPLETED`: Successful completion
- `FAILED`: Transaction failed, funds returned to sending address

**Offramps**:
- `PROCESSING`: Funds received at transfer address
- `COMPLETED`: Transaction completed
- `FAILED`: Transaction failed

### Transaction Failures

Possible failure reasons include:
- `funds_source_mismatch`: Incoming funds sent from wrong bank account
- `name_mismatch`: Funds sent from account with mismatched name
- `price_impact_or_slippage`: Receiving amount below minimum guaranteed
- `destination_returned_funds`: Destination bank account returned funds
- `insufficient_balance`: Insufficient balance in sending wallet
- `missing_chain_id`: Chain ID not specified
- `missing_minimum_guaranteed`: Minimum guaranteed amount not specified

## Slippage and Price Impact

For ERC-20 token conversions with small market caps and volatile prices, significant price impact may occur. Fern offers a minimum guaranteed amount for each transaction. If this amount can't be achieved, the transaction fails and funds return to the sender.

## API Reference

The documentation includes detailed API endpoints for:

- **Customers** - Create, retrieve, and list customers
- **Payment accounts** - Create, retrieve, and list payment accounts
- **Quotes** - Create quotes for transactions
- **Transactions** - Create and retrieve transaction details

Each section includes request/response examples with full JSON schema details.
