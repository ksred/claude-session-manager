// Utility functions for formatting data
import { TokenTimelinePoint } from '../types/api';
export const formatTime = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / (1000 * 60));
  
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m`;
  if (minutes < 1440) return `${Math.floor(minutes / 60)}h`;
  return `${Math.floor(minutes / 1440)}d`;
};

export const formatTokens = (tokens: number): string => {
  if (tokens < 1000) return tokens.toString();
  if (tokens < 1000000) return `${(tokens / 1000).toFixed(1)}K`;
  return `${(tokens / 1000000).toFixed(1)}M`;
};

export const formatCost = (cost: number): string => {
  return `$${cost.toFixed(2)}`;
};

export const formatDuration = (durationSeconds: number): string => {
  const hours = Math.floor(durationSeconds / 3600);
  const minutes = Math.floor((durationSeconds % 3600) / 60);
  
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
};

export const formatModel = (model: string): string => {
  const modelMap: Record<string, string> = {
    'claude-3-5-sonnet-20241022': 'Sonnet 3.5',
    'claude-3-opus-20240229': 'Opus 3',
    'claude-3-haiku-20240307': 'Haiku 3'
  };
  return modelMap[model] || model;
};

export const transformTokenTimelineToChartData = (timeline: TokenTimelinePoint[]) => {
  if (!timeline || !Array.isArray(timeline)) {
    return [];
  }
  
  // Transform the data without filtering by time (backend already handles time range)
  return timeline
    .filter(point => point && typeof point.total_tokens === 'number')
    .map(point => {
      const date = new Date(point.timestamp);
      return {
        timestamp: date,
        value: point.total_tokens || 0,
        label: point.label || formatTimeLabel(date)
      };
    })
    .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
};

const formatTimeLabel = (date: Date): string => {
  const hour = date.getHours();
  const day = date.getDate();
  const month = date.getMonth() + 1;
  
  // Check if we're looking at hourly or daily data
  // If all timestamps are at midnight, it's likely daily data
  if (hour === 0) {
    return `${month}/${day}`;
  }
  
  // Otherwise show hour
  return `${hour}:00`;
};

export const transformTokenTimelineToLineChartData = (timeline: TokenTimelinePoint[]) => {
  if (!timeline || !Array.isArray(timeline)) {
    return [];
  }
  
  return timeline
    .filter(point => point && typeof point.total_tokens === 'number')
    .map(point => {
      const date = new Date(point.timestamp);
      return {
        timestamp: date,
        values: {
          cost: point.estimated_cost || 0,
          messages: point.message_count || 0
        },
        label: point.label || formatTimeLabel(date)
      };
    })
    .sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
};