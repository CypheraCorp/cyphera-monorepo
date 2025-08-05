'use client';

import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Input } from '@/components/ui/input';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { format } from 'date-fns';
import { Pause, Play, Loader2 } from 'lucide-react';
import { usePauseSubscription, useResumeSubscription } from '@/hooks/use-subscription-management';
import type { SubscriptionResponse } from '@/types/subscription';

interface PauseResumeControlsProps {
  subscription: SubscriptionResponse;
  subscriptionId: string;
}

export function PauseResumeControls({ subscription, subscriptionId }: PauseResumeControlsProps) {
  const [showPauseDialog, setShowPauseDialog] = useState(false);
  const [pauseType, setPauseType] = useState<'indefinite' | 'until'>('indefinite');
  const [pauseUntilDate, setPauseUntilDate] = useState<Date | undefined>();

  const pauseMutation = usePauseSubscription(subscriptionId);
  const resumeMutation = useResumeSubscription(subscriptionId);

  const isPaused = !!subscription.paused_at;

  const handlePause = () => {
    const request = {
      reason: 'Customer requested pause',
      pause_until: pauseType === 'until' && pauseUntilDate 
        ? pauseUntilDate.toISOString() 
        : undefined,
    };

    pauseMutation.mutate(request, {
      onSuccess: () => {
        setShowPauseDialog(false);
        setPauseType('indefinite');
        setPauseUntilDate(undefined);
      },
    });
  };

  const handleResume = () => {
    resumeMutation.mutate();
  };

  if (isPaused) {
    return (
      <div className="space-y-3">
        <Alert>
          <Pause className="h-4 w-4" />
          <AlertDescription>
            Your subscription is currently paused.
            {subscription.pause_ends_at && (
              <span> It will automatically resume on {format(new Date(subscription.pause_ends_at), 'MMMM d, yyyy')}.</span>
            )}
          </AlertDescription>
        </Alert>
        <Button
          onClick={handleResume}
          variant="outline"
          className="w-full"
          size="lg"
          disabled={resumeMutation.isPending}
        >
          {resumeMutation.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Play className="mr-2 h-4 w-4" />
          )}
          Resume Subscription
        </Button>
      </div>
    );
  }

  return (
    <>
      <Button
        onClick={() => setShowPauseDialog(true)}
        variant="outline"
        className="w-full"
        size="lg"
      >
        <Pause className="mr-2 h-4 w-4" />
        Pause Subscription
      </Button>

      <Dialog open={showPauseDialog} onOpenChange={setShowPauseDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Pause Subscription</DialogTitle>
            <DialogDescription>
              Temporarily pause your subscription. You can resume anytime.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-6 py-4">
            <div className="space-y-3">
              <Label>How long would you like to pause?</Label>
              <RadioGroup value={pauseType} onValueChange={(value: any) => setPauseType(value)}>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="indefinite" id="indefinite" />
                  <Label htmlFor="indefinite" className="font-normal cursor-pointer">
                    Pause indefinitely (resume manually)
                  </Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="until" id="until" />
                  <Label htmlFor="until" className="font-normal cursor-pointer">
                    Pause until a specific date
                  </Label>
                </div>
              </RadioGroup>
            </div>

            {pauseType === 'until' && (
              <div className="space-y-3">
                <Label htmlFor="pause-until-date">Resume date</Label>
                <Input
                  id="pause-until-date"
                  type="date"
                  value={pauseUntilDate ? pauseUntilDate.toISOString().split('T')[0] : ''}
                  min={new Date().toISOString().split('T')[0]}
                  onChange={(e) => {
                    const date = e.target.value ? new Date(e.target.value) : undefined;
                    setPauseUntilDate(date);
                  }}
                  className="w-full"
                />
                {pauseUntilDate && (
                  <p className="text-sm text-muted-foreground">
                    Will resume on {format(pauseUntilDate, 'MMMM d, yyyy')}
                  </p>
                )}
              </div>
            )}

            <Alert>
              <AlertDescription>
                <strong>What happens when you pause:</strong>
                <ul className="mt-2 space-y-1 text-sm">
                  <li>• Billing is suspended immediately</li>
                  <li>• You'll retain access for the remainder of your current period</li>
                  <li>• No charges while paused</li>
                  <li>• Resume anytime to continue where you left off</li>
                </ul>
              </AlertDescription>
            </Alert>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setShowPauseDialog(false)}>
              Cancel
            </Button>
            <Button
              onClick={handlePause}
              disabled={pauseMutation.isPending || (pauseType === 'until' && !pauseUntilDate)}
            >
              {pauseMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Pause Subscription
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}