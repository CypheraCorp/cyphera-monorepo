'use client';

import React, { useState, useEffect } from 'react';
import { usePrivySmartAccount } from '@/hooks/privy/use-privy-smart-account';
import { createPublicClient, http, formatEther, type Address, type Hash } from 'viem';
import { getNetworkConfig } from '@/lib/web3/dynamic-networks';
import { logger } from '@/lib/core/logger/logger-utils';

interface Transaction {
  hash: Hash;
  from: Address;
  to: Address | null;
  value: bigint;
  formattedValue: string;
  timestamp: Date;
  blockNumber: bigint;
  status: 'success' | 'failed' | 'pending';
  type: 'sent' | 'received' | 'contract' | 'handle_ops' | 'token_transfer';
  method?: string; // The method being called (e.g., 'Handle Ops', 'Transfer', 'Approve')
  tokenSymbol?: string; // For token transfers (e.g., 'USDC', 'ETH')
  tokenAmount?: string; // Formatted token amount
  gasUsed?: bigint;
  gasPrice?: bigint;
  logIndex?: number; // For tracking specific log events
  embeddedTransfers?: Array<{
    tokenAddress: Address;
    tokenSymbol: string;
    amount: string;
    formattedAmount: string;
    decimals: number;
    direction: 'sent' | 'received'; // Whether smart account sent or received this token
  }>; // For Handle Ops that contain token transfers
}

interface TransactionHistoryProps {
  className?: string;
  limit?: number;
}

// Helper function to fetch ETH transactions
async function fetchETHTransactions(
  publicClient: any,
  smartAccountAddress: Address,
  fromBlock: bigint,
  toBlock: bigint,
  limit: number
): Promise<Transaction[]> {
  const transactions: Transaction[] = [];
  
  try {
    // Check recent blocks for ETH transactions
    const blocksToCheck = Math.min(20, Number(toBlock - fromBlock)); // Limit to avoid RPC limits
    const blockPromises = [];
    
    for (let i = 0; i < blocksToCheck && transactions.length < limit; i++) {
      const blockNumber = toBlock - BigInt(i);
      blockPromises.push(
        publicClient.getBlock({
          blockNumber,
          includeTransactions: true,
        })
      );
    }

    const blocks = await Promise.all(blockPromises);
    
    for (const block of blocks) {
      if (transactions.length >= limit) break;
      
      const relevantTxs = block.transactions.filter((tx: any) => 
        (tx.to?.toLowerCase() === smartAccountAddress.toLowerCase() ||
         tx.from.toLowerCase() === smartAccountAddress.toLowerCase()) &&
        tx.value > 0n // Only include transactions with ETH value
      );

      for (const tx of relevantTxs) {
        if (transactions.length >= limit) break;
        
        try {
          const receipt = await publicClient.getTransactionReceipt({ hash: tx.hash });
          const isSent = tx.from.toLowerCase() === smartAccountAddress.toLowerCase();
          
          transactions.push({
            hash: tx.hash,
            from: tx.from,
            to: tx.to,
            value: tx.value,
            formattedValue: formatEther(tx.value),
            timestamp: new Date(Number(block.timestamp) * 1000),
            blockNumber: receipt.blockNumber,
            status: receipt.status === 'success' ? 'success' : 'failed',
            type: isSent ? 'sent' : 'received',
            method: 'Transfer',
            tokenSymbol: 'ETH',
            tokenAmount: formatEther(tx.value),
            gasUsed: receipt.gasUsed,
            gasPrice: tx.gasPrice,
          });
        } catch (error) {
          logger.warn('Failed to process ETH transaction:', tx.hash, error);
        }
      }
    }
  } catch (error) {
    logger.error('Failed to fetch ETH transactions:', error);
  }
  
  return transactions;
}

// Helper function for rate limiting protection
const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

// Rate limiting configuration
const RATE_LIMIT_CONFIG = {
  BATCH_SIZE: 100,
  DELAY_MS: 200,
  MAX_RETRIES: 3,
  INITIAL_BACKOFF: 1000,
  MAX_BACKOFF: 5000,
  MAX_CONCURRENT: 3
};

// Helper function to detect rate limit errors
function isRateLimitError(error: any): boolean {
  const message = error?.message?.toLowerCase() || '';
  return message.includes('rate limit') || 
         message.includes('too many requests') || 
         message.includes('exceeded') ||
         error?.status === 429;
}

// Helper function with exponential backoff retry
async function withRetry<T>(fn: () => Promise<T>, maxRetries = RATE_LIMIT_CONFIG.MAX_RETRIES): Promise<T> {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (isRateLimitError(error) && i < maxRetries - 1) {
        const delay = Math.min(
          RATE_LIMIT_CONFIG.INITIAL_BACKOFF * Math.pow(2, i), 
          RATE_LIMIT_CONFIG.MAX_BACKOFF
        );
        logger.warn(`Rate limit hit, retrying in ${delay}ms...`);
        await sleep(delay);
        continue;
      }
      throw error;
    }
  }
  throw new Error('Max retries exceeded');
}

// Helper function to fetch smart account transactions directly
async function fetchSmartAccountTransactions(
  publicClient: any,
  smartAccountAddress: Address,
  fromBlock: bigint,
  toBlock: bigint,
  limit: number,
  tokens: any[] = []
): Promise<Transaction[]> {
  const transactions: Transaction[] = [];
  
  try {
    // Process in smaller batches to avoid rate limits
    const batchSize = BigInt(RATE_LIMIT_CONFIG.BATCH_SIZE);
    let currentFromBlock = fromBlock;
    
    while (currentFromBlock <= toBlock && transactions.length < limit) {
      const currentToBlock = currentFromBlock + batchSize > toBlock ? toBlock : currentFromBlock + batchSize;
      
      logger.log(`🔍 [Smart Account] Fetching transactions from blocks ${currentFromBlock} to ${currentToBlock}`);
      
      await withRetry(async () => {
        // Get all logs where smart account is involved (any event type)
        const logs = await publicClient.getLogs({
          fromBlock: currentFromBlock,
          toBlock: currentToBlock,
          topics: [
            null, // any event signature
          ]
        });
        
        // Filter logs that involve our smart account
        const smartAccountLogs = logs.filter((log: any) => {
          const logTopics = log.topics || [];
          const smartAccountHex = `0x000000000000000000000000${smartAccountAddress.slice(2).toLowerCase()}`;
          
          // Check if smart account appears in any topic (from/to positions)
          return logTopics.some((topic: string) => 
            topic && topic.toLowerCase() === smartAccountHex.toLowerCase()
          );
        });
        
        logger.log(`📊 [Smart Account] Found ${smartAccountLogs.length} logs involving smart account`);
        
        // Process each relevant log
        for (const log of smartAccountLogs.slice(0, limit - transactions.length)) {
          try {
            const [block, receipt] = await Promise.all([
              publicClient.getBlock({ blockNumber: log.blockNumber }),
              publicClient.getTransactionReceipt({ hash: log.transactionHash })
            ]);
            
            // Determine transaction type based on the log
            let transactionType = 'contract';
            let method = 'Contract Interaction';
            let tokenSymbol = 'ETH';
            let tokenAmount = '0';
            
            // Check if this is a token transfer
            const transferEventSignature = '0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef';
            if (log.topics[0] === transferEventSignature) {
              transactionType = 'token_transfer';
              method = 'Transfer';
              
              // Find the token info
              const token = tokens.find(t => t.address.toLowerCase() === log.address.toLowerCase());
              if (token) {
                tokenSymbol = token.symbol;
                try {
                  const value = BigInt(log.data);
                  const divisor = BigInt(10 ** token.decimals);
                  tokenAmount = (Number(value) / Number(divisor)).toFixed(token.decimals === 18 ? 4 : 2);
                } catch (e) {
                  logger.warn('Failed to decode token amount:', e);
                }
              }
            } else {
              // This might be a Handle Ops or other smart contract interaction
              transactionType = 'handle_ops';
              method = 'Handle Ops';
              
              // Check if there are token transfers in this transaction
              const allTransferLogs = receipt.logs.filter((receiptLog: any) => 
                receiptLog.topics[0] === transferEventSignature
              );
              
              for (const transferLog of allTransferLogs) {
                const fromAddress = `0x${transferLog.topics[1]?.slice(26)}`.toLowerCase();
                const toAddress = `0x${transferLog.topics[2]?.slice(26)}`.toLowerCase();
                const smartAccountLower = smartAccountAddress.toLowerCase();
                
                if (fromAddress === smartAccountLower || toAddress === smartAccountLower) {
                  const token = tokens.find(t => t.address.toLowerCase() === transferLog.address.toLowerCase());
                  if (token) {
                    tokenSymbol = token.symbol;
                    try {
                      const value = BigInt(transferLog.data);
                      const divisor = BigInt(10 ** token.decimals);
                      tokenAmount = (Number(value) / Number(divisor)).toFixed(token.decimals === 18 ? 4 : 2);
                      break; // Use the first token transfer found
                    } catch (e) {
                      logger.warn('Failed to decode token amount from Handle Ops:', e);
                    }
                  }
                }
              }
            }
            
            // Avoid duplicates
            const existingTx = transactions.find(tx => tx.hash === log.transactionHash);
            if (existingTx) continue;
            
            transactions.push({
              hash: log.transactionHash,
              from: smartAccountAddress,
              to: log.address as Address,
              value: 0n,
              formattedValue: '0',
              timestamp: new Date(Number(block.timestamp) * 1000),
              blockNumber: log.blockNumber,
              status: receipt.status === 'success' ? 'success' : 'failed',
              type: transactionType as any,
              method,
              tokenSymbol,
              tokenAmount,
              gasUsed: receipt.gasUsed,
              logIndex: log.logIndex,
            });
            
          } catch (error) {
            logger.warn('Failed to process smart account transaction:', log.transactionHash, error);
          }
        }
      });
      
      currentFromBlock = currentToBlock + 1n;
      
      // Rate limiting delay between batches
      if (currentFromBlock <= toBlock) {
        await sleep(RATE_LIMIT_CONFIG.DELAY_MS);
      }
    }
    
    logger.log(`✅ [Smart Account] Fetched ${transactions.length} transactions`);
    
  } catch (error) {
    logger.error('Failed to fetch smart account transactions:', error);
  }
  
  return transactions;
}

// Simplified helper function to fetch ERC-20 token transfers with rate limiting
async function fetchTokenTransfers(
  publicClient: any,
  smartAccountAddress: Address,
  tokens: any[],
  fromBlock: bigint,
  toBlock: bigint,
  limit: number
): Promise<Transaction[]> {
  const transactions: Transaction[] = [];
  
  try {
    // ERC-20 Transfer event signature: Transfer(address indexed from, address indexed to, uint256 value)
    const transferEventSignature = '0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef';
    const smartAccountHex = `0x000000000000000000000000${smartAccountAddress.slice(2).toLowerCase()}`;
    
    logger.log(`🔍 [Token Transfers] Fetching for ${tokens.length} tokens from blocks ${fromBlock} to ${toBlock}`);
    
    for (const token of tokens) {
      if (transactions.length >= limit) break;
      
      await withRetry(async () => {
        // Get transfers where smart account is sender OR receiver in a single query
        const logs = await publicClient.getLogs({
          address: token.address,
          fromBlock,
          toBlock,
          topics: [
            transferEventSignature,
            [smartAccountHex, null], // from (smart account OR any)
            [null, smartAccountHex], // to (any OR smart account)
          ],
        });
        
        logger.log(`📊 [Token Transfers] Found ${logs.length} ${token.symbol} transfers`);
        
        // Process each transfer log
        for (const log of logs.slice(0, limit - transactions.length)) {
          try {
            // Check if this transfer actually involves our smart account
            const fromAddress = `0x${log.topics[1]?.slice(26)}`.toLowerCase();
            const toAddress = `0x${log.topics[2]?.slice(26)}`.toLowerCase();
            const smartAccountLower = smartAccountAddress.toLowerCase();
            
            if (fromAddress !== smartAccountLower && toAddress !== smartAccountLower) {
              continue; // Skip transfers not involving our smart account
            }
            
            const [block, receipt] = await Promise.all([
              publicClient.getBlock({ blockNumber: log.blockNumber }),
              publicClient.getTransactionReceipt({ hash: log.transactionHash })
            ]);
            
            // Decode the transfer amount
            const value = BigInt(log.data);
            const divisor = BigInt(10 ** token.decimals);
            const formattedAmount = (Number(value) / Number(divisor)).toFixed(token.decimals === 18 ? 4 : 2);
            
            transactions.push({
              hash: log.transactionHash,
              from: fromAddress as Address,
              to: toAddress as Address,
              value: value,
              formattedValue: formattedAmount,
              timestamp: new Date(Number(block.timestamp) * 1000),
              blockNumber: log.blockNumber,
              status: receipt.status === 'success' ? 'success' : 'failed',
              type: 'token_transfer',
              method: 'Transfer',
              tokenSymbol: token.symbol,
              tokenAmount: formattedAmount,
              gasUsed: receipt.gasUsed,
              logIndex: log.logIndex,
            });
          } catch (error) {
            logger.warn('Failed to process token transfer:', log.transactionHash, error);
          }
        }
      });
      
      // Add delay between token queries to avoid rate limits
      if (tokens.indexOf(token) < tokens.length - 1) {
        await sleep(RATE_LIMIT_CONFIG.DELAY_MS);
      }
    }
    
    logger.log(`✅ [Token Transfers] Fetched ${transactions.length} token transfers`);
    
  } catch (error) {
    logger.error('Failed to fetch token transfers:', error);
  }
  
  return transactions;
}


export const TransactionHistory: React.FC<TransactionHistoryProps> = ({ 
  className = '', 
  limit = 10 
}) => {
  const { smartAccountAddress, smartAccountReady, currentChainId } = usePrivySmartAccount();
  
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedToken, setSelectedToken] = useState<string>('all'); // 'all', 'eth', 'usdc', etc.
  const [availableTokens, setAvailableTokens] = useState<Array<{symbol: string, address: Address}>>([]);

  const fetchTransactions = async () => {
    if (!smartAccountAddress || !smartAccountReady) return;

    try {
      setIsLoading(true);
      setError(null);

      let networkConfig = await getNetworkConfig(currentChainId);
      
      // Fallback to hardcoded configurations if dynamic fetch fails (same as smart account hook)
      if (!networkConfig && currentChainId === 84532) {
        logger.log('⚠️ Using fallback Base Sepolia configuration for transaction history');
        const infuraApiKey = process.env.NEXT_PUBLIC_INFURA_API_KEY;
        networkConfig = {
          chain: {
            id: 84532,
            name: 'Base Sepolia',
            nativeCurrency: { name: 'Ethereum', symbol: 'ETH', decimals: 18 },
            rpcUrls: {
              default: {
                http: [infuraApiKey 
                  ? `https://base-sepolia.infura.io/v3/${infuraApiKey}`
                  : 'https://sepolia.base.org'
                ]
              }
            }
          } as any,
          rpcUrl: infuraApiKey 
            ? `https://base-sepolia.infura.io/v3/${infuraApiKey}`
            : 'https://sepolia.base.org',
          circleNetworkType: 'BASE-SEPOLIA',
          isPimlicoSupported: true,
          isCircleSupported: true,
          tokens: [{
            address: '0x036CbD53842c5426634e7929541eC2318f3dCF7e' as Address,
            symbol: 'USDC',
            name: 'USD Coin',
            decimals: 6,
            isGasToken: false,
          }],
        };
      } else if (!networkConfig && currentChainId === 11155111) {
        logger.log('⚠️ Using fallback Ethereum Sepolia configuration for transaction history');
        const infuraApiKey = process.env.NEXT_PUBLIC_INFURA_API_KEY;
        networkConfig = {
          chain: {
            id: 11155111,
            name: 'Ethereum Sepolia',
            nativeCurrency: { name: 'Ethereum', symbol: 'ETH', decimals: 18 },
            rpcUrls: {
              default: {
                http: [infuraApiKey 
                  ? `https://sepolia.infura.io/v3/${infuraApiKey}`
                  : 'https://rpc.sepolia.org'
                ]
              }
            }
          } as any,
          rpcUrl: infuraApiKey 
            ? `https://sepolia.infura.io/v3/${infuraApiKey}`
            : 'https://rpc.sepolia.org',
          circleNetworkType: 'ETH-SEPOLIA',
          isPimlicoSupported: true,
          isCircleSupported: true,
          tokens: [{
            address: '0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238' as Address,
            symbol: 'USDC',
            name: 'USD Coin',
            decimals: 6,
            isGasToken: false,
          }],
        };
      }
      
      if (!networkConfig) {
        throw new Error(`Network configuration not found for chain ${currentChainId}`);
      }
      
      // Set available tokens for filtering (ETH + configured tokens)
      const ethToken = { symbol: 'ETH', address: '0x0000000000000000000000000000000000000000' as Address };
      const tokens = [ethToken, ...networkConfig.tokens.map(t => ({ symbol: t.symbol, address: t.address }))];
      setAvailableTokens(tokens);

      const publicClient = createPublicClient({
        chain: networkConfig.chain,
        transport: http(networkConfig.rpcUrl),
      });

      logger.log('🔍 Fetching transaction history using direct smart account approach...');

      // Get the latest block number
      const latestBlock = await publicClient.getBlockNumber();
      
      // Use a reasonable block range with rate limiting protection
      const blockRange = 500n; // Start with 500 blocks, can be adjusted based on success/failure
      const fromBlock = latestBlock > blockRange ? latestBlock - blockRange : 0n;

      logger.log(`📊 Searching blocks ${fromBlock.toString()} to ${latestBlock.toString()} for smart account transactions`);

      // Use the new direct smart account approach instead of parallel fetching
      const smartAccountTransactions = await fetchSmartAccountTransactions(
        publicClient, 
        smartAccountAddress, 
        fromBlock, 
        latestBlock, 
        limit, 
        networkConfig.tokens
      );
      
      // Also fetch dedicated token transfers for completeness
      const tokenOnlyTransfers = await fetchTokenTransfers(
        publicClient, 
        smartAccountAddress, 
        networkConfig.tokens, 
        fromBlock, 
        latestBlock, 
        limit
      );
      
      // Combine and deduplicate transactions by hash
      const allTransactions = [...smartAccountTransactions, ...tokenOnlyTransfers];
      const uniqueTransactions = allTransactions.filter((tx, index, arr) => 
        arr.findIndex(t => t.hash === tx.hash) === index
      );
      
      // Sort by block number (newest first)
      const sortedTransactions = uniqueTransactions.sort((a, b) => 
        Number(b.blockNumber) - Number(a.blockNumber)
      ).slice(0, limit);

      logger.log(`✅ Final transaction count: ${sortedTransactions.length} (${smartAccountTransactions.length} smart account + ${tokenOnlyTransfers.length} token transfers, deduplicated)`);
      
      setTransactions(sortedTransactions);
    } catch (error) {
      logger.error('❌ Failed to fetch transaction history:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch transactions');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (smartAccountReady && smartAccountAddress) {
      fetchTransactions();
    }
  }, [smartAccountReady, smartAccountAddress, currentChainId]);

  // Filter transactions based on selected token
  const getFilteredTransactions = (): Transaction[] => {
    if (selectedToken === 'all') {
      return transactions;
    }
    
    return transactions.filter(tx => {
      const tokenSymbol = selectedToken.toLowerCase();
      
      // For ETH selection, show ETH transactions and Handle Ops that don't involve other tokens
      if (tokenSymbol === 'eth') {
        if (tx.type === 'sent' || tx.type === 'received') {
          return tx.tokenSymbol?.toLowerCase() === 'eth';
        }
        if (tx.type === 'handle_ops') {
          // Show Handle Ops if they don't have embedded transfers (pure ETH ops) or if they involve ETH
          return !tx.embeddedTransfers || tx.embeddedTransfers.length === 0 || 
                 tx.embeddedTransfers.some(et => et.tokenSymbol.toLowerCase() === 'eth');
        }
        return false;
      }
      
      // For specific token selection (like USDC)
      if (tx.type === 'token_transfer') {
        return tx.tokenSymbol?.toLowerCase() === tokenSymbol;
      }
      
      if (tx.type === 'handle_ops') {
        // Show Handle Ops if they have embedded transfers involving the selected token
        return tx.embeddedTransfers && tx.embeddedTransfers.some(et => et.tokenSymbol.toLowerCase() === tokenSymbol);
      }
      
      // For sent/received, check if it's an ETH transaction with the selected token
      if ((tx.type === 'sent' || tx.type === 'received') && tokenSymbol !== 'eth') {
        return tx.tokenSymbol?.toLowerCase() === tokenSymbol;
      }
      
      return false;
    });
  };

  const getExplorerUrl = (hash: string): string => {
    switch (currentChainId) {
      case 84532: // Base Sepolia
        return `https://sepolia.basescan.org/tx/${hash}`;
      case 11155111: // Ethereum Sepolia
        return `https://sepolia.etherscan.io/tx/${hash}`;
      default:
        return `https://etherscan.io/tx/${hash}`;
    }
  };

  const formatAddress = (address: Address): string => {
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  // Helper function to get transaction icon and color based on type
  const getTransactionTypeInfo = (tx: Transaction) => {
    switch (tx.type) {
      case 'handle_ops':
        return {
          icon: '⚙️',
          color: 'bg-purple-500',
          label: tx.method || 'Handle Ops',
          description: 'Smart Account Operation'
        };
      case 'token_transfer':
        const isSent = tx.from.toLowerCase() === smartAccountAddress?.toLowerCase();
        return {
          icon: isSent ? '↗' : '↙',
          color: isSent ? 'bg-red-500' : 'bg-green-500',
          label: tx.method || 'Transfer',
          description: `${tx.tokenSymbol} Token Transfer`
        };
      case 'sent':
      case 'received':
        return {
          icon: tx.type === 'sent' ? '↗' : '↙',
          color: tx.type === 'sent' ? 'bg-red-500' : 'bg-green-500',
          label: tx.method || 'Transfer',
          description: `${tx.tokenSymbol} Transfer`
        };
      default:
        return {
          icon: '📄',
          color: 'bg-gray-500',
          label: tx.method || 'Contract',
          description: 'Contract Interaction'
        };
    }
  };

  // Helper function to format transaction amount display
  const formatTransactionAmount = (tx: Transaction) => {
    const typeInfo = getTransactionTypeInfo(tx);
    
    if (tx.type === 'handle_ops') {
      // Use embedded transfers for Handle Ops display
      if (tx.embeddedTransfers && tx.embeddedTransfers.length > 0) {
        const primaryTransfer = tx.embeddedTransfers[0];
        const prefix = primaryTransfer.direction === 'sent' ? '-' : '+';
        const color = primaryTransfer.direction === 'sent' ? 'text-red-600' : 'text-green-600';
        
        return {
          amount: primaryTransfer.formattedAmount,
          symbol: primaryTransfer.tokenSymbol,
          prefix,
          color
        };
      }
      
      // Fallback for Handle Ops without embedded transfers
      return {
        amount: '0.00',
        symbol: 'ETH',
        prefix: '',
        color: 'text-gray-600'
      };
    }
    
    if (tx.type === 'token_transfer') {
      const isSent = tx.from.toLowerCase() === smartAccountAddress?.toLowerCase();
      return {
        amount: tx.tokenAmount || tx.formattedValue,
        symbol: tx.tokenSymbol || 'TOKEN',
        prefix: isSent ? '-' : '+',
        color: isSent ? 'text-red-600' : 'text-green-600'
      };
    }
    
    // ETH transactions
    const isSent = tx.type === 'sent';
    return {
      amount: parseFloat(tx.formattedValue).toFixed(4),
      symbol: tx.tokenSymbol || 'ETH',
      prefix: isSent ? '-' : '+',
      color: isSent ? 'text-red-600' : 'text-green-600'
    };
  };

  const formatTimestamp = (timestamp: Date): string => {
    const now = new Date();
    const diffMs = now.getTime() - timestamp.getTime();
    const diffMins = Math.floor(diffMs / (1000 * 60));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    
    return timestamp.toLocaleDateString();
  };

  if (!smartAccountReady) {
    return (
      <div className={`bg-gray-50 border border-gray-200 rounded-lg p-4 ${className}`}>
        <div className="text-center text-gray-500">
          <div className="animate-pulse">Loading transaction history...</div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white border border-gray-200 rounded-lg shadow-sm ${className}`}>
      <div className="p-4 border-b border-gray-100">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-lg font-semibold text-gray-900">Transaction History</h3>
          <button
            onClick={fetchTransactions}
            disabled={isLoading}
            className="text-sm text-blue-600 hover:text-blue-800 disabled:opacity-50"
          >
            {isLoading ? 'Loading...' : '↻ Refresh'}
          </button>
        </div>
        
        {/* Token Filter Dropdown */}
        <div className="flex items-center space-x-2">
          <label className="text-sm font-medium text-gray-700">Filter by token:</label>
          <select
            value={selectedToken}
            onChange={(e) => setSelectedToken(e.target.value)}
            className="px-3 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="all">All Tokens</option>
            {availableTokens.map(token => (
              <option key={token.symbol} value={token.symbol.toLowerCase()}>
                {token.symbol}
              </option>
            ))}
          </select>
        </div>
      </div>

      {error && (
        <div className="p-4 bg-red-50 border-b border-red-100">
          <p className="text-red-800 text-sm">⚠️ {error}</p>
        </div>
      )}

      <div className="p-4">
        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="animate-pulse">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <div className="w-8 h-8 bg-gray-200 rounded-full"></div>
                    <div>
                      <div className="w-20 h-4 bg-gray-200 rounded"></div>
                      <div className="w-32 h-3 bg-gray-200 rounded mt-1"></div>
                    </div>
                  </div>
                  <div>
                    <div className="w-16 h-4 bg-gray-200 rounded"></div>
                    <div className="w-12 h-3 bg-gray-200 rounded mt-1 ml-auto"></div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : getFilteredTransactions().length === 0 ? (
          <div className="text-center text-gray-500 py-8">
            <p>No transactions found</p>
            <p className="text-sm mt-1">Make your first transaction to see history here</p>
          </div>
        ) : (
          <div className="space-y-3">
            {getFilteredTransactions().map((tx) => (
              <div 
                key={`${tx.hash}-${tx.logIndex || 0}`} 
                className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
              >
                <div className="flex items-center space-x-3">
                  {(() => {
                    const typeInfo = getTransactionTypeInfo(tx);
                    return (
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-bold ${typeInfo.color}`}>
                        <span className="text-xs">{typeInfo.icon}</span>
                      </div>
                    );
                  })()}
                  <div>
                    <div className="flex items-center space-x-2">
                      <p className="font-medium text-gray-900">{getTransactionTypeInfo(tx).label}</p>
                      <span className={`px-2 py-1 text-xs rounded ${
                        tx.status === 'success' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                      }`}>
                        {tx.status}
                      </span>
                    </div>
                    <div className="flex items-center space-x-2 text-xs text-gray-500">
                      <span className="text-gray-600">{getTransactionTypeInfo(tx).description}</span>
                      <span>•</span>
                      <span>{formatTimestamp(tx.timestamp)}</span>
                    </div>
                    <div className="text-xs text-gray-500 mt-1">
                      {tx.type === 'handle_ops' ? (
                        tx.embeddedTransfers && tx.embeddedTransfers.length > 0 ? (
                          <span>
                            {tx.embeddedTransfers[0].direction === 'sent' ? 'Sent' : 'Received'} {tx.embeddedTransfers[0].tokenSymbol}
                            {tx.embeddedTransfers.length > 1 && ` (+ ${tx.embeddedTransfers.length - 1} more)`}
                          </span>
                        ) : (
                          <span>Smart Account Operation</span>
                        )
                      ) : tx.type === 'token_transfer' ? (
                        tx.from.toLowerCase() === smartAccountAddress?.toLowerCase() ? (
                          <span>To: {formatAddress(tx.to!)}</span>
                        ) : (
                          <span>From: {formatAddress(tx.from)}</span>
                        )
                      ) : (
                        <span>{tx.type === 'sent' ? 'To:' : 'From:'} {formatAddress(tx.type === 'sent' ? tx.to! : tx.from)}</span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  {(() => {
                    const amountInfo = formatTransactionAmount(tx);
                    return (
                      <p className={`font-mono text-sm ${amountInfo.color}`}>
                        {amountInfo.prefix}{amountInfo.amount} {amountInfo.symbol}
                      </p>
                    );
                  })()}
                  <a
                    href={getExplorerUrl(tx.hash)}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs text-blue-600 hover:text-blue-800"
                  >
                    View →
                  </a>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};