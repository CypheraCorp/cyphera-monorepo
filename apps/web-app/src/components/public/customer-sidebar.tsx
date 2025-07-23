'use client';

import { useState } from 'react';
import { motion } from 'framer-motion';
import { CreditCard, Wallet, LogOut, User, Settings, ShoppingBag } from 'lucide-react';
import Link from 'next/link';
import { Sidebar, SidebarBody, SidebarLink } from '@/components/ui/sidebar';
import Image from 'next/image';
import { useWeb3AuthDisconnect, useWeb3AuthUser, useWeb3Auth } from '@web3auth/modal/react';
import { logger } from '@/lib/core/logger/logger-utils';
// Logout flag to prevent auto-connect after logout
function setCustomerLogoutFlag() {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem('web3auth-customer-logout', 'true');
  }
}

// Safe Web3Auth hook that handles missing context
function useSafeCustomerWeb3AuthDisconnect() {
  const { disconnect } = useWeb3AuthDisconnect();
  return disconnect;
}

// Safe Web3Auth user hook
function useSafeCustomerWeb3AuthUser() {
  const { userInfo } = useWeb3AuthUser();
  const { isConnected } = useWeb3Auth();
  return { userInfo, isConnected };
}

export function CustomerSidebar() {
  const [open, setOpen] = useState(false);

  // Safe Web3Auth hooks
  const disconnect = useSafeCustomerWeb3AuthDisconnect();
  const { userInfo, isConnected } = useSafeCustomerWeb3AuthUser();

  const handleLogout = async () => {
    try {
      logger.log('🔄 Starting customer logout process...');
      setCustomerLogoutFlag();

      // 1. Call logout endpoint to clear server-side session
      await fetch('/api/auth/customer/logout', {
        method: 'POST',
        credentials: 'include',
      });
      logger.log('✅ Server session cleared');

      // 2. Disconnect from Web3Auth
      try {
        await disconnect();
        logger.log('✅ Web3Auth disconnected');
      } catch (web3AuthError) {
        logger.warn('⚠️ Web3Auth disconnect failed (may already be disconnected):', {
          error: web3AuthError,
        });
      }

      // 3. Clear any relevant local storage items
      if (typeof window !== 'undefined') {
        // Clear Web3Auth related items
        const keysToRemove = [
          'web3auth-customer-logout',
          'openlogin_store',
          'Web3Auth-cachedAdapter',
        ];

        keysToRemove.forEach((key) => {
          try {
            localStorage.removeItem(key);
            sessionStorage.removeItem(key);
          } catch (storageError) {
            logger.warn(`⚠️ Could not clear ${key}:`, { error: storageError });
          }
        });

        // Clear any cookies by setting them to expire
        document.cookie =
          'cyphera-customer-session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
        document.cookie = 'cyphera-session=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';

        logger.log('✅ Local storage and cookies cleared');
      }

      // 4. Force a small delay to ensure cleanup completes
      await new Promise((resolve) => setTimeout(resolve, 500));

      // 5. Redirect to lightweight customer signin page
      window.location.href = '/customers/signin';

      logger.log('✅ Customer logged out successfully');
    } catch (error) {
      logger.error('❌ Customer logout failed:', { error });
      // Even if logout fails, redirect to lightweight customer signin to clear the UI state
      window.location.href = '/customers/signin';
    }
  };

  const customerLinks = [
    {
      label: 'Dashboard',
      href: '/customers/dashboard',
      icon: <User className="text-neutral-700 dark:text-neutral-200 h-5 w-5 flex-shrink-0" />,
    },
    {
      label: 'Marketplace',
      href: '/customers/marketplace',
      icon: (
        <ShoppingBag className="text-neutral-700 dark:text-neutral-200 h-5 w-5 flex-shrink-0" />
      ),
    },
    {
      label: 'Subscriptions',
      href: '/customers/subscriptions',
      icon: <CreditCard className="text-neutral-700 dark:text-neutral-200 h-5 w-5 flex-shrink-0" />,
    },
    {
      label: 'My Wallet',
      href: '/customers/wallet',
      icon: <Wallet className="text-neutral-700 dark:text-neutral-200 h-5 w-5 flex-shrink-0" />,
    },
    {
      label: 'Settings',
      href: '/customers/settings',
      icon: <Settings className="text-neutral-700 dark:text-neutral-200 h-5 w-5 flex-shrink-0" />,
    },
  ];

  const logoutLink = {
    label: 'Logout',
    href: '#',
    icon: <LogOut className="text-neutral-700 dark:text-neutral-200 h-5 w-5 flex-shrink-0" />,
    onClick: handleLogout,
  };

  return (
    <div className="md:flex">
      <Sidebar open={open} setOpen={setOpen}>
        <SidebarBody className="justify-between gap-10">
          {/* Top Section - Logo and Navigation */}
          <div className="flex flex-col flex-1 overflow-y-auto overflow-x-hidden">
            {open ? <Logo /> : <LogoIcon />}
            <div className="mt-8 flex flex-col gap-2">
              {customerLinks.map((link, idx) => (
                <SidebarLink key={idx} link={link} />
              ))}
            </div>
          </div>

          {/* Bottom Section - Profile and Logout */}
          {isConnected && userInfo && (
            <div className="flex flex-col gap-2">
              {/* Customer Profile Section */}
              <div className="pt-4 border-t border-neutral-200 dark:border-neutral-700">
                <div
                  className={`flex items-center py-3 ${open ? 'gap-3 px-2' : 'justify-center px-0'}`}
                >
                  <div className="relative h-10 w-10 flex-shrink-0">
                    {userInfo.profileImage ? (
                      <Image
                        src={userInfo.profileImage}
                        className="rounded-full object-cover border-2 border-neutral-200 dark:border-neutral-700"
                        width={40}
                        height={40}
                        alt="Customer Avatar"
                        style={{ width: '40px', height: '40px' }}
                        onError={(e) => {
                          // Hide the image and show fallback if it fails to load
                          e.currentTarget.style.display = 'none';
                          const fallback = e.currentTarget.nextElementSibling as HTMLElement;
                          if (fallback) fallback.style.display = 'flex';
                        }}
                      />
                    ) : null}
                    {/* Fallback avatar - always present but hidden unless image fails */}
                    <div
                      className="h-10 w-10 rounded-full bg-neutral-100 dark:bg-neutral-800 flex items-center justify-center border-2 border-neutral-200 dark:border-neutral-700"
                      style={{ display: userInfo.profileImage ? 'none' : 'flex' }}
                    >
                      <User className="h-5 w-5 text-neutral-600 dark:text-neutral-400" />
                    </div>
                  </div>
                  {open && (
                    <div className="flex flex-col overflow-hidden min-w-0 flex-1">
                      <span className="text-sm font-medium text-neutral-700 dark:text-neutral-200 truncate">
                        {userInfo.name || 'Customer'}
                      </span>
                      <span className="text-xs text-neutral-500 dark:text-neutral-400 truncate">
                        {userInfo.email}
                      </span>
                    </div>
                  )}
                </div>
              </div>

              {/* Logout Link */}
              <div className="pb-2">
                <SidebarLink link={logoutLink} />
              </div>
            </div>
          )}
        </SidebarBody>
      </Sidebar>
    </div>
  );
}

export const Logo = () => {
  return (
    <Link
      href="/customers/dashboard"
      className="font-normal flex space-x-2 items-center text-sm text-black py-1 relative z-20"
    >
      <div className="h-5 w-6 bg-black dark:bg-white rounded-br-lg rounded-tr-sm rounded-tl-lg rounded-bl-sm flex-shrink-0" />
      <motion.span
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="font-medium text-black dark:text-white whitespace-pre"
      >
        Cyphera Customer
      </motion.span>
    </Link>
  );
};

export const LogoIcon = () => {
  return (
    <Link
      href="/customers/dashboard"
      className="font-normal flex space-x-2 items-center text-sm text-black py-1 relative z-20"
    >
      <div className="h-5 w-6 bg-black dark:bg-white rounded-br-lg rounded-tr-sm rounded-tl-lg rounded-bl-sm flex-shrink-0" />
    </Link>
  );
};
