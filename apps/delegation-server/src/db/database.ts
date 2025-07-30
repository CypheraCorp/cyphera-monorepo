import { Pool, PoolClient } from 'pg';
import { logger } from '../utils/utils';

/**
 * Database connection pool
 */
let pool: Pool | null = null;

/**
 * Initialize database connection pool
 */
export function initializeDatabase(): Pool {
  if (pool) {
    return pool;
  }

  // Try multiple sources for database connection
  const connectionString = process.env.DATABASE_URL || 
                          process.env.DATABASE_CONNECTION_STRING ||
                          process.env.TEST_DATABASE_URL;
  
  if (!connectionString) {
    logger.error('Database connection string not found in environment variables');
    logger.error('Checked: DATABASE_URL, DATABASE_CONNECTION_STRING, TEST_DATABASE_URL');
    logger.error('Current env vars:', Object.keys(process.env).filter(key => key.includes('DATABASE')));
    throw new Error('DATABASE_URL or DATABASE_CONNECTION_STRING environment variable is required');
  }

  pool = new Pool({
    connectionString,
    max: 10, // Maximum number of clients in the pool
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 2000,
  });

  pool.on('error', (err) => {
    logger.error('Unexpected error on idle database client', err);
  });

  logger.info('Database connection pool initialized');
  return pool;
}

/**
 * Get a client from the pool
 */
export async function getClient(): Promise<PoolClient> {
  if (!pool) {
    initializeDatabase();
  }
  return pool!.connect();
}

/**
 * Execute a query
 */
export async function query<T = any>(
  text: string,
  params?: any[]
): Promise<{ rows: T[]; rowCount: number }> {
  if (!pool) {
    initializeDatabase();
  }
  
  try {
    const result = await pool!.query(text, params);
    return {
      rows: result.rows as T[],
      rowCount: result.rowCount || 0
    };
  } catch (error) {
    logger.error('Database query error:', { error, query: text, params });
    throw error;
  }
}

/**
 * Close database connection pool
 */
export async function closeDatabase(): Promise<void> {
  if (pool) {
    await pool.end();
    pool = null;
    logger.info('Database connection pool closed');
  }
}