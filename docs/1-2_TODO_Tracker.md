# 1-2 TODO Tracker

This document tracks all TODO comments in the codebase and cross-references them with the platform enhancement implementation plan.

**Last Updated:** 2025-07-28  
**Total TODOs in Code:** 71  
**Legend:** 
- ✅ = Implemented in current session
- 🔄 = In progress 
- ⚠️ = High priority
- 📋 = Matches implementation plan item

---

## Summary Statistics

| Category | Count | Priority Items |
|----------|-------|----------------|
| Backend (Go) | 64 | 38 |
| Frontend (TypeScript) | 6 | 2 |
| Documentation | 1 | 0 |

---

## High Priority TODOs

### ⚠️ Critical Infrastructure

1. **Admin Function Security** ⚠️
   - `/apps/api/handlers/account_handlers.go:126` - Update because this is an admin only function
   - `/apps/api/handlers/account_handlers.go:200` - Update because this is an admin only function
   - **Status:** Need proper admin authentication/authorization

2. **Tax System Implementation** 📋
   - `/libs/go/services/tax_service.go:226` - Implement once tax_calculations table is created
   - `/libs/go/services/invoice_service.go:133-134` - Convert jurisdiction to UUID, get reverse charge from tax calculation
   - **Status:** Phase 4.4 (Tax Service) in implementation plan

3. **Exchange Rate Service** 📋
   - `/libs/go/services/exchange_rate_service.go:236` - Implement actual database query
   - `/libs/go/services/exchange_rate_service.go:264` - Implement database storage
   - **Status:** Phase 4.3 (Exchange Rate Service) in implementation plan

4. **Interface Mismatches** ⚠️
   - `/apps/api/handlers/factory.go:191` - Fix interface mismatch - subscription management service expects different email service interface
   - `/apps/api/handlers/factory.go:194` - Fix interface mismatch - dunning retry engine expects DelegationClientInterface but receives *DelegationClient
   - **Status:** Critical - affects service initialization

---

## Backend TODOs by Service

### Analytics & Dashboard Services

#### Dashboard Metrics
- `/libs/go/services/dashboard_metrics_service.go:210` - Calculate expansion and contraction revenue by comparing subscription changes 📋
- **Status:** Part of Phase 4.1 Enhanced Analytics

#### Analytics Service
- `/libs/go/services/analytics_service.go:533` - Implement metrics refresh using background job system
- **Status:** Needs background job infrastructure

#### Workspace Service
- `/libs/go/services/workspace_service.go:256` - Implement actual statistics queries
- **Status:** Requires database queries for workspace analytics

### Payment & Subscription Services

#### Subscription Management
- `/libs/go/services/subscription_management_service.go:453` - Implement payment processing
- `/apps/api/handlers/subscription_management_handler.go:94` - Implement proper ownership verification
- **Status:** Core payment processing integration needed

#### Subscription Processor
- `/apps/subscription-processor/internal/processor/scheduled_changes_processor.go:364` - Implement downgrade logic based on final_action_config
- **Status:** Dunning system integration point

#### Payment Failure Monitoring
- `/libs/go/services/payment_failure_monitor.go:61` - Create a specific query for failed payments
- `/libs/go/services/payment_failure_monitor.go:138` - Create a query to get failed subscription events  
- `/libs/go/services/payment_failure_monitor.go:177` - Get currency from price
- **Status:** Database queries needed for monitoring

### Dunning System 📋

#### Core Dunning Service
- `/libs/go/services/dunning_service.go:208` - Send recovery success notification
- `/libs/go/services/dunning_service.go:209` - Update analytics
- `/libs/go/services/dunning_service.go:267` - Pause subscription
- `/libs/go/services/dunning_service.go:270` - Downgrade subscription based on config
- **Status:** Integration with subscription management needed

#### Dunning Retry Engine
- `/libs/go/services/dunning_retry_engine.go:356` - Get product name from subscription
- `/libs/go/services/dunning_retry_engine.go:359` - Get actual payment link
- `/libs/go/services/dunning_retry_engine.go:360` - Get support email from workspace settings
- `/libs/go/services/dunning_retry_engine.go:361` - Get merchant name from workspace settings
- `/libs/go/services/dunning_retry_engine.go:416` - Create actual in-app notification
- `/libs/go/services/dunning_retry_engine.go:458` - Query for failed payments that don't have active dunning campaigns
- **Status:** Workspace integration and notification system needed

#### Dunning Processing (Integrated into Subscription Processor)
- Dunning functionality has been integrated directly into the subscription processor
- **Status:** Integrated and operational - no longer requires separate processor

### Invoice System

#### Invoice Service
- `/libs/go/services/invoice_service.go:396` - Get actual customer tier
- `/libs/go/services/invoice_service.go:535-536` - Add ProductID and PriceID if available
- `/libs/go/services/invoice_service.go:570-571` - Add PaymentLinkID and PaymentLinkURL if available
- **Status:** Product and payment link integration needed

#### Invoice Handler
- `/apps/api/handlers/invoice_handler.go:246` - Implement email sending logic
- `/apps/api/handlers/invoice_handler.go:282` - Implement payment link generation
- **Status:** Email service and payment link service integration

### Payment Page System

#### Payment Page Handler
- `/apps/api/handlers/payment_page_handler.go:94` - Add logo URL to workspace table
- `/apps/api/handlers/payment_page_handler.go:95` - Add brand color to workspace settings
- `/apps/api/handlers/payment_page_handler.go:101` - Implement product retrieval
- `/apps/api/handlers/payment_page_handler.go:113` - Implement price retrieval
- `/apps/api/handlers/payment_page_handler.go:180` - Implement proper wallet selection based on network
- `/apps/api/handlers/payment_page_handler.go:188` - Get actual wallet address
- `/apps/api/handlers/payment_page_handler.go:193` - Store payment intent in database/cache for later processing
- `/apps/api/handlers/payment_page_handler.go:214` - Get networks that have wallets configured for this workspace
- `/apps/api/handlers/payment_page_handler.go:235` - Get tokens configured for this product/workspace
- `/apps/api/handlers/payment_page_handler.go:268` - Implement proper gas sponsorship checking
- **Status:** Multiple integrations needed - workspace, product, wallet, gas sponsorship

### Gas & Blockchain Services

#### Gas Fee Service
- `/libs/go/services/gas_fee_service.go:157` - Implement network lookup by name
- **Status:** Network service integration needed

#### Gas Sponsorship Service
- `/libs/go/services/gas_sponsorship_service.go:325` - Implement once GetGasSponsorshipAnalytics query is added
- **Status:** Analytics database queries needed

#### Blockchain Service
- `/libs/go/services/blockchain_service.go:208` - Future blockchain service capabilities
- **Status:** Long-term enhancement

### Product & Pricing

#### Product Service
- `/libs/go/services/product_service.go:296` - Add check for active subscriptions when the method is available
- **Status:** Subscription service integration needed

#### Product Handler
- `/apps/api/handlers/product_handlers.go:729` - Fix type conversion from helpers.ProductTokenResponse to responses.ProductTokenResponse
- **Status:** Type system cleanup needed

### API Routes Implementation

#### Server Routes
- `/apps/api/server/server.go:618` - Implement ListWorkspaceCustomers
- `/apps/api/server/server.go:636` - Implement subscription methods
- `/apps/api/server/server.go:641` - Implement GetSubscriptionWithDetails
- `/apps/api/server/server.go:644` - Implement subscription status methods
- `/apps/api/server/server.go:660` - Implement subscription event analytics methods
- `/apps/api/server/server.go:672` - Implement CreateSubscriptionEvent and UpdateSubscriptionEvent
- `/apps/api/server/server.go:676` - Implement subscription event filtering methods
- **Status:** Multiple handler implementations needed

### Miscellaneous Services

#### Redemption Processor
- `/libs/go/services/redemption_processor_service.go:483` - Add crypto amount and exchange rate if available
- `/libs/go/services/redemption_processor_service.go:487` - Add gas fee information if available
- **Status:** Exchange rate and gas fee integration needed

#### Payment Sync
- `/libs/go/client/payment_sync/stripe/webhook.go:126` - Add cases for other relevant event types
- **Status:** Stripe webhook enhancement

#### Test Utilities
- `/libs/go/testutil/database.go:21` - Implement database cleanup
- **Status:** Testing infrastructure improvement

#### Type Definitions
- `/libs/go/types/business/delegation.go:15` - Add caveat fields
- **Status:** Delegation system enhancement

---

## Frontend TODOs

### Customer Portal
- `/apps/web-app/src/app/customers/settings/page.tsx:35` - Implement profile save functionality
- `/apps/web-app/src/app/customers/settings/page.tsx:40` - Implement account deletion
- `/apps/web-app/src/app/customers/settings/page.tsx:45` - Implement logout functionality
- **Status:** Customer portal functionality needs implementation

### Wallet Management
- `/apps/web-app/src/app/api/wallets/route.ts:22` - If filtering is needed, it should be implemented in the WalletsAPI service
- `/apps/web-app/src/components/wallets/circle-wallet-balances.tsx:60` - Refactor fully for multi-network support (pass networks prop, use generateExplorerLink)
- `/apps/web-app/src/components/wallets/circle-wallet-balances.tsx:123` - Add secondary sort (e.g., native first, then value)
- `/apps/web-app/src/components/wallets/circle-wallet-balances.tsx:202` - Ensure API response includes decimals for gas_token
- **Status:** Multi-network wallet support enhancement

### Authentication
- `/apps/web-app/src/components/env/provider.tsx:6` - Get authentication status from Web3Auth
- `/apps/web-app/src/components/public/web3auth-delegation-button.tsx:394` - Update code for specific caveat enforcers
- **Status:** Web3Auth integration improvements

---

## Documentation TODOs

### Integration Guides
- `/docs/dunning-management-guide.md:189` - TODO: Remaining Integrations
- **Status:** Documentation completeness

---

## Implementation Plan Cross-Reference

### Phase 4 TODOs Status

#### ✅ Phase 4.6: Subscription Management (Completed)
- Database schema created
- SQLC queries implemented
- ProrationCalculator service created
- SubscriptionManagementService implemented
- All subscription operations (upgrade, downgrade, cancel, pause, resume)
- Scheduled changes processor
- Dunning system integration
- Email notifications for all changes

#### 🔄 Phase 4.4: Tax Service
- Tax calculations table - TODO at `/libs/go/services/tax_service.go:226`
- Jurisdiction UUID conversion - TODO at `/libs/go/services/invoice_service.go:133-134`
- **Status:** Database schema and service implementation needed

#### 🔄 Phase 4.3: Exchange Rate Service
- Database implementation - TODO at `/libs/go/services/exchange_rate_service.go:236,264`
- **Status:** Persistent storage and queries needed

#### 🔄 Phase 4.5: Discount Service
- No specific TODOs found in code
- **Status:** Implementation appears complete

#### 🔄 Phase 4.1: Enhanced Analytics
- Analytics refresh job - TODO at `/libs/go/services/analytics_service.go:533`
- Workspace statistics - TODO at `/libs/go/services/workspace_service.go:256`
- Dashboard metrics - TODO at `/libs/go/services/dashboard_metrics_service.go:210`
- **Status:** Background job system and analytics queries needed

#### 🔄 Phase 4.2: API Completeness  
- Multiple TODOs in `/apps/api/server/server.go` for implementing API methods
- **Status:** Handler implementations needed for remaining endpoints

### New Categories Identified

#### Payment Page System (Not in original plan)
- 10 TODOs in `/apps/api/handlers/payment_page_handler.go`
- **Priority:** High - customer-facing payment flow
- **Status:** Needs workspace branding, product integration, wallet management

#### Dunning System Enhancements
- 9 TODOs across dunning services
- **Priority:** Medium - system functionality improvements
- **Status:** Workspace integration and notification system needed

#### Payment Failure Monitoring (Not in original plan)  
- 3 TODOs in `/libs/go/services/payment_failure_monitor.go`
- **Priority:** High - business critical monitoring
- **Status:** Database queries and monitoring infrastructure needed

---

## TODO Priority Matrix

### Immediate Action Required (Next Sprint) ⚠️
1. **Interface Mismatches** - Fix service initialization issues in factory.go
2. **Admin Function Security** - Implement proper admin authentication for account handlers
3. **Payment Failure Monitoring** - Implement database queries for critical monitoring
4. **Payment Page Branding** - Add workspace logo and brand color fields

### Short Term (1-2 Sprints)
1. **Tax Service Implementation** - Complete Phase 4.4 with database schema
2. **Exchange Rate Service** - Complete Phase 4.3 with persistent storage
3. **Payment Page System** - Complete all 10 TODOs for customer payment flow
4. **API Route Implementations** - Complete missing subscription and customer endpoints

### Medium Term (3-4 Sprints)
1. **Analytics Implementation** - Phase 4.1 with background job system
2. **Dunning System Enhancements** - Workspace integration and notifications
3. **Invoice System** - Email sending and payment link generation
4. **Gas Sponsorship Analytics** - Complete analytics queries

### Long Term (5+ Sprints)
1. **Frontend Customer Portal** - Settings page functionality
2. **Multi-network Wallet Support** - Refactor wallet components
3. **Blockchain Service Enhancements** - Future capabilities
4. **Advanced Analytics Features** - Expansion/contraction revenue tracking

---

## Critical Path Analysis

### Blocking Issues (Fix First)
1. **Service Factory Interface Mismatches** - Prevents proper service initialization
2. **Admin Authentication** - Security vulnerability in account handlers
3. **Payment Failure Monitoring** - Business critical for detecting issues

### Customer-Facing Priority
1. **Payment Page System** - Direct customer experience impact
2. **Invoice Email Sending** - Customer communication
3. **Frontend Customer Portal** - User self-service capabilities

### Business Logic Priority  
1. **Tax Service** - Compliance and revenue calculation
2. **Exchange Rate Service** - Accurate crypto pricing
3. **Analytics Enhancements** - Business intelligence

---

## Recently Updated (2025-07-28)

### Changes Made
- Updated total TODO count from 104 to 71 (removed outdated/completed items)
- Reorganized TODOs by service and functional area
- Added accurate file paths and line numbers from current codebase
- Identified new priority categories not in original implementation plan
- Updated phase status based on current completion state

### New High-Priority Categories Identified
1. **Payment Page System** - 10 TODOs, customer-facing
2. **Interface Mismatches** - Critical service initialization issues  
3. **Payment Failure Monitoring** - Business critical monitoring gaps

### Patterns Observed
- Payment and subscription systems have most TODOs (business critical)
- Many TODOs involve service integration points
- Frontend has fewer but important customer-facing TODOs
- Analytics TODOs are numerous but lower priority

---

## Maintenance Notes

This document should be updated:
- When TODOs are completed (mark with ✅)
- When new TODOs are added to the codebase
- After each major feature implementation
- During sprint planning to prioritize work
- When file paths or line numbers change significantly

**Search Command:** `grep -r "TODO:" --include="*.go" --include="*.ts" --include="*.tsx" --include="*.md" .`

**Last Full Audit:** 2025-07-28