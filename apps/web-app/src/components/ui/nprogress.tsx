'use client';

import { useEffect } from 'react';
import { usePathname, useSearchParams } from 'next/navigation';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';

// Configure NProgress
NProgress.configure({
  showSpinner: false,
  trickleSpeed: 200,
  minimum: 0.3,
});

// Custom CSS for NProgress
const styles = `
  #nprogress {
    pointer-events: none;
  }

  #nprogress .bar {
    background: hsl(var(--primary));
    position: fixed;
    z-index: 1031;
    top: 0;
    left: 0;
    width: 100%;
    height: 3px;
  }

  #nprogress .peg {
    display: block;
    position: absolute;
    right: 0px;
    width: 100px;
    height: 100%;
    box-shadow: 0 0 10px hsl(var(--primary)), 0 0 5px hsl(var(--primary));
    opacity: 1.0;
    transform: rotate(3deg) translate(0px, -4px);
  }
`;

export function NavigationProgress() {
  const pathname = usePathname();
  const searchParams = useSearchParams();

  useEffect(() => {
    NProgress.done();
    return () => {
      NProgress.done();
    };
  }, [pathname, searchParams]);

  return (
    <style jsx global>
      {styles}
    </style>
  );
}

// Utility functions for manual control
export const startProgress = () => NProgress.start();
export const stopProgress = () => NProgress.done();
export const setProgress = (value: number) => NProgress.set(value);
