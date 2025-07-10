import React from 'react';
import { cn } from '../../utils/classNames';

interface MetricCardProps {
  title: string;
  value: string | number;
  change?: string;
  icon: React.ReactNode;
  trend?: 'up' | 'down' | 'neutral';
  className?: string;
}

export const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  change,
  icon,
  trend = 'neutral',
  className
}) => {
  return (
    <div className={cn("metric-card", className)}>
      <div className="flex justify-between items-center mb-3">
        <h3 className="text-xs text-gray-400 font-medium">{title}</h3>
        <div className="text-primary text-sm">{icon}</div>
      </div>
      
      <div className="text-2xl font-bold text-white mb-1">
        {typeof value === 'number' ? value.toLocaleString() : value}
      </div>
      
      {change && (
        <div className={cn(
          "text-xs",
          trend === 'up' && "text-success",
          trend === 'down' && "text-error",
          trend === 'neutral' && "text-primary"
        )}>
          {change}
        </div>
      )}
    </div>
  );
};