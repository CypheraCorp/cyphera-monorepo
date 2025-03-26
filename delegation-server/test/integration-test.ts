/**
 * Integration Test for Delegation gRPC Server
 * 
 * This script tests the full flow of the delegation redemption process by:
 * 1. Creating a gRPC client to connect to the delegation server
 * 2. Sending a mock delegation redemption request
 * 3. Verifying the response from the server
 * 
 * To run: 
 *   ts-node test/integration-test.ts
 * 
 * Note: The delegation server must be running before executing this test.
 */

import * as grpc from '@grpc/grpc-js'
import * as protoLoader from '@grpc/proto-loader'
import { resolve } from 'path'
import { randomBytes } from 'crypto'

// Configure the test
const CONFIG = {
  serverAddress: process.env.GRPC_SERVER_ADDRESS || 'localhost:50051',
  protoPath: resolve(__dirname, '../proto/delegation.proto'),
  timeout: 30000, // 30 seconds
}

// Sample delegation data - this should match the format expected by the server
const mockDelegation = {
  delegate: '0x1234567890123456789012345678901234567890', // This should match the server's private key
  delegator: '0x0987654321098765432109876543210987654321',
  authority: '0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789', // EntryPoint address
  caveats: [],
  salt: '0x' + randomBytes(32).toString('hex'),
  signature: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890',
}

/**
 * Loads the proto definition and creates a gRPC client
 */
function createClient() {
  // Load proto definition
  const packageDefinition = protoLoader.loadSync(CONFIG.protoPath, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true
  })

  const protoDescriptor = grpc.loadPackageDefinition(packageDefinition)
  const delegationPackage = protoDescriptor.delegation as any
  
  // Create client
  return new delegationPackage.DelegationService(
    CONFIG.serverAddress,
    grpc.credentials.createInsecure(),
    {
      'grpc.keepalive_time_ms': 10000,
      'grpc.keepalive_timeout_ms': 5000,
      'grpc.keepalive_permit_without_calls': 1,
    }
  )
}

/**
 * Runs the integration test
 */
async function runTest() {
  console.log('Starting delegation server integration test...')
  console.log(`Connecting to gRPC server at ${CONFIG.serverAddress}`)
  
  const client = createClient()
  
  // Prepare request
  const delegationJSON = JSON.stringify(mockDelegation)
  const delegationData = Buffer.from(delegationJSON, 'utf-8')
  
  console.log(`Prepared mock delegation with size: ${delegationData.length} bytes`)
  
  // Create promise-based version of the gRPC call
  const redeemDelegation = (): Promise<any> => {
    return new Promise((resolve, reject) => {
      const deadline = new Date(Date.now() + CONFIG.timeout)
      
      client.redeemDelegation(
        { delegationData }, 
        { deadline }, 
        (error: Error | null, response: any) => {
          if (error) {
            reject(error)
            return
          }
          resolve(response)
        }
      )
    })
  }
  
  try {
    console.log('Sending delegation redemption request...')
    const startTime = Date.now()
    
    const response = await redeemDelegation()
    
    const elapsedTime = (Date.now() - startTime) / 1000
    console.log(`Received response in ${elapsedTime.toFixed(2)} seconds`)
    
    if (response.success) {
      console.log('✅ TEST PASSED: Delegation redemption successful')
      console.log(`Transaction hash: ${response.transactionHash}`)
    } else {
      console.error('❌ TEST FAILED: Delegation redemption failed')
      console.error(`Error message: ${response.errorMessage}`)
      process.exit(1)
    }
  } catch (error) {
    console.error('❌ TEST FAILED: gRPC call error', error)
    process.exit(1)
  } finally {
    // Close the client connection
    client.close()
  }
}

// Run the test when this script is executed directly
if (require.main === module) {
  runTest().catch(error => {
    console.error('Unhandled error in test:', error)
    process.exit(1)
  })
} 