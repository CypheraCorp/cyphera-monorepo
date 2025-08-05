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
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import { Loader2, AlertCircle, Calendar } from 'lucide-react';
import { useDowngradeSubscription } from '@/hooks/use-subscription-management';
import { formatCurrency } from '@/lib/utils/format';
import { format } from 'date-fns';
import type { SubscriptionResponse, LineItemUpdate } from '@/types/subscription';

interface DowngradeModalProps {
  open: boolean;
  onClose: () => void;
  subscription: SubscriptionResponse;
}

// Mock downgrade options - in real app, fetch from products API
const DOWNGRADE_OPTIONS = [
  { 
    id: 'starter-plan', 
    name: 'Starter Plan', 
    price: 900, 
    features: ['Basic features', 'Email support', 'Up to 10 users'] 
  },
  { 
    id: 'basic-plan', 
    name: 'Basic Plan', 
    price: 1900, 
    features: ['Essential features', 'Chat support', 'Up to 50 users'] 
  },
];

export function DowngradeModal({ open, onClose, subscription }: DowngradeModalProps) {
  const [selectedPlan, setSelectedPlan] = useState<string>('');
  const [confirmed, setConfirmed] = useState(false);

  const downgradeMutation = useDowngradeSubscription(subscription.id);

  const currentAmount = subscription.total_amount_in_cents || 0;
  const downgradeOptions = DOWNGRADE_OPTIONS.filter(opt => opt.price < currentAmount);

  const handleDowngrade = () => {
    if (!selectedPlan || !confirmed) return;

    const selectedOption = downgradeOptions.find(opt => opt.id === selectedPlan);
    if (!selectedOption) return;

    const lineItems: LineItemUpdate[] = [
      {
        action: 'update',
        product_id: selectedOption.id,
        quantity: 1,
        unit_amount: selectedOption.price,
      }
    ];

    downgradeMutation.mutate({
      line_items: lineItems,
      reason: 'Customer requested downgrade',
    }, {
      onSuccess: () => {
        onClose();
        setSelectedPlan('');
        setConfirmed(false);
      },
    });
  };

  const selectedOption = downgradeOptions.find(opt => opt.id === selectedPlan);
  const effectiveDate = new Date(subscription.current_period_end);

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Downgrade Your Subscription</DialogTitle>
          <DialogDescription>
            Select a plan that better matches your current needs. Downgrades take effect at the end of your billing period.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Plan Selection */}
          <div className="space-y-4">
            <Label>Select a new plan</Label>
            {downgradeOptions.length === 0 ? (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  You're already on our most basic plan. No downgrade options are available.
                </AlertDescription>
              </Alert>
            ) : (
              <RadioGroup value={selectedPlan} onValueChange={setSelectedPlan}>
                {downgradeOptions.map((option) => (
                  <Card key={option.id} className="cursor-pointer hover:border-primary">
                    <CardHeader className="p-4">
                      <label 
                        htmlFor={option.id} 
                        className="flex items-start space-x-3 cursor-pointer"
                      >
                        <RadioGroupItem value={option.id} id={option.id} />
                        <div className="flex-1">
                          <div className="flex items-baseline justify-between">
                            <h4 className="font-semibold">{option.name}</h4>
                            <span className="text-lg font-semibold">
                              {formatCurrency(option.price / 100)}/mo
                            </span>
                          </div>
                          <ul className="mt-2 space-y-1 text-sm text-muted-foreground">
                            {option.features.map((feature, idx) => (
                              <li key={idx}>• {feature}</li>
                            ))}
                          </ul>
                        </div>
                      </label>
                    </CardHeader>
                  </Card>
                ))}
              </RadioGroup>
            )}
          </div>

          {/* Downgrade Information */}
          {selectedOption && (
            <Card className="border-blue-200 bg-blue-50">
              <CardHeader className="pb-3">
                <h4 className="font-medium flex items-center gap-2">
                  <Calendar className="h-4 w-4" />
                  Downgrade Schedule
                </h4>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Current Plan:</span>
                    <span>{formatCurrency(currentAmount / 100)}/mo</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">New Plan:</span>
                    <span className="font-medium">{formatCurrency(selectedOption.price / 100)}/mo</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Effective Date:</span>
                    <span className="font-medium">{format(effectiveDate, 'MMMM d, yyyy')}</span>
                  </div>
                </div>

                <Alert>
                  <AlertDescription className="text-sm">
                    You'll continue to have full access to your current plan ({subscription.product.name}) 
                    until {format(effectiveDate, 'MMMM d, yyyy')}. Your new plan will start automatically 
                    on that date.
                  </AlertDescription>
                </Alert>

                <Alert variant="default" className="border-amber-200 bg-amber-50">
                  <AlertCircle className="h-4 w-4 text-amber-600" />
                  <AlertDescription className="text-sm">
                    <strong className="font-medium">No refunds or credits</strong> will be issued for downgrades. 
                    You'll have access to your current plan features until the end of the billing period.
                  </AlertDescription>
                </Alert>

                {/* Confirmation Checkbox */}
                <div className="flex items-start space-x-2 pt-2">
                  <input
                    type="checkbox"
                    id="confirm-downgrade"
                    checked={confirmed}
                    onChange={(e) => setConfirmed(e.target.checked)}
                    className="mt-1"
                  />
                  <label htmlFor="confirm-downgrade" className="text-sm leading-relaxed">
                    I understand that my downgrade will take effect on {format(effectiveDate, 'MMMM d, yyyy')} 
                    and I'll continue paying my current rate until then.
                  </label>
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleDowngrade}
            disabled={!selectedPlan || !confirmed || downgradeMutation.isPending}
            variant="secondary"
          >
            {downgradeMutation.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            Schedule Downgrade
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}