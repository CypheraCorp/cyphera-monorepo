'use client';

import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  useWeb3Auth,
  useWeb3AuthUser,
  useWalletUI,
  useWeb3AuthDisconnect,
} from '@web3auth/modal/react';
import { useAccount } from 'wagmi';
import { useQueryClient } from '@tanstack/react-query';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Wallet,
  Copy,
  ExternalLink,
  Settings,
  LogOut,
  CheckCircle,
  Circle,
  Eye,
  Send,
  History,
  Building2,
} from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import { formatAddress } from '@/lib/utils/circle';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import { logger } from '@/lib/core/logger/logger-utils';
interface CustomerWalletDropdownProps {
  className?: string;
}

export function CustomerWalletDropdown({ className = '' }: CustomerWalletDropdownProps) {
  const router = useRouter();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [web3AuthAddress, setWeb3AuthAddress] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [hasAttemptedFetch, setHasAttemptedFetch] = useState(false);
  const [isSwitchingRole, setIsSwitchingRole] = useState(false);

  // Web3Auth hooks
  const { web3Auth, isConnected } = useWeb3Auth();
  const { userInfo } = useWeb3AuthUser();
  const { showWalletUI, loading: walletUILoading } = useWalletUI();
  const { disconnect } = useWeb3AuthDisconnect();

  // Wagmi hook for account info
  const { address: wagmiAddress } = useAccount();

  // Memoize the wallet address - prefer Web3Auth address over wagmi
  const walletAddress = useMemo(() => {
    return web3AuthAddress || wagmiAddress;
  }, [web3AuthAddress, wagmiAddress]);

  // Memoized handlers to prevent recreation on every render
  const handleCopyAddress = useCallback(() => {
    if (walletAddress) {
      navigator.clipboard.writeText(walletAddress);
      toast({
        title: 'Address Copied',
        description: 'Wallet address copied to clipboard',
      });
    }
  }, [walletAddress, toast]);

  const handleViewOnExplorer = useCallback(() => {
    if (walletAddress) {
      // For now, defaulting to Ethereum mainnet explorer
      // This should be updated based on the current network
      const explorerUrl = `https://etherscan.io/address/${walletAddress}`;
      window.open(explorerUrl, '_blank');
    }
  }, [walletAddress]);

  const handleViewWalletDetails = useCallback(async () => {
    try {
      await showWalletUI({ show: true });
    } catch (error) {
      logger.error('❌ Error opening wallet UI:', { error });
      toast({
        title: 'Error',
        description: 'Failed to open wallet details. Please try again.',
        variant: 'destructive',
      });
    }
  }, [showWalletUI, toast]);

  const handleSwitchToMerchant = useCallback(async () => {
    if (!isConnected || !walletAddress) {
      toast({
        title: 'Connection Required',
        description: 'Please connect your wallet first',
        variant: 'destructive',
      });
      return;
    }

    setIsSwitchingRole(true);

    try {
      // Check if user already has a merchant session
      const checkResponse = await fetch('/api/auth/me', {
        method: 'GET',
        credentials: 'include',
      });

      if (checkResponse.ok) {
        // User already has merchant session, redirect to dashboard
        router.push('/merchants/dashboard');
        return;
      }

      // Try to sign in as merchant
      const signinResponse = await fetch('/api/auth/signin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          wallet_address: walletAddress,
          user_info: userInfo,
        }),
      });

      if (signinResponse.ok) {
        const data = await signinResponse.json();

        // Check if merchant needs onboarding
        if (!data.user.finished_onboarding) {
          router.push('/merchants/onboarding');
        } else {
          router.push('/merchants/dashboard');
        }
      } else {
        // If signin fails, redirect to merchant signin page
        router.push('/merchants/signin');
      }
    } catch (error) {
      logger.error('❌ Error switching to merchant:', { error });
      toast({
        title: 'Switch Failed',
        description: 'Failed to switch to merchant mode. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setIsSwitchingRole(false);
    }
  }, [isConnected, walletAddress, userInfo, router, toast]);

  const handleLogout = useCallback(async () => {
    try {
      logger.log('🔄 Starting comprehensive customer logout process...');

      // 1. Set logout flag to prevent auto-signin
      if (typeof window !== 'undefined') {
        window.localStorage.setItem('web3auth-customer-logout', 'true');
      }

      // 2. Call logout endpoint to clear server-side session
      await fetch('/api/auth/customer/logout', {
        method: 'POST',
        credentials: 'include',
      });
      logger.log('✅ Server session cleared');

      // 3. Clear React Query cache
      await queryClient.clear();
      logger.log('✅ React Query cache cleared');

      // 4. Clear API cache (call endpoint to clear server-side cache)
      try {
        await fetch('/api/cache/clear', {
          method: 'POST',
          credentials: 'include',
        });
        logger.log('✅ API cache cleared');
      } catch (apiCacheError) {
        logger.warn('⚠️ API cache clear failed (endpoint may not exist):', {
          error: apiCacheError,
        });
      }

      // 5. Disconnect from Web3Auth
      try {
        await disconnect();
        logger.log('✅ Web3Auth disconnected');
      } catch (web3AuthError) {
        logger.warn('⚠️ Web3Auth disconnect failed (may already be disconnected):', {
          error: web3AuthError,
        });
      }

      // 6. Clear local storage and session storage
      if (typeof window !== 'undefined') {
        // Clear Web3Auth related items
        const keysToRemove = [
          'web3auth-customer-logout',
          'openlogin_store',
          'Web3Auth-cachedAdapter',
          'web3auth_token',
          'web3auth_session',
        ];

        keysToRemove.forEach((key) => {
          try {
            localStorage.removeItem(key);
            sessionStorage.removeItem(key);
          } catch (storageError) {
            logger.warn(`⚠️ Could not clear ${key}:`, { error: storageError });
          }
        });

        // Clear any authentication cookies by setting them to expire
        document.cookie =
          'cyphera-customer-session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
        document.cookie = 'cyphera-session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';

        logger.log('✅ Local storage and cookies cleared');
      }

      // 7. Clear component state
      setWeb3AuthAddress(null);
      setHasAttemptedFetch(false);
      setIsSwitchingRole(false);

      // 8. Force a small delay to ensure cleanup completes
      await new Promise((resolve) => setTimeout(resolve, 500));

      // 9. Determine where to redirect based on current page
      const currentPath = window.location.pathname;

      // Check if we're on a public product page (e.g., /pay/[productId] or /public/prices/[priceId])
      if (currentPath.startsWith('/public/prices/') || currentPath.startsWith('/pay/')) {
        logger.log('🔄 Staying on product page after logout...');
        // Force reload to update the UI state
        window.location.reload();
      } else {
        // For all other pages (customer dashboard, etc), redirect to signin
        logger.log('🔄 Redirecting to signin page...');
        window.location.href = '/customers/signin';
      }

      logger.log('✅ Customer logged out successfully');
    } catch (error) {
      logger.error('❌ Customer logout failed:', { error });
      toast({
        title: 'Logout Failed',
        description: 'Please try again',
        variant: 'destructive',
      });

      // On failure, check current path for redirect logic
      const currentPath = window.location.pathname;
      if (currentPath.startsWith('/public/prices/') || currentPath.startsWith('/pay/')) {
        window.location.reload();
      } else {
        window.location.href = '/customers/signin';
      }
    }
  }, [disconnect, queryClient, toast]);

  // Optimized wallet address fetching with debouncing and race condition prevention
  useEffect(() => {
    // Prevent multiple simultaneous fetch attempts
    if (hasAttemptedFetch) return;

    async function getWeb3AuthAddress() {
      if (!isConnected || !web3Auth?.provider) {
        setWeb3AuthAddress(null);
        setIsLoading(false);
        return;
      }

      try {
        setHasAttemptedFetch(true);

        // Add small delay to prevent race conditions
        await new Promise((resolve) => setTimeout(resolve, 50));

        const accounts = (await web3Auth.provider.request({
          method: 'eth_accounts',
        })) as string[];

        if (accounts && Array.isArray(accounts) && accounts.length > 0) {
          setWeb3AuthAddress(accounts[0]);
        }
      } catch (error) {
        logger.error('❌ Failed to get address from Web3Auth provider:', { error });
        setWeb3AuthAddress(null);
      } finally {
        setIsLoading(false);
      }
    }

    // Debounce the address fetch
    const timeoutId = setTimeout(getWeb3AuthAddress, 100);
    return () => clearTimeout(timeoutId);
  }, [isConnected, web3Auth?.provider, hasAttemptedFetch]);

  // Reset fetch attempt when connection state changes
  useEffect(() => {
    if (!isConnected) {
      setHasAttemptedFetch(false);
      setWeb3AuthAddress(null);
      setIsLoading(false);
    }
  }, [isConnected]);

  // Memoized loading state to prevent unnecessary re-renders
  if (isLoading) {
    return (
      <div className={`flex items-center gap-2 ${className}`}>
        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-purple-600"></div>
        <span className="text-sm text-gray-600">Loading wallet...</span>
      </div>
    );
  }

  // Memoized not connected state
  if (!isConnected || !walletAddress) {
    return (
      <div className={`flex items-center gap-2 ${className}`}>
        <Circle className="h-4 w-4 text-gray-400" />
        <span className="text-sm text-gray-600">Wallet Not Connected</span>
      </div>
    );
  }

  return (
    <div className={`flex items-center ${className}`}>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" className="flex items-center gap-2 h-9">
            {/* Connection Status Indicator */}
            <CheckCircle className="h-4 w-4 text-green-500" />

            {/* Wallet Address */}
            <span className="font-mono text-sm">{formatAddress(walletAddress)}</span>

            {/* Wallet Icon */}
            <Wallet className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent align="end" className="w-64">
          {/* User Info Section */}
          {userInfo && (
            <>
              <div className="flex items-center gap-3 p-3">
                {userInfo.profileImage ? (
                  <Image
                    src={userInfo.profileImage}
                    alt="Profile"
                    width={32}
                    height={32}
                    className="rounded-full"
                  />
                ) : (
                  <div className="w-8 h-8 rounded-full bg-purple-100 flex items-center justify-center">
                    <Wallet className="h-4 w-4 text-purple-600" />
                  </div>
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{userInfo.name || 'Customer'}</p>
                  <p className="text-xs text-gray-500 truncate">{userInfo.email}</p>
                </div>
              </div>
              <DropdownMenuSeparator />
            </>
          )}

          {/* Wallet Info Section */}
          <div className="p-3">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-gray-500">WALLET ADDRESS</span>
              <Badge variant="secondary" className="text-xs">
                <CheckCircle className="h-3 w-3 mr-1 text-green-500" />
                Connected
              </Badge>
            </div>
            <div className="flex items-center gap-2 p-2 bg-gray-50 rounded-md">
              <code className="text-xs font-mono flex-1 truncate">{walletAddress}</code>
              <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={handleCopyAddress}>
                <Copy className="h-3 w-3" />
              </Button>
            </div>
          </div>

          <DropdownMenuSeparator />

          {/* Wallet Actions */}
          <DropdownMenuLabel>Wallet Actions</DropdownMenuLabel>

          <DropdownMenuItem onClick={handleViewWalletDetails} disabled={walletUILoading}>
            <Eye className="h-4 w-4 mr-2" />
            {walletUILoading ? 'Opening...' : 'View Wallet Details'}
          </DropdownMenuItem>

          <DropdownMenuItem onClick={handleViewOnExplorer}>
            <ExternalLink className="h-4 w-4 mr-2" />
            View on Explorer
          </DropdownMenuItem>

          <DropdownMenuItem onClick={() => router.push('/customers/wallet?tab=send')}>
            <Send className="h-4 w-4 mr-2" />
            Send Funds
          </DropdownMenuItem>

          <DropdownMenuItem onClick={() => router.push('/customers/wallet?tab=history')}>
            <History className="h-4 w-4 mr-2" />
            Transaction History
          </DropdownMenuItem>

          <DropdownMenuSeparator />

          {/* Role Switch */}
          <DropdownMenuLabel>Switch Role</DropdownMenuLabel>

          <DropdownMenuItem onClick={handleSwitchToMerchant} disabled={isSwitchingRole}>
            <Building2 className="h-4 w-4 mr-2" />
            {isSwitchingRole ? 'Switching...' : 'Switch to Merchant'}
          </DropdownMenuItem>

          <DropdownMenuSeparator />

          {/* Settings and Logout */}
          <DropdownMenuItem onClick={() => router.push('/customers/settings')}>
            <Settings className="h-4 w-4 mr-2" />
            Settings
          </DropdownMenuItem>

          <DropdownMenuItem onClick={handleLogout} className="text-red-600">
            <LogOut className="h-4 w-4 mr-2" />
            Sign Out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
