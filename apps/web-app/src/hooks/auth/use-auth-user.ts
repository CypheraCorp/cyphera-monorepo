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
  session?: {
    user_id?: string;
    account_id?: string;
    workspace_id?: string;
    email?: string;
    access_token?: string;
  };
}

/**
 * Hook to fetch and keep user data fresh
 * This replaces the old pattern of storing user data in context/store
 */
export function useAuthUser() {
  const { isAuthenticated, accessToken, setUserData, clearAuth } = useAuthStore();
  
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
    data: query.data,
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
  const { data, isLoading, error, refetch } = useAuthUser();
  
  // Debug logging
  console.log('[useAuth] Hook state:', {
    isAuthenticated: authStore.isAuthenticated,
    hasAccessToken: !!authStore.accessToken,
    hasData: !!data,
    isLoading,
    error: error?.message
  });
  
  // Debug log the data structure
  if (data) {
    console.log('[useAuth] API Response data:', data);
  }
  
  // Create a properly formatted user object that matches CypheraUser interface
  const user = data ? {
    id: data.session?.user_id || data.user?.id || '',
    email: data.session?.email || data.user?.email || '',
    user_id: data.session?.user_id,
    account_id: data.session?.account_id || data.account?.id,
    workspace_id: data.session?.workspace_id || data.workspace?.id,
    access_token: data.session?.access_token || authStore.accessToken || '',
    // Additional fields that might be needed
    finished_onboarding: true,
    email_verified: true,
  } : null;
  
  return {
    // Auth state
    isAuthenticated: authStore.isAuthenticated,
    accessToken: authStore.accessToken,
    hasHydrated: authStore.hasHydrated,
    
    // User data (always fresh from server, formatted as CypheraUser)
    user,
    account: data?.account || null,
    workspace: data?.workspace || null,
    
    // Loading states
    loading: authStore.loading || isLoading,
    error: authStore.error || error?.message || null,
    
    // Actions
    login: authStore.login,
    logout: authStore.logout,
    refetch,
  };
}