import * as grpc from '@grpc/grpc-js'
import * as path from 'path'
import * as protoLoader from '@grpc/proto-loader'
import { delegationService } from './services/service'
import { logger } from './utils/utils'
import config from './config'

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
  logger.info(`RPC_URL: ${redactUrl(envVars.RPC_URL)}`);
  logger.info(`BUNDLER_URL: ${redactUrl(envVars.BUNDLER_URL)}`);
  logger.info(`NPM_TOKEN: ${redactPrivateKey(envVars.NPM_TOKEN)}`);

  logger.info(`CHAIN_ID: ${envVars.CHAIN_ID || 'not set'}`);
  logger.info(`PRIVATE_KEY: ${redactPrivateKey(envVars.PRIVATE_KEY)}`);
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
  process.on('SIGINT', () => {
    logger.info('Received SIGINT. Shutting down gracefully...')
    server.tryShutdown(() => {
      logger.info('Server shut down successfully')
      process.exit(0)
    })
  })
}

// Start the server directly when the file is executed
if (require.main === module) {
  logEnvironmentVariables();
  startServer()
} 