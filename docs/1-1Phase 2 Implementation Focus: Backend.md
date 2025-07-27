  ⎿ Phase 2: Backend Business Logic 
    Implementation Plan

    Current State Analysis ✅

    Phase 1 Completed:
    - ✅ Database schema: All core tables 
    implemented (payments, invoice_line_items, 
    dashboard_metrics, payment_links, 
    gas_fee_payments, etc.)
    - ✅ Analytics service: Fully implemented 
    with all endpoints working
    - ✅ Analytics handler: Complete with proper 
    routes
    - ✅ Basic services: Payment, gas 
    sponsorship, dashboard metrics services exist

    Phase 2 Implementation Focus: Backend 
    Business Logic & Services

    2.1 Enhanced Database Services Layer (Week 1)

    Priority: Critical TODOs from Phase 1

    1. Payment Helper Enhancements
      - Implement exchange rate integration with 
    oracle service
      - Add actual gas fee calculation from 
    blockchain transactions
      - Build gas sponsorship checking logic
      - Create comprehensive tax calculation 
    engine
      - Add discount application logic for 
    subscription discounts
      - Complete gas_fee_payment record creation
    2. Enhanced Currency Service
      - Complete migration from hardcoded 
    constants to database-driven currencies
      - Implement proper currency formatting with
     locale support
      - Add exchange rate caching and fallback 
    mechanisms
      - Build currency conversion utilities for 
    multi-currency support
    3. Tax Calculation Service
      - Implement jurisdiction detection logic
      - Build tax rate lookup and calculation 
    engine
      - Add B2B tax handling with reverse charge 
    logic
      - Create tax audit trail with 
    tax_calculations table
      - Support multiple tax jurisdictions per 
    transaction

    2.2 Advanced Invoice Management System (Week 
    2)

    Invoice Builder Service
    - Complete line item management with 
    product/gas/tax separation
    - Automated tax calculation with jurisdiction
     rules
    - Gas fee line items with sponsorship support
    - PDF generation with complete breakdowns
    - Invoice numbering and sequence management
    - Multi-currency invoice support with proper 
    formatting

    Invoice API Enhancements
    - POST /api/v1/invoices - Enhanced invoice 
    creation
    - GET /api/v1/invoices/:id/preview - Invoice 
    preview with all line items
    - POST /api/v1/invoices/:id/finalize - 
    Invoice finalization logic
    - POST /api/v1/invoices/:id/send - Email 
    delivery system
    - GET /api/v1/invoices/:id/payment-link - QR 
    code generation

    2.3 B2B Payment Flow Infrastructure (Week 3)

    Payment Link System
    - Enhanced payment link generation with 
    expiration handling
    - QR code creation with delegation parameters
    - Multi-token support for payment links
    - Customer payment experience optimization
    - Payment confirmation and webhook handling

    Delegation Integration
    - B2B payment flow with delegation requests
    - Customer payment page backend logic
    - Payment processing with gas sponsorship
    - Success/failure handling and notifications
    - Integration with existing subscription 
    system

    2.4 Advanced Subscription System (Week 4)

    Subscription Line Items Implementation
    - Refactor subscription model to support 
    multiple products per subscription
    - Implement subscription_line_items table 
    usage
    - Add support for add-ons, bundles, and mixed
     billing models
    - Build subscription change tracking 
    (upgrades/downgrades)
    - Proration calculation for mid-cycle changes

    Discount & Trial System
    - Complete discount code validation and 
    application
    - Trial period management with proper billing
     cycle handling
    - Usage tracking for discount codes
    - Customer eligibility checking
    - Integration with subscription billing

    2.5 Financial Accuracy & Compliance (Week 5)

    Gas Fee Management
    - Comprehensive gas fee tracking across all 
    transactions
    - Gas sponsorship budget management and 
    threshold checking
    - Gas fee analytics and cost optimization 
    recommendations
    - Network-specific gas handling (Ethereum, 
    Polygon, etc.)
    - Real-time gas price monitoring integration

    Financial Reconciliation
    - Payment to subscription event mapping
    - Revenue recognition compliance
    - Multi-currency financial reporting
    - Audit trail maintenance
    - Discrepancy detection and resolution

    2.6 Enhanced Analytics & Metrics (Week 6)

    Real-time Metrics Engine
    - Complete dashboard_metrics calculation with
     all fields
    - Expansion and contraction revenue tracking
    - Customer cohort analysis
    - Churn prediction algorithms
    - Revenue forecasting models

    Advanced Analytics APIs
    - Cohort analysis endpoints
    - Revenue breakdown by 
    product/customer/region
    - Gas cost analysis and optimization insights
    - Tax reporting and compliance metrics
    - Subscription metrics with line item 
    granularity

    Implementation Architecture

    Service Layer Structure

    libs/go/services/
    ├── invoice_service.go          # Complete 
    invoice management
    ├── payment_link_service.go     # Payment 
    link and QR generation
    ├── tax_service.go             # Tax 
    calculation and compliance
    ├── discount_service.go        # Discount and
     trial management
    ├── subscription_line_item_service.go  # Line
     item management
    ├── financial_reconciliation_service.go # 
    Financial accuracy
    └── advanced_analytics_service.go      # 
    Enhanced analytics

    Handler Enhancements

    apps/api/handlers/
    ├── invoice_handler.go         # Enhanced 
    invoice APIs
    ├── payment_link_handler.go    # Payment link
     management
    ├── b2b_payment_handler.go     # B2B payment 
    flows
    ├── discount_handler.go        # Discount 
    management
    └── financial_handler.go       # Financial 
    reporting

    Critical Database Enhancements

    - Complete all TODOs in payment_helper.go
    - Implement missing SQLC queries for new 
    tables
    - Add proper indexes for performance 
    optimization
    - Create materialized views for complex 
    analytics
    - Add database triggers for real-time metric 
    updates

    Success Criteria

    Technical Metrics:
    - All Phase 1 TODOs resolved ✅
    - Invoice creation time < 2 seconds ✅
    - Payment processing success rate > 99% ✅
    - Dashboard metrics updated in real-time ✅
    - Zero financial discrepancies ✅

    Business Logic Completeness:
    - Complete B2B payment flow from invoice to 
    payment ✅
    - Multi-product subscriptions working ✅
    - Tax calculation accurate for all 
    jurisdictions ✅
    - Gas sponsorship system operational ✅
    - Discount system fully functional ✅

    API Completeness:
    - All invoice management endpoints 
    operational ✅
    - Payment link system fully functional ✅
    - Advanced analytics endpoints complete ✅
    - Multi-currency support throughout ✅
    - Comprehensive error handling and validation
     ✅

    Dependencies & Integration Points

    External Services:
    - Exchange rate oracle integration 
    (CoinMarketCap/other)
    - Blockchain RPC integration for gas fee 
    calculation  
    - Email service for invoice delivery
    - PDF generation service for invoice 
    rendering

    Internal Integration:
    - Delegation server for B2B payments
    - Existing analytics system enhancement
    - Payment sync service updates
    - Subscription processor modifications

    This phase will complete the backend 
    foundation, making the system ready for 
    advanced features and frontend integration in
     future phases.

    Phase 4: Dunning Management System Implementation

    COMPLETED Components:

    4.1 Database Infrastructure ✅
    - Created 5 new tables for dunning management:
      - dunning_configurations: Stores retry schedules and settings per workspace
      - dunning_campaigns: Tracks individual failed payment recovery attempts
      - dunning_attempts: Records each retry attempt with outcomes
      - dunning_email_templates: Customizable email templates for each retry stage
      - dunning_analytics: Aggregated metrics for recovery performance
    - Implemented comprehensive SQLC queries for all operations
    - Added proper indexes and foreign key constraints

    4.2 Core Services ✅
    - DunningService: Manages configurations, campaigns, templates, and analytics
    - DunningRetryEngine: Processes due campaigns and sends notification emails
    - EmailService: Resend integration for transactional email delivery
    - Integrated with existing payment and subscription services

    4.3 API Endpoints ✅
    - POST /api/v1/dunning/configurations - Create retry configurations
    - GET /api/v1/dunning/configurations - List workspace configurations
    - POST /api/v1/dunning/campaigns - Create recovery campaigns
    - GET /api/v1/dunning/campaigns - List active campaigns
    - POST /api/v1/dunning/email-templates - Create email templates
    - POST /api/v1/dunning/process - Manual trigger for testing

    4.4 Email Integration ✅
    - Fully integrated Resend for email delivery
    - Go template-based email rendering with variable substitution
    - HTML and plain text email support
    - Tracking of email delivery status

    REMAINING TODOs (Detailed Implementation Requirements):

    4.5 Payment Retry Integration with Delegation Server
    **Description**: Integrate the dunning retry engine with the delegation server to automatically retry failed blockchain payments using stored delegation credentials.
    
    **Technical Requirements**:
    - Modify DunningRetryEngine.ProcessDueCampaigns to include payment retry logic
    - Create new gRPC method in delegation server: RetrySubscriptionPayment
    - Store and retrieve encrypted delegation credentials for retry attempts
    - Handle gas fee calculation and sponsorship for retry transactions
    - Update dunning_attempts table with blockchain transaction details
    - Implement retry logic with exponential backoff for blockchain congestion
    - Add circuit breaker pattern to prevent excessive blockchain calls
    
    **Integration Points**:
    - libs/go/services/dunning_retry_engine.go - Add payment retry logic
    - apps/delegation-server/proto/delegation.proto - Define RetrySubscriptionPayment RPC
    - libs/go/client/delegation/client.go - Add retry payment method
    - libs/go/services/payment_service.go - Expose retry payment functionality
    
    **Success Criteria**:
    - Automatic payment retries execute according to configured schedule
    - Blockchain transaction receipts stored in dunning_attempts
    - Failed retries properly logged with specific error reasons
    - Gas sponsorship correctly applied to retry attempts

    4.6 Comprehensive Subscription Management System ✅
    **Description**: Implement a full subscription lifecycle management system supporting upgrades, downgrades, cancellations, pauses, and resumptions with proper proration logic and billing cycle management. This includes both manual customer-initiated changes and automated dunning-triggered actions.
    
    **Technical Requirements**:
    - Create subscription_schedule_changes table for tracking all pending changes
    - Create subscription_prorations table for detailed proration calculations
    - Create subscription_state_history table for complete audit trail
    - Implement ProrationCalculator service for accurate billing calculations
    - Build SubscriptionManagementService with methods for all operations:
      - UpgradeSubscription (immediate with proration)
      - DowngradeSubscription (scheduled for end of period)
      - CancelSubscription (scheduled for end of period)
      - PauseSubscription (immediate or scheduled)
      - ResumeSubscription (starts new billing cycle)
    - Add scheduled change processing for background execution
    - Implement preview endpoints for cost transparency
    - Add dunning integration for automated actions
    
    **Billing Logic**:
    - Upgrades: Immediate effect with prorated credit for unused time
    - Downgrades: Take effect at end of current period (no proration)
    - Cancellations: Take effect at end of current period (no refund)
    - Pauses: Can be immediate or scheduled
    - Resumes: Start new billing cycle from resume date
    
    **Database Schema**:
    ```sql
    -- Schedule changes tracking
    CREATE TABLE subscription_schedule_changes (
        id UUID PRIMARY KEY,
        subscription_id UUID NOT NULL,
        change_type VARCHAR(50) NOT NULL,
        scheduled_for TIMESTAMPTZ NOT NULL,
        from_line_items JSONB,
        to_line_items JSONB,
        proration_amount_cents BIGINT,
        proration_calculation JSONB,
        status VARCHAR(20) DEFAULT 'scheduled',
        initiated_by VARCHAR(50),
        reason TEXT
    );
    
    -- Proration records
    CREATE TABLE subscription_prorations (
        id UUID PRIMARY KEY,
        subscription_id UUID NOT NULL,
        proration_type VARCHAR(50),
        period_start TIMESTAMPTZ,
        period_end TIMESTAMPTZ,
        days_total INTEGER,
        days_used INTEGER,
        credit_amount_cents BIGINT
    );
    ```
    
    **API Endpoints**:
    - PUT /api/v1/subscriptions/:id/upgrade
    - PUT /api/v1/subscriptions/:id/downgrade
    - DELETE /api/v1/subscriptions/:id (cancel)
    - POST /api/v1/subscriptions/:id/pause
    - POST /api/v1/subscriptions/:id/resume
    - POST /api/v1/subscriptions/:id/preview-change
    - GET /api/v1/subscriptions/:id/scheduled-changes
    - DELETE /api/v1/subscriptions/:id/scheduled-changes/:changeId
    
    **Integration Points**:
    - libs/go/services/subscription_management_service.go - Core management logic
    - libs/go/services/proration_calculator.go - Proration calculations
    - libs/go/db/queries/subscription_changes.sql - Change tracking queries
    - apps/api/handlers/subscription_management_handler.go - API endpoints
    - apps/subscription-processor/internal/processor/scheduled_changes.go - Background processing
    - libs/go/services/dunning_action_service.go - Dunning integration
    
    **Success Criteria**:
    - All subscription changes handle billing correctly
    - Proration calculations accurate to the day
    - No double-billing or unfair charges
    - Clear preview of costs before confirmation
    - Scheduled changes execute reliably
    - Complete audit trail of all changes
    - Email notifications for all actions
    - Dunning can trigger automated actions
    - Customers can reactivate before cancellation
    - Revenue protected on downgrades

    4.7 Webhook Notifications for Dunning Events
    **Description**: Implement a comprehensive webhook system to notify merchants about dunning events in real-time, allowing them to integrate with their own systems and take custom actions.
    
    **Technical Requirements**:
    - Create webhook_endpoints table with URL, events, and authentication
    - Define dunning webhook event types:
      - dunning.campaign.created
      - dunning.attempt.initiated
      - dunning.attempt.succeeded
      - dunning.attempt.failed
      - dunning.action.executed
      - dunning.campaign.completed
    - Implement webhook delivery service with retry logic
    - Add webhook signing for security (HMAC-SHA256)
    - Create webhook event queue for reliable delivery
    - Implement webhook testing endpoint for merchants
    - Add webhook delivery logs and analytics
    
    **Integration Points**:
    - libs/go/services/webhook_service.go - New webhook delivery service
    - libs/go/db/queries/webhooks.sql - Webhook configuration queries
    - apps/api/handlers/webhook_handler.go - Webhook management endpoints
    - libs/go/middleware/webhook_auth.go - Webhook signature verification
    
    **Success Criteria**:
    - All dunning events trigger appropriate webhooks
    - Failed deliveries retry with exponential backoff
    - Webhook signatures validate correctly
    - Merchants can configure event subscriptions
    - Complete delivery logs available for debugging

    4.8 Automated Campaign Creation on Payment Failure ✅
    **Description**: Automatically detect failed recurring payments and create dunning campaigns without manual intervention, ensuring no failed payment goes unaddressed.
    
    **Technical Requirements**:
    - Modify subscription processor to detect payment failures
    - Create PaymentFailureDetector service
    - Implement logic to check for existing active campaigns before creating new ones
    - Add configuration for auto-campaign creation rules per workspace
    - Support different campaign strategies based on:
      - Customer payment history
      - Subscription value
      - Previous dunning success rate
    - Implement campaign deduplication to prevent multiple campaigns
    - Add support for immediate vs delayed campaign start
    
    **Integration Points**:
    - apps/subscription-processor/internal/processor/processor.go - Add failure detection
    - libs/go/services/payment_failure_detector.go - New detection service
    - libs/go/services/dunning_service.go - Add auto-creation methods
    - libs/go/db/queries/payment_sync_events.sql - Query failed payments
    
    **Success Criteria**:
    - Failed payments automatically trigger campaign creation
    - No duplicate campaigns for same failed payment
    - Configurable rules apply correctly
    - Campaign creation happens within 5 minutes of failure
    - Proper error handling for edge cases

    4.9 Background Job Scheduler for Retry Engine (AWS Lambda) ✅
    **Description**: Implement a robust background job scheduling system as an AWS Lambda function to run the dunning retry engine at regular intervals, ensuring timely processing of due campaigns. This will follow the same pattern as the existing subscription-processor Lambda.
    
    **Technical Requirements**:
    - Create new Lambda function: apps/dunning-processor/cmd/main.go
    - Implement AWS EventBridge (CloudWatch Events) scheduled trigger
    - Use AWS SAM template for deployment (template-dunning.yaml)
    - Implement distributed locking using DynamoDB or database flags
    - Add Lambda context handling for timeout management
    - Configure environment-based scheduling (dev: 5 min, prod: 1 min)
    - Implement idempotency to handle Lambda retry behavior
    - Add CloudWatch metrics for monitoring
    - Use existing AWS Secrets Manager for configuration
    - Implement graceful shutdown before Lambda timeout
    
    **Lambda Structure**:
    ```
    apps/dunning-processor/
    ├── cmd/
    │   └── main.go              # Lambda handler entry point
    ├── internal/
    │   ├── processor/
    │   │   └── processor.go     # Core dunning processing logic
    │   └── config/
    │       └── config.go        # Lambda configuration
    ├── Makefile                 # Build commands
    └── template-dunning.yaml    # SAM deployment template
    ```
    
    **Integration Points**:
    - apps/dunning-processor/cmd/main.go - Lambda handler implementation
    - libs/go/services/dunning_retry_engine.go - Reuse existing retry engine
    - template-dunning.yaml - SAM template for Lambda deployment
    - Makefile - Add build-dunning-processor target
    - .github/workflows/deploy.yml - Add dunning processor to CI/CD
    
    **SAM Template Configuration**:
    - Runtime: provided.al2 (Go custom runtime)
    - Memory: 512 MB (adjustable based on load)
    - Timeout: 300 seconds (5 minutes)
    - EventBridge Rule: rate(1 minute) for production
    - Environment variables from AWS Secrets Manager
    - VPC configuration for database access
    - IAM roles for Secrets Manager and CloudWatch
    
    **Success Criteria**:
    - Lambda executes reliably on EventBridge schedule
    - No duplicate processing through distributed locking
    - Complete CloudWatch logs for debugging
    - Metrics visible in CloudWatch dashboard
    - Graceful handling of Lambda timeouts
    - Zero message loss during processing
    - Deployment automated through GitHub Actions

    4.10 Dunning Analytics and Reporting Dashboard
    **Description**: Build comprehensive analytics and reporting capabilities for dunning performance, providing insights into recovery rates, optimal retry schedules, and campaign effectiveness.
    
    **Technical Requirements**:
    - Enhance dunning_analytics table with additional metrics:
      - Recovery rate by attempt number
      - Average time to recovery
      - Revenue recovered by campaign type
      - Customer segment performance
    - Create materialized views for complex calculations
    - Implement real-time analytics updates
    - Build analytics API endpoints:
      - GET /api/v1/dunning/analytics/overview
      - GET /api/v1/dunning/analytics/campaigns/:id
      - GET /api/v1/dunning/analytics/trends
      - GET /api/v1/dunning/analytics/optimization-suggestions
    - Add export functionality (CSV, JSON)
    - Implement analytics caching for performance
    
    **Integration Points**:
    - libs/go/services/dunning_analytics_service.go - New analytics service
    - libs/go/db/queries/dunning_analytics_advanced.sql - Complex queries
    - apps/api/handlers/dunning_analytics_handler.go - Analytics endpoints
    - libs/go/helpers/analytics_calculator.go - Calculation helpers
    
    **Success Criteria**:
    - Real-time analytics update within 1 minute
    - Historical trend analysis available
    - Actionable optimization suggestions generated
    - Export functionality works for large datasets
    - Performance remains fast with millions of records

    4.11 Multi-Channel Notification Support
    **Description**: Extend the dunning notification system beyond email to support SMS, in-app notifications, and push notifications for better customer reach.
    
    **Technical Requirements**:
    - Abstract notification delivery into channel-agnostic interface
    - Implement SMS integration (Twilio/similar)
    - Add in-app notification system
    - Implement push notification support (FCM/APNS)
    - Create channel preference management per customer
    - Add channel fallback logic (email → SMS → push)
    - Implement notification templates for each channel
    - Add delivery tracking across all channels
    
    **Integration Points**:
    - libs/go/services/notification_service.go - Multi-channel orchestrator
    - libs/go/services/sms_service.go - SMS delivery implementation
    - libs/go/services/push_service.go - Push notification service
    - libs/go/db/queries/notification_preferences.sql - Channel preferences
    
    **Success Criteria**:
    - All channels deliver notifications successfully
    - Fallback logic works when primary channel fails
    - Customer preferences respected
    - Unified delivery tracking across channels
    - Template rendering works for all channels

    Implementation Priority Order:
    1. Payment Retry Integration (Critical for core functionality)
    2. Automated Campaign Creation (Ensures no missed failures)
    3. Background Job Scheduler (Enables automatic processing)
    4. Webhook Notifications (Merchant integration capability)
    5. Subscription Management Actions (Enhanced dunning actions)
    6. Analytics and Reporting (Business insights)
    7. Multi-Channel Notifications (Enhanced customer reach)

    Each TODO should be implemented with:
    - Comprehensive error handling and logging
    - Unit and integration tests
    - API documentation updates
    - Performance optimization
    - Security best practices
    - Backwards compatibility