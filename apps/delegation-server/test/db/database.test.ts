import { describe, expect, it, jest, beforeEach, afterEach } from '@jest/globals';
import { Pool, PoolClient } from 'pg';
import { initializeDatabase, getClient, query, closeDatabase } from '../../src/db/database';

// Mock pg module
jest.mock('pg', () => {
  const mockPool = {
    query: jest.fn(),
    connect: jest.fn(),
    end: jest.fn(),
    on: jest.fn()
  };
  
  return {
    Pool: jest.fn(() => mockPool)
  };
});

describe('Database Module', () => {
  let mockPool: any;
  
  beforeEach(() => {
    jest.clearAllMocks();
    // Get the mocked pool instance
    mockPool = new (Pool as any)();
    // Reset environment variables
    delete process.env.DATABASE_URL;
    delete process.env.DATABASE_CONNECTION_STRING;
  });
  
  afterEach(async () => {
    await closeDatabase();
  });

  describe('initializeDatabase', () => {
    it('should initialize pool with DATABASE_URL', () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      
      const pool = initializeDatabase();
      
      expect(Pool).toHaveBeenCalledWith({
        connectionString: 'postgresql://test:test@localhost:5432/test',
        max: 10,
        idleTimeoutMillis: 30000,
        connectionTimeoutMillis: 2000
      });
      expect(pool).toBe(mockPool);
      expect(mockPool.on).toHaveBeenCalledWith('error', expect.any(Function));
    });
    
    it('should initialize pool with DATABASE_CONNECTION_STRING', () => {
      process.env.DATABASE_CONNECTION_STRING = 'postgresql://test:test@localhost:5432/test';
      
      const pool = initializeDatabase();
      
      expect(Pool).toHaveBeenCalledWith({
        connectionString: 'postgresql://test:test@localhost:5432/test',
        max: 10,
        idleTimeoutMillis: 30000,
        connectionTimeoutMillis: 2000
      });
    });
    
    it('should throw error if no connection string is provided', () => {
      expect(() => initializeDatabase())
        .toThrow('DATABASE_URL or DATABASE_CONNECTION_STRING environment variable is required');
    });
    
    it('should return existing pool if already initialized', () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      
      const pool1 = initializeDatabase();
      const pool2 = initializeDatabase();
      
      expect(pool1).toBe(pool2);
      expect(Pool).toHaveBeenCalledTimes(1);
    });
  });

  describe('getClient', () => {
    it('should return a pool client', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      const mockClient = { release: jest.fn() };
      mockPool.connect.mockResolvedValue(mockClient);
      
      const client = await getClient();
      
      expect(mockPool.connect).toHaveBeenCalled();
      expect(client).toBe(mockClient);
    });
    
    it('should initialize pool if not already initialized', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      const mockClient = { release: jest.fn() };
      mockPool.connect.mockResolvedValue(mockClient);
      
      const client = await getClient();
      
      expect(Pool).toHaveBeenCalled();
      expect(client).toBe(mockClient);
    });
  });

  describe('query', () => {
    it('should execute query and return results', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      const mockResult = {
        rows: [{ id: 1, name: 'Test' }],
        rowCount: 1
      };
      mockPool.query.mockResolvedValue(mockResult);
      
      const result = await query('SELECT * FROM test WHERE id = $1', [1]);
      
      expect(mockPool.query).toHaveBeenCalledWith('SELECT * FROM test WHERE id = $1', [1]);
      expect(result).toEqual({
        rows: [{ id: 1, name: 'Test' }],
        rowCount: 1
      });
    });
    
    it('should handle query without parameters', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      const mockResult = {
        rows: [{ id: 1 }, { id: 2 }],
        rowCount: 2
      };
      mockPool.query.mockResolvedValue(mockResult);
      
      const result = await query('SELECT * FROM test');
      
      expect(mockPool.query).toHaveBeenCalledWith('SELECT * FROM test', undefined);
      expect(result.rowCount).toBe(2);
    });
    
    it('should handle empty results', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      mockPool.query.mockResolvedValue({
        rows: [],
        rowCount: null
      });
      
      const result = await query('SELECT * FROM test WHERE id = $1', [999]);
      
      expect(result).toEqual({
        rows: [],
        rowCount: 0
      });
    });
    
    it('should throw and log on query error', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      const error = new Error('Query failed');
      mockPool.query.mockRejectedValue(error);
      
      await expect(query('SELECT * FROM invalid_table'))
        .rejects.toThrow('Query failed');
    });
  });

  describe('closeDatabase', () => {
    it('should close the pool', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      initializeDatabase();
      mockPool.end.mockResolvedValue(undefined);
      
      await closeDatabase();
      
      expect(mockPool.end).toHaveBeenCalled();
    });
    
    it('should handle closing when pool is not initialized', async () => {
      await expect(closeDatabase()).resolves.not.toThrow();
    });
    
    it('should allow reinitializing after close', async () => {
      process.env.DATABASE_URL = 'postgresql://test:test@localhost:5432/test';
      
      initializeDatabase();
      await closeDatabase();
      
      const newPool = initializeDatabase();
      expect(Pool).toHaveBeenCalledTimes(2);
      expect(newPool).toBe(mockPool);
    });
  });
});