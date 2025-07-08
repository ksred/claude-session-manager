import api from './api';

// Model Performance Types
export interface ModelStats {
  total_sessions: number;
  total_tokens: number;
  total_cost: number;
  avg_tokens_per_session: number;
  avg_cost_per_session: number;
  cache_efficiency: number;
  avg_session_duration_seconds: number;
}

export interface ModelPerformance {
  model: string;
  display_name: string;
  stats: ModelStats;
}

export interface ModelPerformanceResponse {
  models: ModelPerformance[];
}

// Time Series Types
export interface ModelUsage {
  sessions: number;
  tokens: number;
}

export interface DailyMetric {
  date: string;
  sessions: number;
  messages: number;
  total_tokens: number;
  total_cost: number;
  models: Record<string, ModelUsage>;
}

export interface TimeSeriesResponse {
  period: 'day' | 'week' | 'month';
  data: DailyMetric[];
}

// Cost Analytics Types
export interface TokenBreakdown {
  total: number;
  cached: number;
  fresh: number;
}

export interface CostBreakdownItem {
  name: string;
  cost: number;
  tokens: TokenBreakdown;
  sessions: number;
  percentage: number;
}

export interface CostProjection {
  daily_average: number;
  monthly_estimate: number;
}

export interface CostAnalyticsResponse {
  total_cost: number;
  cache_savings: number;
  breakdown: CostBreakdownItem[];
  projection: CostProjection;
}

// Analytics Service
export const analyticsService = {
  getModelPerformance: async (): Promise<ModelPerformanceResponse> => {
    const response = await api.get('/analytics/models');
    return response.data;
  },

  getTimeSeries: async (
    period: 'day' | 'week' | 'month' = 'day',
    days: number = 30
  ): Promise<TimeSeriesResponse> => {
    const response = await api.get('/analytics/timeseries', {
      params: { period, days }
    });
    return response.data;
  },

  getCostAnalytics: async (
    groupBy: 'project' | 'model' | 'day' = 'project',
    days: number = 30
  ): Promise<CostAnalyticsResponse> => {
    const response = await api.get('/analytics/costs', {
      params: { group_by: groupBy, days }
    });
    return response.data;
  }
};