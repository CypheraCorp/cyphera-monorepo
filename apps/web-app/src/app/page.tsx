'use client';

import { Suspense } from 'react';
import dynamic from 'next/dynamic';

// Dynamically import RoleSwitcher to reduce initial bundle size
const RoleSwitcher = dynamic(
  () => import('@/components/auth/role-switcher').then((mod) => ({ default: mod.RoleSwitcher })),
  {
    loading: () => (
      <div className="max-w-4xl mx-auto">
        <div className="grid md:grid-cols-2 gap-6">
          <div className="h-64 bg-white dark:bg-gray-800 rounded-lg border animate-pulse" />
          <div className="h-64 bg-white dark:bg-gray-800 rounded-lg border animate-pulse" />
        </div>
      </div>
    ),
    ssr: false,
  }
);

export default function HomePage() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-purple-50 to-cyan-50 dark:from-gray-900 dark:via-blue-900 dark:to-purple-900">
      <div className="container mx-auto px-4 py-16">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-4">
            Welcome to Cyphera
          </h1>
          <p className="text-xl text-gray-600 dark:text-gray-300 max-w-2xl mx-auto">
            Your Web3 payment infrastructure platform. Choose your role to get started.
          </p>
        </div>

        <div className="max-w-4xl mx-auto">
          <Suspense
            fallback={
              <div className="grid md:grid-cols-2 gap-6">
                <div className="h-64 bg-white dark:bg-gray-800 rounded-lg border animate-pulse" />
                <div className="h-64 bg-white dark:bg-gray-800 rounded-lg border animate-pulse" />
              </div>
            }
          >
            <RoleSwitcher />
          </Suspense>
        </div>
      </div>
    </div>
  );
}
