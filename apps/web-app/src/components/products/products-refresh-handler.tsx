'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export function ProductsRefreshHandler() {
  const router = useRouter();

  useEffect(() => {
    // Force a refresh when the component mounts
    router.refresh();
  }, [router]); // Add router to dependency array

  return null;
}
