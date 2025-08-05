'use client';

import { useState, useEffect } from 'react';
import { type Address, formatUnits } from 'viem';
import { useWeb3AuthSmartAccount } from '@/hooks/auth';
import { useWeb3AuthInitialization } from '@/hooks/auth';
import { useWeb3Auth, useSwitchChain } from '@web3auth/modal/react';
import { formatDelegation } from '@/lib/web3/utils/delegation';
import { MetaMaskSmartAccount } from '@metamask/delegation-toolkit';
import { createAndSignDelegation } from '@cyphera/delegation';
import { Button } from '@/components/ui/button';
import { useToast } from '@/components/ui/use-toast';
import { useEnvConfig } from '@/components/env/client';
import { useNetworkStore } from '@/store/network';
import { Loader2, CheckCircle, ExternalLink, FileText, ChevronRight, Info } from 'lucide-react';
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

interface Web3AuthDelegationButtonProps {
  productId: string; // Product ID for the subscription
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

type DelegationStatus = 
  | 'idle' 
  | 'switching-network'
  | 'checking' 
  | 'deploying' 
  | 'signing' 
  | 'subscribing';

// Timeout wrapper helper
async function withTimeout<T>(
  promise: Promise<T>, 
  timeoutMs: number, 
  operation: string
): Promise<T> {
  const timeoutPromise = new Promise<never>((_, reject) => {
    setTimeout(() => {
      reject(new Error(`${operation} timed out after ${timeoutMs / 1000} seconds`));
    }, timeoutMs);
  });

  return Promise.race([promise, timeoutPromise]);
}

export function Web3AuthDelegationButton({
  productId,
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
    refreshNetworkState,
  } = useWeb3AuthSmartAccount();

  // Get Web3Auth instance to access the underlying smart account
  const { web3Auth } = useWeb3Auth();
  
  // Use Web3Auth's chain switching hook
  const { switchChain, loading: isSwitchingChain, error: switchChainError } = useSwitchChain();

  // Get current network context
  const currentNetwork = useNetworkStore((state) => state.currentNetwork);

  const [status, setStatus] = useState<DelegationStatus>('idle');
  const [showSuccessDialog, setShowSuccessDialog] = useState(false);
  const [transactionDetails, setTransactionDetails] = useState<{
    hash: string;
    network: string;
    walletAddress: string;
    productName: string;
    tokenAmount: string;
    tokenSymbol: string;
    totalAmountCents?: number;
  } | null>(null);
  const [signedDelegation, setSignedDelegation] = useState<string | null>(null);
  const [networkInfo, setNetworkInfo] = useState<NetworkWithTokensResponse | null>(null);
  const envConfig = useEnvConfig();
  const { toast } = useToast();

  // Auto-switch network on mount if authenticated
  useEffect(() => {
    if (!isAuthenticated || !networkName) return;

    const checkAndSwitchNetwork = async () => {
      try {
        // Log network check attempt
        logger.log('🌐 [DEBUG] Auto-checking network on mount for:', networkName);
        
        // Get chain ID for network
        const chainId = await getChainIdFromNetworkName(networkName);
        if (chainId && switchChain) {
          const hexChainId = `0x${chainId.toString(16)}`;
          await switchChain(hexChainId);
          logger.log('✅ [DEBUG] Auto-switched to correct network on mount');
        }
      } catch (error) {
        logger.warn('⚠️ [DEBUG] Auto-switch failed on mount, user will need to switch manually:', { error });
      }
    };

    checkAndSwitchNetwork();
  }, [isAuthenticated, networkName, switchChain]);

  async function handleCreateSubscription() {
    logger.log('🚀 [DEBUG] handleCreateSubscription called with:', {
      productTokenId,
      productId,
      networkName,
      tokenDecimals,
      priceDisplay,
      intervalType,
      productName: productName,
      termLength,
      tokenAmount: tokenAmount?.toString(),
    });

    if (status !== 'idle') {
      logger.warn('⚠️ [DEBUG] Already processing, status:', { status });
      toast({
        title: 'Please wait',
        description: 'A request is already being processed.',
        variant: 'destructive',
      });
      return;
    }

    try {
      // Step 1: Ensure user is logged in
      if (!isAuthenticated) {
        logger.log('🔐 [DEBUG] User not authenticated');
        toast({
          title: 'Authentication required',
          description: 'Please sign in with Web3Auth to subscribe.',
          variant: 'destructive',
        });
        return;
      }

      // Step 2: Ensure correct network
      if (networkName) {
        setStatus('switching-network');
        logger.log('🔍 [DEBUG] Ensuring correct network:', networkName);
        
        try {
          const chainId = await getChainIdFromNetworkName(networkName);
          if (chainId && switchChain) {
            const hexChainId = `0x${chainId.toString(16)}`;
            await withTimeout(
              switchChain(hexChainId),
              30000,
              'Network switch'
            );
            logger.log('✅ [DEBUG] Network switch successful');
            
            toast({
              title: 'Network Ready',
              description: `Connected to ${networkName}`,
            });
          }
        } catch (switchError) {
          logger.error('❌ [DEBUG] Failed to ensure correct network:', { error: switchError });
          toast({
            title: 'Network Switch Failed',
            description: `Please manually switch to ${networkName} in your wallet and try again.`,
            variant: 'destructive',
          });
          setStatus('idle');
          return;
        }
      }

      // Step 2b: Check deployment status
      if (isDeployed === null) {
        setStatus('checking');
        logger.log('🔍 [DEBUG] Checking smart account deployment status...');
        logger.log('🔍 [DEBUG] Smart account address:', smartAccountAddress);
        
        try {
          const deploymentStatus = await withTimeout(
            checkDeploymentStatus(),
            15000,
            'Deployment check'
          );
          logger.log('🔍 [DEBUG] Deployment check result:', deploymentStatus);
        } catch (checkError) {
          logger.error('❌ [DEBUG] Deployment check failed:', { error: checkError });
          // Continue anyway, deployment might still work
        }
      }

      // Step 2c: Deploy if needed
      if (isDeployed === false) {
        setStatus('deploying');
        logger.log('🚀 [DEBUG] Smart account not deployed, deploying now...');
        
        try {
          await withTimeout(
            deploySmartAccount(),
            60000, // 60 seconds for deployment
            'Smart account deployment'
          );
          logger.log('✅ [DEBUG] Smart account deployment completed');
          
          // Re-check deployment status
          const checkAgain = await checkDeploymentStatus();
          if (!checkAgain) {
            throw new Error('Smart account deployment verification failed');
          }
        } catch (deployError) {
          logger.error('❌ [DEBUG] Smart account deployment failed:', { error: deployError });
          
          // Check if it's actually deployed now
          const checkAgain = await checkDeploymentStatus();
          if (!checkAgain) {
            // Only throw if it's really not deployed
            throw deployError;
          }
        }

        logger.log('✅ [DEBUG] Smart account deployed successfully!');

        toast({
          title: 'Smart Account Deployed',
          description: 'Your smart account has been deployed with sponsored gas.',
        });
      } else {
        logger.log('✅ [DEBUG] Smart account is already deployed (bytecode verified)');
      }

      // Step 3: Create delegation
      setStatus('signing');
      logger.log('🔐 [DEBUG] Creating delegation...');

      // Get delegate address
      const delegateAddress = await getCypheraDelegateAddress(envConfig.delegateAddress);

      // Access the smart account through Web3Auth's AccountAbstractionProvider
      if (!web3Auth?.accountAbstractionProvider?.smartAccount) {
        logger.error('❌ [DEBUG] Smart account not available:', {
          web3Auth: !!web3Auth,
          accountAbstractionProvider: !!web3Auth?.accountAbstractionProvider,
          smartAccount: !!web3Auth?.accountAbstractionProvider?.smartAccount,
          smartAccountAddress,
        });
        throw new Error('Web3Auth smart account not available for delegation signing.');
      }

      logger.log('🔐 [DEBUG] Creating delegation with smart account:', smartAccountAddress);
      logger.log('🔐 [DEBUG] Delegating to:', delegateAddress);
      
      // Cast the Web3Auth smart account to MetaMaskSmartAccount to access signDelegation method
      const smartAccount = web3Auth.accountAbstractionProvider.smartAccount as MetaMaskSmartAccount;

      // Debug smart account interface
      logger.log('🔍 [DEBUG] Smart account type:', smartAccount.constructor.name);
      logger.log('🔍 [DEBUG] Smart account address from object:', smartAccount.address);
      logger.log('🔍 [DEBUG] Has signDelegation method:', typeof smartAccount.signDelegation === 'function');
      
      // Log available methods on the smart account
      const smartAccountMethods = Object.getOwnPropertyNames(Object.getPrototypeOf(smartAccount));
      logger.log('🔍 [DEBUG] Available smart account methods:', smartAccountMethods);

      // Use the delegation factory to create and sign the delegation with proper transformation
      logger.log('⏳ [DEBUG] Calling createAndSignDelegation...');
      
      let signedDelegation;
      try {
        signedDelegation = await withTimeout(
          createAndSignDelegation(smartAccount, delegateAddress),
          30000, // 30 second timeout
          'Delegation signing'
        );
        logger.log('✅ [DEBUG] Delegation signed successfully');
      } catch (delegationError) {
        logger.error('❌ [DEBUG] Delegation signing failed:', {
          error: delegationError,
          errorMessage: delegationError instanceof Error ? delegationError.message : 'Unknown error',
          errorStack: delegationError instanceof Error ? delegationError.stack : undefined,
        });
        
        // Check if it's a timeout
        if (delegationError instanceof Error && delegationError.message.includes('timed out')) {
          toast({
            title: 'Delegation Timeout',
            description: 'The delegation signing took too long. Please try again.',
            variant: 'destructive',
          });
        }
        
        throw delegationError;
      }

      logger.log('🔐 [DEBUG] Final signed delegation with transformed authority:', signedDelegation);

      // Step 4: Create subscription
      setStatus('subscribing');
      logger.log('📝 [DEBUG] Creating subscription...');

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
      logger.log('📤 [DEBUG] Sending subscription payload:', JSON.stringify(subscriptionPayload, null, 2));
      logger.log('[DEBUG] Formatted delegation for display:', formattedDelegation);
      logger.log('[DEBUG] Raw delegation object being sent:', signedDelegation);

      const response = await fetch('/api/pay/' + productId + '/subscribe', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(subscriptionPayload),
      });

      // Log the raw response for debugging
      logger.log('📥 [DEBUG] Response status:', response.status);
      const responseText = await response.text();
      logger.log('📥 [DEBUG] Response body:', responseText);

      if (!response.ok) {
        let errorData;
        try {
          errorData = JSON.parse(responseText);
        } catch {
          errorData = { message: responseText };
        }
        
        // Log the full error details
        logger.error('❌ [DEBUG] Subscription error details:', errorData);
        
        throw new Error(errorData.message || errorData.error || 'Failed to create subscription');
      }
      
      // Parse successful response
      const subscriptionResult: SubscriptionResponse = JSON.parse(responseText);
      logger.log('✅ [DEBUG] Subscription created successfully:', subscriptionResult);

      // Extract transaction and subscription information
      const txHash = subscriptionResult.initial_transaction_hash;
      const walletAddress = subscriptionResult.metadata?.wallet_address;
      const networkId = subscriptionResult.product_token?.network_id;
      const tokenSymbol = subscriptionResult.product_token?.token_symbol;
      const subscriptionTokenAmount = subscriptionResult.token_amount;
      const totalAmountCents = subscriptionResult.total_amount_in_cents;
      const productName = subscriptionResult.product?.name;
      const customerName = subscriptionResult.customer_name;

      // Fetch network information for block explorer
      try {
        const networksResponse = await fetch('/api/networks');
        if (networksResponse.ok) {
          const networks = await networksResponse.json();
          const network = networks.find((n: NetworkWithTokensResponse) => n.network.id === networkId);
          if (network) {
            setNetworkInfo(network);
          }
        }
      } catch (error) {
        logger.error('[DEBUG] Failed to fetch network information:', { error });
      }

      setTransactionDetails({
        hash: txHash || '',
        network: networkName || '',
        walletAddress: walletAddress || smartAccountAddress || '',
        productName: productName || 'Subscription',
        tokenAmount: subscriptionTokenAmount.toString(),
        tokenSymbol: tokenSymbol || 'USDC',
        totalAmountCents,
      });

      setShowSuccessDialog(true);

      toast({
        title: 'Subscription created!',
        description: `Your smart account subscription is now active.`,
      });

      logger.log('🎉 [DEBUG] Subscription flow completed successfully!');

    } catch (error) {
      logger.error('❌ [DEBUG] Web3Auth subscription error:', { 
        error,
        errorMessage: error instanceof Error ? error.message : 'Unknown error',
        errorStack: error instanceof Error ? error.stack : undefined,
      });
      
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
        return isAuthenticated ? 'Subscribe with Smart Account' : 'Sign In to Subscribe';
    }
  };

  const isButtonDisabled = status !== 'idle';

  return (
    <>
      <div className="mt-4 sm:mt-6 flex flex-col gap-4">
        <Button
          onClick={handleCreateSubscription}
          disabled={isButtonDisabled}
          className="w-full py-6 text-base bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white disabled:opacity-50"
          size="lg"
        >
          {status !== 'idle' ? (
            <>
              <Loader2 className="mr-2 h-5 w-5 animate-spin" />
              {getButtonText()}
            </>
          ) : (
            <>
              {getButtonText()}
              <ChevronRight className="ml-2 h-5 w-5" />
            </>
          )}
        </Button>

        <div className="text-xs text-gray-500 dark:text-gray-400 text-center space-y-1">
          <div className="flex items-center justify-center gap-1">
            <Info className="h-3 w-3" />
            <span>Powered by Web3Auth & MetaMask Smart Accounts</span>
          </div>
          <div>Gas fees are sponsored • Cancel anytime</div>
        </div>
      </div>

      {/* Success Dialog */}
      <Dialog open={showSuccessDialog} onOpenChange={setShowSuccessDialog}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <div className="h-8 w-8 bg-green-100 rounded-full flex items-center justify-center">
                <CheckCircle className="h-5 w-5 text-green-600" />
              </div>
              Subscription Created Successfully!
            </DialogTitle>
            <DialogDescription>
              Your subscription is now active and the payment has been confirmed on the blockchain.
            </DialogDescription>
          </DialogHeader>

          {transactionDetails && (
            <div className="space-y-4">
              {/* Subscription Summary */}
              <div className="bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 rounded-lg p-4 border border-green-200 dark:border-green-800">
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Product</span>
                    <span className="font-semibold">{productName || 'Subscription'}</span>
                  </div>
                  
                  {productDescription && (
                    <div className="flex justify-between items-start">
                      <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Description</span>
                      <span className="text-sm text-right max-w-[200px] text-gray-700 dark:text-gray-300">
                        {productDescription}
                      </span>
                    </div>
                  )}
                  
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Amount Paid</span>
                    <span className="font-semibold">
                      {(() => {
                        const decimals = tokenDecimals || 6;
                        const displayDecimals = Math.min(decimals, 6);
                        const formattedAmount = formatUnits(BigInt(transactionDetails.tokenAmount), decimals);
                        return `${parseFloat(formattedAmount).toFixed(displayDecimals)} ${transactionDetails.tokenSymbol}`;
                      })()}
                      {transactionDetails.totalAmountCents && (
                        <span className="text-sm text-gray-500 ml-1">
                          (${(transactionDetails.totalAmountCents / 100).toFixed(2)})
                        </span>
                      )}
                    </span>
                  </div>
                  
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Price</span>
                    <span className="text-sm text-gray-700 dark:text-gray-300">{priceDisplay}</span>
                  </div>
                  
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Billing</span>
                    <span className="text-sm text-gray-700 dark:text-gray-300">per {intervalType}</span>
                  </div>
                  
                  {termLength && (
                    <div className="flex justify-between items-center">
                      <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Term</span>
                      <span className="text-sm text-gray-700 dark:text-gray-300">{termLength} payments</span>
                    </div>
                  )}
                  
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Network</span>
                    <span className="text-sm text-gray-700 dark:text-gray-300">{networkName}</span>
                  </div>
                </div>
              </div>

              {/* Transaction Details */}
              <div className="space-y-3">
                <div className="text-sm font-medium text-gray-900 dark:text-gray-100">Transaction Details</div>
                <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-3 border">
                  <div className="space-y-2">
                    <div className="flex justify-between items-start">
                      <span className="text-xs font-medium text-gray-600 dark:text-gray-400">Transaction Hash</span>
                      <button
                        onClick={() => {
                          navigator.clipboard.writeText(transactionDetails.hash);
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
                              transactionDetails.hash
                            );
                            return explorerLink || '#';
                          })()}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 hover:underline"
                        >
                          {transactionDetails.hash}
                        </a>
                      ) : (
                        transactionDetails.hash
                      )}
                    </div>
                    
                    <div className="flex justify-between items-center pt-2 border-t border-gray-200 dark:border-gray-700">
                      <span className="text-xs font-medium text-gray-600 dark:text-gray-400">Gas</span>
                      <span className="text-xs font-semibold text-green-600 dark:text-green-400">Free</span>
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
                          transactionDetails.hash
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
                    onClick={() => window.location.href = '/dashboard'}
                    variant="default"
                    size="sm"
                    className="flex-1"
                  >
                    Go to Dashboard
                  </Button>
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}