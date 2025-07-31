'use client';

import { useCustomerChart } from '@/hooks/use-analytics';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCompactNumber } from '@/lib/utils/format/format';
import { useState } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export function CustomerChart() {
  const [metric, setMetric] = useState<'total' | 'new' | 'churned' | 'growth_rate'>('total');
  const [days, setDays] = useState(30);
  
  const { data: chartData, isLoading, error } = useCustomerChart({ 
    metric, 
    period: 'daily', 
    days 
  });

  const metricOptions = [
    { value: 'total', label: 'Total Customers' },
    { value: 'new', label: 'New Customers' },
    { value: 'churned', label: 'Churned Customers' },
    { value: 'growth_rate', label: 'Growth Rate (%)' },
  ];

  if (error) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="text-center text-red-600">
            Failed to load customer data
          </div>
        </CardContent>
      </Card>
    );
  }

  const formatTooltipValue = (value: number) => {
    if (metric === 'growth_rate') {
      return `${value.toFixed(2)}%`;
    }
    return formatCompactNumber(value);
  };

  const formatYAxisValue = (value: number) => {
    if (metric === 'growth_rate') {
      return `${value}%`;
    }
    return formatCompactNumber(value);
  };

  const getLineColor = () => {
    switch (metric) {
      case 'new':
        return 'hsl(142, 76%, 36%)'; // green
      case 'churned':
        return 'hsl(0, 84%, 60%)'; // red
      case 'growth_rate':
        return 'hsl(217, 91%, 60%)'; // blue
      default:
        return 'hsl(var(--primary))';
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Customer Analytics</CardTitle>
          <div className="flex items-center gap-2">
            <Select value={metric} onValueChange={(v) => setMetric(v as any)}>
              <SelectTrigger className="w-40">
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
            <LineChart
              data={chartData?.data || []}
              margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
            >
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
              <Line
                type="monotone"
                dataKey="value"
                stroke={getLineColor()}
                strokeWidth={2}
                dot={false}
                name={chartData?.title || 'Customers'}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </CardContent>
    </Card>
  );
}