"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.config = void 0;
exports.validateConfig = validateConfig;
const dotenv_1 = __importDefault(require("dotenv"));
const path_1 = require("path");
// Load environment variables from .env file
dotenv_1.default.config({ path: (0, path_1.resolve)(__dirname, '../../.env') });
// Configuration object with all environment variables
exports.config = {
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
};
// Validate required configuration
function validateConfig() {
    const requiredVars = [
        { key: 'blockchain.rpcUrl', value: exports.config.blockchain.rpcUrl },
        { key: 'blockchain.bundlerUrl', value: exports.config.blockchain.bundlerUrl },
        { key: 'blockchain.privateKey', value: exports.config.blockchain.privateKey },
    ];
    const missingVars = requiredVars.filter(v => !v.value);
    if (missingVars.length > 0) {
        const missingKeys = missingVars.map(v => v.key).join(', ');
        throw new Error(`Missing required environment variables: ${missingKeys}`);
    }
    // Validate private key format
    if (exports.config.blockchain.privateKey) {
        const pkRegex = /^0x[0-9a-fA-F]{64}$/;
        if (!pkRegex.test(exports.config.blockchain.privateKey)) {
            throw new Error('PRIVATE_KEY must be a valid 32-byte hex string with 0x prefix (66 characters total)');
        }
    }
}
