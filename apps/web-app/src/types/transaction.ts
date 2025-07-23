/**
 * Transaction type types
 */
export type TransactionType = 'charge' | 'refund' | 'transfer' | 'deposit';

/**
 * Transaction status types
 */
export type TransactionStatus = 'succeeded' | 'failed' | 'pending' | 'processing';

/**
 * Transaction interface
 */
export interface TransactionResponse {
  id: string;
  date: string;
  type: TransactionType;
  amount: number;
  status: TransactionStatus;
  customer: string;
  description: string;
  paymentMethod: string;
  // Circle wallet specific fields
  walletId?: string;
  blockchain?: string;
  sourceAddress?: string;
  destinationAddress?: string;
  tokenSymbol?: string;
  txHash?: string;
}
