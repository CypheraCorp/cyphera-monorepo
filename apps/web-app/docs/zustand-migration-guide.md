# Zustand Migration Guide

This guide shows how to migrate components from Context API to Zustand stores with fresh data patterns.

## Migration Examples

### 1. Auth Migration

#### Before (Context):
```typescript
import { useAuth } from '@/contexts/auth-context';

function MyComponent() {
  const { auth, loading, error, refreshAuth } = useAuth();
  
  if (loading) return <Spinner />;
  if (!auth.isAuthenticated) return <Login />;
  
  return <div>Welcome {auth.user?.email}</div>;
}
```

#### After (Zustand + React Query):
```typescript
import { useAuth } from '@/hooks/auth/use-auth-user';

function MyComponent() {
  const { isAuthenticated, user, loading, error, refetch } = useAuth();
  
  if (loading) return <Spinner />;
  if (!isAuthenticated) return <Login />;
  
  return <div>Welcome {user?.email}</div>;
}
```

### 2. Wallet Migration

#### Before (Context):
```typescript
import { useWalletContext } from '@/contexts/wallet-context';

function WalletList() {
  const { wallets, loading, selectedWallet, setSelectedWallet } = useWalletContext();
  
  return (
    <div>
      {wallets.map(wallet => (
        <WalletCard 
          key={wallet.id}
          wallet={wallet}
          isSelected={wallet.id === selectedWallet?.id}
          onSelect={() => setSelectedWallet(wallet)}
        />
      ))}
    </div>
  );
}
```

#### After (Zustand + React Query):
```typescript
import { useWalletsWithUI } from '@/hooks/wallets/use-wallets-data';

function WalletList() {
  const { 
    wallets, 
    isLoading, 
    selectedWalletId, 
    setSelectedWallet,
    refetch 
  } = useWalletsWithUI();
  
  if (isLoading) return <WalletListSkeleton />;
  
  return (
    <div>
      <Button onClick={() => refetch()}>Refresh</Button>
      {wallets.map(wallet => (
        <WalletCard 
          key={wallet.id}
          wallet={wallet}
          isSelected={wallet.id === selectedWalletId}
          onSelect={() => setSelectedWallet(wallet.id)}
        />
      ))}
    </div>
  );
}
```

### 3. Network Migration

#### Before (Context):
```typescript
import { useNetwork } from '@/contexts/network-context';

function NetworkSelector() {
  const { 
    networks, 
    currentNetwork, 
    switchNetwork, 
    isSwitching 
  } = useNetwork();
  
  return (
    <Select 
      value={currentNetwork?.chainId}
      onValueChange={(chainId) => switchNetwork(Number(chainId))}
      disabled={isSwitching}
    >
      {networks.map(network => (
        <SelectItem key={network.chainId} value={network.chainId}>
          {network.name}
        </SelectItem>
      ))}
    </Select>
  );
}
```

#### After (Zustand + React Query):
```typescript
import { useNetworksWithUI } from '@/hooks/networks/use-networks-data';
import { useSwitchNetwork } from 'wagmi';

function NetworkSelector() {
  const { 
    networks, 
    currentChainId, 
    isNetworkSelectorOpen,
    setNetworkSelectorOpen,
    startNetworkSwitch,
    endNetworkSwitch,
    isSwitchingNetwork
  } = useNetworksWithUI();
  
  const { switchNetwork } = useSwitchNetwork({
    onMutate: (chainId) => startNetworkSwitch(chainId),
    onSettled: () => endNetworkSwitch(),
  });
  
  return (
    <Select 
      value={currentChainId?.toString()}
      onValueChange={(chainId) => switchNetwork?.(Number(chainId))}
      disabled={isSwitchingNetwork}
      open={isNetworkSelectorOpen}
      onOpenChange={setNetworkSelectorOpen}
    >
      {networks.map(network => (
        <SelectItem 
          key={network.network.chain_id} 
          value={network.network.chain_id.toString()}
        >
          {network.network.name}
        </SelectItem>
      ))}
    </Select>
  );
}
```

### 4. Create Product Dialog Migration

#### Before (Prop Drilling):
```typescript
function CreateProductDialog({ 
  open, 
  onOpenChange, 
  networks, 
  wallets,
  onSuccess 
}) {
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState({});
  
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <Step1 
        data={formData} 
        onChange={setFormData}
        networks={networks}
        wallets={wallets}
        onNext={() => setStep(2)}
      />
      {/* More prop drilling... */}
    </Dialog>
  );
}
```

#### After (Zustand):
```typescript
// store/create-product.ts
export const useCreateProductStore = create((set) => ({
  isOpen: false,
  step: 1,
  formData: {},
  setOpen: (open) => set({ isOpen: open }),
  setStep: (step) => set({ step }),
  updateFormData: (data) => set((state) => ({ 
    formData: { ...state.formData, ...data } 
  })),
  reset: () => set({ step: 1, formData: {} }),
}));

// Component
function CreateProductDialog() {
  const { isOpen, setOpen, step } = useCreateProductStore();
  const { networks } = useNetworksWithUI();
  const { wallets } = useWalletsWithUI();
  
  return (
    <Dialog open={isOpen} onOpenChange={setOpen}>
      <Step1 /> {/* No props needed! */}
    </Dialog>
  );
}

function Step1() {
  const { formData, updateFormData, setStep } = useCreateProductStore();
  const { networks } = useNetworksWithUI();
  
  // Direct access to stores, no prop drilling!
  return <div>...</div>;
}
```

## Migration Checklist

When migrating a component:

1. **Identify State Types**
   - [ ] Server data → Use React Query hooks
   - [ ] UI state → Use Zustand stores
   - [ ] Form state → Consider React Hook Form or Zustand

2. **Update Imports**
   ```typescript
   // Remove
   import { useAuth } from '@/contexts/auth-context';
   
   // Add
   import { useAuth } from '@/hooks/auth/use-auth-user';
   ```

3. **Add Fresh Data Features**
   - [ ] Add refresh buttons where appropriate
   - [ ] Show loading states during refetch
   - [ ] Handle errors gracefully

4. **Remove Context Providers**
   ```typescript
   // Before
   <AuthProvider>
     <NetworkProvider>
       <WalletProvider>
         <App />
       </WalletProvider>
     </NetworkProvider>
   </AuthProvider>
   
   // After
   <App /> // No providers needed!
   ```

5. **Test Data Freshness**
   - [ ] Page refresh shows latest data
   - [ ] CRUD operations update immediately
   - [ ] No stale data issues

## Common Patterns

### Pattern 1: Combine Multiple Stores
```typescript
function MyComponent() {
  const { user } = useAuth();
  const { currentNetwork } = useNetworksWithUI();
  const { selectedWallet } = useWalletsWithUI();
  const { theme } = useUIStore();
  
  // Use data from multiple stores
}
```

### Pattern 2: Derived State
```typescript
function MyComponent() {
  const { wallets } = useWalletsWithUI();
  const { currentChainId } = useNetworksWithUI();
  
  // Derive state from multiple sources
  const walletsOnCurrentNetwork = wallets.filter(
    w => w.chainId === currentChainId
  );
}
```

### Pattern 3: Actions Across Stores
```typescript
function LogoutButton() {
  const { logout } = useAuthStore();
  const { reset: resetWalletUI } = useWalletUIStore();
  const { reset: resetNetworkUI } = useNetworkUIStore();
  const queryClient = useQueryClient();
  
  const handleLogout = async () => {
    logout();
    resetWalletUI();
    resetNetworkUI();
    queryClient.clear(); // Clear all cached data
  };
  
  return <Button onClick={handleLogout}>Logout</Button>;
}
```

## Benefits After Migration

1. **No Provider Hell**: Zero nested providers
2. **No Prop Drilling**: Direct store access from any component
3. **Always Fresh Data**: React Query ensures latest data
4. **Better Performance**: Selective subscriptions reduce re-renders
5. **Persistent Preferences**: User settings survive page refresh
6. **Cleaner Code**: Less boilerplate, more readable

## Troubleshooting

### Issue: Stale data after mutation
**Solution**: Always invalidate queries after mutations
```typescript
onSuccess: () => {
  queryClient.invalidateQueries({ queryKey: ['products'] });
}
```

### Issue: Hydration mismatch
**Solution**: Check hydration state before rendering
```typescript
const { hasHydrated } = useAuthStore();
if (!hasHydrated) return null;
```

### Issue: Lost state on refresh
**Solution**: Only persist UI state, not server data
```typescript
persist(
  (set) => ({ ... }),
  {
    partialize: (state) => ({
      // Only UI state
      theme: state.theme,
      selectedId: state.selectedId,
      // NOT server data
      // products: state.products, ❌
    })
  }
)
```