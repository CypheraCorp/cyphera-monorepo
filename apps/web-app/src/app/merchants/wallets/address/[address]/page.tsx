import { ChevronLeft } from 'lucide-react';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { CypheraAPIClient } from '@/services/cyphera-api';
import { cookies } from 'next/headers';
import { WalletResponse } from '@/types/wallet';
import { NetworkWithTokensResponse } from '@/types/network';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { WalletDetailActions } from '@/components/wallets/wallet-detail-actions';
import { WalletDetailsContent } from '@/components/wallets/wallet-details-content';
import { requireAuth } from '@/lib/auth/guards/require-auth';
import { NetworkSelector } from '@/components/wallets/network-selector';

interface WalletsByAddressPageProps {
  params: Promise<{
    address: string;
  }>;
  searchParams: Promise<{ network?: string }>;
}

export const dynamic = 'force-dynamic'; // Opt out of static rendering

/**
 * Type for wallets by address API response
 */
interface WalletsByAddressResponse {
  address: string;
  wallets: Array<{
    wallet: WalletResponse;
    network: {
      id: string;
      name: string;
      chain_id: number;
      circle_network_type?: string;
      is_testnet: boolean;
      block_explorer_url?: string;
    } | null;
  }>;
  count: number;
}

/**
 * Server-side function to fetch wallets by address and network data
 */
async function getPageData(
  address: string,
  selectedNetworkId?: string
): Promise<{ 
  walletsData: WalletsByAddressResponse; 
  networks: NetworkWithTokensResponse[];
  selectedWallet: WalletResponse | null;
}> {
  await requireAuth();

  // Get session from cookies
  const cookieStore = await cookies();
  const sessionCookie = cookieStore.get('cyphera-session');

  if (!sessionCookie) {
    throw new Error('No session cookie found');
  }

  // Define session data type
  interface SessionData {
    access_token: string;
    account_id?: string;
    user_id?: string;
    workspace_id?: string;
    expires_at?: number;
  }

  // Decode session data from cookie
  let sessionData: SessionData;
  try {
    const decodedSession = Buffer.from(sessionCookie.value, 'base64').toString('utf-8');
    sessionData = JSON.parse(decodedSession) as SessionData;

    // Check if session is expired
    if (sessionData.expires_at && sessionData.expires_at < Date.now() / 1000) {
      throw new Error('Session expired');
    }
  } catch (_error) {
    throw new Error('Invalid session format');
  }

  // Create user context from session data
  const userContext = {
    access_token: sessionData.access_token,
    account_id: sessionData.account_id,
    user_id: sessionData.user_id,
    workspace_id: sessionData.workspace_id,
  };

  // Create API client instance
  const api = new CypheraAPIClient();
  
  // Fetch wallets by address using the API client
  const walletsData = await api.wallets.getWalletsByAddress(userContext, address);
  
  // Fetch networks data
  const networks = await api.networks.getNetworksWithTokens({ active: true });
  
  // Find the selected wallet based on network ID
  let selectedWallet: WalletResponse | null = null;
  if (selectedNetworkId && walletsData.wallets.length > 0) {
    const walletWithNetwork = walletsData.wallets.find(
      w => w.wallet.network_id === selectedNetworkId
    );
    selectedWallet = walletWithNetwork?.wallet || walletsData.wallets[0].wallet;
  } else if (walletsData.wallets.length > 0) {
    // Default to first wallet if no network selected
    selectedWallet = walletsData.wallets[0].wallet;
  }
  
  return { walletsData, networks, selectedWallet };
}

/**
 * Helper function to get status badge for Circle wallets
 */
function getStatusBadge(state: string) {
  switch (state.toLowerCase()) {
    case 'active':
      return (
        <Badge className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-100">
          Active
        </Badge>
      );
    case 'pending':
      return (
        <Badge className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-100">
          Pending
        </Badge>
      );
    default:
      return <Badge variant="outline">{state}</Badge>;
  }
}

/**
 * Get wallet type display name
 */
function getWalletTypeDisplay(wallet: WalletResponse): string {
  if (wallet.wallet_type === 'circle_wallet' || wallet.wallet_type === 'circle' || wallet.circle_data) {
    return 'Circle Wallet';
  } else if (wallet.wallet_type === 'web3auth') {
    return 'Web3Auth Wallet';
  } else if (wallet.wallet_type === 'wallet' || wallet.wallet_type === 'metamask') {
    return 'External Wallet';
  }
  return 'Unknown Wallet';
}

export default async function WalletsByAddressPage({ params, searchParams }: WalletsByAddressPageProps) {
  // Await params as required by Next.js 15
  const { address } = await params;
  const { network } = await searchParams;
  
  const decodedAddress = decodeURIComponent(address);
  const { walletsData, networks, selectedWallet } = await getPageData(decodedAddress, network);

  if (!selectedWallet) {
    return (
      <div className="container mx-auto py-6 space-y-8">
        <div className="flex items-center gap-2">
          <Link href="/merchants/wallets">
            <Button variant="ghost" size="icon">
              <ChevronLeft className="h-5 w-5" />
            </Button>
          </Link>
          <h1 className="text-2xl font-bold">No Wallets Found</h1>
        </div>
        <p>No wallets found for address {decodedAddress}</p>
      </div>
    );
  }

  // Check if this is a Circle wallet
  const isCircleWallet = selectedWallet.wallet_type === 'circle_wallet' || 
                        selectedWallet.wallet_type === 'circle' || 
                        !!selectedWallet.circle_data;

  // Get available networks for this address
  const availableNetworks = walletsData.wallets
    .filter(w => w.wallet.network_id) // Only include wallets with network_id
    .map(w => ({
      id: w.wallet.network_id as string, // We know it's defined because of the filter
      name: w.network?.name || 'Unknown Network',
      chainId: w.network?.chain_id || 0,
      isTestnet: w.network?.is_testnet || false,
    }));

  return (
    <div className="container mx-auto py-6 space-y-8">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Link href="/merchants/wallets">
            <Button variant="ghost" size="icon">
              <ChevronLeft className="h-5 w-5" />
            </Button>
          </Link>
          <h1 className="text-2xl font-bold">Wallet Details</h1>
          {availableNetworks.length > 1 && (
            <NetworkSelector 
              networks={availableNetworks}
              selectedNetworkId={selectedWallet.network_id || ''}
              walletAddress={decodedAddress}
            />
          )}
        </div>
        <WalletDetailActions wallet={selectedWallet} />
      </div>

      <div className={`grid grid-cols-1 ${isCircleWallet ? 'md:grid-cols-2' : ''} gap-6`}>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-xl flex items-center gap-2">
              {selectedWallet.nickname || 'Unnamed Wallet'}
              {selectedWallet.circle_data &&
                selectedWallet.circle_data.state &&
                getStatusBadge(selectedWallet.circle_data.state)}
            </CardTitle>
            {selectedWallet.is_primary && <Badge variant="secondary">Primary</Badge>}
          </CardHeader>
          <CardContent>
            <div className="flex flex-col space-y-2">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Address:</span>
                <div className="flex items-center gap-1">
                  <span className="font-mono text-sm truncate max-w-[200px]">
                    {selectedWallet.wallet_address}
                  </span>
                </div>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Type:</span>
                <span>{getWalletTypeDisplay(selectedWallet)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Network:</span>
                <span>
                  {walletsData.wallets.find(w => w.wallet.id === selectedWallet.id)?.network?.name || 'Unknown'}
                </span>
              </div>
              {selectedWallet.circle_data && selectedWallet.circle_data.chain_id && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Chain ID:</span>
                  <span>{selectedWallet.circle_data.chain_id}</span>
                </div>
              )}
              {selectedWallet.created_at && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Created:</span>
                  <span>{new Date(selectedWallet.created_at).toLocaleDateString()}</span>
                </div>
              )}
              {selectedWallet.last_used_at && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Last Used:</span>
                  <span>{new Date(selectedWallet.last_used_at * 1000).toLocaleDateString()}</span>
                </div>
              )}
              {selectedWallet.wallet_type === 'web3auth' && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Status:</span>
                  <Badge className="bg-purple-100 text-purple-700">Managed by Web3Auth</Badge>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Client-side content for Circle wallets */}
        <WalletDetailsContent wallet={selectedWallet} networks={networks} />
      </div>
    </div>
  );
}