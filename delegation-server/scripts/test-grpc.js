#!/usr/bin/env node

/**
 * Test script for the delegation gRPC server
 * This script creates a gRPC client and sends a test request to the server
 */

const path = require('path');
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
require('dotenv').config({ path: path.resolve(__dirname, '../.env') });

// Function to check if environment variables are set
function checkEnvVars() {
  const requiredEnvVars = ['GRPC_PORT', 'GRPC_HOST'];
  const missingVars = requiredEnvVars.filter(envVar => !process.env[envVar]);
  
  if (missingVars.length > 0) {
    console.error(`Error: The following environment variables are missing: ${missingVars.join(', ')}`);
    console.error('Please set them in a .env file or export them in your shell');
    process.exit(1);
  }
}

// Main function to run the test
async function main() {
  try {
    // Check environment variables
    checkEnvVars();
    
    const GRPC_HOST = process.env.GRPC_HOST || '0.0.0.0';
    const GRPC_PORT = process.env.GRPC_PORT || '50051';
    
    console.log(`Connecting to gRPC server at ${GRPC_HOST}:${GRPC_PORT}...`);
    
    // Load the proto file
    const PROTO_PATH = path.join(__dirname, '../src/proto/delegation.proto');
    
    // Check if proto file exists
    const fs = require('fs');
    if (!fs.existsSync(PROTO_PATH)) {
      console.error(`Error: Proto file not found at ${PROTO_PATH}`);
      process.exit(1);
    }
    
    // Load the proto definition
    const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true
    });
    
    // Create gRPC client
    const delegationProto = grpc.loadPackageDefinition(packageDefinition).delegation;
    const client = new delegationProto.DelegationService(
      `${GRPC_HOST}:${GRPC_PORT}`,
      grpc.credentials.createInsecure()
    );
    
    // Create a dummy delegation data (just for testing connectivity)
    // In a real scenario, this would be a valid delegation structure
    const mockDelegationData = Buffer.from('This is a test delegation');
    
    console.log('Sending test request to server...');
    
    // Call the gRPC method with a timeout
    client.redeemDelegation(
      { delegationData: mockDelegationData },
      { deadline: new Date(Date.now() + 5000) }, // 5 second timeout
      (err, response) => {
        if (err) {
          console.error('Error calling gRPC service:', err.message);
          process.exit(1);
        }
        
        console.log('Received response from server:');
        console.log(JSON.stringify(response, null, 2));
        
        // Check if the call was successful (the server handled the request)
        if (response.success) {
          console.log('✅ gRPC server is running and responsive!');
        } else {
          console.log('❌ Server returned an error:', response.errorMessage);
          console.log('This may be expected if you sent invalid delegation data');
        }
        
        process.exit(0);
      }
    );
  } catch (error) {
    console.error('Error in test script:', error);
    process.exit(1);
  }
}

// Run the main function
main().catch(err => {
  console.error('Unhandled error:', err);
  process.exit(1);
}); 