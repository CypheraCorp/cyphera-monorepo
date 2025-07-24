# Cyphera API Implementation Plan

## Executive Summary

This implementation plan addresses critical improvements for the Cyphera API application to transform it into a production-ready, secure, and maintainable service. The plan is organized by priority, combining criticality and difficulty assessments.

**Current State**: Go-based API using Gin framework, SQLC for database operations, deployed on AWS Lambda  
**Target State**: A secure, well-tested, observable API with clean architecture and optimal performance

## Task Completion Criteria

For each task to be considered complete, it must meet ALL of the following criteria:
1. **No Breaking Changes**: All existing API endpoints must continue to work as expected
2. **No New Bugs**: Comprehensive testing to ensure no regressions
3. **Build Success**: The API must build successfully with no errors
4. **No Linting Issues**: Must pass all linting checks
5. **Backwards Compatible**: Changes must be backwards compatible with existing clients

## Priority Matrix

Tasks are ranked using a composite score of Criticality (1-5) and Difficulty (1-5):
- **Priority Score** = (Criticality × 2) + (6 - Difficulty)
- Higher scores indicate higher priority (critical + easier to implement)

| Task | Criticality | Difficulty | Priority Score | Category |
|------|-------------|------------|----------------|----------|
| Implement Rate Limiting | 5 | 2 | 14 | 🔴 Critical |
| Add Input Validation Middleware | 5 | 2 | 14 | 🔴 Critical |
| Hash API Keys in Storage | 5 | 2 | 14 | 🔴 Critical |
| Add CSRF Protection | 4 | 2 | 12 | 🔴 Critical |
| Fix Transaction Rollback Patterns | 4 | 2 | 12 | 🟠 High |
| Add Request Correlation IDs | 4 | 2 | 12 | 🟠 High |
| Create Integration Tests | 4 | 3 | 11 | 🟠 High |
| Implement Circuit Breakers | 3 | 3 | 9 | 🟡 Medium |
| Add Comprehensive Logging | 3 | 2 | 10 | 🟡 Medium |
| Fix N+1 Query Problems | 3 | 3 | 9 | 🟡 Medium |
| Extract Service Layer | 2 | 4 | 6 | 🟢 Low |
| Add Metrics Collection | 2 | 3 | 7 | 🟢 Low |

## Phase 1: Critical Security Fixes (Week 1)

### 1.1 Implement Rate Limiting
**Criticality**: 5/5 | **Difficulty**: 2/5 | **Effort**: 1 day

**Current Issue**: No rate limiting exposes API to DDoS and brute force attacks

**Implementation**:
```go
// middleware/ratelimit.go
import "github.com/gin-gonic/gin"
import "golang.org/x/time/rate"

func RateLimitMiddleware(rps int) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(rps), rps*2)
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "Too many requests"})
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**Tasks**:
- [ ] Install rate limiting package
- [ ] Create rate limit middleware
- [ ] Configure per-endpoint limits
- [ ] Add IP-based rate limiting
- [ ] Implement rate limit headers

### 1.2 Add Input Validation Middleware ✅
**Criticality**: 5/5 | **Difficulty**: 2/5 | **Effort**: 2 days

**Current Issue**: Limited validation on inputs creates security vulnerabilities

**Implementation**:
```go
// middleware/validation.go
type ValidationRules struct {
    MaxStringLength int
    AllowedChars    string
    Required        []string
}

func ValidateInput(rules ValidationRules) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Implement validation logic
    }
}
```

**Tasks**:
- [x] Create validation middleware
- [x] Define validation rules per endpoint
- [x] Add SQL injection prevention
- [x] Implement XSS protection
- [x] Add request size limits

**Completed Implementation**:
- Created comprehensive validation middleware in `/libs/go/middleware/validation.go`
- Added specific validation rules for all major endpoints in `/libs/go/middleware/validation_rules.go`
- Integrated validation into API routes for products, customers, wallets, API keys, users, and subscriptions
- Added sanitization to prevent XSS attacks
- Implemented request size limits per endpoint
- Created unit tests to verify validation functionality
- Supports custom validation functions for complex validation logic

### 1.3 Hash API Keys in Storage ✅
**Criticality**: 5/5 | **Difficulty**: 2/5 | **Effort**: 1 day

**Current Issue**: API keys stored in plain text

**Implementation**:
```go
// Before storing: hash the key
hashedKey := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)

// Store only a prefix for identification
keyPrefix := apiKey[:8]
```

**Tasks**:
- [x] Create migration to add hashed_key column
- [x] Update API key generation logic
- [x] Implement key rotation mechanism
- [x] Update authentication middleware
- [x] Backfill existing keys (with rotation)

**Completed Implementation**:
- Created API key helper functions in `/libs/go/helpers/apikey.go` for generation and hashing
- Updated API key creation to generate secure random keys with bcrypt hashing
- Modified authentication middleware to use bcrypt comparison instead of direct lookup
- Added `key_prefix` column to store first part of key for identification
- Updated frontend .env files to use new admin key format
- Created setup script for generating admin key hash
- API now returns the full key only during creation, never again

### 1.4 Add CSRF Protection
**Criticality**: 4/5 | **Difficulty**: 2/5 | **Effort**: 1 day

**Current Issue**: No CSRF protection for state-changing operations

**Tasks**:
- [ ] Implement CSRF token generation
- [ ] Add CSRF validation middleware
- [ ] Configure token storage
- [ ] Update frontend integration docs
- [ ] Add CSRF headers to responses

## Phase 2: Stability Improvements (Week 2)

### 2.1 Fix Transaction Rollback Patterns ✅
**Criticality**: 4/5 | **Difficulty**: 2/5 | **Effort**: 1 day

**Current Issue**: Inconsistent transaction handling

**Implementation**:
```go
// helpers/transaction.go
func WithTransaction(ctx context.Context, db *pgxpool.Pool, fn func(*pgx.Tx) error) error {
    tx, err := db.Begin(ctx)
    if err != nil {
        return err
    }
    
    defer func() {
        if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
            logger.Error("Failed to rollback", zap.Error(err))
        }
    }()
    
    if err := fn(tx); err != nil {
        return err
    }
    
    return tx.Commit(ctx)
}
```

**Tasks**:
- [x] Create transaction wrapper helper
- [x] Update all handlers to use wrapper
- [x] Add transaction timeout handling
- [x] Implement retry logic for deadlocks
- [x] Add transaction metrics

**Completed Implementation**:
- Created transaction helpers in `/libs/go/helpers/transaction.go`
- Added `WithTransaction` for basic transaction handling with automatic rollback
- Added `WithTransactionRetry` for automatic retry on serialization failures
- Added `WithTransactionOptions` for custom isolation levels
- Created convenience methods in CommonServices (`RunInTransaction`, `RunInTransactionWithRetry`)
- Fixed circular import issue by removing helpers import from logger package
- Created documentation with refactoring examples in `/docs/transaction-refactor-example.md`

### 2.2 Add Request Correlation IDs
**Criticality**: 4/5 | **Difficulty**: 2/5 | **Effort**: 1 day

**Current Issue**: Cannot trace requests through the system

**Implementation**:
```go
// middleware/correlation.go
func CorrelationIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        correlationID := c.GetHeader("X-Correlation-ID")
        if correlationID == "" {
            correlationID = uuid.New().String()
        }
        
        c.Set("correlationID", correlationID)
        c.Header("X-Correlation-ID", correlationID)
        
        // Add to logger context
        ctx := logger.WithCorrelationID(c.Request.Context(), correlationID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Next()
    }
}
```

**Tasks**:
- [ ] Create correlation ID middleware
- [ ] Update logger to include correlation ID
- [ ] Propagate ID to external services
- [ ] Add to all log statements
- [ ] Update error responses

### 2.3 Create Integration Tests
**Criticality**: 4/5 | **Difficulty**: 3/5 | **Effort**: 3 days

**Current Issue**: No automated testing for API endpoints

**Structure**:
```
tests/
├── integration/
│   ├── auth_test.go
│   ├── customer_test.go
│   ├── product_test.go
│   └── subscription_test.go
├── fixtures/
│   ├── users.go
│   └── products.go
└── helpers/
    ├── db.go
    └── api.go
```

**Tasks**:
- [ ] Set up test database infrastructure
- [ ] Create test helpers and fixtures
- [ ] Write auth flow tests
- [ ] Write CRUD operation tests
- [ ] Add CI/CD integration

## Phase 3: Performance Optimization (Week 3)

### 3.1 Fix N+1 Query Problems
**Criticality**: 3/5 | **Difficulty**: 3/5 | **Effort**: 2 days

**Current Issue**: Inefficient database queries in list operations

**Tasks**:
- [ ] Identify N+1 queries using query logging
- [ ] Refactor to use JOINs or batch queries
- [ ] Add query performance tests
- [ ] Implement eager loading helpers
- [ ] Add database query analyzer

### 3.2 Implement Circuit Breakers
**Criticality**: 3/5 | **Difficulty**: 3/5 | **Effort**: 2 days

**Current Issue**: No protection against cascading failures

**Implementation**:
```go
// clients/circuitbreaker.go
import "github.com/sony/gobreaker"

func NewCircuitBreaker(name string) *gobreaker.CircuitBreaker {
    settings := gobreaker.Settings{
        Name:        name,
        MaxRequests: 5,
        Interval:    60 * time.Second,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures > 3
        },
    }
    return gobreaker.NewCircuitBreaker(settings)
}
```

**Tasks**:
- [ ] Add circuit breaker library
- [ ] Wrap external service calls
- [ ] Implement fallback mechanisms
- [ ] Add circuit breaker metrics
- [ ] Create health check endpoints

### 3.3 Add Comprehensive Logging
**Criticality**: 3/5 | **Difficulty**: 2/5 | **Effort**: 2 days

**Current Issue**: Inconsistent logging patterns

**Tasks**:
- [ ] Standardize log levels across codebase
- [ ] Add performance timing logs
- [ ] Implement log sampling
- [ ] Add structured logging fields
- [ ] Create log aggregation queries

## Phase 4: Code Quality (Week 4)

### 4.1 Extract Service Layer
**Criticality**: 2/5 | **Difficulty**: 4/5 | **Effort**: 5 days

**Current Issue**: Business logic mixed with HTTP handling

**Structure**:
```
services/
├── customer_service.go
├── product_service.go
├── subscription_service.go
└── transaction_service.go
```

**Tasks**:
- [ ] Design service interfaces
- [ ] Extract business logic from handlers
- [ ] Create service tests
- [ ] Update handler dependencies
- [ ] Document service contracts

### 4.2 Add Metrics Collection
**Criticality**: 2/5 | **Difficulty**: 3/5 | **Effort**: 2 days

**Current Issue**: No visibility into API performance

**Implementation**:
```go
// metrics/prometheus.go
var (
    httpDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_duration_seconds",
            Help: "Duration of HTTP requests.",
        },
        []string{"path", "method", "status"},
    )
)
```

**Tasks**:
- [ ] Add Prometheus client
- [ ] Create metrics middleware
- [ ] Add custom business metrics
- [ ] Set up Grafana dashboards
- [ ] Create alerting rules

## Implementation Timeline

### Week 1: Critical Security
- Monday-Tuesday: Rate limiting & input validation
- Wednesday: API key hashing
- Thursday-Friday: CSRF protection & testing

### Week 2: Stability
- Monday: Transaction patterns
- Tuesday: Correlation IDs
- Wednesday-Friday: Integration tests

### Week 3: Performance
- Monday-Tuesday: N+1 query fixes
- Wednesday-Thursday: Circuit breakers
- Friday: Logging improvements

### Week 4: Code Quality
- Full week: Service layer extraction & metrics

## Success Metrics

1. **Security**
   - 0 plain text API keys in database
   - 100% of endpoints rate limited
   - All inputs validated

2. **Reliability**
   - <1% error rate
   - <100ms p95 latency
   - 99.9% uptime

3. **Quality**
   - >80% test coverage
   - 0 critical security vulnerabilities
   - <10% code duplication

## Risk Mitigation

1. **Rollback Plan**: Each change should be feature-flagged
2. **Monitoring**: Add alerts before deploying changes
3. **Gradual Rollout**: Deploy to staging first
4. **Documentation**: Update API docs with each change

## Additional Recommendations

### Quick Wins (Can be done in parallel)
- Remove debug endpoints in production
- Add security headers (HSTS, CSP)
- Implement request timeouts
- Add health check endpoints
- Configure CORS properly for production

### Long-term Improvements
- Implement event sourcing for audit trails
- Add GraphQL API option
- Implement API versioning strategy
- Create SDK for API consumers
- Add WebSocket support for real-time updates

## Conclusion

This implementation plan provides a structured approach to improving the Cyphera API. By focusing on security first, then stability, performance, and finally code quality, we ensure that the most critical issues are addressed while building a solid foundation for future development.

The phased approach allows for continuous delivery of improvements while maintaining system stability. Each phase builds upon the previous one, creating a more robust and maintainable API.