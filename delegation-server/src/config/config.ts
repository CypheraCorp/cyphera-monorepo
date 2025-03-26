import dotenv from 'dotenv'
import { resolve } from 'path'
import { formatPrivateKey } from '../utils/utils'

// Load environment variables from .env file
dotenv.config({ path: resolve(__dirname, '../../.env') })

// Configuration object with all environment variables
export const config = {
  grpc: {
    port: parseInt(process.env.GRPC_PORT || '50051', 10),
    host: process.env.GRPC_HOST || '0.0.0.0'
  },
  blockchain: {
    rpcUrl: process.env.RPC_URL,
    bundlerUrl: process.env.BUNDLER_URL,
    chainId: parseInt(process.env.CHAIN_ID || '11155111', 10),
    privateKey: process.env.PRIVATE_KEY
  },
  logging: {
    level: process.env.LOG_LEVEL || 'info'
  }
}

// Validate required configuration
export function validateConfig(): void {
  const requiredVars = [
    { key: 'blockchain.rpcUrl', value: config.blockchain.rpcUrl },
    { key: 'blockchain.bundlerUrl', value: config.blockchain.bundlerUrl },
    { key: 'blockchain.privateKey', value: config.blockchain.privateKey },
  ]
  
  const missingVars = requiredVars.filter(v => !v.value)
  
  if (missingVars.length > 0) {
    const missingKeys = missingVars.map(v => v.key).join(', ')
    throw new Error(`Missing required environment variables: ${missingKeys}`)
  }
  
  // Validate private key format
  if (config.blockchain.privateKey) {
    const pkRegex = /^0x[0-9a-fA-F]{64}$/
    if (!pkRegex.test(config.blockchain.privateKey)) {
      throw new Error('PRIVATE_KEY must be a valid 32-byte hex string with 0x prefix (66 characters total)')
    }
  }
} 