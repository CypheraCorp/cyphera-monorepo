'use client';

import { useState, use } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, MoreVertical, AlertCircle, CheckCircle2, Clock, CreditCard, User, Wallet, Package, Calendar, Pause, Play, XCircle, RefreshCw } from 'lucide-react';
import { format } from 'date-fns';
import Link from 'next/link';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Skeleton } from '@/components/ui/skeleton';

import { useSubscription } from '@/hooks/use-subscription';
import { formatCurrency, formatTokenAmount } from '@/lib/utils/format';
import { CancelModal } from '@/components/subscriptions/cancel-modal';
import { PauseResumeControls } from '@/components/subscriptions/pause-resume-controls';
import type { SubscriptionResponse } from '@/types/subscription';

interface PageProps {
  params: Promise<{ subscriptionId: string }>;
}

export default function MerchantSubscriptionDetailsPage({ params }: PageProps) {
  const { subscriptionId } = use(params);
  const router = useRouter();
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [showPauseDialog, setShowPauseDialog] = useState(false);

  const { data: subscription, isLoading, error } = useSubscription(subscriptionId);

  if (isLoading) {
    return <SubscriptionDetailsSkeleton />;
  }

  if (error || !subscription) {
    return (
      <div className="container max-w-7xl mx-auto py-8 px-4">
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-destructive">
              <AlertCircle className="h-5 w-5" />
              Error Loading Subscription
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground mb-4">
              {error?.message || 'Failed to load subscription details'}
            </p>
            <Button variant="outline" onClick={() => router.back()}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Go Back
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <CheckCircle2 className="h-4 w-4" />;
      case 'canceled':
        return <XCircle className="h-4 w-4" />;
      case 'past_due':
        return <AlertCircle className="h-4 w-4" />;
      default:
        return <Clock className="h-4 w-4" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-500/10 text-green-700 border-green-200';
      case 'canceled':
        return 'bg-red-500/10 text-red-700 border-red-200';
      case 'completed':
        return 'bg-purple-500/10 text-purple-700 border-purple-200';
      case 'past_due':
        return 'bg-yellow-500/10 text-yellow-700 border-yellow-200';
      default:
        return 'bg-gray-500/10 text-gray-700 border-gray-200';
    }
  };

  // Parse metadata to get additional subscription details
  const getMetadataField = (field: string): any => {
    if (!subscription.metadata) return null;
    
    try {
      // Check if metadata is a base64 string
      if (typeof subscription.metadata === 'string') {
        const decoded = atob(subscription.metadata);
        const parsed = JSON.parse(decoded);
        return parsed[field];
      }
      // If metadata is already an object
      return subscription.metadata[field];
    } catch (error) {
      console.error('Error parsing metadata:', error);
      return null;
    }
  };

  return (
    <div className="container max-w-7xl mx-auto py-8 px-4 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.push('/merchants/subscriptions')}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">Subscription Details</h1>
            <p className="text-muted-foreground">
              Manage subscription #{subscription.num_id}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Badge className={`px-3 py-1 flex items-center gap-1.5 ${getStatusColor(subscription.status)}`}>
            {getStatusIcon(subscription.status)}
            {subscription.status.charAt(0).toUpperCase() + subscription.status.slice(1)}
          </Badge>
          
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <DropdownMenuLabel>Quick Actions</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => setShowPauseDialog(true)}>
                <Pause className="mr-2 h-4 w-4" />
                Pause Subscription
              </DropdownMenuItem>
              <DropdownMenuItem 
                onClick={() => setShowCancelModal(true)}
                className="text-destructive focus:text-destructive"
              >
                <XCircle className="mr-2 h-4 w-4" />
                Cancel Subscription
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Monthly Revenue
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatCurrency((subscription.total_amount_in_cents || 0) / 100)}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Per {getMetadataField('interval_type') || 'month'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total Redemptions
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {subscription.total_redemptions || 0}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              Successful payments
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Next Billing
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {format(new Date(subscription.current_period_end), 'MMM d')}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {format(new Date(subscription.current_period_end), 'yyyy')}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Customer Since
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {format(new Date(subscription.created_at), 'MMM d, yyyy')}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {Math.floor((new Date().getTime() - new Date(subscription.created_at).getTime()) / (1000 * 60 * 60 * 24))} days
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Main Content */}
      <div className="space-y-6">
          {/* Product Information */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Package className="h-5 w-5" />
                Product Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 md:grid-cols-2">
                <div>
                  <h3 className="font-semibold text-lg mb-2">{subscription.product?.name || 'Unnamed Product'}</h3>
                  <p className="text-muted-foreground text-sm">
                    Product ID: {subscription.product?.id || 'N/A'}
                  </p>
                </div>
                <div className="space-y-3">
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-muted-foreground">Price</span>
                    <span className="font-medium">
                      {formatCurrency((subscription.product?.unit_amount_in_pennies || 0) / 100)}
                      /{subscription.product?.interval_type || 'month'}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-muted-foreground">Token</span>
                    <span className="font-medium">
                      {subscription.product_token?.token_symbol || 'Unknown'} on {subscription.product_token?.network_name || 'Unknown'}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-muted-foreground">Token Amount</span>
                    <span className="font-medium">
                      {formatTokenAmount(subscription.token_amount, subscription.product_token?.token_decimals || 18)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-sm text-muted-foreground">Billing Interval</span>
                    <span className="font-medium">
                      Every {subscription.product?.interval_type || 'N/A'}
                      {subscription.product?.term_length && subscription.product.term_length > 1 && ` (${subscription.product.term_length} periods)`}
                    </span>
                  </div>
                </div>
              </div>

              {/* Line Items */}
              {subscription.line_items && subscription.line_items.length > 0 && (
                <>
                  <Separator />
                  <div>
                    <h4 className="font-medium mb-3">Line Items</h4>
                    <div className="space-y-2">
                      {subscription.line_items.map((item) => (
                        <div key={item.id} className="flex justify-between items-center py-2">
                          <div>
                            <span className="font-medium">
                              {item.product?.name || 'Product'}
                            </span>
                            <Badge variant="outline" className="ml-2 text-xs">
                              {item.line_item_type}
                            </Badge>
                          </div>
                          <div className="text-right">
                            <span className="font-medium">
                              {formatCurrency(item.total_amount_in_pennies / 100)}
                            </span>
                            <span className="text-sm text-muted-foreground ml-1">
                              (×{item.quantity})
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* Management Actions */}
          <Card>
            <CardHeader>
              <CardTitle>Subscription Management</CardTitle>
              <CardDescription>
                Make changes to this subscription
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex justify-center gap-4">
                <Button 
                  variant="outline" 
                  onClick={() => setShowPauseDialog(true)}
                >
                  <Pause className="mr-2 h-4 w-4" />
                  Pause Subscription
                </Button>
                <Button 
                  variant="outline" 
                  className="text-destructive hover:text-destructive"
                  onClick={() => setShowCancelModal(true)}
                >
                  <XCircle className="mr-2 h-4 w-4" />
                  Cancel Subscription
                </Button>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CreditCard className="h-5 w-5" />
                Billing Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 md:grid-cols-2">
                <div className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Current Period
                    </label>
                    <p className="text-sm">
                      {format(new Date(subscription.current_period_start), 'MMM d, yyyy')} - 
                      {' '}{format(new Date(subscription.current_period_end), 'MMM d, yyyy')}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Next Redemption
                    </label>
                    <p className="text-sm">
                      {subscription.next_redemption_date 
                        ? format(new Date(subscription.next_redemption_date), 'MMM d, yyyy h:mm a')
                        : 'Not scheduled'}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Total Paid
                    </label>
                    <p className="text-sm font-medium">
                      {formatCurrency((subscription.total_amount_in_cents || 0) / 100)}
                    </p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <User className="h-5 w-5" />
                Customer Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 md:grid-cols-2">
                <div className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Name</label>
                    <p className="text-sm font-medium">
                      {subscription.customer?.name || subscription.customer_name || 'Not provided'}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Email</label>
                    <p className="text-sm font-medium">
                      {subscription.customer?.email || subscription.customer_email || 'Not provided'}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Phone</label>
                    <p className="text-sm">
                      {subscription.customer?.phone || 'Not provided'}
                    </p>
                  </div>
                </div>
                <div className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Customer ID
                    </label>
                    <p className="text-sm font-mono">
                      #{subscription.customer?.num_id || 'N/A'}
                    </p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Customer Since
                    </label>
                    <p className="text-sm">
                      {subscription.customer?.created_at 
                        ? format(new Date(subscription.customer.created_at), 'MMM d, yyyy')
                        : 'N/A'}
                    </p>
                  </div>
                </div>
              </div>
              
              {subscription.customer?.metadata && Object.keys(subscription.customer.metadata).length > 0 && (
                <>
                  <Separator />
                  <div>
                    <h4 className="font-medium mb-3">Customer Metadata</h4>
                    <div className="bg-muted rounded-lg p-4">
                      <pre className="text-xs overflow-auto">
                        {JSON.stringify(subscription.customer.metadata, null, 2)}
                      </pre>
                    </div>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Wallet className="h-5 w-5" />
                Technical Details
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 md:grid-cols-2">
                <div className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Subscription ID
                    </label>
                    <p className="text-sm font-mono break-all">{subscription.id}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Workspace ID
                    </label>
                    <p className="text-sm font-mono break-all">{subscription.workspace_id}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Delegation ID
                    </label>
                    <p className="text-sm font-mono break-all">{subscription.delegation_id}</p>
                  </div>
                </div>
                <div className="space-y-4">
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">
                      Customer Wallet ID
                    </label>
                    <p className="text-sm font-mono break-all">
                      {subscription.customer_wallet_id || 'Not set'}
                    </p>
                  </div>
                </div>
              </div>

              <Separator />

              <div className="space-y-4">
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Created At
                  </label>
                  <p className="text-sm">
                    {format(new Date(subscription.created_at), 'MMM d, yyyy h:mm:ss a')}
                  </p>
                </div>
                <div>
                  <label className="text-sm font-medium text-muted-foreground">
                    Last Updated
                  </label>
                  <p className="text-sm">
                    {format(new Date(subscription.updated_at), 'MMM d, yyyy h:mm:ss a')}
                  </p>
                </div>
              </div>

              {subscription.metadata && Object.keys(subscription.metadata).length > 0 && (
                <>
                  <Separator />
                  <div>
                    <h4 className="font-medium mb-3">Subscription Metadata</h4>
                    <div className="bg-muted rounded-lg p-4">
                      <pre className="text-xs overflow-auto">
                        {JSON.stringify(subscription.metadata, null, 2)}
                      </pre>
                    </div>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
      </div>

      {/* Modals */}
      {subscription && (
        <>
          <CancelModal
            subscription={subscription}
            open={showCancelModal}
            onClose={() => setShowCancelModal(false)}
          />
          {showPauseDialog && (
            <div className="fixed inset-0 z-50 flex items-center justify-center">
              <div className="fixed inset-0 bg-black/50" onClick={() => setShowPauseDialog(false)} />
              <div className="relative bg-background rounded-lg p-6 max-w-md w-full mx-4">
                <PauseResumeControls
                  subscription={subscription}
                  subscriptionId={subscription.id}
                />
                <Button
                  variant="outline"
                  className="mt-4 w-full"
                  onClick={() => setShowPauseDialog(false)}
                >
                  Close
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function SubscriptionDetailsSkeleton() {
  return (
    <div className="container max-w-7xl mx-auto py-8 px-4 space-y-8">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Skeleton className="h-10 w-10" />
          <div>
            <Skeleton className="h-8 w-48 mb-2" />
            <Skeleton className="h-4 w-32" />
          </div>
        </div>
        <div className="flex gap-3">
          <Skeleton className="h-8 w-24" />
          <Skeleton className="h-10 w-10" />
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Card key={i}>
            <CardHeader className="pb-3">
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-32 mb-1" />
              <Skeleton className="h-3 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-20 w-full" />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}