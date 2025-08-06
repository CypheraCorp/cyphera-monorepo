'use client';

import { PrivyProvider } from '@/components/providers/privy-provider';
import { PrivySmartAccountProvider } from '@/hooks/privy/use-privy-smart-account';
import { PrivyBasicTest } from '@/components/test/privy-basic-test';
import { PrivySmartAccountTest } from '@/components/test/privy-smart-account-test';
import { useState } from 'react';

export default function TestPrivyPage() {
  const [activeTab, setActiveTab] = useState<'auth' | 'smart-account'>('auth');

  return (
    <PrivyProvider>
      <PrivySmartAccountProvider>
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
          <div className="container mx-auto px-4 py-8">
            <h1 className="text-3xl font-bold mb-8 text-center">
              Privy Integration Test
            </h1>
            
            {/* Tab Navigation */}
            <div className="max-w-2xl mx-auto mb-6">
              <div className="flex rounded-lg bg-gray-200 dark:bg-gray-700 p-1">
                <button
                  onClick={() => setActiveTab('auth')}
                  className={`flex-1 px-4 py-2 rounded-md font-medium transition-colors ${
                    activeTab === 'auth'
                      ? 'bg-white dark:bg-gray-800 text-blue-600 dark:text-blue-400'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
                  }`}
                >
                  Authentication
                </button>
                <button
                  onClick={() => setActiveTab('smart-account')}
                  className={`flex-1 px-4 py-2 rounded-md font-medium transition-colors ${
                    activeTab === 'smart-account'
                      ? 'bg-white dark:bg-gray-800 text-blue-600 dark:text-blue-400'
                      : 'text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200'
                  }`}
                >
                  Smart Account
                </button>
              </div>
            </div>
            
            {/* Tab Content */}
            <div className="max-w-2xl mx-auto">
              {activeTab === 'auth' ? (
                <PrivyBasicTest />
              ) : (
                <PrivySmartAccountTest />
              )}
            </div>
          </div>
        </div>
      </PrivySmartAccountProvider>
    </PrivyProvider>
  );
}