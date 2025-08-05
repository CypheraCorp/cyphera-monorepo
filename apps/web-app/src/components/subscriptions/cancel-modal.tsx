'use client';

import React, { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Card, CardContent } from '@/components/ui/card';
import { Loader2, AlertCircle, Calendar } from 'lucide-react';
import { useCancelSubscription } from '@/hooks/use-subscription-management';
import { format } from 'date-fns';
import type { SubscriptionResponse } from '@/types/subscription';

interface CancelModalProps {
  open: boolean;
  onClose: () => void;
  subscription: SubscriptionResponse;
}

const CANCELLATION_REASONS = [
  { value: 'too_expensive', label: 'Too expensive' },
  { value: 'not_using', label: 'Not using it enough' },
  { value: 'missing_features', label: 'Missing features I need' },
  { value: 'found_alternative', label: 'Found a better alternative' },
  { value: 'technical_issues', label: 'Technical issues' },
  { value: 'other', label: 'Other reason' },
];

export function CancelModal({ open, onClose, subscription }: CancelModalProps) {
  const [reason, setReason] = useState<string>('');
  const [feedback, setFeedback] = useState<string>('');
  const [confirmed, setConfirmed] = useState(false);

  const cancelMutation = useCancelSubscription(subscription.id);

  const handleCancel = () => {
    if (!reason || !confirmed) return;

    cancelMutation.mutate({
      reason,
      feedback: feedback.trim() || undefined,
    }, {
      onSuccess: () => {
        onClose();
        setReason('');
        setFeedback('');
        setConfirmed(false);
      },
    });
  };

  const cancelDate = new Date(subscription.current_period_end);

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Cancel Subscription</DialogTitle>
          <DialogDescription>
            We're sorry to see you go. Your subscription will remain active until the end of your billing period.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Cancellation Info */}
          <Card className="border-blue-200 bg-blue-50">
            <CardContent className="pt-6">
              <div className="flex items-start space-x-3">
                <Calendar className="h-5 w-5 text-blue-600 mt-0.5" />
                <div className="flex-1 space-y-1">
                  <p className="font-medium">Cancellation will take effect on:</p>
                  <p className="text-2xl font-semibold">{format(cancelDate, 'MMMM d, yyyy')}</p>
                  <p className="text-sm text-muted-foreground">
                    You'll continue to have full access to {subscription.product.name} until this date.
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Reason Selection */}
          <div className="space-y-3">
            <Label htmlFor="reason">Why are you cancelling? (Required)</Label>
            <RadioGroup value={reason} onValueChange={setReason}>
              {CANCELLATION_REASONS.map((option) => (
                <div key={option.value} className="flex items-center space-x-2">
                  <RadioGroupItem value={option.value} id={option.value} />
                  <Label htmlFor={option.value} className="font-normal cursor-pointer">
                    {option.label}
                  </Label>
                </div>
              ))}
            </RadioGroup>
          </div>

          {/* Additional Feedback */}
          <div className="space-y-3">
            <Label htmlFor="feedback">
              Additional feedback (Optional)
              <span className="text-sm font-normal text-muted-foreground ml-2">
                Help us improve by sharing more details
              </span>
            </Label>
            <Textarea
              id="feedback"
              value={feedback}
              onChange={(e) => setFeedback(e.target.value)}
              placeholder="Tell us more about your experience..."
              rows={4}
              className="resize-none"
            />
          </div>

          {/* What happens next */}
          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              <strong>What happens when you cancel:</strong>
              <ul className="mt-2 space-y-1 text-sm">
                <li>• You'll keep access until {format(cancelDate, 'MMMM d, yyyy')}</li>
                <li>• No more charges after your current billing period</li>
                <li>• Your data will be retained for 30 days after cancellation</li>
                <li>• You can reactivate anytime before the cancellation date</li>
              </ul>
            </AlertDescription>
          </Alert>

          {/* Confirmation */}
          <div className="flex items-start space-x-2">
            <input
              type="checkbox"
              id="confirm-cancel"
              checked={confirmed}
              onChange={(e) => setConfirmed(e.target.checked)}
              className="mt-1"
            />
            <label htmlFor="confirm-cancel" className="text-sm leading-relaxed">
              I understand that my subscription will be cancelled on {format(cancelDate, 'MMMM d, yyyy')} 
              and I'll lose access to premium features after that date.
            </label>
          </div>
        </div>

        <DialogFooter className="flex-col sm:flex-row gap-3">
          <Button 
            variant="outline" 
            onClick={onClose}
            className="sm:mr-auto"
          >
            Keep Subscription
          </Button>
          <div className="flex gap-3 w-full sm:w-auto">
            <Button
              variant="link"
              onClick={onClose}
              className="text-muted-foreground"
            >
              Cancel
            </Button>
            <Button
              onClick={handleCancel}
              disabled={!reason || !confirmed || cancelMutation.isPending}
              variant="destructive"
            >
              {cancelMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Confirm Cancellation
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}