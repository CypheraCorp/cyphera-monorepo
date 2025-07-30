import { describe, expect, it, jest, beforeEach } from '@jest/globals';
import {
  getChainById,
  createFetchTransport,
  initializeBlockchainClients,
  createNetworkConfigFromUrls
} from './blockchain-clients';
import { RedemptionErrorType } from './types';
import * as chains from 'viem/chains';

// Mock node-fetch
jest.mock('node-fetch');

describe('Blockchain Clients', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getChainById', () => {
    it('should return chain for known chain IDs', () => {
      const ethereum = getChainById(1);
      expect(ethereum.id).toBe(1);
      expect(ethereum.name).toBe('Ethereum');

      const polygon = getChainById(137);
      expect(polygon.id).toBe(137);
      expect(polygon.name).toBe('Polygon');
    });

    it('should create a minimal chain object for unknown chain IDs', () => {
      const unknownChain = getChainById(999999);
      expect(unknownChain.id).toBe(999999);
      expect(unknownChain.name).toBe('Chain 999999');
      expect(unknownChain.nativeCurrency).toEqual({
        name: 'ETH',
        symbol: 'ETH',
        decimals: 18
      });
    });
  });

  describe('createFetchTransport', () => {
    it('should throw error if URL is not provided', () => {
      expect(() => createFetchTransport(undefined))
        .toThrow('URL is required for transport');
    });

    it('should create a transport with valid URL', () => {
      const transport = createFetchTransport('https://mainnet.infura.io/v3/key');
      expect(transport).toBeDefined();
      expect(typeof transport).toBe('function');
    });
  });

  describe('createNetworkConfigFromUrls', () => {
    it('should create network config from URLs', () => {
      const config = createNetworkConfigFromUrls(
        'Ethereum Mainnet',
        1,
        'https://mainnet.infura.io/v3/key',
        'https://api.pimlico.io/v2/1/rpc?apikey=key'
      );

      expect(config).toEqual({
        chainId: 1,
        name: 'Ethereum Mainnet',
        rpcUrl: 'https://mainnet.infura.io/v3/key',
        bundlerUrl: 'https://api.pimlico.io/v2/1/rpc?apikey=key',
        nativeCurrency: {
          decimals: 18,
          name: 'Ether',
          symbol: 'ETH'
        },
        blockExplorer: 'https://etherscan.io'
      });
    });

    it('should handle unknown chains', () => {
      const config = createNetworkConfigFromUrls(
        'Custom Chain',
        999999,
        'https://custom-rpc.example.com',
        'https://custom-bundler.example.com'
      );

      expect(config).toEqual({
        chainId: 999999,
        name: 'Custom Chain',
        rpcUrl: 'https://custom-rpc.example.com',
        bundlerUrl: 'https://custom-bundler.example.com',
        nativeCurrency: {
          decimals: 18,
          name: 'ETH',
          symbol: 'ETH'
        },
        blockExplorer: undefined
      });
    });
  });

  describe('initializeBlockchainClients', () => {
    it('should initialize all blockchain clients', async () => {
      const networkConfig = {
        chainId: 1,
        name: 'Ethereum Mainnet',
        rpcUrl: 'https://mainnet.infura.io/v3/key',
        bundlerUrl: 'https://api.pimlico.io/v2/1/rpc?apikey=key',
        nativeCurrency: {
          name: 'Ether',
          symbol: 'ETH',
          decimals: 18
        }
      };

      const chain = chains.mainnet;

      // Mock the fetch response
      const mockFetch = jest.fn();
      mockFetch.mockResolvedValue({
        ok: true,
        json: jest.fn().mockResolvedValue({ result: '0x1' })
      });
      (global as any).fetch = mockFetch;

      const clients = await initializeBlockchainClients(networkConfig, chain);

      expect(clients).toHaveProperty('publicClient');
      expect(clients).toHaveProperty('bundlerClient');
      expect(clients).toHaveProperty('pimlicoClient');
    });

    it('should handle initialization errors', async () => {
      const networkConfig = {
        chainId: 1,
        name: 'Ethereum Mainnet',
        rpcUrl: undefined as any, // Invalid URL
        bundlerUrl: 'https://api.pimlico.io/v2/1/rpc?apikey=key',
        nativeCurrency: {
          name: 'Ether',
          symbol: 'ETH',
          decimals: 18
        }
      };

      const chain = chains.mainnet;

      await expect(initializeBlockchainClients(networkConfig, chain))
        .rejects.toThrow('Failed to initialize blockchain clients');
    });
  });
});