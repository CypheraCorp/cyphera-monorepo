'use client';

import { 
  DashboardMetrics, 
  CustomerChart,
  SubscriptionChart,
} from '@/components/dashboard';
import { PaymentMetricsChart } from '@/components/dashboard/payment-metrics-chart';
import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMetricsRefresh, useDashboardSummary } from '@/hooks/use-analytics';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { DashboardEmptyState, OnboardingBanner } from '@/components/dashboard/empty-states';
import { useState, useEffect } from 'react';
import type { DashboardSummary } from '@/types/analytics';

export default function DashboardPage() {
  const refreshMetrics = useMetricsRefresh();
  const { data: summary, isFetching } = useDashboardSummary();
  const [showOnboarding, setShowOnboarding] = useState(true);
  const [hasTriggeredInitialCalculation, setHasTriggeredInitialCalculation] = useState(false);
  
  // Check if we have any data or are calculating
  const dashboardData = summary as DashboardSummary & { is_calculating?: boolean };
  const hasRealData = dashboardData && (
    (dashboardData.mrr?.amount_cents ?? 0) > 0 || 
    (dashboardData.total_customers ?? 0) > 0 || 
    (dashboardData.active_subscriptions ?? 0) > 0
  );
  const isCalculating = dashboardData?.is_calculating || false;
  
  // Show metrics view if we have data (even if zeros) OR if calculating
  // The key change: if we have dashboardData (successful API response), show metrics
  const shouldShowMetrics = !!dashboardData || isCalculating;
  
  // Auto-trigger metrics calculation when first loading and no data exists
  useEffect(() => {
    if (dashboardData && isCalculating && !hasTriggeredInitialCalculation && !refreshMetrics.isPending) {
      setHasTriggeredInitialCalculation(true);
      refreshMetrics.mutate(undefined);
    }
  }, [dashboardData, isCalculating, hasTriggeredInitialCalculation, refreshMetrics]);
  
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
          <div className="flex items-center gap-2">
            <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
            {isFetching && !refreshMetrics.isPending && (
              <div className="h-2 w-2 bg-primary rounded-full animate-pulse" />
            )}
          </div>
          <p className="text-muted-foreground">
            Monitor your subscription business performance
            {isFetching && !refreshMetrics.isPending && (
              <span className="ml-2 text-xs">• Updating...</span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-4">
          <Button
            variant="outline"
            size="sm"
            onClick={() => refreshMetrics.mutate(undefined)}
            disabled={refreshMetrics.isPending}
          >
            <RefreshCw 
              className={`h-4 w-4 mr-2 ${refreshMetrics.isPending || isFetching ? 'animate-spin' : ''}`} 
            />
            Refresh
          </Button>
        </div>
      </div>

      {/* Onboarding Banner for new users */}
      {!hasRealData && showOnboarding && !isCalculating && (
        <OnboardingBanner 
          currentStep={currentStep}
          totalSteps={onboardingSteps}
          onDismiss={() => setShowOnboarding(false)}
        />
      )}

      {/* Empty state when no data */}
      {!shouldShowMetrics && (
        <DashboardEmptyState 
          type="no-data" 
          onRefresh={() => refreshMetrics.mutate(undefined)}
          isRefreshing={refreshMetrics.isPending}
        />
      )}

      {/* Calculating state */}
      {isCalculating && !hasRealData && (
        <DashboardEmptyState 
          type="calculating" 
          isFetching={isFetching}
        />
      )}

      {/* Show metrics and charts when we have data or are calculating */}
      {shouldShowMetrics && (
        <>
          {/* Metrics Grid */}
          <DashboardMetrics />

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
