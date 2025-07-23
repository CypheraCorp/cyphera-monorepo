'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Loader2, UserIcon, Store, ArrowRight } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import { logger } from '@/lib/core/logger/logger-utils';

interface UserRole {
  type: 'merchant' | 'customer';
  isAuthenticated: boolean;
  isOnboarded: boolean;
  email?: string;
  name?: string;
}

interface RoleSwitcherProps {
  currentRole?: 'merchant' | 'customer';
  onRoleSwitch?: (role: 'merchant' | 'customer') => void;
}

export function RoleSwitcher({ currentRole, onRoleSwitch }: RoleSwitcherProps) {
  const [isLoading, setIsLoading] = useState(true);
  const [merchantRole, setMerchantRole] = useState<UserRole>({
    type: 'merchant',
    isAuthenticated: false,
    isOnboarded: false,
  });
  const [customerRole, setCustomerRole] = useState<UserRole>({
    type: 'customer',
    isAuthenticated: false,
    isOnboarded: false,
  });
  const [isSwitching, setIsSwitching] = useState(false);

  const router = useRouter();
  const { toast } = useToast();

  // Check authentication status for both roles
  useEffect(() => {
    async function checkRoleStatus() {
      setIsLoading(true);

      try {
        // Check merchant session
        const merchantResponse = await fetch('/api/auth/me', {
          credentials: 'include',
        });

        if (merchantResponse.ok) {
          const merchantData = await merchantResponse.json();
          setMerchantRole({
            type: 'merchant',
            isAuthenticated: true,
            isOnboarded: merchantData.account?.finished_onboarding || false,
            email: merchantData.user?.email,
            name: merchantData.user?.name,
          });
        }

        // Check customer session
        const customerResponse = await fetch('/api/auth/customer/me', {
          credentials: 'include',
        });

        if (customerResponse.ok) {
          const customerData = await customerResponse.json();
          setCustomerRole({
            type: 'customer',
            isAuthenticated: true,
            isOnboarded: customerData.customer?.finished_onboarding || false,
            email: customerData.customer?.customer_email,
            name: customerData.customer?.customer_name,
          });
        }
      } catch (error) {
        logger.error('Error checking role status:', error);
      } finally {
        setIsLoading(false);
      }
    }

    checkRoleStatus();
  }, []);

  const handleRoleSwitch = async (targetRole: 'merchant' | 'customer') => {
    setIsSwitching(true);

    try {
      const role = targetRole === 'merchant' ? merchantRole : customerRole;

      if (!role.isAuthenticated) {
        // Redirect to signin for the target role
        const signinUrl = targetRole === 'merchant' ? '/merchants/signin' : '/customers/signin';
        router.push(signinUrl);
        return;
      }

      // Skip onboarding checks for merchants - go directly to dashboard

      // User is authenticated (and onboarded if merchant), redirect to dashboard
      const dashboardUrl =
        targetRole === 'merchant' ? '/merchants/dashboard' : '/customers/dashboard';

      // Call the callback if provided
      if (onRoleSwitch) {
        onRoleSwitch(targetRole);
      }

      router.push(dashboardUrl);
    } catch (error) {
      logger.error('Error switching roles:', error);
      toast({
        title: 'Error',
        description: 'Failed to switch roles. Please try again.',
        variant: 'destructive',
      });
    } finally {
      setIsSwitching(false);
    }
  };

  const getRoleStatus = (role: UserRole) => {
    if (!role.isAuthenticated) {
      return { status: 'Not signed in', color: 'secondary' as const };
    }
    // Skip onboarding status checks - all authenticated users are ready
    return { status: 'Ready', color: 'default' as const };
  };

  const getRoleAction = (role: UserRole) => {
    if (!role.isAuthenticated) {
      return 'Sign In';
    }
    // Skip onboarding actions - all authenticated users go to dashboard
    return 'Switch to Dashboard';
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin" />
            <span className="ml-2">Checking role status...</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">Choose Your Role</h2>
        <p className="text-gray-600 dark:text-gray-400">
          Switch between merchant and customer experiences
        </p>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        {/* Merchant Role */}
        <Card
          className={`transition-all ${currentRole === 'merchant' ? 'ring-2 ring-blue-500' : ''}`}
        >
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Store className="h-5 w-5 text-blue-600" />
                <CardTitle>Merchant</CardTitle>
              </div>
              <Badge variant={getRoleStatus(merchantRole).color}>
                {getRoleStatus(merchantRole).status}
              </Badge>
            </div>
            <CardDescription>Manage your business, products, and subscriptions</CardDescription>
          </CardHeader>
          <CardContent>
            {merchantRole.isAuthenticated && (
              <div className="mb-4 text-sm text-gray-600 dark:text-gray-400">
                <p>Signed in as: {merchantRole.email}</p>
                {merchantRole.name && <p>Name: {merchantRole.name}</p>}
              </div>
            )}
            <Button
              onClick={() => handleRoleSwitch('merchant')}
              disabled={isSwitching}
              className="w-full"
              variant={currentRole === 'merchant' ? 'default' : 'outline'}
            >
              {isSwitching ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <ArrowRight className="mr-2 h-4 w-4" />
              )}
              {getRoleAction(merchantRole)}
            </Button>
          </CardContent>
        </Card>

        {/* Customer Role */}
        <Card
          className={`transition-all ${currentRole === 'customer' ? 'ring-2 ring-purple-500' : ''}`}
        >
          <CardHeader>
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <UserIcon className="h-5 w-5 text-purple-600" />
                <CardTitle>Customer</CardTitle>
              </div>
              <Badge variant={getRoleStatus(customerRole).color}>
                {getRoleStatus(customerRole).status}
              </Badge>
            </div>
            <CardDescription>
              Browse products, manage subscriptions, and make purchases
            </CardDescription>
          </CardHeader>
          <CardContent>
            {customerRole.isAuthenticated && (
              <div className="mb-4 text-sm text-gray-600 dark:text-gray-400">
                <p>Signed in as: {customerRole.email}</p>
                {customerRole.name && <p>Name: {customerRole.name}</p>}
              </div>
            )}
            <Button
              onClick={() => handleRoleSwitch('customer')}
              disabled={isSwitching}
              className="w-full"
              variant={currentRole === 'customer' ? 'default' : 'outline'}
            >
              {isSwitching ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <ArrowRight className="mr-2 h-4 w-4" />
              )}
              {getRoleAction(customerRole)}
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
