'use client';

import { useDashboardSummary } from '@/hooks/use-analytics';
import { MetricCard } from './metric-card';
import {
  TrendingUp,
  Users,
  CreditCard,
  DollarSign,
} from 'lucide-react';
import { formatCompactNumber } from '@/lib/utils/format/format';
import type { DashboardSummary } from '@/types/analytics';

export function DashboardMetrics() {
  const { data: summary, error, isFetching } = useDashboardSummary();

  if (error && !summary) {
    return (
      <div className="text-center p-6 text-red-600">
        Failed to load dashboard metrics. Please try again later.
      </div>
    );
  }

  // Cast to expected type
  const dashboardData = summary as DashboardSummary & { is_calculating?: boolean };

  const metrics = [
    {
      title: 'Monthly Recurring Revenue',
      value: dashboardData?.mrr?.formatted || '$0.00',
      icon: DollarSign,
      iconColor: 'text-green-600',
      subtitle: 'MRR',
    },
    {
      title: 'Total Revenue',
      value: dashboardData?.total_revenue?.formatted || '$0.00',
      icon: TrendingUp,
      iconColor: 'text-purple-600',
      trend: dashboardData?.revenue_growth ? {
        value: dashboardData.revenue_growth.growth_percentage,
        isPositive: dashboardData.revenue_growth.growth_percentage > 0,
      } : undefined,
    },
    {
      title: 'Active Subscriptions',
      value: formatCompactNumber(dashboardData?.active_subscriptions || 0),
      icon: CreditCard,
      iconColor: 'text-indigo-600',
    },
    {
      title: 'Active Customers',
      value: formatCompactNumber(dashboardData?.total_customers || 0),
      icon: Users,
      iconColor: 'text-orange-600',
    },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      {metrics.map((metric, index) => (
        <MetricCard
          key={index}
          {...metric}
          loading={dashboardData?.is_calculating && isFetching}
        />
      ))}
    </div>
  );
}