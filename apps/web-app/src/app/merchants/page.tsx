'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function MerchantsPage() {
  const router = useRouter();

  useEffect(() => {
    // Redirect to merchant dashboard
    router.push('/merchants/dashboard');
  }, [router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-neutral-50 dark:bg-neutral-900">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
        <p className="text-muted-foreground">Redirecting to merchant dashboard...</p>
      </div>
    </div>
  );
}
