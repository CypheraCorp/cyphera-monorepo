'use client';

import { useState, type FormEvent } from 'react';
import { parseEther, formatEther, type Hex } from 'viem';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { AlertCircle, Send, Wallet, CheckCircle2, Loader2 } from 'lucide-react';
import { useAccount, useBalance, useSendTransaction, useWaitForTransactionReceipt } from 'wagmi';
import { useWeb3AuthUser } from '@web3auth/modal/react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { logger } from '@/lib/core/logger/logger-utils';
export function CustomerSmartAccountActions() {
  const { address, isConnected } = useAccount();
  const { userInfo } = useWeb3AuthUser();
  const { data: balance, isLoading: isLoadingBalance } = useBalance({ address });
  const {
    data: transactionHash,
    error: transactionError,
    isPending: isTransactionPending,
    sendTransaction,
  } = useSendTransaction();
  const { isLoading: isConfirming, isSuccess: isConfirmed } = useWaitForTransactionReceipt({
    hash: transactionHash,
  });

  // Smart account is ready if we have an address and connection
  const isSmartAccountReady = isConnected && !!address;
  const isSmartAccount = true; // With web3auth, we always use smart accounts

  const [toAddress, setToAddress] = useState('');
  const [amount, setAmount] = useState('');

  const handleSendTransaction = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!toAddress || !amount) return;

    try {
      sendTransaction({
        to: toAddress as Hex,
        value: parseEther(amount),
      });
    } catch (error) {
      logger.error('Transaction failed:', error);
    }
  };

  if (!isConnected) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Wallet className="h-5 w-5" />
            Smart Account Actions
          </CardTitle>
          <CardDescription>
            Connect your Web3Auth wallet to access smart account features
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Please sign in with Web3Auth to access your smart account.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Smart Account Status */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Wallet className="h-5 w-5" />
            Smart Account Status
          </CardTitle>
          <CardDescription>
            Your Web3Auth embedded wallet with smart account capabilities
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label className="text-sm font-medium">Account Type</Label>
              <div className="flex items-center gap-2 mt-1">
                {isSmartAccount ? (
                  <Badge variant="default" className="bg-green-100 text-green-800">
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Smart Account
                  </Badge>
                ) : (
                  <Badge variant="secondary">
                    <AlertCircle className="h-3 w-3 mr-1" />
                    Regular Account
                  </Badge>
                )}
              </div>
            </div>

            <div>
              <Label className="text-sm font-medium">Status</Label>
              <div className="flex items-center gap-2 mt-1">
                {isSmartAccountReady ? (
                  <Badge variant="default" className="bg-green-100 text-green-800">
                    <CheckCircle2 className="h-3 w-3 mr-1" />
                    Ready
                  </Badge>
                ) : (
                  <Badge variant="outline">
                    <AlertCircle className="h-3 w-3 mr-1" />
                    Not Ready
                  </Badge>
                )}
              </div>
            </div>
          </div>

          <div>
            <Label className="text-sm font-medium">Smart Account Address</Label>
            <div className="font-mono text-sm bg-muted p-2 rounded mt-1 break-all">
              {address || 'Not connected'}
            </div>
          </div>

          <div>
            <Label className="text-sm font-medium">Balance</Label>
            <div className="text-lg font-semibold mt-1">
              {isLoadingBalance ? (
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading...
                </div>
              ) : balance ? (
                `${formatEther(balance.value)} ${balance.symbol}`
              ) : (
                '0 ETH'
              )}
            </div>
          </div>

          {userInfo && (
            <div>
              <Label className="text-sm font-medium">Account Owner</Label>
              <div className="text-sm text-muted-foreground mt-1">
                {userInfo.name} ({userInfo.email})
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Send Transaction */}
      {isSmartAccountReady && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Send className="h-5 w-5" />
              Send Transaction
            </CardTitle>
            <CardDescription>
              Send ETH using your smart account (gas fees sponsored by Pimlico)
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSendTransaction} className="space-y-4">
              <div>
                <Label htmlFor="to-address">Recipient Address</Label>
                <Input
                  id="to-address"
                  type="text"
                  placeholder="0x..."
                  value={toAddress}
                  onChange={(e) => setToAddress(e.target.value)}
                  required
                />
              </div>

              <div>
                <Label htmlFor="amount">Amount (ETH)</Label>
                <Input
                  id="amount"
                  type="number"
                  step="0.000000001"
                  placeholder="0.001"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  required
                />
              </div>

              <Button
                type="submit"
                disabled={isTransactionPending || isConfirming || !toAddress || !amount}
                className="w-full"
              >
                {isTransactionPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    Confirming...
                  </>
                ) : isConfirming ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    Waiting for confirmation...
                  </>
                ) : (
                  <>
                    <Send className="h-4 w-4 mr-2" />
                    Send Transaction
                  </>
                )}
              </Button>
            </form>

            {/* Transaction Status */}
            {transactionHash && (
              <div className="mt-4 space-y-2">
                <div className="text-sm">
                  <Label>Transaction Hash:</Label>
                  <div className="font-mono text-xs bg-muted p-2 rounded mt-1 break-all">
                    {transactionHash}
                  </div>
                </div>

                {isConfirming && (
                  <Alert>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <AlertDescription>Waiting for transaction confirmation...</AlertDescription>
                  </Alert>
                )}

                {isConfirmed && (
                  <Alert className="border-green-200 bg-green-50">
                    <CheckCircle2 className="h-4 w-4 text-green-600" />
                    <AlertDescription className="text-green-800">
                      Transaction confirmed successfully!
                    </AlertDescription>
                  </Alert>
                )}
              </div>
            )}

            {transactionError && (
              <Alert variant="destructive" className="mt-4">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>Transaction failed: {transactionError.message}</AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      )}

      {/* Smart Account Benefits */}
      <Card>
        <CardHeader>
          <CardTitle>Smart Account Benefits</CardTitle>
          <CardDescription>Advantages of using Web3Auth smart accounts</CardDescription>
        </CardHeader>
        <CardContent>
          <ul className="space-y-2 text-sm">
            <li className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span>Gas fees sponsored by Pimlico paymaster</span>
            </li>
            <li className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span>No need to hold native tokens for gas</span>
            </li>
            <li className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span>Enhanced security with account abstraction</span>
            </li>
            <li className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span>Seamless user experience</span>
            </li>
            <li className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span>Social login with embedded wallet</span>
            </li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
