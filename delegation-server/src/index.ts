import * as grpc from '@grpc/grpc-js'
import * as path from 'path'
import * as protoLoader from '@grpc/proto-loader'

// Simple logger implementation
const logger = {
  info: (message: string, ...args: any[]) => console.log(`[${new Date().toISOString()}] INFO: ${message}`, ...args),
  error: (message: string, ...args: any[]) => console.error(`[${new Date().toISOString()}] ERROR: ${message}`, ...args),
  debug: (message: string, ...args: any[]) => console.debug(`[${new Date().toISOString()}] DEBUG: ${message}`, ...args),
}

import config from './config'

// Mock implementation of blockchain service
class MockBlockchainService {
  async redeemDelegation(delegationData: any): Promise<string> {
    // Simulate blockchain transaction
    logger.info('Mock redeeming delegation', delegationData)
    // Return a mock transaction hash
    return `0x${Math.random().toString(16).substring(2)}`
  }
}

// Implementation of the delegation service
class DelegationServiceImpl {
  private blockchainService: MockBlockchainService

  constructor(blockchainService: MockBlockchainService) {
    this.blockchainService = blockchainService
  }

  // Add index signature to make TypeScript happy with gRPC service implementation
  [key: string]: any

  async redeemDelegation(call: any, callback: any) {
    try {
      const request = call.request
      logger.info('Received redemption request')

      if (!request.delegationData || request.delegationData.length === 0) {
        return callback({
          code: grpc.status.INVALID_ARGUMENT,
          message: 'Invalid delegation data'
        })
      }

      // Parse the delegation data
      let delegationData
      try {
        delegationData = JSON.parse(request.delegationData.toString())
      } catch (e) {
        return callback({
          code: grpc.status.INVALID_ARGUMENT,
          message: 'Invalid JSON in delegation data'
        })
      }

      // Submit the redemption to the blockchain
      const txHash = await this.blockchainService.redeemDelegation(delegationData)
      
      // Return the transaction hash
      callback(null, { txHash })
    } catch (error) {
      logger.error('Error processing redemption:', error)
      callback({
        code: grpc.status.INTERNAL,
        message: 'Internal server error'
      })
    }
  }
}

// Create mock blockchain service based on config
const blockchainService = new MockBlockchainService()

if (config.mockMode) {
  logger.info('Running in MOCK MODE - using mock blockchain service')
}

// Load the protobuf definition
const PROTO_PATH = path.join(__dirname, '../proto/delegation.proto')

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
  new DelegationServiceImpl(blockchainService)
)

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

// No calls to startServer() - server is started directly in this file 