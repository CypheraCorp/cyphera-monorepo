export interface MoneyAmount {
  amount_cents: number;
  currency: string;
  formatted: string;
}

export interface RevenueGrowth {
  current_period: string;
  current_revenue: MoneyAmount;
  previous_revenue: MoneyAmount;
  growth_percentage: number;
}

export interface DashboardSummary {
  mrr: MoneyAmount;
  arr: MoneyAmount;
  total_revenue: MoneyAmount;
  active_subscriptions: number;
  total_customers: number;
  churn_rate: number;
  growth_rate: number;
  payment_success_rate: number;
  last_updated: string;
  revenue_growth?: RevenueGrowth;
  is_stale?: boolean;
  is_calculating?: boolean;
}

export interface ChartDataPoint {
  date: string;
  value: number;
  label?: string;
}

export interface ChartData {
  chart_type: string;
  title: string;
  data: ChartDataPoint[];
  period: string;
}

export interface PieChartDataPoint {
  label: string;
  value: number;
  color?: string;
}

export interface PieChartData {
  chart_type: string;
  title: string;
  data: PieChartDataPoint[];
  total: MoneyAmount;
}

export interface GasMetrics {
  total_gas_fees: MoneyAmount;
  sponsored_gas_fees: MoneyAmount;
}

export interface PaymentMetrics {
  total_successful: number;
  total_failed: number;
  total_volume: MoneyAmount;
  success_rate: number;
  gas_metrics: GasMetrics;
  period_days: number;
}

export interface NetworkMetrics {
  payments: number;
  volume_cents: number;
  gas_fee_cents: number;
}

export interface TokenMetrics {
  payments: number;
  volume_cents: number;
  avg_price_cents: number;
}

export interface NetworkBreakdown {
  date: string;
  networks: Record<string, NetworkMetrics>;
  tokens: Record<string, TokenMetrics>;
}

export interface HourlyDataPoint {
  hour: number;
  revenue: number;
  payments: number;
  new_users: number;
}

export interface HourlyMetrics {
  date: string;
  hourly_data: HourlyDataPoint[];
  currency: string;
}

export interface Currency {
  code: string;
  name: string;
  symbol: string;
  decimals: number;
  is_default: boolean;
}