import * as grpc from '@grpc/grpc-js'
import * as path from 'path'
import * as protoLoader from '@grpc/proto-loader'
import { HealthImplementation, ServingStatusMap } from 'grpc-health-check'

import { delegationService } from './services/service'
import { logger } from './utils/utils'
import config from './config'
import { initializeDatabase, closeDatabase } from './db/database'

// Debug environment variables on startup
function logEnvironmentVariables() {
  const envVars = { ...process.env };

    // Helper function to redact sensitive URLs
    const redactPrivateKey = (url: string | undefined) => {
      if (!url || url === 'not set') return 'not set'
      return url.substring(0, 5) + '...[REDACTED]'
    }
  
  // Helper function to redact sensitive values
  const redactUrl = (value: string | undefined) => {
    if (!value || value === 'not set') return 'not set'
    if (value.length <= 20) return '...[REDACTED]'
    return value.slice(0, -20) + '...[REDACTED]'
  }
  
  logger.info('===== ENVIRONMENT VARIABLES =====');
  logger.info(`MOCK_MODE: ${envVars.MOCK_MODE || 'not set'}`);
  logger.info(`GRPC_HOST: ${envVars.GRPC_HOST || 'not set'}`);
  logger.info(`GRPC_PORT: ${envVars.GRPC_PORT || 'not set'}`);
  logger.info(`DATABASE_URL: ${redactUrl(envVars.DATABASE_URL)}`);
  logger.info(`RPC_URL: ${redactUrl(envVars.RPC_URL)}`);
  logger.info(`INFURA_API_KEY_ARN: ${envVars.INFURA_API_KEY_ARN || 'not set'}`);
  logger.info(`INFURA_API_KEY: ${redactPrivateKey(envVars.INFURA_API_KEY || 'not set')}`);
  logger.info(`PIMLICO_API_KEY_ARN: ${envVars.PIMLICO_API_KEY_ARN || 'not set'}`);
  logger.info(`PIMLICO_API_KEY: ${redactPrivateKey(envVars.PIMLICO_API_KEY || 'not set')}`);
  logger.info(`PRIVATE_KEY_ARN: ${envVars.PRIVATE_KEY_ARN || 'not set'}`);
  logger.info(`PRIVATE_KEY: ${redactPrivateKey(envVars.PRIVATE_KEY || 'not set')}`);
  logger.info(`LOG_LEVEL: ${envVars.LOG_LEVEL || 'not set'}`);
  logger.info('==================================');
  
  // Log loaded configuration
  logger.info('Config loaded:');
  logger.info(`mockMode: ${config.mockMode}`);
  logger.info(`serverAddress: ${config.serverAddress}`);
}

// Load the protobuf definition
const PROTO_PATH = path.join(__dirname, 'proto/delegation.proto')

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true
})

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition)
const delegationPackage = protoDescriptor.delegation as any

// Create a gRPC server
const server = new grpc.Server()

// Register the service
server.addService(
  delegationPackage.DelegationService.service,
  delegationService
)

// --- Setup and Register Health Check Service --- 
// Define service status map. Key is the service name (empty string for overall server status)
const statusMap: ServingStatusMap = {
  // We can add 'delegation.DelegationService': 'SERVING' if needed, but ALB checks ''
  '': 'SERVING', // Indicates the overall server is serving
};

// Construct the health service implementation
const healthImpl = new HealthImplementation(statusMap);

// Add the health service to the server
// This internally loads the health proto and adds the service definition
healthImpl.addToServer(server);

// Example: If you needed to dynamically update status (not strictly required for basic ALB check):
// setTimeout(() => {
//   healthImpl.setStatus('', 'NOT_SERVING');
//   logger.info('Set overall server status to NOT_SERVING');
// }, 5000); 
// --- Done Setting up Health Check --- 

// Function to start the server - exported for backward compatibility
export function startServer() {
  // Bind and start the server
  server.bindAsync(
    config.serverAddress,
    grpc.ServerCredentials.createInsecure(),
    (err, port) => {
      if (err) {
        logger.error(`Failed to bind server: ${err}`)
        return
      }
      server.start()
      logger.info(`Delegation server running on ${config.serverAddress}`)
    }
  )

  // Handle graceful shutdown
  process.on('SIGINT', async () => {
    logger.info('Received SIGINT. Shutting down gracefully...')
    server.tryShutdown(async () => {
      try {
        await closeDatabase();
        logger.info('Database connection closed');
      } catch (error) {
        logger.error('Error closing database connection:', error);
      }
      logger.info('Server shut down successfully')
      process.exit(0)
    })
  })
}

// Start the server directly when the file is executed
if (require.main === module) {
  logEnvironmentVariables();
  
  // Initialize database connection before starting server
  try {
    initializeDatabase();
    logger.info('Database connection initialized');
  } catch (error) {
    logger.error('Failed to initialize database connection:', error);
    process.exit(1);
  }
  
  startServer();
} 