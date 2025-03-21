/**
 * Unit tests for delegation helper functions
 * 
 * To run:
 *   npx jest test/delegation-helpers.test.ts
 */

import { parseDelegation, validateDelegation } from '../src/utils/delegation-helpers';
import { DelegationStruct } from '../src/types/delegation';
import { describe, expect, it } from '@jest/globals';

describe('Delegation Helper Functions', () => {
  describe('parseDelegation', () => {
    it('should parse valid JSON delegation data', () => {
      const mockDelegation = {
        delegator: '0x1234567890123456789012345678901234567890',
        delegate: '0x0987654321098765432109876543210987654321',
        authority: {
          scheme: '0x00',
          signature: '0xsig',
          signer: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
        },
        caveats: [],
        salt: '0x123456789',
        signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890'
      };
      
      const jsonData = JSON.stringify(mockDelegation);
      const buffer = Buffer.from(jsonData, 'utf-8');
      
      const result = parseDelegation(buffer);
      
      expect(result).toEqual(mockDelegation);
    });
    
    it('should throw error for invalid JSON', () => {
      const buffer = Buffer.from('{invalid:json}', 'utf-8');
      
      expect(() => parseDelegation(buffer)).toThrow('Failed to parse delegation');
    });
    
    it('should throw error for missing required fields', () => {
      const jsonData = JSON.stringify({});
      const buffer = Buffer.from(jsonData, 'utf-8');
      
      expect(() => parseDelegation(buffer)).toThrow(/Failed to parse delegation.*Binary format not supported/);
    });
  });
  
  describe('validateDelegation', () => {
    it('should validate a complete delegation object', () => {
      const mockDelegation: any = {
        delegator: '0x1234567890123456789012345678901234567890',
        delegate: '0x0987654321098765432109876543210987654321',
        authority: {
          scheme: '0x00',
          signature: '0xsig',
          signer: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
        },
        caveats: [],
        salt: '0x123456789',
        signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        scheme: '0x00',
        invocations: [],
        expiry: BigInt(0)
      };
      
      expect(validateDelegation(mockDelegation)).toBe(true);
    });
    
    it('should throw for missing delegator', () => {
      const mockDelegation = {
        delegate: '0x0987654321098765432109876543210987654321',
        signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
      } as unknown as DelegationStruct;
      
      expect(() => validateDelegation(mockDelegation)).toThrow('missing delegator');
    });
    
    it('should throw for missing delegate', () => {
      const mockDelegation = {
        delegator: '0x1234567890123456789012345678901234567890',
        signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
      } as unknown as DelegationStruct;
      
      expect(() => validateDelegation(mockDelegation)).toThrow('missing delegate');
    });
    
    it('should throw for missing signature', () => {
      const mockDelegation = {
        delegator: '0x1234567890123456789012345678901234567890',
        delegate: '0x0987654321098765432109876543210987654321',
      } as unknown as DelegationStruct;
      
      expect(() => validateDelegation(mockDelegation)).toThrow('missing signature');
    });
    
    it('should throw for expired delegation', () => {
      const yesterday = Math.floor(Date.now() / 1000) - 24 * 60 * 60;
      
      const mockDelegation: any = {
        delegator: '0x1234567890123456789012345678901234567890',
        delegate: '0x0987654321098765432109876543210987654321',
        authority: {
          scheme: '0x00',
          signature: '0xsig',
          signer: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789'
        },
        caveats: [],
        salt: '0x123456789',
        signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
        scheme: '0x00',
        invocations: [],
        expiry: BigInt(yesterday)
      };
      
      expect(() => validateDelegation(mockDelegation)).toThrow('Delegation is expired');
    });
  });
}); 