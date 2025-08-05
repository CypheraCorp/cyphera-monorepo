'use client';

import React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { formatCurrency } from '@/lib/utils/format';
import { format } from 'date-fns';
import { Settings, AlertCircle, Calendar } from 'lucide-react';
import Link from 'next/link';
import type { SubscriptionResponse } from '@/types/subscription';

interface SubscriptionCardProps {
  subscription: SubscriptionResponse;
}

export function SubscriptionCard({ subscription }: SubscriptionCardProps) {
  const isActive = subscription.status === 'active';
  const hasScheduledCancellation = !!subscription.cancel_at;
  const hasScheduledChanges = subscription.scheduled_changes && subscription.scheduled_changes.length > 0;
  const amount = subscription.total_amount_in_cents || 0;

  const getStatusVariant = (status: string): "default" | "secondary" | "destructive" | "outline" => {
    switch (status) {
      case 'active':
        return 'default';
      case 'canceled':
        return 'destructive';
      case 'past_due':
        return 'destructive';
      default:
        return 'secondary';
    }
  };

  return (
    <Card>
      <CardHeader className="pb-4">
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="text-lg">{subscription.product.name}</CardTitle>
            <div className="mt-1 flex items-center gap-2">
              <span className="text-2xl font-semibold">
                {formatCurrency(amount / 100)}
              </span>
              <span className="text-muted-foreground">/month</span>
            </div>
          </div>
          <Badge variant={getStatusVariant(subscription.status)}>
            {subscription.status}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Calendar className="h-4 w-4" />
          <span>
            Next billing: {format(new Date(subscription.current_period_end), 'MMM d, yyyy')}
          </span>
        </div>

        {hasScheduledCancellation && (
          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription className="text-sm">
              Cancelling on {format(new Date(subscription.cancel_at!), 'MMM d, yyyy')}
            </AlertDescription>
          </Alert>
        )}

        {hasScheduledChanges && subscription.scheduled_changes!.map((change) => (
          <Alert key={change.id}>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription className="text-sm">
              {change.change_type === 'downgrade' && 'Downgrade'} 
              {change.change_type === 'upgrade' && 'Upgrade'} 
              {' scheduled for '}
              {format(new Date(change.scheduled_for), 'MMM d, yyyy')}
            </AlertDescription>
          </Alert>
        ))}

        <Link href={`/dashboard/subscription?id=${subscription.id}`}>
          <Button variant="outline" className="w-full">
            <Settings className="mr-2 h-4 w-4" />
            Manage Subscription
          </Button>
        </Link>
      </CardContent>
    </Card>
  );
}