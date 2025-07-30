import { describe, expect, it, jest } from '@jest/globals';
import {
  prepareTokenAmount,
  encodeERC20Transfer,
  buildExecutionStruct,
  encodeDelegationRedemption,
  prepareRedemptionUserOperationPayload,
  prepareBatchRedemptionPayload
} from './payload-builder';
import { RedemptionErrorType } from './types';
import type { Address, Hex } from 'viem';
import type { Delegation } from '@metamask/delegation-toolkit';

describe('Payload Builder', () => {
  const mockDelegation: Delegation = {
    delegate: '0x1234567890123456789012345678901234567890' as Address,
    delegator: '0xabcdefabcdefabcdefabcdefabcdefabcdefabcd' as Address,
    authority: '0xdeadbeef' as Hex,
    caveats: [],
    salt: '0x0000000000000000000000000000000000000000000000000000000000000000' as Hex,
    signature: '0xsignature' as Hex
  };

  describe('prepareTokenAmount', () => {
    it('should convert number to bigint with correct decimals', () => {
      // 1 USDC (6 decimals)
      expect(prepareTokenAmount(1000000, 6)).toBe(1000000n);
      
      // 1.5 USDC (6 decimals) = 1500000
      expect(prepareTokenAmount(1500000, 6)).toBe(1500000n);
    });

    it('should return bigint as-is', () => {
      const amount = BigInt('1000000000000000000');
      expect(prepareTokenAmount(amount, 18)).toBe(amount);
    });

    it('should throw error for invalid decimals', () => {
      expect(() => prepareTokenAmount(1000000, 0))
        .toThrow('Token decimals must be positive for parsing');
      
      expect(() => prepareTokenAmount(1000000, -1))
        .toThrow('Token decimals must be positive for parsing');
    });
  });

  describe('encodeERC20Transfer', () => {
    it('should encode ERC20 transfer function call', () => {
      const recipient = '0x1234567890123456789012345678901234567890' as Address;
      const amount = 1000000n;
      
      const encoded = encodeERC20Transfer(recipient, amount);
      
      expect(encoded).toMatch(/^0x[a-f0-9]+$/);
      // Should start with transfer function selector (0xa9059cbb)
      expect(encoded.slice(0, 10)).toBe('0xa9059cbb');
    });
  });

  describe('buildExecutionStruct', () => {
    it('should build execution struct with default value', () => {
      const target = '0x1234567890123456789012345678901234567890' as Address;
      const callData = '0xabcdef' as Hex;
      
      const execution = buildExecutionStruct(target, callData);
      
      expect(execution).toEqual({
        target,
        value: 0n,
        callData
      });
    });

    it('should build execution struct with custom value', () => {
      const target = '0x1234567890123456789012345678901234567890' as Address;
      const callData = '0xabcdef' as Hex;
      const value = 1000000n;
      
      const execution = buildExecutionStruct(target, callData, value);
      
      expect(execution).toEqual({
        target,
        value,
        callData
      });
    });
  });

  describe('prepareRedemptionUserOperationPayload', () => {
    it('should prepare complete redemption payload', () => {
      const merchantAddress = '0x9876543210987654321098765432109876543210';
      const tokenContractAddress = '0xfedcbafedcbafedcbafedcbafedcbafedcbafedc';
      const tokenAmount = 1000000; // 1 USDC
      const tokenDecimals = 6;
      const redeemerAddress = '0x1111111111111111111111111111111111111111' as Address;
      
      const calls = prepareRedemptionUserOperationPayload(
        mockDelegation,
        merchantAddress,
        tokenContractAddress,
        tokenAmount,
        tokenDecimals,
        redeemerAddress
      );
      
      expect(calls).toHaveLength(1);
      expect(calls[0].to).toBe(redeemerAddress);
      expect(calls[0].data).toMatch(/^0x[a-f0-9]+$/);
    });

    it('should handle bigint token amounts', () => {
      const calls = prepareRedemptionUserOperationPayload(
        mockDelegation,
        '0x9876543210987654321098765432109876543210',
        '0xfedcbafedcbafedcbafedcbafedcbafedcbafedc',
        BigInt('1000000000000000000'), // 1 ETH in wei
        18,
        '0x1111111111111111111111111111111111111111' as Address
      );
      
      expect(calls).toHaveLength(1);
      expect(calls[0].data).toBeDefined();
    });

    it('should throw error on payload preparation failure', () => {
      // Pass invalid parameters to trigger error
      expect(() => prepareRedemptionUserOperationPayload(
        mockDelegation,
        '0x9876543210987654321098765432109876543210',
        '0xfedcbafedcbafedcbafedcbafedcbafedcbafedc',
        1000000,
        -1, // Invalid decimals
        '0x1111111111111111111111111111111111111111' as Address
      )).toThrow('Failed to prepare redemption payload');
    });
  });

  describe('prepareBatchRedemptionPayload', () => {
    it('should prepare batch redemption payload', () => {
      const redemptions = [
        {
          delegation: mockDelegation,
          merchantAddress: '0x9876543210987654321098765432109876543210' as Address,
          tokenContractAddress: '0xfedcbafedcbafedcbafedcbafedcbafedcbafedc' as Address,
          tokenAmount: 1000000n,
          tokenDecimals: 6
        },
        {
          delegation: { ...mockDelegation, salt: '0x0000000000000000000000000000000000000000000000000000000000000001' as Hex },
          merchantAddress: '0x1234567890123456789012345678901234567890' as Address,
          tokenContractAddress: '0xabcdefabcdefabcdefabcdefabcdefabcdefabcd' as Address,
          tokenAmount: BigInt('1000000000000000000'),
          tokenDecimals: 18
        }
      ];
      
      const redeemerAddress = '0x1111111111111111111111111111111111111111' as Address;
      
      const calls = prepareBatchRedemptionPayload(redemptions, redeemerAddress);
      
      expect(calls).toHaveLength(1);
      expect(calls[0].to).toBe(redeemerAddress);
      expect(calls[0].data).toMatch(/^0x[a-f0-9]+$/);
    });

    it('should handle empty batch', () => {
      const calls = prepareBatchRedemptionPayload(
        [],
        '0x1111111111111111111111111111111111111111' as Address
      );
      
      expect(calls).toHaveLength(1);
      expect(calls[0].data).toBeDefined();
    });
  });
});