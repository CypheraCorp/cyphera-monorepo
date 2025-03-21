import path from 'path'
import * as grpc from '@grpc/grpc-js'
import * as protoLoader from '@grpc/proto-loader'
import { config, validateConfig } from './config/config'
import { delegationService } from './services/service'
import { logger } from './utils/utils'

/**
 * Main entry point for the delegation redemption gRPC server
 */
function startServer() {
  try {
    // Validate configuration
    validateConfig()
    
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

    // Create gRPC server
    const server = new grpc.Server()

    // Add the delegation service
    server.addService(delegationPackage.DelegationService.service, delegationService)

    // Start the server
    const serverAddress = `${config.grpc.host}:${config.grpc.port}`
    server.bindAsync(
      serverAddress, 
      grpc.ServerCredentials.createInsecure(),
      (err: Error | null, port: number) => {
        if (err) {
          logger.error('Failed to bind server:', err)
          process.exit(1)
        }
        
        server.start()
        logger.info(`gRPC server started on ${serverAddress}`)
        logger.info('Ready to receive delegation redemption requests')
      }
    )

    // Handle shutdown gracefully
    const shutdownGracefully = () => {
      logger.info('Shutting down gRPC server gracefully...')
      server.tryShutdown(() => {
        logger.info('Server shut down successfully')
        process.exit(0)
      })
    }

    process.on('SIGINT', shutdownGracefully)
    process.on('SIGTERM', shutdownGracefully)
  } catch (error) {
    logger.error('Failed to start server:', error)
    process.exit(1)
  }
}

// Start the server when this file is executed directly
if (require.main === module) {
  startServer()
}

export { startServer } 