# Subscription Management System Design

## Overview

This document outlines a comprehensive subscription management system that handles the full lifecycle of subscriptions including upgrades, downgrades, cancellations, pauses, and resumptions with proper proration and billing cycle management.

## Core Principles

1. **Upgrades**: Take effect immediately with proration credit
2. **Downgrades**: Take effect at the end of current billing period
3. **Cancellations**: Take effect at the end of current billing period
4. **Pauses**: Can be immediate or scheduled
5. **Resumptions**: Reactivate with new billing cycle

## Database Schema Updates

### 1. Subscription Schedule Changes Table

```sql
CREATE TABLE subscription_schedule_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    
    -- Change details
    change_type VARCHAR(50) NOT NULL, -- 'upgrade', 'downgrade', 'cancel', 'pause', 'resume', 'modify_items'
    scheduled_for TIMESTAMPTZ NOT NULL, -- When the change takes effect
    
    -- For upgrades/downgrades
    from_line_items JSONB, -- Current line items snapshot
    to_line_items JSONB,   -- New line items after change
    
    -- Proration details
    proration_amount_cents BIGINT, -- Credit for unused time (negative for credit)
    proration_calculation JSONB,   -- Detailed calculation breakdown
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled', -- 'scheduled', 'processing', 'completed', 'cancelled'
    processed_at TIMESTAMPTZ,
    
    -- Reason and metadata
    reason TEXT,
    initiated_by VARCHAR(50), -- 'customer', 'admin', 'system', 'dunning'
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_schedule_changes_subscription ON subscription_schedule_changes(subscription_id, scheduled_for);
CREATE INDEX idx_schedule_changes_status ON subscription_schedule_changes(status, scheduled_for);
```

### 2. Subscription Proration Records

```sql
CREATE TABLE subscription_prorations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    schedule_change_id UUID REFERENCES subscription_schedule_changes(id),
    
    -- Proration details
    proration_type VARCHAR(50) NOT NULL, -- 'upgrade_credit', 'downgrade_adjustment', 'cancellation_credit'
    
    -- Time period
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    days_total INTEGER NOT NULL,
    days_used INTEGER NOT NULL,
    days_remaining INTEGER NOT NULL,
    
    -- Amounts
    original_amount_cents BIGINT NOT NULL, -- Full period amount
    used_amount_cents BIGINT NOT NULL,     -- Prorated used amount
    credit_amount_cents BIGINT NOT NULL,   -- Credit to apply
    
    -- Applied to
    applied_to_invoice_id UUID REFERENCES invoices(id),
    applied_to_payment_id UUID REFERENCES payments(id),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_prorations_subscription ON subscription_prorations(subscription_id);
```

### 3. Subscription States History

```sql
CREATE TABLE subscription_state_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id),
    
    -- State transition
    from_status subscription_status,
    to_status subscription_status NOT NULL,
    
    -- Billing changes
    from_amount_cents BIGINT,
    to_amount_cents BIGINT,
    
    -- Line items snapshot
    line_items_snapshot JSONB,
    
    -- Context
    change_reason TEXT,
    schedule_change_id UUID REFERENCES subscription_schedule_changes(id),
    
    occurred_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_state_history_subscription ON subscription_state_history(subscription_id, occurred_at DESC);
```

## Proration Calculation Logic

### Upgrade Proration (Immediate)

```go
type ProrationCalculator struct {
    calendar *Calendar // Handles date/time calculations
}

func (pc *ProrationCalculator) CalculateUpgradeProration(
    currentPeriodStart, currentPeriodEnd time.Time,
    oldAmountCents, newAmountCents int64,
    changeDate time.Time,
) *ProrationResult {
    // Calculate days in period
    totalDays := pc.calendar.DaysBetween(currentPeriodStart, currentPeriodEnd)
    usedDays := pc.calendar.DaysBetween(currentPeriodStart, changeDate)
    remainingDays := totalDays - usedDays
    
    // Calculate prorated amounts
    dailyRateOld := float64(oldAmountCents) / float64(totalDays)
    dailyRateNew := float64(newAmountCents) / float64(totalDays)
    
    // Credit for unused time at old rate
    unusedCredit := int64(dailyRateOld * float64(remainingDays))
    
    // Charge for remaining time at new rate
    newCharge := int64(dailyRateNew * float64(remainingDays))
    
    // Net amount to charge now
    immediateCharge := newCharge - unusedCredit
    
    return &ProrationResult{
        CreditAmount: unusedCredit,
        ChargeAmount: newCharge,
        NetAmount:    immediateCharge,
        Calculation: map[string]interface{}{
            "total_days":      totalDays,
            "used_days":       usedDays,
            "remaining_days":  remainingDays,
            "old_daily_rate":  dailyRateOld,
            "new_daily_rate":  dailyRateNew,
            "unused_credit":   unusedCredit,
            "new_charge":      newCharge,
        },
    }
}
```

### Downgrade Scheduling (End of Period)

```go
func (pc *ProrationCalculator) ScheduleDowngrade(
    subscription *Subscription,
    newLineItems []LineItem,
    requestedDate time.Time,
) *ScheduleChangeResult {
    // Downgrade happens at end of current period
    effectiveDate := subscription.CurrentPeriodEnd
    
    // No proration needed - customer keeps current service until period end
    return &ScheduleChangeResult{
        ScheduledFor: effectiveDate,
        ChangeType:   "downgrade",
        NoProration:  true,
        Message:      fmt.Sprintf("Downgrade scheduled for %s. You'll continue with current plan until then.", 
                      effectiveDate.Format("Jan 2, 2006")),
    }
}
```

## Subscription Management Service

### Core Operations

```go
type SubscriptionManagementService struct {
    db           *db.Queries
    calculator   *ProrationCalculator
    paymentSvc   *PaymentService
    emailSvc     *EmailService
    delegationSvc *DelegationService
}

// Upgrade Subscription (Immediate with proration)
func (sms *SubscriptionManagementService) UpgradeSubscription(
    ctx context.Context,
    subscriptionID uuid.UUID,
    newLineItems []LineItemUpdate,
    reason string,
) error {
    // Start transaction
    tx, err := sms.db.BeginTx(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Get current subscription
    sub, err := tx.GetSubscriptionWithLineItems(ctx, subscriptionID)
    if err != nil {
        return err
    }
    
    // Validate upgrade
    if err := sms.validateUpgrade(sub, newLineItems); err != nil {
        return err
    }
    
    // Calculate new total
    newTotal := sms.calculateNewTotal(newLineItems)
    oldTotal := sub.TotalAmountCents
    
    // Calculate proration
    proration := sms.calculator.CalculateUpgradeProration(
        sub.CurrentPeriodStart,
        sub.CurrentPeriodEnd,
        oldTotal,
        newTotal,
        time.Now(),
    )
    
    // Create schedule change record
    scheduleChange, err := tx.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
        SubscriptionID:        subscriptionID,
        ChangeType:           "upgrade",
        ScheduledFor:         time.Now(), // Immediate
        FromLineItems:        sub.CurrentLineItemsJSON(),
        ToLineItems:          newLineItemsJSON(newLineItems),
        ProrationAmountCents: proration.NetAmount,
        ProrationCalculation: proration.CalculationJSON(),
        Status:               "processing",
        Reason:               reason,
        InitiatedBy:         "customer",
    })
    if err != nil {
        return err
    }
    
    // Update line items immediately
    for _, item := range newLineItems {
        if item.Action == "add" {
            err = tx.AddSubscriptionLineItem(ctx, db.AddLineItemParams{
                SubscriptionID: subscriptionID,
                ProductID:      item.ProductID,
                Quantity:       item.Quantity,
                UnitAmount:     item.UnitAmount,
            })
        } else if item.Action == "update" {
            err = tx.UpdateLineItemQuantity(ctx, db.UpdateQuantityParams{
                LineItemID: item.LineItemID,
                Quantity:   item.Quantity,
            })
        } else if item.Action == "remove" {
            err = tx.RemoveLineItem(ctx, item.LineItemID)
        }
        
        if err != nil {
            return err
        }
    }
    
    // Update subscription total
    err = tx.UpdateSubscriptionTotal(ctx, db.UpdateTotalParams{
        SubscriptionID: subscriptionID,
        TotalAmount:    newTotal,
    })
    if err != nil {
        return err
    }
    
    // Process immediate payment if net charge
    if proration.NetAmount > 0 {
        payment, err := sms.paymentSvc.ProcessProrationPayment(ctx, tx, db.ProrationPaymentParams{
            SubscriptionID:    subscriptionID,
            Amount:           proration.NetAmount,
            Description:      "Subscription upgrade proration",
            ScheduleChangeID: scheduleChange.ID,
        })
        if err != nil {
            return fmt.Errorf("failed to process proration payment: %w", err)
        }
    }
    
    // Record state change
    err = tx.RecordStateChange(ctx, db.RecordStateChangeParams{
        SubscriptionID:   subscriptionID,
        FromStatus:       sub.Status,
        ToStatus:         sub.Status, // Status doesn't change on upgrade
        FromAmountCents:  oldTotal,
        ToAmountCents:    newTotal,
        ChangeReason:     reason,
        ScheduleChangeID: scheduleChange.ID,
    })
    if err != nil {
        return err
    }
    
    // Mark schedule change as completed
    err = tx.UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
        ID:          scheduleChange.ID,
        Status:      "completed",
        ProcessedAt: time.Now(),
    })
    if err != nil {
        return err
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return err
    }
    
    // Send confirmation email
    sms.emailSvc.SendUpgradeConfirmation(ctx, sub.CustomerEmail, UpgradeDetails{
        OldPlan:         sub.PlanName,
        NewPlan:         sms.getNewPlanName(newLineItems),
        ProrationCredit: proration.CreditAmount,
        AmountCharged:   proration.NetAmount,
        EffectiveDate:   time.Now(),
    })
    
    return nil
}

// Downgrade Subscription (Scheduled for end of period)
func (sms *SubscriptionManagementService) DowngradeSubscription(
    ctx context.Context,
    subscriptionID uuid.UUID,
    newLineItems []LineItemUpdate,
    reason string,
) error {
    // Get current subscription
    sub, err := sms.db.GetSubscriptionWithLineItems(ctx, subscriptionID)
    if err != nil {
        return err
    }
    
    // Schedule for end of period
    scheduleResult := sms.calculator.ScheduleDowngrade(sub, newLineItems, time.Now())
    
    // Create schedule change record
    _, err = sms.db.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
        SubscriptionID: subscriptionID,
        ChangeType:     "downgrade",
        ScheduledFor:   scheduleResult.ScheduledFor,
        FromLineItems:  sub.CurrentLineItemsJSON(),
        ToLineItems:    newLineItemsJSON(newLineItems),
        Status:         "scheduled",
        Reason:         reason,
        InitiatedBy:    "customer",
    })
    if err != nil {
        return err
    }
    
    // Send confirmation
    sms.emailSvc.SendDowngradeScheduled(ctx, sub.CustomerEmail, DowngradeDetails{
        CurrentPlan:   sub.PlanName,
        NewPlan:       sms.getNewPlanName(newLineItems),
        EffectiveDate: scheduleResult.ScheduledFor,
        Message:       scheduleResult.Message,
    })
    
    return nil
}

// Cancel Subscription (End of period)
func (sms *SubscriptionManagementService) CancelSubscription(
    ctx context.Context,
    subscriptionID uuid.UUID,
    reason string,
    feedback string,
) error {
    // Get subscription
    sub, err := sms.db.GetSubscription(ctx, subscriptionID)
    if err != nil {
        return err
    }
    
    // Check if already cancelled
    if sub.Status == "cancelled" || sub.CancelAt != nil {
        return fmt.Errorf("subscription already cancelled")
    }
    
    // Set cancellation for end of period
    cancelDate := sub.CurrentPeriodEnd
    
    // Update subscription
    err = sms.db.ScheduleSubscriptionCancellation(ctx, db.ScheduleCancellationParams{
        SubscriptionID:    subscriptionID,
        CancelAt:          cancelDate,
        CancellationReason: reason,
        CancellationFeedback: feedback,
    })
    if err != nil {
        return err
    }
    
    // Create schedule change
    _, err = sms.db.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
        SubscriptionID: subscriptionID,
        ChangeType:     "cancel",
        ScheduledFor:   cancelDate,
        Status:         "scheduled",
        Reason:         reason,
        InitiatedBy:    "customer",
        Metadata:       map[string]interface{}{"feedback": feedback},
    })
    
    // Send confirmation
    sms.emailSvc.SendCancellationScheduled(ctx, sub.CustomerEmail, CancellationDetails{
        PlanName:          sub.PlanName,
        CancellationDate:  cancelDate,
        AccessUntil:       cancelDate,
        RefundAmount:      0, // No refund for end-of-period cancellation
    })
    
    return nil
}

// Pause Subscription
func (sms *SubscriptionManagementService) PauseSubscription(
    ctx context.Context,
    subscriptionID uuid.UUID,
    pauseUntil *time.Time,
    reason string,
) error {
    sub, err := sms.db.GetSubscription(ctx, subscriptionID)
    if err != nil {
        return err
    }
    
    // Validate pause request
    if sub.Status != "active" {
        return fmt.Errorf("can only pause active subscriptions")
    }
    
    // Update subscription status
    err = sms.db.UpdateSubscriptionStatus(ctx, db.UpdateStatusParams{
        SubscriptionID: subscriptionID,
        Status:         "paused",
        PausedAt:       time.Now(),
        PauseEndsAt:    pauseUntil,
    })
    if err != nil {
        return err
    }
    
    // Create schedule change for automatic resume
    if pauseUntil != nil {
        _, err = sms.db.CreateScheduleChange(ctx, db.CreateScheduleChangeParams{
            SubscriptionID: subscriptionID,
            ChangeType:     "resume",
            ScheduledFor:   *pauseUntil,
            Status:         "scheduled",
            Reason:         "Automatic resume after pause period",
            InitiatedBy:    "system",
        })
    }
    
    return nil
}

// Resume Subscription
func (sms *SubscriptionManagementService) ResumeSubscription(
    ctx context.Context,
    subscriptionID uuid.UUID,
) error {
    sub, err := sms.db.GetSubscription(ctx, subscriptionID)
    if err != nil {
        return err
    }
    
    if sub.Status != "paused" {
        return fmt.Errorf("can only resume paused subscriptions")
    }
    
    // Calculate new billing cycle
    newPeriodStart := time.Now()
    newPeriodEnd := sms.calendar.AddBillingPeriod(newPeriodStart, sub.IntervalType, sub.IntervalCount)
    
    // Update subscription
    err = sms.db.ResumeSubscription(ctx, db.ResumeSubscriptionParams{
        SubscriptionID:      subscriptionID,
        Status:              "active",
        CurrentPeriodStart:  newPeriodStart,
        CurrentPeriodEnd:    newPeriodEnd,
        NextBillingDate:     newPeriodEnd,
    })
    
    // Process immediate payment for new period
    _, err = sms.paymentSvc.ProcessSubscriptionPayment(ctx, subscriptionID)
    
    return err
}

// Process Scheduled Changes (Background job)
func (sms *SubscriptionManagementService) ProcessScheduledChanges(ctx context.Context) error {
    // Get all due scheduled changes
    changes, err := sms.db.GetDueScheduledChanges(ctx, time.Now())
    if err != nil {
        return err
    }
    
    for _, change := range changes {
        switch change.ChangeType {
        case "downgrade":
            err = sms.executeDowngrade(ctx, change)
        case "cancel":
            err = sms.executeCancellation(ctx, change)
        case "resume":
            err = sms.executeResume(ctx, change)
        }
        
        if err != nil {
            log.Printf("Failed to process scheduled change %s: %v", change.ID, err)
            // Mark as failed but continue processing others
            sms.db.UpdateScheduleChangeStatus(ctx, db.UpdateScheduleChangeStatusParams{
                ID:     change.ID,
                Status: "failed",
                Metadata: map[string]interface{}{"error": err.Error()},
            })
        }
    }
    
    return nil
}
```

## API Endpoints

### Subscription Management Endpoints

```go
// PUT /api/v1/subscriptions/:id/upgrade
type UpgradeSubscriptionRequest struct {
    LineItems []LineItemUpdate `json:"line_items" binding:"required"`
    Reason    string          `json:"reason"`
}

type LineItemUpdate struct {
    Action       string    `json:"action" binding:"required,oneof=add update remove"`
    LineItemID   uuid.UUID `json:"line_item_id,omitempty"` // For update/remove
    ProductID    uuid.UUID `json:"product_id,omitempty"`   // For add
    PriceID      uuid.UUID `json:"price_id,omitempty"`     // For add
    Quantity     int       `json:"quantity"`               // For add/update
}

// PUT /api/v1/subscriptions/:id/downgrade
type DowngradeSubscriptionRequest struct {
    LineItems []LineItemUpdate `json:"line_items" binding:"required"`
    Reason    string          `json:"reason"`
}

// DELETE /api/v1/subscriptions/:id
type CancelSubscriptionRequest struct {
    Reason   string `json:"reason" binding:"required"`
    Feedback string `json:"feedback"`
}

// POST /api/v1/subscriptions/:id/pause
type PauseSubscriptionRequest struct {
    PauseUntil *time.Time `json:"pause_until"` // Optional end date
    Reason     string     `json:"reason"`
}

// POST /api/v1/subscriptions/:id/resume
// No body required

// GET /api/v1/subscriptions/:id/scheduled-changes
type ScheduledChangesResponse struct {
    Changes []ScheduledChange `json:"scheduled_changes"`
}

// DELETE /api/v1/subscriptions/:id/scheduled-changes/:changeId
// Cancel a scheduled change

// POST /api/v1/subscriptions/:id/preview-change
type PreviewChangeRequest struct {
    ChangeType string           `json:"change_type" binding:"required,oneof=upgrade downgrade"`
    LineItems  []LineItemUpdate `json:"line_items" binding:"required"`
}

type PreviewChangeResponse struct {
    CurrentAmount      MoneyAmount       `json:"current_amount"`
    NewAmount          MoneyAmount       `json:"new_amount"`
    ProrationCredit    MoneyAmount       `json:"proration_credit,omitempty"`
    ImmediateCharge    MoneyAmount       `json:"immediate_charge,omitempty"`
    EffectiveDate      time.Time         `json:"effective_date"`
    BillingImpact      []BillingImpact   `json:"billing_impact"`
}
```

## Handler Implementation

```go
func (h *SubscriptionHandler) UpgradeSubscription(c *gin.Context) {
    subscriptionID := c.Param("id")
    var req UpgradeSubscriptionRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Verify ownership
    sub, err := h.queries.GetSubscription(c, uuid.MustParse(subscriptionID))
    if err != nil {
        c.JSON(404, gin.H{"error": "Subscription not found"})
        return
    }
    
    if sub.WorkspaceID != c.GetString("workspace_id") {
        c.JSON(403, gin.H{"error": "Forbidden"})
        return
    }
    
    // Process upgrade
    err = h.managementService.UpgradeSubscription(
        c,
        sub.ID,
        req.LineItems,
        req.Reason,
    )
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Return updated subscription
    updated, _ := h.queries.GetSubscriptionWithDetails(c, sub.ID)
    c.JSON(200, gin.H{
        "subscription": updated,
        "message": "Subscription upgraded successfully",
    })
}

func (h *SubscriptionHandler) PreviewChange(c *gin.Context) {
    subscriptionID := c.Param("id")
    var req PreviewChangeRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Get subscription
    sub, err := h.queries.GetSubscriptionWithLineItems(c, uuid.MustParse(subscriptionID))
    if err != nil {
        c.JSON(404, gin.H{"error": "Subscription not found"})
        return
    }
    
    // Calculate preview based on change type
    var preview *PreviewChangeResponse
    
    if req.ChangeType == "upgrade" {
        preview = h.previewUpgrade(sub, req.LineItems)
    } else {
        preview = h.previewDowngrade(sub, req.LineItems)
    }
    
    c.JSON(200, preview)
}
```

## Frontend Components

### Subscription Management UI

```typescript
interface SubscriptionManagementProps {
    subscription: Subscription;
}

export function SubscriptionManagement({ subscription }: SubscriptionManagementProps) {
    const [showUpgradeModal, setShowUpgradeModal] = useState(false);
    const [showCancelModal, setShowCancelModal] = useState(false);
    
    const { mutate: cancelSubscription } = useMutation({
        mutationFn: (data: CancelRequest) => 
            api.subscriptions.cancel(subscription.id, data),
        onSuccess: () => {
            toast.success('Subscription cancelled. You have access until ' + 
                formatDate(subscription.currentPeriodEnd));
        },
    });
    
    return (
        <Card>
            <CardHeader>
                <CardTitle>Subscription Management</CardTitle>
                <CardDescription>
                    Current plan: {subscription.planName} - ${subscription.amount / 100}/mo
                </CardDescription>
            </CardHeader>
            
            <CardContent>
                <div className="space-y-4">
                    {subscription.status === 'active' && (
                        <>
                            <Button 
                                onClick={() => setShowUpgradeModal(true)}
                                className="w-full"
                            >
                                Upgrade Plan
                            </Button>
                            
                            <Button 
                                variant="outline" 
                                onClick={() => setShowCancelModal(true)}
                                className="w-full"
                            >
                                Cancel Subscription
                            </Button>
                        </>
                    )}
                    
                    {subscription.cancelAt && (
                        <Alert>
                            <AlertDescription>
                                Subscription will cancel on {formatDate(subscription.cancelAt)}.
                                You'll have access until then.
                            </AlertDescription>
                            <Button 
                                size="sm" 
                                variant="link"
                                onClick={() => reactivateSubscription(subscription.id)}
                            >
                                Keep Subscription
                            </Button>
                        </Alert>
                    )}
                    
                    {subscription.scheduledChanges?.map(change => (
                        <Alert key={change.id}>
                            <AlertDescription>
                                {change.changeType === 'downgrade' && 
                                    `Downgrade scheduled for ${formatDate(change.scheduledFor)}`}
                            </AlertDescription>
                        </Alert>
                    ))}
                </div>
            </CardContent>
            
            <UpgradeModal 
                open={showUpgradeModal}
                onClose={() => setShowUpgradeModal(false)}
                currentSubscription={subscription}
            />
            
            <CancelModal
                open={showCancelModal}
                onClose={() => setShowCancelModal(false)}
                onConfirm={(reason, feedback) => {
                    cancelSubscription({ reason, feedback });
                    setShowCancelModal(false);
                }}
            />
        </Card>
    );
}
```

### Upgrade Modal with Proration Preview

```typescript
export function UpgradeModal({ open, onClose, currentSubscription }) {
    const [selectedItems, setSelectedItems] = useState<LineItemUpdate[]>([]);
    
    const { data: preview } = useQuery({
        queryKey: ['preview-upgrade', selectedItems],
        queryFn: () => api.subscriptions.previewChange(currentSubscription.id, {
            changeType: 'upgrade',
            lineItems: selectedItems,
        }),
        enabled: selectedItems.length > 0,
    });
    
    const { mutate: upgrade } = useMutation({
        mutationFn: () => api.subscriptions.upgrade(currentSubscription.id, {
            lineItems: selectedItems,
            reason: 'Customer requested upgrade',
        }),
        onSuccess: () => {
            toast.success('Subscription upgraded successfully!');
            onClose();
        },
    });
    
    return (
        <Dialog open={open} onOpenChange={onClose}>
            <DialogContent className="max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Upgrade Your Subscription</DialogTitle>
                </DialogHeader>
                
                <div className="space-y-6">
                    <ProductSelector 
                        currentItems={currentSubscription.lineItems}
                        onItemsChange={setSelectedItems}
                    />
                    
                    {preview && (
                        <Card>
                            <CardHeader>
                                <CardTitle>Upgrade Summary</CardTitle>
                            </CardHeader>
                            <CardContent>
                                <div className="space-y-2">
                                    <div className="flex justify-between">
                                        <span>Current Plan:</span>
                                        <span>${preview.currentAmount.value}</span>
                                    </div>
                                    <div className="flex justify-between">
                                        <span>New Plan:</span>
                                        <span>${preview.newAmount.value}</span>
                                    </div>
                                    
                                    <Separator />
                                    
                                    <div className="flex justify-between text-sm text-gray-600">
                                        <span>Proration Credit:</span>
                                        <span>-${preview.prorationCredit.value}</span>
                                    </div>
                                    
                                    <div className="flex justify-between font-semibold">
                                        <span>Due Today:</span>
                                        <span>${preview.immediateCharge.value}</span>
                                    </div>
                                </div>
                                
                                <Alert className="mt-4">
                                    <InfoIcon className="h-4 w-4" />
                                    <AlertDescription>
                                        Your new plan starts immediately. Future bills will be 
                                        ${preview.newAmount.value} starting {formatDate(preview.nextBillingDate)}.
                                    </AlertDescription>
                                </Alert>
                            </CardContent>
                        </Card>
                    )}
                    
                    <div className="flex justify-end space-x-4">
                        <Button variant="outline" onClick={onClose}>
                            Cancel
                        </Button>
                        <Button 
                            onClick={() => upgrade()}
                            disabled={!preview || preview.immediateCharge.value < 0}
                        >
                            Upgrade Now {preview && `($${preview.immediateCharge.value})`}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
```

## Background Job for Scheduled Changes

```go
// Lambda handler or cron job
func ProcessScheduledSubscriptionChanges(ctx context.Context) error {
    svc := NewSubscriptionManagementService(db, logger)
    
    // Process all due changes
    err := svc.ProcessScheduledChanges(ctx)
    if err != nil {
        logger.Error("Failed to process scheduled changes", zap.Error(err))
        return err
    }
    
    // Also check for subscriptions that need renewal
    err = svc.ProcessSubscriptionRenewals(ctx)
    if err != nil {
        logger.Error("Failed to process renewals", zap.Error(err))
        return err
    }
    
    return nil
}
```

## Key Benefits

1. **Fair Billing**: Customers only pay for what they use
2. **Flexibility**: Support all subscription lifecycle operations
3. **Transparency**: Clear proration calculations shown upfront
4. **Revenue Protection**: Downgrades don't cause immediate revenue loss
5. **Customer Satisfaction**: No double-billing or unfair charges

This system handles all the complex subscription management scenarios while maintaining simplicity for the customer and protecting your revenue.