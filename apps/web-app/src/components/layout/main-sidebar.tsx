'use client';

import { useState } from 'react';
import { Sidebar, SidebarBody, SidebarLink } from '@/components/ui/sidebar';
import {
  LayoutDashboard,
  Settings,
  Package,
  CreditCard,
  Users2,
  Receipt,
  Wallet,
  LogOut,
  FileText,
} from 'lucide-react';
import Link from 'next/link';
import Image from 'next/image';
import { motion } from 'framer-motion';
import { useRouter, usePathname } from 'next/navigation';
import { useWeb3AuthDisconnect } from '@web3auth/modal/react';
import { RoleSwitchButton } from '@/components/auth/role-switch-button';
import { logger } from '@/lib/core/logger/logger-utils';
// Logout flag to prevent auto-connect after logout
function setLogoutFlag() {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem('web3auth-logout', 'true');
  }
}

// Safe Web3Auth hook that handles missing context
function useSafeWeb3AuthDisconnect() {
  try {
    return useWeb3AuthDisconnect();
  } catch {
    // If Web3Auth context is not available, return fallback values
    logger.warn('⚠️ Web3Auth context not available in sidebar, using fallback');
    return {
      disconnect: async () => {
        logger.log('🔄 Web3Auth not available, skipping disconnect');
      },
      loading: false,
    };
  }
}

const iconAnimation = {
  initial: { y: 0, scale: 1 },
  hover: {
    y: -3,
    scale: 1.3,
    transition: {
      type: 'spring' as const,
      stiffness: 400,
      damping: 10,
    },
  },
};

export function MainSidebar() {
  const router = useRouter();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const { disconnect, loading: disconnectLoading } = useSafeWeb3AuthDisconnect();

  const handleSignOut = async () => {
    try {
      // First, disconnect from Web3Auth
      try {
        await disconnect();
      } catch {
        logger.warn('⚠️ Web3Auth disconnect failed (may not be connected)');
      }

      // Call logout API with POST method to clear session
      const response = await fetch('/api/auth/logout', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (response.ok) {
        // Set logout flag and redirect to merchant signin
        setLogoutFlag();
        window.location.href = '/merchants/signin';
      } else {
        logger.error('❌ Logout API failed:', response.status);
        // Still try to redirect even if API fails
        setLogoutFlag();
        window.location.href = '/merchants/signin';
      }
    } catch (error) {
      logger.error('❌ Logout failed:', error);
      // Force redirect anyway
      setLogoutFlag();
      window.location.href = '/merchants/signin';
    }
  };

  const navigationLinks = [
    {
      label: 'Dashboard',
      href: '/merchants/dashboard',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <LayoutDashboard
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/dashboard' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
    },
    {
      label: 'Customers',
      href: '/merchants/customers',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <Users2
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/customers' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
    },
    {
      label: 'Products',
      href: '/merchants/products',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <Package
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/products' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
      onClick: () => {
        router.push('/merchants/products');
        router.refresh();
      },
    },
    {
      label: 'Subscriptions',
      href: '/merchants/subscriptions',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <CreditCard
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/subscriptions' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
      onClick: () => router.push('/merchants/subscriptions'),
    },
    {
      label: 'Transactions',
      href: '/merchants/transactions',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <Receipt
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/transactions' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
    },
    {
      label: 'Invoices',
      href: '/merchants/invoices',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <FileText
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/invoices' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
    },
    {
      label: 'Wallets',
      href: '/merchants/wallets',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <Wallet
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/wallets' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
    },
    {
      label: 'Settings',
      href: '/merchants/settings',
      icon: (
        <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
          <Settings
            className={`h-5 w-5 flex-shrink-0 ${pathname === '/merchants/settings' ? 'text-gray-400' : 'text-neutral-700 dark:text-neutral-200'}`}
          />
        </motion.div>
      ),
    },
  ];

  const signOutLink = {
    label: disconnectLoading ? 'Signing Out...' : 'Sign Out',
    icon: (
      <motion.div whileHover="hover" initial="initial" variants={iconAnimation}>
        <LogOut className="text-neutral-700 dark:text-neutral-200 h-5 w-5 flex-shrink-0" />
      </motion.div>
    ),
    href: '#',
    onClick: disconnectLoading ? undefined : handleSignOut,
  };

  return (
    <Sidebar open={open} setOpen={setOpen}>
      <SidebarBody className="flex flex-col h-full">
        <div className="flex flex-col flex-1">
          {/* Logo */}
          {open ? <Logo /> : <LogoIcon />}

          {/* Navigation Links */}
          <div className="mt-8 flex flex-col gap-2">
            {navigationLinks.map((link, idx) => (
              <SidebarLink
                key={idx}
                link={link}
                className={pathname === link.href ? '[&_span]:text-gray-400' : ''}
              />
            ))}
          </div>
        </div>

        {/* Role Switch and Sign Out at Bottom */}
        <div className="mt-auto pt-4 border-t border-neutral-200 dark:border-neutral-700">
          {/* Role Switch Button */}
          <RoleSwitchButton currentRole="merchant" variant="sidebar" />

          {/* Sign Out Link */}
          <SidebarLink link={signOutLink} />
        </div>
      </SidebarBody>
    </Sidebar>
  );
}

export const Logo = () => {
  return (
    <Link
      href="/merchants/dashboard"
      className="font-normal flex space-x-2 items-center text-sm text-black py-1 relative z-20"
    >
      <div className="h-8 w-8 relative flex-shrink-0">
        <Image
          src="/images/icon.webp"
          alt="Cyphera Logo"
          width={32}
          height={32}
          className="object-contain"
          priority={true}
          placeholder="blur"
          blurDataURL="data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAAIAAoDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAhEAACAQMDBQAAAAAAAAAAAAABAgMABAUGIWGRkqGx0f/EABUBAQEAAAAAAAAAAAAAAAAAAAMF/8QAGhEAAgIDAAAAAAAAAAAAAAAAAAECEgMRkf/aAAwDAQACEQMRAD8AltJagyeH0AthI5xdrLcNM91BF5pX2HaH9bcfaSXWGaRmknyLDSv/2Q=="
        />
      </div>
      <motion.span
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="font-medium text-black dark:text-white whitespace-pre"
      >
        Cyphera
      </motion.span>
    </Link>
  );
};

export const LogoIcon = () => {
  return (
    <Link
      href="/merchants/dashboard"
      className="font-normal flex space-x-2 items-center text-sm text-black py-1 relative z-20"
    >
      <div className="h-8 w-8 relative flex-shrink-0">
        <Image
          src="/images/icon.webp"
          alt="Cyphera Logo"
          width={32}
          height={32}
          className="object-contain"
          priority={true}
          placeholder="blur"
          blurDataURL="data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/wAARCAAIAAoDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAhEAACAQMDBQAAAAAAAAAAAAABAgMABAUGIWGRkqGx0f/EABUBAQEAAAAAAAAAAAAAAAAAAAMF/8QAGhEAAgIDAAAAAAAAAAAAAAAAAAECEgMRkf/aAAwDAQACEQMRAD8AltJagyeH0AthI5xdrLcNM91BF5pX2HaH9bcfaSXWGaRmknyLDSv/2Q=="
        />
      </div>
    </Link>
  );
};
