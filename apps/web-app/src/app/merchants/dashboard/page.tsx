'use client';

import { 
  DashboardMetrics, 
  SubscriptionChart,
} from '@/components/dashboard';
import { PaymentMetricsChart } from '@/components/dashboard/payment-metrics-chart';
import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMetricsRefresh, useDashboardSummary } from '@/hooks/use-analytics';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { OnboardingBanner } from '@/components/dashboard/empty-states';
import { useState, useEffect } from 'react';

export default function DashboardPage() {
  const refreshMetrics = useMetricsRefresh();
  const { data: summary, isError } = useDashboardSummary();
  const [showOnboarding, setShowOnboarding] = useState(true);
  
  // Check dashboard state
  const hasRealData = summary && (
    (summary.mrr?.amount_cents ?? 0) > 0 || 
    (summary.total_customers ?? 0) > 0 || 
    (summary.active_subscriptions ?? 0) > 0
  );
  
  // Show metrics view if we have data
  const shouldShowMetrics = !!summary && !isError;
  
  // Determine onboarding progress
  const onboardingSteps = 4;
  const currentStep = hasRealData ? 4 : 1; // Simplified - you'd calculate this based on actual progress
  
  useEffect(() => {
    // Hide onboarding if user has real data
    if (hasRealData) {
      const timer = setTimeout(() => setShowOnboarding(false), 5000);
      return () => clearTimeout(timer);
    }
  }, [hasRealData]);

  return (
    <div className="space-y-6">
      {/* Header with actions */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground">
            Monitor your subscription business performance
          </p>
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={() => refreshMetrics.mutate(undefined)}
          className="h-8 w-8"
        >
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>

      {/* Onboarding Banner for new users */}
      {!hasRealData && showOnboarding && !isError && (
        <OnboardingBanner 
          currentStep={currentStep}
          totalSteps={onboardingSteps}
          onDismiss={() => setShowOnboarding(false)}
        />
      )}

      {/* Always show metrics - either with real data, stale data, or default zeros */}
      <DashboardMetrics />

      {/* Show charts when we have data */}
      {shouldShowMetrics && (
        <>
          {/* Analytics Tabs */}
          <Tabs defaultValue="subscriptions" className="space-y-4">
            <TabsList>
              <TabsTrigger value="subscriptions">Subscriptions</TabsTrigger>
              <TabsTrigger value="customers">Customers</TabsTrigger>
              <TabsTrigger value="payments">Payments</TabsTrigger>
            </TabsList>
            
            <TabsContent value="customers" className="space-y-4">
              <div className="bg-white p-6 rounded-lg border">
                <h3 className="text-lg font-semibold mb-4">Customer Analytics</h3>
                <div className="h-[300px] flex items-center justify-center bg-gray-50 rounded">
                  <p className="text-gray-500">Customer analytics data being refined - coming soon</p>
                </div>
              </div>
            </TabsContent>
            
            <TabsContent value="subscriptions" className="space-y-4">
              <SubscriptionChart />
            </TabsContent>
            
            <TabsContent value="payments" className="space-y-4">
              <PaymentMetricsChart />
            </TabsContent>
          </Tabs>
        </>
      )}
    </div>
  );
}
