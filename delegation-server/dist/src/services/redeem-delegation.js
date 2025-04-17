"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.redeemDelegation = exports.getFeePerGas = exports.createMetaMaskAccount = exports.bundlerClient = exports.publicClient = void 0;
exports.getBundlerClient = getBundlerClient;
const delegation_toolkit_1 = require("@metamask/delegation-toolkit");
const viem_1 = require("viem");
const accounts_1 = require("viem/accounts");
const chains_1 = require("viem/chains");
const config_1 = require("../config/config");
const utils_1 = require("../utils/utils");
const delegation_helpers_1 = require("../utils/delegation-helpers");
const erc20_1 = require("../abis/erc20");
// Import account abstraction types and bundler clients
const account_abstraction_1 = require("viem/account-abstraction");
const pimlico_1 = require("permissionless/clients/pimlico");
const node_fetch_1 = __importDefault(require("node-fetch"));
// Initialize clients
const chain = chains_1.sepolia;
// Create a custom transport that uses node-fetch
const createTransport = (url) => {
    if (!url) {
        throw new Error('URL is required for transport');
    }
    return (0, viem_1.custom)({
        async request({ method, params }) {
            const response = await (0, node_fetch_1.default)(url, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    jsonrpc: '2.0',
                    method,
                    params,
                    id: 1,
                }),
            });
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            const data = await response.json();
            if (data.error) {
                throw new Error(data.error.message);
            }
            return data.result;
        }
    });
};
/**
 * Create public client for reading blockchain state
 */
exports.publicClient = (0, viem_1.createPublicClient)({
    chain,
    transport: createTransport(config_1.config.blockchain.rpcUrl)
});
/**
 * Create bundler client for user operations
 */
exports.bundlerClient = getBundlerClient();
/**
 * Creates a bundler client based on configuration settings
 */
function getBundlerClient() {
    if (!config_1.config.blockchain.bundlerUrl) {
        throw new Error('Bundler URL is not configured');
    }
    const paymasterClient = (0, account_abstraction_1.createPaymasterClient)({
        transport: createTransport(config_1.config.blockchain.bundlerUrl)
    });
    const bundlerClient = (0, account_abstraction_1.createBundlerClient)({
        transport: createTransport(config_1.config.blockchain.bundlerUrl),
        chain,
        paymaster: paymasterClient,
    });
    return bundlerClient;
}
/**
 * Creates a MetaMask smart account from a private key
 *
 * @param privateKey - The private key to create the account from
 * @returns A MetaMask smart account instance
 */
const createMetaMaskAccount = async (privateKey) => {
    try {
        const formattedKey = (0, utils_1.formatPrivateKey)(privateKey);
        const account = (0, accounts_1.privateKeyToAccount)(formattedKey);
        utils_1.logger.info(`Creating MetaMask Smart Account for address: ${account.address}`);
        const smartAccount = await (0, delegation_toolkit_1.toMetaMaskSmartAccount)({
            client: exports.publicClient,
            implementation: delegation_toolkit_1.Implementation.Hybrid,
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
 *
 * @returns Gas fee parameters (maxFeePerGas and maxPriorityFeePerGas)
 */
const getFeePerGas = async () => {
    // The method for determining fee per gas is dependent on the bundler
    // implementation. For this reason, this is centralized here.
    const pimlicoClient = (0, pimlico_1.createPimlicoClient)({
        chain,
        transport: createTransport(config_1.config.blockchain.bundlerUrl),
    });
    const { fast } = await pimlicoClient.getUserOperationGasPrice();
    return fast;
};
exports.getFeePerGas = getFeePerGas;
/**
 * Redeems a delegation, executing actions on behalf of the delegator
 *
 * @param delegationData - The serialized delegation data
 * @param merchantAddress - The address of the merchant
 * @param tokenContractAddress - The address of the token contract
 * @param price - The price of the token
 * @returns The transaction hash of the redemption
 */
const redeemDelegation = async (delegationData, merchantAddress, tokenContractAddress, price) => {
    try {
        // Validate required parameters
        if (!delegationData || delegationData.length === 0) {
            throw new Error('Delegation data is required');
        }
        if (!merchantAddress || merchantAddress === '0x0000000000000000000000000000000000000000') {
            throw new Error('Valid merchant address is required');
        }
        if (!tokenContractAddress || tokenContractAddress === '0x0000000000000000000000000000000000000000') {
            throw new Error('Valid token contract address is required');
        }
        if (!price || price === '0') {
            throw new Error('Valid price is required');
        }
        // Parse the delegation data using our helper
        const delegation = (0, delegation_helpers_1.parseDelegation)(delegationData);
        // Validate the delegation
        (0, delegation_helpers_1.validateDelegation)(delegation);
        utils_1.logger.info("Redeeming delegation...");
        utils_1.logger.debug("Delegation details:", {
            delegate: delegation.delegate,
            delegator: delegation.delegator,
            merchantAddress,
            tokenContractAddress,
            price
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
        // Create ERC20 transfer calldata
        const transferCalldata = (0, viem_1.encodeFunctionData)({
            abi: erc20_1.erc20Abi, // ABI for the ERC20 contract
            functionName: 'transfer', // Name of the function to call
            args: [merchantAddress, (0, viem_1.parseUnits)(price, 6)] // Assuming USDC with 6 decimals
        });
        // The execution that will be performed on behalf of the delegator
        // target is the address of the merchant (the recipient of the ERC20 transfer)
        // value is 0 because we are not sending any ETH with the transaction
        // callData is the calldata for the ERC20 transfer
        const executions = [
            {
                target: tokenContractAddress, // Address of the ERC20 contract
                value: 0n, // No ETH value for ERC20 transfers
                callData: transferCalldata // Calldata for the ERC20 transfer
            }
        ];
        // Format the delegation for the framework
        const delegationForFramework = delegation;
        const delegationChain = [delegationForFramework];
        // Create the calldata for redeeming the delegation
        const redeemDelegationCalldata = delegation_toolkit_1.DelegationFramework.encode.redeemDelegations({
            delegations: [delegationChain],
            modes: [delegation_toolkit_1.SINGLE_DEFAULT_MODE],
            executions: [executions]
        });
        // The call to the delegation framework to redeem the delegation
        const calls = [
            {
                to: redeemer.address,
                data: redeemDelegationCalldata
            }
        ];
        // Get fee per gas for the transaction
        const feePerGas = await (0, exports.getFeePerGas)();
        utils_1.logger.info("Sending UserOperation...");
        // Start timer for overall transaction operation
        const overallStartTime = Date.now();
        // Properly type our account for the bundler client
        // Note: This assertion is necessary because the MetaMask smart account
        // implementation doesn't exactly match what the bundler expects
        const sendOpStartTime = Date.now();
        const userOperationHash = await exports.bundlerClient.sendUserOperation({
            account: redeemer,
            calls,
            ...feePerGas
        });
        const sendOpTime = (Date.now() - sendOpStartTime) / 1000;
        utils_1.logger.info(`UserOperation hash (sent in ${sendOpTime.toFixed(2)}s):`, userOperationHash);
        // Wait for the user operation to be included in a transaction
        utils_1.logger.info("Waiting for transaction receipt...");
        const receiptStartTime = Date.now();
        const receipt = await exports.bundlerClient.waitForUserOperationReceipt({
            hash: userOperationHash,
            timeout: 60000 // 60 second timeout
        });
        const receiptWaitTime = (Date.now() - receiptStartTime) / 1000;
        const transactionHash = receipt.receipt.transactionHash;
        // Calculate and log elapsed time
        const totalElapsedTimeSeconds = (Date.now() - overallStartTime) / 1000;
        utils_1.logger.info(`Transaction confirmed in ${totalElapsedTimeSeconds.toFixed(2)}s total (${sendOpTime.toFixed(2)}s to send, ${receiptWaitTime.toFixed(2)}s to confirm):`, transactionHash);
        return transactionHash;
    }
    catch (error) {
        utils_1.logger.error("Error redeeming delegation:", error);
        throw error;
    }
};
exports.redeemDelegation = redeemDelegation;
