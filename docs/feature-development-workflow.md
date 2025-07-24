# Feature Development Workflow

This document outlines the standard workflow for developing new features in the Cyphera platform, ensuring consistency, quality, and proper testing across both backend and frontend components.

## Overview

The feature development workflow follows a systematic approach:

1. **Backend Implementation** → 2. **Backend Testing** → 3. **Frontend Implementation** → 4. **Frontend Testing** → 5. **E2E Testing** → 6. **Documentation**

## Detailed Workflow

### Phase 1: Backend Implementation

#### 1.1 Planning
- Review requirements and create tasks in the todo list
- Identify API endpoints needed
- Plan database schema changes if required
- Consider security implications (authentication, authorization, validation)

#### 1.2 Database Changes
```bash
# Update schema if needed
vi libs/go/db/schema.sql

# Update queries
vi libs/go/db/queries/*.sql

# Generate SQLC code
make gen
```

#### 1.3 API Implementation
1. **Add validation schemas** in `/libs/go/middleware/validation.go`
2. **Create/update handlers** in `/apps/api/handlers/`
3. **Add middleware** (rate limiting, validation, etc.)
4. **Update routes** in `/apps/api/server/server.go`

#### 1.4 Backend Verification
```bash
# Build the API
go build -o main ./apps/api/cmd/main

# Run linting
golangci-lint run ./...

# Run tests
go test ./...

# Start the API server
make dev

# Test endpoints manually
curl -X POST http://localhost:8000/api/v1/endpoint \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'
```

### Phase 2: Frontend Implementation

#### 2.1 Planning
- Design UI/UX for the feature
- Identify components to create/modify
- Plan state management approach
- Consider error handling and loading states

#### 2.2 Component Development
1. **Create API route handlers** in `/apps/web-app/src/app/api/`
2. **Build UI components** in `/apps/web-app/src/components/`
3. **Add type definitions** in `/apps/web-app/src/types/`
4. **Implement hooks** if needed in `/apps/web-app/src/hooks/`
5. **Handle errors** with correlation IDs for debugging

#### 2.3 Frontend Verification
```bash
# Install dependencies
npm install

# Build to check for TypeScript errors
npm run build

# Run development server
npm run dev

# Check for linting issues
npm run lint

# Format code
npm run format
```

### Phase 3: End-to-End Testing

#### 3.1 Write E2E Tests
Create test files in `/apps/web-app/tests/e2e/` following this structure:

```typescript
import { test, expect } from '@playwright/test';

test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    // Setup steps
  });

  test('should perform main user flow', async ({ page }) => {
    // Test implementation
  });

  test('should handle error cases', async ({ page }) => {
    // Error scenario testing
  });
});
```

#### 3.2 Run E2E Tests
```bash
# Install Playwright browsers (first time only)
npx playwright install

# Run all tests
npm run test:e2e

# Run tests with UI
npm run test:e2e:ui

# Debug specific test
npm run test:e2e:debug

# Generate test code
npm run test:e2e:codegen
```

### Phase 4: Documentation

#### 4.1 Update Documentation
- Update API documentation (Swagger annotations)
- Update relevant markdown files in `/docs/`
- Add inline code comments for complex logic
- Update CLAUDE.md if adding new patterns

#### 4.2 Commit Changes
```bash
# Stage changes
git add -A

# Commit with descriptive message
git commit -m "feat: implement [feature name]

- Add backend API endpoints for [feature]
- Implement frontend UI components
- Add comprehensive E2E tests
- Include correlation ID support
- Update documentation

[Any additional notes or breaking changes]"

# Push to branch
git push origin feature-branch
```

## Example: API Key Management Feature

Here's how we implemented the API key management feature following this workflow:

### 1. Backend Implementation
- Added API key hashing in `/libs/go/helpers/apikey.go`
- Created validation rules in `/libs/go/middleware/validation.go`
- Updated handlers in `/apps/api/handlers/apikey_handlers.go`
- Added rate limiting to sensitive endpoints

### 2. Frontend Implementation
- Created API routes in `/apps/web-app/src/app/api/api-keys/`
- Built UI component in `/apps/web-app/src/components/settings/api-keys-tab.tsx`
- Added to settings page in `/apps/web-app/src/components/settings/settings-form.tsx`
- Implemented correlation ID support for debugging

### 3. E2E Testing
- Created comprehensive test in `/apps/web-app/tests/e2e/api-keys.spec.ts`
- Tests full user flow: create, view, use, and delete API keys
- Includes validation error testing

### 4. Security Considerations
- API keys are hashed with bcrypt before storage
- Keys are only shown once during creation
- Rate limiting applied to prevent brute force
- Input validation prevents injection attacks

## Best Practices

### Security First
- Always validate input on both frontend and backend
- Use proper authentication and authorization
- Hash sensitive data before storage
- Implement rate limiting for sensitive operations
- Log security events with correlation IDs

### Error Handling
- Provide meaningful error messages
- Include correlation IDs in error responses
- Handle edge cases gracefully
- Test error scenarios in E2E tests

### Performance
- Use database transactions appropriately
- Implement pagination for list endpoints
- Add caching where beneficial
- Monitor API response times

### Code Quality
- Follow existing code patterns
- Write self-documenting code
- Add tests for critical paths
- Run linters before committing
- Keep components focused and reusable

## Task Completion Criteria

For a feature to be considered complete, it must meet ALL of these criteria:

1. **No Breaking Changes**: Existing functionality continues to work
2. **No New Bugs**: Comprehensive testing ensures stability
3. **Build Success**: Both API and frontend build without errors
4. **No Linting Issues**: All code passes linting checks
5. **Backwards Compatible**: Changes work with existing clients
6. **E2E Tests Pass**: User flows are verified through automated tests
7. **Documentation Updated**: All relevant docs reflect the changes

## Troubleshooting

### Common Issues

1. **Build Failures**
   - Check for TypeScript errors: `npm run type-check`
   - Verify all imports are correct
   - Ensure environment variables are set

2. **API Connection Issues**
   - Verify API is running: `curl http://localhost:8000/health`
   - Check CORS configuration
   - Validate authentication headers

3. **E2E Test Failures**
   - Ensure both frontend and backend are running
   - Check test selectors match current UI
   - Verify test data is properly cleaned up

4. **Database Issues**
   - Run migrations: `make migrate`
   - Check connection string in `.env`
   - Verify PostgreSQL is running

## Tools and Commands Reference

### Backend
```bash
make dev                    # Run API with hot reload
make gen                    # Generate SQLC code
make test                   # Run tests
make lint                   # Run linter
make build                  # Build API binary
```

### Frontend
```bash
npm run dev                 # Start development server
npm run build              # Build for production
npm run lint               # Run ESLint
npm run format             # Format with Prettier
npm run test:e2e           # Run E2E tests
```

### Database
```bash
docker-compose up postgres  # Start PostgreSQL
make migrate               # Run migrations
make reset-db              # Reset database
```

## Conclusion

Following this workflow ensures that every feature is properly implemented, tested, and documented. The systematic approach reduces bugs, improves code quality, and makes the codebase more maintainable.