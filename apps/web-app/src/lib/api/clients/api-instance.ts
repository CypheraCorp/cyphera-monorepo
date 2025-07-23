import { CypheraAPIClient } from '@/services/cyphera-api';

/**
 * Singleton instance of the CypheraAPIClient (STATELESS version)
 * Provides a single instance primarily for accessing public/unauthenticated methods.
 * For authenticated requests, use getAPIContext() which creates a new instance.
 */
class CypheraAPIInstance {
  private static instance: CypheraAPIClient | null = null;

  /**
   * Get the singleton instance of CypheraAPIClient
   * Creates a new instance if one doesn't exist
   */
  public static getInstance(): CypheraAPIClient {
    if (!CypheraAPIInstance.instance) {
      CypheraAPIInstance.instance = new CypheraAPIClient();
    }
    return CypheraAPIInstance.instance;
  }
}

/**
 * Get the base, stateless, singleton API client instance.
 * Suitable for accessing public methods or methods not requiring user context.
 */
export function getBaseAPI(): CypheraAPIClient {
  return CypheraAPIInstance.getInstance();
}
