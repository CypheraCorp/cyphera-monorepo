# Interface Refactoring Guide

This guide explains how to refactor the Cyphera codebase to use interfaces and generated mocks for better testability and decoupling.

## Overview

We're using [uber-go/mock](https://github.com/uber-go/mock) (formerly gomock) to generate mocks for our interfaces. This allows us to:

1. Write unit tests without depending on real implementations
2. Decouple services from concrete implementations
3. Make the codebase more modular and testable

## Setup

### Installation

```bash
go install go.uber.org/mock/mockgen@latest
```

### Project Structure

```
libs/go/
├── interfaces/          # Interface definitions
│   ├── services.go      # Service interfaces
│   └── clients.go       # External client interfaces
├── mocks/              # Generated mocks
│   ├── mock_querier.go  # Generated from db.Querier
│   ├── mock_services.go # Generated from service interfaces
│   └── mock_clients.go  # Generated from client interfaces
└── services/           # Service implementations
```

## Refactoring Process

### Step 1: Define Interfaces

For each service dependency, define an interface in `libs/go/interfaces/`:

```go
// PaymentService handles payment processing operations
type PaymentService interface {
    ProcessPayment(ctx context.Context, payment *db.Payment) error
    CreatePayment(ctx context.Context, params db.CreatePaymentParams) (*db.Payment, error)
    // Add only the methods that are actually used by consumers
}
```

### Step 2: Generate Mocks

Generate mocks for your interfaces:

```bash
# For service interfaces
mockgen -source=libs/go/interfaces/services.go -destination=libs/go/mocks/mock_services.go -package=mocks

# For client interfaces  
mockgen -source=libs/go/interfaces/clients.go -destination=libs/go/mocks/mock_clients.go -package=mocks

# For database interface (already exists as db.Querier)
mockgen -source=libs/go/db/querier.go -destination=libs/go/mocks/mock_querier.go -package=mocks
```

### Step 3: Refactor Services to Use Interfaces

#### Before:
```go
type SubscriptionManagementService struct {
    db             *db.Queries
    paymentService *PaymentService
    emailService   *EmailService
}

func NewSubscriptionManagementService(
    db *db.Queries,
    paymentService *PaymentService,
    emailService *EmailService,
) *SubscriptionManagementService {
    // ...
}
```

#### After:
```go
type SubscriptionManagementService struct {
    db             db.Querier // Use the interface
    paymentService PaymentServiceInterface
    emailService   EmailServiceInterface
}

// Define local interfaces for dependencies
type PaymentServiceInterface interface {
    ProcessPayment(ctx context.Context, payment *db.Payment) error
    CreatePayment(ctx context.Context, params db.CreatePaymentParams) (*db.Payment, error)
}

type EmailServiceInterface interface {
    SendTransactionalEmail(ctx context.Context, params TransactionalEmailParams) error
}

func NewSubscriptionManagementService(
    db db.Querier,
    paymentService PaymentServiceInterface,
    emailService EmailServiceInterface,
) *SubscriptionManagementService {
    // ...
}
```

### Step 4: Update Handlers

Update handlers to use interfaces:

```go
// Before
type SubscriptionHandler struct {
    service *services.SubscriptionManagementService
}

// After  
type SubscriptionHandler struct {
    service SubscriptionManagementServiceInterface
}

type SubscriptionManagementServiceInterface interface {
    UpgradeSubscription(ctx context.Context, subscriptionID, newPriceID uuid.UUID) error
    DowngradeSubscription(ctx context.Context, subscriptionID, newPriceID uuid.UUID) error
    // ... other methods
}
```

### Step 5: Write Tests with Mocks

```go
func TestSubscriptionManagementService_UpgradeSubscription(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    // Create mocks
    mockDB := mocks.NewMockQuerier(ctrl)
    mockPaymentService := &MockPaymentService{}
    mockEmailService := &MockEmailService{}
    
    // Create service with mocked dependencies
    service := NewSubscriptionManagementService(
        mockDB,
        mockPaymentService,
        mockEmailService,
    )

    // Set up expectations
    mockDB.EXPECT().
        GetSubscription(ctx, subscriptionID).
        Return(subscription, nil)

    mockEmailService.On("SendTransactionalEmail", ctx, mock.Anything).Return(nil)

    // Execute test
    err := service.UpgradeSubscription(ctx, subscriptionID, newPriceID)

    // Assert
    assert.NoError(t, err)
    mockEmailService.AssertExpectations(t)
}
```

## Best Practices

### 1. Define Minimal Interfaces

Only include methods that are actually used by consumers:

```go
// Good: Minimal interface
type PaymentProcessor interface {
    ProcessPayment(ctx context.Context, payment *db.Payment) error
}

// Avoid: Large interface with unused methods
type PaymentService interface {
    ProcessPayment(ctx context.Context, payment *db.Payment) error
    CreatePayment(...) error
    UpdatePayment(...) error
    DeletePayment(...) error
    // ... 20 more methods
}
```

### 2. Use Interface Composition

Break large interfaces into smaller ones:

```go
type PaymentReader interface {
    GetPayment(ctx context.Context, id uuid.UUID) (*db.Payment, error)
}

type PaymentWriter interface {
    CreatePayment(ctx context.Context, params db.CreatePaymentParams) (*db.Payment, error)
    UpdatePayment(ctx context.Context, id uuid.UUID, params db.UpdatePaymentParams) error
}

type PaymentService interface {
    PaymentReader
    PaymentWriter
}
```

### 3. Mock at the Right Level

- Mock external dependencies (database, APIs, email services)
- Don't mock simple value objects or pure functions
- Consider using real implementations for simple calculators

### 4. Use Constructor Injection

Always inject dependencies through constructors:

```go
func NewService(
    db db.Querier,
    payment PaymentServiceInterface,
    email EmailServiceInterface,
) *Service {
    return &Service{
        db:      db,
        payment: payment,
        email:   email,
    }
}
```

### 5. Create Test Helpers

Create helper functions for common test setups:

```go
func setupMocks(t *testing.T) (*gomock.Controller, *mocks.MockQuerier) {
    ctrl := gomock.NewController(t)
    mockDB := mocks.NewMockQuerier(ctrl)
    return ctrl, mockDB
}

func createTestSubscription(id uuid.UUID) db.Subscription {
    return db.Subscription{
        ID:     id,
        Status: db.SubscriptionStatusActive,
        // ... other fields
    }
}
```

## Migration Strategy

1. **Start with new code**: Use interfaces for all new services and handlers
2. **Refactor during testing**: When adding tests to existing code, refactor to use interfaces
3. **Gradual migration**: Refactor one service at a time, starting with the most tested/critical ones
4. **Update handlers last**: After services use interfaces, update handlers to depend on interfaces

## Common Patterns

### Pattern 1: Service with Multiple Dependencies

```go
type ComplexService struct {
    db        db.Querier
    payment   PaymentServiceInterface
    email     EmailServiceInterface
    cache     CacheInterface
    metrics   MetricsInterface
}

func NewComplexService(opts ComplexServiceOptions) *ComplexService {
    return &ComplexService{
        db:      opts.DB,
        payment: opts.PaymentService,
        email:   opts.EmailService,
        cache:   opts.Cache,
        metrics: opts.Metrics,
    }
}

type ComplexServiceOptions struct {
    DB             db.Querier
    PaymentService PaymentServiceInterface
    EmailService   EmailServiceInterface
    Cache          CacheInterface
    Metrics        MetricsInterface
}
```

### Pattern 2: Optional Dependencies

```go
func NewServiceWithDefaults(db db.Querier) *Service {
    return NewService(
        db,
        services.NewPaymentService(db),
        services.NewEmailService(),
    )
}

func NewService(
    db db.Querier,
    payment PaymentServiceInterface,
    email EmailServiceInterface,
) *Service {
    // Full constructor for testing
}
```

### Pattern 3: Interface Assertions

Ensure your implementations satisfy the interfaces:

```go
// In payment_service.go
var _ PaymentServiceInterface = (*PaymentService)(nil)

// In email_service.go  
var _ EmailServiceInterface = (*EmailService)(nil)
```

## Regenerating Mocks

When interfaces change, regenerate the mocks:

```bash
# Add to Makefile
.PHONY: generate-mocks
generate-mocks:
	mockgen -source=libs/go/db/querier.go -destination=libs/go/mocks/mock_querier.go -package=mocks
	mockgen -source=libs/go/interfaces/services.go -destination=libs/go/mocks/mock_services.go -package=mocks
	mockgen -source=libs/go/interfaces/clients.go -destination=libs/go/mocks/mock_clients.go -package=mocks
```

Then run:
```bash
make generate-mocks
```

## Troubleshooting

### Issue: "undefined: mock.Call"

Make sure to import the mock package:
```go
import "github.com/stretchr/testify/mock"
```

### Issue: "cannot use mockDB (type *mocks.MockQuerier) as type db.Querier"

The generated mock should implement the interface. Check that:
1. The mock was generated from the correct source file
2. The interface hasn't changed since mock generation
3. Regenerate the mocks if needed

### Issue: "too many arguments in call to NewService"

You might be using the old constructor. Update to use the interface-based constructor or create a compatibility wrapper.

## Next Steps

1. Refactor all services in `libs/go/services/` to use interfaces
2. Update all handlers in `apps/api/handlers/` to depend on interfaces
3. Create integration test helpers that use real implementations
4. Document which interfaces each service implements
5. Consider creating a dependency injection container for complex setups