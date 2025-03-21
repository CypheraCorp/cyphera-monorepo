"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.redeemDelegation = exports.getFeePerGas = exports.createMetaMaskAccount = exports.bundlerClient = exports.publicClient = void 0;
const delegator_core_viem_1 = require("@metamask-private/delegator-core-viem");
const viem_1 = require("viem");
const accounts_1 = require("viem/accounts");
const chains_1 = require("viem/chains");
const config_1 = require("../config/config");
// Try to use viem/account-abstraction first, or fall back to permissionless
let createBundlerClient, createPaymasterClient;
try {
    // Attempt to import from viem/account-abstraction (viem >= 2.x)
    const viemAA = require("viem/account-abstraction");
    createBundlerClient = viemAA.createBundlerClient;
    createPaymasterClient = viemAA.createPaymasterClient;
    console.log("Using viem/account-abstraction for bundler client");
}
catch (error) {
    // Fall back to permissionless if viem/account-abstraction doesn't exist
    console.log("viem/account-abstraction not found, falling back to permissionless");
    const permissionless = require("permissionless/clients/bundler");
    createBundlerClient = permissionless.createBundlerClient;
    createPaymasterClient = permissionless.createPaymasterClient;
}
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
    const account = (0, accounts_1.privateKeyToAccount)(privateKey);
    console.log(`Creating MetaMask Smart Account for address: ${account.address}`);
    const smartAccount = await (0, delegator_core_viem_1.toMetaMaskSmartAccount)({
        client: exports.publicClient,
        implementation: delegator_core_viem_1.Implementation.Hybrid,
        deployParams: [account.address, [], [], []],
        deploySalt: "0x",
        signatory: { account }
    });
    console.log(`Smart Account address: ${smartAccount.address}`);
    return smartAccount;
};
exports.createMetaMaskAccount = createMetaMaskAccount;
/**
 * Gets the fee per gas for a user operation
 */
const getFeePerGas = async () => {
    // Simplified fee estimation for viem v2
    return {
        maxFeePerGas: (0, viem_1.parseEther)("0.00000001"),
        maxPriorityFeePerGas: (0, viem_1.parseEther)("0.000000001")
    };
};
exports.getFeePerGas = getFeePerGas;
/**
 * Redeems a delegation, executing actions on behalf of the delegator
 * @param delegation The signed delegation
 * @param delegatorFactoryArgs Optional factory args if delegator account needs to be deployed
 * @returns The user operation hash
 */
const redeemDelegation = async (delegation, delegatorFactoryArgs) => {
    console.log("Redeeming delegation...");
    console.log("Delegate address:", delegation.delegate);
    console.log("Delegator address:", delegation.delegator);
    // Create redeemer account from private key
    const redeemer = await (0, exports.createMetaMaskAccount)(config_1.config.blockchain.privateKey);
    // Verify redeemer address matches delegate in delegation
    if (!(0, viem_1.isAddressEqual)(redeemer.address, delegation.delegate)) {
        throw new Error(`Redeemer account address does not match delegate in delegation. ` +
            `Redeemer: ${redeemer.address}, delegate: ${delegation.delegate}`);
    }
    const delegationChain = [delegation];
    // The execution that will be performed on behalf of the delegator
    // In this case, sending 0.001 ETH from delegator to redeemer
    const executions = [
        {
            target: redeemer.address,
            value: (0, viem_1.parseEther)("0.001"),
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
    // If delegator account needs to be deployed, add the deployment call
    if (delegatorFactoryArgs) {
        const { factory, factoryData } = delegatorFactoryArgs;
        calls.unshift({
            to: factory,
            data: factoryData
        });
    }
    // Get fee per gas
    const feePerGas = await (0, exports.getFeePerGas)();
    try {
        // Send the user operation
        console.log("Sending UserOperation...");
        // Handle different account interface methods
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
        const userOperationHash = await exports.bundlerClient.sendUserOperation({
            account: redeemer,
            userOperation: {
                callData,
                maxFeePerGas: feePerGas.maxFeePerGas,
                maxPriorityFeePerGas: feePerGas.maxPriorityFeePerGas
            },
            entryPoint: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789' // Ethereum EntryPoint v0.6
        });
        console.log("UserOperation hash:", userOperationHash);
        return userOperationHash;
    }
    catch (error) {
        console.error("Error sending user operation:", error);
        throw error;
    }
};
exports.redeemDelegation = redeemDelegation;
