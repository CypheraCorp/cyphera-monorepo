/**
 * Network switching utilities for Web3 providers
 * Provides consistent network switching behavior across different wallet implementations
 */

export interface Web3Provider {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
}

/**
 * Switch to a specific network using a Web3 provider
 * @param provider - Web3 provider that supports wallet_switchEthereumChain
 * @param targetChainId - The chain ID to switch to
 * @throws Error if network switch fails
 */
export async function switchToNetwork(
  provider: Web3Provider,
  targetChainId: number
): Promise<void> {
  try {
    const hexChainId = `0x${targetChainId.toString(16)}`;

    // First try to switch to the network
    try {
      await provider.request({
        method: 'wallet_switchEthereumChain',
        params: [{ chainId: hexChainId }],
      });
      console.log(`✅ Successfully switched to chain ${targetChainId}`);
    } catch (switchError) {
      // If the network doesn't exist, we might need to add it
      if ((switchError as { code?: number }).code === 4902) {
        console.log(`Network ${targetChainId} not found, would need to add it`);
        throw new Error(`Network ${targetChainId} not configured in wallet`);
      } else {
        throw switchError;
      }
    }
  } catch (error) {
    console.error('Failed to switch network:', { error });
    throw error;
  }
}

/**
 * Get the current chain ID from a Web3 provider
 * @param provider - Web3 provider
 * @returns Current chain ID as number
 */
export async function getCurrentChainId(provider: Web3Provider): Promise<number> {
  try {
    const currentChainId = (await provider.request({
      method: 'eth_chainId',
    })) as string;
    return parseInt(currentChainId, 16);
  } catch (error) {
    console.error('Failed to get current chain ID:', { error });
    throw new Error('Could not determine current network');
  }
}

/**
 * Fetch chain ID from network name via API
 * @param networkName - Name of the network
 * @returns Chain ID or null if not found
 */
export async function getChainIdFromNetworkName(networkName: string): Promise<number | null> {
  try {
    const response = await fetch('/api/networks?active=true');
    if (!response.ok) return null;

    const networks = await response.json();
    const network = networks.find(
      (n: { network: { name: string; chain_id: number } }) => 
        n.network.name.toLowerCase() === networkName.toLowerCase()
    );

    return network?.network.chain_id || null;
  } catch (error) {
    console.error('Failed to get chain ID from network name:', { error });
    return null;
  }
}

/**
 * Check if provider is on the correct network and switch if needed
 * @param provider - Web3 provider
 * @param networkName - Target network name
 * @returns Promise that resolves when on correct network
 */
export async function ensureCorrectNetwork(
  provider: Web3Provider,
  networkName: string
): Promise<void> {
  const requiredChainId = await getChainIdFromNetworkName(networkName);
  if (!requiredChainId) {
    throw new Error(`Network ${networkName} not found`);
  }

  // Add a small delay to ensure provider is ready
  await new Promise(resolve => setTimeout(resolve, 500));

  const currentChainId = await getCurrentChainId(provider);

  if (currentChainId !== requiredChainId) {
    console.log(`🔄 Switching from chain ${currentChainId} to ${requiredChainId}`);
    await switchToNetwork(provider, requiredChainId);
    
    // Give a moment for the network switch to propagate
    await new Promise(resolve => setTimeout(resolve, 1500));
  } else {
    console.log('✅ Already on correct network');
  }
}