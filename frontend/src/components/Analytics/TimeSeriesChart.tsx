import React, { useMemo } from 'react';
import { useUsageStats } from '../../hooks/useSessionData';
import { LoadingState } from '../Common/LoadingState';
import { ErrorMessage } from '../Common/ErrorMessage';
import { cn } from '../../utils/classNames';

interface TimeSeriesChartProps {
  className?: string;
}

export const TimeSeriesChart: React.FC<TimeSeriesChartProps> = ({ className }) => {
  const { data: usageData, isLoading, error } = useUsageStats();

  // Transform usage stats to time series format
  const data = useMemo(() => {
    if (!usageData?.daily_sessions) return null;

    return {
      period: 'day' as const,
      data: usageData.daily_sessions.map(day => ({
        date: day.date,
        sessions: day.count,
        messages: 0, // Not available in current API
        total_tokens: 0, // Not available in current API
        total_cost: 0, // Not available in current API
        models: {}
      }))
    };
  }, [usageData]);

  if (isLoading) {
    return <LoadingState message="Loading time series data..." />;
  }

  if (error) {
    return <ErrorMessage title="Failed to load time series" message={error.message} />;
  }

  if (!data || data.data.length === 0) {
    return (
      <div className={cn("analytics-section", className)}>
        <h3 className="text-lg font-semibold text-primary mb-4">Usage Trends</h3>
        <p className="text-gray-400">No usage data available</p>
      </div>
    );
  }

  // Calculate max value for scaling
  const maxValue = Math.max(...data.data.map(d => d.sessions));

  const formatValue = (value: number) => {
    return value.toString();
  };

  return (
    <div className={cn("analytics-section", className)}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-primary flex items-center">
          <span className="mr-2">📊</span>
          Usage Trends
        </h3>
        
        <div className="text-xs text-gray-400">
          Daily session count
        </div>
      </div>

      {/* Chart */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
        <div className="h-48 relative mb-2">
          <div className="absolute inset-0 flex items-end">
            {data.data.slice(-Math.min(data.data.length, 30)).map((day, index) => {
              const value = day.sessions;
              const height = maxValue > 0 ? (value / maxValue) * 100 : 0;
              
              return (
                <div 
                  key={index}
                  className="flex-1 px-0.5 h-full flex flex-col justify-end group"
                >
                  <div className="relative">
                    {/* Tooltip */}
                    <div className="absolute bottom-full mb-2 left-1/2 transform -translate-x-1/2 
                                  bg-gray-900 border border-gray-700 rounded px-2 py-1 text-xs 
                                  opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none
                                  whitespace-nowrap z-10">
                      <div className="text-primary">{day.date}</div>
                      <div className="text-white">{formatValue(value)} sessions</div>
                    </div>
                    
                    {/* Bar */}
                    <div 
                      className="bg-primary hover:bg-primary-light transition-colors rounded-t"
                      style={{ height: `${height}%`, minHeight: height > 0 ? '2px' : '0' }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
          
          {/* Y-axis labels */}
          <div className="absolute left-0 top-0 h-full flex flex-col justify-between text-xs text-gray-400 -ml-8">
            <span>{formatValue(maxValue)}</span>
            <span>{formatValue(maxValue / 2)}</span>
            <span>0</span>
          </div>
        </div>
        
        {/* X-axis labels */}
        <div className="flex justify-between text-xs text-gray-400 mt-2">
          <span>{data.data[0]?.date}</span>
          <span>{data.data[data.data.length - 1]?.date}</span>
        </div>
      </div>

      {/* Model distribution from usage stats */}
      {usageData?.model_usage && Object.keys(usageData.model_usage).length > 0 && (
        <div className="mt-4 p-3 bg-gray-800/50 rounded-lg border border-gray-700">
          <h4 className="text-sm font-medium text-white mb-2">Model Distribution</h4>
          <div className="space-y-2">
            {Object.entries(usageData.model_usage).map(([model, count]) => (
              <div key={model} className="flex justify-between text-xs">
                <span className="text-gray-400 truncate mr-2">{model.split('-').slice(0, 3).join('-')}</span>
                <span className="text-white">{count} sessions</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};