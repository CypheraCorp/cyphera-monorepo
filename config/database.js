const { Pool } = require('pg');
require('dotenv').config();

const createPool = () => {
  const pool = new Pool({
    user: process.env.POSTGRES_USER,
    host: process.env.POSTGRES_HOST,
    database: process.env.POSTGRES_DB,
    password: process.env.POSTGRES_PASSWORD,
    port: process.env.POSTGRES_PORT || 5432,
  });

  pool.on('connect', () => {
    console.log('Connected to PostgreSQL database');
  });

  pool.on('error', (err) => {
    console.error('Unexpected error on idle client', err);
  });

  return pool;
};

const connectWithRetry = async () => {
  let retries = 5;
  while (retries) {
    try {
      const pool = createPool();
      await pool.query('SELECT 1');
      return pool;
    } catch (err) {
      console.log(`Failed to connect to PostgreSQL. Retries left: ${retries}`);
      retries -= 1;
      await new Promise(resolve => setTimeout(resolve, 5000));
    }
  }
  throw new Error('Could not connect to PostgreSQL');
};

module.exports = connectWithRetry(); 