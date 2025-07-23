# Cyphera Web Documentation

This folder contains the technical documentation for the Cyphera Web application.

## Table of Contents

- [Overview](#overview)
- [Architecture and Flows](#architecture-and-flows)
  - [System Architecture](#system-architecture)
  - [Authentication Flow](#authentication-flow)
  - [Wallet Management Flow](#wallet-management-flow)
  - [Subscription Management Flow](#subscription-management-flow)
- [Key Components](#key-components-explanation)
- [Technical Stack](#technical-stack-details)
- [User Journeys](#user-journeys)
- [Conclusion](#conclusion)
- [Additional Documentation](#additional-documentation)

## Overview

Cyphera Web is a comprehensive cryptocurrency-based subscription platform that enables businesses to create, manage, and monetize subscription products using blockchain technology. The platform integrates with Circle's Programmable Wallet technology to provide self-custody wallet solutions, transaction handling, and subscription management capabilities.

## Architecture and Flows

### System Architecture

![Architecture Diagram](./architecture_diagram.png)

**Architecture Diagram Files:**

- [Interactive Mermaid Source](./architecture_diagram.mmd)
- [Static PNG Image](./architecture_diagram.png)

### Authentication Flow

![Authentication Flow](./authentication_flow.png)

**Authentication Flow Files:**

- [Interactive Mermaid Source](./authentication_flow.mmd)
- [Static PNG Image](./authentication_flow.png)

### Wallet Management Flow

![Wallet Management Flow](./wallet_management_flow.png)

**Wallet Management Flow Files:**

- [Interactive Mermaid Source](./wallet_management_flow.mmd)
- [Static PNG Image](./wallet_management_flow.png)

### Subscription Management Flow

![Subscription Management Flow](./subscription_management_flow.png)

**Subscription Management Flow Files:**

- [Interactive Mermaid Source](./subscription_management_flow.mmd)
- [Static PNG Image](./subscription_management_flow.png)

## Key Components Explanation

### Authentication System

Cyphera uses Supabase for authentication, providing a secure and reliable way for users to log in and manage their accounts. The process includes:

1. **User Registration**: New users can create accounts using email/password
2. **Email Verification**: Ensures user emails are valid
3. **Session Management**: Secure token-based session handling
4. **Middleware Protection**: Routes are protected based on authentication state

### Wallet Management

The wallet management functionality is built around Circle's Programmable Wallet technology:

1. **User Initialization**: First-time users are initialized with Circle's system
2. **PIN Setup**: Users establish a secure PIN for transaction authorization
3. **Wallet Creation**: Users can create multiple blockchain wallets
4. **Balance Management**: View token balances across wallets
5. **Transaction History**: Track and review past transactions

### Product Management

Merchants can create and manage subscription products with features including:

1. **Product Creation**: Set up new subscription offerings
2. **Pricing Configuration**: Set prices in crypto tokens
3. **Subscription Terms**: Configure billing intervals and conditions
4. **Product Catalog**: Showcase offerings to potential customers

### Subscription Management

The platform handles the entire subscription lifecycle:

1. **Subscription Processing**: Handle customer subscriptions to products
2. **Payment Collection**: Automate recurring payments through smart contracts
3. **Subscription Tracking**: Monitor active subscriptions
4. **Renewal Management**: Handle subscription renewals and cancellations

### Transaction Management

Comprehensive transaction capabilities allow for:

1. **Transaction Creation**: Send tokens between wallets
2. **Fee Estimation**: Calculate network fees before submission
3. **Transaction Signing**: Secure PIN verification for authorizing transactions
4. **Transaction Monitoring**: Track status of pending and completed transactions

### Customer Management

Merchants can manage their customer relationships with:

1. **Customer Profiles**: View customer information
2. **Subscription Tracking**: See which products customers are subscribed to
3. **Communication Tools**: Engage with customers

## Technical Stack Details

### Frontend

- **Framework**: Next.js 14 with App Router
- **State Management**: React Query (TanStack Query)
- **UI Components**: Shadcn UI, Radix UI
- **Styling**: Tailwind CSS
- **Web3 Integration**: Viem v2, Wagmi v2
- **Form Handling**: React Hook Form with Zod validation

### Backend Integration

- **Authentication**: Supabase Auth
- **API Client**: Custom CypheraAPI client for backend communication
- **Circle Integration**: Dedicated CircleAPI client for Circle services
- **Smart Account Delegation**: MetaMask Delegation Toolkit

### External Integrations

- **Circle Programmable Wallets**: For wallet and transaction management
- **Blockchain Networks**: For on-chain operations
- **Supabase**: For authentication and data storage

## User Journeys

### Merchant Journey

1. **Onboarding**: Sign up, verify email, complete profile
2. **Product Setup**: Create subscription products with pricing and terms
3. **Dashboard**: Monitor active subscriptions and revenue
4. **Customer Management**: Track and manage customer relationships
5. **Wallet Management**: Handle crypto assets and transactions

### Customer Journey

1. **Discovery**: Browse available subscription products
2. **Subscription**: Select a product and complete the subscription process
3. **Wallet Setup**: Create or connect a wallet for payments
4. **Transaction**: Authorize payment for subscription
5. **Access**: Gain access to subscribed content or services

## Conclusion

The Cyphera Web application provides a comprehensive platform for crypto-based subscription management, combining the power of blockchain technology with user-friendly interfaces. The system's modular architecture allows for flexibility and scalability while maintaining a secure and reliable user experience for both merchants and customers.

## Additional Documentation

This README serves as the comprehensive documentation for the Cyphera Web application. All relevant information has been consolidated into this single document for clarity and ease of reference.

## Viewing Mermaid Diagrams

The `.mmd` files contain Mermaid syntax that can be rendered by:

1. Using the [Mermaid Live Editor](https://mermaid.live/)
2. Using browser extensions that render Mermaid syntax
3. Using GitHub's built-in Mermaid rendering (for GitHub repositories)
4. Using VS Code extensions like "Markdown Preview Mermaid Support"

## Generating Diagram Images

To regenerate diagram PNG files from the Mermaid source, use the following command:

```bash
mmdc -i docs/[filename].mmd -o docs/[filename].png -w 1600 -H 1200 -s 2
```

Example:

```bash
mmdc -i docs/architecture_diagram.mmd -o docs/architecture_diagram.png -w 1600 -H 1200 -s 2
```
