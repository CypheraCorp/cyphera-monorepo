/**
 * Consolidated delegation components and utilities
 */

// Main consolidated component
export { SmartAccountDelegationButton } from './smart-account-delegation-button';

// Provider implementations
export { useWagmiSmartAccountProvider } from './providers/wagmi-provider';
export { useWeb3AuthSmartAccountProvider } from './providers/web3auth-provider';
export { usePrivySmartAccountProvider } from './providers/privy-provider';

// Types
export type {
  SmartAccountProvider,
  SmartAccountProviderType,
  SmartAccountState,
  SmartAccountActions,
  DelegationStatus,
  SubscriptionParams,
  SubscriptionInfo,
  DelegationResult,
} from './types';

// Example of how to create a custom delegation button for any provider:
/*
import { SmartAccountDelegationButton, useWeb3AuthSmartAccountProvider } from '@/components/delegation';

function MyCustomSubscriptionButton() {
  const provider = useWeb3AuthSmartAccountProvider();
  
  return (
    <SmartAccountDelegationButton
      provider={provider}
      mode="subscription"
      priceId="price_123"
      productName="Premium Plan"
      // ... other props
    />
  );
}
*/