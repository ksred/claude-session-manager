import React, { useMemo } from 'react';
import { useAllSessions, useMetricsSummary } from '../../hooks/useSessionData';
import { formatTokens, formatCost } from '../../utils/formatters';
import { LoadingState } from '../Common/LoadingState';
import { ErrorMessage } from '../Common/ErrorMessage';
import { cn } from '../../utils/classNames';
import { groupSessionsByProject } from '../../utils/projectHelpers';

interface CostAnalyticsProps {
  className?: string;
}

export const CostAnalytics: React.FC<CostAnalyticsProps> = ({ className }) => {
  const { data: sessionsData, isLoading: sessionsLoading, error: sessionsError } = useAllSessions();
  const { data: metricsData, isLoading: metricsLoading, error: metricsError } = useMetricsSummary();

  const costData = useMemo(() => {
    if (!sessionsData?.sessions || !metricsData) return null;

    const sessions = sessionsData.sessions;
    const totalCost = metricsData.total_estimated_cost || 0;
    
    // Calculate cache savings (estimate: cache tokens save ~90% of cost)
    let totalCacheTokens = 0;
    let totalTokens = 0;
    
    sessions.forEach(session => {
      totalCacheTokens += session.tokens_used.cache_creation_input_tokens + session.tokens_used.cache_read_input_tokens;
      totalTokens += session.tokens_used.total_tokens;
    });
    
    const cacheRatio = totalTokens > 0 ? totalCacheTokens / totalTokens : 0;
    const cacheSavings = totalCost * cacheRatio * 0.9; // 90% savings on cached tokens

    // Group by project
    const projects = groupSessionsByProject(sessions);
    const breakdown = projects.map(project => ({
      name: project.name,
      cost: project.totalCost,
      tokens: {
        total: project.totalTokens.total_tokens,
        cached: project.totalTokens.cache_creation_input_tokens + project.totalTokens.cache_read_input_tokens,
        fresh: project.totalTokens.total_tokens - (project.totalTokens.cache_creation_input_tokens + project.totalTokens.cache_read_input_tokens)
      },
      sessions: project.totalSessions,
      percentage: totalCost > 0 ? project.totalCost / totalCost : 0
    })).sort((a, b) => b.cost - a.cost);

    // Calculate projections
    const avgSessionsPerDay = metricsData.total_sessions / 30; // Assume 30 days
    const avgCostPerSession = metricsData.total_sessions > 0 ? totalCost / metricsData.total_sessions : 0;
    const dailyAverage = avgSessionsPerDay * avgCostPerSession;
    const monthlyEstimate = dailyAverage * 30;

    return {
      total_cost: totalCost,
      cache_savings: cacheSavings,
      breakdown,
      projection: {
        daily_average: dailyAverage,
        monthly_estimate: monthlyEstimate
      }
    };
  }, [sessionsData, metricsData]);

  if (sessionsLoading || metricsLoading) {
    return <LoadingState message="Loading cost analytics..." />;
  }

  if (sessionsError || metricsError) {
    return <ErrorMessage title="Failed to load cost analytics" message={sessionsError?.message || metricsError?.message || 'Unknown error'} />;
  }

  if (!costData) {
    return (
      <div className={cn("analytics-section", className)}>
        <h3 className="text-lg font-semibold text-primary mb-4">Cost Analytics</h3>
        <p className="text-gray-400">No cost data available</p>
      </div>
    );
  }

  const getItemColor = (index: number) => {
    const colors = ['bg-purple-500', 'bg-blue-500', 'bg-green-500', 'bg-yellow-500', 'bg-pink-500'];
    return colors[index % colors.length];
  };

  return (
    <div className={cn("analytics-section", className)}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-primary flex items-center">
          <span className="mr-2">💰</span>
          Cost Analytics
        </h3>
        
        <div className="text-xs text-gray-400">
          Project cost breakdown
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-gray-400">Total Cost</span>
            <span className="text-xs text-warning">⚡</span>
          </div>
          <p className="text-2xl font-bold text-white">{formatCost(costData.total_cost)}</p>
          <p className="text-xs text-gray-400 mt-1">All time</p>
        </div>
        
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-gray-400">Cache Savings</span>
            <span className="text-xs text-success">💚</span>
          </div>
          <p className="text-2xl font-bold text-success">{formatCost(costData.cache_savings)}</p>
          <p className="text-xs text-gray-400 mt-1">
            {((costData.cache_savings / (costData.total_cost + costData.cache_savings)) * 100).toFixed(1)}% saved
          </p>
        </div>
        
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-gray-400">Monthly Projection</span>
            <span className="text-xs text-blue-400">📈</span>
          </div>
          <p className="text-2xl font-bold text-white">{formatCost(costData.projection.monthly_estimate)}</p>
          <p className="text-xs text-gray-400 mt-1">
            ${costData.projection.daily_average.toFixed(2)}/day avg
          </p>
        </div>
      </div>

      {/* Cost Breakdown */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
        <h4 className="text-sm font-medium text-white mb-3">Cost Breakdown</h4>
        
        {/* Visual bar */}
        <div className="h-8 bg-gray-700 rounded-full overflow-hidden mb-4 flex">
          {costData.breakdown.slice(0, 5).map((item, index) => (
            <div
              key={index}
              className={cn(getItemColor(index), "transition-all duration-300")}
              style={{ width: `${item.percentage * 100}%` }}
              title={`${item.name}: ${(item.percentage * 100).toFixed(1)}%`}
            />
          ))}
          {costData.breakdown.length > 5 && (
            <div
              className="bg-gray-600"
              style={{ 
                width: `${costData.breakdown.slice(5).reduce((sum, item) => sum + item.percentage, 0) * 100}%` 
              }}
              title="Others"
            />
          )}
        </div>
        
        {/* Detailed breakdown */}
        <div className="space-y-3">
          {costData.breakdown.slice(0, 5).map((item, index) => (
            <div key={index} className="flex items-center justify-between">
              <div className="flex items-center gap-2 flex-1 min-w-0">
                <div className={cn("w-3 h-3 rounded-full flex-shrink-0", getItemColor(index))} />
                <span className="text-sm text-white truncate">{item.name}</span>
              </div>
              
              <div className="flex items-center gap-4 text-xs">
                <div className="text-right">
                  <p className="text-white font-medium">{formatCost(item.cost)}</p>
                  <p className="text-gray-400">{(item.percentage * 100).toFixed(1)}%</p>
                </div>
                
                <div className="text-right text-gray-400">
                  <p>{formatTokens(item.tokens.total)} tokens</p>
                  <p>{item.sessions} sessions</p>
                </div>
              </div>
            </div>
          ))}
          
          {costData.breakdown.length > 5 && (
            <div className="text-xs text-gray-400 italic">
              +{costData.breakdown.length - 5} more items...
            </div>
          )}
        </div>
        
        {/* Token efficiency */}
        <div className="mt-4 pt-4 border-t border-gray-700">
          <div className="flex justify-between text-xs">
            <span className="text-gray-400">Cache Token Usage</span>
            <span className="text-white">
              {costData.breakdown.reduce((sum, item) => sum + item.tokens.cached, 0).toLocaleString()} / 
              {costData.breakdown.reduce((sum, item) => sum + item.tokens.total, 0).toLocaleString()} tokens
            </span>
          </div>
          <div className="mt-2 h-2 bg-gray-700 rounded-full overflow-hidden">
            <div 
              className="h-full bg-success transition-all duration-500"
              style={{ 
                width: `${(
                  costData.breakdown.reduce((sum, item) => sum + item.tokens.cached, 0) / 
                  costData.breakdown.reduce((sum, item) => sum + item.tokens.total, 0)
                ) * 100}%` 
              }}
            />
          </div>
        </div>
      </div>
    </div>
  );
};