'use client';

import { useSubscriptionChart } from '@/hooks/use-analytics';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCompactNumber, formatPercentage } from '@/lib/utils/format/format';
import { useState } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export function SubscriptionChart() {
  const [metric, setMetric] = useState<'active' | 'new' | 'cancelled' | 'churn_rate'>('active');
  const [days, setDays] = useState(30);
  
  const { data: chartData, isLoading, error } = useSubscriptionChart({ 
    metric, 
    period: 'daily', 
    days 
  });

  const metricOptions = [
    { value: 'active', label: 'Active Subscriptions' },
    { value: 'new', label: 'New Subscriptions' },
    { value: 'cancelled', label: 'Cancelled Subscriptions' },
    { value: 'churn_rate', label: 'Churn Rate (%)' },
  ];

  if (error) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="text-center text-red-600">
            Failed to load subscription data
          </div>
        </CardContent>
      </Card>
    );
  }

  const formatTooltipValue = (value: number) => {
    if (metric === 'churn_rate') {
      return formatPercentage(value);
    }
    return formatCompactNumber(value);
  };

  const formatYAxisValue = (value: number) => {
    if (metric === 'churn_rate') {
      return `${value}%`;
    }
    return formatCompactNumber(value);
  };

  const getAreaColor = () => {
    switch (metric) {
      case 'new':
        return 'hsl(142, 76%, 36%)'; // green
      case 'cancelled':
        return 'hsl(0, 84%, 60%)'; // red
      case 'churn_rate':
        return 'hsl(24, 95%, 53%)'; // orange
      default:
        return 'hsl(var(--primary))';
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Subscription Metrics</CardTitle>
          <div className="flex items-center gap-2">
            <Select value={metric} onValueChange={(v) => setMetric(v as any)}>
              <SelectTrigger className="w-44">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {metricOptions.map(opt => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={days.toString()} onValueChange={(v) => setDays(parseInt(v))}>
              <SelectTrigger className="w-24">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="7">7 days</SelectItem>
                <SelectItem value="14">14 days</SelectItem>
                <SelectItem value="30">30 days</SelectItem>
                <SelectItem value="60">60 days</SelectItem>
                <SelectItem value="90">90 days</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[300px] w-full" />
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart
              data={chartData?.data || []}
              margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
            >
              <defs>
                <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={getAreaColor()} stopOpacity={0.8}/>
                  <stop offset="95%" stopColor={getAreaColor()} stopOpacity={0.1}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis
                dataKey="date"
                className="text-xs"
                tickFormatter={(date) => {
                  const d = new Date(date);
                  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
                }}
              />
              <YAxis
                className="text-xs"
                tickFormatter={formatYAxisValue}
              />
              <Tooltip
                formatter={formatTooltipValue}
                labelFormatter={(label) => {
                  const d = new Date(label);
                  return d.toLocaleDateString('en-US', {
                    weekday: 'short',
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  });
                }}
                contentStyle={{
                  backgroundColor: 'hsl(var(--background))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '8px',
                }}
              />
              <Legend />
              <Area
                type="monotone"
                dataKey="value"
                stroke={getAreaColor()}
                fillOpacity={1}
                fill="url(#colorValue)"
                name={chartData?.title || 'Subscriptions'}
              />
            </AreaChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}