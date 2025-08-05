'use client';

import { useState } from 'react';
import { format } from 'date-fns';
import { Loader2, AlertCircle, TrendingUp, TrendingDown } from 'lucide-react';

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { formatCurrency } from '@/lib/utils/format';
import { toast } from 'sonner';
import type { SubscriptionResponse } from '@/types/subscription';

interface ChangePriceModalProps {
  subscription: SubscriptionResponse;
  open: boolean;
  onClose: () => void;
}

interface PricePreview {
  current_amount: number;
  new_amount: number;
  proration_credit?: number;
  immediate_charge?: number;
  effective_date: string;
  message: string;
  next_interval_amount?: number;
  next_interval_start?: string;
  proration_details?: {
    days_total: number;
    days_used: number;
    days_remaining: number;
    old_daily_rate: number;
    new_daily_rate: number;
  };
}

export function ChangePriceModal({ subscription, open, onClose }: ChangePriceModalProps) {
  const [newPrice, setNewPrice] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [preview, setPreview] = useState<PricePreview | null>(null);
  const [error, setError] = useState('');

  const currentPriceCents = subscription.product?.unit_amount_in_pennies || 0;
  const currentPrice = currentPriceCents / 100;

  const handlePriceChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    // Allow only numbers and decimal point
    if (/^\d*\.?\d{0,2}$/.test(value) || value === '') {
      setNewPrice(value);
      setError('');
      setPreview(null);
    }
  };

  const handlePreview = async () => {
    const priceValue = parseFloat(newPrice);
    if (isNaN(priceValue) || priceValue <= 0) {
      setError('Please enter a valid price');
      return;
    }

    const newPriceCents = Math.round(priceValue * 100);
    if (newPriceCents === currentPriceCents) {
      setError('New price must be different from current price');
      return;
    }

    setIsPreviewing(true);
    setError('');

    try {
      // Determine if this is an upgrade or downgrade
      const changeType = newPriceCents > currentPriceCents ? 'upgrade' : 'downgrade';
      
      const response = await fetch(`/api/subscriptions/${subscription.id}/preview-change`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          change_type: changeType,
          line_items: [{
            action: 'update',
            product_id: subscription.product?.id,
            unit_amount: newPriceCents,
            quantity: 1,
          }],
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to preview changes');
      }

      const data = await response.json();
      setPreview(data);
    } catch (err) {
      setError('Failed to preview price change. Please try again.');
      console.error('Preview error:', err);
    } finally {
      setIsPreviewing(false);
    }
  };

  const handleConfirm = async () => {
    if (!preview) return;

    setIsLoading(true);
    setError('');

    try {
      const newPriceCents = Math.round(parseFloat(newPrice) * 100);
      
      const response = await fetch(`/api/subscriptions/${subscription.id}/change-price`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          new_price_cents: newPriceCents,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to update subscription price');
      }

      toast.success(
        preview.immediate_charge 
          ? `Your subscription has been upgraded. You've been charged ${formatCurrency(preview.immediate_charge / 100)}.`
          : 'Your subscription price change has been scheduled.'
      );

      onClose();
      // Refresh the page to show updated data
      window.location.reload();
    } catch (err) {
      setError('Failed to update subscription price. Please try again.');
      console.error('Update error:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const isUpgrade = preview && preview.new_amount > preview.current_amount;

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Change Subscription Price</DialogTitle>
          <DialogDescription>
            Update your subscription pricing. Changes take effect based on whether you're upgrading or downgrading.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Current Price</Label>
            <div className="text-2xl font-bold">
              {formatCurrency(currentPrice)}/{subscription.product?.interval_type || 'month'}
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="new-price">New Price</Label>
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground">$</span>
              <Input
                id="new-price"
                type="text"
                value={newPrice}
                onChange={handlePriceChange}
                placeholder="0.00"
                className="pl-8"
                disabled={isLoading || isPreviewing}
              />
            </div>
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {!preview && newPrice && !error && (
            <Button 
              onClick={handlePreview} 
              className="w-full"
              disabled={isPreviewing}
            >
              {isPreviewing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Calculating...
                </>
              ) : (
                'Preview Changes'
              )}
            </Button>
          )}

          {preview && (
            <div className="space-y-4 rounded-lg border p-4">
              <div className="flex items-center gap-2">
                {isUpgrade ? (
                  <>
                    <TrendingUp className="h-5 w-5 text-green-600" />
                    <span className="font-semibold text-green-600">Upgrade</span>
                  </>
                ) : (
                  <>
                    <TrendingDown className="h-5 w-5 text-blue-600" />
                    <span className="font-semibold text-blue-600">Downgrade</span>
                  </>
                )}
              </div>

              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Current Price:</span>
                  <span>{formatCurrency(preview.current_amount / 100)}/{subscription.product?.interval_type}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">New Price:</span>
                  <span className="font-semibold">{formatCurrency(preview.new_amount / 100)}/{subscription.product?.interval_type}</span>
                </div>

                {isUpgrade && preview.immediate_charge && (
                  <>
                    <div className="border-t pt-3">
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Proration Credit:</span>
                        <span className="text-green-600">-{formatCurrency((preview.proration_credit || 0) / 100)}</span>
                      </div>
                      <div className="flex justify-between font-semibold">
                        <span>Due Today:</span>
                        <span>{formatCurrency(preview.immediate_charge / 100)}</span>
                      </div>
                    </div>
                    {preview.proration_details && (
                      <div className="text-sm text-muted-foreground">
                        {preview.proration_details.days_remaining} days remaining in current period
                      </div>
                    )}
                  </>
                )}

                <div className="border-t pt-3">
                  <div className="text-sm font-medium">Effective Date</div>
                  <div className="text-sm text-muted-foreground">
                    {format(new Date(preview.effective_date), 'MMM d, yyyy')}
                  </div>
                </div>

                {preview.next_interval_start && (
                  <div className="border-t pt-3">
                    <div className="text-sm font-medium">Next Billing</div>
                    <div className="text-sm text-muted-foreground">
                      {formatCurrency((preview.next_interval_amount || preview.new_amount) / 100)} on{' '}
                      {format(new Date(preview.next_interval_start), 'MMM d, yyyy')}
                    </div>
                  </div>
                )}

                <Alert>
                  <AlertDescription className="text-sm">
                    {preview.message}
                  </AlertDescription>
                </Alert>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          {preview && (
            <Button onClick={handleConfirm} disabled={isLoading}>
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Processing...
                </>
              ) : (
                'Confirm Change'
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}