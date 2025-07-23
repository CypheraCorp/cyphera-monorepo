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

interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  account: Account | null;
  workspace: Workspace | null;
  loading: boolean;
  error: string | null;
}

interface AuthActions {
  refreshAuth: () => Promise<void>;
  setAuth: (auth: Partial<AuthState>) => void;
  logout: () => void;
  clearError: () => void;
}

export const useAuthStore = create<AuthState & AuthActions>()(
  devtools(
    persist(
      (set) => ({
        // State
        isAuthenticated: false,
        user: null,
        account: null,
        workspace: null,
        loading: true,
        error: null,

        // Actions
        refreshAuth: async () => {
          set({ loading: true, error: null });
          try {
            const response = await fetch('/api/auth/me');
            if (!response.ok) {
              throw new Error('Auth check failed');
            }
            const data = await response.json();
            set({
              isAuthenticated: true,
              user: data.user,
              account: data.account,
              workspace: data.workspace,
              loading: false,
            });
          } catch (err) {
            set({
              isAuthenticated: false,
              user: null,
              account: null,
              workspace: null,
              error: err instanceof Error ? err.message : 'Unknown error',
              loading: false,
            });
          }
        },

        setAuth: (auth) => set(auth),

        logout: () => {
          set({
            isAuthenticated: false,
            user: null,
            account: null,
            workspace: null,
            error: null,
          });
        },

        clearError: () => set({ error: null }),
      }),
      {
        name: 'auth-storage',
        partialize: (state) => ({
          isAuthenticated: state.isAuthenticated,
          user: state.user,
          account: state.account,
          workspace: state.workspace,
        }),
      }
    )
  )
);
