// Development authentication bypass utility
// This should only be used in development environments

export function isDevAuthBypassAvailable(): boolean {
  return process.env.NODE_ENV === 'development' && process.env.NEXT_PUBLIC_DEV_AUTH_BYPASS === 'true';
}

export function isDevAuthEnabled(): boolean {
  return isDevAuthBypassAvailable();
}

export function getDevAuthUser() {
  if (!isDevAuthEnabled()) {
    return null;
  }

  // Return a mock user for development
  return {
    id: 'dev-user-123',
    email: 'dev@example.com',
    name: 'Dev User',
    role: 'merchant',
  };
}

export const DEV_TEST_USERS = {
  merchant: {
    id: 'dev-merchant-123',
    email: 'merchant@dev.test',
    name: 'Test Merchant',
    role: 'merchant',
    smartAccountAddress: '0x1234567890123456789012345678901234567890',
  },
  customer: {
    id: 'dev-customer-123',
    email: 'customer@dev.test',
    name: 'Test Customer',
    role: 'customer',
    smartAccountAddress: '0x0987654321098765432109876543210987654321',
  },
};

export function createDevSessionToken(userType: 'merchant' | 'customer'): string {
  // Create a mock session token for development
  const user = DEV_TEST_USERS[userType];
  const payload = {
    userId: user.id,
    exp: Date.now() + 24 * 60 * 60 * 1000, // 24 hours
    iat: Date.now(),
  };
  return Buffer.from(JSON.stringify(payload)).toString('base64');
}

export function logDevBypassWarning(context: string): void {
  console.warn(`[DEV AUTH BYPASS] ${context} - This should not be used in production!`);
}