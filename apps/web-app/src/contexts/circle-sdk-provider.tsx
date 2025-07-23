'use client';

import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import type { W3SSdk } from '@circle-fin/w3s-pw-web-sdk';
import { logger } from '@/lib/core/logger/logger-utils';

// Type for Circle SDK challenge execution result
interface ChallengeExecutionResult {
  challengeId?: string;
  status?: string;
  type?: string;
  error?: {
    code?: string;
    message?: string;
  };
  [key: string]: unknown;
}

interface CircleSDKContext {
  client: W3SSdk | null;
  isInitialized: boolean;
  isAuthenticated: boolean;
  userToken: string | null;
  initializeSDK: () => Promise<void>;
  authenticateUser: (userToken: string, encryptionKey: string) => Promise<boolean>;
  executeChallenge: (challengeId: string) => Promise<ChallengeExecutionResult>;
}

// Create context with initial empty values
export const CircleSDKContext = createContext<CircleSDKContext | null>(null);

// Use this hook to access the Circle SDK context
export function useCircleSDKContext() {
  const context = useContext(CircleSDKContext);
  if (!context) {
    throw new Error('useCircleSDKContext must be used within a CircleSDKProvider');
  }
  return context;
}

export interface CircleSDKProviderProps {
  children: ReactNode;
}

/**
 * CircleSDKProvider
 * Provides the Circle Web SDK client and authentication methods to child components
 */
export function CircleSDKProvider({ children }: CircleSDKProviderProps) {
  const [client, setClient] = useState<W3SSdk | null>(null);
  const [isInitialized, setIsInitialized] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [userToken, setUserToken] = useState<string | null>(null);

  // Initialize the SDK on the client side
  useEffect(() => {
    if (typeof window !== 'undefined' && !isInitialized) {
      const initSDK = async () => {
        try {
          await initializeSDK();
        } catch (error) {
          logger.error('Failed to initialize Circle SDK:', error);
        }
      };

      initSDK();
    }
  }, [isInitialized]);

  // Initialize the SDK and create a client instance
  const initializeSDK = async (): Promise<void> => {
    try {
      // Make sure we're on the client side
      if (typeof window === 'undefined') {
        logger.warn('Cannot initialize Circle SDK server-side');
        return;
      }

      // Dynamically import the SDK only on client side
      const { W3SSdk } = await import('@circle-fin/w3s-pw-web-sdk');

      const sdkClient = new W3SSdk();

      // Set app settings with app ID from our environment config
      sdkClient.setAppSettings({
        appId: process.env.NEXT_PUBLIC_CIRCLE_APP_ID || '',
      });

      // Customize UI resources if needed
      sdkClient.setResources({
        fontFamily: {
          name: 'Inter',
          url: 'https://fonts.cdnfonts.com/css/inter',
        },
      });

      setClient(sdkClient);
      setIsInitialized(true);
    } catch (error) {
      logger.error('Error initializing Circle SDK:', error);
      setIsInitialized(false);
      throw error;
    }
  };

  // Authenticate a user with a user token and encryption key
  const authenticateUser = async (userToken: string, encryptionKey: string): Promise<boolean> => {
    if (!client) {
      logger.error('Circle SDK client not initialized');
      return false;
    }

    try {
      // Authenticate the user with the provided token and encryption key
      client.setAuthentication({
        userToken,
        encryptionKey,
      });

      setUserToken(userToken);
      setIsAuthenticated(true);
      return true;
    } catch (error) {
      logger.error('Error authenticating user with Circle SDK:', error);
      setIsAuthenticated(false);
      setUserToken(null);
      return false;
    }
  };

  // Execute a challenge (like PIN setup)
  const executeChallenge = async (challengeId: string): Promise<ChallengeExecutionResult> => {
    if (!client) {
      logger.error('Circle SDK client not initialized');
      throw new Error('Circle SDK client not initialized');
    }

    if (!isAuthenticated) {
      logger.error('User not authenticated');
      throw new Error('User not authenticated');
    }

    try {
      // Execute the challenge with the given ID
      return new Promise((resolve, reject) => {
        client.execute(challengeId, (error, result) => {
          if (error) {
            logger.error('Error executing challenge:', { error });
            reject(error);
          } else if (result) {
            resolve(result as ChallengeExecutionResult);
          } else {
            reject(new Error('Challenge execution returned no result'));
          }
        });
      });
    } catch (error) {
      logger.error('Error executing challenge:', { error });
      throw error;
    }
  };

  const value: CircleSDKContext = {
    client,
    isInitialized,
    isAuthenticated,
    userToken,
    initializeSDK,
    authenticateUser,
    executeChallenge,
  };

  return <CircleSDKContext.Provider value={value}>{children}</CircleSDKContext.Provider>;
}
