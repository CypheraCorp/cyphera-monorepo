// Circle transaction related types

/**
 * Fee levels for Circle transactions
 */
export type CircleTransactionFeeLevel = 'LOW' | 'MEDIUM' | 'HIGH';

/**
 * Estimated fee response from Circle API
 */
export interface CircleEstimatedFee {
  low: {
    amount: string;
    amountInUSD: string;
  };
  medium: {
    amount: string;
    amountInUSD: string;
  };
  high: {
    amount: string;
    amountInUSD: string;
  };
}

/**
 * Circle transaction creation request
 */
export interface CircleTransactionRequest {
  idempotency_key: string;
  amounts: string[];
  destination_address: string;
  token_id?: string;
  wallet_id: string;
  fee_level: CircleTransactionFeeLevel;
  ref_id?: string;
}

/**
 * Circle transaction creation response
 */
export interface CircleTransactionResponse {
  challenge_id: string;
  message?: string;
}

// Add the CircleTransaction interface
export interface CircleTransaction {
  id: string;
  walletId: string;
  destinationAddress?: string;
  amounts?: string[];
  state: string;
  errorReason?: string;
  createDate: string;
  txHash?: string;
}

/**
 * Circle API Types
 */

// Circle User Response
export interface CircleUserResponse {
  data: {
    user: {
      id: string;
      status: string;
      pinStatus: string;
      secretsStatus: string;
      userChecksStatus: string;
      accountType: string;
      createDate: string;
      updateDate: string;
    };
  };
}

// Circle Create User Request
export interface CreateUserWithPinAuthRequest {
  external_user_id: string;
}

// Pin Details
export interface PinDetails {
  failedAttempts: number;
  lockedDate: string;
  lockedExpiryDate: string;
  lastLockOverrideDate: string;
}

// Circle User Data
export interface CircleUserData {
  id: string;
  createDate: string;
  pinStatus: string;
  status: string;
  securityQuestionStatus: string;
  isPinSetUp: boolean;
  pinDetails: PinDetails;
  securityQuestionDetails: PinDetails;
}

// Circle Requeset with Idempotency Key and Circle User Token
export interface CircleRequestWithIdempotencyKeyAndToken {
  idempotency_key: string;
  user_token: string;
}

// Token Data
export interface TokenData {
  userToken: string;
  encryptionKey: string;
}
// Circle User Token Response
export interface CircleUserTokenResponse {
  data: {
    userToken: string;
    encryptionKey: string;
  };
}

// Circle Challenge Response
export interface CircleChallengeResponse {
  data: {
    challenge: {
      id: string;
      status: string;
      createDate: string;
      updateDate: string;
      expireDate: string;
    };
  };
}

export interface CircleCreateChallengeResponse {
  data: {
    challengeId: string;
  };
}

// Circle User Initialization Response
export interface CircleUserInitResponse {
  data: {
    challengeId: string;
  };
  message?: string;
}

// Circle Create Wallets Request
export interface CreateWalletsRequest {
  idempotency_key: string;
  blockchains: string[];
  account_type: string;
  user_token: string;
  metadata?: Array<{
    name: string;
    ref_id: string;
  }>;
}

// Circle Create Wallets Response
export interface CircleCreateWalletsResponse {
  challenge_id: string;
  message?: string;
}

// Circle Wallet Response from the API
export interface CircleWalletResponse {
  data: {
    wallet: CircleWallet;
  };
}

// Circle Wallet List Response
export interface CircleWalletListResponse {
  data: {
    wallets: CircleWallet[];
  };
  pagination?: {
    hasBefore?: boolean;
    hasAfter?: boolean;
    before?: string;
    after?: string;
  };
}

// Circle Wallet
export interface CircleWallet {
  id: string;
  custodyType: string;
  address: string;
  blockchain: string;
  accountType: string;
  updateDate: string;
  createDate: string;
  state: string;
  walletSetId?: string;
}

// Circle Wallet Balance
interface WalletBalance {
  amount: string;
  token: {
    id: string;
    symbol: string;
    decimals: number;
    isNative: boolean;
    blockchain: string;
    address?: string;
    standard?: string;
  } | null;
}

// Circle Wallet Balance Response
export interface CircleWalletBalanceResponse {
  data: {
    balances: WalletBalance[];
  };
  pagination?: {
    hasBefore?: boolean;
    hasAfter?: boolean;
    before?: string;
    after?: string;
  };
}

// Circle Transaction Types
export interface CircleTransaction {
  id: string;
  abiFunctionSignature?: string;
  abiParameters?: string[];
  amounts?: string[];
  amountInUSD?: string;
  blockHash?: string;
  blockHeight?: number;
  blockchain: string;
  contractAddress?: string;
  createDate: string;
  custodyType: string;
  destinationAddress?: string;
  errorReason?: string;
  errorDetails?: string;
  estimatedFee?: {
    gasLimit: string;
    gasPrice?: string;
    maxFee?: string;
    priorityFee?: string;
    baseFee?: string;
    networkFee?: string;
  };
  feeLevel?: string;
  firstConfirmDate?: string;
  networkFee?: string;
  networkFeeInUSD?: string;
  nfts?: string[];
  operation: string;
  refId?: string;
  sourceAddress?: string;
  state: string;
  tokenId?: string;
  transactionType: string;
  txHash?: string;
  updateDate: string;
  userId: string;
  walletId: string;
  transactionScreeningEvaluation?: {
    ruleName: string;
    actions: string[];
    screeningDate: string;
    reasons: {
      source: string;
      sourceValue: string;
      riskScore: string;
      riskCategories: string[];
      type: string;
    }[];
  };
}

// Circle Transaction List Response
export interface CircleTransactionListResponse {
  data: {
    transactions: CircleTransaction[];
  };
  pagination?: {
    hasBefore?: boolean;
    hasAfter?: boolean;
    before?: string;
    after?: string;
  };
}

// Request for initializing user
// Export this interface
export interface InitializeUserRequest {
  idempotency_key: string;
  account_type?: string;
  blockchains: string[];
  metadata?: Array<{
    name: string;
    ref_id: string;
  }>;
}
