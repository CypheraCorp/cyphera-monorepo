import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { Toaster } from '@/components/ui/toaster';
import { Web3Provider } from '@/components/providers/web3-provider';
import EnvProvider from '@/components/env/provider';
import { CircleSDKProvider } from '@/contexts/circle-sdk-provider';
import { NavigationProgress } from '@/components/ui/nprogress';
import { ServiceWorkerProvider } from '@/components/providers/service-worker-provider';
import { Suspense } from 'react';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'Cyphera',
  description: 'Web3 Payment Infrastructure',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <ServiceWorkerProvider>
          <Suspense fallback={null}>
            <NavigationProgress />
          </Suspense>
          <EnvProvider>
            <Web3Provider>
              <CircleSDKProvider>
                {children}
                <Toaster />
              </CircleSDKProvider>
            </Web3Provider>
          </EnvProvider>
        </ServiceWorkerProvider>
      </body>
    </html>
  );
}
