"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.redeemDelegation = exports.getFeePerGas = exports.createMetaMaskAccount = exports.bundlerClient = exports.publicClient = void 0;
const delegator_core_viem_1 = require("@metamask-private/delegator-core-viem");
const viem_1 = require("viem");
const accounts_1 = require("viem/accounts");
const chains_1 = require("viem/chains");
const config_1 = require("../config/config");
const utils_1 = require("../utils/utils");
const delegation_helpers_1 = require("../utils/delegation-helpers");
// Try to use viem/account-abstraction first, or fall back to permissionless
let createBundlerClient, createPaymasterClient;
try {
    // Attempt to import from viem/account-abstraction (viem >= 2.x)
    const viemAA = require("viem/account-abstraction");
    createBundlerClient = viemAA.createBundlerClient;
    createPaymasterClient = viemAA.createPaymasterClient;
    utils_1.logger.info("Using viem/account-abstraction for bundler client");
}
catch (error) {
    // Fall back to permissionless if viem/account-abstraction doesn't exist
    utils_1.logger.info("viem/account-abstraction not found, falling back to permissionless");
    const permissionless = require("permissionless/clients/bundler");
    createBundlerClient = permissionless.createBundlerClient;
    createPaymasterClient = permissionless.createPaymasterClient;
}
// Use a const for the EntryPoint address to avoid hardcoding in multiple places
const ENTRY_POINT_ADDRESS = '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'; // Ethereum EntryPoint v0.6
// Initialize clients
const chain = chains_1.sepolia;
// Create public client for reading blockchain state
exports.publicClient = (0, viem_1.createPublicClient)({
    chain,
    transport: (0, viem_1.http)(config_1.config.blockchain.rpcUrl)
});
// Create bundler client for user operations
exports.bundlerClient = createBundlerClient({
    chain,
    transport: (0, viem_1.http)(config_1.config.blockchain.bundlerUrl),
    paymaster: config_1.config.blockchain.paymasterUrl
        ? createPaymasterClient({
            transport: (0, viem_1.http)(config_1.config.blockchain.paymasterUrl)
        })
        : undefined
});
/**
 * Creates a MetaMask smart account from a private key
 */
const createMetaMaskAccount = async (privateKey) => {
    try {
        const formattedKey = (0, utils_1.formatPrivateKey)(privateKey);
        const account = (0, accounts_1.privateKeyToAccount)(formattedKey);
        utils_1.logger.info(`Creating MetaMask Smart Account for address: ${account.address}`);
        const smartAccount = await (0, delegator_core_viem_1.toMetaMaskSmartAccount)({
            client: exports.publicClient,
            implementation: delegator_core_viem_1.Implementation.Hybrid,
            deployParams: [account.address, [], [], []],
            deploySalt: "0x",
            signatory: { account }
        });
        utils_1.logger.info(`Smart Account address: ${smartAccount.address}`);
        return smartAccount;
    }
    catch (error) {
        utils_1.logger.error(`Failed to create MetaMask account:`, error);
        throw error;
    }
};
exports.createMetaMaskAccount = createMetaMaskAccount;
/**
 * Gets the fee per gas for a user operation
 */
const getFeePerGas = async () => {
    try {
        // Simplified fee estimation - could be improved with actual gas estimation
        return {
            maxFeePerGas: (0, viem_1.parseEther)("0.00000001"),
            maxPriorityFeePerGas: (0, viem_1.parseEther)("0.000000001")
        };
    }
    catch (error) {
        utils_1.logger.error(`Failed to get fee per gas:`, error);
        throw error;
    }
};
exports.getFeePerGas = getFeePerGas;
/**
 * Redeems a delegation, executing actions on behalf of the delegator
 * @param delegationData The serialized delegation data
 * @returns The transaction hash
 */
const redeemDelegation = async (delegationData) => {
    try {
        // Parse the delegation data using our helper
        const delegation = (0, delegation_helpers_1.parseDelegation)(delegationData);
        // Validate the delegation
        (0, delegation_helpers_1.validateDelegation)(delegation);
        utils_1.logger.info("Redeeming delegation...");
        utils_1.logger.debug("Delegation details:", {
            delegate: delegation.delegate,
            delegator: delegation.delegator,
            expiry: delegation.expiry?.toString()
        });
        // Create redeemer account from private key
        if (!config_1.config.blockchain.privateKey) {
            throw new Error('Private key is not configured');
        }
        const redeemer = await (0, exports.createMetaMaskAccount)(config_1.config.blockchain.privateKey);
        // Verify redeemer address matches delegate in delegation
        if (!(0, viem_1.isAddressEqual)(redeemer.address, delegation.delegate)) {
            throw new Error(`Redeemer account address does not match delegate in delegation. ` +
                `Redeemer: ${redeemer.address}, delegate: ${delegation.delegate}`);
        }
        // We need to treat the delegation as the required type for DelegationFramework
        // This casting is necessary because our types might differ slightly from the framework's types
        const delegationForFramework = delegation;
        const delegationChain = [delegationForFramework];
        // The execution that will be performed on behalf of the delegator
        // In this case, sending a minimal amount of ETH from delegator to redeemer as proof
        const executions = [
            {
                target: redeemer.address,
                value: (0, viem_1.parseEther)("0.000001"), // Minimal value for proof of successful execution
                callData: "0x"
            }
        ];
        // Create the calldata for redeeming the delegation
        const redeemDelegationCalldata = delegator_core_viem_1.DelegationFramework.encode.redeemDelegations([delegationChain], [delegator_core_viem_1.SINGLE_DEFAULT_MODE], [executions]);
        // The call to the delegation framework to redeem the delegation
        const calls = [
            {
                to: redeemer.address,
                data: redeemDelegationCalldata
            }
        ];
        // Get fee per gas
        const feePerGas = await (0, exports.getFeePerGas)();
        // Encode calldata based on account interface
        let callData;
        if ('encodeCallData' in redeemer && typeof redeemer.encodeCallData === 'function') {
            callData = redeemer.encodeCallData(calls);
        }
        else if ('encodeCalls' in redeemer && typeof redeemer.encodeCalls === 'function') {
            callData = redeemer.encodeCalls(calls);
        }
        else {
            throw new Error('Account does not have encodeCallData or encodeCalls method');
        }
        utils_1.logger.info("Sending UserOperation...");
        const userOperationHash = await exports.bundlerClient.sendUserOperation({
            account: redeemer,
            userOperation: {
                callData,
                maxFeePerGas: feePerGas.maxFeePerGas,
                maxPriorityFeePerGas: feePerGas.maxPriorityFeePerGas
            },
            entryPoint: ENTRY_POINT_ADDRESS
        });
        utils_1.logger.info("UserOperation hash:", userOperationHash);
        // Wait for the user operation to be included in a transaction
        utils_1.logger.info("Waiting for transaction receipt...");
        const receipt = await exports.bundlerClient.waitForUserOperationReceipt({
            hash: userOperationHash,
            timeout: 60000 // 60 second timeout
        });
        const transactionHash = receipt.receipt.transactionHash;
        utils_1.logger.info("Transaction confirmed:", transactionHash);
        return transactionHash;
    }
    catch (error) {
        utils_1.logger.error("Error redeeming delegation:", error);
        throw error;
    }
};
exports.redeemDelegation = redeemDelegation;
