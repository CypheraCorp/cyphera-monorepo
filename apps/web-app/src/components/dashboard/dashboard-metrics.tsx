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

export function DashboardMetrics() {
  const { data: summary } = useDashboardSummary();

  // Don't show error state, just use default values
  // This provides a better user experience
  const metrics = [
    {
      title: 'Monthly Recurring Revenue',
      value: summary?.mrr?.formatted || '$0.00',
      icon: DollarSign,
      iconColor: 'text-green-600',
      subtitle: 'MRR',
    },
    {
      title: 'Total Revenue',
      value: summary?.total_revenue?.formatted || '$0.00',
      icon: TrendingUp,
      iconColor: 'text-purple-600',
      trend: summary?.revenue_growth ? {
        value: summary.revenue_growth.growth_percentage,
        isPositive: summary.revenue_growth.growth_percentage > 0,
      } : undefined,
    },
    {
      title: 'Active Subscriptions',
      value: formatCompactNumber(summary?.active_subscriptions || 0),
      icon: CreditCard,
      iconColor: 'text-indigo-600',
    },
    {
      title: 'Active Customers',
      value: formatCompactNumber(summary?.total_customers || 0),
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
        />
      ))}
    </div>
  );
}