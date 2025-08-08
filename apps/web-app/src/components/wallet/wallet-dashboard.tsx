'use client';

import React, { useState } from 'react';
import { usePrivy } from '@privy-io/react-auth';
import { usePrivySmartAccount } from '@/hooks/privy/use-privy-smart-account';
import { WalletBalance } from './wallet-balance';
import { TransactionHistory } from './transaction-history';
import { SendTransaction } from './send-transaction';
import { NetworkSwitcher } from './network-switcher';

interface WalletDashboardProps {
  className?: string;
}

type ActiveTab = 'overview' | 'send' | 'history';

export const WalletDashboard: React.FC<WalletDashboardProps> = ({ className = '' }) => {
  const { authenticated } = usePrivy();
  const { 
    smartAccountReady,
    smartAccountAddress,
    isDeployed,
    switchNetwork,
    currentChainId,
  } = usePrivySmartAccount();

  const [activeTab, setActiveTab] = useState<ActiveTab>('overview');
  const [isNetworkSwitching, setIsNetworkSwitching] = useState(false);

  const handleNetworkSwitch = async (chainId: number) => {
    if (isNetworkSwitching) return;

    try {
      setIsNetworkSwitching(true);
      await switchNetwork(chainId);
    } catch (error) {
      console.error('Failed to switch network:', error);
    } finally {
      setIsNetworkSwitching(false);
    }
  };

  const handleTransactionSent = (hash: string) => {
    // Switch to history tab to show the new transaction
    setActiveTab('history');
  };

  if (!authenticated) {
    return (
      <div className={`bg-yellow-50 border border-yellow-200 rounded-lg p-6 ${className}`}>
        <div className="text-center">
          <h2 className="text-lg font-semibold text-yellow-800 mb-2">
            Wallet Dashboard
          </h2>
          <p className="text-yellow-700">
            Please authenticate with Privy to access your wallet dashboard.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white rounded-lg shadow-lg ${className}`}>
      {/* Header */}
      <div className="p-6 border-b border-gray-200">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Wallet Dashboard</h1>
            <p className="text-gray-600">
              Manage your smart account and transactions
            </p>
          </div>
          
          {/* Network Switcher */}
          <div className="max-w-xs">
            <NetworkSwitcher
              currentChainId={currentChainId}
              onNetworkSwitch={handleNetworkSwitch}
              isNetworkSwitching={isNetworkSwitching}
            />
          </div>
        </div>

        {/* Status Indicators */}
        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2">
            <div className={`w-2 h-2 rounded-full ${
              smartAccountReady ? 'bg-green-500' : 'bg-yellow-500'
            }`} />
            <span className="text-sm text-gray-600">
              {smartAccountReady ? 'Wallet Ready' : 'Initializing...'}
            </span>
          </div>
          
          {smartAccountAddress && (
            <div className="flex items-center space-x-2">
              <div className={`w-2 h-2 rounded-full ${
                isDeployed ? 'bg-green-500' : 'bg-orange-500'
              }`} />
              <span className="text-sm text-gray-600">
                {isDeployed ? 'Deployed' : 'Not Deployed'}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Navigation Tabs */}
      <div className="px-6 border-b border-gray-200">
        <nav className="flex space-x-8">
          {[
            { id: 'overview', label: 'Overview', icon: '📊' },
            { id: 'send', label: 'Send', icon: '💸' },
            { id: 'history', label: 'History', icon: '📋' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as ActiveTab)}
              className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
              disabled={!smartAccountReady}
            >
              <span className="mr-2">{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="p-6">
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Wallet Balance */}
            <WalletBalance />
            
            {/* Quick Actions */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <button
                onClick={() => setActiveTab('send')}
                disabled={!smartAccountReady || !isDeployed}
                className={`p-4 border rounded-lg text-left transition-colors ${
                  !smartAccountReady || !isDeployed
                    ? 'border-gray-200 text-gray-400 cursor-not-allowed'
                    : 'border-gray-200 hover:border-blue-300 hover:bg-blue-50'
                }`}
              >
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center text-blue-600">
                    💸
                  </div>
                  <div>
                    <h3 className="font-medium">Send Transaction</h3>
                    <p className="text-sm text-gray-600">
                      Send ETH or tokens to another address
                    </p>
                  </div>
                </div>
              </button>

              <button
                onClick={() => setActiveTab('history')}
                disabled={!smartAccountReady}
                className={`p-4 border rounded-lg text-left transition-colors ${
                  !smartAccountReady
                    ? 'border-gray-200 text-gray-400 cursor-not-allowed'
                    : 'border-gray-200 hover:border-blue-300 hover:bg-blue-50'
                }`}
              >
                <div className="flex items-center space-x-3">
                  <div className="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center text-green-600">
                    📋
                  </div>
                  <div>
                    <h3 className="font-medium">Transaction History</h3>
                    <p className="text-sm text-gray-600">
                      View your recent transactions
                    </p>
                  </div>
                </div>
              </button>
            </div>

            {/* Recent Transactions Preview */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold">Recent Transactions</h3>
                <button
                  onClick={() => setActiveTab('history')}
                  className="text-blue-600 hover:text-blue-800 text-sm"
                >
                  View All →
                </button>
              </div>
              <TransactionHistory limit={3} />
            </div>
          </div>
        )}

        {activeTab === 'send' && (
          <SendTransaction 
            onTransactionSent={handleTransactionSent}
          />
        )}

        {activeTab === 'history' && (
          <TransactionHistory limit={20} />
        )}
      </div>

      {/* Footer Info */}
      {smartAccountAddress && (
        <div className="px-6 py-4 bg-gray-50 rounded-b-lg">
          <div className="text-sm text-gray-600">
            <div className="flex items-center justify-between">
              <span>Smart Account:</span>
              <div className="flex items-center space-x-2">
                <code className="text-xs bg-white px-2 py-1 rounded">
                  {`${smartAccountAddress.slice(0, 6)}...${smartAccountAddress.slice(-4)}`}
                </code>
                <button
                  onClick={() => navigator.clipboard.writeText(smartAccountAddress)}
                  className="text-blue-600 hover:text-blue-800"
                >
                  Copy
                </button>
              </div>
            </div>
            <div className="mt-2 text-xs text-gray-500">
              Powered by MetaMask Delegation Toolkit & Pimlico Paymaster
            </div>
          </div>
        </div>
      )}
    </div>
  );
};