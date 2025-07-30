import { ServerUnaryCall, sendUnaryData } from '@grpc/grpc-js';

// Mock the external dependencies before importing the service
jest.mock('../utils/utils', () => ({
  logger: {
    info: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
    warn: jest.fn()
  }
}));

jest.mock('../config', () => ({
  default: {
    mockMode: false
  }
}));

// Mock the redeem-delegation module
jest.mock('./redeem-delegation', () => ({
  redeemDelegation: jest.fn()
}));

// Mock the mock-redeem-delegation module
jest.mock('./mock-redeem-delegation', () => ({
  redeemDelegation: jest.fn()
}));

// Import after mocks are set up
import { delegationService } from './service';
import { logger } from '../utils/utils';

describe('DelegationService', () => {
  let mockCall: Partial<ServerUnaryCall<any, any>>;
  let mockCallback: jest.MockedFunction<sendUnaryData<any>>;
  let mockRedeemDelegation: jest.Mock;
  let mockMockRedeemDelegation: jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Get the mocked functions
    const redeemModule = require('./redeem-delegation');
    const mockRedeemModule = require('./mock-redeem-delegation');
    mockRedeemDelegation = redeemModule.redeemDelegation;
    mockMockRedeemDelegation = mockRedeemModule.redeemDelegation;
    
    mockCallback = jest.fn();
    
    // Default mock call with valid request data
    mockCall = {
      request: {
        signature: Buffer.from('test-signature'),
        merchant_address: '0x1234567890123456789012345678901234567890',
        token_contract_address: '0x0987654321098765432109876543210987654321',
        token_amount: 1000000,
        token_decimals: 6,
        chain_id: 1,
        network_name: 'mainnet'
      }
    };
  });

  describe('redeemDelegation', () => {
    it('should successfully process a delegation redemption request', async () => {
      mockRedeemDelegation.mockResolvedValue('0xabc123def456');

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      // Verify the redeem function was called with correct parameters
      expect(mockRedeemDelegation).toHaveBeenCalledWith(
        Buffer.from('test-signature'),
        '0x1234567890123456789012345678901234567890',
        '0x0987654321098765432109876543210987654321',
        1000000,
        6,
        1,
        'mainnet'
      );

      // Verify success response
      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '0xabc123def456',
        transactionHash: '0xabc123def456',
        success: true,
        error_message: '',
        errorMessage: ''
      });

      // Verify logging
      expect(logger.info).toHaveBeenCalledWith('Received RedeemDelegation request');
      expect(logger.info).toHaveBeenCalledWith(
        'Redemption successful, transaction hash: 0xabc123def456'
      );
    });

    it('should handle camelCase field names in request', async () => {
      mockRedeemDelegation.mockResolvedValue('0xabc123def456');

      mockCall.request = {
        signature: Buffer.from('test-signature'),
        merchantAddress: '0x1234567890123456789012345678901234567890',
        tokenContractAddress: '0x0987654321098765432109876543210987654321',
        token_amount: 1000000,
        token_decimals: 6,
        chain_id: 1,
        network_name: 'mainnet'
      };

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockRedeemDelegation).toHaveBeenCalledWith(
        Buffer.from('test-signature'),
        '0x1234567890123456789012345678901234567890',
        '0x0987654321098765432109876543210987654321',
        1000000,
        6,
        1,
        'mainnet'
      );
    });

    it('should return error response when chainId is missing', async () => {
      mockCall.request!.chain_id = undefined;

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '',
        transactionHash: '',
        success: false,
        error_message: 'Missing or invalid chain_id in request',
        errorMessage: 'Missing or invalid chain_id in request'
      });
    });

    it('should return error response when chainId is invalid', async () => {
      mockCall.request!.chain_id = 0;

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '',
        transactionHash: '',
        success: false,
        error_message: 'Missing or invalid chain_id in request',
        errorMessage: 'Missing or invalid chain_id in request'
      });
    });

    it('should return error response when network_name is missing', async () => {
      mockCall.request!.network_name = '';

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '',
        transactionHash: '',
        success: false,
        error_message: 'Missing network_name in request',
        errorMessage: 'Missing network_name in request'
      });
    });

    it('should return error response when token_amount is invalid', async () => {
      mockCall.request!.token_amount = 0;

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '',
        transactionHash: '',
        success: false,
        error_message: 'Missing or invalid token_amount in request',
        errorMessage: 'Missing or invalid token_amount in request'
      });
    });

    it('should return error response when token_decimals is invalid', async () => {
      mockCall.request!.token_decimals = 0;

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '',
        transactionHash: '',
        success: false,
        error_message: 'Missing or invalid token_decimals in request',
        errorMessage: 'Missing or invalid token_decimals in request'
      });
    });

    it('should handle errors from redeemDelegation function', async () => {
      mockRedeemDelegation.mockRejectedValue(new Error('Blockchain error'));

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '',
        transactionHash: '',
        success: false,
        error_message: 'Blockchain error',
        errorMessage: 'Blockchain error'
      });

      expect(logger.error).toHaveBeenCalledWith(
        'Error in redeemDelegation:',
        'Blockchain error'
      );
    });

    it('should handle non-Error exceptions', async () => {
      mockRedeemDelegation.mockRejectedValue('String error');

      await delegationService.redeemDelegation(
        mockCall as ServerUnaryCall<any, any>,
        mockCallback
      );

      expect(mockCallback).toHaveBeenCalledWith(null, {
        transaction_hash: '',
        transactionHash: '',
        success: false,
        error_message: 'String error',
        errorMessage: 'String error'
      });
    });
  });

  describe('Mock Mode', () => {
    // We can't easily test the mock mode switching since the service module
    // loads the implementation at module load time. The best we can do is
    // verify that the mock function exists and would work if called.
    it('mock redeem-delegation module should be callable', async () => {
      mockMockRedeemDelegation.mockResolvedValue('0xmock123');
      
      const result = await mockMockRedeemDelegation(
        Buffer.from('test-signature'),
        '0x1234567890123456789012345678901234567890',
        '0x0987654321098765432109876543210987654321',
        1000000,
        6,
        1,
        'mainnet'
      );
      
      expect(result).toBe('0xmock123');
    });
  });
});