'use client';

import { Button } from '@/components/ui/button';
import { PackageX } from 'lucide-react';
import { useRouter } from 'next/navigation';

export default function NotFound() {
  const router = useRouter();

  return (
    <div className="max-w-4xl mx-auto">
      <div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
        <PackageX className="h-12 w-12 text-muted-foreground mb-4" />
        <h2 className="text-2xl font-bold mb-2">Product Not Found</h2>
        <p className="text-muted-foreground mb-6 max-w-md">
          The product you&apos;re looking for doesn&apos;t exist or is no longer available.
        </p>
        <Button onClick={() => router.push('/')}>Return Home</Button>
      </div>
    </div>
  );
}
