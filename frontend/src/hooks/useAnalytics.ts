import { useQuery } from '@tanstack/react-query';
import { analyticsService } from '../services/analyticsService';

// Query key factory
export const analyticsKeys = {
  all: ['analytics'] as const,
  models: () => [...analyticsKeys.all, 'models'] as const,
  timeSeries: (period: string, days: number) => 
    [...analyticsKeys.all, 'timeseries', { period, days }] as const,
  costs: (groupBy: string, days: number) => 
    [...analyticsKeys.all, 'costs', { groupBy, days }] as const,
};

// Model Performance Hook
export const useModelPerformance = () => {
  return useQuery({
    queryKey: analyticsKeys.models(),
    queryFn: analyticsService.getModelPerformance,
    staleTime: 5 * 60 * 1000, // 5 minutes
    refetchInterval: 5 * 60 * 1000,
  });
};

// Time Series Hook
export const useTimeSeries = (
  period: 'day' | 'week' | 'month' = 'day',
  days: number = 30
) => {
  return useQuery({
    queryKey: analyticsKeys.timeSeries(period, days),
    queryFn: () => analyticsService.getTimeSeries(period, days),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 5 * 60 * 1000,
  });
};

// Cost Analytics Hook
export const useCostAnalytics = (
  groupBy: 'project' | 'model' | 'day' = 'project',
  days: number = 30
) => {
  return useQuery({
    queryKey: analyticsKeys.costs(groupBy, days),
    queryFn: () => analyticsService.getCostAnalytics(groupBy, days),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 5 * 60 * 1000,
  });
};