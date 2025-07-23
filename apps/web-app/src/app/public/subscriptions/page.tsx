'use client';

import React from 'react';
import { useWeb3AuthUser, useWeb3Auth } from '@web3auth/modal/react';
import { CustomerWeb3AuthLogin } from '@/components/auth/customer-web3auth-login';
import { CreditCard, Calendar, DollarSign, ExternalLink, Wallet } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

// Safe Web3Auth user hook - removed as it violates React hooks rules
// Instead, we'll handle the error at the component level

// Mock subscription data - in real app this would come from API
const mockSubscriptions = [
  {
    id: '1',
    product_name: 'Premium API Access',
    description: 'High-rate API access with advanced features',
    price: '$29.99',
    billing_cycle: 'monthly',
    status: 'active',
    next_billing: '2024-02-15',
    created_at: '2024-01-15',
  },
  {
    id: '2',
    product_name: 'Data Analytics Pro',
    description: 'Advanced analytics and reporting tools',
    price: '$99.99',
    billing_cycle: 'monthly',
    status: 'active',
    next_billing: '2024-02-20',
    created_at: '2024-01-20',
  },
];

export default function CustomerSubscriptionsPage() {
  // Web3Auth hooks - must be called unconditionally at the top level
  const userResult = useWeb3AuthUser();
  const authResult = useWeb3Auth();

  // Extract values with proper error handling
  const userInfo = userResult?.userInfo || null;
  const isConnected = authResult?.isConnected || false;

  // Show authentication form if not connected
  if (!isConnected || !userInfo) {
    return (
      <div className="flex-1 container mx-auto p-8 space-y-8">
        <div className="max-w-2xl mx-auto">
          <div className="text-center mb-8">
            <CreditCard className="h-16 w-16 mx-auto mb-4 text-purple-600" />
            <h1 className="text-4xl font-bold mb-2">My Subscriptions</h1>
            <p className="text-lg text-muted-foreground">
              Manage your active subscriptions and billing
            </p>
          </div>

          <div className="bg-white dark:bg-neutral-800 rounded-lg border p-8 shadow-sm">
            <div className="text-center mb-6">
              <h2 className="text-2xl font-semibold mb-2">Sign In Required</h2>
              <p className="text-muted-foreground">
                Please sign in with Web3Auth to view your subscriptions.
              </p>
            </div>

            <CustomerWeb3AuthLogin autoConnect={false} redirectTo="/public/subscriptions" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 container mx-auto p-8 space-y-8">
      <div className="flex items-center gap-3 mb-8">
        <CreditCard className="h-8 w-8 text-purple-600" />
        <div>
          <h1 className="text-4xl font-bold">My Subscriptions</h1>
          <p className="text-lg text-muted-foreground">
            Manage your active subscriptions and billing
          </p>
        </div>
      </div>

      {/* Subscriptions Overview */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Subscriptions</CardTitle>
            <CreditCard className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{mockSubscriptions.length}</div>
            <p className="text-xs text-muted-foreground">Currently active</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Monthly Spend</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">$129.98</div>
            <p className="text-xs text-muted-foreground">Total monthly cost</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Next Billing</CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">Feb 15</div>
            <p className="text-xs text-muted-foreground">Premium API Access</p>
          </CardContent>
        </Card>
      </div>

      {/* Subscriptions List */}
      <div className="space-y-4">
        <h2 className="text-2xl font-semibold">Active Subscriptions</h2>

        {mockSubscriptions.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12">
              <CreditCard className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-xl font-semibold mb-2">No Active Subscriptions</h3>
              <p className="text-muted-foreground text-center mb-4">
                You don&apos;t have any active subscriptions yet. Browse our products to get
                started.
              </p>
              <Button asChild>
                <a href="/public/signin">Sign In to Browse Products</a>
              </Button>
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-4">
            {mockSubscriptions.map((subscription) => (
              <Card key={subscription.id}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div>
                      <CardTitle className="flex items-center gap-2">
                        {subscription.product_name}
                        <Badge variant={subscription.status === 'active' ? 'default' : 'secondary'}>
                          {subscription.status}
                        </Badge>
                      </CardTitle>
                      <CardDescription>{subscription.description}</CardDescription>
                    </div>
                    <div className="text-right">
                      <div className="text-2xl font-bold">{subscription.price}</div>
                      <div className="text-sm text-muted-foreground">
                        per {subscription.billing_cycle}
                      </div>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid gap-4 md:grid-cols-3">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Started</label>
                      <p className="text-sm">
                        {new Date(subscription.created_at).toLocaleDateString()}
                      </p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">
                        Next Billing
                      </label>
                      <p className="text-sm">
                        {new Date(subscription.next_billing).toLocaleDateString()}
                      </p>
                    </div>
                    <div className="flex items-end">
                      <Button variant="outline" size="sm" className="ml-auto">
                        Manage
                        <ExternalLink className="h-4 w-4 ml-1" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>

      {/* Billing Information */}
      <Card>
        <CardHeader>
          <CardTitle>Billing Information</CardTitle>
          <CardDescription>Your payment method and billing details</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">Payment Method</label>
              <div className="flex items-center gap-2 mt-1">
                <div className="flex items-center gap-2 px-3 py-2 bg-muted rounded-md">
                  <Wallet className="h-4 w-4" />
                  <span className="text-sm">Web3Auth Wallet</span>
                </div>
                <Badge variant="secondary">USDC</Badge>
              </div>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">Wallet Address</label>
              <p className="text-sm font-mono mt-1">
                {userInfo?.email ? `Connected to ${userInfo.email}` : 'Wallet connected'}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
