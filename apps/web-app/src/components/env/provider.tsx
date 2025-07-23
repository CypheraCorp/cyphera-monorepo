import EnvConfigClient from './client';
import React from 'react';

// This is a server component (no "use client" directive)
export default async function EnvConfigProvider({ children }: { children: React.ReactNode }) {
  // TODO: Get authentication status from Web3Auth
  const isAuthenticated = false; // Will be determined by Web3Auth client-side

  // Access all environment variables securely
  // These can be either NEXT_PUBLIC_ prefixed or private variables
  const config = {
    // Web3Auth
    web3AuthClientId: process.env.NEXT_PUBLIC_WEB3AUTH_CLIENT_ID || '',

    // Web3/Blockchain
    usdcContractAddress: process.env.NEXT_PUBLIC_USDC_CONTRACT_ADDRESS || '',
    delegateAddress: process.env.CYPHERA_DELEGATE_ADDRESS || '',

    // Circle
    circleAppId: process.env.NEXT_PUBLIC_CIRCLE_APP_ID || '',
    circleApiUrl: process.env.CIRCLE_API_URL || '',

    // Maps
    googleMapsApiKey: process.env.GOOGLE_MAPS_API_KEY || '',

    // API
    apiEndpoint: process.env.CYPHERA_API_BASE_URL || '',

    // Auth
    isAuthenticated,
  };

  return <EnvConfigClient config={config}>{children}</EnvConfigClient>;
}
