/**
 * Unit tests for the delegation service
 * 
 * To run:
 *   npx jest test/service.test.ts
 */

import { delegationService } from '../src/services/service';
import { describe, expect, it, jest, beforeEach } from '@jest/globals';

// Mock the blockchain service
jest.mock('../src/services/blockchain', () => ({
  redeemDelegation: jest.fn()
}));

// Import the mocked version
import { redeemDelegation } from '../src/services/blockchain';

describe('Delegation Service', () => {
  beforeEach(() => {
    // Clear all mocks before each test
    jest.clearAllMocks();
  });

  describe('redeemDelegation', () => {
    it('should successfully redeem a delegation', async () => {
      // Mock the successful transaction hash
      const mockTxHash = '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890';
      (redeemDelegation as jest.MockedFunction<typeof redeemDelegation>).mockResolvedValueOnce(mockTxHash);
      
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
      const mockCallback = jest.fn();
      
      // Call the service method
      await delegationService.redeemDelegation(mockCall, mockCallback);
      
      // Verify the blockchain service was called with correct data
      expect(redeemDelegation).toHaveBeenCalledWith(expect.any(Uint8Array));
      
      // Verify the callback was called with the success response
      expect(mockCallback).toHaveBeenCalledWith(null, {
        transactionHash: mockTxHash,
        success: true,
        errorMessage: ''
      });
    });
    
    it('should handle empty delegation data', async () => {
      // Mock gRPC call with empty delegation data
      const mockCall = {
        request: {
          delegationData: Buffer.from('')
        }
      };
      
      // Create a mock callback
      const mockCallback = jest.fn();
      
      // Call the service method
      await delegationService.redeemDelegation(mockCall, mockCallback);
      
      // Verify the blockchain service was not called
      expect(redeemDelegation).not.toHaveBeenCalled();
      
      // Verify the callback was called with the error response
      expect(mockCallback).toHaveBeenCalledWith(null, {
        transactionHash: '',
        success: false,
        errorMessage: expect.stringContaining('Delegation data is empty')
      });
    });
    
    it('should handle blockchain service errors', async () => {
      // Mock a failure in the blockchain service
      const mockError = new Error('Transaction failed');
      (redeemDelegation as jest.MockedFunction<typeof redeemDelegation>).mockRejectedValueOnce(mockError);
      
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
      const mockCallback = jest.fn();
      
      // Call the service method
      await delegationService.redeemDelegation(mockCall, mockCallback);
      
      // Verify the blockchain service was called
      expect(redeemDelegation).toHaveBeenCalled();
      
      // Verify the callback was called with the error response
      expect(mockCallback).toHaveBeenCalledWith(null, {
        transactionHash: '',
        success: false,
        errorMessage: 'Transaction failed'
      });
    });
  });
}); 