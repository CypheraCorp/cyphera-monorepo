import { SmartAccountProvider, SmartAccountProviderType } from '../types';
import { MetaMaskSmartAccount } from '@metamask/delegation-toolkit';
import type { Address } from 'viem';
import type { Web3Provider } from '@/lib/web3/utils/delegation';

/**
 * Privy Smart Account Provider implementation
 * Uses Privy's embedded wallet to create and manage smart accounts
 */
export class PrivySmartAccountProvider implements SmartAccountProvider {
  private privySmartAccountHook: any;
  
  // Required SmartAccountProvider properties
  readonly type: SmartAccountProviderType = 'privy';
  
  constructor(privySmartAccountHook: any) {
    this.privySmartAccountHook = privySmartAccountHook;
  }

  // SmartAccountState properties
  get isConnected(): boolean {
    // For Privy, connected means authenticated and smart account is ready
    return this.privySmartAccountHook.isAuthenticated && this.privySmartAccountHook.smartAccountReady;
  }

  get isAuthenticated(): boolean {
    return this.privySmartAccountHook.isAuthenticated;
  }

  get smartAccountAddress(): Address | null {
    return this.privySmartAccountHook.smartAccountAddress || null;
  }

  get smartAccount(): MetaMaskSmartAccount | null {
    // Cast the Privy smart account to MetaMask format for delegation
    if (!this.privySmartAccountHook.smartAccount) return null;
    return this.privySmartAccountHook.smartAccount as unknown as MetaMaskSmartAccount;
  }

  get isSmartAccountReady(): boolean {
    return this.privySmartAccountHook.smartAccountReady;
  }

  get isDeployed(): boolean | null {
    return this.privySmartAccountHook.isDeployed;
  }

  get deploymentSupported(): boolean {
    // Privy supports deployment if bundler client is available
    return Boolean(this.privySmartAccountHook.bundlerClient);
  }

  get isWalletCompatible(): boolean {
    // Privy embedded wallets are always compatible
    return true;
  }

  get provider(): Web3Provider | undefined {
    // Privy doesn't expose a direct Web3Provider, return undefined
    return undefined;
  }

  // SmartAccountActions methods
  async connect(): Promise<void> {
    // For Privy, connect is handled through authentication
    // The smart account is created automatically on auth
    if (!this.privySmartAccountHook.isAuthenticated) {
      throw new Error('Please authenticate with Privy first');
    }
    // Wait for smart account to be ready
    while (!this.privySmartAccountHook.smartAccountReady) {
      await new Promise(resolve => setTimeout(resolve, 100));
    }
  }

  async createSmartAccount(): Promise<void> {
    // Smart account is created automatically with Privy
    // Just wait for it to be ready
    await this.connect();
  }

  async checkDeploymentStatus(): Promise<boolean> {
    return this.privySmartAccountHook.checkDeploymentStatus();
  }

  async deploySmartAccount(): Promise<void> {
    return this.privySmartAccountHook.deploySmartAccount();
  }

  async switchNetwork(networkName: string): Promise<void> {
    // Convert network name to chain ID
    const chainIdMap: Record<string, number> = {
      'base-sepolia': 84532,
      'base': 8453,
      'polygon': 137,
      'arbitrum': 42161,
      'optimism': 10,
      'ethereum': 1,
    };
    
    const chainId = chainIdMap[networkName.toLowerCase()];
    if (!chainId) {
      throw new Error(`Unknown network: ${networkName}`);
    }
    
    return this.privySmartAccountHook.switchNetwork(chainId);
  }

  // Provider-specific methods
  getDisplayName(): string {
    return 'Privy';
  }

  getButtonText(): string {
    return this.privySmartAccountHook.getButtonText();
  }

  isButtonDisabled(): boolean {
    return this.privySmartAccountHook.isButtonDisabled();
  }
}