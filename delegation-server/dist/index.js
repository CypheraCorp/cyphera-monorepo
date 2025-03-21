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
const path_1 = __importDefault(require("path"));
const grpc = __importStar(require("@grpc/grpc-js"));
const protoLoader = __importStar(require("@grpc/proto-loader"));
const config_1 = require("./config/config");
const service_1 = require("./services/service");
/**
 * Main entry point for the delegation redemption gRPC server
 */
function startServer() {
    // Load the protobuf definition
    const PROTO_PATH = path_1.default.join(__dirname, 'proto/delegation.proto');
    const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true
    });
    const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
    const delegationPackage = protoDescriptor.delegation;
    // Create gRPC server
    const server = new grpc.Server();
    // Add the delegation service
    server.addService(delegationPackage.DelegationService.service, service_1.delegationService);
    // Start the server
    const serverAddress = `${config_1.config.grpc.host}:${config_1.config.grpc.port}`;
    server.bindAsync(serverAddress, grpc.ServerCredentials.createInsecure(), (err, port) => {
        if (err) {
            console.error('Failed to bind server:', err);
            process.exit(1);
        }
        server.start();
        console.log(`gRPC server started on ${serverAddress}`);
        console.log('Ready to receive delegation redemption requests from Go backend');
    });
    // Handle shutdown gracefully
    const shutdownGracefully = () => {
        console.log('Shutting down gRPC server gracefully...');
        server.tryShutdown(() => {
            console.log('Server shut down successfully');
            process.exit(0);
        });
    };
    process.on('SIGINT', shutdownGracefully);
    process.on('SIGTERM', shutdownGracefully);
}
// Start the server when this file is executed directly
if (require.main === module) {
    startServer();
}
