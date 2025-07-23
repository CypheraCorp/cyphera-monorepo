'use client';

import { CustomerWalletDropdown } from './customer-wallet-dropdown';

interface CustomerHeaderProps {
  title?: string;
  subtitle?: string;
  className?: string;
}

export function CustomerHeader({ title, subtitle, className = '' }: CustomerHeaderProps) {
  return (
    <header
      className={`bg-white dark:bg-neutral-900 border-b border-neutral-200 dark:border-neutral-700 px-6 py-4 ${className}`}
    >
      <div className="flex items-center justify-between">
        {/* Title Section */}
        <div className="flex flex-col">
          {title && (
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-white">{title}</h1>
          )}
          {subtitle && <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{subtitle}</p>}
        </div>

        {/* Wallet Dropdown */}
        <div className="flex items-center gap-4">
          <CustomerWalletDropdown />
        </div>
      </div>
    </header>
  );
}
