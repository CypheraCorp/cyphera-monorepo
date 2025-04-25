"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const redeem_delegation_1 = require("../src/services/redeem-delegation");
const config_1 = require("../src/config/config");
const utils_1 = require("../src/utils/utils");
// --- Comprehensive Mocking ---
// Define mock client objects first
/* // Removing external mock objects
const mockBundlerClient = {
  sendUserOperation: jest.fn(),
  waitForUserOperationReceipt: jest.fn(),
};
const mockPaymasterClient = {};
const mockPimlicoClient = {
  getUserOperationGasPrice: jest.fn(),
};
*/
// Mock configuration
jest.mock('../src/config/config', () => ({
    config: {
        blockchain: {
            rpcUrl: 'mock-rpc-url',
            bundlerUrl: 'mock-bundler-url',
            privateKey: '0x' + '1'.repeat(64), // Mock private key
        },
    },
}));
// Mock logger
jest.mock('../src/utils/utils', () => ({
    logger: {
        info: jest.fn(),
        error: jest.fn(),
        debug: jest.fn(),
    },
    formatPrivateKey: jest.fn((key) => key), // Simple pass-through mock
}));
// Mock delegation helpers
jest.mock('../src/utils/delegation-helpers', () => ({
    parseDelegation: jest.fn(),
    validateDelegation: jest.fn(),
}));
// Mock viem core functions
jest.mock('viem', () => ({
    ...jest.requireActual('viem'), // Keep actual functions we might need indirectly
    createPublicClient: jest.fn(() => ({ mock: 'publicClient' })), // Return a simple mock object
    http: jest.fn(),
    encodeFunctionData: jest.fn(),
    isAddressEqual: jest.requireActual('viem').isAddressEqual, // Use actual implementation
    parseUnits: jest.fn(),
}));
// Mock viem/accounts
jest.mock('viem/accounts', () => ({
    privateKeyToAccount: jest.fn(),
}));
// Mock viem/chains
// We don't need to mock this directly as sepolia is just an object
jest.mock('@metamask/delegation-toolkit', () => ({
    toMetaMaskSmartAccount: jest.fn(),
    DelegationFramework: {
        encode: {
            redeemDelegations: jest.fn(),
        },
    },
    SINGLE_DEFAULT_MODE: 'mock-single-default-mode',
    Implementation: {
        Hybrid: 'mock-hybrid-implementation',
    },
    // Use actual isAddressEqual for comparison logic
    isAddressEqual: jest.requireActual('viem').isAddressEqual,
}));
// Mock viem/account-abstraction with inline mocks
jest.mock('viem/account-abstraction', () => ({
    createBundlerClient: jest.fn().mockReturnValue({
        sendUserOperation: jest.fn(),
        waitForUserOperationReceipt: jest.fn(),
    }),
    createPaymasterClient: jest.fn().mockReturnValue({}), // Define mock directly
}));
// Mock permissionless/clients/pimlico with inline mock
jest.mock('permissionless/clients/pimlico', () => ({
    createPimlicoClient: jest.fn().mockReturnValue({
        getUserOperationGasPrice: jest.fn(),
    }),
}));
// --- Test Suite ---
describe('redeem-delegation service', () => {
    // Access mocked functions after jest.mock has run
    let mockSendUserOperation;
    let mockWaitForUserOperationReceipt;
    let mockGetUserOperationGasPrice;
    let mockCreatePimlicoClient;
    let mockCreateBundlerClient;
    // Mock data defaults (Using valid example addresses)
    const mockDelegationData = Buffer.from('mock-delegation-data');
    // Use valid checksummed addresses as mocks
    const mockMerchantAddress = '0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045'; // Vitalik's address
    const mockTokenContractAddress = '0x742d35Cc6634C0532925a3b844Bc454e4438f44e'; // Example Token
    const mockPrice = '100';
    const mockTxHash = '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd';
    const mockUserOpHash = '0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd';
    const mockRedeemerAddress = '0xBE0eB53F46cd790Cd13851d5EFf43D12404d33E8'; // Another example address
    const mockDelegatorAddress = '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266'; // Anvil default
    const mockDelegateAddress = mockRedeemerAddress; // Should match redeemer for happy path
    const mockSmartAccountAddress = mockRedeemerAddress; // Smart account derived for redeemer
    const mockEOA = { address: mockRedeemerAddress, signMessage: jest.fn(), signTransaction: jest.fn(), signTypedData: jest.fn() };
    const mockSmartAccount = { address: mockSmartAccountAddress };
    const mockParsedDelegation = { delegate: mockDelegateAddress, delegator: mockDelegatorAddress };
    const mockGasPrice = { fast: { maxFeePerGas: 100n, maxPriorityFeePerGas: 10n } };
    const mockReceipt = { receipt: { transactionHash: mockTxHash } };
    const mockEncodedTransfer = '0xEncodedTransfer';
    const mockRedeemCalldata = '0xRedeemCalldata';
    beforeEach(() => {
        // Reset all mocks
        jest.clearAllMocks();
        // Re-require mocked modules
        const accountAbstractionMocks = require('viem/account-abstraction');
        const pimlicoMocks = require('permissionless/clients/pimlico');
        const coreViemMocks = require('@metamask/delegation-toolkit');
        const viemAccountMocks = require('viem/accounts');
        const viemMocks = require('viem');
        const delegationHelperMocks = require('../src/utils/delegation-helpers');
        // Assign mock functions
        mockCreateBundlerClient = accountAbstractionMocks.createBundlerClient;
        mockSendUserOperation = mockCreateBundlerClient().sendUserOperation;
        mockWaitForUserOperationReceipt = mockCreateBundlerClient().waitForUserOperationReceipt;
        mockCreatePimlicoClient = pimlicoMocks.createPimlicoClient;
        mockGetUserOperationGasPrice = mockCreatePimlicoClient().getUserOperationGasPrice;
        // Setup default happy path mocks
        delegationHelperMocks.parseDelegation.mockReturnValue(mockParsedDelegation);
        delegationHelperMocks.validateDelegation.mockImplementation(() => { });
        viemAccountMocks.privateKeyToAccount.mockReturnValue(mockEOA);
        coreViemMocks.toMetaMaskSmartAccount.mockResolvedValue(mockSmartAccount);
        mockGetUserOperationGasPrice.mockResolvedValue(mockGasPrice);
        viemMocks.encodeFunctionData.mockReturnValue(mockEncodedTransfer);
        coreViemMocks.DelegationFramework.encode.redeemDelegations.mockReturnValue(mockRedeemCalldata);
        mockSendUserOperation.mockResolvedValue(mockUserOpHash);
        mockWaitForUserOperationReceipt.mockResolvedValue(mockReceipt);
        viemMocks.parseUnits.mockReturnValue(BigInt(mockPrice + '000000'));
    });
    // --- redeemDelegation Tests ---
    describe('redeemDelegation input validation', () => {
        it('should throw if delegationData is missing', async () => {
            await expect((0, redeem_delegation_1.redeemDelegation)(null, mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow('Delegation data is required');
            await expect((0, redeem_delegation_1.redeemDelegation)(Buffer.from(''), mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow('Delegation data is required');
        });
        it('should throw if merchantAddress is missing or zero', async () => {
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, '', mockTokenContractAddress, mockPrice))
                .rejects.toThrow('Valid merchant address is required');
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, '0x0000000000000000000000000000000000000000', mockTokenContractAddress, mockPrice))
                .rejects.toThrow('Valid merchant address is required');
        });
        it('should throw if tokenContractAddress is missing or zero', async () => {
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, '', mockPrice))
                .rejects.toThrow('Valid token contract address is required');
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, '0x0000000000000000000000000000000000000000', mockPrice))
                .rejects.toThrow('Valid token contract address is required');
        });
        it('should throw if price is missing or zero', async () => {
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, ''))
                .rejects.toThrow('Valid price is required');
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, '0'))
                .rejects.toThrow('Valid price is required');
        });
        it('should throw if private key is not configured', async () => {
            const originalKey = config_1.config.blockchain.privateKey;
            config_1.config.blockchain.privateKey = ''; // Temporarily unset the key
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow('Private key is not configured');
            config_1.config.blockchain.privateKey = originalKey; // Restore key
        });
        it('should throw if redeemer address does not match delegate', async () => {
            const { toMetaMaskSmartAccount: mockToMMA } = require('@metamask/delegation-toolkit');
            // Use a different valid address for the delegate in the parsed delegation
            const differentDelegate = '0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B'; // Another example
            const parsedMismatchDelegation = { ...mockParsedDelegation, delegate: differentDelegate };
            require('../src/utils/delegation-helpers').parseDelegation.mockReturnValue(parsedMismatchDelegation);
            mockToMMA.mockResolvedValue(mockSmartAccount); // Ensure account is created (with mockSmartAccountAddress)
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow(`Redeemer account address does not match delegate in delegation. Redeemer: ${mockSmartAccountAddress}, delegate: ${differentDelegate}`);
        });
    }); // end input validation describe
    describe('redeemDelegation happy path', () => {
        it('should successfully redeem a valid delegation', async () => {
            // Mocks are set up in beforeEach
            const { toMetaMaskSmartAccount: mockToMMA, DelegationFramework: mockDF } = require('@metamask/delegation-toolkit');
            const { parseDelegation: mockParse, validateDelegation: mockValidate } = require('../src/utils/delegation-helpers');
            const { privateKeyToAccount: mockPKToAccount } = require('viem/accounts');
            const { encodeFunctionData: mockEncode, parseUnits: mockParseUnits, isAddressEqual } = require('viem');
            const result = await (0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, mockPrice);
            // Assertions
            expect(mockParse).toHaveBeenCalledWith(mockDelegationData);
            expect(mockValidate).toHaveBeenCalledWith(mockParsedDelegation);
            expect(mockPKToAccount).toHaveBeenCalledWith(config_1.config.blockchain.privateKey);
            expect(mockToMMA).toHaveBeenCalled();
            expect(mockEncode).toHaveBeenCalledWith({
                abi: expect.any(Array),
                functionName: 'transfer',
                args: [mockMerchantAddress, BigInt(mockPrice + '000000')]
            });
            expect(mockParseUnits).toHaveBeenCalledWith(mockPrice, 6);
            expect(mockDF.encode.redeemDelegations).toHaveBeenCalledWith({
                delegations: [[mockParsedDelegation]],
                modes: ['mock-single-default-mode'],
                executions: [[{ target: mockTokenContractAddress, value: 0n, callData: mockEncodedTransfer }]]
            });
            expect(mockGetUserOperationGasPrice).toHaveBeenCalled();
            expect(mockSendUserOperation).toHaveBeenCalledWith({
                account: mockSmartAccount,
                calls: [{ to: mockSmartAccountAddress, data: mockRedeemCalldata }],
                maxFeePerGas: mockGasPrice.fast.maxFeePerGas,
                maxPriorityFeePerGas: mockGasPrice.fast.maxPriorityFeePerGas
            });
            expect(mockWaitForUserOperationReceipt).toHaveBeenCalledWith({
                hash: mockUserOpHash,
                timeout: 60000
            });
            expect(result).toBe(mockTxHash);
            // Check that logger.info was called, and the last call contained "Transaction confirmed"
            expect(utils_1.logger.info).toHaveBeenCalled();
            const lastLoggerInfoCallArgs = utils_1.logger.info.mock.calls[utils_1.logger.info.mock.calls.length - 1];
            expect(lastLoggerInfoCallArgs[0]).toContain('Transaction confirmed');
            // Optionally check if the hash is also logged
            expect(lastLoggerInfoCallArgs[1]).toBe(mockTxHash);
        });
    }); // end happy path describe
    describe('redeemDelegation error paths', () => {
        it('should throw if sendUserOperation fails', async () => {
            const sendError = new Error('Bundler connection failed');
            mockSendUserOperation.mockRejectedValue(sendError);
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow(sendError);
            expect(utils_1.logger.error).toHaveBeenCalledWith("Error redeeming delegation:", sendError);
        });
        it('should throw if waitForUserOperationReceipt fails', async () => {
            const waitError = new Error('Receipt timeout');
            mockWaitForUserOperationReceipt.mockRejectedValue(waitError);
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow(waitError);
            expect(utils_1.logger.error).toHaveBeenCalledWith("Error redeeming delegation:", waitError);
        });
        it('should throw if createMetaMaskAccount (toMetaMaskSmartAccount) fails', async () => {
            const accountError = new Error('Failed to derive account');
            const { toMetaMaskSmartAccount: mockToMMA } = require('@metamask/delegation-toolkit');
            mockToMMA.mockRejectedValue(accountError);
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow(accountError);
            // Error is logged within createMetaMaskAccount mock/original if called,
            // and then re-logged by redeemDelegation's catch block
            expect(utils_1.logger.error).toHaveBeenCalledWith("Error redeeming delegation:", accountError);
        });
        it('should throw if getFeePerGas fails', async () => {
            const feeError = new Error('Could not fetch gas price');
            mockGetUserOperationGasPrice.mockRejectedValue(feeError);
            await expect((0, redeem_delegation_1.redeemDelegation)(mockDelegationData, mockMerchantAddress, mockTokenContractAddress, mockPrice))
                .rejects.toThrow(feeError);
            expect(utils_1.logger.error).toHaveBeenCalledWith("Error redeeming delegation:", feeError);
        });
    }); // end error paths describe
    // --- TODO: Add tests for createMetaMaskAccount, getFeePerGas, getBundlerClient ---
    // These would involve mocking their specific dependencies (e.g., createPublicClient, Pimlico client methods)
});
