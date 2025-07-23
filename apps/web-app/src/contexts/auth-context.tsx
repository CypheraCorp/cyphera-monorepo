import { createContext, useContext, useState, useEffect, ReactNode } from 'react';

interface AuthState {
  isAuthenticated: boolean;
  user: { id: string; email: string } | null;
  account: { id: string } | null;
  workspace: { id: string } | null;
}

interface AuthContextType {
  auth: AuthState;
  loading: boolean;
  error: string | null;
  refreshAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [auth, setAuth] = useState<AuthState>({
    isAuthenticated: false,
    user: null,
    account: null,
    workspace: null,
  });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchAuth = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/auth/me');
      if (!response.ok) {
        throw new Error('Auth check failed');
      }
      const data = await response.json();
      setAuth({
        isAuthenticated: true,
        user: data.user,
        account: data.account,
        workspace: data.workspace,
      });
    } catch (err) {
      setAuth({
        isAuthenticated: false,
        user: null,
        account: null,
        workspace: null,
      });
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAuth();
  }, []);

  return (
    <AuthContext.Provider value={{ auth, loading, error, refreshAuth: fetchAuth }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (undefined === context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
