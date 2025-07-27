# Backend Remaining Tasks Before Frontend Development

## Overview
This document outlines all remaining backend tasks that need to be completed before moving to frontend development. Tasks are prioritized based on their impact on core functionality.

## Completed Backend Components ✅

### Phase 1-3: Core Infrastructure
- Database schema with all tables
- Analytics service and endpoints  
- Payment, subscription, and dashboard services
- Invoice management system
- B2B payment flows
- Multi-currency support
- Tax calculation engine
- Discount and trial system

### Phase 4: Dunning Management (Partially Complete)
- Database infrastructure (5 tables)
- Core dunning services
- API endpoints for configuration
- Email integration with Resend
- Automated campaign creation on payment failure ✅
- Background job scheduler (AWS Lambda) ✅

## Remaining Backend Tasks (Priority Order)

### 1. Payment Retry Integration with Delegation Server 🔴 HIGH PRIORITY
**Why Critical**: Core functionality - without this, dunning campaigns can only send emails but cannot actually retry failed payments.

**Implementation Requirements**:
- Create `RetrySubscriptionPayment` gRPC method in delegation server
- Integrate dunning retry engine with blockchain payment retries
- Handle gas fee calculation and sponsorship
- Store transaction receipts in dunning_attempts
- Add circuit breaker for blockchain calls

**Estimated Time**: 3-4 days

### 2. Webhook Notifications for Dunning Events 🟡 MEDIUM PRIORITY
**Why Important**: Allows merchants to integrate dunning events with their systems and take custom actions.

**Implementation Requirements**:
- Create webhook_endpoints table
- Define 6 dunning webhook event types
- Implement webhook delivery service with retries
- Add HMAC-SHA256 signing
- Create webhook management endpoints

**Estimated Time**: 2-3 days

### 3. Subscription Management Actions (Pause/Cancel/Downgrade) 🟡 MEDIUM PRIORITY
**Why Important**: Provides automated actions based on failed payment attempts, preventing revenue loss.

**Implementation Requirements**:
- Add action configuration to dunning_configurations
- Create DunningActionService
- Implement pause, cancel, and downgrade logic
- Add grace period handling
- Create action audit logs

**Estimated Time**: 2-3 days

### 4. Dunning Analytics and Reporting 🟢 LOW PRIORITY
**Why Lower Priority**: Can be built incrementally after core functionality is working.

**Implementation Requirements**:
- Enhance analytics table with metrics
- Create materialized views
- Build analytics API endpoints
- Add export functionality
- Implement caching layer

**Estimated Time**: 2 days

### 5. Multi-Channel Notification Support 🟢 LOW PRIORITY
**Why Lower Priority**: Email notifications are sufficient for MVP; other channels can be added later.

**Implementation Requirements**:
- Abstract notification interface
- Add SMS integration (Twilio)
- Implement push notifications
- Channel preference management
- Fallback logic between channels

**Estimated Time**: 3-4 days

## Summary

### Must Complete Before Frontend (7-10 days total):
1. **Payment Retry Integration** - Without this, the dunning system cannot actually recover payments
2. **Webhook Notifications** - Critical for merchant integrations and monitoring
3. **Subscription Management Actions** - Essential for automated subscription lifecycle management

### Can Be Done In Parallel with Frontend (5-7 days):
4. **Analytics and Reporting** - Can start with basic metrics and enhance over time
5. **Multi-Channel Notifications** - Email is sufficient for initial release

## Recommended Approach

1. **Week 1**: Focus on Payment Retry Integration and Webhook Notifications
   - These are the most critical for a functional dunning system
   - Can be worked on in parallel by different developers

2. **Week 2**: Complete Subscription Management Actions
   - While starting frontend development in parallel
   - Frontend can begin with existing APIs

3. **Post-MVP**: Analytics and Multi-Channel Support
   - These enhance the system but aren't blockers for initial release

## Testing Requirements

Each remaining component needs:
- Unit tests with mocked dependencies
- Integration tests with test database
- End-to-end tests for critical flows
- Performance testing for high-volume scenarios
- Security testing for webhook signatures and payment retries

## Dependencies

- Delegation server must support payment retry operations
- AWS infrastructure must be configured for webhooks
- Resend API key must have appropriate permissions
- Test blockchain networks for retry testing