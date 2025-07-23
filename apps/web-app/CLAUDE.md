# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cyphera Web is a crypto-based subscription platform built with Next.js 14 that enables businesses to create and manage subscription products using blockchain technology. It integrates Circle's Programmable Wallet technology and MetaMask's Delegation Toolkit.

## Essential Commands

### Development

```bash
# Start development server
npm run dev

# Build for production
npm run build

# Start production server
npm start

# Run linting
npm run lint

# Format code
npm run format

# Clean build artifacts
npm run clean
```

### Code Quality

Before committing changes, always run:

```bash
npm run lint
npm run format
```

## Architecture Overview

### Dual User System

The platform has two primary user types with separate flows:

- **Merchants**: Create and manage subscription products, view analytics
- **Customers**: Subscribe to products, manage payments

### Key Technical Components

1. **Wallet Infrastructure**
   - Circle Programmable Wallet SDK integration in `src/lib/circle/`
   - MetaMask Delegation Toolkit in `src/hooks/use-smart-accounts.ts`
   - Wallet context in `src/contexts/WalletContext.tsx`

2. **API Service Layer** (`src/services/`)
   - `cypheraApiService.ts`: Main API client with interceptors
   - `userService.ts`: User management operations
   - `subscriptionService.ts`: Subscription CRUD operations
   - `transactionService.ts`: Transaction management

3. **Authentication Flow**
   - Supabase authentication in `src/lib/supabase/`
   - Auth context in `src/contexts/AuthContext.tsx`
   - Protected routes via middleware

4. **Smart Account System**
   - Implementation in `src/hooks/use-smart-accounts.ts`
   - MetaMask Delegation Toolkit integration
   - Multi-network support configuration

### Page Structure

- `/app/(merchant)/` - Merchant dashboard and management pages
- `/app/(customer)/` - Customer subscription and payment pages
- `/app/auth/` - Authentication pages
- `/app/api/` - API routes

### Component Organization

- `src/components/ui/` - Shadcn UI components
- `src/components/wallet/` - Wallet-related components
- `src/components/merchant/` - Merchant-specific components
- `src/components/customer/` - Customer-specific components
- `src/components/subscription/` - Subscription management components

## Development Guidelines

### Working with Circle Wallet

- PIN management is critical - see `src/contexts/WalletContext.tsx`
- Wallet operations require proper error handling
- Check `/docs/circle-wallet-integration.md` for integration details

### API Integration

- All API calls go through `cypheraApiService`
- Use React Query hooks for data fetching
- Error handling is standardized via interceptors

### Type Safety

- Strict TypeScript enabled
- Types defined in `src/types/`
- Use Zod schemas for runtime validation

### Environment Variables

Required environment variables are documented in `.env.example`. Key variables include:

- `NEXT_PUBLIC_SUPABASE_*` - Supabase configuration
- `NEXT_PUBLIC_CIRCLE_*` - Circle API configuration
- `NEXT_PUBLIC_CYPHERA_API_URL` - Backend API URL
- `NEXT_PUBLIC_INFURA_API_KEY` - Blockchain provider

## Testing Approach

The project uses manual testing. Key areas to test:

- Wallet creation and PIN management
- Transaction flows
- Subscription creation and management
- Multi-network operations

## Important Notes

- Always handle wallet operations with proper error boundaries
- Circle API has rate limits - implement proper retry logic
- MetaMask Delegation requires user approval for operations
- Check `/docs/` directory for detailed flow diagrams
