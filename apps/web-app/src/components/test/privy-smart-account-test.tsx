'use client';

import React, { useState } from 'react';
import { usePrivy } from '@privy-io/react-auth';
import { usePrivySmartAccount } from '@/hooks/privy/use-privy-smart-account';
import { logger } from '@/lib/core/logger/logger-utils';
import { formatEther, parseEther } from 'viem';

export const PrivySmartAccountTest: React.FC = () => {
  const { authenticated } = usePrivy();
  const {
    smartAccount,
    smartAccountAddress,
    smartAccountReady,
    isDeployed,
    bundlerClient,
    deploySmartAccount,
    checkDeploymentStatus,
  } = usePrivySmartAccount();

  const [isDeploying, setIsDeploying] = useState(false);
  const [isSendingTx, setIsSendingTx] = useState(false);
  const [txHash, setTxHash] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleDeploy = async () => {
    setIsDeploying(true);
    setError(null);
    try {
      logger.log('🚀 Starting smart account deployment...');
      await deploySmartAccount();
      logger.log('✅ Smart account deployed successfully!');
    } catch (err: any) {
      logger.error('❌ Deployment failed:', err);
      setError(err.message || 'Failed to deploy smart account');
    } finally {
      setIsDeploying(false);
    }
  };

  const handleCheckStatus = async () => {
    try {
      const deployed = await checkDeploymentStatus();
      logger.log(`📍 Deployment status: ${deployed ? 'DEPLOYED' : 'NOT DEPLOYED'}`);
    } catch (err: any) {
      logger.error('❌ Failed to check status:', err);
      setError(err.message || 'Failed to check deployment status');
    }
  };

  const handleSendTestTransaction = async () => {
    if (!bundlerClient || !smartAccount) {
      setError('Smart account not ready');
      return;
    }

    setIsSendingTx(true);
    setError(null);
    setTxHash(null);

    try {
      logger.log('📤 Sending test transaction...');
      
      // Send a minimal self-transfer to test the smart account
      const userOpHash = await bundlerClient.sendUserOperation({
        account: smartAccount,
        calls: [{
          to: smartAccountAddress!,
          value: BigInt(0),
          data: '0x' as `0x${string}`,
        }],
      });

      logger.log('⏳ UserOperation sent:', userOpHash);
      
      // Wait for receipt
      const receipt = await bundlerClient.waitForUserOperationReceipt({
        hash: userOpHash,
        timeout: 60000,
      });

      const hash = receipt.receipt.transactionHash;
      setTxHash(hash);
      logger.log('✅ Transaction successful! Hash:', hash);
    } catch (err: any) {
      logger.error('❌ Transaction failed:', err);
      setError(err.message || 'Failed to send transaction');
    } finally {
      setIsSendingTx(false);
    }
  };

  if (!authenticated) {
    return (
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
        <p className="text-yellow-800">Please authenticate first to test smart account features.</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-2xl font-bold mb-4">Smart Account Testing</h2>
      
      {/* Status Section */}
      <div className="mb-6 p-4 bg-gray-50 rounded-lg">
        <h3 className="font-semibold mb-3">Smart Account Status:</h3>
        <div className="space-y-2 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium">Ready:</span>
            <span className={`px-2 py-1 rounded ${smartAccountReady ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
              {smartAccountReady ? 'Yes' : 'Initializing...'}
            </span>
          </div>
          
          <div className="flex items-center justify-between">
            <span className="font-medium">Bundler Client:</span>
            <span className={`px-2 py-1 rounded ${bundlerClient ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
              {bundlerClient ? 'Connected' : 'Not Connected'}
            </span>
          </div>
          
          {smartAccountAddress && (
            <>
              <div className="flex items-start justify-between">
                <span className="font-medium">Address:</span>
                <code className="bg-gray-100 px-2 py-1 rounded text-xs ml-2 break-all">
                  {smartAccountAddress}
                </code>
              </div>
              
              <div className="flex items-center justify-between">
                <span className="font-medium">Deployed:</span>
                <span className={`px-2 py-1 rounded ${
                  isDeployed === null ? 'bg-gray-100 text-gray-800' :
                  isDeployed ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                }`}>
                  {isDeployed === null ? 'Checking...' : isDeployed ? 'Yes' : 'No'}
                </span>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-800 text-sm">{error}</p>
        </div>
      )}

      {/* Success Display */}
      {txHash && (
        <div className="mb-4 p-3 bg-green-50 border border-green-200 rounded-lg">
          <p className="text-green-800 text-sm">
            Transaction successful!
            <br />
            <a 
              href={`https://sepolia.basescan.org/tx/${txHash}`}
              target="_blank"
              rel="noopener noreferrer"
              className="underline hover:text-green-900"
            >
              View on Explorer →
            </a>
          </p>
        </div>
      )}

      {/* Action Buttons */}
      <div className="space-y-3">
        {/* Check Status Button */}
        <button
          onClick={handleCheckStatus}
          disabled={!smartAccountReady}
          className="w-full px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Check Deployment Status
        </button>

        {/* Deploy Button */}
        {!isDeployed && smartAccountReady && (
          <button
            onClick={handleDeploy}
            disabled={isDeploying || !bundlerClient}
            className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isDeploying ? 'Deploying...' : 'Deploy Smart Account'}
          </button>
        )}

        {/* Test Transaction Button */}
        {isDeployed && (
          <button
            onClick={handleSendTestTransaction}
            disabled={isSendingTx || !bundlerClient}
            className="w-full px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSendingTx ? 'Sending Transaction...' : 'Send Test Transaction'}
          </button>
        )}
      </div>

      {/* Info Section */}
      <div className="mt-6 p-4 bg-blue-50 rounded-lg">
        <h4 className="font-semibold text-blue-900 mb-2">ℹ️ How This Works:</h4>
        <ol className="text-sm text-blue-800 space-y-1 list-decimal list-inside">
          <li>Privy creates an embedded wallet (EOA) for you</li>
          <li>We use this EOA to create a MetaMask Smart Account</li>
          <li>The smart account uses Pimlico for bundler operations</li>
          <li>Deploy the account by sending your first transaction</li>
          <li>Once deployed, you can create delegations for subscriptions</li>
        </ol>
      </div>

      {/* Debug Information */}
      <div className="mt-6 p-4 bg-gray-100 rounded-lg">
        <h4 className="font-semibold mb-2">Debug Information</h4>
        <pre className="text-xs overflow-auto bg-white p-3 rounded">
          {JSON.stringify({
            smartAccountReady,
            smartAccountAddress,
            isDeployed,
            hasBundlerClient: !!bundlerClient,
            hasSmartAccount: !!smartAccount,
            smartAccountType: smartAccount ? smartAccount.constructor.name : null,
          }, null, 2)}
        </pre>
      </div>
    </div>
  );
};