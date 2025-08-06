'use client';

import React from 'react';
import { usePrivy, useWallets } from '@privy-io/react-auth';
import { logger } from '@/lib/core/logger/logger-utils';

export const PrivyBasicTest: React.FC = () => {
  const { ready, authenticated, user, login, logout } = usePrivy();
  const { wallets } = useWallets();

  // Find the embedded wallet
  const embeddedWallet = wallets.find(
    (wallet) => wallet.walletClientType === 'privy'
  );

  const handleLogin = async () => {
    try {
      logger.log('🔐 Starting Privy login...');
      await login();
      logger.log('✅ Privy login successful');
    } catch (error) {
      logger.error('❌ Privy login failed:', error);
    }
  };

  const handleLogout = async () => {
    try {
      logger.log('🔐 Logging out...');
      await logout();
      logger.log('✅ Logout successful');
    } catch (error) {
      logger.error('❌ Logout failed:', error);
    }
  };

  // Display loading state
  if (!ready) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] p-8">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        <p className="mt-4 text-gray-600">Initializing Privy...</p>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto p-6 space-y-6">
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-2xl font-bold mb-4">Privy Authentication Test</h2>
        
        {/* Authentication Status */}
        <div className="mb-6 p-4 bg-gray-50 rounded-lg">
          <h3 className="font-semibold mb-2">Status:</h3>
          <div className="space-y-2 text-sm">
            <div className="flex items-center">
              <span className="font-medium w-32">Privy Ready:</span>
              <span className={`px-2 py-1 rounded ${ready ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                {ready ? 'Yes' : 'No'}
              </span>
            </div>
            <div className="flex items-center">
              <span className="font-medium w-32">Authenticated:</span>
              <span className={`px-2 py-1 rounded ${authenticated ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                {authenticated ? 'Yes' : 'No'}
              </span>
            </div>
            <div className="flex items-center">
              <span className="font-medium w-32">Embedded Wallet:</span>
              <span className={`px-2 py-1 rounded ${embeddedWallet ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'}`}>
                {embeddedWallet ? 'Created' : 'Not Created'}
              </span>
            </div>
          </div>
        </div>

        {/* User Info */}
        {authenticated && user && (
          <div className="mb-6 p-4 bg-blue-50 rounded-lg">
            <h3 className="font-semibold mb-2">User Info:</h3>
            <div className="space-y-2 text-sm">
              <div>
                <span className="font-medium">User ID:</span> {user.id}
              </div>
              {user.email && (
                <div>
                  <span className="font-medium">Email:</span> {user.email.address}
                </div>
              )}
              {user.phone && (
                <div>
                  <span className="font-medium">Phone:</span> {user.phone.number}
                </div>
              )}
              <div>
                <span className="font-medium">Created At:</span>{' '}
                {new Date(user.createdAt).toLocaleString()}
              </div>
            </div>
          </div>
        )}

        {/* Wallet Info */}
        {embeddedWallet && (
          <div className="mb-6 p-4 bg-purple-50 rounded-lg">
            <h3 className="font-semibold mb-2">Embedded Wallet:</h3>
            <div className="space-y-2 text-sm">
              <div>
                <span className="font-medium">Address:</span>{' '}
                <code className="bg-gray-100 px-2 py-1 rounded text-xs">
                  {embeddedWallet.address}
                </code>
              </div>
              <div>
                <span className="font-medium">Chain ID:</span> {embeddedWallet.chainId}
              </div>
              <div>
                <span className="font-medium">Type:</span> {embeddedWallet.walletClientType}
              </div>
            </div>
          </div>
        )}

        {/* Connected Wallets */}
        {wallets.length > 0 && (
          <div className="mb-6 p-4 bg-green-50 rounded-lg">
            <h3 className="font-semibold mb-2">All Connected Wallets ({wallets.length}):</h3>
            <div className="space-y-2">
              {wallets.map((wallet, index) => (
                <div key={index} className="text-sm p-2 bg-white rounded border">
                  <div className="font-medium">{wallet.walletClientType}</div>
                  <div className="text-xs text-gray-600 mt-1">
                    {wallet.address}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex gap-4">
          {!authenticated ? (
            <button
              onClick={handleLogin}
              className="flex-1 bg-blue-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-blue-700 transition-colors"
            >
              Sign In with Privy
            </button>
          ) : (
            <>
              <button
                onClick={handleLogout}
                className="flex-1 bg-red-600 text-white px-6 py-3 rounded-lg font-medium hover:bg-red-700 transition-colors"
              >
                Sign Out
              </button>
            </>
          )}
        </div>

        {/* Debug Info */}
        <details className="mt-6">
          <summary className="cursor-pointer text-sm text-gray-600 hover:text-gray-800">
            Debug Information
          </summary>
          <div className="mt-2 p-3 bg-gray-100 rounded text-xs font-mono overflow-x-auto">
            <pre>{JSON.stringify({ ready, authenticated, user, wallets }, null, 2)}</pre>
          </div>
        </details>
      </div>
    </div>
  );
};