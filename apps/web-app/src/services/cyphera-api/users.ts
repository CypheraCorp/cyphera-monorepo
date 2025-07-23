import { CypheraAPI, UserRequestContext } from './api';
import type { UserResponse } from '@/types/user';
import { logger } from '@/lib/core/logger/logger-utils';
/**
 * Users API class for handling user-related API requests
 * Extends the base CypheraAPI class (STATELESS regarding user)
 */
export class UsersAPI extends CypheraAPI {
  /**
   * Gets user information by Supabase ID
   * @param context - The user request context (token, IDs)
   * @param supabaseId - The Supabase ID of the user
   * @returns Promise with the user response
   * @throws Error if the request fails
   */
  async getUserBySupabaseId(
    context: UserRequestContext,
    supabaseId: string
  ): Promise<UserResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/users/supabase/${supabaseId}`, {
        method: 'GET',
        headers: this.getHeaders(context),
      });
      return await this.handleResponse<UserResponse>(response);
    } catch (error) {
      logger.error('User info fetch by Supabase ID failed:', error);
      throw error;
    }
  }
}
