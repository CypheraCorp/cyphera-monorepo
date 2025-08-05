'use client';

import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { Loader2, Info, Calculator } from 'lucide-react';
import { usePreviewSubscriptionChange, useUpgradeSubscription } from '@/hooks/use-subscription-management';
import { formatCurrency } from '@/lib/utils/format';
import { format } from 'date-fns';
import type { SubscriptionResponse, LineItemUpdate } from '@/types/subscription';

interface UpgradeModalProps {
  open: boolean;
  onClose: () => void;
  subscription: SubscriptionResponse;
}

// Mock upgrade options - in real app, fetch from products API
const UPGRADE_OPTIONS = [
  { 
    id: 'pro-plan', 
    name: 'Pro Plan', 
    price: 4900, 
    features: ['All Basic features', 'Priority support', 'Advanced analytics'] 
  },
  { 
    id: 'enterprise-plan', 
    name: 'Enterprise Plan', 
    price: 9900, 
    features: ['All Pro features', 'Dedicated support', 'Custom integrations', 'SLA'] 
  },
];

export function UpgradeModal({ open, onClose, subscription }: UpgradeModalProps) {
  const [selectedPlan, setSelectedPlan] = useState<string>('');
  const [isCalculating, setIsCalculating] = useState(false);

  const previewMutation = usePreviewSubscriptionChange(subscription.id);
  const upgradeMutation = useUpgradeSubscription(subscription.id);

  const currentAmount = subscription.total_amount_in_cents || 0;
  const upgradeOptions = UPGRADE_OPTIONS.filter(opt => opt.price > currentAmount);

  useEffect(() => {
    if (selectedPlan && open) {
      setIsCalculating(true);
      const selectedOption = upgradeOptions.find(opt => opt.id === selectedPlan);
      
      if (selectedOption) {
        // Create line item update for the upgrade
        const lineItems: LineItemUpdate[] = [
          {
            action: 'update',
            product_id: selectedOption.id,
            quantity: 1,
            unit_amount: selectedOption.price,
          }
        ];

        previewMutation.mutate({
          change_type: 'upgrade',
          line_items: lineItems,
        }, {
          onSettled: () => setIsCalculating(false),
        });
      }
    }
  }, [selectedPlan]);

  const handleUpgrade = () => {
    if (!selectedPlan) return;

    const selectedOption = upgradeOptions.find(opt => opt.id === selectedPlan);
    if (!selectedOption) return;

    const lineItems: LineItemUpdate[] = [
      {
        action: 'update',
        product_id: selectedOption.id,
        quantity: 1,
        unit_amount: selectedOption.price,
      }
    ];

    upgradeMutation.mutate({
      line_items: lineItems,
      reason: 'Customer requested upgrade',
    }, {
      onSuccess: () => {
        onClose();
        setSelectedPlan('');
      },
    });
  };

  const preview = previewMutation.data;
  const hasPreview = !!preview && !isCalculating;

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Upgrade Your Subscription</DialogTitle>
          <DialogDescription>
            Choose a plan that better fits your needs. Upgrades take effect immediately with proration.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Plan Selection */}
          <div className="space-y-4">
            <Label>Select a new plan</Label>
            <RadioGroup value={selectedPlan} onValueChange={setSelectedPlan}>
              {upgradeOptions.map((option) => (
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
          </div>

          {/* Proration Preview */}
          {hasPreview && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center gap-2">
                  <Calculator className="h-4 w-4" />
                  Upgrade Summary
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Current Plan:</span>
                    <span>{formatCurrency(preview.current_amount / 100)}/mo</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">New Plan:</span>
                    <span className="font-medium">{formatCurrency(preview.new_amount / 100)}/mo</span>
                  </div>
                  
                  {preview.proration_credit && preview.proration_credit > 0 && (
                    <>
                      <Separator />
                      <div className="flex justify-between text-green-600">
                        <span>Proration Credit:</span>
                        <span>-{formatCurrency(preview.proration_credit / 100)}</span>
                      </div>
                    </>
                  )}
                  
                  <Separator />
                  
                  <div className="flex justify-between font-semibold">
                    <span>Due Today:</span>
                    <span className="text-lg">
                      {formatCurrency((preview.immediate_charge || 0) / 100)}
                    </span>
                  </div>
                </div>

                {preview.proration_details && (
                  <Alert>
                    <Info className="h-4 w-4" />
                    <AlertDescription className="text-xs">
                      You have {preview.proration_details.days_remaining} days remaining in your current billing period.
                      We're crediting you for the unused time at your current rate.
                    </AlertDescription>
                  </Alert>
                )}

                <Alert>
                  <AlertDescription className="text-sm">
                    Your new plan starts immediately. Future bills will be{' '}
                    {formatCurrency(preview.new_amount / 100)} starting{' '}
                    {format(new Date(subscription.current_period_end), 'MMMM d, yyyy')}.
                  </AlertDescription>
                </Alert>
              </CardContent>
            </Card>
          )}

          {isCalculating && (
            <Card>
              <CardContent className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                <span className="ml-2 text-sm text-muted-foreground">Calculating proration...</span>
              </CardContent>
            </Card>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleUpgrade}
            disabled={!selectedPlan || upgradeMutation.isPending || isCalculating}
          >
            {upgradeMutation.isPending && (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            )}
            {hasPreview && preview.immediate_charge 
              ? `Upgrade Now (${formatCurrency((preview.immediate_charge || 0) / 100)})`
              : 'Upgrade Now'
            }
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}