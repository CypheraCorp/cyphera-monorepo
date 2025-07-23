import { ChevronLeft } from 'lucide-react';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { CypheraAPIClient } from '@/services/cyphera-api';
import { cookies } from 'next/headers';
import { WalletResponse } from '@/types/wallet';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { WalletDetailActions } from '@/components/wallets/wallet-detail-actions';
import { requireAuth } from '@/lib/auth/guards/require-auth';

interface WalletDetailsPageProps {
  params: Promise<{
    id: string;
  }>;
}

export const dynamic = 'force-dynamic'; // Opt out of static rendering

/**
 * Server-side function to fetch wallet data
 */
async function getWalletData(id: string): Promise<WalletResponse> {
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
  const wallet = await api.wallets.getWallet(userContext, id);
  return wallet;
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

export default async function WalletDetailsPage({ params }: WalletDetailsPageProps) {
  // Await params as required by Next.js 15
  const { id } = await params;
  const wallet = await getWalletData(id);

  return (
    <div className="container mx-auto py-6 space-y-8">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Link href="/wallets">
            <Button variant="ghost" size="icon">
              <ChevronLeft className="h-5 w-5" />
            </Button>
          </Link>
          <h1 className="text-2xl font-bold">Wallet Details</h1>
        </div>
        <WalletDetailActions wallet={wallet} />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-xl flex items-center gap-2">
              {wallet.nickname || 'Unnamed Wallet'}
              {wallet.circle_data &&
                wallet.circle_data.state &&
                getStatusBadge(wallet.circle_data.state)}
            </CardTitle>
            {wallet.is_primary && <Badge variant="secondary">Primary</Badge>}
          </CardHeader>
          <CardContent>
            <div className="flex flex-col space-y-2">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Address:</span>
                <div className="flex items-center gap-1">
                  <span className="font-mono text-sm truncate max-w-[200px]">
                    {wallet.wallet_address}
                  </span>
                </div>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Type:</span>
                <span>
                  {wallet.wallet_type === 'circle_wallet' || wallet.circle_data
                    ? 'Circle Wallet'
                    : 'Standard Wallet'}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Network:</span>
                <span>{wallet.network_type}</span>
              </div>
              {wallet.circle_data && wallet.circle_data.chain_id && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Chain ID:</span>
                  <span>{wallet.circle_data.chain_id}</span>
                </div>
              )}
              {wallet.last_used_at && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Last Used:</span>
                  <span>{new Date(wallet.last_used_at * 1000).toLocaleString()}</span>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
