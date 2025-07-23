'use client';

import { useEffect, useState } from 'react';
import { Suspense } from 'react';
import dynamic from 'next/dynamic';
import { getUser } from '@/lib/auth/session/session-client';
import type { CypheraUser } from '@/lib/auth/session/session';
import { useRouter } from 'next/navigation';
import { clientLogger } from '@/lib/core/logger/logger-client';

// Dynamically import components to reduce initial bundle size

const SettingsForm = dynamic(
  () =>
    import('@/components/settings/settings-form').then((mod) => ({ default: mod.SettingsForm })),
  {
    loading: () => <div className="h-64 w-full bg-muted animate-pulse rounded-md" />,
    ssr: false,
  }
);

/**
 * Settings page component
 * Allows users to manage their account settings
 */
export default function SettingsPage() {
  const [user, setUser] = useState<CypheraUser | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    async function loadUser() {
      try {
        const currentUser = await getUser();
        if (!currentUser) {
          router.push('/merchants/signin');
          return;
        }
        setUser(currentUser);
      } catch (error) {
        clientLogger.error('Failed to load user', {
          error: error instanceof Error ? error.message : error,
        });
        router.push('/merchants/signin');
      } finally {
        setLoading(false);
      }
    }
    loadUser();
  }, [router]);

  if (loading) {
    return <div className="h-screen w-full bg-muted animate-pulse rounded-md" />;
  }

  if (!user) {
    return null;
  }

  return (
    <Suspense fallback={<div className="h-screen w-full bg-muted animate-pulse rounded-md" />}>
      <div className="container mx-auto py-6 px-4">
        <Suspense fallback={<div className="h-64 w-full bg-muted animate-pulse rounded-md" />}>
          <SettingsForm user={user} />
        </Suspense>
      </div>
    </Suspense>
  );
}
