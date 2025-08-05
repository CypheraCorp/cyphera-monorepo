'use client';

import { useRouter } from 'next/navigation';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface NetworkSelectorProps {
  networks: Array<{
    id: string;
    name: string;
    chainId: number;
    isTestnet: boolean;
  }>;
  selectedNetworkId: string;
  walletAddress: string;
}

export function NetworkSelector({ networks, selectedNetworkId, walletAddress }: NetworkSelectorProps) {
  const router = useRouter();

  const handleNetworkChange = (networkId: string) => {
    // Navigate to the same address with different network
    router.push(`/merchants/wallets/address/${encodeURIComponent(walletAddress)}?network=${networkId}`);
  };

  return (
    <Select value={selectedNetworkId} onValueChange={handleNetworkChange}>
      <SelectTrigger className="w-[200px]">
        <SelectValue placeholder="Select network" />
      </SelectTrigger>
      <SelectContent>
        {networks.map((network) => (
          <SelectItem key={network.id} value={network.id}>
            {network.name}
            {network.isTestnet && (
              <span className="ml-2 text-xs text-muted-foreground">(Testnet)</span>
            )}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}