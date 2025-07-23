'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { UserIcon } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import { motion } from 'framer-motion';
import { cn } from '@/lib/utils';
import { useSidebar } from '@/components/ui/sidebar';
import { logger } from '@/lib/core/logger/logger-utils';

interface RoleSwitchButtonProps {
  currentRole: 'merchant' | 'customer';
  className?: string;
  variant?: 'default' | 'sidebar';
}

// Helper function to safely use sidebar context
function useSidebarSafe() {
  try {
    return useSidebar();
  } catch {
    return { open: true, animate: false };
  }
}

export function RoleSwitchButton({
  currentRole,
  className = '',
  variant = 'default',
}: RoleSwitchButtonProps) {
  const [isSwitching, setIsSwitching] = useState(false);
  const router = useRouter();
  const { toast } = useToast();

  // Always call hooks at the top level
  const { open, animate } = useSidebarSafe();

  // Only show the button for merchants
  if (currentRole !== 'merchant') {
    return null;
  }

  const handleSwitchToCustomer = async () => {
    setIsSwitching(true);

    try {
      // Check if user has a customer session
      const customerResponse = await fetch('/api/auth/customer/me', {
        credentials: 'include',
      });

      if (customerResponse.ok) {
        // User is already signed in as customer, go to dashboard
        router.push('/customers/dashboard');
      } else {
        // User needs to sign in as customer
        router.push('/customers/signin');
      }
    } catch (error) {
      logger.error('Error switching to customer', error);
      toast({
        title: 'Error',
        description: 'Failed to switch to customer view. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setIsSwitching(false);
    }
  };

  // Sidebar variant - styled to match sidebar items
  if (variant === 'sidebar') {
    return (
      <motion.div
        whileHover={{ scale: 1.02 }}
        whileTap={{ scale: 0.98 }}
        className={cn(
          'flex items-center justify-start gap-2 group/sidebar py-2 px-3 rounded-md cursor-pointer transition-colors',
          'hover:bg-neutral-100 dark:hover:bg-neutral-800',
          'text-neutral-700 dark:text-neutral-200 hover:text-neutral-900 dark:hover:text-neutral-100',
          isSwitching && 'opacity-50 cursor-not-allowed',
          className
        )}
        onClick={isSwitching ? undefined : handleSwitchToCustomer}
      >
        <motion.div
          initial={{ y: 0, scale: 1 }}
          whileHover={{
            y: -2,
            scale: 1.1,
            transition: {
              type: 'spring',
              stiffness: 400,
              damping: 10,
            },
          }}
        >
          <UserIcon className="h-5 w-5 flex-shrink-0" />
        </motion.div>
        <motion.span
          animate={{
            display: animate ? (open ? 'inline-block' : 'none') : 'inline-block',
            opacity: animate ? (open ? 1 : 0) : 1,
          }}
          className="text-sm font-medium whitespace-pre inline-block !p-0 !m-0"
        >
          {isSwitching ? 'Switching...' : 'Switch to Customer'}
        </motion.span>
      </motion.div>
    );
  }

  // Default variant - original button styling
  return (
    <Button
      variant="outline"
      size="sm"
      className={className}
      onClick={handleSwitchToCustomer}
      disabled={isSwitching}
    >
      <UserIcon className="h-4 w-4 mr-2" />
      {isSwitching ? 'Switching...' : 'Switch to Customer'}
    </Button>
  );
}
