import { CypheraAPI, UserRequestContext } from './api';
import type {
  DashboardSummary,
  ChartData,
  PieChartData,
  PaymentMetrics,
  NetworkBreakdown,
  HourlyMetrics,
} from '@/types/analytics';

export interface AnalyticsParams {
  workspaceId: string;
  currency?: string;
}

export interface ChartParams extends AnalyticsParams {
  period?: 'hourly' | 'daily' | 'weekly' | 'monthly';
  days?: number;
}

export interface CustomerChartParams extends ChartParams {
  metric?: 'total' | 'new' | 'churned' | 'growth_rate';
}

export interface SubscriptionChartParams extends ChartParams {
  metric?: 'active' | 'new' | 'cancelled' | 'churn_rate';
}

export interface MRRChartParams extends AnalyticsParams {
  metric?: 'mrr' | 'arr';
  period?: 'daily' | 'weekly' | 'monthly';
  months?: number;
}

class AnalyticsService extends CypheraAPI {
  private userContext: UserRequestContext | null = null;

  /**
   * Set user context for authenticated requests
   */
  setUserContext(context: UserRequestContext) {
    this.userContext = context;
  }

  /**
   * Get dashboard summary metrics
   */
  async getDashboardSummary(params: AnalyticsParams): Promise<DashboardSummary> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) {
      queryParams.append('currency', params.currency);
    }

    const url = `${this.baseUrl}/analytics/dashboard${queryParams.toString() ? `?${queryParams}` : ''}`;
    console.log('DEBUG: Dashboard summary URL:', url);
    const response = await this.fetchWithRateLimit<DashboardSummary>(
      url,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get revenue chart data
   */
  async getRevenueChart(params: ChartParams): Promise<ChartData> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);
    if (params.period) queryParams.append('period', params.period);
    if (params.days) queryParams.append('days', params.days.toString());

    const response = await this.fetchWithRateLimit<ChartData>(
      `${this.baseUrl}/analytics/revenue-chart?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get customer chart data
   */
  async getCustomerChart(params: CustomerChartParams): Promise<ChartData> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);
    if (params.metric) queryParams.append('metric', params.metric);
    if (params.period) queryParams.append('period', params.period);
    if (params.days) queryParams.append('days', params.days.toString());

    const response = await this.fetchWithRateLimit<ChartData>(
      `${this.baseUrl}/analytics/customer-chart?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get subscription chart data
   */
  async getSubscriptionChart(params: SubscriptionChartParams): Promise<ChartData> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);
    if (params.metric) queryParams.append('metric', params.metric);
    if (params.period) queryParams.append('period', params.period);
    if (params.days) queryParams.append('days', params.days.toString());

    const response = await this.fetchWithRateLimit<ChartData>(
      `${this.baseUrl}/analytics/subscription-chart?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get MRR/ARR chart data
   */
  async getMRRChart(params: MRRChartParams): Promise<ChartData> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);
    if (params.metric) queryParams.append('metric', params.metric);
    if (params.period) queryParams.append('period', params.period);
    if (params.months) queryParams.append('months', params.months.toString());

    const response = await this.fetchWithRateLimit<ChartData>(
      `${this.baseUrl}/analytics/mrr-chart?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get payment metrics
   */
  async getPaymentMetrics(params: ChartParams): Promise<PaymentMetrics> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);
    if (params.days) queryParams.append('days', params.days.toString());

    const response = await this.fetchWithRateLimit<PaymentMetrics>(
      `${this.baseUrl}/analytics/payment-metrics?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get gas fee pie chart data
   */
  async getGasFeePieChart(params: ChartParams): Promise<PieChartData> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);
    if (params.days) queryParams.append('days', params.days.toString());

    const response = await this.fetchWithRateLimit<PieChartData>(
      `${this.baseUrl}/analytics/gas-fee-pie?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get network breakdown data
   */
  async getNetworkBreakdown(params: AnalyticsParams & { date?: string }): Promise<NetworkBreakdown> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);
    if (params.date) queryParams.append('date', params.date);

    const response = await this.fetchWithRateLimit<NetworkBreakdown>(
      `${this.baseUrl}/analytics/network-breakdown?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Get hourly metrics for today
   */
  async getHourlyMetrics(params: AnalyticsParams): Promise<HourlyMetrics> {
    if (!this.userContext) throw new Error('User context not set');
    
    const queryParams = new URLSearchParams();
    if (params.currency) queryParams.append('currency', params.currency);

    const response = await this.fetchWithRateLimit<HourlyMetrics>(
      `${this.baseUrl}/analytics/hourly?${queryParams}`,
      {
        method: 'GET',
        headers: this.getHeaders(this.userContext),
      }
    );

    return response;
  }

  /**
   * Trigger metrics refresh
   */
  async triggerMetricsRefresh(params: AnalyticsParams & { date?: string }): Promise<void> {
    if (!this.userContext) throw new Error('User context not set');
    
    await this.fetchWithRateLimit<void>(
      `${this.baseUrl}/analytics/refresh`,
      {
        method: 'POST',
        headers: this.getHeaders(this.userContext),
        body: JSON.stringify({
          date: params.date || new Date().toISOString(),
        }),
      }
    );
  }
}

export const analyticsService = new AnalyticsService();