import { useEffect, useState } from 'react';
import { useWeb3Auth, useSwitchChain } from '@web3auth/modal/react';
import { logger } from '@/lib/core/logger/logger-utils';
import { useToast } from '@/components/ui/use-toast';

interface Web3AuthProvider {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
}

/**
 * Hook that automatically switches to a required network after Web3Auth login
 * This runs at a higher level than individual components to ensure network is ready
 */
export function useWeb3AuthAutoNetwork(requiredChainId?: number, networkName?: string) {
  const { web3Auth, isConnected, status } = useWeb3Auth();
  const { switchChain } = useSwitchChain();
  const { toast } = useToast();
  const [isNetworkReady, setIsNetworkReady] = useState(false);
  const [isSwitching, setIsSwitching] = useState(false);

  useEffect(() => {
    if (!isConnected || !requiredChainId || !web3Auth?.provider || !switchChain || isSwitching) {
      return;
    }

    const checkAndSwitchNetwork = async () => {
      try {
        // Small delay to ensure Web3Auth provider is fully ready after login
        await new Promise((resolve) => setTimeout(resolve, 1000));

        // Get current chain ID
        let currentChainIdDecimal: number;
        try {
          const currentChainId = (await (web3Auth.provider as Web3AuthProvider).request({
            method: 'eth_chainId',
          })) as string;
          currentChainIdDecimal = parseInt(currentChainId, 16);
        } catch (error) {
          logger.error('Failed to get current chain ID:', { error });
          return;
        }

        logger.log('🔍 [useWeb3AuthAutoNetwork] Network check on connect:', {
          requiredChainId,
          currentChainId: currentChainIdDecimal,
          networkName,
          needsSwitch: currentChainIdDecimal !== requiredChainId,
        });

        if (currentChainIdDecimal !== requiredChainId) {
          setIsSwitching(true);
          logger.log(`🔄 Auto-switching from chain ${currentChainIdDecimal} to ${requiredChainId}`);

          try {
            const hexChainId = `0x${requiredChainId.toString(16)}`;
            await switchChain(hexChainId);

            // Poll for network change confirmation
            let retries = 0;
            const maxRetries = 20;
            let verifiedChainId = currentChainIdDecimal;

            while (verifiedChainId !== requiredChainId && retries < maxRetries) {
              await new Promise(resolve => setTimeout(resolve, Math.min(200 * Math.pow(1.5, retries), 2000)));

              try {
                const checkChainId = (await (web3Auth.provider as Web3AuthProvider).request({
                  method: 'eth_chainId',
                })) as string;
                verifiedChainId = parseInt(checkChainId, 16);

                logger.log(`[useWeb3AuthAutoNetwork] Verification ${retries + 1}/${maxRetries}:`, {
                  verifiedChainId,
                  requiredChainId,
                });
              } catch (checkError) {
                logger.warn('Failed to verify chain ID:', checkError as Record<string, unknown>);
              }

              retries++;
            }

            if (verifiedChainId === requiredChainId) {
              logger.log('✅ Successfully auto-switched to required network');
              setIsNetworkReady(true);
              
              toast({
                title: 'Network Ready',
                description: `Connected to ${networkName || `Chain ${requiredChainId}`}`,
              });
            } else {
              logger.warn('⚠️ Auto-switch verification failed');
              toast({
                title: 'Network Switch Required',
                description: `Please manually switch to ${networkName || `Chain ${requiredChainId}`}`,
                variant: 'destructive',
              });
            }
          } catch (error) {
            logger.error('❌ Auto-switch failed:', { error });
          } finally {
            setIsSwitching(false);
          }
        } else {
          logger.log('✅ Already on correct network');
          setIsNetworkReady(true);
        }
      } catch (error) {
        logger.error('Error in auto network check:', { error });
      }
    };

    // Run check when Web3Auth connects or status changes to 'connected'
    if (status === 'connected' || (isConnected && status === 'ready')) {
      checkAndSwitchNetwork();
    }
  }, [isConnected, status, requiredChainId, networkName, web3Auth?.provider, switchChain, toast, isSwitching]);

  return { isNetworkReady, isSwitching };
}