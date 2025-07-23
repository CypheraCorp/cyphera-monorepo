// Legacy exports (kept for backward compatibility)
export type { CypheraUser, CypheraSession, UserProfileResponse } from './session';
export { getSession as getLegacySession, getUser, isSessionValid, clearSession } from './session';
export { getSessionFromCookie, isAuthenticated, clearClientSession } from './session-client';

// Unified Session Service (New API)
// Server-side exports
export {
  UnifiedSessionService,
  type Session,
  type MerchantSession,
  type CustomerSession,
  type UserType,
  isMerchantSession,
  isCustomerSession,
  requireMerchantSession,
  requireCustomerSession,
  requireSession,
} from './unified-session';

// Client-side exports
export {
  UnifiedSessionClient,
  getSession,
  getMerchantSession,
  getCustomerSession,
  clearMerchantSession,
  clearCustomerSession,
  clearAllSessions,
  hasBothSessions,
  getAllSessions,
} from './unified-session-client';
