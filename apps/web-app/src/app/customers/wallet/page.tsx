'use client';

import { useState, useEffect } from 'react';
import { useWeb3AuthUser, useWeb3Auth, useWalletUI } from '@web3auth/modal/react';
import { useAccount } from 'wagmi';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Wallet,
  Copy,
  ExternalLink,
  RefreshCw,
  Send,
  ArrowDownToLine,
  Settings,
} from 'lucide-react';
import Link from 'next/link';
import { logger } from '@/lib/core/logger/logger-utils';

// Safe Web3Auth user hook
function useSafeCustomerAuth() {
  const { userInfo } = useWeb3AuthUser();
  const { isConnected } = useWeb3Auth();
  return { userInfo, isConnected };
}

export default function CustomerWalletPage() {
  const [isClient, setIsClient] = useState(false);
  const { userInfo, isConnected } = useSafeCustomerAuth();
  const { address: walletAddress } = useAccount();
  const { showWalletUI, loading: walletUILoading, error: walletUIError } = useWalletUI();

  useEffect(() => {
    setIsClient(true);
  }, []);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    // You could add a toast notification here
  };

  const formatAddress = (address: string) => {
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  const handleWalletSettings = async () => {
    try {
      await showWalletUI();
    } catch (error) {
      logger.error('Error opening wallet UI', error);
    }
  };

  if (!isClient) {
    return (
      <div className="container mx-auto p-8">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600 mx-auto"></div>
      </div>
    );
  }

  if (!isConnected || !userInfo) {
    return (
      <div className="container mx-auto p-8">
        <Card>
          <CardHeader>
            <CardTitle>Authentication Required</CardTitle>
            <CardDescription>Please sign in to access your wallet</CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild>
              <Link href="/customers/signin">Sign In</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-8 space-y-8">
      {/* Header */}
      <div className="flex items-center gap-3 mb-8">
        <Wallet className="h-8 w-8 text-blue-600" />
        <div>
          <h1 className="text-4xl font-bold">My Wallet</h1>
          <p className="text-lg text-muted-foreground">
            Manage your crypto wallet and view balances
          </p>
        </div>
      </div>

      {/* Wallet Status */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Wallet className="h-5 w-5" />
            Wallet Connection
            <Badge variant="outline" className="ml-auto">
              <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
              Connected
            </Badge>
          </CardTitle>
          <CardDescription>Your Web3Auth wallet is connected and ready to use</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium text-muted-foreground">Wallet Address</label>
            <div className="flex items-center gap-2 mt-1">
              <code className="flex-1 px-3 py-2 bg-muted rounded-md text-sm font-mono">
                {walletAddress ? formatAddress(walletAddress) : 'Loading...'}
              </code>
              {walletAddress && (
                <Button variant="outline" size="sm" onClick={() => copyToClipboard(walletAddress)}>
                  <Copy className="h-4 w-4" />
                </Button>
              )}
            </div>
          </div>

          <div>
            <label className="text-sm font-medium text-muted-foreground">Connected Account</label>
            <p className="text-sm mt-1">{userInfo.email || 'Not provided'}</p>
          </div>

          <div>
            <label className="text-sm font-medium text-muted-foreground">Wallet Type</label>
            <p className="text-sm mt-1">Web3Auth Embedded Wallet</p>
          </div>
        </CardContent>
      </Card>

      {/* Balance Overview */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">USDC Balance</CardTitle>
            <RefreshCw className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">$0.00</div>
            <p className="text-xs text-muted-foreground">0.000000 USDC</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">ETH Balance</CardTitle>
            <RefreshCw className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">$0.00</div>
            <p className="text-xs text-muted-foreground">0.000000 ETH</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Value</CardTitle>
            <Wallet className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">$0.00</div>
            <p className="text-xs text-muted-foreground">USD Value</p>
          </CardContent>
        </Card>
      </div>

      {/* Wallet Actions */}
      <div className="grid gap-6 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ArrowDownToLine className="h-5 w-5" />
              Receive Funds
            </CardTitle>
            <CardDescription>Get your wallet address to receive crypto payments</CardDescription>
          </CardHeader>
          <CardContent>
            <Button className="w-full" variant="outline">
              Show QR Code
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Send className="h-5 w-5" />
              Send Funds
            </CardTitle>
            <CardDescription>Send crypto to another wallet address</CardDescription>
          </CardHeader>
          <CardContent>
            <Button className="w-full" disabled>
              Coming Soon
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Wallet Settings
            </CardTitle>
            <CardDescription>
              Access your Web3Auth wallet settings and configuration
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              className="w-full"
              variant="outline"
              onClick={handleWalletSettings}
              disabled={walletUILoading}
            >
              {walletUILoading ? (
                <>
                  <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  Opening Wallet UI...
                </>
              ) : (
                <>
                  <Settings className="mr-2 h-4 w-4" />
                  Wallet Settings
                </>
              )}
            </Button>
            {walletUIError && (
              <div className="mt-2 text-sm text-red-600">Error: {walletUIError.message}</div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Recent Transactions */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Transactions</CardTitle>
          <CardDescription>Your recent wallet activity and transactions</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-12">
            <Wallet className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Transactions Yet</h3>
            <p className="text-muted-foreground text-center mb-4">
              Your transaction history will appear here once you start using your wallet.
            </p>
            <Button asChild variant="outline">
              <Link href="/customers/subscriptions">
                <ExternalLink className="mr-2 h-4 w-4" />
                View Subscriptions
              </Link>
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Wallet Security */}
      <Card>
        <CardHeader>
          <CardTitle>Wallet Security</CardTitle>
          <CardDescription>Your wallet security and backup information</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
            <div>
              <h4 className="font-medium">Web3Auth Protection</h4>
              <p className="text-sm text-muted-foreground">
                Your wallet is secured by Web3Auth&apos;s decentralized infrastructure
              </p>
            </div>
            <Badge variant="outline" className="text-green-600">
              Secured
            </Badge>
          </div>

          <div className="flex items-center justify-between p-4 bg-muted rounded-lg">
            <div>
              <h4 className="font-medium">Social Login Recovery</h4>
              <p className="text-sm text-muted-foreground">
                Access your wallet using your {(userInfo as Record<string, unknown>)?.typeOfLogin as string || 'social'} account
              </p>
            </div>
            <Badge variant="outline" className="text-blue-600">
              Enabled
            </Badge>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
