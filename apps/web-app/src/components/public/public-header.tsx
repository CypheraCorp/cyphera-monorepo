'use client';

import React from 'react';
import { Button } from '@/components/ui/button';
import { useWeb3AuthConnect } from '@web3auth/modal/react';
import { CustomerWalletDropdown } from './customer-wallet-dropdown';
import { useRouter } from 'next/navigation';
import { useWeb3AuthInitialization } from '@/hooks/auth';
import { logger } from '@/lib/core/logger/logger-utils';

interface CustomerData {
  id: string;
  email: string;
  name?: string;
  wallet_address?: string;
  [key: string]: unknown;
}

interface PublicHeaderProps {
  onLoginSuccess?: (customerData: CustomerData) => void;
}

export function PublicHeader({}: PublicHeaderProps) {
  const router = useRouter();
  const { isInitializing, isAuthenticated } = useWeb3AuthInitialization();

  // Web3Auth connect hook - must be called unconditionally at the top level
  const connectResult = useWeb3AuthConnect();
  const connect = connectResult?.connect || null;

  const handleLoginClick = async () => {
    try {
      if (connect) {
        logger.log('🔄 Starting Web3Auth connection...');
        await connect();
        logger.log('✅ Web3Auth connection initiated');
        
        // Add a stabilization delay after login to ensure provider is fully initialized
        logger.log('⏳ Allowing Web3Auth provider to stabilize...');
        await new Promise(resolve => setTimeout(resolve, 2000));
        logger.log('✅ Web3Auth provider stabilization complete');
      } else {
        logger.warn('⚠️ Web3Auth connect function not available');
      }
    } catch (error) {
      logger.error('❌ Web3Auth connection failed:', error);
    }
  };

  return (
    <>
      <header className="bg-white dark:bg-neutral-900 border-b border-neutral-200 dark:border-neutral-700 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          {/* Logo */}
          <div className="flex items-center">
            {!isInitializing && isAuthenticated ? (
              <button
                onClick={() => router.push('/customers/dashboard')}
                className="text-2xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent hover:opacity-80 transition-opacity cursor-pointer"
              >
                Cyphera
              </button>
            ) : (
              <h1 className="text-2xl font-bold bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent">
                Cyphera
              </h1>
            )}
          </div>

          {/* Auth Buttons */}
          <div className="flex items-center gap-4">
            {isInitializing ? (
              // Web3Auth is still initializing - show loading state
              <div className="flex items-center gap-2">
                <div className="animate-pulse flex space-x-2">
                  <div className="h-8 w-16 bg-gray-200 dark:bg-gray-700 rounded"></div>
                  <div className="h-8 w-20 bg-gray-200 dark:bg-gray-700 rounded"></div>
                </div>
              </div>
            ) : isAuthenticated ? (
              // User is logged in - show wallet dropdown
              <CustomerWalletDropdown />
            ) : (
              // User is not logged in
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleLoginClick}
                  className="text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
                >
                  Log In
                </Button>
                <Button
                  size="sm"
                  onClick={handleLoginClick}
                  className="bg-purple-600 hover:bg-purple-700 text-white"
                >
                  Sign Up
                </Button>
              </div>
            )}
          </div>
        </div>
      </header>
    </>
  );
}
