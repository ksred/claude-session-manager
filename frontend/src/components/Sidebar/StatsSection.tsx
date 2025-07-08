import React from 'react';
import { MetricsSummary } from '../../types/session';
import { formatTokens, formatCost, formatModel } from '../../utils/formatters';
import { cn } from '../../utils/classNames';

interface StatsSectionProps {
  stats: MetricsSummary;
  className?: string;
}

export const StatsSection: React.FC<StatsSectionProps> = ({ stats, className }) => {
  const statItems = [
    { label: 'Active Sessions', value: stats.active_sessions },
    { label: 'Total Sessions', value: stats.total_sessions },
    { label: 'Total Messages', value: stats.total_messages.toLocaleString() },
    { label: 'Tokens Used', value: formatTokens(stats.total_tokens_used) },
    { label: 'Total Cost', value: formatCost(stats.total_estimated_cost) },
    { label: 'Avg Duration', value: `${stats.average_session_duration_minutes.toFixed(1)}m` },
    { label: 'Top Model', value: formatModel(stats.most_used_model) }
  ];

  return (
    <div className={cn(
      "border-t border-gray-700 p-4 bg-gray-850",
      className
    )}>
      <div className="flex items-center text-primary text-xs font-semibold mb-3">
        <span className="mr-2">📊</span>
        System Stats
      </div>
      
      <div className="space-y-2">
        {statItems.map(item => (
          <div key={item.label} className="flex justify-between text-xs">
            <span className="text-gray-400">{item.label}</span>
            <span className="text-white font-medium">{item.value}</span>
          </div>
        ))}
      </div>
    </div>
  );
};