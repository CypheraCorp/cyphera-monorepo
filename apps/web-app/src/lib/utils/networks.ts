import { NetworkWithTokensResponse } from '@/types/network';

/**
 * Groups selected tokens by network for better organization
 * @param selectedTokens - Array of selected token IDs in the format "networkId:tokenId"
 * @param networks - Available network options
 * @returns Grouped and formatted selection data
 */
export function formatSelectedTokens(
  selectedTokens: string[],
  networks: NetworkWithTokensResponse[]
): { network: string; token: string; symbol: string }[] {
  return selectedTokens.map((selection) => {
    const [networkId, tokenId] = selection.split(':');
    const network = networks.find((n) => n.network.id === networkId);
    const token = network?.network.tokens?.find((t) => t.id === tokenId);

    return {
      network: network?.network.name || 'Unknown Network',
      token: token?.name || 'Unknown Token',
      symbol: token?.symbol || '???',
    };
  });
}
