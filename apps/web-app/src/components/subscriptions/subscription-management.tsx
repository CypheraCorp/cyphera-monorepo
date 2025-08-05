'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { 
  useReactivateSubscription, 
  useSubscription 
} from '@/hooks/use-subscription-management';
import { UpgradeModal } from './upgrade-modal';
import { DowngradeModal } from './downgrade-modal';
import { CancelModal } from './cancel-modal';
import { PauseResumeControls } from './pause-resume-controls';
import { formatCurrency } from '@/lib/utils/format';
import { format } from 'date-fns';
import { Loader2, AlertCircle, Info } from 'lucide-react';

interface SubscriptionManagementProps {
  subscriptionId: string;
}

export function SubscriptionManagement({ subscriptionId }: SubscriptionManagementProps) {
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);
  const [showDowngradeModal, setShowDowngradeModal] = useState(false);
  const [showCancelModal, setShowCancelModal] = useState(false);

  const { data: subscription, isLoading, error } = useSubscription(subscriptionId);
  const reactivateMutation = useReactivateSubscription(subscriptionId);

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    );
  }

  if (error || !subscription) {
    return (
      <Card>
        <CardContent className="py-12">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Failed to load subscription details. Please try again later.
            </AlertDescription>
          </Alert>
        </CardContent>
      </Card>
    );
  }

  const isActive = subscription.status === 'active';
  const isPaused = !!subscription.paused_at;
  const hasScheduledCancellation = !!subscription.cancel_at;
  const currentPlanAmount = subscription.total_amount_in_cents || 0;

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Subscription Management</CardTitle>
              <CardDescription>
                Manage your {subscription.product.name} subscription
              </CardDescription>
            </div>
            <Badge variant={isActive ? 'default' : 'secondary'}>
              {subscription.status}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Current Plan Details */}
          <div className="space-y-2">
            <h3 className="text-sm font-medium text-muted-foreground">Current Plan</h3>
            <div className="flex items-baseline justify-between">
              <span className="text-2xl font-semibold">{subscription.product.name}</span>
              <span className="text-xl">
                {formatCurrency(currentPlanAmount / 100)}/month
              </span>
            </div>
            <p className="text-sm text-muted-foreground">
              Billing period: {format(new Date(subscription.current_period_start), 'MMM d')} - 
              {' '}{format(new Date(subscription.current_period_end), 'MMM d, yyyy')}
            </p>
          </div>

          <Separator />

          {/* Scheduled Changes Alert */}
          {hasScheduledCancellation && (
            <Alert>
              <Info className="h-4 w-4" />
              <AlertDescription className="flex items-center justify-between">
                <span>
                  Subscription will cancel on {format(new Date(subscription.cancel_at!), 'MMMM d, yyyy')}.
                  You'll have access until then.
                </span>
                <Button
                  size="sm"
                  variant="link"
                  onClick={() => reactivateMutation.mutate()}
                  disabled={reactivateMutation.isPending}
                >
                  Keep Subscription
                </Button>
              </AlertDescription>
            </Alert>
          )}

          {subscription.scheduled_changes?.map((change) => (
            <Alert key={change.id}>
              <Info className="h-4 w-4" />
              <AlertDescription>
                {change.change_type === 'downgrade' && 
                  `Downgrade scheduled for ${format(new Date(change.scheduled_for), 'MMMM d, yyyy')}`}
                {change.change_type === 'upgrade' && 
                  `Upgrade scheduled for ${format(new Date(change.scheduled_for), 'MMMM d, yyyy')}`}
              </AlertDescription>
            </Alert>
          ))}

          {/* Action Buttons */}
          <div className="space-y-3">
            {isActive && !hasScheduledCancellation && (
              <>
                <Button 
                  onClick={() => setShowUpgradeModal(true)}
                  className="w-full"
                  size="lg"
                >
                  Upgrade Plan
                </Button>
                
                <Button 
                  onClick={() => setShowDowngradeModal(true)}
                  variant="outline"
                  className="w-full"
                  size="lg"
                >
                  Downgrade Plan
                </Button>

                <PauseResumeControls 
                  subscription={subscription}
                  subscriptionId={subscriptionId}
                />

                <Separator />
                
                <Button 
                  onClick={() => setShowCancelModal(true)}
                  variant="ghost"
                  className="w-full text-destructive hover:text-destructive"
                  size="lg"
                >
                  Cancel Subscription
                </Button>
              </>
            )}

            {isPaused && (
              <Alert>
                <AlertDescription>
                  Your subscription is currently paused.
                  {subscription.pause_ends_at && 
                    ` It will automatically resume on ${format(new Date(subscription.pause_ends_at), 'MMMM d, yyyy')}.`
                  }
                </AlertDescription>
              </Alert>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Modals */}
      <UpgradeModal
        open={showUpgradeModal}
        onClose={() => setShowUpgradeModal(false)}
        subscription={subscription}
      />

      <DowngradeModal
        open={showDowngradeModal}
        onClose={() => setShowDowngradeModal(false)}
        subscription={subscription}
      />

      <CancelModal
        open={showCancelModal}
        onClose={() => setShowCancelModal(false)}
        subscription={subscription}
      />
    </>
  );
}