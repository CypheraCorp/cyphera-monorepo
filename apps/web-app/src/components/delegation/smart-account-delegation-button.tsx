'use client';

import { useState, useEffect } from 'react';
import { formatUnits } from 'viem';
import { Button } from '@/components/ui/button';
import { useToast } from '@/components/ui/use-toast';
import { useEnvConfig } from '@/components/env/client';
import { Loader2, CheckCircle, ExternalLink, FileText, Shield } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Checkbox } from '@/components/ui/checkbox';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { 
  createAndSignDelegation, 
  formatDelegation,
  getCypheraDelegateAddress 
} from '@/lib/web3/utils/delegation';
import { generateExplorerLink } from '@/lib/utils/explorers';
import { NetworkWithTokensResponse } from '@/types/network';
import { SubscriptionResponse } from '@/types/subscription';
import { logger } from '@/lib/core/logger/logger-utils';
import type { 
  SmartAccountProvider, 
  DelegationStatus, 
  SubscriptionParams, 
  SubscriptionInfo,
  DelegationResult 
} from './types';


interface SmartAccountDelegationButtonProps extends SubscriptionParams {
  /** Smart account provider to use */
  provider: SmartAccountProvider;
  /** Whether this is subscription mode (creates subscription) or delegation mode (just creates delegation) */
  mode?: 'delegation' | 'subscription';
  /** Additional disabled state */
  disabled?: boolean;
  /** Custom button variant */
  variant?: 'default' | 'outline';
  /** Custom button className */
  className?: string;
}

export function SmartAccountDelegationButton({
  provider,
  mode = 'delegation',
  disabled = false,
  variant = 'default',
  className,
  priceId,
  productTokenId,
  tokenAmount,
  productName,
  productDescription,
  networkName,
  priceDisplay,
  intervalType,
  termLength,
  tokenDecimals,
}: SmartAccountDelegationButtonProps) {
  const envConfig = useEnvConfig();
  const { toast } = useToast();

  // UI State
  const [showDelegationDialog, setShowDelegationDialog] = useState(false);
  const [showConfirmationDialog, setShowConfirmationDialog] = useState(false);
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [mounted, setMounted] = useState(false);
  
  // Delegation State
  const [status, setStatus] = useState<DelegationStatus>('idle');
  const [result, setResult] = useState<DelegationResult | null>(null);
  const [networkInfo, setNetworkInfo] = useState<NetworkWithTokensResponse | null>(null);

  useEffect(() => setMounted(true), []);

  // Auto-switch network on mount if provider supports it
  useEffect(() => {
    if (!provider.isAuthenticated || !networkName || !provider.switchNetwork || !mounted) return;

    const checkAndSwitchNetwork = async () => {
      try {
        await provider.switchNetwork!(networkName);
        logger.log('✅ Auto-switched to correct network on mount');
      } catch (error) {
        logger.warn('⚠️ Auto-switch failed on mount, user will need to switch manually:', { error });
      }
    };

    checkAndSwitchNetwork();
  }, [provider.isAuthenticated, networkName, provider.switchNetwork, mounted]);

  async function handleCreateDelegation() {
    if (status !== 'idle') {
      toast({
        title: 'Please wait',
        description: 'A request is already being processed.',
        variant: 'destructive',
      });
      return;
    }

    if (!provider.isAuthenticated) {
      toast({
        title: `${provider.getDisplayName()} not connected`,
        description: `Please connect your ${provider.getDisplayName()} wallet to continue.`,
        variant: 'destructive',
      });
      return;
    }

    // For subscription mode, show confirmation dialog first
    if (mode === 'subscription') {
      if (!productTokenId) {
        toast({
          title: 'Missing Product Token',
          description: 'Product token ID is required for subscription.',
          variant: 'destructive',
        });
        return;
      }
      setShowConfirmationDialog(true);
      return;
    }

    // For delegation mode, proceed directly
    await proceedWithDelegation();
  }

  async function proceedWithDelegation() {
    if (mode === 'subscription' && !termsAccepted) {
      toast({
        title: 'Terms Required',
        description: 'Please accept the terms and conditions to proceed.',
        variant: 'destructive',
      });
      return;
    }

    setShowConfirmationDialog(false);

    try {
      // Step 1: Ensure correct network
      if (networkName && provider.switchNetwork) {
        setStatus('switching-network');
        logger.log('🔍 Ensuring correct network...');
        
        try {
          await provider.switchNetwork(networkName);
          toast({
            title: 'Network Ready',
            description: `Connected to ${networkName}`,
          });
        } catch (switchError) {
          logger.error('❌ Failed to ensure correct network:', { error: switchError });
          toast({
            title: 'Network Switch Failed',
            description: `Please manually switch to ${networkName} in your wallet and try again.`,
            variant: 'destructive',
          });
          setStatus('idle');
          return;
        }
      }

      // Step 2: Check deployment status
      if (provider.isDeployed === null) {
        setStatus('checking');
        logger.log('🔍 Checking smart account deployment status...');
        await provider.checkDeploymentStatus();
      }

      // Step 3: Deploy if needed
      if (provider.isDeployed === false) {
        setStatus('deploying');
        logger.log('🚀 Deploying smart account...');
        
        try {
          await provider.deploySmartAccount();
          logger.log('✅ Smart account deployed successfully!');
          
          toast({
            title: 'Smart Account Deployed',
            description: 'Your smart account has been deployed with sponsored gas.',
          });
        } catch (deployError) {
          logger.warn('⚠️ Deployment failed, but continuing:', { deployError });
          // Check if it's actually deployed now
          const checkAgain = await provider.checkDeploymentStatus();
          if (!checkAgain) {
            throw deployError;
          }
        }
      }

      // Step 4: Create delegation
      setStatus('signing');
      logger.log('🔐 Creating delegation...');

      if (!provider.smartAccount) {
        throw new Error(`${provider.getDisplayName()} smart account not available for delegation signing.`);
      }

      const delegateAddress = await getCypheraDelegateAddress(envConfig.delegateAddress);
      const signedDelegation = await createAndSignDelegation(provider.smartAccount, delegateAddress);
      const formattedDelegation = formatDelegation(signedDelegation);

      logger.log('🔐 Delegation created successfully');

      // Step 5: Create subscription if in subscription mode
      if (mode === 'subscription') {
        setStatus('subscribing');
        logger.log('📝 Creating subscription...');

        const subscriptionPayload = {
          product_token_id: productTokenId!,
          subscriber_address: provider.smartAccountAddress!,
          token_amount: tokenAmount?.toString() || '0',
          delegation: signedDelegation,
        };

        const response = await fetch(`/api/pay/${priceId}/subscribe`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(subscriptionPayload),
        });

        const responseText = await response.text();
        
        if (!response.ok) {
          let errorData;
          try {
            errorData = JSON.parse(responseText);
          } catch {
            errorData = { message: responseText };
          }
          throw new Error(errorData.message || errorData.error || 'Failed to create subscription');
        }

        const subscriptionResult: SubscriptionResponse = JSON.parse(responseText);
        logger.log('✅ Subscription created successfully:', subscriptionResult);

        // Extract subscription information
        const subscriptionInfo: SubscriptionInfo = {
          id: subscriptionResult.id,
          productName: subscriptionResult.product?.name,
          customerName: subscriptionResult.customer_name,
          tokenSymbol: subscriptionResult.product_token?.token_symbol,
          tokenAmount: subscriptionResult.token_amount.toString(),
          totalAmountCents: subscriptionResult.total_amount_in_cents,
          walletAddress: subscriptionResult.metadata?.wallet_address,
          subscriptionStatus: subscriptionResult.status as string,
          currentPeriodEnd: subscriptionResult.current_period_end ? new Date(subscriptionResult.current_period_end).getTime() : undefined,
          nextRedemptionDate: subscriptionResult.next_redemption_date,
          networkId: subscriptionResult.product_token?.network_id,
          transactionHash: subscriptionResult.initial_transaction_hash,
        };

        // Fetch network information for block explorer
        try {
          const networksResponse = await fetch('/api/networks');
          if (networksResponse.ok) {
            const networks = await networksResponse.json();
            const network = networks.find(
              (n: NetworkWithTokensResponse) => n.network.id === subscriptionInfo.networkId
            );
            if (network) {
              setNetworkInfo(network);
            }
          }
        } catch (error) {
          logger.error('Failed to fetch network information:', { error });
        }

        setResult({
          delegation: formattedDelegation,
          subscription: subscriptionInfo,
          transactionHash: subscriptionInfo.transactionHash,
        });

        toast({
          title: 'Subscription created!',
          description: `Your ${provider.getDisplayName()} smart account subscription is now active.`,
        });
      } else {
        // Delegation mode
        setResult({
          delegation: formattedDelegation,
        });

        toast({
          title: 'Delegation created!',
          description: `You have successfully created a delegation for your ${provider.getDisplayName()} smart account.`,
        });
      }

      setShowDelegationDialog(true);
    } catch (error) {
      logger.error(`${provider.getDisplayName()} delegation error:`, { error });
      toast({
        title: `${mode === 'subscription' ? 'Subscription' : 'Delegation'} failed`,
        description: error instanceof Error ? error.message : 'An unexpected error occurred',
        variant: 'destructive',
      });
    } finally {
      setStatus('idle');
    }
  }

  const getStatusText = () => {
    switch (status) {
      case 'connecting':
        return 'Connecting...';
      case 'creating':
        return 'Creating Smart Account...';
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
        return provider.getButtonText();
    }
  };

  const getStatusIcon = () => {
    if (status !== 'idle') return <Loader2 className="mr-2 h-4 w-4 animate-spin" />;
    if (mode === 'delegation') return <Shield className="mr-2 h-4 w-4" />;
    return null;
  };

  const tooltipText = provider.isButtonDisabled() 
    ? `Connect your ${provider.getDisplayName()} wallet to ${mode === 'subscription' ? 'subscribe' : 'create delegation'}`
    : status !== 'idle'
      ? getStatusText()
      : `${mode === 'subscription' ? 'Subscribe' : 'Create delegation'} using ${provider.getDisplayName()}`;

  return (
    <>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex w-full">
              <Button
                onClick={handleCreateDelegation}
                disabled={disabled || provider.isButtonDisabled() || status !== 'idle'}
                variant={variant}
                className={`w-full ${mode === 'subscription' ? 'py-6 text-base bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white disabled:opacity-50' : ''} ${className || ''}`}
                suppressHydrationWarning
              >
                {status !== 'idle' && mounted ? (
                  <>
                    {getStatusIcon()}
                    <span suppressHydrationWarning>{getStatusText()}</span>
                  </>
                ) : (
                  <>
                    {getStatusIcon()}
                    <span suppressHydrationWarning>{getStatusText()}</span>
                  </>
                )}
              </Button>
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <p>{tooltipText}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      {/* Results Dialog */}
      {result && (
        <Dialog open={showDelegationDialog} onOpenChange={setShowDelegationDialog}>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                {result.subscription ? (
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
                {result.subscription
                  ? 'Your subscription is now active and the payment has been confirmed on the blockchain.'
                  : 'Your delegation has been created successfully. You can share this delegation to allow others to perform actions on behalf of your smart account.'}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              {result.subscription ? (
                <SubscriptionSuccessContent 
                  subscription={result.subscription}
                  productName={productName}
                  productDescription={productDescription}
                  priceDisplay={priceDisplay}
                  intervalType={intervalType}
                  termLength={termLength}
                  networkName={networkName}
                  tokenDecimals={tokenDecimals}
                  transactionHash={result.transactionHash}
                  networkInfo={networkInfo}
                  toast={toast}
                />
              ) : (
                <DelegationContent delegation={result.delegation} toast={toast} />
              )}
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Subscription Confirmation Dialog */}
      {mode === 'subscription' && (
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
              <SubscriptionConfirmationContent
                productName={productName}
                productDescription={productDescription}
                priceDisplay={priceDisplay}
                intervalType={intervalType}
                termLength={termLength}
                networkName={networkName}
              />

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
                  onClick={proceedWithDelegation}
                  disabled={!termsAccepted}
                  className="flex-1 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700"
                >
                  Confirm & Subscribe
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}

// Helper components for dialog content
function SubscriptionSuccessContent({ 
  subscription, 
  productName, 
  productDescription, 
  priceDisplay, 
  intervalType, 
  termLength, 
  networkName, 
  tokenDecimals, 
  transactionHash, 
  networkInfo, 
  toast 
}: {
  subscription: SubscriptionInfo;
  productName?: string;
  productDescription?: string;
  priceDisplay?: string;
  intervalType?: string;
  termLength?: number;
  networkName?: string;
  tokenDecimals?: number;
  transactionHash?: string;
  networkInfo?: NetworkWithTokensResponse | null;
  toast: any;
}) {
  return (
    <>
      {/* Subscription Summary */}
      <div className="bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 rounded-lg p-4 border border-green-200 dark:border-green-800">
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Product</span>
            <span className="font-semibold">{productName || subscription.productName || 'N/A'}</span>
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
                const formattedAmount = formatUnits(BigInt(subscription.tokenAmount), decimals);
                return `${parseFloat(formattedAmount).toFixed(displayDecimals)} ${subscription.tokenSymbol}`;
              })()}
              {subscription.totalAmountCents && (
                <span className="text-sm text-gray-500 ml-1">
                  (${(subscription.totalAmountCents / 100).toFixed(2)})
                </span>
              )}
            </span>
          </div>

          {priceDisplay && (
            <div className="flex justify-between items-center">
              <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Price</span>
              <span className="text-sm text-gray-700 dark:text-gray-300">{priceDisplay}</span>
            </div>
          )}

          {intervalType && (
            <div className="flex justify-between items-center">
              <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Billing</span>
              <span className="text-sm text-gray-700 dark:text-gray-300">per {intervalType}</span>
            </div>
          )}

          {termLength && (
            <div className="flex justify-between items-center">
              <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Term</span>
              <span className="text-sm text-gray-700 dark:text-gray-300">{termLength} payments</span>
            </div>
          )}

          {networkName && (
            <div className="flex justify-between items-center">
              <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Network</span>
              <span className="text-sm text-gray-700 dark:text-gray-300">{networkName}</span>
            </div>
          )}

          {subscription.nextRedemptionDate && (
            <div className="flex justify-between items-center">
              <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Next Payment Date</span>
              <span className="text-sm text-gray-700 dark:text-gray-300">
                {new Date(subscription.nextRedemptionDate).toLocaleDateString()}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Transaction Details */}
      {transactionHash && (
        <div className="space-y-3">
          <div className="text-sm font-medium text-gray-900 dark:text-gray-100">Transaction Details</div>
          <div className="bg-gray-50 dark:bg-gray-900 rounded-lg p-3 border">
            <div className="space-y-2">
              <div className="flex justify-between items-start">
                <span className="text-xs font-medium text-gray-600 dark:text-gray-400">Transaction Hash</span>
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
              onClick={() => {}}
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
  );
}

function DelegationContent({ delegation, toast }: { delegation: string; toast: any }) {
  return (
    <>
      <div className="bg-muted p-4 rounded-md overflow-auto">
        <pre className="text-xs whitespace-pre-wrap break-all">{delegation}</pre>
      </div>
      <Button
        onClick={() => {
          navigator.clipboard.writeText(delegation);
          toast({
            title: 'Copied!',
            description: 'Delegation copied to clipboard',
          });
        }}
        className="w-full"
      >
        Copy Delegation
      </Button>
    </>
  );
}

function SubscriptionConfirmationContent({
  productName,
  productDescription,
  priceDisplay,
  intervalType,
  termLength,
  networkName,
}: {
  productName?: string;
  productDescription?: string;
  priceDisplay?: string;
  intervalType?: string;
  termLength?: number;
  networkName?: string;
}) {
  return (
    <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 rounded-lg p-4 border border-blue-200 dark:border-blue-800">
      <div className="space-y-2">
        <div className="flex justify-between items-center">
          <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Product</span>
          <span className="font-semibold">{productName || 'Subscription'}</span>
        </div>

        {productDescription && (
          <div className="flex justify-between items-start">
            <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Description</span>
            <span className="text-sm text-right max-w-[180px] text-gray-700 dark:text-gray-300">
              {productDescription}
            </span>
          </div>
        )}

        {priceDisplay && (
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Price</span>
            <span className="font-semibold text-blue-600 dark:text-blue-400">{priceDisplay}</span>
          </div>
        )}

        {intervalType && (
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Billing</span>
            <span className="text-sm text-gray-700 dark:text-gray-300">per {intervalType}</span>
          </div>
        )}

        {termLength && (
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Term</span>
            <span className="text-sm text-gray-700 dark:text-gray-300">{termLength} payments</span>
          </div>
        )}

        {networkName && (
          <div className="flex justify-between items-center">
            <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Network</span>
            <span className="text-sm text-gray-700 dark:text-gray-300">{networkName}</span>
          </div>
        )}
      </div>
    </div>
  );
}