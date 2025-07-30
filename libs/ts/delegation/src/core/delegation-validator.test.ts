/**
 * Tests for delegation validator functions
 */

import { describe, expect, it, jest } from '@jest/globals';
import { isValidEthereumAddress, validateDelegationStructure } from './delegation-validator';
import type { Delegation } from '@metamask/delegation-toolkit';

describe('Delegation Validator', () => {
  describe('isValidEthereumAddress', () => {
    it('should return true for valid addresses', () => {
      const validAddresses = [
        '0x1234567890123456789012345678901234567890',
        '0xabcdefABCDEF1234567890123456789012345678',
        '0x0000000000000000000000000000000000000000'
      ];

      validAddresses.forEach(address => {
        expect(isValidEthereumAddress(address)).toBe(true);
      });
    });

    it('should return false for invalid addresses', () => {
      const invalidAddresses = [
        '', // empty
        '1234567890123456789012345678901234567890', // no 0x prefix  
        '0x123456789012345678901234567890123456789', // too short
        '0x12345678901234567890123456789012345678901', // too long
        '0x123456789012345678901234567890123456789g', // invalid hex char
        null, // null
        undefined // undefined
      ];

      invalidAddresses.forEach(address => {
        expect(isValidEthereumAddress(address as any)).toBe(false);
      });
    });
  });

  describe('validateDelegationStructure', () => {
    const validDelegation: Delegation = {
      delegator: '0x1234567890123456789012345678901234567890',
      delegate: '0xabcdefABCDEF1234567890123456789012345678',
      signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
      // Add other required fields based on MetaMask delegation toolkit
    } as any;

    it('should validate a correct delegation structure', () => {
      const result = validateDelegationStructure(validDelegation);
      expect(result.isValid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('should detect missing delegator', () => {
      const delegation = { ...validDelegation };
      delete (delegation as any).delegator;
      
      const result = validateDelegationStructure(delegation);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Missing delegator');
    });

    it('should detect missing delegate', () => {
      const delegation = { ...validDelegation };
      delete (delegation as any).delegate;
      
      const result = validateDelegationStructure(delegation);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Missing delegate');
    });

    it('should detect missing signature', () => {
      const delegation = { ...validDelegation };
      delete (delegation as any).signature;
      
      const result = validateDelegationStructure(delegation);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Missing signature');
    });

    it('should detect invalid delegator address', () => {
      const delegation = {
        ...validDelegation,
        delegator: 'invalid-address' as any
      };
      
      const result = validateDelegationStructure(delegation as any);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Invalid delegator address format');
    });

    it('should detect invalid delegate address', () => {
      const delegation = {
        ...validDelegation,
        delegate: 'invalid-address' as any
      };
      
      const result = validateDelegationStructure(delegation as any);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Invalid delegate address format');
    });

    it('should collect multiple errors', () => {
      const delegation = {
        delegator: 'invalid-delegator' as any,
        delegate: 'invalid-delegate' as any,
        // missing signature
      };
      
      const result = validateDelegationStructure(delegation as any);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Missing signature');
      expect(result.errors).toContain('Invalid delegator address format');
      expect(result.errors).toContain('Invalid delegate address format');
    });
  });
});