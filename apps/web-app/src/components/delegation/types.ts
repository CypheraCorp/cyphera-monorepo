import { type Address } from 'viem';
import { type MetaMaskSmartAccount } from '@metamask/delegation-toolkit';
import type { Web3Provider } from '@/lib/web3/utils/delegation';

/**
 * Smart account provider types and interfaces for delegation
 */

export type SmartAccountProviderType = 'wagmi' | 'web3auth' | 'privy';

export type DelegationStatus = 
  | 'idle' 
  | 'connecting' 
  | 'creating' 
  | 'checking' 
  | 'deploying' 
  | 'switching-network' 
  | 'signing' 
  | 'subscribing';

export interface SmartAccountState {
  isConnected: boolean;
  isAuthenticated: boolean;
  smartAccountAddress: Address | null;
  smartAccount: MetaMaskSmartAccount | null;
  isSmartAccountReady: boolean;
  isDeployed: boolean | null;
  deploymentSupported: boolean;
  isWalletCompatible?: boolean;
  provider?: Web3Provider;
}

export interface SmartAccountActions {
  connect: () => Promise<void>;
  createSmartAccount: () => Promise<void>;
  checkDeploymentStatus: () => Promise<boolean>;
  deploySmartAccount: () => Promise<void>;
  switchNetwork?: (networkName: string) => Promise<void>;
}

export interface SmartAccountProvider extends SmartAccountState, SmartAccountActions {
  type: SmartAccountProviderType;
  getDisplayName: () => string;
  getButtonText: () => string;
  isButtonDisabled: () => boolean;
}

export interface SubscriptionParams {
  priceId: string; // Product ID (named priceId for prop compatibility)
  productTokenId?: string;
  tokenAmount?: bigint | null;
  productName?: string;
  productDescription?: string;
  networkName?: string;
  priceDisplay?: string;
  intervalType?: string;
  termLength?: number;
  tokenDecimals?: number;
}

export interface SubscriptionInfo {
  id: string;
  productName?: string;
  customerName?: string;
  tokenSymbol?: string;
  tokenAmount: string;
  totalAmountCents?: number;
  walletAddress?: string;
  subscriptionStatus?: string;
  currentPeriodEnd?: number;
  nextRedemptionDate?: string;
  networkId?: string;
  transactionHash?: string;
}

export interface DelegationResult {
  delegation: string; // formatted delegation
  subscription?: SubscriptionInfo;
  transactionHash?: string;
}