/**
 * Blockchain constants and utility functions
 * NOTE: Hardcoded blockchain data (CIRCLE_BLOCKCHAINS) and related functions
 * (getExplorerUrl, getAddressExplorerUrl, getBlockchainInfo) have been removed
 * as this data should now be fetched dynamically from the API.
 * Use `generateExplorerLink` from `@/lib/utils/explorers` for explorer URLs.
 */

// Keep generic types for now, might be superseded by API types
export type BlockchainType = 'ETH' | 'MATIC' | 'ARB' | 'BASE';
export type BlockchainEnvironment = 'MAINNET' | 'TESTNET';
