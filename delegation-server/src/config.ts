// config.ts
export interface Config {
  mockMode: boolean;
  serverAddress: string;
  // Add other configuration options as needed
}

function getEnvVar(name: string, defaultValue: string): string {
  return process.env[name] || defaultValue;
}

// Read environment variables or use defaults
const config: Config = {
  mockMode: process.env.MOCK_MODE === 'true',
  serverAddress: `${getEnvVar('GRPC_HOST', '0.0.0.0')}:${getEnvVar('GRPC_PORT', '50051')}`,
  // Add other configuration options as needed
};

export default config; 