import { useEffect } from 'react';
import { useAuthStore } from '@/store/auth';
import { useAuth as useAuthContext } from '@/contexts/auth-context';

/**
 * Migration hook that syncs AuthContext with Zustand store
 * This allows gradual migration by keeping both in sync
 */
export function useAuthMigration() {
  const contextAuth = useAuthContext();
  const setAuth = useAuthStore((state) => state.setAuth);
  const refreshAuth = useAuthStore((state) => state.refreshAuth);

  // Sync context state to store
  useEffect(() => {
    if (contextAuth.auth.isAuthenticated) {
      setAuth({
        isAuthenticated: contextAuth.auth.isAuthenticated,
        user: contextAuth.auth.user,
        account: contextAuth.auth.account,
        workspace: contextAuth.auth.workspace,
        loading: contextAuth.loading,
        error: contextAuth.error,
      });
    }
  }, [contextAuth, setAuth]);

  // Initial auth check using store
  useEffect(() => {
    refreshAuth();
  }, [refreshAuth]);
}

/**
 * Drop-in replacement for useAuth context hook
 * Uses Zustand store instead of context
 */
export function useAuth() {
  const auth = useAuthStore((state) => ({
    isAuthenticated: state.isAuthenticated,
    user: state.user,
    account: state.account,
    workspace: state.workspace,
  }));

  const loading = useAuthStore((state) => state.loading);
  const error = useAuthStore((state) => state.error);
  const refreshAuth = useAuthStore((state) => state.refreshAuth);

  return {
    auth,
    loading,
    error,
    refreshAuth,
  };
}
