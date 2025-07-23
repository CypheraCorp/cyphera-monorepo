import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/store/auth';

interface AuthUserResponse {
  user: {
    id: string;
    email: string;
  };
  account: {
    id: string;
  };
  workspace: {
    id: string;
  };
}

/**
 * Hook to fetch and keep user data fresh
 * This replaces the old pattern of storing user data in context/store
 */
export function useAuthUser() {
  const { isAuthenticated, accessToken, setUserData, setLoading, setError, clearAuth } = useAuthStore();
  
  const query = useQuery({
    queryKey: ['auth', 'user'],
    queryFn: async (): Promise<AuthUserResponse> => {
      const response = await fetch('/api/auth/me', {
        headers: {
          'Authorization': `Bearer ${accessToken}`,
        },
      });
      
      if (!response.ok) {
        if (response.status === 401) {
          // Token expired or invalid
          clearAuth();
          throw new Error('Authentication expired');
        }
        throw new Error('Failed to fetch user data');
      }
      
      return response.json();
    },
    enabled: isAuthenticated && !!accessToken,
    staleTime: 0, // Always consider stale to ensure fresh data
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    retry: (failureCount, error) => {
      // Don't retry on auth errors
      if (error.message === 'Authentication expired') {
        return false;
      }
      return failureCount < 3;
    },
  });
  
  // Update store with fresh data when query succeeds
  useQuery({
    queryKey: ['auth', 'user', 'sync'],
    queryFn: async () => {
      if (query.data) {
        setUserData(query.data.user, query.data.account, query.data.workspace);
      }
      return null;
    },
    enabled: !!query.data,
  });
  
  return {
    user: query.data?.user || null,
    account: query.data?.account || null,
    workspace: query.data?.workspace || null,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
  };
}

/**
 * Combined hook that provides both auth state and fresh user data
 * Drop-in replacement for useAuth() from context
 */
export function useAuth() {
  const authStore = useAuthStore();
  const userData = useAuthUser();
  
  return {
    // Auth state
    isAuthenticated: authStore.isAuthenticated,
    accessToken: authStore.accessToken,
    hasHydrated: authStore.hasHydrated,
    
    // User data (always fresh from server)
    user: userData.user,
    account: userData.account,
    workspace: userData.workspace,
    
    // Loading states
    loading: authStore.loading || userData.isLoading,
    error: authStore.error || userData.error?.message || null,
    
    // Actions
    login: authStore.login,
    logout: authStore.logout,
    refetch: userData.refetch,
  };
}