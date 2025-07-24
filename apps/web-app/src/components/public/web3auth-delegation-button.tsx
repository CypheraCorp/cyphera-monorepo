'use client';

import { useState, useEffect } from 'react';
import { Hex, type Address, formatUnits } from 'viem';
import { useWeb3AuthSmartAccount } from '@/hooks/auth';
import { useWeb3AuthInitialization } from '@/hooks/auth';
import { useWeb3Auth } from '@web3auth/modal/react';
import { formatDelegation } from '@/lib/web3/utils/delegation';
import { createDelegation, MetaMaskSmartAccount } from '@metamask/delegation-toolkit';
import { Button } from '@/components/ui/button';
import { useToast } from '@/components/ui/use-toast';
import { useEnvConfig } from '@/components/env/client';
import { useNetworkStore } from '@/store/network';
import { Loader2, CheckCircle, ExternalLink, FileText } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

import { Checkbox } from '@/components/ui/checkbox';
import { generateExplorerLink } from '@/lib/utils/explorers';
import { NetworkWithTokensResponse } from '@/types/network';
import { SubscriptionResponse } from '@/types/subscription';
import { logger } from '@/lib/core/logger/logger-utils';
// Type for Web3Auth provider
interface Web3AuthProvider {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
}

// Get delegate address from env or API
async function getCypheraDelegateAddress(envDelegateAddress?: string): Promise<`0x${string}`> {
  try {
    // First check if we have the address from our environment config
    if (envDelegateAddress?.startsWith('0x')) return envDelegateAddress as `0x${string}`;

    // Fallback to API request if not available in config
    const response = await fetch('/api/config/delegate-address');
    const data = await response.json();
    if (!data.success || !data.address) throw new Error('Failed to get delegate address');
    return data.address as `0x${string}`;
  } catch (error) {
    logger.error('Error getting delegate address:', { error });
    throw new Error('Cyphera delegate address is not configured');
  }
}

// Helper function to get chain ID from network name
async function getChainIdFromNetworkName(networkName: string): Promise<number | null> {
  try {
    const response = await fetch('/api/networks?active=true');
    if (!response.ok) return null;

    const networks: NetworkWithTokensResponse[] = await response.json();
    const network = networks.find(
      (n) => n.network.name.toLowerCase() === networkName.toLowerCase()
    );

    return network?.network.chain_id || null;
  } catch (error) {
    logger.error('Failed to get chain ID from network name:', { error });
    return null;
  }
}

// Helper function to switch network using Web3Auth
async function switchToNetwork(
  web3AuthProvider: Web3AuthProvider,
  targetChainId: number
): Promise<void> {
  try {
    const hexChainId = `0x${targetChainId.toString(16)}`;

    // First try to switch to the network
    try {
      await web3AuthProvider.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: hexChainId }],
      });
      logger.log(`✅ Successfully switched to chain ${targetChainId}`);
    } catch (switchError) {
      // If the network doesn't exist, we might need to add it
      if ((switchError as { code?: number }).code === 4902) {
        logger.log(`Network ${targetChainId} not found, would need to add it`);
        throw new Error(`Network ${targetChainId} not configured in wallet`);
      } else {
        throw switchError;
      }
    }
  } catch (error) {
    logger.error('Failed to switch network:', { error });
    throw error;
  }
}

interface Web3AuthDelegationButtonProps {
  priceId: string;
  productTokenId?: string;
  disabled?: boolean;
  requiredChainId?: number;
  tokenAddress?: Address | undefined;
  tokenAmount?: bigint | null;
  tokenSymbol?: string | undefined;
  productName?: string;
  productDescription?: string;
  networkName?: string;
  priceDisplay?: string;
  intervalType?: string;
  termLength?: number;
  tokenDecimals?: number;
}

export function Web3AuthDelegationButton({
  priceId,
  productTokenId,
  disabled = false,
  tokenAmount,
  productName,
  productDescription,
  networkName,
  priceDisplay,
  intervalType,
  termLength,
  tokenDecimals,
}: Web3AuthDelegationButtonProps) {
  const { isAuthenticated } = useWeb3AuthInitialization();
  const {
    smartAccountAddress,
    isSmartAccountReady,
    isDeployed,
    deploymentSupported,
    checkDeploymentStatus,
    deploySmartAccount,
  } = useWeb3AuthSmartAccount();

  // Get Web3Auth instance to access the underlying smart account
  const { web3Auth } = useWeb3Auth();

  // Get current network context
  const currentNetwork = useNetworkStore((state) => state.currentNetwork);

  const envConfig = useEnvConfig();
  const { toast } = useToast();

  const [showDelegationDialog, setShowDelegationDialog] = useState(false);
  const [showConfirmationDialog, setShowConfirmationDialog] = useState(false);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [signedDelegation, setSignedDelegation] = useState<string | null>(null);
  const [mounted, setMounted] = useState(false);
  const [subscriptionSuccessful, setSubscriptionSuccessful] = useState(false);
  const [transactionHash, setTransactionHash] = useState<string | null>(null);
  const [networkInfo, setNetworkInfo] = useState<NetworkWithTokensResponse | null>(null);
  const [status, setStatus] = useState<
    'idle' | 'checking' | 'deploying' | 'signing' | 'subscribing' | 'switching-network'
  >('idle');
  const [subscriptionInfo, setSubscriptionInfo] = useState<{
    id: string;
    productName?: string;
    customerName?: string;
    tokenSymbol?: string;
    tokenAmount: string;
    totalAmountCents?: number;
    walletAddress?: string;
    subscriptionStatus?: string;
    currentPeriodEnd?: number;
    nextRedemptionDate?: string;
    networkId?: string;
  } | null>(null);

  useEffect(() => setMounted(true), []);

  // Auto-switch network on mount if authenticated and network is specified
  useEffect(() => {
    if (!isAuthenticated || !networkName || !web3Auth?.provider || !mounted) return;

    const checkAndSwitchNetwork = async () => {
      try {
        const requiredChainId = await getChainIdFromNetworkName(networkName);
        if (!requiredChainId) return;

        // Small delay to ensure provider is ready
        await new Promise((resolve) => setTimeout(resolve, 500));

        let currentChainIdDecimal: number;
        try {
          const currentChainId = (await (web3Auth.provider as Web3AuthProvider).request({
            method: 'eth_chainId',
          })) as string;
          currentChainIdDecimal = parseInt(currentChainId, 16);
        } catch (error) {
          logger.error('Failed to get current chain ID on mount:', { error });
          return;
        }

        if (currentChainIdDecimal !== requiredChainId) {
          logger.log(
            `🔄 Auto-switching from chain ${currentChainIdDecimal} to ${requiredChainId} on mount`
          );

          try {
            await switchToNetwork(web3Auth.provider as Web3AuthProvider, requiredChainId);
            logger.log('✅ Auto-switched to correct network on mount');
          } catch (error) {
            logger.warn('⚠️ Auto-switch failed on mount, user will need to switch manually:', {
              error,
            });
          }
        }
      } catch (error) {
        logger.error('Error in auto-switch network check:', { error });
      }
    };

    checkAndSwitchNetwork();
  }, [isAuthenticated, networkName, web3Auth?.provider, mounted]);

  async function handleCreateDelegation() {
    if (status !== 'idle') {
      toast({
        title: 'Please wait',
        description: 'A request is already being processed.',
        variant: 'destructive',
      });
      return;
    }

    if (!isAuthenticated) {
      toast({
        title: 'Authentication required',
        description: 'Please sign in with Web3Auth to subscribe.',
        variant: 'destructive',
      });
      return;
    }

    if (!productTokenId) {
      toast({
        title: 'Missing Product Token',
        description: 'Product token ID is required.',
        variant: 'destructive',
      });
      return;
    }

    if (!smartAccountAddress) {
      toast({
        title: 'Smart Account Not Ready',
        description: 'Web3Auth smart account is not available.',
        variant: 'destructive',
      });
      return;
    }

    // Show confirmation dialog first
    setShowConfirmationDialog(true);
  }

  async function proceedWithSubscription() {
    if (!termsAccepted) {
      toast({
        title: 'Terms Required',
        description: 'Please accept the terms and conditions to proceed.',
        variant: 'destructive',
      });
      return;
    }

    setShowConfirmationDialog(false);

    try {
      // Step 0: Ensure we're on the correct network
      if (networkName && web3Auth?.provider) {
        setStatus('switching-network');
        logger.log('🔍 Checking network compatibility...');

        const requiredChainId = await getChainIdFromNetworkName(networkName);
        if (requiredChainId) {
          // Add a small delay to ensure Web3Auth provider is fully initialized
          await new Promise((resolve) => setTimeout(resolve, 500));

          // Get current chain ID
          let currentChainIdDecimal: number;
          try {
            const currentChainId = (await (web3Auth.provider as Web3AuthProvider).request({
              method: 'eth_chainId',
            })) as string;
            currentChainIdDecimal = parseInt(currentChainId, 16);
          } catch (error) {
            logger.error('❌ Failed to get current chain ID:', { error });
            // Default to mainnet if we can't get the chain ID
            currentChainIdDecimal = 1;
          }

          logger.log('🔍 Network check:', {
            networkName,
            requiredChainId,
            currentChainId: currentChainIdDecimal,
            needsSwitch: currentChainIdDecimal !== requiredChainId,
          });

          if (currentChainIdDecimal !== requiredChainId) {
            logger.log(`🔄 Switching from chain ${currentChainIdDecimal} to ${requiredChainId}`);

            try {
              await switchToNetwork(web3Auth.provider as Web3AuthProvider, requiredChainId);

              toast({
                title: 'Network Switched',
                description: `Successfully switched to ${networkName}`,
              });

              // Give a moment for the network switch to propagate
              await new Promise((resolve) => setTimeout(resolve, 1500));
            } catch (switchError) {
              logger.error('❌ Failed to switch network:', { error: switchError });
              toast({
                title: 'Network Switch Failed',
                description: `Please manually switch to ${networkName} in your wallet and try again.`,
                variant: 'destructive',
              });
              setStatus('idle');
              return;
            }
          } else {
            logger.log('✅ Already on correct network');
          }
        }
      }

      // Step 1: Check deployment status
      setStatus('checking');
      logger.log('🔍 Checking smart account deployment status via bytecode...');

      let accountIsDeployed = isDeployed;

      // If deployment status is unknown, check it via bytecode
      if (accountIsDeployed === null) {
        accountIsDeployed = await checkDeploymentStatus();
      }

      // Step 2: Deploy only if not deployed (based on bytecode check)
      if (!accountIsDeployed) {
        // Try deployment even if deploymentSupported is false, as it might be a detection issue
        logger.log('⚠️ Attempting deployment despite deploymentSupported:', deploymentSupported);

        setStatus('deploying');
        logger.log(
          '🚀 Smart account not deployed (no bytecode found), deploying via AccountAbstractionProvider...'
        );

        try {
          await deploySmartAccount();
        } catch (deployError) {
          // If deployment fails, log the error but continue - the account might already be deployed
          logger.warn('⚠️ Deployment failed, but continuing:', deployError as Record<string, unknown>);
          // Check if it's actually deployed now
          const checkAgain = await checkDeploymentStatus();
          if (!checkAgain) {
            // Only throw if it's really not deployed
            throw deployError;
          }
        }

        logger.log('✅ Smart account deployed successfully!');

        toast({
          title: 'Smart Account Deployed',
          description: 'Your smart account has been deployed with sponsored gas.',
        });
      } else {
        logger.log('✅ Smart account is already deployed (bytecode verified)');
      }

      // Step 3: Create delegation
      setStatus('signing');
      logger.log('🔐 Creating delegation...');

      // Get delegate address
      const delegateAddress = await getCypheraDelegateAddress(envConfig.delegateAddress);

      // Access the smart account through Web3Auth's AccountAbstractionProvider
      if (!web3Auth?.accountAbstractionProvider?.smartAccount) {
        throw new Error('Web3Auth smart account not available for delegation signing.');
      }

      logger.log('🔐 Creating delegation with smart account:', smartAccountAddress);
      logger.log('🔐 Delegating to:', delegateAddress);

      // Cast the Web3Auth smart account to MetaMaskSmartAccount to access signDelegation method
      const smartAccount = web3Auth.accountAbstractionProvider.smartAccount as MetaMaskSmartAccount;

      //TODO: update code for specific caveat enforcers
      // Create delegation using the MetaMask delegation toolkit
      const delegation = createDelegation({
        from: smartAccountAddress as `0x${string}`, // The subscriber's smart account address
        to: delegateAddress, // The Cyphera delegate address
        caveats: [], // No caveats for this implementation
      });

      logger.log('🔐 Delegation created:', delegation);

      // Sign the delegation using the smart account's signDelegation method
      const signature = (await smartAccount.signDelegation({
        delegation,
      })) as Hex;

      logger.log('🔐 Delegation signed:', signature);

      // Create the final signed delegation
      const signedDelegation = {
        ...delegation,
        signature,
      };

      logger.log('🔐 Final signed delegation:', signedDelegation);

      // Step 4: Create subscription
      setStatus('subscribing');
      logger.log('📝 Creating subscription...');

      // Parse the delegation back to object format for the API
      const formattedDelegation = formatDelegation(signedDelegation);
      setSignedDelegation(formattedDelegation);

      const subscriptionPayload = {
        product_token_id: productTokenId,
        subscriber_address: smartAccountAddress, // Required by backend
        token_amount: tokenAmount?.toString() || '0', // Required by backend
        delegation: signedDelegation, // Send the object, not the formatted string
      };

      // Log the payload we're sending
      console.log('📤 Sending subscription payload:', JSON.stringify(subscriptionPayload, null, 2));
      console.log('Formatted delegation for display:', formattedDelegation);
      console.log('Raw delegation object being sent:', signedDelegation);

      const response = await fetch('/api/public/prices/' + priceId + '/subscribe', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(subscriptionPayload),
      });

      // Log the raw response for debugging
      console.log('📥 Response status:', response.status);
      const responseText = await response.text();
      console.log('📥 Response body:', responseText);

      if (!response.ok) {
        let errorData;
        try {
          errorData = JSON.parse(responseText);
        } catch {
          errorData = { message: responseText };
        }
        
        // Log the full error details
        console.error('❌ Subscription error details:', errorData);
        
        throw new Error(errorData.message || errorData.error || 'Failed to create subscription');
      }
      
      // Parse successful response
      const subscriptionResult: SubscriptionResponse = JSON.parse(responseText);
      logger.log('✅ Subscription created successfully:', subscriptionResult);

      // Extract transaction and subscription information
      const txHash = subscriptionResult.initial_transaction_hash;
      const walletAddress = subscriptionResult.metadata?.wallet_address;
      const networkId = subscriptionResult.product_token?.network_id;
      const tokenSymbol = subscriptionResult.product_token?.token_symbol;
      const subscriptionTokenAmount = subscriptionResult.token_amount;
      const totalAmountCents = subscriptionResult.total_amount_in_cents;
      const productName = subscriptionResult.product?.name;
      const customerName = subscriptionResult.customer_name;
      const subscriptionStatus = subscriptionResult.status;
      const currentPeriodEnd = subscriptionResult.current_period_end;
      const nextRedemptionDate = subscriptionResult.next_redemption_date;

      // Store the subscription data
      if (txHash) {
        setTransactionHash(txHash);
      }

      // Store additional subscription info for display
      setSubscriptionInfo({
        id: subscriptionResult.id,
        productName,
        customerName,
        tokenSymbol,
        tokenAmount: subscriptionTokenAmount,
        totalAmountCents,
        walletAddress,
        subscriptionStatus: subscriptionStatus as string | undefined,
        currentPeriodEnd: currentPeriodEnd ? new Date(currentPeriodEnd).getTime() : undefined,
        nextRedemptionDate,
        networkId,
      });

      // Fetch network information for block explorer link
      try {
        const networksResponse = await fetch('/api/networks');
        if (networksResponse.ok) {
          const networks = await networksResponse.json();
          const network = networks.find(
            (n: NetworkWithTokensResponse) => n.network.id === networkId
          );
          if (network) {
            setNetworkInfo(network);
          }
        }
      } catch (error) {
        logger.error('Failed to fetch network information:', { error });
      }

      setSubscriptionSuccessful(true);
      setShowDelegationDialog(true);

      toast({
        title: 'Subscription created!',
        description: 'Your Web3Auth smart account subscription is now active.',
      });
    } catch (error) {
      logger.error('Web3Auth delegation error:', { error });
      toast({
        title: 'Subscription failed',
        description: error instanceof Error ? error.message : 'An unexpected error occurred',
        variant: 'destructive',
      });
    } finally {
      setStatus('idle');
    }
  }

  const getButtonText = () => {
    // Debug logging to understand the state
    logger.log('🔍 [Web3AuthDelegationButton] Button state debug:', {
      isAuthenticated,
      isSmartAccountReady,
      deploymentSupported,
      isDeployed,
      smartAccountAddress,
      networkName,
      currentNetworkChainId: currentNetwork?.network.chain_id,
      web3AuthAccountAbstractionProvider: !!web3Auth?.accountAbstractionProvider,
    });

    if (!isAuthenticated) return 'Sign In to Subscribe';
    if (!isSmartAccountReady) return 'Subscribe';
    // For now, don't check deploymentSupported as it may give false negatives
    // if (!deploymentSupported) return 'Network Not Supported';
    if (isDeployed === false && deploymentSupported) return 'Subscribe';
    return 'Subscribe';
  };

  const getStatusText = () => {
    switch (status) {
      case 'switching-network':
        return 'Switching network...';
      case 'checking':
        return 'Checking smart account...';
      case 'deploying':
        return 'Deploying smart account...';
      case 'signing':
        return 'Waiting for signature...';
      case 'subscribing':
        return 'Creating subscription...';
      default:
        return getButtonText();
    }
  };

  const getStatusIcon = () => {
    if (status === 'switching-network') return <Loader2 className="mr-2 h-4 w-4 animate-spin" />;
    if (status === 'checking') return <Loader2 className="mr-2 h-4 w-4 animate-spin" />;
    if (status === 'deploying') return <Loader2 className="mr-2 h-4 w-4 animate-spin" />;
    if (status === 'signing') return <Loader2 className="mr-2 h-4 w-4 animate-spin" />;
    if (status === 'subscribing') return <Loader2 className="mr-2 h-4 w-4 animate-spin" />;
    return null;
  };

  const isButtonDisabled = () => {
    return (
      disabled ||
      !isAuthenticated ||
      !isSmartAccountReady ||
      status !== 'idle'
      // Removed deploymentSupported check as it may give false negatives
      // || !deploymentSupported
    );
  };

  return (
    <>
      <div className="space-y-3">
        <Button
          onClick={handleCreateDelegation}
          disabled={isButtonDisabled()}
          variant="default"
          className="w-full py-6 text-base bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white disabled:opacity-50"
          suppressHydrationWarning
        >
          {status !== 'idle' && mounted ? (
            <>
              {getStatusIcon()}
              <span suppressHydrationWarning>{getStatusText()}</span>
            </>
          ) : (
            <span suppressHydrationWarning>{getButtonText()}</span>
          )}
        </Button>
      </div>

      {(signedDelegation || subscriptionSuccessful) && (
        <Dialog open={showDelegationDialog} onOpenChange={setShowDelegationDialog}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                {subscriptionSuccessful ? (
                  <>
                    <div className="h-8 w-8 bg-green-100 rounded-full flex items-center justify-center">
                      <CheckCircle className="h-5 w-5 text-green-600" />
                    </div>
                    Subscription Created Successfully!
                  </>
                ) : (
                  'Delegation Created'
                )}
              </DialogTitle>
              <DialogDescription>
                {subscriptionSuccessful
                  ? 'Your subscription is now active and the payment has been confirmed on the blockchain.'
                  : 'Share this delegation link with someone else to let them pay for this product.'}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              {subscriptionSuccessful && subscriptionInfo ? (
                <>
                  {/* Subscription Summary */}
                  <div className="bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 rounded-lg p-4 border border-green-200 dark:border-green-800">
                    <div className="space-y-3">
                      <div className="flex justify-between items-center">
                        <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                          Product
                        </span>
                        <span className="font-semibold">
                          {productName || subscriptionInfo.productName || 'N/A'}
                        </span>
                      </div>

                      {productDescription && (
                        <div className="flex justify-between items-start">
                          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                            Description
                          </span>
                          <span className="text-sm text-right max-w-[200px] text-gray-700 dark:text-gray-300">
                            {productDescription}
                          </span>
                        </div>
                      )}

                      <div className="flex justify-between items-center">
                        <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                          Amount Paid
                        </span>
                        <span className="font-semibold">
                          {(() => {
                            const decimals = tokenDecimals || 6;
                            const displayDecimals = Math.min(decimals, 6);
                            const formattedAmount = formatUnits(
                              BigInt(subscriptionInfo.tokenAmount),
                              decimals
                            );
                            return `${parseFloat(formattedAmount).toFixed(displayDecimals)} ${subscriptionInfo.tokenSymbol}`;
                          })()}
                          {subscriptionInfo.totalAmountCents && (
                            <span className="text-sm text-gray-500 ml-1">
                              (${(subscriptionInfo.totalAmountCents / 100).toFixed(2)})
                            </span>
                          )}
                        </span>
                      </div>

                      {priceDisplay && (
                        <div className="flex justify-between items-center">
                          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                            Price
                          </span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">
                            {priceDisplay}
                          </span>
                        </div>
                      )}

                      {intervalType && (
                        <div className="flex justify-between items-center">
                          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                            Billing
                          </span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">
                            per {intervalType}
                          </span>
                        </div>
                      )}

                      {termLength && (
                        <div className="flex justify-between items-center">
                          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                            Term
                          </span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">
                            {termLength} payments
                          </span>
                        </div>
                      )}

                      {networkName && (
                        <div className="flex justify-between items-center">
                          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                            Network
                          </span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">
                            {networkName}
                          </span>
                        </div>
                      )}

                      {subscriptionInfo.nextRedemptionDate && (
                        <div className="flex justify-between items-center">
                          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                            Next Payment Date
                          </span>
                          <span className="text-sm text-gray-700 dark:text-gray-300">
                            {new Date(subscriptionInfo.nextRedemptionDate).toLocaleDateString()}
                          </span>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Transaction Details */}
                  {transactionHash && (
                    <div className="space-y-3">
                      <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        Transaction Details
                      </div>

                      <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-3 border">
                        <div className="space-y-2">
                          <div className="flex justify-between items-start">
                            <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
                              Transaction Hash
                            </span>
                            <button
                              onClick={() => {
                                navigator.clipboard.writeText(transactionHash);
                                toast({
                                  title: 'Copied',
                                  description: 'Transaction hash copied to clipboard',
                                });
                              }}
                              className="text-xs text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
                            >
                              Copy
                            </button>
                          </div>
                          <div className="text-xs font-mono break-all text-gray-800 dark:text-gray-200 bg-white dark:bg-gray-800 rounded p-2">
                            {networkInfo ? (
                              <a
                                href={(() => {
                                  const explorerLink = generateExplorerLink(
                                    [networkInfo],
                                    networkInfo.network.chain_id,
                                    'tx',
                                    transactionHash
                                  );
                                  return explorerLink || '#';
                                })()}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 hover:underline"
                              >
                                {transactionHash}
                              </a>
                            ) : (
                              transactionHash
                            )}
                          </div>

                          <div className="flex justify-between items-center pt-2 border-t border-gray-200 dark:border-gray-700">
                            <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
                              Gas
                            </span>
                            <span className="text-xs font-semibold text-green-600 dark:text-green-400">
                              Free
                            </span>
                          </div>
                        </div>
                      </div>

                      <div className="flex space-x-2">
                        {networkInfo && (
                          <Button
                            onClick={() => {
                              const explorerLink = generateExplorerLink(
                                [networkInfo],
                                networkInfo.network.chain_id,
                                'tx',
                                transactionHash
                              );
                              if (explorerLink) {
                                window.open(explorerLink, '_blank');
                              }
                            }}
                            variant="outline"
                            size="sm"
                            className="flex-1"
                          >
                            <ExternalLink className="h-4 w-4 mr-2" />
                            View on Explorer
                          </Button>
                        )}
                        <Button
                          onClick={() => setShowDelegationDialog(false)}
                          variant="default"
                          size="sm"
                          className="flex-1"
                        >
                          Done
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              ) : !subscriptionSuccessful ? (
                // Keep the existing delegation display for non-subscription cases
                <>
                  <div className="bg-muted p-4 rounded-md overflow-auto">
                    <pre className="text-xs whitespace-pre-wrap break-all">{signedDelegation}</pre>
                  </div>
                  <Button
                    onClick={() => {
                      navigator.clipboard.writeText(signedDelegation || '');
                      toast({
                        title: 'Copied',
                        description: 'Delegation copied to clipboard',
                      });
                    }}
                  >
                    Copy Delegation
                  </Button>
                </>
              ) : (
                <div className="text-center text-muted-foreground">
                  Subscription created but transaction information not available
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Subscription Confirmation Dialog */}
      <Dialog open={showConfirmationDialog} onOpenChange={setShowConfirmationDialog}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <div className="h-8 w-8 bg-blue-100 rounded-full flex items-center justify-center">
                <FileText className="h-5 w-5 text-blue-600" />
              </div>
              Confirm Subscription
            </DialogTitle>
            <DialogDescription>
              Please review your subscription details and accept the terms to proceed.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {/* Subscription Summary */}
            <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg p-4 border border-blue-200 dark:border-blue-800">
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                    Product
                  </span>
                  <span className="font-semibold">{productName || 'Subscription'}</span>
                </div>

                {productDescription && (
                  <div className="flex justify-between items-start">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      Description
                    </span>
                    <span className="text-sm text-right max-w-[180px] text-gray-700 dark:text-gray-300">
                      {productDescription}
                    </span>
                  </div>
                )}

                {priceDisplay && (
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      Price
                    </span>
                    <span className="font-semibold text-blue-600 dark:text-blue-400">
                      {priceDisplay}
                    </span>
                  </div>
                )}

                {intervalType && (
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      Billing
                    </span>
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      per {intervalType}
                    </span>
                  </div>
                )}

                {termLength && (
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      Term
                    </span>
                    <span className="text-sm text-gray-700 dark:text-gray-300">
                      {termLength} payments
                    </span>
                  </div>
                )}

                {networkName && (
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      Network
                    </span>
                    <span className="text-sm text-gray-700 dark:text-gray-300">{networkName}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Terms Acceptance */}
            <div className="space-y-3">
              <div className="flex items-start space-x-3">
                <Checkbox
                  id="terms"
                  checked={termsAccepted}
                  onCheckedChange={(checked) => setTermsAccepted(checked === true)}
                  className="mt-1"
                />
                <label
                  htmlFor="terms"
                  className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed cursor-pointer"
                >
                  I acknowledge and accept the{' '}
                  <a
                    href="/terms"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 underline"
                  >
                    Terms and Conditions
                  </a>{' '}
                  for this subscription service.
                </label>
              </div>

              <div className="text-xs text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-900 rounded p-3 border">
                <p>
                  By proceeding, you authorize automatic payments according to the billing schedule.
                  Gas fees are sponsored.
                </p>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex space-x-3 pt-2">
              <Button
                onClick={() => setShowConfirmationDialog(false)}
                variant="outline"
                className="flex-1"
              >
                Cancel
              </Button>
              <Button
                onClick={proceedWithSubscription}
                disabled={!termsAccepted}
                className="flex-1 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700"
              >
                Confirm & Subscribe
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
