import React from 'react';
import { ChartDataPoint } from '../../types/session';
import { cn } from '../../utils/classNames';

interface SimpleBarChartProps {
  data: ChartDataPoint[];
  title?: string;
  timeRange?: number;
  timeGranularity?: 'minute' | 'hour' | 'day';
  onTimeRangeChange?: (hours: number) => void;
  onTimeGranularityChange?: (granularity: 'minute' | 'hour' | 'day') => void;
  showControls?: boolean;
  className?: string;
  isLoading?: boolean;
}

export const SimpleBarChart: React.FC<SimpleBarChartProps> = ({
  data,
  title = "Token Usage",
  timeRange = 168,
  timeGranularity = 'hour',
  onTimeRangeChange,
  onTimeGranularityChange,
  showControls = false,
  className,
  isLoading = false
}) => {
  // Show loading state
  if (isLoading) {
    return (
      <div className={cn("activity-card", className)}>
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-sm font-semibold text-white flex items-center">
            <span className="mr-2">🔥</span>
            {title}
          </h3>
        </div>
        <div className="h-48 flex items-center justify-center">
          <div className="flex items-center gap-2 text-gray-400">
            <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary"></div>
            <span>Loading chart data...</span>
          </div>
        </div>
      </div>
    );
  }
  
  // Filter out invalid data points and ensure we have valid values
  const validData = data.filter(d => d && typeof d.value === 'number');
  
  if (validData.length === 0) {
    return (
      <div className={cn("activity-card", className)}>
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-sm font-semibold text-white flex items-center">
            <span className="mr-2">🔥</span>
            {title}
          </h3>
        </div>
        <div className="h-48 flex items-center justify-center text-gray-500">
          No data available
        </div>
      </div>
    );
  }
  
  const maxValue = Math.max(...validData.map(d => d.value));

  return (
    <div className={cn("activity-card", className)}>
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-sm font-semibold text-white flex items-center">
          <span className="mr-2">🔥</span>
          {title}
        </h3>
        
        {showControls && onTimeRangeChange && onTimeGranularityChange && (
          <div className="flex gap-2">
            {/* Time Range Dropdown */}
            <select
              value={timeRange}
              onChange={(e) => onTimeRangeChange(Number(e.target.value))}
              className="px-2 py-1 text-xs bg-gray-800 border border-gray-700 rounded text-white focus:outline-none focus:border-primary"
            >
              <option value={24}>Last 24 hours</option>
              <option value={48}>Last 48 hours</option>
              <option value={72}>Last 3 days</option>
              <option value={168}>Last 7 days</option>
              <option value={720}>Last 30 days</option>
            </select>
            
            {/* Granularity Dropdown */}
            <select
              value={timeGranularity}
              onChange={(e) => onTimeGranularityChange(e.target.value as 'minute' | 'hour' | 'day')}
              className="px-2 py-1 text-xs bg-gray-800 border border-gray-700 rounded text-white focus:outline-none focus:border-primary"
            >
              <option value="minute">Minute</option>
              <option value="hour">Hourly</option>
              <option value="day">Daily</option>
            </select>
          </div>
        )}
      </div>
      
      <div className="space-y-2">
        {/* Chart bars */}
        <div className="h-48 flex items-end justify-start gap-1 px-2">
          {validData.map((point, index) => {
            const height = maxValue > 0 ? (point.value / maxValue) * 180 : 0; // max height 180px
            const barCount = validData.length;
            const barWidth = barCount > 0 ? 100 / barCount : 100;
            
            return (
              <div
                key={index}
                className="bg-gradient-to-t from-primary to-secondary rounded-t-sm transition-all duration-300 hover:opacity-80 cursor-pointer group relative"
                style={{ 
                  height: `${Math.max(height, 4)}px`, // minimum height of 4px
                  width: `${barWidth}%`, // evenly distribute across container
                  minWidth: '4px', // minimum width
                  maxWidth: '40px' // maximum width for readability
                }}
                title={`${point.label}: ${point.value.toLocaleString()} tokens`}
              >
                {/* Tooltip on hover */}
                <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-2 py-1 bg-gray-800 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none whitespace-nowrap z-10">
                  {point.label}: {point.value.toLocaleString()} tokens
                </div>
              </div>
            );
          })}
        </div>
        
        {/* X-axis labels */}
        <div className="flex justify-start gap-1 px-2 text-xs text-gray-500">
          {validData.map((point, index) => {
            const barCount = validData.length;
            const barWidth = barCount > 0 ? 100 / barCount : 100;
            // Show fewer labels for readability
            const showLabel = barCount <= 12 || index % Math.ceil(barCount / 12) === 0;
            
            return (
              <div
                key={index}
                className="text-center truncate"
                style={{ 
                  width: `${barWidth}%`, 
                  minWidth: '4px',
                  maxWidth: '40px'
                }}
              >
                {showLabel ? point.label : ''}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};