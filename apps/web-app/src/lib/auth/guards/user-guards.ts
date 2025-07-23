import type { CypheraUser } from '@/lib/auth/session/session';

/**
 * Type for user data with all required fields
 */
interface CypheraWeb3AuthUser extends CypheraUser {
  user_id: string;
  account_id: string;
  workspace_id: string;
  access_token: string;
}

/**
 * Type guard to check if a user has all required account information
 * @param user - The user object to check
 * @returns A boolean indicating if the user has all required fields
 */
export function hasRequiredAccountInfo(user: CypheraUser | null): boolean {
  if (!user) return false;
  return !!(user.user_id && user.account_id && user.workspace_id);
}

/**
 * Convert a CypheraUser to a CypheraWeb3AuthUser if it has all required fields
 * @param user - The user object to convert
 * @returns The converted user object or null if missing required fields
 */
export function asRequiredUser(user: CypheraUser | null): Partial<CypheraWeb3AuthUser> | null {
  if (!hasRequiredAccountInfo(user) || !user) return null;

  return {
    ...user,
    user_id: user.user_id as string,
    account_id: user.account_id as string,
    workspace_id: user.workspace_id as string,
    access_token: user.access_token || '',
  };
}

/**
 * Extract account info into a new object with required fields
 * @param user - The user object to extract from
 * @returns An object with account information
 */
export function getAccountInfo(user: CypheraUser | null) {
  if (!user) return null;

  return {
    userId: user.user_id,
    accountId: user.account_id,
    workspaceId: user.workspace_id,
    email: user.email,
    fullName: user.user_metadata?.full_name,
  };
}
