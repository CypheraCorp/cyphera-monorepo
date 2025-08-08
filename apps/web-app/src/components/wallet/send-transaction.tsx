'use client';

import React, { useState, useEffect } from 'react';
import { usePrivySmartAccount } from '@/hooks/privy/use-privy-smart-account';
import { parseEther, isAddress, type Address, formatEther } from 'viem';
import { getUSDCAddress } from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';

interface SendTransactionProps {
  className?: string;
  onTransactionSent?: (hash: string) => void;
}

interface TokenOption {
  symbol: string;
  name: string;
  address: Address | null;
  decimals: number;
  isNative: boolean;
}

export const SendTransaction: React.FC<SendTransactionProps> = ({ 
  className = '',
  onTransactionSent 
}) => {
  const { 
    smartAccount,
    smartAccountAddress,
    smartAccountReady,
    bundlerClient,
    pimlicoClient,
    currentChainId,
  } = usePrivySmartAccount();

  const [recipient, setRecipient] = useState('');
  const [amount, setAmount] = useState('');
  const [selectedToken, setSelectedToken] = useState<TokenOption | null>(null);
  const [availableTokens, setAvailableTokens] = useState<TokenOption[]>([]);
  const [isSending, setIsSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Initialize available tokens based on current network
  useEffect(() => {
    const initializeTokens = async () => {
      const networkConfig = {
        84532: { name: 'Base Sepolia', nativeSymbol: 'ETH' },
        11155111: { name: 'Ethereum Sepolia', nativeSymbol: 'ETH' },
      }[currentChainId] || { name: `Chain ${currentChainId}`, nativeSymbol: 'ETH' };

      const tokens: TokenOption[] = [
        {
          symbol: networkConfig.nativeSymbol,
          name: `${networkConfig.nativeSymbol} (Native)`,
          address: null,
          decimals: 18,
          isNative: true,
        },
      ];

      // Add USDC if available on this network
      const usdcAddress = await getUSDCAddress(currentChainId);
      if (usdcAddress) {
        tokens.push({
          symbol: 'USDC',
          name: 'USD Coin',
          address: usdcAddress,
          decimals: 6,
          isNative: false,
        });
      }

      setAvailableTokens(tokens);
      setSelectedToken(tokens[0]); // Default to native token
    };

    if (smartAccountReady) {
      initializeTokens();
    }
  }, [smartAccountReady, currentChainId]);

  const validateInputs = (): string | null => {
    if (!recipient.trim()) {
      return 'Recipient address is required';
    }

    if (!isAddress(recipient)) {
      return 'Invalid recipient address';
    }

    if (!amount.trim()) {
      return 'Amount is required';
    }

    const numAmount = parseFloat(amount);
    if (isNaN(numAmount) || numAmount <= 0) {
      return 'Amount must be a positive number';
    }

    if (!selectedToken) {
      return 'Please select a token';
    }

    return null;
  };

  const handleSendTransaction = async () => {
    if (!smartAccount || !bundlerClient || !pimlicoClient) {
      setError('Wallet not ready. Please wait for initialization.');
      return;
    }

    const validationError = validateInputs();
    if (validationError) {
      setError(validationError);
      return;
    }

    try {
      setIsSending(true);
      setError(null);
      setSuccess(null);

      logger.log('💸 Starting transaction send...', {
        recipient,
        amount,
        token: selectedToken?.symbol,
        chainId: currentChainId,
      });

      let calls: any[] = [];

      if (selectedToken!.isNative) {
        // Native token transfer (ETH)
        const value = parseEther(amount);
        calls = [{
          to: recipient as Address,
          value,
          data: '0x' as `0x${string}`,
        }];
      } else {
        // ERC-20 token transfer
        const tokenAddress = selectedToken!.address!;
        const decimals = selectedToken!.decimals;
        const value = BigInt(parseFloat(amount) * (10 ** decimals));

        // ERC-20 transfer function signature
        const transferData = `0xa9059cbb${recipient.slice(2).padStart(64, '0')}${value.toString(16).padStart(64, '0')}`;
        
        calls = [{
          to: tokenAddress,
          value: BigInt(0),
          data: transferData as `0x${string}`,
        }];
      }

      // Fetch gas prices
      logger.log('⛽ Fetching gas prices...');
      const gasInfo = await pimlicoClient.getUserOperationGasPrice();
      const gasPrices = {
        maxFeePerGas: BigInt(gasInfo.fast.maxFeePerGas),
        maxPriorityFeePerGas: BigInt(gasInfo.fast.maxPriorityFeePerGas),
      };

      logger.log('📤 Sending UserOperation...');
      const userOpHash = await bundlerClient.sendUserOperation({
        account: smartAccount,
        calls,
        maxFeePerGas: gasPrices.maxFeePerGas,
        maxPriorityFeePerGas: gasPrices.maxPriorityFeePerGas,
      });

      logger.log('📊 UserOperation sent:', userOpHash);
      logger.log('⏳ Waiting for confirmation...');

      const receipt = await bundlerClient.waitForUserOperationReceipt({
        hash: userOpHash,
        timeout: 60000,
      });

      if (!receipt.success) {
        throw new Error('Transaction failed');
      }

      const txHash = receipt.receipt.transactionHash;
      setSuccess(`Transaction sent successfully! Hash: ${txHash}`);
      
      // Reset form
      setRecipient('');
      setAmount('');
      
      // Notify parent component
      if (onTransactionSent) {
        onTransactionSent(txHash);
      }

      logger.log('✅ Transaction successful:', txHash);
    } catch (error) {
      logger.error('❌ Transaction failed:', error);
      setError(error instanceof Error ? error.message : 'Transaction failed');
    } finally {
      setIsSending(false);
    }
  };

  const getExplorerUrl = (hash: string): string => {
    switch (currentChainId) {
      case 84532:
        return `https://sepolia.basescan.org/tx/${hash}`;
      case 11155111:
        return `https://sepolia.etherscan.io/tx/${hash}`;
      default:
        return `https://etherscan.io/tx/${hash}`;
    }
  };

  if (!smartAccountReady) {
    return (
      <div className={`bg-gray-50 border border-gray-200 rounded-lg p-4 ${className}`}>
        <div className="text-center text-gray-500">
          <div className="animate-pulse">Initializing wallet...</div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white border border-gray-200 rounded-lg shadow-sm ${className}`}>
      <div className="p-4 border-b border-gray-100">
        <h3 className="text-lg font-semibold text-gray-900">Send Transaction</h3>
      </div>

      <div className="p-4 space-y-4">
        {/* Token Selection */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Token
          </label>
          <select
            value={selectedToken?.symbol || ''}
            onChange={(e) => {
              const token = availableTokens.find(t => t.symbol === e.target.value);
              setSelectedToken(token || null);
            }}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            disabled={isSending}
          >
            {availableTokens.map((token) => (
              <option key={token.symbol} value={token.symbol}>
                {token.name} ({token.symbol})
              </option>
            ))}
          </select>
        </div>

        {/* Recipient Address */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Recipient Address
          </label>
          <input
            type="text"
            value={recipient}
            onChange={(e) => setRecipient(e.target.value)}
            placeholder="0x..."
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            disabled={isSending}
          />
          {recipient && !isAddress(recipient) && (
            <p className="mt-1 text-sm text-red-600">Invalid address format</p>
          )}
        </div>

        {/* Amount */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Amount
          </label>
          <div className="relative">
            <input
              type="number"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="0.0"
              step="any"
              min="0"
              className="w-full px-3 py-2 pr-16 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={isSending}
            />
            <div className="absolute inset-y-0 right-0 flex items-center pr-3">
              <span className="text-sm text-gray-500">
                {selectedToken?.symbol}
              </span>
            </div>
          </div>
        </div>

        {/* Error Display */}
        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-red-800 text-sm">⚠️ {error}</p>
          </div>
        )}

        {/* Success Display */}
        {success && (
          <div className="p-3 bg-green-50 border border-green-200 rounded-lg">
            <p className="text-green-800 text-sm">✅ {success}</p>
            {success.includes('Hash:') && (
              <a
                href={getExplorerUrl(success.split('Hash: ')[1])}
                target="_blank"
                rel="noopener noreferrer"
                className="text-green-700 hover:text-green-900 text-sm underline mt-1 block"
              >
                View on Explorer →
              </a>
            )}
          </div>
        )}

        {/* Send Button */}
        <button
          onClick={handleSendTransaction}
          disabled={isSending || !smartAccount || !bundlerClient}
          className={`w-full py-3 px-4 rounded-lg font-medium transition-colors ${
            isSending || !smartAccount || !bundlerClient
              ? 'bg-gray-300 text-gray-500 cursor-not-allowed'
              : 'bg-blue-600 text-white hover:bg-blue-700'
          }`}
        >
          {isSending ? 'Sending Transaction...' : 'Send Transaction'}
        </button>

        {/* Info Section */}
        <div className="p-3 bg-blue-50 rounded-lg">
          <h4 className="font-medium text-blue-900 mb-1">ℹ️ Transaction Info:</h4>
          <ul className="text-sm text-blue-800 space-y-1">
            <li>• Transactions are sponsored by Pimlico paymaster</li>
            <li>• You don't need ETH for gas fees</li>
            <li>• Transactions are processed on {
              currentChainId === 84532 ? 'Base Sepolia' : 
              currentChainId === 11155111 ? 'Ethereum Sepolia' : 
              `Chain ${currentChainId}`
            }</li>
          </ul>
        </div>
      </div>
    </div>
  );
};