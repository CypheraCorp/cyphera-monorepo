import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

interface User {
  id: string;
  email: string;
}

interface Account {
  id: string;
}

interface Workspace {
  id: string;
}

export interface AuthState {
  // Auth state (persisted)
  isAuthenticated: boolean;
  accessToken: string | null;
  refreshToken: string | null;
  
  // User data (fetched fresh, not persisted)
  user: User | null;
  account: Account | null;
  workspace: Workspace | null;
  
  // UI state
  loading: boolean;
  error: string | null;
  hasHydrated: boolean;
}

export interface AuthActions {
  // Auth actions
  setTokens: (accessToken: string, refreshToken?: string) => void;
  setUserData: (user: User, account: Account, workspace: Workspace) => void;
  clearAuth: () => void;
  
  // UI actions
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setHasHydrated: (hydrated: boolean) => void;
  
  // Combined actions
  login: (accessToken: string, refreshToken: string, user: User, account: Account, workspace: Workspace) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState & AuthActions>()(
  devtools(
    persist(
      (set) => ({
        // State
        isAuthenticated: false,
        accessToken: null,
        refreshToken: null,
        user: null,
        account: null,
        workspace: null,
        loading: false,
        error: null,
        hasHydrated: false,

        // Actions
        setTokens: (accessToken, refreshToken) => 
          set({ 
            accessToken, 
            refreshToken: refreshToken || null,
            isAuthenticated: true 
          }),
          
        setUserData: (user, account, workspace) => 
          set({ user, account, workspace }),
          
        clearAuth: () => 
          set({
            isAuthenticated: false,
            accessToken: null,
            refreshToken: null,
            user: null,
            account: null,
            workspace: null,
            error: null,
          }),

        setLoading: (loading) => set({ loading }),
        setError: (error) => set({ error }),
        setHasHydrated: (hasHydrated) => set({ hasHydrated }),

        login: (accessToken, refreshToken, user, account, workspace) => 
          set({
            isAuthenticated: true,
            accessToken,
            refreshToken,
            user,
            account,
            workspace,
            error: null,
          }),

        logout: () => {
          // Clear auth state
          set({
            isAuthenticated: false,
            accessToken: null,
            refreshToken: null,
            user: null,
            account: null,
            workspace: null,
            error: null,
          });
          
          // Clear any cached data
          if (typeof window !== 'undefined') {
            // This will trigger React Query to refetch
            window.dispatchEvent(new Event('auth-logout'));
          }
        },
      }),
      {
        name: 'auth-storage',
        // Only persist tokens and auth state, not user data
        partialize: (state) => ({
          isAuthenticated: state.isAuthenticated,
          accessToken: state.accessToken,
          refreshToken: state.refreshToken,
        }),
        onRehydrateStorage: () => (state) => {
          state?.setHasHydrated(true);
        },
      }
    ),
    {
      name: 'auth-store',
    }
  )
);
