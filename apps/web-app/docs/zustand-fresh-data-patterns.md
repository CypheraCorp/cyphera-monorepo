# Zustand with Fresh Data Patterns

## Overview

This document explains how to use Zustand for state management while ensuring fresh data from the backend on page refreshes and CRUD operations.

## Key Concepts

### 1. Separation of Concerns
- **Server State**: Use React Query (already implemented)
- **Client State**: Use Zustand
- **Never store server data in Zustand permanently**

### 2. Pattern Examples

#### ❌ Wrong: Storing Server Data in Zustand
```typescript
// DON'T DO THIS
const useProductStore = create((set) => ({
  products: [],
  fetchProducts: async () => {
    const products = await api.getProducts();
    set({ products }); // Products become stale over time
  }
}));
```

#### ✅ Correct: React Query for Server Data + Zustand for UI State
```typescript
// products/hooks/use-products.ts
export function useProducts() {
  return useQuery({
    queryKey: ['products'],
    queryFn: () => api.getProducts(),
    staleTime: 0, // Always fetch fresh data
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes
  });
}

// store/ui.ts - Only UI state in Zustand
const useUIStore = create((set) => ({
  selectedProductId: null,
  isCreateModalOpen: false,
  filters: { category: 'all', status: 'active' },
  setSelectedProduct: (id) => set({ selectedProductId: id }),
  setCreateModalOpen: (open) => set({ isCreateModalOpen: open }),
  setFilters: (filters) => set({ filters }),
}));
```

### 3. Fresh Data Patterns

#### Pattern 1: Invalidate on Mutations
```typescript
// When creating/updating/deleting, invalidate React Query cache
const createProduct = useMutation({
  mutationFn: (data) => api.createProduct(data),
  onSuccess: () => {
    // Invalidate and refetch products
    queryClient.invalidateQueries({ queryKey: ['products'] });
    // Update UI state only
    uiStore.setCreateModalOpen(false);
  },
});
```

#### Pattern 2: Optimistic Updates with Rollback
```typescript
const updateProduct = useMutation({
  mutationFn: (data) => api.updateProduct(data),
  onMutate: async (newData) => {
    // Cancel in-flight queries
    await queryClient.cancelQueries({ queryKey: ['products', newData.id] });
    
    // Snapshot previous value
    const previousProduct = queryClient.getQueryData(['products', newData.id]);
    
    // Optimistically update
    queryClient.setQueryData(['products', newData.id], newData);
    
    return { previousProduct };
  },
  onError: (err, newData, context) => {
    // Rollback on error
    queryClient.setQueryData(
      ['products', newData.id], 
      context.previousProduct
    );
  },
  onSettled: () => {
    // Always refetch after error or success
    queryClient.invalidateQueries({ queryKey: ['products'] });
  },
});
```

#### Pattern 3: Force Refresh Pattern
```typescript
// Add refresh functionality to your components
function ProductList() {
  const { data: products, refetch, isRefetching } = useProducts();
  
  return (
    <div>
      <Button onClick={() => refetch()} disabled={isRefetching}>
        <RefreshIcon /> Refresh
      </Button>
      {/* Product list */}
    </div>
  );
}
```

### 4. Zustand Store Best Practices

#### Auth Store Example (Session-based data that needs persistence)
```typescript
interface AuthStore {
  // State
  user: User | null;
  accessToken: string | null;
  
  // Actions
  setAuth: (user: User, token: string) => void;
  clearAuth: () => void;
  
  // Hydration flag
  hasHydrated: boolean;
  setHasHydrated: (state: boolean) => void;
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      hasHydrated: false,
      
      setAuth: (user, accessToken) => set({ user, accessToken }),
      clearAuth: () => set({ user: null, accessToken: null }),
      setHasHydrated: (state) => set({ hasHydrated: state }),
    }),
    {
      name: 'auth-storage',
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);
```

#### Wallet Store Example (Temporary state, fetched fresh)
```typescript
interface WalletStore {
  // UI State only
  selectedWalletId: string | null;
  isCreatingWallet: boolean;
  
  // Actions
  setSelectedWallet: (id: string | null) => void;
  setIsCreatingWallet: (state: boolean) => void;
  
  // No wallet data stored here!
}

// Actual wallet data comes from React Query
export function useWallets() {
  const { user } = useAuthStore();
  
  return useQuery({
    queryKey: ['wallets', user?.id],
    queryFn: () => api.getWallets(),
    enabled: !!user,
    staleTime: 0, // Always fresh
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
  });
}
```

### 5. Implementation Checklist

When migrating from Context to Zustand:

1. **Identify State Types**:
   - Server State → Keep in React Query
   - UI State → Move to Zustand
   - Form State → Consider React Hook Form or Zustand
   - Auth State → Zustand with persistence

2. **Set Proper Cache Times**:
   ```typescript
   // Critical data - always fresh
   staleTime: 0
   
   // Semi-static data - cache for 5 minutes
   staleTime: 5 * 60 * 1000
   
   // Static data - cache for 1 hour
   staleTime: 60 * 60 * 1000
   ```

3. **Add Refresh Mechanisms**:
   - Pull-to-refresh on mobile
   - Refresh buttons on data tables
   - Auto-refresh on window focus
   - Invalidate after mutations

4. **Handle Loading States**:
   ```typescript
   function MyComponent() {
     const { data, isLoading, isRefetching } = useData();
     const { someUIState } = useUIStore();
     
     if (isLoading) return <Skeleton />;
     
     return (
       <div>
         {isRefetching && <RefreshIndicator />}
         {/* Component content */}
       </div>
     );
   }
   ```

### 6. Common Pitfalls to Avoid

1. **Don't store server data in Zustand**
2. **Don't use Zustand as a cache** - Use React Query
3. **Don't forget to invalidate queries after mutations**
4. **Don't ignore loading states during refetch**
5. **Don't persist sensitive data** - Only UI preferences

### 7. Example Migration

#### Before (Context with stale data issues):
```typescript
const WalletContext = createContext();

function WalletProvider({ children }) {
  const [wallets, setWallets] = useState([]);
  
  useEffect(() => {
    fetchWallets().then(setWallets);
  }, []); // Only fetches once!
  
  return (
    <WalletContext.Provider value={{ wallets }}>
      {children}
    </WalletContext.Provider>
  );
}
```

#### After (React Query + Zustand):
```typescript
// hooks/use-wallets.ts
export function useWallets() {
  return useQuery({
    queryKey: ['wallets'],
    queryFn: api.getWallets,
    staleTime: 0,
    refetchOnMount: 'always',
  });
}

// store/wallet-ui.ts
export const useWalletUIStore = create((set) => ({
  selectedWalletId: null,
  isCreateModalOpen: false,
  setSelectedWallet: (id) => set({ selectedWalletId: id }),
  setCreateModalOpen: (open) => set({ isCreateModalOpen: open }),
}));

// components/wallet-list.tsx
function WalletList() {
  const { data: wallets, isLoading, refetch } = useWallets();
  const { selectedWalletId, setSelectedWallet } = useWalletUIStore();
  
  // Always fresh data!
  return <div>...</div>;
}
```

## Summary

- Use React Query for all server data (products, wallets, subscriptions, etc.)
- Use Zustand only for UI state and user preferences
- Set `staleTime: 0` for critical data that must be fresh
- Invalidate queries after mutations
- Add manual refresh options for users
- Don't persist server data in Zustand