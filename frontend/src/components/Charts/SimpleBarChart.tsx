import React from 'react';
import { ChartDataPoint } from '../../types/session';
import { cn } from '../../utils/classNames';

interface SimpleBarChartProps {
  data: ChartDataPoint[];
  title?: string;
  className?: string;
}

export const SimpleBarChart: React.FC<SimpleBarChartProps> = ({
  data,
  title = "Token Usage (Last 24h)",
  className
}) => {
  const maxValue = Math.max(...data.map(d => d.value));

  return (
    <div className={cn("activity-card", className)}>
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-sm font-semibold text-white flex items-center">
          <span className="mr-2">🔥</span>
          {title}
        </h3>
      </div>
      
      <div className="space-y-2">
        {/* Chart bars */}
        <div className="h-48 flex items-end justify-start gap-1 px-2">
          {data.slice(-24).map((point, index) => {
            const height = (point.value / maxValue) * 180; // max height 180px
            
            return (
              <div
                key={index}
                className="bg-gradient-to-t from-primary to-secondary rounded-t-sm transition-all duration-300 hover:opacity-80 cursor-pointer group relative"
                style={{ 
                  height: `${Math.max(height, 4)}px`, // minimum height of 4px
                  width: `${100 / 24}%`, // evenly distribute across container
                  minWidth: '8px' // minimum width
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
        
        {/* Hour labels */}
        <div className="flex justify-start gap-1 px-2 text-xs text-gray-500">
          {data.slice(-24).map((point, index) => (
            <div
              key={index}
              className="text-center"
              style={{ width: `${100 / 24}%`, minWidth: '8px' }}
            >
              {index % 4 === 0 ? point.label : ''}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};