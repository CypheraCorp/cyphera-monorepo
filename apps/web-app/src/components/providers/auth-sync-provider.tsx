'use client';

import { useEffect, ReactNode } from 'react';
import { useAuthStore } from '@/store/auth';
import { useAuth as useAuthContext } from '@/contexts/auth-context';

/**
 * Temporary provider that syncs AuthContext with Zustand store
 * This allows gradual migration while keeping both systems in sync
 */
export function AuthSyncProvider({ children }: { children: ReactNode }) {
  const contextAuth = useAuthContext();
  const setAuth = useAuthStore((state) => state.setAuth);

  // Sync context state to store whenever it changes
  useEffect(() => {
    setAuth({
      isAuthenticated: contextAuth.auth.isAuthenticated,
      user: contextAuth.auth.user,
      account: contextAuth.auth.account,
      workspace: contextAuth.auth.workspace,
      loading: contextAuth.loading,
      error: contextAuth.error,
    });
  }, [
    contextAuth.auth.isAuthenticated,
    contextAuth.auth.user,
    contextAuth.auth.account,
    contextAuth.auth.workspace,
    contextAuth.loading,
    contextAuth.error,
    setAuth,
  ]);

  return <>{children}</>;
}
