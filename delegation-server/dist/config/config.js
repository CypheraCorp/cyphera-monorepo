"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.config = void 0;
const dotenv_1 = __importDefault(require("dotenv"));
const utils_1 = require("../utils/utils");
// Load environment variables
dotenv_1.default.config();
// Required environment variables
const requiredEnvVars = [
    'RPC_URL',
    'BUNDLER_URL',
    'PRIVATE_KEY'
];
// Check for required environment variables
for (const envVar of requiredEnvVars) {
    if (!process.env[envVar]) {
        console.error(`Error: Environment variable ${envVar} is required but not set`);
        process.exit(1);
    }
}
// Server configuration
exports.config = {
    // gRPC server config
    grpc: {
        port: process.env.GRPC_PORT || '50051',
        host: process.env.GRPC_HOST || '0.0.0.0'
    },
    // Blockchain config
    blockchain: {
        rpcUrl: process.env.RPC_URL,
        bundlerUrl: process.env.BUNDLER_URL,
        privateKey: (0, utils_1.formatPrivateKey)(process.env.PRIVATE_KEY),
        paymasterUrl: process.env.PAYMASTER_URL || '',
        chainId: parseInt(process.env.CHAIN_ID || '11155111') // Default to Sepolia
    }
};
