"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.startServer = startServer;
const grpc = __importStar(require("@grpc/grpc-js"));
const path = __importStar(require("path"));
const protoLoader = __importStar(require("@grpc/proto-loader"));
const service_1 = require("./services/service");
const utils_1 = require("./utils/utils");
const config_1 = __importDefault(require("./config"));
// Debug environment variables on startup
function logEnvironmentVariables() {
    const envVars = { ...process.env };
    // Redact sensitive information
    if (envVars.PRIVATE_KEY) {
        envVars.PRIVATE_KEY = envVars.PRIVATE_KEY.substring(0, 6) + '...[REDACTED]';
    }
    utils_1.logger.info('===== ENVIRONMENT VARIABLES =====');
    utils_1.logger.info(`MOCK_MODE: ${envVars.MOCK_MODE || 'not set'}`);
    utils_1.logger.info(`GRPC_HOST: ${envVars.GRPC_HOST || 'not set'}`);
    utils_1.logger.info(`GRPC_PORT: ${envVars.GRPC_PORT || 'not set'}`);
    utils_1.logger.info(`RPC_URL: ${envVars.RPC_URL || 'not set'}`);
    utils_1.logger.info(`BUNDLER_URL: ${envVars.BUNDLER_URL || 'not set'}`);
    utils_1.logger.info(`CHAIN_ID: ${envVars.CHAIN_ID || 'not set'}`);
    utils_1.logger.info(`PRIVATE_KEY: ${envVars.PRIVATE_KEY || 'not set'}`);
    utils_1.logger.info(`LOG_LEVEL: ${envVars.LOG_LEVEL || 'not set'}`);
    utils_1.logger.info('==================================');
    // Log loaded configuration
    utils_1.logger.info('Config loaded:');
    utils_1.logger.info(`mockMode: ${config_1.default.mockMode}`);
    utils_1.logger.info(`serverAddress: ${config_1.default.serverAddress}`);
}
// Load the protobuf definition
const PROTO_PATH = path.join(__dirname, 'proto/delegation.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true
});
const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const delegationPackage = protoDescriptor.delegation;
// Create a gRPC server
const server = new grpc.Server();
// Register the service
server.addService(delegationPackage.DelegationService.service, service_1.delegationService);
// Function to start the server - exported for backward compatibility
function startServer() {
    // Bind and start the server
    server.bindAsync(config_1.default.serverAddress, grpc.ServerCredentials.createInsecure(), (err, port) => {
        if (err) {
            utils_1.logger.error(`Failed to bind server: ${err}`);
            return;
        }
        server.start();
        utils_1.logger.info(`Delegation server running on ${config_1.default.serverAddress}`);
    });
    // Handle graceful shutdown
    process.on('SIGINT', () => {
        utils_1.logger.info('Received SIGINT. Shutting down gracefully...');
        server.tryShutdown(() => {
            utils_1.logger.info('Server shut down successfully');
            process.exit(0);
        });
    });
}
// Start the server directly when the file is executed
if (require.main === module) {
    logEnvironmentVariables();
    startServer();
}
