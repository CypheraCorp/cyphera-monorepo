import * as React from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { WalletResponse } from '@/types/wallet';
import { useCreateWallet } from '@/hooks/data';
import { useToast } from '@/components/ui/use-toast';
import { Loader2, X } from 'lucide-react';
import { getNetworkConfigs } from '@/lib/web3/config/networks';
import { logger } from '@/lib/core/logger/logger-utils';

interface CreateWalletInlineFormProps {
  onWalletCreated: (wallet: WalletResponse) => void;
  onCancel: () => void;
  preSelectedNetworkId?: string;
}

export function CreateWalletInlineForm({
  onWalletCreated,
  onCancel,
  preSelectedNetworkId,
}: CreateWalletInlineFormProps) {
  const [address, setAddress] = React.useState('');
  const [networkId, setNetworkId] = React.useState(preSelectedNetworkId || '');
  const [networks, setNetworks] = React.useState<Array<{ id: string; name: string }>>([]);

  const { toast } = useToast();
  const createWalletMutation = useCreateWallet();

  React.useEffect(() => {
    async function loadNetworks() {
      try {
        const configs = await getNetworkConfigs();
        setNetworks(
          configs.map((c) => ({
            id: c.chain.id.toString(),
            name: c.chain.name,
          }))
        );
      } catch (error) {
        logger.error('Failed to load networks:', error);
      }
    }
    loadNetworks();
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!address || !networkId) {
      toast({
        title: 'Validation Error',
        description: 'Please fill in all required fields',
        variant: 'destructive',
      });
      return;
    }

    try {
      const wallet = await createWalletMutation.mutateAsync({
        wallet_address: address.toLowerCase(),
        network_id: networkId,
        is_primary: false,
        verified: false,
      });

      toast({
        title: 'Wallet Created',
        description: 'Your new wallet has been added successfully',
      });

      onWalletCreated(wallet);
    } catch (error) {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Failed to create wallet',
        variant: 'destructive',
      });
    }
  };

  return (
    <Card>
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">Create New Wallet</CardTitle>
          <Button type="button" variant="ghost" size="icon" onClick={onCancel} className="h-8 w-8">
            <X className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="wallet-address">Wallet Address *</Label>
            <Input
              id="wallet-address"
              placeholder="0x..."
              value={address}
              onChange={(e) => setAddress(e.target.value)}
              pattern="^0x[a-fA-F0-9]{40}$"
              title="Must be a valid Ethereum address starting with 0x"
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="network">Network *</Label>
            <Select
              value={networkId}
              onValueChange={setNetworkId}
              required
              disabled={!!preSelectedNetworkId}
            >
              <SelectTrigger id="network">
                <SelectValue placeholder="Select a network" />
              </SelectTrigger>
              <SelectContent>
                {networks.map((network) => (
                  <SelectItem key={network.id} value={network.id}>
                    {network.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={onCancel}
              disabled={createWalletMutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createWalletMutation.isPending}>
              {createWalletMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                'Create Wallet'
              )}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
