'use client';

// TEMPORARY DEV ROUTE - Remove in production
// This bypasses authentication to test dashboard functionality

import { 
  DashboardMetrics, 
  RevenueChart,
  CustomerChart,
  SubscriptionChart,
  MRRChart,
} from '@/components/dashboard';
import { PaymentMetricsChart } from '@/components/dashboard/payment-metrics-chart';
import { RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { DashboardEmptyState, OnboardingBanner } from '@/components/dashboard/empty-states';
import { useState, useEffect } from 'react';

// Mock data that simulates the API response with realistic values
const mockDashboardData = {
  mrr: { amount_cents: 299700, currency: 'USD', formatted: '$2,997.00' },
  arr: { amount_cents: 3596400, currency: 'USD', formatted: '$35,964.00' },
  total_revenue: { amount_cents: 1847200, currency: 'USD', formatted: '$18,472.00' },
  active_subscriptions: 15,
  total_customers: 23,
  churn_rate: 0.05,
  growth_rate: 0.12,
  payment_success_rate: 0.94,
  last_updated: new Date().toISOString(),
  is_calculating: false,
};

export default function DevDashboardPage() {
  const [showOnboarding, setShowOnboarding] = useState(true);
  const [isLoading, setIsLoading] = useState(false);
  
  // Simulate the dashboard logic
  const summary = mockDashboardData;
  const dashboardData = summary;
  const hasRealData = dashboardData && (
    (dashboardData.mrr?.amount_cents ?? 0) > 0 || 
    (dashboardData.total_customers ?? 0) > 0 || 
    (dashboardData.active_subscriptions ?? 0) > 0
  );
  const isCalculating = dashboardData?.is_calculating || false;
  
  // Show metrics view if we have data (even if zeros) OR if calculating
  const shouldShowMetrics = !!dashboardData || isCalculating;
  
  // Determine onboarding progress
  const onboardingSteps = 4;
  const currentStep = hasRealData ? 4 : 1;
  
  useEffect(() => {
    // Hide onboarding if user has real data
    if (hasRealData) {
      const timer = setTimeout(() => setShowOnboarding(false), 5000);
      return () => clearTimeout(timer);
    }
  }, [hasRealData]);

  const refreshMetrics = () => {
    setIsLoading(true);
    setTimeout(() => setIsLoading(false), 1000);
  };

  return (
    <div className="min-h-screen bg-background p-8">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Header */}
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6">
          <h2 className="text-yellow-800 font-semibold">🚧 Development Dashboard Test</h2>
          <p className="text-yellow-700 text-sm mt-1">
            This is a temporary route to test dashboard functionality while Web3Auth is having issues.
            Visit this at: <code className="bg-yellow-100 px-1 rounded">http://localhost:3000/dev-dashboard</code>
          </p>
        </div>

        {/* Header with actions */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
            </div>
            <p className="text-muted-foreground">
              Monitor your subscription business performance
            </p>
          </div>
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              size="sm"
              onClick={refreshMetrics}
              disabled={isLoading}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>
        </div>

        {/* Show the current logic result */}
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <h3 className="text-blue-800 font-semibold mb-2">Dashboard Logic Test Results:</h3>
          <div className="text-sm text-blue-700 space-y-1">
            <p><strong>hasRealData:</strong> {hasRealData ? 'true' : 'false'} (checks if any values &gt; 0)</p>
            <p><strong>isCalculating:</strong> {isCalculating ? 'true' : 'false'} (from API response)</p>
            <p><strong>shouldShowMetrics:</strong> {shouldShowMetrics ? 'true' : 'false'} (!!dashboardData || isCalculating)</p>
            <p><strong>Will show:</strong> {shouldShowMetrics ? 'Metrics View' : 'Onboarding View'}</p>
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
            onRefresh={refreshMetrics}
            isRefreshing={isLoading}
          />
        )}

        {/* Calculating state */}
        {isCalculating && !hasRealData && (
          <DashboardEmptyState 
            type="calculating" 
            isFetching={false}
          />
        )}

        {/* Show metrics and charts when we have data or are calculating */}
        {shouldShowMetrics && (
          <>
            {/* Metrics Grid */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
              <div className="bg-white p-6 rounded-lg border">
                <h3 className="text-sm font-medium text-gray-500">Monthly Recurring Revenue</h3>
                <p className="text-2xl font-bold text-gray-900 mt-2">$0.00</p>
              </div>
              <div className="bg-white p-6 rounded-lg border">
                <h3 className="text-sm font-medium text-gray-500">Total Revenue</h3>
                <p className="text-2xl font-bold text-gray-900 mt-2">$0.00</p>
              </div>
              <div className="bg-white p-6 rounded-lg border">
                <h3 className="text-sm font-medium text-gray-500">Active Subscriptions</h3>
                <p className="text-2xl font-bold text-gray-900 mt-2">0</p>
              </div>
              <div className="bg-white p-6 rounded-lg border">
                <h3 className="text-sm font-medium text-gray-500">Total Customers</h3>
                <p className="text-2xl font-bold text-gray-900 mt-2">0</p>
              </div>
            </div>

            {/* Charts placeholder */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <div className="bg-white p-6 rounded-lg border">
                <h3 className="text-lg font-semibold mb-4">Revenue Chart</h3>
                <div className="h-[300px] flex items-center justify-center bg-gray-50 rounded">
                  <p className="text-gray-500">Chart would load here with actual data</p>
                </div>
              </div>
              <div className="bg-white p-6 rounded-lg border">
                <h3 className="text-lg font-semibold mb-4">MRR Chart</h3>
                <div className="h-[300px] flex items-center justify-center bg-gray-50 rounded">
                  <p className="text-gray-500">Chart would load here with actual data</p>
                </div>
              </div>
            </div>

            {/* Tabs */}
            <Tabs defaultValue="customers" className="space-y-4">
              <TabsList>
                <TabsTrigger value="customers">Customers</TabsTrigger>
                <TabsTrigger value="subscriptions">Subscriptions</TabsTrigger>
                <TabsTrigger value="payments">Payments</TabsTrigger>
              </TabsList>
              
              <TabsContent value="customers" className="space-y-4">
                <div className="bg-white p-6 rounded-lg border">
                  <h3 className="text-lg font-semibold mb-4">Customer Chart</h3>
                  <div className="h-[300px] flex items-center justify-center bg-gray-50 rounded">
                    <p className="text-gray-500">Customer charts would load here</p>
                  </div>
                </div>
              </TabsContent>
              
              <TabsContent value="subscriptions" className="space-y-4">
                <div className="bg-white p-6 rounded-lg border">
                  <h3 className="text-lg font-semibold mb-4">Subscription Chart</h3>
                  <div className="h-[300px] flex items-center justify-center bg-gray-50 rounded">
                    <p className="text-gray-500">Subscription charts would load here</p>
                  </div>
                </div>
              </TabsContent>
              
              <TabsContent value="payments" className="space-y-4">
                <div className="bg-white p-6 rounded-lg border">
                  <h3 className="text-lg font-semibold mb-4">Payment Metrics</h3>
                  <div className="h-[300px] flex items-center justify-center bg-gray-50 rounded">
                    <p className="text-gray-500">Payment charts would load here</p>
                  </div>
                </div>
              </TabsContent>
            </Tabs>
          </>
        )}
      </div>
    </div>
  );
}