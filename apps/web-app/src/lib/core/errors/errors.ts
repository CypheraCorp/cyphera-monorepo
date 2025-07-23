/**
 * Custom error class for authentication errors
 */
export class AuthError extends Error {
  constructor(
    message: string,
    public code?: string
  ) {
    super(message);
    this.name = 'AuthError';
  }
}

/**
 * Custom error for missing account setup
 */
export class AccountSetupError extends AuthError {
  constructor(message = 'Account setup is incomplete') {
    super(message, 'account_setup_incomplete');
    this.name = 'AccountSetupError';
  }
}

/**
 * Formats an authentication error for display
 */
export function formatAuthError(error: unknown): string {
  if (error instanceof AuthError) {
    return error.message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return 'An unknown authentication error occurred';
}

/**
 * Map of common error messages and their user-friendly versions
 */
export const AUTH_ERROR_MESSAGES = {
  account_setup_incomplete: 'Your account setup is incomplete. Please try signing out and back in.',
  not_authenticated: 'You must be logged in to access this resource.',
  credentials_invalid: 'Invalid email or password.',
  unauthorized: 'You are not authorized to access this resource.',
  session_expired: 'Your session has expired. Please sign in again.',
} as const;
