"use strict";
/**
 * Unit tests for the delegation service
 *
 * To run:
 *   npx jest test/service.test.ts
 */
Object.defineProperty(exports, "__esModule", { value: true });
const service_1 = require("../src/services/service");
const globals_1 = require("@jest/globals");
// Mock the blockchain service
globals_1.jest.mock('../src/services/blockchain', () => ({
    redeemDelegation: globals_1.jest.fn()
}));
// Import the mocked version
const blockchain_1 = require("../src/services/blockchain");
(0, globals_1.describe)('Delegation Service', () => {
    (0, globals_1.beforeEach)(() => {
        // Clear all mocks before each test
        globals_1.jest.clearAllMocks();
    });
    (0, globals_1.describe)('redeemDelegation', () => {
        (0, globals_1.it)('should successfully redeem a delegation', async () => {
            // Mock the successful transaction hash
            const mockTxHash = '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890';
            blockchain_1.redeemDelegation.mockResolvedValueOnce(mockTxHash);
            // Create mock request data
            const mockDelegationData = {
                delegator: '0x1234567890123456789012345678901234567890',
                delegate: '0x0987654321098765432109876543210987654321',
                caveats: [],
                salt: '0x123',
                signature: '0xsignature'
            };
            const delegationBuffer = Buffer.from(JSON.stringify(mockDelegationData));
            // Mock gRPC call parameters
            const mockCall = {
                request: {
                    delegationData: delegationBuffer
                }
            };
            // Create a mock callback
            const mockCallback = globals_1.jest.fn();
            // Call the service method
            await service_1.delegationService.redeemDelegation(mockCall, mockCallback);
            // Verify the blockchain service was called with correct data
            (0, globals_1.expect)(blockchain_1.redeemDelegation).toHaveBeenCalledWith(globals_1.expect.any(Uint8Array));
            // Verify the callback was called with the success response
            (0, globals_1.expect)(mockCallback).toHaveBeenCalledWith(null, {
                transactionHash: mockTxHash,
                success: true,
                errorMessage: ''
            });
        });
        (0, globals_1.it)('should handle empty delegation data', async () => {
            // Mock gRPC call with empty delegation data
            const mockCall = {
                request: {
                    delegationData: Buffer.from('')
                }
            };
            // Create a mock callback
            const mockCallback = globals_1.jest.fn();
            // Call the service method
            await service_1.delegationService.redeemDelegation(mockCall, mockCallback);
            // Verify the blockchain service was not called
            (0, globals_1.expect)(blockchain_1.redeemDelegation).not.toHaveBeenCalled();
            // Verify the callback was called with the error response
            (0, globals_1.expect)(mockCallback).toHaveBeenCalledWith(null, {
                transactionHash: '',
                success: false,
                errorMessage: globals_1.expect.stringContaining('Delegation data is empty')
            });
        });
        (0, globals_1.it)('should handle blockchain service errors', async () => {
            // Mock a failure in the blockchain service
            const mockError = new Error('Transaction failed');
            blockchain_1.redeemDelegation.mockRejectedValueOnce(mockError);
            // Create mock request data
            const mockDelegationData = {
                delegator: '0x1234567890123456789012345678901234567890',
                delegate: '0x0987654321098765432109876543210987654321',
                caveats: [],
                salt: '0x123',
                signature: '0xsignature'
            };
            const delegationBuffer = Buffer.from(JSON.stringify(mockDelegationData));
            // Mock gRPC call parameters
            const mockCall = {
                request: {
                    delegationData: delegationBuffer
                }
            };
            // Create a mock callback
            const mockCallback = globals_1.jest.fn();
            // Call the service method
            await service_1.delegationService.redeemDelegation(mockCall, mockCallback);
            // Verify the blockchain service was called
            (0, globals_1.expect)(blockchain_1.redeemDelegation).toHaveBeenCalled();
            // Verify the callback was called with the error response
            (0, globals_1.expect)(mockCallback).toHaveBeenCalledWith(null, {
                transactionHash: '',
                success: false,
                errorMessage: 'Transaction failed'
            });
        });
    });
});
