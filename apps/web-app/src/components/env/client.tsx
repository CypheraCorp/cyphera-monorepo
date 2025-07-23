'use client';

import { createContext, useContext, ReactNode } from 'react';

// Define the shape of our config context
interface PublicConfig {
  // Web3Auth
  web3AuthClientId: string;

  // Web3/Blockchain
  usdcContractAddress: string;
  delegateAddress: string;

  // Circle
  circleAppId: string;
  circleApiUrl: string;

  // Maps
  googleMapsApiKey: string;

  // Auth and session
  isAuthenticated: boolean;
  apiEndpoint?: string;
}

// Create context with a default empty state
const EnvConfigContext = createContext<PublicConfig>({
  // Web3Auth
  web3AuthClientId: '',

  // Web3/Blockchain
  usdcContractAddress: '',
  delegateAddress: '',

  // Circle
  circleAppId: '',
  circleApiUrl: '',

  // Maps
  googleMapsApiKey: '',

  // Auth and session
  isAuthenticated: false,
  apiEndpoint: '',
});

// Custom hook to access the config
export function useEnvConfig() {
  return useContext(EnvConfigContext);
}

// Client component that receives config from server and provides it to children
export default function EnvConfigClient({
  config,
  children,
}: {
  config: PublicConfig;
  children?: ReactNode;
}) {
  return <EnvConfigContext.Provider value={config}>{children}</EnvConfigContext.Provider>;
}
