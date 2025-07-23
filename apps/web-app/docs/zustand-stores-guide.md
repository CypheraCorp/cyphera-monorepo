# Zustand Stores Guide

## Overview

This guide documents all Zustand stores in the Cyphera web application, their purpose, and usage patterns.

## Store Architecture

### Core Principle: Separation of Concerns

- **Zustand**: UI state, user preferences, temporary state
- **React Query**: Server data (products, wallets, subscriptions, etc.)
- **Never mix the two**: Server data should not be stored in Zustand

## Store Categories

### 1. Core Stores

#### Auth Store (`/store/auth.ts`)
**Purpose**: Authentication state and tokens
**Persisted**: Yes (tokens only)
**Key State**:
- `isAuthenticated`: Boolean auth status
- `accessToken`, `refreshToken`: Auth tokens
- `hasHydrated`: Hydration status for SSR

**Usage**:
```typescript
import { useAuth } from '@/hooks/auth/use-auth-user';

// This hook combines auth store + fresh user data
const { isAuthenticated, user, login, logout } = useAuth();
```

#### UI Store (`/store/ui.ts`)
**Purpose**: Global UI preferences
**Persisted**: Yes (user preferences)
**Key State**:
- `theme`: User's theme preference
- `sidebarCollapsed`: Sidebar state
- `activeModal`: Current modal
- `userType`: merchant/customer
- `preferredCurrency`: Default currency

**Usage**:
```typescript
import { useUIStore } from '@/store';

const theme = useUIStore((state) => state.theme);
const setTheme = useUIStore((state) => state.setTheme);
```

### 2. Domain UI Stores

#### Wallet UI Store (`/store/wallet-ui.ts`)
**Purpose**: Wallet-related UI state
**Persisted**: No
**Key State**:
- `selectedWalletId`: Currently selected wallet
- `isCreateModalOpen`: Create wallet modal state
- `viewMode`: Grid/list view preference
- `filters`: Active filters

#### Network UI Store (`/store/network-ui.ts`)
**Purpose**: Network selection UI
**Persisted**: Yes (preferences only)
**Key State**:
- `preferredChainId`: User's preferred network
- `showTestnets`: Display testnet toggle
- `isSwitchingNetwork`: Network switch in progress

#### Product UI Store (`/store/product-ui.ts`)
**Purpose**: Product management UI
**Persisted**: No
**Key State**:
- `selectedProductId`: Selected product
- `viewMode`: Display mode
- `filters`: Active filters
- `bulkActionMode`: Bulk selection state

#### Subscription UI Store (`/store/subscription-ui.ts`)
**Purpose**: Subscription management UI
**Persisted**: No
**Key State**:
- `selectedSubscriptionId`: Selected subscription
- `filters`: Status, date range filters
- `cancelModalOpen`: Cancel modal state
- `sortBy`, `sortOrder`: Sort preferences

#### Transaction UI Store (`/store/transaction-ui.ts`)
**Purpose**: Transaction list UI
**Persisted**: Yes (view preferences)
**Key State**:
- `viewMode`: Table/cards view
- `itemsPerPage`: Pagination preference
- `filters`: Transaction filters
- `exportFormat`: Export preferences

#### Customer UI Store (`/store/customer-ui.ts`)
**Purpose**: Customer-specific UI preferences
**Persisted**: Yes (preferences)
**Key State**:
- `dashboardLayout`: Layout preference
- `marketplace`: Search and filter state
- `preferredPaymentMethod`: Default payment
- `showBalanceInUSD`: Display preference

### 3. Feature Stores

#### Create Product Store (`/store/create-product.ts`)
**Purpose**: Multi-step product creation form
**Persisted**: No
**Key State**:
- `currentStep`: Form step
- `formData`: Product form data
- `validatedSteps`: Completed steps
- `isCreating`: Submit state

## Usage Patterns

### 1. Simple State Access
```typescript
// Direct hook usage
const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
const logout = useAuthStore((state) => state.logout);
```

### 2. Using Selectors
```typescript
// Pre-defined selectors for common use cases
import { useSelectedWallet, useWalletFilters } from '@/store';

const selectedWalletId = useSelectedWallet();
const filters = useWalletFilters();
```

### 3. Combining with React Query
```typescript
import { useWalletsWithUI } from '@/hooks/wallets/use-wallets-data';

function WalletList() {
  // This hook combines wallet data (React Query) with UI state (Zustand)
  const { 
    wallets,        // Fresh data from API
    isLoading,      // Loading state
    selectedWalletId,  // UI selection state
    setSelectedWallet, // UI action
    refetch         // Refresh data
  } = useWalletsWithUI();
}
```

### 4. Subscribing to Specific State
```typescript
// Only re-render when theme changes
const theme = useUIStore((state) => state.theme);

// Subscribe to multiple values
const { isOpen, currentStep } = useCreateProductStore((state) => ({
  isOpen: state.isOpen,
  currentStep: state.currentStep,
}));
```

### 5. Using Actions
```typescript
const { openEditModal, closeEditModal } = useProductUIStore();

// In component
<Button onClick={() => openEditModal(product.id)}>
  Edit
</Button>
```

## Best Practices

### 1. State Organization

✅ **Do**:
- Keep stores focused on a single domain
- Use clear, descriptive action names
- Implement reset functions for cleanup
- Add selectors for commonly used state

❌ **Don't**:
- Store server data in Zustand
- Create deeply nested state
- Mix unrelated concerns in one store

### 2. Persistence

Only persist:
- User preferences (theme, layout, view modes)
- Auth tokens
- Settings that should survive page refresh

Never persist:
- Server data (use React Query)
- Modal states
- Temporary UI state

### 3. Performance

```typescript
// ❌ Bad: Creates new object every render
const state = useStore((state) => ({
  a: state.a,
  b: state.b,
}));

// ✅ Good: Use multiple selectors
const a = useStore((state) => state.a);
const b = useStore((state) => state.b);

// ✅ Good: Or use shallow equality
import { shallow } from 'zustand/shallow';
const { a, b } = useStore(
  (state) => ({ a: state.a, b: state.b }),
  shallow
);
```

### 4. TypeScript

Always type your stores:
```typescript
interface StoreState {
  count: number;
}

interface StoreActions {
  increment: () => void;
  decrement: () => void;
}

const useStore = create<StoreState & StoreActions>()(...)
```

## Migration from Context

When migrating from Context API:

1. **Identify state type**:
   - Server data → React Query
   - UI state → Zustand

2. **Create appropriate store**:
   ```typescript
   // Before: Context with mixed concerns
   const WalletContext = createContext({
     wallets: [],      // Server data
     selectedId: null, // UI state
   });

   // After: Proper separation
   // Zustand for UI
   const useWalletUIStore = create(() => ({
     selectedId: null,
   }));

   // React Query for data
   const useWallets = () => useQuery({
     queryKey: ['wallets'],
     queryFn: fetchWallets,
   });
   ```

3. **Update components**:
   ```typescript
   // Before
   const { wallets, selectedId } = useContext(WalletContext);

   // After
   const { wallets } = useWallets();
   const selectedId = useWalletUIStore((s) => s.selectedId);
   ```

## Debugging

In development, all stores are available on window:

```javascript
// View all stores
window.__ZUSTAND_STORES__

// Get current state
window.__ZUSTAND_STORES__.auth.getState()

// Reset all stores
window.__resetAllStores()

// Subscribe to changes
const unsubscribe = window.__ZUSTAND_STORES__.ui.subscribe(
  (state) => console.log('UI state changed:', state)
)
```

## Store Reference

| Store | Purpose | Persisted | Key Features |
|-------|---------|-----------|--------------|
| auth | Authentication | Tokens only | Login/logout, token management |
| ui | Global UI prefs | Yes | Theme, sidebar, modals |
| wallet-ui | Wallet UI | No | Selection, filters, modals |
| network-ui | Network UI | Preferences | Chain selection, testnets |
| product-ui | Product UI | No | CRUD modals, bulk actions |
| subscription-ui | Subscription UI | No | Filters, cancel flow |
| transaction-ui | Transaction UI | View prefs | Export, pagination |
| customer-ui | Customer prefs | Yes | Dashboard, marketplace |
| create-product | Product form | No | Multi-step form state |

## Common Patterns

### Loading States
```typescript
// Combine Zustand UI loading with React Query data loading
const isCreating = useProductUIStore((s) => s.isCreating);
const { isLoading: isDataLoading } = useProducts();
const isLoading = isCreating || isDataLoading;
```

### Modal Management
```typescript
// Centralized modal state
const { activeModal, openModal, closeModal } = useUIStore();

// Domain-specific modals
const { editModalOpen, openEditModal } = useProductUIStore();
```

### Filters and Search
```typescript
// Combine UI filters with data fetching
const filters = useProductFilters();
const { data } = useProducts({ filters });
```

This architecture ensures clean separation of concerns, optimal performance, and excellent developer experience.