import React, { useMemo } from 'react';
import { useAllSessions, useMetricsSummary } from '../../hooks/useSessionData';
import { formatTokens, formatCost } from '../../utils/formatters';
import { LoadingState } from '../Common/LoadingState';
import { ErrorMessage } from '../Common/ErrorMessage';
import { cn } from '../../utils/classNames';

interface ModelPerformanceProps {
  className?: string;
}

export const ModelPerformance: React.FC<ModelPerformanceProps> = ({ className }) => {
  const { data: sessionsData, isLoading: sessionsLoading, error: sessionsError } = useAllSessions();
  const { isLoading: metricsLoading, error: metricsError } = useMetricsSummary();

  const modelStats = useMemo(() => {
    if (!sessionsData?.sessions) return [];

    const models = new Map<string, {
      total_sessions: number;
      total_tokens: number;
      total_cost: number;
      total_cache_tokens: number;
      total_duration: number;
    }>();

    sessionsData.sessions.forEach(session => {
      const model = session.model || 'unknown';
      const stats = models.get(model) || {
        total_sessions: 0,
        total_tokens: 0,
        total_cost: 0,
        total_cache_tokens: 0,
        total_duration: 0
      };

      stats.total_sessions++;
      stats.total_tokens += session.tokens_used.total_tokens;
      stats.total_cost += session.tokens_used.estimated_cost;
      stats.total_cache_tokens += session.tokens_used.cache_creation_input_tokens + session.tokens_used.cache_read_input_tokens;
      stats.total_duration += session.duration_seconds;

      models.set(model, stats);
    });

    return Array.from(models.entries()).map(([model, stats]) => ({
      model,
      display_name: model.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' '),
      stats: {
        total_sessions: stats.total_sessions,
        total_tokens: stats.total_tokens,
        total_cost: stats.total_cost,
        avg_tokens_per_session: Math.round(stats.total_tokens / stats.total_sessions),
        avg_cost_per_session: stats.total_cost / stats.total_sessions,
        cache_efficiency: stats.total_tokens > 0 ? stats.total_cache_tokens / stats.total_tokens : 0,
        avg_session_duration_seconds: Math.round(stats.total_duration / stats.total_sessions)
      }
    }));
  }, [sessionsData]);

  if (sessionsLoading || metricsLoading) {
    return <LoadingState message="Loading model performance..." />;
  }

  if (sessionsError || metricsError) {
    return <ErrorMessage title="Failed to load model performance" message={sessionsError?.message || metricsError?.message || 'Unknown error'} />;
  }

  if (modelStats.length === 0) {
    return (
      <div className={cn("analytics-section", className)}>
        <h3 className="text-lg font-semibold text-primary mb-4">Model Performance</h3>
        <p className="text-gray-400">No model data available</p>
      </div>
    );
  }

  const getModelColor = (model: string) => {
    if (model.includes('opus')) return 'text-purple-400';
    if (model.includes('sonnet')) return 'text-blue-400';
    if (model.includes('haiku')) return 'text-green-400';
    return 'text-gray-400';
  };

  return (
    <div className={cn("analytics-section", className)}>
      <h3 className="text-lg font-semibold text-primary mb-4 flex items-center">
        <span className="mr-2">🤖</span>
        Model Performance Comparison
      </h3>

      <div className="space-y-4">
        {modelStats.map((model) => (
          <div 
            key={model.model}
            className="bg-gray-800 border border-gray-700 rounded-lg p-4 hover:border-primary/50 transition-colors"
          >
            <div className="flex items-center justify-between mb-3">
              <h4 className={cn("font-semibold", getModelColor(model.model))}>
                {model.display_name}
              </h4>
              <span className="text-xs text-gray-500 font-mono">{model.model}</span>
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <div>
                <p className="text-xs text-gray-400">Sessions</p>
                <p className="text-sm font-medium text-white">{model.stats.total_sessions}</p>
              </div>
              
              <div>
                <p className="text-xs text-gray-400">Avg Cost/Session</p>
                <p className="text-sm font-medium text-white">
                  {formatCost(model.stats.avg_cost_per_session)}
                </p>
              </div>
              
              <div>
                <p className="text-xs text-gray-400">Avg Tokens/Session</p>
                <p className="text-sm font-medium text-white">
                  {formatTokens(model.stats.avg_tokens_per_session)}
                </p>
              </div>
              
              <div>
                <p className="text-xs text-gray-400">Cache Efficiency</p>
                <p className="text-sm font-medium text-white">
                  {(model.stats.cache_efficiency * 100).toFixed(1)}%
                </p>
              </div>
            </div>

            <div className="mt-3 pt-3 border-t border-gray-700">
              <div className="flex justify-between text-xs">
                <span className="text-gray-400">Total Cost</span>
                <span className="text-white font-medium">{formatCost(model.stats.total_cost)}</span>
              </div>
              <div className="flex justify-between text-xs mt-1">
                <span className="text-gray-400">Total Tokens</span>
                <span className="text-white font-medium">{formatTokens(model.stats.total_tokens)}</span>
              </div>
            </div>

            {/* Performance bar visualization */}
            <div className="mt-3">
              <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
                <div 
                  className={cn("h-full transition-all duration-500", {
                    'bg-purple-500': model.model.includes('opus'),
                    'bg-blue-500': model.model.includes('sonnet'),
                    'bg-green-500': model.model.includes('haiku'),
                  })}
                  style={{ 
                    width: `${(model.stats.total_cost / Math.max(...modelStats.map(m => m.stats.total_cost))) * 100}%` 
                  }}
                />
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Summary insights */}
      {modelStats.length > 0 && (
        <div className="mt-4 p-3 bg-gray-800/50 rounded-lg border border-gray-700">
          <p className="text-xs text-gray-400">
            💡 Most cost-effective: {
              modelStats.reduce((best, current) => 
                current.stats.avg_cost_per_session < best.stats.avg_cost_per_session ? current : best
              ).display_name
            } • 
            Best cache efficiency: {
              modelStats.reduce((best, current) => 
                current.stats.cache_efficiency > best.stats.cache_efficiency ? current : best
              ).display_name
            }
          </p>
        </div>
      )}
    </div>
  );
};