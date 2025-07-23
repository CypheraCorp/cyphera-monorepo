import { Metadata } from 'next';

// Force dynamic rendering to prevent caching
export const dynamic = 'force-dynamic';
export const revalidate = 0;

export const metadata: Metadata = {
  title: 'Subscribe - Cyphera',
  description: 'Subscribe to products on Cyphera',
};

export default function PublicLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-neutral-50 dark:bg-neutral-900">
      <main className="flex-1">{children}</main>
    </div>
  );
}
