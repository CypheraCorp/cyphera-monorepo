'use client';

import React from 'react';
import { WalletResponse } from '@/types/wallet';
import { NetworkWithTokensResponse } from '@/types/network';
import { CircleWalletBalances } from '@/components/wallets/circle-wallet-balances';
import { TransactionHistory } from '@/components/wallets/transaction-history';

interface WalletDetailsContentProps {
  wallet: WalletResponse;
  networks: NetworkWithTokensResponse[];
}

export function WalletDetailsContent({ wallet, networks }: WalletDetailsContentProps) {
  // Check if this is a Circle wallet
  const isCircleWallet = wallet.wallet_type === 'circle_wallet' || 
                        wallet.wallet_type === 'circle' || 
                        !!wallet.circle_data;

  return (
    <>
      {/* Show balances for Circle wallets in the second column */}
      {isCircleWallet && (
        <CircleWalletBalances 
          wallet={wallet} 
          workspaceId={wallet.workspace_id}
          networks={networks}
        />
      )}

      {/* Show transaction history for Circle wallets below the cards */}
      {isCircleWallet && (
        <div className="mt-6">
          <TransactionHistory
            wallet={wallet}
            workspaceId={wallet.workspace_id}
            networks={networks}
          />
        </div>
      )}
    </>
  );
}