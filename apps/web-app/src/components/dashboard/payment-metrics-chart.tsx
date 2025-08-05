'use client';

import { usePaymentMetrics } from '@/hooks/use-analytics';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatMoney, formatCompactNumber } from '@/lib/utils/format/format';
import { useState } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export function PaymentMetricsChart() {
  const [days, setDays] = useState(30);
  
  const { data: metricsData, isLoading, error } = usePaymentMetrics({ days });

  if (error) {
    return (
      <Card>
        <CardContent className="p-6">
          <div className="text-center text-red-600">
            Failed to load payment metrics
          </div>
        </CardContent>
      </Card>
    );
  }

  // Payment status data using actual PaymentMetrics interface
  const paymentStatusData = [
    {
      name: 'Successful',
      value: metricsData?.total_successful || 0,
      color: '#22c55e',
    },
    {
      name: 'Failed',
      value: metricsData?.total_failed || 0,
      color: '#ef4444',
    },
  ];

  const paymentVolumeData = [
    {
      period: `Last ${days} days`,
      volume: metricsData?.total_volume?.amount_cents || 0,
    },
  ];


  const formatVolumeTooltip = (value: number) => {
    return formatMoney(value, metricsData?.total_volume?.currency || 'USD');
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      {/* Payment Status Distribution */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Payment Status Distribution</CardTitle>
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
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Skeleton className="h-[300px] w-full" />
          ) : (
            <div>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={paymentStatusData}
                    cx="50%"
                    cy="50%"
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                    label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(0)}%`}
                  >
                    {paymentStatusData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value: number) => [formatCompactNumber(value), 'Payments']}
                    contentStyle={{
                      backgroundColor: 'hsl(var(--background))',
                      border: '1px solid hsl(var(--border))',
                      borderRadius: '8px',
                    }}
                  />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
              <div className="mt-4 grid grid-cols-3 gap-4 text-center">
                <div>
                  <p className="text-sm text-muted-foreground">Success Rate</p>
                  <p className="text-lg font-semibold text-green-600">
                    {metricsData?.success_rate ? `${(metricsData.success_rate * 100).toFixed(1)}%` : '0%'}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Total Payments</p>
                  <p className="text-lg font-semibold">
                    {formatCompactNumber((metricsData?.total_successful || 0) + (metricsData?.total_failed || 0))}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Total Volume</p>
                  <p className="text-lg font-semibold">
                    {metricsData?.total_volume?.formatted || '$0.00'}
                  </p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Payment Volume Over Time */}
      <Card>
        <CardHeader>
          <CardTitle>Payment Volume</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Skeleton className="h-[300px] w-full" />
          ) : (
            <div>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={paymentVolumeData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis 
                    dataKey="period" 
                    tick={{ fontSize: 12 }}
                    tickMargin={10}
                  />
                  <YAxis 
                    tick={{ fontSize: 12 }}
                    tickFormatter={(value) => formatMoney(value, metricsData?.total_volume?.currency || 'USD')}
                  />
                  <Tooltip
                    formatter={formatVolumeTooltip}
                    labelFormatter={(label) => `Period: ${label}`}
                    contentStyle={{
                      backgroundColor: 'hsl(var(--background))',
                      border: '1px solid hsl(var(--border))',
                      borderRadius: '8px',
                    }}
                  />
                  <Bar 
                    dataKey="volume" 
                    fill="#3b82f6" 
                    radius={[4, 4, 0, 0]}
                  />
                </BarChart>
              </ResponsiveContainer>
              <div className="mt-4 grid grid-cols-2 gap-4 text-center">
                <div>
                  <p className="text-sm text-muted-foreground">Average Payment</p>
                  <p className="text-lg font-semibold">
                    {(() => {
                      const totalPayments = (metricsData?.total_successful || 0) + (metricsData?.total_failed || 0);
                      const avgAmount = totalPayments > 0 ? (metricsData?.total_volume?.amount_cents || 0) / totalPayments : 0;
                      return formatMoney(avgAmount, metricsData?.total_volume?.currency || 'USD');
                    })()}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Payment Methods</p>
                  <p className="text-lg font-semibold text-blue-600">
                    Crypto Only
                  </p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}