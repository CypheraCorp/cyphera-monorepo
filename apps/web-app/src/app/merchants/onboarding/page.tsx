'use client';

import dynamic from 'next/dynamic';
import { Skeleton } from '@/components/ui/skeleton';

const OnboardingForm = dynamic(
  () => import('@/components/onboarding/onboarding-form').then((mod) => mod.OnboardingForm),
  {
    loading: () => (
      <div className="space-y-6">
        <Skeleton className="h-8 w-3/4" />
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-10 w-32" />
      </div>
    ),
    ssr: false,
  }
);

export default function OnboardingPage() {
  return (
    <div className="container mx-auto py-8 px-4">
      <OnboardingForm />
    </div>
  );
}
