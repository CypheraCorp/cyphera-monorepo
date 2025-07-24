'use client';

import { useAuth } from '@/hooks/auth/use-auth-user';
import { SettingsForm } from '@/components/settings/settings-form';
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';

/**
 * Settings page component
 * Allows users to manage their account settings
 */
export default function SettingsPage() {
  const { user, loading, error, isAuthenticated } = useAuth();
  
  // Debug logging
  console.log('[Settings Page] Auth state:', {
    isAuthenticated,
    loading,
    user,
    error
  });

  if (loading) {
    return (
      <div className="container mx-auto py-6 px-4">
        <div className="h-64 w-full bg-muted animate-pulse rounded-md" />
      </div>
    );
  }

  if (!user) {
    return (
      <div className="container mx-auto py-6 px-4">
        <Card>
          <CardHeader>
            <CardTitle>Authentication Required</CardTitle>
            <CardDescription>
              {error ? `Error: ${error}` : 'Please sign in to access your settings.'}
              <br />
              <span className="text-xs text-muted-foreground">
                Auth status: {isAuthenticated ? 'Authenticated' : 'Not authenticated'}
              </span>
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-6 px-4">
      <SettingsForm user={user} />
    </div>
  );
}