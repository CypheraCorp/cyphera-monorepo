import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/store/auth';
import { useAuthUser } from './use-auth-user';

/**
 * Hook that ensures auth data is fresh on component mount
 * Use this in root layouts to trigger auth refresh
 */
export function useAuthRefresh() {
  const { isAuthenticated, hasHydrated, accessToken } = useAuthStore();
  const { refetch } = useAuthUser();
  
  useEffect(() => {
    // After hydration, if we have a token, fetch fresh user data
    if (hasHydrated && isAuthenticated && accessToken) {
      refetch();
    }
  }, [hasHydrated, isAuthenticated, accessToken, refetch]);
}

/**
 * Hook for handling auth state changes
 * Clears React Query cache on logout
 */
export function useAuthStateListener() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const queryClient = useQueryClient();
  
  useEffect(() => {
    const handleAuthLogout = () => {
      // Clear all React Query caches
      queryClient.clear();
    };
    
    // Listen for auth logout event
    window.addEventListener('auth-logout', handleAuthLogout);
    
    return () => {
      window.removeEventListener('auth-logout', handleAuthLogout);
    };
  }, [queryClient]);
  
  // Clear cache when auth state changes to logged out
  useEffect(() => {
    if (!isAuthenticated) {
      queryClient.clear();
    }
  }, [isAuthenticated, queryClient]);
}