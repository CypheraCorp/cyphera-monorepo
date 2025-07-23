'use client';

import { useState, useEffect } from 'react';
import { useWeb3AuthUser, useWeb3Auth } from '@web3auth/modal/react';
import { useAccount } from 'wagmi';
import { CustomerWeb3AuthLogin } from '@/components/auth/customer-web3auth-login';
import { USDCBalanceCard } from '@/components/public/usdc-balance-card';
import { WalletDelegationButton } from '@/components/public/wallet-delegation-button';
import { CustomerSmartAccountActions } from '@/components/public/customer-smart-account-actions';
import { Wallet, Copy, ExternalLink, TestTube } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useToast } from '@/components/ui/use-toast';
import { logger } from '@/lib/core/logger/logger-utils';

// Safe Web3Auth user hook - removed as it violates React hooks rules
// Instead, we'll handle the error at the component level

export default function CustomerWalletPage() {
  const [web3AuthAddress, setWeb3AuthAddress] = useState<string | null>(null);
  const [isRequestingFaucet, setIsRequestingFaucet] = useState(false);
  const { toast } = useToast();

  // Web3Auth hooks - must be called unconditionally at the top level
  const userResult = useWeb3AuthUser();
  const authResult = useWeb3Auth();

  // Extract values with proper error handling
  const userInfo = userResult?.userInfo || null;
  const isConnected = authResult?.isConnected || false;
  const web3Auth = authResult?.web3Auth || null;

  // Wagmi hook for account info
  const { address: wagmiAddress } = useAccount();

  // Get wallet address directly from Web3Auth provider when connected
  useEffect(() => {
    async function getWeb3AuthAddress() {
      if (isConnected && web3Auth?.provider) {
        try {
          // Get accounts directly from Web3Auth provider
          const accounts = (await web3Auth.provider.request({
            method: 'eth_accounts',
          })) as string[];

          if (accounts && Array.isArray(accounts) && accounts.length > 0) {
            logger.debug('Customer wallet address from Web3Auth', { address: accounts[0] });
            setWeb3AuthAddress(accounts[0]);
          } else {
            logger.warn('No accounts found in Web3Auth provider');
          }
        } catch (error) {
          logger.error('Failed to get address from Web3Auth provider', error);
        }
      }
    }

    if (isConnected) {
      getWeb3AuthAddress();
    }
  }, [isConnected, web3Auth]);

  const walletAddress = web3AuthAddress || wagmiAddress;

  const copyAddress = async () => {
    if (walletAddress) {
      await navigator.clipboard.writeText(walletAddress);
      toast({
        title: 'Address Copied',
        description: 'Wallet address copied to clipboard',
      });
    }
  };

  const requestTestUSDC = async () => {
    if (!walletAddress) return;

    setIsRequestingFaucet(true);

    try {
      const response = await fetch('/api/faucet', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          wallet_address: walletAddress,
          amount: 100, // Request 100 test USDC
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to request test USDC');
      }

      const result = await response.json();

      toast({
        title: 'Test USDC Requested',
        description: `Successfully requested 100 test USDC. Transaction: ${result.transaction_id?.slice(0, 10)}...`,
      });
    } catch (error) {
      logger.error('Failed to request test USDC', error);
      toast({
        title: 'Request Failed',
        description: 'Failed to request test USDC. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setIsRequestingFaucet(false);
    }
  };

  // Show authentication form if not connected
  if (!isConnected || !userInfo) {
    return (
      <div className="flex-1 container mx-auto p-8 space-y-8">
        <div className="max-w-2xl mx-auto">
          <div className="text-center mb-8">
            <Wallet className="h-16 w-16 mx-auto mb-4 text-purple-600" />
            <h1 className="text-4xl font-bold mb-2">My Wallet</h1>
            <p className="text-lg text-muted-foreground">
              Manage your embedded Web3Auth wallet and crypto assets
            </p>
          </div>

          <div className="bg-white dark:bg-neutral-800 rounded-lg border p-8 shadow-sm">
            <div className="text-center mb-6">
              <h2 className="text-2xl font-semibold mb-2">Sign In Required</h2>
              <p className="text-muted-foreground">
                Please sign in with Web3Auth to access your wallet.
              </p>
            </div>

            <CustomerWeb3AuthLogin autoConnect={false} redirectTo="/public/wallet" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 container mx-auto p-8 space-y-8">
      <div className="flex items-center gap-3 mb-8">
        <Wallet className="h-8 w-8 text-purple-600" />
        <div>
          <h1 className="text-4xl font-bold">My Wallet</h1>
          <p className="text-lg text-muted-foreground">
            Your embedded Web3Auth wallet and crypto assets
          </p>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        {/* Smart Account Actions */}
        <div className="md:col-span-2">
          <CustomerSmartAccountActions />
        </div>

        {/* Wallet Info Card */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Wallet className="h-5 w-5" />
              Wallet Information
            </CardTitle>
            <CardDescription>Your embedded wallet created with Web3Auth</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">Wallet Address</label>
              <div className="flex items-center gap-2 mt-1">
                <code className="flex-1 px-3 py-2 bg-muted rounded-md text-sm font-mono">
                  {walletAddress
                    ? `${walletAddress.slice(0, 6)}...${walletAddress.slice(-4)}`
                    : 'Loading...'}
                </code>
                <Button variant="outline" size="sm" onClick={copyAddress} disabled={!walletAddress}>
                  <Copy className="h-4 w-4" />
                </Button>
                {walletAddress && (
                  <Button variant="outline" size="sm" asChild>
                    <a
                      href={`https://sepolia.basescan.org/address/${walletAddress}`}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      <ExternalLink className="h-4 w-4" />
                    </a>
                  </Button>
                )}
              </div>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">Wallet Type</label>
              <p className="mt-1 text-sm">Web3Auth Embedded Wallet</p>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">Network</label>
              <p className="mt-1 text-sm">Base Sepolia (Testnet)</p>
            </div>
          </CardContent>
        </Card>

        {/* USDC Balance Card */}
        {walletAddress && <USDCBalanceCard />}
      </div>

      {/* Actions Section */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Test USDC Faucet */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <TestTube className="h-5 w-5" />
              Test USDC Faucet
            </CardTitle>
            <CardDescription>Request test USDC tokens for testing subscriptions</CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              onClick={requestTestUSDC}
              disabled={!walletAddress || isRequestingFaucet}
              className="w-full"
            >
              {isRequestingFaucet ? 'Requesting...' : 'Request 100 Test USDC'}
            </Button>
            <p className="text-xs text-muted-foreground mt-2">
              Test USDC has no real value and is only used for testing purposes.
            </p>
          </CardContent>
        </Card>

        {/* Delegation Controls */}
        {walletAddress && (
          <Card>
            <CardHeader>
              <CardTitle>Smart Account Delegation</CardTitle>
              <CardDescription>Manage permissions for your smart account</CardDescription>
            </CardHeader>
            <CardContent>
              <WalletDelegationButton />
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
