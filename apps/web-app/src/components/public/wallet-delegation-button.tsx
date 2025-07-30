'use client';

import { SmartAccountDelegationButton } from '@/components/delegation/smart-account-delegation-button';
import { useWagmiSmartAccountProvider } from '@/components/delegation/providers/wagmi-provider';

interface WalletDelegationButtonProps {
  disabled?: boolean;
}

/**
 * Wagmi-based delegation button component
 * Now uses the consolidated SmartAccountDelegationButton with Wagmi provider
 */
export function WalletDelegationButton({ disabled = false }: WalletDelegationButtonProps) {
  const provider = useWagmiSmartAccountProvider();

  return (
    <SmartAccountDelegationButton
      provider={provider}
      mode="delegation"
      disabled={disabled}
      priceId="" // Not used in delegation mode
    />
  );
}
