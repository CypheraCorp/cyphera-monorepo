# Invoice System Implementation Plan

## Overview
This document outlines the phased implementation plan for the Cyphera invoice system, supporting subscription invoices, one-time payments, and manual invoicing for contractors/businesses.

## Current Status (Completed)
- ✅ Database schema updates (added currency to subscriptions table)
- ✅ SQL queries for subscription-based invoice generation
- ✅ Invoice service methods:
  - GenerateInvoiceFromSubscription
  - Enhanced CreateInvoice (supports subscription line items)
  - VoidInvoice
  - MarkInvoicePaid
  - CreateInvoiceLineItemsFromSubscription
  - GeneratePendingInvoices (batch processing)
- ✅ Updated InvoiceCreateParams with Status, PeriodStart, PeriodEnd fields

## Invoice States
Our system uses these core invoice states:
1. **draft** - Invoice created but not sent (editable)
2. **open** - Invoice finalized and sent (cannot be edited, only voided)
3. **paid** - Invoice successfully paid
4. **void** - Invoice cancelled
5. **uncollectible** - Written off as bad debt

Additional derived states for UI:
- **overdue** - Open invoice past due date
- **processing** - Payment being processed
- **partially_paid** - For future partial payment support

## Implementation Phases

### Phase 1: Complete Backend Invoice Infrastructure (Current Phase)
**Goal: Full backend support for all invoice operations**

#### 1.1 Add Missing API Handlers
- [ ] POST /api/v1/invoices/:id/void - Call VoidInvoice service method
- [ ] POST /api/v1/invoices/:id/mark-paid - Call MarkInvoicePaid service method
- [ ] POST /api/v1/invoices/:id/mark-uncollectible - New service method needed
- [ ] POST /api/v1/invoices/:id/duplicate - New service method needed
- [ ] GET /api/v1/invoices/:id/activity - Audit trail (new table needed)
- [ ] POST /api/v1/invoices/:id/send-reminder - Send payment reminder
- [ ] POST /api/v1/invoices/bulk-generate - Generate subscription invoices
- [ ] GET /api/v1/invoices/stats - Invoice statistics

#### 1.2 Add Invoice Service Methods
```go
// New methods needed:
- MarkInvoiceUncollectible(ctx, workspaceID, invoiceID uuid.UUID) (*db.Invoice, error)
- DuplicateInvoice(ctx, workspaceID, invoiceID uuid.UUID) (*responses.InvoiceResponse, error)
- GetInvoiceActivity(ctx, workspaceID, invoiceID uuid.UUID) ([]InvoiceActivity, error)
- SendInvoiceReminder(ctx, invoiceID uuid.UUID) error
- GetInvoiceStats(ctx, workspaceID uuid.UUID, dateRange DateRange) (*InvoiceStats, error)
```

#### 1.3 Database Enhancements
```sql
-- Invoice activities table for audit trail
CREATE TABLE invoice_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    activity_type VARCHAR(50) NOT NULL, -- 'created', 'status_changed', 'sent', 'viewed', 'paid', etc.
    from_status VARCHAR(50),
    to_status VARCHAR(50),
    performed_by UUID REFERENCES users(id),
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add to invoices table
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS reminder_sent_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS reminder_count INTEGER DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS terms TEXT;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS footer TEXT;
```

#### 1.4 Enhanced Request/Response Types
```go
// Update CreateInvoiceRequest
type CreateInvoiceRequest struct {
    // Existing fields...
    
    // New fields
    InvoiceType    string  // "subscription", "one_time", "manual"
    Notes          *string // Customer-visible notes
    Terms          *string // Payment terms
    Footer         *string // Invoice footer text
}

// New types
type InvoiceActivity struct {
    ID           uuid.UUID
    InvoiceID    uuid.UUID
    ActivityType string
    FromStatus   *string
    ToStatus     *string
    PerformedBy  *uuid.UUID
    Description  string
    Metadata     map[string]interface{}
    CreatedAt    time.Time
}

type InvoiceStats struct {
    TotalInvoices     int
    TotalAmount       int64
    PaidAmount        int64
    OverdueAmount     int64
    AveragePaymentTime int // in days
    ByStatus          map[string]int
}
```

### Phase 2: Automated Invoice Processing
**Goal: Fully automated subscription billing**

1. **Background Jobs**
   - Invoice generation cron job (daily)
   - Overdue invoice reminder job (daily)
   - Failed payment retry logic
   - Invoice status sync job

2. **Event System**
   - Invoice created event
   - Invoice status changed event
   - Payment received event
   - Invoice overdue event

3. **Business Logic**
   - Late fee calculation
   - Grace period handling
   - Dunning management (payment retry strategy)

### Phase 3: API Enhancement & Testing
**Goal: Production-ready API with comprehensive testing**

1. **API Enhancements**
   - Advanced filtering (date ranges, amounts, multiple statuses)
   - Sorting options (date, amount, customer, status)
   - Bulk operations (void multiple, send reminders)
   - Webhook endpoints for payment providers

2. **Testing**
   - Unit tests for all service methods
   - Integration tests for invoice workflows
   - Load testing for batch operations
   - End-to-end subscription billing tests

3. **Documentation**
   - OpenAPI/Swagger documentation
   - Webhook payload schemas
   - Error code reference
   - Integration guides

### Phase 4: Basic Frontend Implementation
**Goal: Functional invoice management UI**

1. **Invoice List View**
   ```typescript
   // Core components needed:
   - InvoiceTable with status badges
   - Filter sidebar (status, date, customer)
   - Bulk action toolbar
   - Quick search
   ```

2. **Invoice Creation**
   ```typescript
   // Three creation modes:
   - From subscription (auto-populated)
   - Quick invoice (simple form)
   - Custom invoice (full builder)
   ```

3. **Invoice Details**
   ```typescript
   // Detail view features:
   - Status management buttons
   - Line items display
   - Activity timeline
   - Action history
   ```

### Phase 5: Advanced Frontend Features
**Goal: Enhanced user experience**

1. **Enhanced Creation Flow**
   - Multi-step wizard
   - Template library
   - Live preview
   - Draft auto-save

2. **Customer Portal**
   - Public invoice view
   - Payment page
   - Receipt download
   - Payment history

3. **Dashboard & Analytics**
   - Invoice metrics
   - Aging reports
   - Collection insights
   - Revenue forecasting

### Phase 6: Polish & Optimization
**Goal: Production-ready system**

1. **Performance**
   - Query optimization
   - Redis caching
   - Pagination optimization
   - Lazy loading

2. **User Experience**
   - Mobile responsive
   - Keyboard shortcuts
   - Real-time updates
   - Offline support

3. **Operations**
   - Monitoring setup
   - Admin tools
   - Bulk import/export
   - Audit logging

## API Endpoint Summary

### Existing Endpoints
- POST /api/v1/invoices - Create invoice
- GET /api/v1/invoices/:id - Get invoice details
- GET /api/v1/invoices - List invoices
- POST /api/v1/invoices/:id/finalize - Finalize draft
- POST /api/v1/invoices/:id/send - Send invoice (stub)
- GET /api/v1/invoices/:id/payment-link - Get payment link

### New Endpoints (Phase 1)
- POST /api/v1/invoices/:id/void - Void invoice
- POST /api/v1/invoices/:id/mark-paid - Mark as paid
- POST /api/v1/invoices/:id/mark-uncollectible - Mark as uncollectible
- POST /api/v1/invoices/:id/duplicate - Duplicate invoice
- GET /api/v1/invoices/:id/activity - Get activity log
- POST /api/v1/invoices/:id/send-reminder - Send reminder
- POST /api/v1/invoices/bulk-generate - Bulk generate
- GET /api/v1/invoices/stats - Get statistics

## Frontend Component Structure
```
/components/invoices/
├── InvoiceList/
│   ├── InvoiceTable.tsx
│   ├── InvoiceFilters.tsx
│   ├── InvoiceActions.tsx
│   └── InvoiceBulkActions.tsx
├── InvoiceCreate/
│   ├── InvoiceTypeSelector.tsx
│   ├── SubscriptionInvoiceForm.tsx
│   ├── QuickInvoiceForm.tsx
│   ├── CustomInvoiceBuilder.tsx
│   └── InvoicePreview.tsx
├── InvoiceDetail/
│   ├── InvoiceHeader.tsx
│   ├── InvoiceLineItems.tsx
│   ├── InvoiceActions.tsx
│   ├── InvoiceTimeline.tsx
│   └── InvoiceNotes.tsx
├── CustomerPortal/
│   ├── InvoicePaymentPage.tsx
│   ├── PaymentMethodSelector.tsx
│   └── PaymentConfirmation.tsx
└── shared/
    ├── InvoiceStatusBadge.tsx
    ├── InvoiceCalculations.tsx
    └── InvoiceCurrencyDisplay.tsx
```

## Key Design Decisions

1. **Invoice Types**: Support three types - subscription (auto-generated), one_time (quick charges), manual (custom)
2. **Status Simplification**: Use 5 core states, derive others for UI
3. **Audit Trail**: Track all invoice changes in invoice_activities table
4. **Bulk Operations**: Essential for subscription businesses
5. **Customer Portal**: Separate, simplified view for customers
6. **No Proforma**: Skip proforma invoices initially for simplicity

## Success Metrics
- Invoice generation time < 100ms
- Bulk generation handling 1000+ invoices
- Payment success rate > 95%
- Customer portal load time < 2s
- Zero data loss on status transitions

## Notes
- Priority is backend stability before frontend polish
- Each phase should be fully tested before moving on
- Customer experience is paramount for payment pages
- Consider accessibility from the start