# API Routes Implementation Plan

This document outlines all the API routes that need to be implemented based on the platform enhancement plan. Routes are organized by handler and include validation requirements.

## Current Route Analysis

### Existing Routes
- `/api/v1/subscriptions/*` - Currently only has `GET /` (ListSubscriptions)
- `/api/v1/subscription-events/*` - Limited to transaction listing
- No invoice routes exist
- No payment routes exist
- No payment link routes exist
- No discount routes exist
- No dashboard/analytics routes exist

## Required Handler Additions

Add these handler definitions to server.go:

```go
// Handler Definitions (Add to existing)
var (
    // Existing handlers...
    
    // New handlers needed
    invoiceHandler           *handlers.InvoiceHandler
    paymentHandler           *handlers.PaymentHandler  
    paymentLinkHandler       *handlers.PaymentLinkHandler
    discountHandler          *handlers.DiscountHandler
    dashboardHandler         *handlers.DashboardHandler
    currencyHandler          *handlers.CurrencyHandler
)
```

## Handler Initialization

Add to InitializeHandlers():

```go
// Initialize new handlers
invoiceHandler = handlers.NewInvoiceHandler(commonServices)
paymentHandler = handlers.NewPaymentHandler(commonServices)
paymentLinkHandler = handlers.NewPaymentLinkHandler(commonServices, delegationClient)
discountHandler = handlers.NewDiscountHandler(commonServices)
dashboardHandler = handlers.NewDashboardHandler(commonServices)
currencyHandler = handlers.NewCurrencyHandler(commonServices)
```

## Route Implementation Plan

### 1. Invoice Routes

```go
// Invoices
invoices := protected.Group("/invoices")
{
    // List and filtering
    invoices.GET("", middleware.ValidateQueryParams(middleware.ListQueryValidation), invoiceHandler.ListInvoices)
    invoices.GET("/customer/:customer_id", invoiceHandler.ListInvoicesByCustomer)
    invoices.GET("/status/:status", invoiceHandler.ListInvoicesByStatus)
    invoices.GET("/overdue", invoiceHandler.ListOverdueInvoices)
    
    // CRUD operations
    invoices.POST("", middleware.ValidateInput(middleware.CreateInvoiceValidation), invoiceHandler.CreateInvoice)
    invoices.GET("/:invoice_id", invoiceHandler.GetInvoice)
    invoices.PUT("/:invoice_id", middleware.ValidateInput(middleware.UpdateInvoiceValidation), invoiceHandler.UpdateInvoice)
    invoices.DELETE("/:invoice_id", invoiceHandler.DeleteInvoice)
    
    // Invoice actions
    invoices.POST("/:invoice_id/send", invoiceHandler.SendInvoice)
    invoices.POST("/:invoice_id/finalize", invoiceHandler.FinalizeInvoice)
    invoices.POST("/:invoice_id/void", invoiceHandler.VoidInvoice)
    invoices.POST("/:invoice_id/mark-paid", invoiceHandler.MarkInvoiceAsPaid)
    
    // Invoice line items
    invoices.GET("/:invoice_id/line-items", invoiceHandler.GetInvoiceLineItems)
    invoices.POST("/:invoice_id/line-items", middleware.ValidateInput(middleware.CreateInvoiceLineItemValidation), invoiceHandler.AddInvoiceLineItem)
    invoices.PUT("/:invoice_id/line-items/:line_item_id", middleware.ValidateInput(middleware.UpdateInvoiceLineItemValidation), invoiceHandler.UpdateInvoiceLineItem)
    invoices.DELETE("/:invoice_id/line-items/:line_item_id", invoiceHandler.DeleteInvoiceLineItem)
    
    // Invoice payments
    invoices.GET("/:invoice_id/payments", invoiceHandler.ListInvoicePayments)
    invoices.POST("/:invoice_id/pay", middleware.ValidateInput(middleware.PayInvoiceValidation), invoiceHandler.PayInvoice)
}
```

### 2. Payment Routes

```go
// Payments
payments := protected.Group("/payments")
{
    // List and filtering
    payments.GET("", middleware.ValidateQueryParams(middleware.ListQueryValidation), paymentHandler.ListPayments)
    payments.GET("/customer/:customer_id", paymentHandler.ListPaymentsByCustomer)
    payments.GET("/invoice/:invoice_id", paymentHandler.ListPaymentsByInvoice)
    payments.GET("/subscription/:subscription_id", paymentHandler.ListPaymentsBySubscription)
    payments.GET("/status/:status", paymentHandler.ListPaymentsByStatus)
    
    // Payment details
    payments.GET("/:payment_id", paymentHandler.GetPayment)
    payments.GET("/transaction/:tx_hash", paymentHandler.GetPaymentByTransactionHash)
    
    // Payment actions
    payments.POST("/:payment_id/refund", middleware.ValidateInput(middleware.RefundPaymentValidation), paymentHandler.RefundPayment)
    payments.POST("/:payment_id/retry", paymentHandler.RetryPayment)
    
    // Analytics
    payments.GET("/volume", middleware.ValidateQueryParams(middleware.DateRangeValidation), paymentHandler.GetPaymentVolume)
    payments.GET("/revenue", middleware.ValidateQueryParams(middleware.DateRangeValidation), paymentHandler.GetRevenue)
}
```

### 3. Payment Link Routes

```go
// Payment Links
paymentLinks := protected.Group("/payment-links")
{
    // CRUD operations
    paymentLinks.GET("", middleware.ValidateQueryParams(middleware.ListQueryValidation), paymentLinkHandler.ListPaymentLinks)
    paymentLinks.POST("", middleware.ValidateInput(middleware.CreatePaymentLinkValidation), paymentLinkHandler.CreatePaymentLink)
    paymentLinks.GET("/:link_id", paymentLinkHandler.GetPaymentLink)
    paymentLinks.PUT("/:link_id", middleware.ValidateInput(middleware.UpdatePaymentLinkValidation), paymentLinkHandler.UpdatePaymentLink)
    paymentLinks.DELETE("/:link_id", paymentLinkHandler.DeletePaymentLink)
    
    // Link actions
    paymentLinks.POST("/:link_id/activate", paymentLinkHandler.ActivatePaymentLink)
    paymentLinks.POST("/:link_id/deactivate", paymentLinkHandler.DeactivatePaymentLink)
    
    // Link usage
    paymentLinks.GET("/:link_id/usage", paymentLinkHandler.GetPaymentLinkUsage)
    paymentLinks.GET("/:link_id/payments", paymentLinkHandler.ListPaymentLinkPayments)
}

// Public payment link endpoints (no auth required)
public := v1.Group("/public")
{
    // Public payment link access
    public.GET("/pay/:link_code", paymentLinkHandler.GetPublicPaymentLink)
    public.POST("/pay/:link_code", middleware.ValidateInput(middleware.ProcessPaymentLinkValidation), paymentLinkHandler.ProcessPaymentLink)
}
```

### 4. Discount Routes

```go
// Discounts
discounts := protected.Group("/discounts")
{
    // Discount codes
    discounts.GET("/codes", middleware.ValidateQueryParams(middleware.ListQueryValidation), discountHandler.ListDiscountCodes)
    discounts.POST("/codes", middleware.ValidateInput(middleware.CreateDiscountCodeValidation), discountHandler.CreateDiscountCode)
    discounts.GET("/codes/:code", discountHandler.GetDiscountCode)
    discounts.PUT("/codes/:code_id", middleware.ValidateInput(middleware.UpdateDiscountCodeValidation), discountHandler.UpdateDiscountCode)
    discounts.DELETE("/codes/:code_id", discountHandler.DeleteDiscountCode)
    
    // Validate discount
    discounts.POST("/validate", middleware.ValidateInput(middleware.ValidateDiscountValidation), discountHandler.ValidateDiscount)
    
    // Discount usage
    discounts.GET("/codes/:code_id/usage", discountHandler.GetDiscountUsage)
    discounts.GET("/customer/:customer_id", discountHandler.ListCustomerDiscounts)
    
    // Active discounts on subscriptions
    discounts.GET("/subscriptions/:subscription_id", discountHandler.GetSubscriptionDiscounts)
}
```

### 5. Enhanced Subscription Routes

Update existing subscription routes:

```go
// Subscriptions (enhanced)
subscriptions := protected.Group("/subscriptions")
{
    // Existing routes...
    subscriptions.GET("", subscriptionHandler.ListSubscriptions)
    
    // Enable these existing routes
    subscriptions.GET("/active", subscriptionHandler.ListActiveSubscriptions)
    subscriptions.GET("/expired", subscriptionHandler.GetExpiredSubscriptions)
    subscriptions.POST("", middleware.ValidateInput(middleware.CreateSubscriptionValidation), subscriptionHandler.CreateSubscription)
    subscriptions.GET("/:subscription_id", subscriptionHandler.GetSubscription)
    subscriptions.GET("/:subscription_id/details", subscriptionHandler.GetSubscriptionWithDetails)
    subscriptions.PUT("/:subscription_id", middleware.ValidateInput(middleware.UpdateSubscriptionValidation), subscriptionHandler.UpdateSubscription)
    subscriptions.PATCH("/:subscription_id/status", middleware.ValidateInput(middleware.UpdateSubscriptionStatusValidation), subscriptionHandler.UpdateSubscriptionStatus)
    subscriptions.POST("/:subscription_id/cancel", subscriptionHandler.CancelSubscription)
    subscriptions.DELETE("/:subscription_id", subscriptionHandler.DeleteSubscription)
    
    // New line item routes
    subscriptions.GET("/:subscription_id/line-items", subscriptionHandler.GetSubscriptionLineItems)
    subscriptions.POST("/:subscription_id/line-items", middleware.ValidateInput(middleware.AddSubscriptionLineItemValidation), subscriptionHandler.AddSubscriptionLineItem)
    subscriptions.PUT("/:subscription_id/line-items/:line_item_id", middleware.ValidateInput(middleware.UpdateSubscriptionLineItemValidation), subscriptionHandler.UpdateSubscriptionLineItem)
    subscriptions.DELETE("/:subscription_id/line-items/:line_item_id", subscriptionHandler.RemoveSubscriptionLineItem)
    
    // Subscription discounts
    subscriptions.POST("/:subscription_id/discounts", middleware.ValidateInput(middleware.ApplyDiscountValidation), subscriptionHandler.ApplyDiscount)
    subscriptions.DELETE("/:subscription_id/discounts/:discount_id", subscriptionHandler.RemoveDiscount)
    
    // Subscription trials
    subscriptions.POST("/:subscription_id/trial", middleware.ValidateInput(middleware.StartTrialValidation), subscriptionHandler.StartTrial)
    subscriptions.POST("/:subscription_id/trial/end", subscriptionHandler.EndTrial)
    
    // Subscription invoices
    subscriptions.GET("/:subscription_id/invoices", subscriptionHandler.GetSubscriptionInvoices)
    subscriptions.POST("/:subscription_id/invoice", subscriptionHandler.GenerateInvoice)
    
    // Analytics (enable existing)
    subscriptions.GET("/:subscription_id/total-amount", subscriptionEventHandler.GetTotalAmountBySubscription)
    subscriptions.GET("/:subscription_id/redemption-count", subscriptionEventHandler.GetSuccessfulRedemptionCount)
    subscriptions.GET("/:subscription_id/latest-event", subscriptionEventHandler.GetLatestSubscriptionEvent)
    subscriptions.GET("/:subscription_id/events", subscriptionEventHandler.ListSubscriptionEventsBySubscription)
}
```

### 6. Dashboard/Analytics Routes

```go
// Dashboard
dashboard := protected.Group("/dashboard")
{
    // Overview metrics
    dashboard.GET("/overview", dashboardHandler.GetOverview)
    dashboard.GET("/metrics", middleware.ValidateQueryParams(middleware.DateRangeValidation), dashboardHandler.GetMetrics)
    
    // Revenue analytics
    dashboard.GET("/revenue/mrr", dashboardHandler.GetMRR)
    dashboard.GET("/revenue/arr", dashboardHandler.GetARR)
    dashboard.GET("/revenue/growth", middleware.ValidateQueryParams(middleware.DateRangeValidation), dashboardHandler.GetRevenueGrowth)
    dashboard.GET("/revenue/by-product", dashboardHandler.GetRevenueByProduct)
    dashboard.GET("/revenue/by-token", dashboardHandler.GetRevenueByToken)
    
    // Customer analytics
    dashboard.GET("/customers/count", dashboardHandler.GetCustomerCount)
    dashboard.GET("/customers/growth", middleware.ValidateQueryParams(middleware.DateRangeValidation), dashboardHandler.GetCustomerGrowth)
    dashboard.GET("/customers/churn", dashboardHandler.GetChurnRate)
    dashboard.GET("/customers/ltv", dashboardHandler.GetCustomerLTV)
    
    // Subscription analytics
    dashboard.GET("/subscriptions/active", dashboardHandler.GetActiveSubscriptions)
    dashboard.GET("/subscriptions/by-status", dashboardHandler.GetSubscriptionsByStatus)
    dashboard.GET("/subscriptions/conversion", dashboardHandler.GetConversionRate)
    
    // Transaction analytics
    dashboard.GET("/transactions/volume", middleware.ValidateQueryParams(middleware.DateRangeValidation), dashboardHandler.GetTransactionVolume)
    dashboard.GET("/transactions/failed", dashboardHandler.GetFailedTransactions)
    dashboard.GET("/transactions/gas-costs", dashboardHandler.GetGasCosts)
}
```

### 7. Currency Routes

```go
// Currencies
currencies := protected.Group("/currencies")
{
    currencies.GET("", currencyHandler.ListCurrencies)
    currencies.GET("/:code", currencyHandler.GetCurrency)
    currencies.POST("/:code/enable", currencyHandler.EnableCurrency)
    currencies.POST("/:code/disable", currencyHandler.DisableCurrency)
}
```

### 8. Gas Sponsorship Routes

```go
// Gas sponsorship management
gasSponsorship := protected.Group("/gas-sponsorship")
{
    // Configuration
    gasSponsorship.GET("/config", dashboardHandler.GetGasSponsorshipConfig)
    gasSponsorship.PUT("/config", middleware.ValidateInput(middleware.UpdateGasSponsorshipValidation), dashboardHandler.UpdateGasSponsorshipConfig)
    
    // Analytics
    gasSponsorship.GET("/analytics", middleware.ValidateQueryParams(middleware.DateRangeValidation), dashboardHandler.GetGasAnalytics)
    gasSponsorship.GET("/usage", dashboardHandler.GetGasSponsorshipUsage)
    gasSponsorship.GET("/breakdown", dashboardHandler.GetGasBreakdown)
    
    // Rules management
    gasSponsorship.POST("/rules/products", middleware.ValidateInput(middleware.GasSponsorshipRuleValidation), dashboardHandler.AddProductSponsorshipRule)
    gasSponsorship.DELETE("/rules/products/:product_id", dashboardHandler.RemoveProductSponsorshipRule)
    gasSponsorship.POST("/rules/customers", middleware.ValidateInput(middleware.GasSponsorshipRuleValidation), dashboardHandler.AddCustomerSponsorshipRule)
    gasSponsorship.DELETE("/rules/customers/:customer_id", dashboardHandler.RemoveCustomerSponsorshipRule)
}
```

### 9. Customer Enhancement Routes

Add to existing customer routes:

```go
// In customers group, add:
// Customer payment methods
customers.GET("/:customer_id/payment-methods", customerHandler.GetPaymentMethods)
customers.POST("/:customer_id/payment-methods", middleware.ValidateInput(middleware.AddPaymentMethodValidation), customerHandler.AddPaymentMethod)
customers.DELETE("/:customer_id/payment-methods/:method_id", customerHandler.RemovePaymentMethod)

// Customer invoices
customers.GET("/:customer_id/invoices", customerHandler.GetCustomerInvoices)

// Customer payments
customers.GET("/:customer_id/payments", customerHandler.GetCustomerPayments)
```

## Validation Middleware Requirements

New validation rules needed in `/libs/go/middleware/validation_rules.go`:

```go
// Invoice validations
var CreateInvoiceValidation = ValidationRules{...}
var UpdateInvoiceValidation = ValidationRules{...}
var CreateInvoiceLineItemValidation = ValidationRules{...}
var UpdateInvoiceLineItemValidation = ValidationRules{...}
var PayInvoiceValidation = ValidationRules{...}

// Payment validations
var RefundPaymentValidation = ValidationRules{...}

// Payment link validations
var CreatePaymentLinkValidation = ValidationRules{...}
var UpdatePaymentLinkValidation = ValidationRules{...}
var ProcessPaymentLinkValidation = ValidationRules{...}

// Discount validations
var CreateDiscountCodeValidation = ValidationRules{...}
var UpdateDiscountCodeValidation = ValidationRules{...}
var ValidateDiscountValidation = ValidationRules{...}
var ApplyDiscountValidation = ValidationRules{...}

// Subscription enhancement validations
var CreateSubscriptionValidation = ValidationRules{...}
var UpdateSubscriptionValidation = ValidationRules{...}
var UpdateSubscriptionStatusValidation = ValidationRules{...}
var AddSubscriptionLineItemValidation = ValidationRules{...}
var UpdateSubscriptionLineItemValidation = ValidationRules{...}
var StartTrialValidation = ValidationRules{...}

// Query validations
var DateRangeValidation = ValidationRules{...}

// Payment method validations
var AddPaymentMethodValidation = ValidationRules{...}

// Gas sponsorship validations
var UpdateGasSponsorshipValidation = ValidationRules{...}
var GasSponsorshipRuleValidation = ValidationRules{...}
```

## Implementation Order

1. **Phase 1: Core Infrastructure**
   - Currency handler and routes
   - Enhanced subscription routes with line items
   - Basic invoice creation and management

2. **Phase 2: Payment Processing**
   - Payment handler and routes
   - Payment link functionality
   - Invoice payment processing

3. **Phase 3: Business Features**
   - Discount system
   - Trial management
   - Customer payment methods

4. **Phase 4: Analytics**
   - Dashboard handler
   - Revenue metrics
   - Customer analytics
   - Real-time metrics updates

## Route Security Considerations

1. **Authentication**: All routes except public payment links require authentication
2. **Workspace Isolation**: All handlers must enforce workspace-level data isolation
3. **Rate Limiting**: Apply appropriate rate limits, especially on:
   - Payment processing endpoints
   - Dashboard/analytics endpoints
   - Public payment link access
4. **Validation**: Input validation on all POST/PUT/PATCH endpoints
5. **Audit Logging**: Log all financial transactions and state changes

## Error Handling

All handlers should return consistent error responses:

```json
{
  "error": {
    "code": "INVOICE_NOT_FOUND",
    "message": "Invoice not found",
    "correlation_id": "xxx-xxx-xxx"
  }
}
```

## Pagination

All list endpoints should support standard pagination:
- `?page=1&limit=20`
- Return total count in headers
- Support filtering and sorting where applicable