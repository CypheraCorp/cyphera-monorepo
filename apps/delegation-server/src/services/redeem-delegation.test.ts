import { type Address } from "viem";
import { redeemDelegation } from './redeem-delegation';
import { getNetworkConfig } from '../config/config';
import { logger } from '../utils/utils';
import { getSecretValue } from '../utils/secrets_manager';
import * as delegationLib from '@cyphera/delegation';

// Mock all external dependencies
jest.mock('../config/config');
jest.mock('../utils/utils', () => ({
  logger: {
    info: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
    warn: jest.fn()
  }
}));
jest.mock('../utils/secrets_manager');

// Mock the delegation library
jest.mock('@cyphera/delegation', () => ({
  parseDelegation: jest.fn(),
  validateDelegation: jest.fn(),
  validateRedemptionInputs: jest.fn(),
  getChainById: jest.fn(),
  initializeBlockchainClients: jest.fn(),
  createNetworkConfigFromUrls: jest.fn(),
  getOrCreateRedeemerAccount: jest.fn(),
  validateDelegateMatch: jest.fn(),
  prepareRedemptionUserOperationPayload: jest.fn(),
  sendAndConfirmUserOperation: jest.fn()
}));

describe('redeemDelegation', () => {
  const mockDelegationData = Buffer.from('test-delegation-data');
  const mockMerchantAddress = '0x1234567890123456789012345678901234567890';
  const mockTokenContractAddress = '0x0987654321098765432109876543210987654321';
  const mockTokenAmount = 1000000;
  const mockTokenDecimals = 6;
  const mockChainId = 1;
  const mockNetworkName = 'mainnet';
  const mockTransactionHash = '0xabc123def456789';
  const mockRedeemerAddress = '0xredeemer1234567890123456789012345678901234' as Address;
  const mockDelegatorAddress = '0xdelegator234567890123456789012345678901234' as Address;

  const mockDelegation = {
    delegate: mockRedeemerAddress,
    delegator: mockDelegatorAddress,
    authority: '0x',
    caveats: [],
    salt: '0x',
    signature: '0x'
  };

  const mockChain = { id: mockChainId, name: 'mainnet' };
  const mockPublicClient = { request: jest.fn() };
  const mockBundlerClient = { sendUserOperation: jest.fn() };
  const mockPimlicoClient = { getUserOperationGasPrice: jest.fn() };
  const mockRedeemer = { 
    address: mockRedeemerAddress,
    signDelegation: jest.fn()
  };

  beforeEach(() => {
    jest.clearAllMocks();

    // Setup default mocks
    (getNetworkConfig as jest.Mock).mockResolvedValue({
      rpcUrl: 'https://mainnet.infura.io/v3/test',
      bundlerUrl: 'https://api.pimlico.io/v2/1/rpc?apikey=test'
    });

    (getSecretValue as jest.Mock).mockResolvedValue('0xprivatekey123');

    (delegationLib.getChainById as jest.Mock).mockReturnValue(mockChain);
    (delegationLib.createNetworkConfigFromUrls as jest.Mock).mockReturnValue({
      name: mockNetworkName,
      chainId: mockChainId,
      rpcUrl: 'https://mainnet.infura.io/v3/test',
      bundlerUrl: 'https://api.pimlico.io/v2/1/rpc?apikey=test'
    });

    (delegationLib.initializeBlockchainClients as jest.Mock).mockResolvedValue({
      publicClient: mockPublicClient,
      bundlerClient: mockBundlerClient,
      pimlicoClient: mockPimlicoClient
    });

    (delegationLib.parseDelegation as jest.Mock).mockReturnValue(mockDelegation);
    (delegationLib.validateDelegation as jest.Mock).mockResolvedValue(true);
    (delegationLib.validateRedemptionInputs as jest.Mock).mockReturnValue(undefined);
    (delegationLib.getOrCreateRedeemerAccount as jest.Mock).mockResolvedValue(mockRedeemer);
    (delegationLib.validateDelegateMatch as jest.Mock).mockReturnValue(undefined);
    (delegationLib.prepareRedemptionUserOperationPayload as jest.Mock).mockReturnValue([
      {
        to: mockRedeemerAddress,
        data: '0xcalldata'
      }
    ]);
    (delegationLib.sendAndConfirmUserOperation as jest.Mock).mockResolvedValue(mockTransactionHash);
  });

  describe('Success cases', () => {
    it('should successfully redeem a delegation', async () => {
      const result = await redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      );

      expect(result).toBe(mockTransactionHash);

      // Verify all steps were called correctly
      expect(delegationLib.validateRedemptionInputs).toHaveBeenCalledWith({
        delegationData: mockDelegationData,
        merchantAddress: mockMerchantAddress,
        tokenContractAddress: mockTokenContractAddress,
        tokenAmount: mockTokenAmount,
        tokenDecimals: mockTokenDecimals,
        chainId: mockChainId,
        networkName: mockNetworkName
      });

      expect(getNetworkConfig).toHaveBeenCalledWith(mockNetworkName, mockChainId);
      expect(delegationLib.getChainById).toHaveBeenCalledWith(mockChainId);
      expect(delegationLib.parseDelegation).toHaveBeenCalledWith(mockDelegationData);
      expect(delegationLib.validateDelegation).toHaveBeenCalledWith(mockDelegation, mockPublicClient);
      expect(getSecretValue).toHaveBeenCalledWith('PRIVATE_KEY_ARN', 'PRIVATE_KEY');
      expect(delegationLib.getOrCreateRedeemerAccount).toHaveBeenCalledWith(
        mockPublicClient,
        {
          privateKey: '0xprivatekey123',
          deploySalt: '0x'
        }
      );
      expect(delegationLib.validateDelegateMatch).toHaveBeenCalledWith(
        mockRedeemerAddress,
        mockDelegation.delegate
      );
      expect(delegationLib.prepareRedemptionUserOperationPayload).toHaveBeenCalledWith(
        mockDelegation,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockRedeemerAddress
      );
      expect(delegationLib.sendAndConfirmUserOperation).toHaveBeenCalled();
    });

    it('should handle status updates during UserOperation sending', async () => {
      const statusCallback = jest.fn();
      
      (delegationLib.sendAndConfirmUserOperation as jest.Mock).mockImplementation(
        async (bundlerClient, pimlicoClient, redeemer, calls, publicClient, options) => {
          options.onStatusUpdate('Sending UserOperation...');
          options.onStatusUpdate('UserOperation confirmed!');
          return mockTransactionHash;
        }
      );

      await redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      );

      expect(logger.info).toHaveBeenCalledWith('Sending UserOperation...');
      expect(logger.info).toHaveBeenCalledWith('UserOperation confirmed!');
    });
  });

  describe('Error cases', () => {
    it('should throw error when validation fails', async () => {
      (delegationLib.validateRedemptionInputs as jest.Mock).mockImplementation(() => {
        throw new Error('Invalid input parameters');
      });

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Invalid input parameters');

      expect(logger.error).toHaveBeenCalledWith(
        'Critical error in redeemDelegation service:',
        expect.objectContaining({
          message: 'Invalid input parameters'
        })
      );
    });

    it('should throw error when network config is not found', async () => {
      (getNetworkConfig as jest.Mock).mockRejectedValue(new Error('Network not found'));

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Network not found');
    });

    it('should throw error when chain is not supported', async () => {
      (delegationLib.getChainById as jest.Mock).mockImplementation(() => {
        throw new Error('Unsupported chainId');
      });

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Unsupported chainId');
    });

    it('should throw error when delegation parsing fails', async () => {
      (delegationLib.parseDelegation as jest.Mock).mockImplementation(() => {
        throw new Error('Invalid delegation format');
      });

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Invalid delegation format');
    });

    it('should throw error when delegation validation fails', async () => {
      (delegationLib.validateDelegation as jest.Mock).mockRejectedValue(
        new Error('Delegation signature invalid')
      );

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Delegation signature invalid');
    });

    it('should throw error when private key retrieval fails', async () => {
      (getSecretValue as jest.Mock).mockRejectedValue(new Error('Secret not found'));

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Secret not found');
    });

    it('should throw error when delegate does not match redeemer', async () => {
      (delegationLib.validateDelegateMatch as jest.Mock).mockImplementation(() => {
        throw new Error('Delegate mismatch');
      });

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Delegate mismatch');
    });

    it('should throw error when UserOperation fails', async () => {
      (delegationLib.sendAndConfirmUserOperation as jest.Mock).mockRejectedValue(
        new Error('UserOperation execution failed')
      );

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: UserOperation execution failed');
    });

    it('should handle non-Error exceptions', async () => {
      (delegationLib.parseDelegation as jest.Mock).mockImplementation(() => {
        throw 'String error';
      });

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed due to an unknown error.');

      expect(logger.error).toHaveBeenCalledWith(
        'Critical error in redeemDelegation service:',
        expect.objectContaining({
          errorObject: 'String error'
        })
      );
    });
  });

  describe('Edge cases', () => {
    it('should handle empty delegation data', async () => {
      const emptyData = Buffer.from('');
      
      (delegationLib.validateRedemptionInputs as jest.Mock).mockImplementation(() => {
        throw new Error('Delegation data is required');
      });

      await expect(redeemDelegation(
        emptyData,
        mockMerchantAddress,
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Delegation data is required');
    });

    it('should handle zero token amount', async () => {
      (delegationLib.validateRedemptionInputs as jest.Mock).mockImplementation(() => {
        throw new Error('Valid token amount is required');
      });

      await expect(redeemDelegation(
        mockDelegationData,
        mockMerchantAddress,
        mockTokenContractAddress,
        0,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Valid token amount is required');
    });

    it('should handle invalid addresses', async () => {
      (delegationLib.validateRedemptionInputs as jest.Mock).mockImplementation(() => {
        throw new Error('Valid merchant address is required');
      });

      await expect(redeemDelegation(
        mockDelegationData,
        '0x0000000000000000000000000000000000000000',
        mockTokenContractAddress,
        mockTokenAmount,
        mockTokenDecimals,
        mockChainId,
        mockNetworkName
      )).rejects.toThrow('RedeemDelegation failed: Valid merchant address is required');
    });
  });
});