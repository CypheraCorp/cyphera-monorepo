/**
 * Integration tests for shared delegation library usage in delegation server
 * 
 * This test verifies that the delegation server properly uses the shared
 * delegation library while maintaining existing functionality.
 */

import { describe, expect, it, jest, beforeEach } from '@jest/globals';

// Mock the shared library before importing delegation helpers
jest.mock('@cyphera/delegation', () => ({
  isValidEthereumAddress: jest.fn(),
  parseDelegation: jest.fn(),
  validateDelegation: jest.fn(),
}));

// Mock the local utils logger
jest.mock('../src/utils/utils', () => ({
  logger: {
    info: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
    warn: jest.fn(),
  },
}));

// Import after mocking
import { 
  isValidEthereumAddress, 
  parseDelegation, 
  validateDelegation 
} from '../src/utils/delegation-helpers';

// Import the mocked functions for assertions
import * as sharedLibMocks from '@cyphera/delegation';

describe('Shared Library Integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('isValidEthereumAddress', () => {
    it('should delegate to shared library implementation', () => {
      const testAddress = '0x1234567890123456789012345678901234567890';
      (sharedLibMocks.isValidEthereumAddress as jest.Mock).mockReturnValue(true);
      
      const result = isValidEthereumAddress(testAddress);
      
      expect(sharedLibMocks.isValidEthereumAddress).toHaveBeenCalledWith(testAddress);
      expect(result).toBe(true);
    });

    it('should handle invalid addresses through shared library', () => {
      const invalidAddress = 'invalid-address';
      (sharedLibMocks.isValidEthereumAddress as jest.Mock).mockReturnValue(false);
      
      const result = isValidEthereumAddress(invalidAddress);
      
      expect(sharedLibMocks.isValidEthereumAddress).toHaveBeenCalledWith(invalidAddress);
      expect(result).toBe(false);
    });
  });

  describe('parseDelegation', () => {
    it('should delegate to shared library implementation', () => {
      const testData = Buffer.from('{"delegator":"0x123","delegate":"0x456"}');
      const expectedResult = { delegator: '0x123', delegate: '0x456' };
      (sharedLibMocks.parseDelegation as jest.Mock).mockReturnValue(expectedResult);
      
      const result = parseDelegation(testData);
      
      expect(sharedLibMocks.parseDelegation).toHaveBeenCalledWith(testData);
      expect(result).toEqual(expectedResult);
    });

    it('should propagate errors from shared library', () => {
      const testData = Buffer.from('invalid-json');
      const error = new Error('Failed to parse delegation');
      (sharedLibMocks.parseDelegation as jest.Mock).mockImplementation(() => {
        throw error;
      });
      
      expect(() => parseDelegation(testData)).toThrow('Failed to parse delegation');
      expect(sharedLibMocks.parseDelegation).toHaveBeenCalledWith(testData);
    });
  });

  describe('validateDelegation', () => {
    it('should delegate to shared library implementation', async () => {
      const testDelegation = { 
        delegator: '0x123', 
        delegate: '0x456', 
        signature: '0xabc' 
      };
      const mockPublicClient = { getBytecode: jest.fn() };
      (sharedLibMocks.validateDelegation as any).mockResolvedValue(true);
      
      const result = await validateDelegation(testDelegation as any, mockPublicClient as any);
      
      expect(sharedLibMocks.validateDelegation).toHaveBeenCalledWith(testDelegation, mockPublicClient);
      expect(result).toBe(true);
    });

    it('should propagate validation errors from shared library', async () => {
      const testDelegation = { 
        delegator: '', // invalid
        delegate: '0x456', 
        signature: '0xabc' 
      };
      const mockPublicClient = { getBytecode: jest.fn() };
      const error = new Error('Invalid delegation: missing delegator');
      (sharedLibMocks.validateDelegation as any).mockRejectedValue(error);
      
      await expect(validateDelegation(testDelegation as any, mockPublicClient as any))
        .rejects.toThrow('Invalid delegation: missing delegator');
      
      expect(sharedLibMocks.validateDelegation).toHaveBeenCalledWith(testDelegation, mockPublicClient);
    });
  });

  describe('Library Consistency', () => {
    it('should use the same validation logic as the shared library', () => {
      // Test that we're truly using the shared library and not duplicate logic
      const testCases = [
        '0x1234567890123456789012345678901234567890', // valid
        'invalid-address', // invalid
        '', // empty
        '0x123', // too short
      ];

      testCases.forEach((address, index) => {
        (sharedLibMocks.isValidEthereumAddress as jest.Mock)
          .mockReturnValueOnce(address.startsWith('0x') && address.length === 42);
        
        isValidEthereumAddress(address);
      });

      expect(sharedLibMocks.isValidEthereumAddress).toHaveBeenCalledTimes(testCases.length);
    });
  });

  describe('Backward Compatibility', () => {
    it('should maintain the same API as before refactoring', () => {
      // Ensure that the delegation-helpers still export the same functions
      // with the same signatures as before the refactoring
      
      expect(typeof isValidEthereumAddress).toBe('function');
      expect(typeof parseDelegation).toBe('function');
      expect(typeof validateDelegation).toBe('function');
      
      // These functions should have the same number of parameters as before
      expect(isValidEthereumAddress.length).toBe(1); // 1 parameter: address
      expect(parseDelegation.length).toBe(1); // 1 parameter: delegationData
      expect(validateDelegation.length).toBe(2); // 2 parameters: delegation, publicClient
    });

    it('should work with existing delegation server code', () => {
      // Mock a realistic scenario from the delegation server
      const mockDelegationData = Buffer.from(JSON.stringify({
        delegator: '0x1234567890123456789012345678901234567890',
        delegate: '0xabcdefABCDEF1234567890123456789012345678',
        signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        caveats: [],
        salt: '0x123456789',
        authority: {
          scheme: '0x00',
          signature: '0xsig',
          signer: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
        }
      }));

      const expectedParsedDelegation = {
        delegator: '0x1234567890123456789012345678901234567890',
        delegate: '0xabcdefABCDEF1234567890123456789012345678',
        signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890'
      };

      (sharedLibMocks.parseDelegation as jest.Mock).mockReturnValue(expectedParsedDelegation);
      (sharedLibMocks.isValidEthereumAddress as jest.Mock).mockReturnValue(true);

      // This should work exactly as it did before the refactoring
      const parsedDelegation = parseDelegation(mockDelegationData);
      const isDelegatorValid = isValidEthereumAddress(parsedDelegation.delegator);
      const isDelegateValid = isValidEthereumAddress(parsedDelegation.delegate);

      expect(parsedDelegation).toEqual(expectedParsedDelegation);
      expect(isDelegatorValid).toBe(true);
      expect(isDelegateValid).toBe(true);
    });
  });
});