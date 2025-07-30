import { describe, expect, it, jest } from '@jest/globals';
import {
  validateRedemptionInputs,
  validateTokenAmount,
  validateAddress
} from './validation';
import { RedemptionParams, RedemptionErrorType } from './types';
import type { Address } from 'viem';

describe('Redemption Validation', () => {
  const validParams: RedemptionParams = {
    delegationData: new Uint8Array([1, 2, 3]),
    merchantAddress: '0x1234567890123456789012345678901234567890' as Address,
    tokenContractAddress: '0xabcdefabcdefabcdefabcdefabcdefabcdefabcd' as Address,
    tokenAmount: 1000000,
    tokenDecimals: 6,
    chainId: 1,
    networkName: 'Ethereum Mainnet'
  };

  describe('validateRedemptionInputs', () => {
    it('should pass validation with valid inputs', () => {
      expect(() => validateRedemptionInputs(validParams)).not.toThrow();
    });

    it('should throw error for empty delegation data', () => {
      const params = { ...validParams, delegationData: new Uint8Array() };
      expect(() => validateRedemptionInputs(params))
        .toThrow('Delegation data cannot be empty');
    });

    it('should throw error for invalid merchant address', () => {
      const params = { ...validParams, merchantAddress: 'invalid' as Address };
      expect(() => validateRedemptionInputs(params))
        .toThrow('Invalid merchant address');
    });

    it('should throw error for invalid chain ID', () => {
      const params = { ...validParams, chainId: 0 };
      expect(() => validateRedemptionInputs(params))
        .toThrow('Invalid chain ID');
    });
  });

  describe('validateTokenAmount', () => {
    it('should pass validation for valid token amounts', () => {
      expect(() => validateTokenAmount(1000000)).not.toThrow();
      expect(() => validateTokenAmount(BigInt('1000000000000000000'))).not.toThrow();
    });

    it('should throw error for zero amount', () => {
      expect(() => validateTokenAmount(0))
        .toThrow('Token amount must be greater than 0');
    });

    it('should throw error for negative amount', () => {
      expect(() => validateTokenAmount(-1000))
        .toThrow('Token amount must be greater than 0');
    });
  });

  describe('validateAddress', () => {
    it('should pass validation for valid Ethereum addresses', () => {
      expect(() => validateAddress(
        '0x1234567890123456789012345678901234567890',
        'merchant'
      )).not.toThrow();
    });

    it('should throw error for invalid address format', () => {
      expect(() => validateAddress(
        '0xinvalid',
        'merchant'
      )).toThrow('Invalid merchant address format');
    });

    it('should throw error for empty address', () => {
      expect(() => validateAddress(
        '',
        'token'
      )).toThrow('Valid token address is required');
    });

    it('should throw error for zero address', () => {
      expect(() => validateAddress(
        '0x0000000000000000000000000000000000000000',
        'merchant'
      )).toThrow('Valid merchant address is required');
    });
  });

});