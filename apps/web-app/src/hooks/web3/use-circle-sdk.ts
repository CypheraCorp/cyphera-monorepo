import { useCircleSDKContext } from '@/contexts/circle-sdk-provider';

/**
 * Hook to access Circle SDK functionality
 * @returns Circle SDK context with methods for initialization, authentication, and challenge execution
 */
export function useCircleSDK() {
  return useCircleSDKContext();
}
