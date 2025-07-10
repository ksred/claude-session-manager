import React from 'react';
import { Session } from '../../types/session';
import { MetricCard } from './MetricCard';
import { formatTokens, formatCost, formatDuration, formatModel } from '../../utils/formatters';
import { cn } from '../../utils/classNames';

interface MetricsGridProps {
  session: Session | null;
  className?: string;
}

export const MetricsGrid: React.FC<MetricsGridProps> = ({ session, className }) => {
  if (!session) {
    return (
      <div className={cn(
        "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 xl:grid-cols-6 gap-4 mb-6",
        className
      )}>
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="metric-card animate-pulse">
            <div className="h-4 bg-gray-700 rounded mb-3"></div>
            <div className="h-8 bg-gray-700 rounded mb-1"></div>
            <div className="h-3 bg-gray-700 rounded w-2/3"></div>
          </div>
        ))}
      </div>
    );
  }

  const cacheTokens = session.tokens_used.cache_creation_input_tokens + session.tokens_used.cache_read_input_tokens;
  const cachePercentage = ((cacheTokens / session.tokens_used.total_tokens) * 100).toFixed(1);

  const metrics = [
    {
      title: 'Total Tokens',
      value: formatTokens(session.tokens_used.total_tokens),
      change: `${session.message_count} messages`,
      icon: '🔢',
      trend: 'neutral' as const
    },
    {
      title: 'Session Cost',
      value: formatCost(session.tokens_used.estimated_cost),
      change: `${formatModel(session.model)}`,
      icon: '💰',
      trend: 'up' as const
    },
    {
      title: 'Cache Tokens',
      value: formatTokens(cacheTokens),
      change: `${cachePercentage}% cache hit`,
      icon: '⚡',
      trend: 'up' as const
    },
    {
      title: 'Input/Output',
      value: `${formatTokens(session.tokens_used.input_tokens)}/${formatTokens(session.tokens_used.output_tokens)}`,
      change: 'In/Out ratio',
      icon: '📊',
      trend: 'neutral' as const
    },
    {
      title: 'Files Modified',
      value: session.files_modified?.length || 0,
      change: 'Changed files',
      icon: '📝',
      trend: 'neutral' as const
    },
    {
      title: 'Session Time',
      value: formatDuration(session.duration_seconds),
      change: session.is_active ? 'Currently active' : 'Session ended',
      icon: '⏱️',
      trend: session.is_active ? 'up' as const : 'neutral' as const
    }
  ];

  return (
    <div className={cn(
      "grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4 mb-6",
      className
    )}>
      {metrics.map(metric => (
        <MetricCard
          key={metric.title}
          title={metric.title}
          value={metric.value}
          change={metric.change}
          icon={metric.icon}
          trend={metric.trend}
        />
      ))}
    </div>
  );
};