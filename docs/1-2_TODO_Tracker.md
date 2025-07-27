# 1-2 TODO Tracker

This document tracks all TODO comments in the codebase and cross-references them with the platform enhancement implementation plan.

**Last Updated:** 2025-07-26  
**Total TODOs in Code:** 104  
**Legend:** 
- ✅ = Implemented in current session
- 🔄 = In progress 
- ⚠️ = High priority
- 📋 = Matches implementation plan item

---

## Summary Statistics

| Category | Count | Priority Items |
|----------|-------|----------------|
| Backend (Go) | 72 | 42 |
| Frontend (TypeScript) | 8 | 3 |
| Documentation | 24 | N/A |

---

## High Priority TODOs

### ⚠️ Critical Infrastructure

1. **Environment Configuration**
   - `/apps/api/handlers/invoice_handler.go:38` - Get baseURL from environment
   - `/apps/api/handlers/payment_link_handler.go:24` - Get baseURL from environment
   - `/apps/api/handlers/payment_page_handler.go:21` - Get baseURL from environment
   - **Status:** Not started - Needs centralized configuration

2. **Tax System Implementation** 📋
   - `/libs/go/services/tax_service.go:263` - Implement tax_calculations table
   - `/libs/go/services/invoice_service.go:189-190` - Convert jurisdiction to UUID, get from tax calculation
   - **Status:** Phase 4.4 (Tax Service) in implementation plan

3. **Exchange Rate Service** 📋
   - `/libs/go/services/exchange_rate_service.go:245,271` - Implement database query/storage
   - **Status:** Phase 4.3 (Exchange Rate Service) in implementation plan

---

## Backend TODOs by Service

### Payment & Subscription Services

#### Payment Processing
- `/libs/go/services/subscription_management_service.go:452` - Implement payment processing for resume ✅
- `/apps/api/handlers/payment_page_handler.go:283,291,296` - Implement wallet selection and payment intent storage
- `/libs/go/services/payment_failure_monitor.go:176` - Get currency from price

#### Invoice Service
- `/apps/api/handlers/invoice_handler.go:323` - Implement email sending logic
- `/apps/api/handlers/invoice_handler.go:359` - Implement payment link generation
- `/libs/go/services/invoice_service.go:189-190` - Tax calculation integration

### Dunning System 📋

#### Core Dunning
- `/libs/go/services/dunning_service.go:223-224` - Send recovery success notification ✅ (Email service added)
- `/libs/go/services/dunning_service.go:282` - Pause subscription implementation
- `/libs/go/services/dunning_service.go:285` - Downgrade subscription based on config
- `/apps/subscription-processor/internal/processor/scheduled_changes_processor.go:299` - Implement downgrade logic

#### Dunning Retry Engine
- `/libs/go/services/dunning_retry_engine.go:166` - Integrate with delegation server to retry payment
- `/libs/go/services/dunning_retry_engine.go:245-250` - Get actual product/workspace details
- `/libs/go/services/dunning_retry_engine.go:307` - Create in-app notification
- `/libs/go/services/dunning_retry_engine.go:349` - Query failed payments

#### Dunning Scheduled Job
- `/apps/api/handlers/dunning_handler.go:574` - Should be called by scheduled job in production ⚠️
- **Status:** Partially implemented in subscription processor

### Gas Sponsorship Service 📋
- `/libs/go/services/gas_fee_service.go:206` - Implement network lookup by name
- `/libs/go/services/gas_sponsorship_service.go:361` - Implement GetGasSponsorshipAnalytics query
- `/apps/api/handlers/payment_page_handler.go:338,371` - Check gas sponsorship configuration

### Analytics & Metrics
- `/libs/go/services/workspace_service.go:290` - Implement actual statistics queries
- `/libs/go/services/analytics_service.go:643` - Implement metrics refresh using background job
- `/libs/go/services/dashboard_metrics_service.go:209` - Calculate expansion/contraction revenue
- `/apps/dunning-processor/internal/processor/processor.go:46` - Implement proper statistics tracking

### Workspace Features
- `/apps/api/handlers/payment_page_handler.go:197-198` - Add logo URL and brand color to workspace ⚠️
- `/apps/api/handlers/payment_page_handler.go:204,216` - Implement product/price retrieval

### API Routes Implementation
- `/apps/api/server/server.go:614` - Implement ListSubscriptions method
- `/apps/api/server/server.go:632` - Implement ListCustomers method
- `/apps/api/server/server.go:637` - Implement GetCustomer method
- `/apps/api/server/server.go:640` - Implement UpdateCustomer method
- `/apps/api/server/server.go:656` - Implement GetProduct method
- `/apps/api/server/server.go:668` - Implement ListInvoices method
- `/apps/api/server/server.go:672` - Implement GetInvoice method

### Miscellaneous Backend
- `/apps/api/handlers/account_handlers.go:176,250` - Update admin-only functions
- `/apps/api/handlers/product_handlers.go:155` - Add caveat fields
- `/apps/api/handlers/subscription_management_handler.go:93` - Implement proper ownership verification ✅
- `/apps/api/handlers/redemption_processor.go:502,506` - Add crypto amount/exchange rate, gas fee info
- `/libs/go/client/payment_sync/stripe/webhook.go:126` - Add cases for other event types
- `/libs/go/services/product_service.go:340` - Add check for active subscriptions
- `/libs/go/services/blockchain_service.go:234` - Future blockchain service capabilities

---

## Frontend TODOs

### Customer Portal
- `/apps/web-app/src/app/customers/settings/page.tsx:35` - Implement profile save functionality
- `/apps/web-app/src/app/customers/settings/page.tsx:40` - Implement account deletion
- `/apps/web-app/src/app/customers/settings/page.tsx:45` - Implement logout functionality

### Wallet Management
- `/apps/web-app/src/app/api/wallets/route.ts:22` - Implement filtering in WalletsAPI service
- `/apps/web-app/src/components/wallets/circle-wallet-balances.tsx:60` - Refactor for multi-network support
- `/apps/web-app/src/components/wallets/circle-wallet-balances.tsx:123` - Add secondary sort by asset name
- `/apps/web-app/src/components/wallets/circle-wallet-balances.tsx:202` - Ensure API includes decimals

### Authentication
- `/apps/web-app/src/components/env/provider.tsx:6` - Get authentication status from Web3Auth
- `/apps/web-app/src/components/public/web3auth-delegation-button.tsx:394` - Update code for specific caveat enforcers

---

## Implementation Plan Cross-Reference

### Phase 4 TODOs Status

#### ✅ Phase 4.6: Subscription Management (Completed in this session)
- Database schema created
- SQLC queries implemented
- ProrationCalculator service created
- SubscriptionManagementService implemented
- All subscription operations (upgrade, downgrade, cancel, pause, resume)
- Scheduled changes processor
- Dunning system integration
- Email notifications for all changes

#### 🔄 Phase 4.4: Tax Service
- Tax calculations table - TODO at `/libs/go/services/tax_service.go:263`
- Jurisdiction UUID conversion - TODO at `/libs/go/services/invoice_service.go:189-190`

#### 🔄 Phase 4.3: Exchange Rate Service
- Database implementation - TODO at `/libs/go/services/exchange_rate_service.go:245,271`

#### 🔄 Phase 4.5: Discount Service
- No specific TODOs found in code

#### Phase 4.1: Enhanced Analytics (Not started)
- Analytics refresh job - TODO at `/libs/go/services/analytics_service.go:643`
- Workspace statistics - TODO at `/libs/go/services/workspace_service.go:290`
- Dashboard metrics - TODO at `/libs/go/services/dashboard_metrics_service.go:209`

#### Phase 4.2: API Completeness (Not started)
- Multiple TODOs in `/apps/api/server/server.go` for implementing API methods

---

## TODO Priority Matrix

### Immediate Action Required (Next Sprint)
1. **Environment Configuration** - Centralize baseURL configuration
2. **Dunning Scheduled Job** - Move from manual trigger to scheduled Lambda
3. **Payment Intent Storage** - Complete payment page implementation
4. **Workspace Branding** - Add logo and color fields to workspace

### Short Term (1-2 Sprints)
1. **Tax Service Implementation** - Complete Phase 4.4
2. **Exchange Rate Service** - Complete Phase 4.3
3. **API Route Implementations** - Complete missing endpoints
4. **Email Sending Logic** - Invoice and payment link emails

### Medium Term (3-4 Sprints)
1. **Analytics Implementation** - Phase 4.1
2. **Gas Sponsorship Analytics** - Complete queries
3. **Dunning Retry Payment** - Delegation server integration
4. **Frontend Customer Portal** - Settings page functionality

### Long Term
1. **Blockchain Service Enhancements**
2. **Additional Webhook Events**
3. **Advanced Analytics Features**
4. **Multi-network Support Refactor**

---

## Notes

1. **Email Notifications**: ✅ Completed for all subscription changes in this session
2. **Dunning Integration**: ✅ Completed automatic cancellation after failed attempts
3. **Scheduled Changes**: ✅ Integrated into subscription processor Lambda
4. **API Security**: ✅ Added @Security annotations to all subscription management endpoints

### Recently Completed (This Session)
- Phase 4.6: Subscription Management - All core features
- Email notifications for subscription changes
- Dunning system integration for automatic actions
- Swagger documentation updates with security annotations

### Patterns Observed
- Many TODOs relate to configuration management (baseURL, environment variables)
- Payment processing integration points need completion
- Analytics and metrics are lower priority but widespread
- Frontend TODOs are mostly in customer-facing features

---

## Maintenance Notes

This document should be updated:
- When TODOs are completed
- When new TODOs are added
- After each major feature implementation
- During sprint planning to prioritize work

Use `grep -r "TODO" --include="*.go" --include="*.ts" --include="*.tsx" .` to find all TODOs in the codebase.