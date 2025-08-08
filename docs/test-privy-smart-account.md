## Test Privy Smart Account Guide

This document explains the end-to-end flow implemented by the web app's `test-privy` page: authentication with Privy, connecting a MetaMask Delegation Toolkit smart account using viem, deploying the smart account on-chain, switching networks, and sending a transaction.

- Page: `apps/web-app/src/app/test-privy/page.tsx`
- Providers: `apps/web-app/src/components/providers/privy-provider.tsx`, `apps/web-app/src/hooks/privy/use-privy-smart-account.tsx`
- UI: `apps/web-app/src/components/test/privy-basic-test.tsx`, `apps/web-app/src/components/test/privy-smart-account-test.tsx`
- Wallet components: `apps/web-app/src/components/wallet/`
- Network helpers: `apps/web-app/src/lib/web3/dynamic-networks.ts`

### High-level Architecture

- The page renders two tabs:
  - Authentication: `PrivyBasicTest`
  - Smart Account: `PrivySmartAccountTest`
- The root wraps content in two providers:
  - `PrivyProvider`: Configures the Privy SDK, supported chains, and TanStack Query.
  - `PrivySmartAccountProvider`: Creates and manages a MetaMask Hybrid Smart Account, a Pimlico Bundler and Paymaster, and exposes actions like deploy and network switch.

### Environment Variables

Set these env vars for complete functionality:

- `NEXT_PUBLIC_PRIVY_APP_ID`: Privy application ID (required).
- `NEXT_PUBLIC_WALLET_CONNECT_PROJECT_ID`: WalletConnect Cloud project ID (optional for external wallets).
- `NEXT_PUBLIC_INFURA_API_KEY`: Used to build performant RPC endpoints for supported chains.
- `NEXT_PUBLIC_PIMLICO_API_KEY`: Used for Sponsored AA via Pimlico Bundler/Paymaster v2 endpoints.

### Networks and Tokens

- Supported test networks in the UI: Base Sepolia (84532) and Ethereum Sepolia (11155111).
- Dynamic network configuration is loaded from `/api/networks?active=true` via `getAllNetworkConfigs` and transformed in `dynamic-networks.ts`.
- Hardcoded USDC addresses as fallback:
  - Base Sepolia: `0x036CbD53842c5426634e7929541eC2318f3dCF7e`
  - Ethereum Sepolia: `0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238`

### Authentication Flow (Privy)

Files:
- `apps/web-app/src/components/providers/privy-provider.tsx`
- `apps/web-app/src/components/test/privy-basic-test.tsx`

Flow:
1. `PrivyProvider` initializes Privy with dynamic `supportedChains` and a `defaultChain` using `getAllNetworkConfigs`. It ensures Base Sepolia and Ethereum Sepolia are included.
2. `PrivyBasicTest` uses `usePrivy()` to expose `login`, `logout`, `authenticated`, and `ready`. It also reads `useWallets()` to surface the Privy embedded wallet (EOA) once created.
3. On login, Privy creates an embedded EOA for the user. This EOA is used as the signer for the smart account.

Key settings:
- `embeddedWallets.createOnLogin: 'users-without-wallets'` creates an embedded wallet for new users.
- `loginMethods` include email/social and `wallet` (external wallets) as fallback.

### Smart Account Initialization (MetaMask Delegation Toolkit + viem)

File:
- `apps/web-app/src/hooks/privy/use-privy-smart-account.tsx`

Core steps:
1. Identify the Privy embedded wallet via `useWallets()` where `walletClientType === 'privy'`.
2. Obtain an EIP-1193 provider from the embedded EOA: `eoa.getEthereumProvider()`.
3. Determine the current `chainId` (supports CAIP-2 string, hex, or decimal) and resolve `networkConfig` via `getNetworkConfig(chainId)` with fallbacks for Base Sepolia and Ethereum Sepolia.
4. Create a viem `publicClient` with the resolved chain and RPC URL.
5. Load the MetaMask Delegation Toolkit via `getDelegationToolkit()` and verify the Smart Account Factory is deployed at `0x69Aa2f9fe1572F1B640E1bbc512f5c3a734fc77c`.
6. Create a viem `WalletClient` using the EIP-1193 provider (official Privy + viem integration pattern).
7. Create a MetaMask Hybrid Smart Account using the toolkit’s `toMetaMaskSmartAccount` and set initial state in context:
   - `smartAccount` and `smartAccountAddress`
   - `smartAccountReady` (true when initialized)
8. Initialize AA clients if `NEXT_PUBLIC_PIMLICO_API_KEY` is set:
   - `createPaymasterClient` (sponsored gas)
   - `createBundlerClient` with integrated paymaster
   - `createPimlicoClient` for gas price operations
9. Detect deployment status via `smartAccount.isDeployed()` and expose `isDeployed`.

Exposed context API:
- `smartAccount`, `smartAccountAddress`, `smartAccountReady`
- `bundlerClient`, `pimlicoClient`
- `isDeployed`, `checkDeploymentStatus()`
- `deploySmartAccount()`
- `switchNetwork(chainId)`
- `currentChainId`

### Deploying the Smart Account

Where used:
- `apps/web-app/src/components/test/privy-smart-account-test.tsx`

Behavior:
1. If not already deployed, `deploySmartAccount()` submits a minimal UserOperation via the bundler:
   - Calls: a 0-value self-call to the smart account address to trigger deployment.
   - Gas pricing: fetched from Pimlico via `getUserOperationGasPrice()` and forwarded as BigInt.
   - Paymaster: configured for gas sponsorship.
2. The hook retries up to 3 times with 2s backoff on errors and finally checks `isDeployed()` and, if needed, bytecode presence via `publicClient.getCode` to confirm.

### Switching Networks

Files:
- `apps/web-app/src/components/wallet/network-switcher.tsx`
- `apps/web-app/src/hooks/privy/use-privy-smart-account.tsx` (method `switchNetwork`)

Behavior:
1. `NetworkSwitcher` exposes Base Sepolia and Ethereum Sepolia with an in-UI dropdown.
2. On selection, `switchNetwork(chainId)` calls `embeddedWallet.switchChain(chainId)` and resets smart account state (clients and flags), causing the hook to re-initialize on the new chain.

### Sending a Transaction (UserOperation)

Files:
- `apps/web-app/src/components/wallet/send-transaction.tsx`

Flow:
1. The form collects recipient, amount, and token (native or USDC when available via `getUSDCAddress`).
2. For native token, constructs a single call `{ to, value, data: '0x' }`.
3. For ERC-20, constructs an `a9059cbb` transfer calldata with properly padded recipient and amount (decimals-aware).
4. Gas pricing fetched from Pimlico (`getUserOperationGasPrice`) and forwarded as BigInt to `bundlerClient.sendUserOperation`.
5. Waits for `waitForUserOperationReceipt`. On success, shows explorer link (BaseScan or Etherscan for Sepolia).
6. Transactions are sponsored via the configured paymaster; no native ETH required for gas.

### Wallet Dashboard

File:
- `apps/web-app/src/components/wallet/wallet-dashboard.tsx`

Includes:
- `WalletBalance`: Fetches native and USDC balances via `publicClient.getBalance` and `readContract(balanceOf/decimals)`.
- `TransactionHistory`: Fetches transactions using a robust, rate-limit-friendly approach:
  - Direct smart account log scan across recent blocks with batching and exponential backoff.
  - Dedicated ERC-20 Transfer event queries per configured token.
  - Deduplicates by transaction hash and sorts by block number (newest first).
  - Filter dropdown (All/ETH/USDC) and explorer links by chain.
- `SendTransaction`: The UI version of the sending flow described above, with form validation and success/error UX.

Rate limiting protections in `TransactionHistory`:
- Batches of 100 blocks with 200ms delays between batches.
- Up to 3 retries with exponential backoff (1s → 2s → 4s) on 429/limit errors.

### The Test Page Composition

File:
- `apps/web-app/src/app/test-privy/page.tsx`

Structure:
1. Wraps with `PrivyProvider` and `PrivySmartAccountProvider`.
2. Tabbed UI:
   - Authentication: `PrivyBasicTest` renders login/logout, readiness, user info, and embedded EOA details.
   - Smart Account: `PrivySmartAccountTest` renders `WalletDashboard`, a technical panel for network switching, deployment, and sending a test transaction.

### Explorer Links (Sepolia Testnets)

- Base Sepolia (84532): `https://sepolia.basescan.org`
- Ethereum Sepolia (11155111): `https://sepolia.etherscan.io`

### Logging and Error Handling

- All major steps log through `logger` utilities for debugging.
- Smart account creation and deployment paths include verbose context and error stacks.
- Network fetches use resilient fallbacks if backend config is unavailable.

### Troubleshooting

- Missing Privy App ID: Ensure `NEXT_PUBLIC_PRIVY_APP_ID` is set; the provider logs an error if not.
- No embedded wallet after login: Confirm `embeddedWallets.createOnLogin` is enabled and the user completed authentication.
- Pimlico not sponsoring: Make sure `NEXT_PUBLIC_PIMLICO_API_KEY` is set and the chain is supported.
- Rate limits when fetching history: The component backs off automatically; consider reducing the `blockRange` or increasing delays if needed.
- Smart Account factory check fails: Verify the factory address is deployed on the selected testnet.

### File Reference Index

- Page and layout:
  - `apps/web-app/src/app/test-privy/page.tsx`
  - `apps/web-app/src/app/test-privy/layout.tsx`
- Providers and hooks:
  - `apps/web-app/src/components/providers/privy-provider.tsx`
  - `apps/web-app/src/hooks/privy/use-privy-smart-account.tsx`
- Test components:
  - `apps/web-app/src/components/test/privy-basic-test.tsx`
  - `apps/web-app/src/components/test/privy-smart-account-test.tsx`
- Wallet components:
  - `apps/web-app/src/components/wallet/wallet-dashboard.tsx`
  - `apps/web-app/src/components/wallet/wallet-balance.tsx`
  - `apps/web-app/src/components/wallet/transaction-history.tsx`
  - `apps/web-app/src/components/wallet/send-transaction.tsx`
  - `apps/web-app/src/components/wallet/network-switcher.tsx`
- Network helpers:
  - `apps/web-app/src/lib/web3/dynamic-networks.ts`

