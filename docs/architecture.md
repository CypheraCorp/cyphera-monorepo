# Cyphera Platform Architecture

> **Navigation:** [← Back to README](../README.md) | [Quick Start →](quick-start.md) | [API Reference →](api-reference.md)

## Table of Contents

- [System Overview](#system-overview)
- [Core Components](#core-components)
- [Service Architecture](#service-architecture)
- [Data Flow](#data-flow)
- [Technology Stack](#technology-stack)
- [Security Architecture](#security-architecture)
- [Network Architecture](#network-architecture)

## System Overview

Cyphera is a Web3 payment infrastructure platform that enables merchants to accept cryptocurrency subscriptions with automatic billing through MetaMask delegation. The platform consists of multiple microservices working together to provide a seamless payment experience.

### High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend  │    │  Mobile Apps    │    │  Merchant API   │
│   (Next.js)     │    │   (Future)      │    │   Integration   │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │     Main API Server     │
                    │        (Go/Gin)         │
                    └────────────┬────────────┘
                                 │
            ┌────────────────────┼────────────────────┐
            │                    │                    │
   ┌────────▼─────────┐ ┌────────▼─────────┐ ┌───────▼────────┐
   │  Delegation      │ │   Subscription   │ │   Database     │
   │    Server        │ │    Processor     │ │  (PostgreSQL)  │
   │  (Node.js/gRPC)  │ │      (Go)        │ │                │
   └──────┬───────────┘ └──────────────────┘ └────────────────┘
          │
   ┌──────▼───────┐
   │  Blockchain  │
   │   Networks   │
   │ (Ethereum,   │
   │  Polygon,    │
   │  Arbitrum)   │
   └──────────────┘
```

## Core Components

### 1. Web Application ([`apps/web-app/`](../apps/web-app/README.md))
- **Technology:** Next.js 15 with App Router
- **Purpose:** Frontend interface for merchants and customers
- **Key Features:**
  - Merchant dashboard for subscription management
  - Customer portal for subscription payments
  - Web3Auth integration for authentication
  - Real-time wallet and transaction management

### 2. Main API Server ([`apps/api/`](../apps/api/README.md))
- **Technology:** Go with Gin framework, deployed as AWS Lambda
- **Purpose:** Core business logic and API orchestration
- **Key Features:**
  - RESTful API endpoints
  - Authentication middleware (JWT/Web3Auth)
  - Database operations via SQLC
  - Integration with external services (Circle, Stripe)

### 3. Delegation Server ([`apps/delegation-server/`](../apps/delegation-server/README.md))
- **Technology:** Node.js with gRPC
- **Purpose:** Blockchain operations and MetaMask delegation management
- **Key Features:**
  - Smart account creation and management
  - Transaction signing and execution
  - MetaMask Delegation Toolkit integration
  - Multi-network support (Ethereum, Polygon, Arbitrum)

### 4. Subscription Processor ([`apps/subscription-processor/`](../apps/subscription-processor/README.md))
- **Technology:** Go background service
- **Purpose:** Automated subscription billing and processing
- **Key Features:**
  - Recurring payment processing
  - Failed payment retry logic
  - Delegation credential management
  - Dead letter queue handling

### 5. Database Layer
- **Technology:** PostgreSQL with SQLC for type-safe queries
- **Purpose:** Persistent data storage and management
- **Key Features:**
  - User and workspace management
  - Subscription and billing data
  - Wallet and transaction records
  - Audit logging and event tracking

## Service Architecture

### API Communication Patterns

#### Synchronous Communication
- **Frontend ↔ Main API:** REST over HTTPS
- **Main API ↔ Delegation Server:** gRPC (high performance for blockchain ops)
- **Main API ↔ Database:** Direct PostgreSQL connections

#### Asynchronous Communication
- **Webhook Processing:** SQS queues with Lambda processors
- **Background Jobs:** Subscription processor with scheduled execution
- **Event Sourcing:** Database triggers and event tables

### Authentication Flow

```
┌──────────┐    ┌─────────────┐    ┌──────────────┐    ┌────────────┐
│   User   │    │  Web3Auth   │    │   Main API   │    │  Database  │
└─────┬────┘    └──────┬──────┘    └──────┬───────┘    └─────┬──────┘
      │                │                  │                  │
      │ 1. Login       │                  │                  │
      ├───────────────▶│                  │                  │
      │                │ 2. Verify &      │                  │
      │                │    Issue JWT     │                  │
      │◀───────────────┤                  │                  │
      │                │                  │                  │
      │ 3. API Request with JWT           │                  │
      ├──────────────────────────────────▶│                  │
      │                │                  │ 4. Validate JWT  │
      │                │                  │ & Get User       │
      │                │                  ├─────────────────▶│
      │                │                  │◀─────────────────┤
      │                │                  │ 5. Response      │
      │◀──────────────────────────────────┤                  │
```

## Data Flow

### Subscription Creation Flow

1. **Merchant Setup:**
   - Creates product and pricing via web interface
   - Configures supported networks and tokens
   - Sets up delegation permissions

2. **Customer Subscription:**
   - Customer connects wallet via Web3Auth
   - Selects subscription plan and payment method
   - Delegates spending permission to platform
   - Initial payment processed immediately

3. **Recurring Processing:**
   - Subscription processor runs scheduled jobs
   - Checks for due subscriptions
   - Executes payments using stored delegations
   - Handles failures and retry logic

### Payment Processing Flow

```
Customer → Web App → Main API → Delegation Server → Blockchain
    ↓                    ↓              ↓               ↓
Database ← Event Log ← Transaction ← Smart Contract ← USDC Transfer
```

## Technology Stack

### Backend Services
- **Languages:** Go (APIs, processors), Node.js (delegation server)
- **Frameworks:** Gin (REST), gRPC, Express
- **Database:** PostgreSQL with SQLC for type-safe queries
- **Deployment:** AWS Lambda, ECS, RDS
- **Monitoring:** CloudWatch, Structured logging

### Frontend
- **Framework:** Next.js 15 with App Router
- **Styling:** Tailwind CSS with shadcn/ui components
- **State Management:** Zustand for client state
- **Authentication:** Web3Auth with JWT tokens
- **Blockchain:** Viem, Wagmi, MetaMask Delegation Toolkit

### Infrastructure
- **Cloud Provider:** AWS
- **Container Orchestration:** Docker with ECS
- **Load Balancing:** Application Load Balancer
- **DNS/CDN:** CloudFront
- **Secrets Management:** AWS Secrets Manager
- **Message Queues:** SQS with dead letter queues

### Development Tools
- **Monorepo:** Nx workspace with Go workspaces
- **Code Generation:** SQLC (database), Protocol Buffers (gRPC)
- **Testing:** Go test, Jest, Playwright
- **CI/CD:** GitHub Actions
- **Documentation:** Swagger/OpenAPI

## Security Architecture

### Authentication & Authorization
- **Web3Auth Integration:** Social logins with Web3 wallet creation
- **JWT Tokens:** Stateless authentication with workspace context
- **Role-Based Access:** Merchant/customer roles with granular permissions
- **API Key Management:** Service-to-service authentication

### Blockchain Security
- **Delegation Management:** Secure storage of delegation credentials
- **Transaction Signing:** Hardware security modules for production
- **Network Isolation:** VPC with private subnets for sensitive operations
- **Audit Logging:** Comprehensive transaction and event logging

### Data Protection
- **Encryption at Rest:** Database and file storage encryption
- **Encryption in Transit:** TLS 1.3 for all communications
- **PII Handling:** Minimal data collection with anonymization
- **Compliance:** Prepared for PCI DSS requirements

## Network Architecture

### Supported Blockchain Networks

#### Production Networks
- **Ethereum Mainnet:** Primary network for enterprise customers
- **Polygon:** Lower cost transactions for retail subscriptions
- **Arbitrum:** Layer 2 scaling with Ethereum security

#### Development Networks
- **Sepolia:** Ethereum testnet for development and testing
- **Mumbai:** Polygon testnet for Mumbai-specific testing
- **Arbitrum Sepolia:** Arbitrum testnet for L2 development

### Network Configuration
- **Dynamic Network Support:** Runtime network switching
- **Token Support:** USDC and USDT across all networks
- **Gas Management:** Automated gas price optimization
- **Failover Logic:** Automatic network switching on congestion

### Circle API Integration
- **Wallet Management:** Programmable wallets for user funds
- **USDC Operations:** Native USDC minting and transfers
- **KYC/Compliance:** Integrated identity verification
- **Settlement:** Fiat on/off ramps through Circle

---

## Related Documentation

- **[Quick Start Guide](quick-start.md)** - Get started with development
- **[API Reference](api-reference.md)** - Complete API documentation
- **[Deployment Guide](deployment.md)** - Production deployment instructions
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

## Service-Specific Documentation

- **[Web Application](../apps/web-app/README.md)** - Frontend development guide
- **[Main API Server](../apps/api/README.md)** - Backend API development
- **[Delegation Server](../apps/delegation-server/README.md)** - Blockchain operations
- **[Subscription Processor](../apps/subscription-processor/README.md)** - Background job processing

---

*Last updated: $(date '+%Y-%m-%d')*
*For questions or contributions, see our [Contributing Guide](contributing.md)*