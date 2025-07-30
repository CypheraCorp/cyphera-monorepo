import { describe, expect, it, jest, beforeEach } from '@jest/globals';
import {
  getNetworkByChainId,
  getNetworkByName,
  getAllNetworks,
  getTokensForNetwork,
  getTokenByAddress,
  validateTokenSupport
} from '../../src/db/network-service';
import * as database from '../../src/db/database';

// Mock the database module
jest.mock('../../src/db/database');

describe('Network Service', () => {
  const mockQuery = database.query as jest.MockedFunction<typeof database.query>;
  
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const mockNetwork = {
    id: 'net-123',
    name: 'mainnet',
    type: 'ethereum',
    network_type: 'ETHEREUM',
    rpc_id: 'mainnet',
    block_explorer_url: 'https://etherscan.io',
    chain_id: 1,
    is_testnet: false,
    active: true,
    display_name: 'Ethereum Mainnet',
    chain_namespace: 'eip155',
    deleted_at: null
  };

  const mockToken = {
    id: 'token-123',
    network_id: 'net-123',
    gas_token: false,
    name: 'USD Coin',
    symbol: 'USDC',
    contract_address: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',
    active: true,
    decimals: 6,
    deleted_at: null
  };

  describe('getNetworkByChainId', () => {
    it('should return network for valid chain ID', async () => {
      mockQuery.mockResolvedValue({
        rows: [mockNetwork],
        rowCount: 1
      });

      const network = await getNetworkByChainId(1);

      expect(mockQuery).toHaveBeenCalledWith(
        expect.stringContaining('WHERE chain_id = $1'),
        [1]
      );
      expect(network).toEqual(mockNetwork);
    });

    it('should return null for non-existent chain ID', async () => {
      mockQuery.mockResolvedValue({
        rows: [],
        rowCount: 0
      });

      const network = await getNetworkByChainId(999);

      expect(network).toBeNull();
    });

    it('should throw error on database failure', async () => {
      mockQuery.mockRejectedValue(new Error('Database error'));

      await expect(getNetworkByChainId(1))
        .rejects.toThrow('Failed to fetch network for chain ID 1');
    });
  });

  describe('getNetworkByName', () => {
    it('should return network by name (case insensitive)', async () => {
      mockQuery.mockResolvedValue({
        rows: [mockNetwork],
        rowCount: 1
      });

      const network = await getNetworkByName('MAINNET');

      expect(mockQuery).toHaveBeenCalledWith(
        expect.stringContaining('LOWER(name) = LOWER($1)'),
        ['MAINNET']
      );
      expect(network).toEqual(mockNetwork);
    });

    it('should return network by display name', async () => {
      mockQuery.mockResolvedValue({
        rows: [mockNetwork],
        rowCount: 1
      });

      const network = await getNetworkByName('Ethereum Mainnet');

      expect(mockQuery).toHaveBeenCalledWith(
        expect.stringContaining('LOWER(display_name) = LOWER($1)'),
        ['Ethereum Mainnet']
      );
      expect(network).toEqual(mockNetwork);
    });

    it('should return null for non-existent network', async () => {
      mockQuery.mockResolvedValue({
        rows: [],
        rowCount: 0
      });

      const network = await getNetworkByName('Unknown Network');

      expect(network).toBeNull();
    });
  });

  describe('getAllNetworks', () => {
    it('should return all active networks ordered correctly', async () => {
      const networks = [
        { ...mockNetwork, is_testnet: false, display_name: 'Ethereum' },
        { ...mockNetwork, id: 'net-456', chain_id: 11155111, is_testnet: true, display_name: 'Sepolia' }
      ];
      
      mockQuery.mockResolvedValue({
        rows: networks,
        rowCount: 2
      });

      const result = await getAllNetworks();

      expect(mockQuery).toHaveBeenCalledWith(
        expect.stringContaining('ORDER BY is_testnet ASC, display_name ASC')
      );
      expect(result).toEqual(networks);
    });

    it('should return empty array when no networks exist', async () => {
      mockQuery.mockResolvedValue({
        rows: [],
        rowCount: 0
      });

      const result = await getAllNetworks();

      expect(result).toEqual([]);
    });
  });

  describe('getTokensForNetwork', () => {
    it('should return tokens for network ordered by gas token and symbol', async () => {
      const tokens = [
        { ...mockToken, gas_token: true, symbol: 'ETH' },
        mockToken
      ];
      
      mockQuery.mockResolvedValue({
        rows: tokens,
        rowCount: 2
      });

      const result = await getTokensForNetwork('net-123');

      expect(mockQuery).toHaveBeenCalledWith(
        expect.stringContaining('ORDER BY gas_token DESC, symbol ASC'),
        ['net-123']
      );
      expect(result).toEqual(tokens);
    });

    it('should return empty array for network with no tokens', async () => {
      mockQuery.mockResolvedValue({
        rows: [],
        rowCount: 0
      });

      const result = await getTokensForNetwork('net-999');

      expect(result).toEqual([]);
    });
  });

  describe('getTokenByAddress', () => {
    it('should return token by address (case insensitive)', async () => {
      mockQuery.mockResolvedValue({
        rows: [mockToken],
        rowCount: 1
      });

      const token = await getTokenByAddress('net-123', '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48');

      expect(mockQuery).toHaveBeenCalledWith(
        expect.stringContaining('LOWER(contract_address) = LOWER($2)'),
        ['net-123', '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48']
      );
      expect(token).toEqual(mockToken);
    });

    it('should return null for non-existent token', async () => {
      mockQuery.mockResolvedValue({
        rows: [],
        rowCount: 0
      });

      const token = await getTokenByAddress('net-123', '0x0000000000000000000000000000000000000000');

      expect(token).toBeNull();
    });
  });

  describe('validateTokenSupport', () => {
    it('should validate supported token', async () => {
      // Mock getNetworkByChainId
      mockQuery.mockResolvedValueOnce({
        rows: [mockNetwork],
        rowCount: 1
      });
      
      // Mock getTokenByAddress
      mockQuery.mockResolvedValueOnce({
        rows: [mockToken],
        rowCount: 1
      });

      const result = await validateTokenSupport(1, '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48');

      expect(result).toEqual({
        valid: true,
        token: mockToken
      });
    });

    it('should return error for unsupported network', async () => {
      mockQuery.mockResolvedValue({
        rows: [],
        rowCount: 0
      });

      const result = await validateTokenSupport(999, '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48');

      expect(result).toEqual({
        valid: false,
        error: 'Network with chain ID 999 not found or not active'
      });
    });

    it('should return error for unsupported token', async () => {
      // Mock getNetworkByChainId
      mockQuery.mockResolvedValueOnce({
        rows: [mockNetwork],
        rowCount: 1
      });
      
      // Mock getTokenByAddress
      mockQuery.mockResolvedValueOnce({
        rows: [],
        rowCount: 0
      });

      const result = await validateTokenSupport(1, '0x0000000000000000000000000000000000000000');

      expect(result).toEqual({
        valid: false,
        error: 'Token 0x0000000000000000000000000000000000000000 not found or not active on network Ethereum Mainnet'
      });
    });

    it('should handle database errors gracefully', async () => {
      mockQuery.mockRejectedValue(new Error('Database connection failed'));

      const result = await validateTokenSupport(1, '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48');

      expect(result).toEqual({
        valid: false,
        error: 'Failed to validate token support: Error: Database connection failed'
      });
    });
  });
});