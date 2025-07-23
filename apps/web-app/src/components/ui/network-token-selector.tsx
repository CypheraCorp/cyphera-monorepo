'use client';

import { useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { Check, Zap, Globe, Coins, ChevronRight, ArrowLeft } from 'lucide-react';
import type { NetworkWithTokensResponse } from '@/types/network';

interface NetworkTokenSelectorProps {
  networks: NetworkWithTokensResponse[];
  selectedTokens: Array<{
    network_id: string;
    token_id: string;
    network_name: string;
    token_symbol: string;
    token_name: string;
    is_gas_token: boolean;
  }>;
  onTokensChange: (tokens: NetworkTokenSelectorProps['selectedTokens']) => void;
  disabled?: boolean;
  className?: string;
}

export function NetworkTokenSelector({
  networks,
  selectedTokens,
  onTokensChange,
  disabled = false,
  className,
}: NetworkTokenSelectorProps) {
  const [selectedNetwork, setSelectedNetwork] = useState<string | null>(null);
  const [step, setStep] = useState<'network' | 'tokens'>('network');

  const isTokenSelected = (networkId: string, tokenId: string) => {
    return selectedTokens.some(
      (token) => token.network_id === networkId && token.token_id === tokenId
    );
  };

  const toggleToken = (
    network: NetworkWithTokensResponse['network'],
    token: NetworkWithTokensResponse['tokens'][0]
  ) => {
    if (disabled) return;

    const isSelected = isTokenSelected(network.id, token.id);

    if (isSelected) {
      // Remove token
      const newTokens = selectedTokens.filter(
        (t) => !(t.network_id === network.id && t.token_id === token.id)
      );
      onTokensChange(newTokens);
    } else {
      // Remove all tokens from other networks and add this token
      const newTokens = selectedTokens.filter((t) => t.network_id === network.id);
      newTokens.push({
        network_id: network.id,
        token_id: token.id,
        network_name: network.name,
        token_symbol: token.symbol,
        token_name: token.name,
        is_gas_token: token.gas_token,
      });
      onTokensChange(newTokens);
    }
  };

  const handleNetworkSelect = (networkId: string) => {
    setSelectedNetwork(networkId);
    setStep('tokens');
  };

  const handleBack = () => {
    setStep('network');
  };

  const getNetworkSelectionCount = (networkId: string) => {
    return selectedTokens.filter((token) => token.network_id === networkId).length;
  };

  // Filter networks to only show those with tokens
  const networksWithTokens = networks.filter(({ tokens }) => tokens.length > 0);
  const currentNetwork = networksWithTokens.find((n) => n.network.id === selectedNetwork);

  return (
    <div className={cn('space-y-4', className)}>
      {/* Step Indicator */}
      <div className="flex items-center gap-2 text-sm">
        <div
          className={cn(
            'flex items-center gap-1',
            step === 'network' ? 'text-blue-600 font-medium' : 'text-green-600'
          )}
        >
          <Globe className="h-4 w-4" />
          <span>1. Choose Network</span>
          {selectedNetwork && <Check className="h-4 w-4" />}
        </div>
        <div className="h-px bg-border flex-1" />
        <div
          className={cn(
            'flex items-center gap-1',
            step === 'tokens' && selectedNetwork
              ? 'text-blue-600 font-medium'
              : 'text-muted-foreground'
          )}
        >
          <Coins className="h-4 w-4" />
          <span>2. Select Tokens</span>
          {selectedTokens.length > 0 && <Check className="h-4 w-4 text-green-600" />}
        </div>
      </div>

      {/* Network Selection Step */}
      {step === 'network' && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h4 className="font-medium">Select a Payment Network</h4>
            <Badge variant="outline" className="text-xs">
              Step 1 of 2
            </Badge>
          </div>

          {/* Network Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {networksWithTokens.map(({ network, tokens }) => {
              const selectionCount = getNetworkSelectionCount(network.id);
              const isCurrentlySelected = selectedNetwork === network.id;

              return (
                <Card
                  key={network.id}
                  className={cn(
                    'relative cursor-pointer transition-all duration-200 hover:shadow-lg hover:scale-[1.02]',
                    isCurrentlySelected && 'ring-2 ring-blue-500',
                    selectionCount > 0 &&
                      !isCurrentlySelected &&
                      'border-green-200 dark:border-green-800'
                  )}
                  onClick={() => !disabled && handleNetworkSelect(network.id)}
                >
                  <CardContent className="p-5">
                    <div className="flex items-start justify-between">
                      <div className="flex items-start gap-3">
                        <div
                          className={cn(
                            'w-12 h-12 rounded-full flex items-center justify-center',
                            'bg-gradient-to-br from-blue-400 to-blue-600 shadow-md'
                          )}
                        >
                          <Globe className="h-6 w-6 text-white" />
                        </div>
                        <div className="flex-1">
                          <h5 className="font-semibold text-base">{network.name}</h5>
                          <p className="text-sm text-muted-foreground mt-1">
                            {tokens.length} token{tokens.length > 1 ? 's' : ''} available
                          </p>
                          <div className="flex items-center gap-2 mt-2">
                            {network.is_testnet && (
                              <Badge variant="secondary" className="text-xs">
                                Testnet
                              </Badge>
                            )}
                            {selectionCount > 0 && (
                              <Badge
                                variant="secondary"
                                className="text-xs bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300"
                              >
                                {selectionCount} selected
                              </Badge>
                            )}
                          </div>
                        </div>
                      </div>
                      <ChevronRight className="h-5 w-5 text-muted-foreground mt-1" />
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>

          {networksWithTokens.length === 0 && (
            <Card className="border-dashed border-2">
              <CardContent className="p-6 text-center">
                <Globe className="h-8 w-8 text-muted-foreground mx-auto mb-2" />
                <p className="text-sm text-muted-foreground">
                  No payment networks available. Please configure product tokens for your networks.
                </p>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {/* Token Selection Step */}
      {step === 'tokens' && currentNetwork && (
        <div className="space-y-4 animate-in slide-in-from-right duration-200">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm" onClick={handleBack} className="h-8 w-8 p-0">
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <h4 className="font-medium">Select Payment Tokens</h4>
            </div>
            <Badge variant="outline" className="text-xs">
              Step 2 of 2
            </Badge>
          </div>

          {/* Selected Network Info */}
          <Card className="bg-muted/50">
            <CardContent className="p-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center">
                  <Globe className="h-5 w-5 text-white" />
                </div>
                <div>
                  <p className="font-medium">{currentNetwork.network.name}</p>
                  <p className="text-sm text-muted-foreground">
                    Select one or more tokens that customers can use to pay
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Token List */}
          <div className="space-y-2">
            {currentNetwork.tokens.map((token) => {
              const isSelected = isTokenSelected(currentNetwork.network.id, token.id);

              return (
                <Card
                  key={token.id}
                  className={cn(
                    'cursor-pointer transition-all duration-200 hover:shadow-md',
                    isSelected
                      ? 'border-green-500 bg-green-50 dark:bg-green-950 shadow-md'
                      : 'hover:border-green-200',
                    disabled && 'cursor-default opacity-50'
                  )}
                  onClick={() => toggleToken(currentNetwork.network, token)}
                >
                  <CardContent className="p-4">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div
                          className={cn(
                            'w-10 h-10 rounded-full flex items-center justify-center',
                            isSelected
                              ? 'bg-gradient-to-br from-green-400 to-green-600'
                              : 'bg-gradient-to-br from-gray-400 to-gray-600'
                          )}
                        >
                          {token.gas_token ? (
                            <Zap className="h-5 w-5 text-white" />
                          ) : (
                            <Coins className="h-5 w-5 text-white" />
                          )}
                        </div>
                        <div>
                          <h5 className="font-medium flex items-center gap-2">
                            {token.name}
                            <span className="text-muted-foreground">({token.symbol})</span>
                            {token.gas_token && (
                              <Badge variant="secondary" className="text-xs">
                                Gas Token
                              </Badge>
                            )}
                          </h5>
                          <p className="text-sm text-muted-foreground">{token.decimals} decimals</p>
                        </div>
                      </div>
                      <div
                        className={cn(
                          'w-6 h-6 rounded-full border-2 flex items-center justify-center transition-all',
                          isSelected
                            ? 'bg-green-600 border-green-600'
                            : 'border-gray-300 dark:border-gray-600'
                        )}
                      >
                        {isSelected && <Check className="h-4 w-4 text-white" />}
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </div>
      )}

      {/* Selection Summary */}
      {selectedTokens.length > 0 && (
        <Card className="bg-green-50 dark:bg-green-950 border-green-200 dark:border-green-800">
          <CardContent className="p-4">
            <div className="flex items-center gap-2">
              <Check className="h-5 w-5 text-green-600" />
              <div className="flex-1">
                <p className="font-medium text-sm">Payment Options Selected</p>
                <div className="flex flex-wrap gap-2 mt-2">
                  {selectedTokens.map((token) => (
                    <Badge
                      key={`${token.network_id}-${token.token_id}`}
                      variant="outline"
                      className="text-xs"
                    >
                      {token.token_symbol} on {token.network_name}
                    </Badge>
                  ))}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
