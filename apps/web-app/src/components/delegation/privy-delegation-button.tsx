'use client';

import { usePrivySmartAccount } from '@/hooks/privy/use-privy-smart-account';
import { SmartAccountDelegationButton } from './smart-account-delegation-button';
import { PrivySmartAccountProvider } from './providers/privy-smart-account-provider';
import type { SubscriptionParams } from './types';

interface PrivyDelegationButtonProps extends SubscriptionParams {
  mode?: 'delegation' | 'subscription';
  disabled?: boolean;
  variant?: 'default' | 'outline';
  className?: string;
}

/**
 * Privy-specific delegation button that uses Privy's embedded wallet
 * to create smart accounts and sign delegations with MetaMask toolkit
 */
export function PrivyDelegationButton(props: PrivyDelegationButtonProps) {
  const privySmartAccount = usePrivySmartAccount();
  
  // Create provider instance
  const provider = new PrivySmartAccountProvider(privySmartAccount);
  
  // Use the existing SmartAccountDelegationButton with Privy provider
  return (
    <SmartAccountDelegationButton
      {...props}
      provider={provider}
    />
  );
}