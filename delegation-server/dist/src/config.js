"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
function getEnvVar(name, defaultValue) {
    return process.env[name] || defaultValue;
}
// Read environment variables or use defaults
const config = {
    mockMode: process.env.MOCK_MODE === 'true',
    serverAddress: `${getEnvVar('GRPC_HOST', '0.0.0.0')}:${getEnvVar('GRPC_PORT', '50051')}`,
    // Add other configuration options as needed
};
exports.default = config;
