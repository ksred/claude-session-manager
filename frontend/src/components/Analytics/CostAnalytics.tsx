import React, { useState } from 'react';
import { useCostAnalytics } from '../../hooks/useAnalytics';
import { formatTokens, formatCost } from '../../utils/formatters';
import { LoadingState } from '../Common/LoadingState';
import { ErrorMessage } from '../Common/ErrorMessage';
import { cn } from '../../utils/classNames';

interface CostAnalyticsProps {
  className?: string;
}

export const CostAnalytics: React.FC<CostAnalyticsProps> = ({ className }) => {
  const [groupBy, setGroupBy] = useState<'project' | 'model' | 'day'>('project');
  const [days, setDays] = useState(30);
  
  const { data, isLoading, error } = useCostAnalytics(groupBy, days);

  if (isLoading) {
    return <LoadingState message="Loading cost analytics..." />;
  }

  if (error) {
    return <ErrorMessage title="Failed to load cost analytics" message={error.message} />;
  }

  if (!data) {
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
        
        <div className="flex gap-2">
          <select 
            value={groupBy}
            onChange={(e) => setGroupBy(e.target.value as any)}
            className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-white"
          >
            <option value="project">By Project</option>
            <option value="model">By Model</option>
            <option value="day">By Day</option>
          </select>
          
          <select 
            value={days}
            onChange={(e) => setDays(Number(e.target.value))}
            className="bg-gray-800 border border-gray-700 rounded px-2 py-1 text-xs text-white"
          >
            <option value="7">Last 7 days</option>
            <option value="30">Last 30 days</option>
            <option value="90">Last 90 days</option>
          </select>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-gray-400">Total Cost</span>
            <span className="text-xs text-warning">⚡</span>
          </div>
          <p className="text-2xl font-bold text-white">{formatCost(data.total_cost)}</p>
          <p className="text-xs text-gray-400 mt-1">Last {days} days</p>
        </div>
        
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-gray-400">Cache Savings</span>
            <span className="text-xs text-success">💚</span>
          </div>
          <p className="text-2xl font-bold text-success">{formatCost(data.cache_savings)}</p>
          <p className="text-xs text-gray-400 mt-1">
            {((data.cache_savings / (data.total_cost + data.cache_savings)) * 100).toFixed(1)}% saved
          </p>
        </div>
        
        <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs text-gray-400">Monthly Projection</span>
            <span className="text-xs text-blue-400">📈</span>
          </div>
          <p className="text-2xl font-bold text-white">{formatCost(data.projection.monthly_estimate)}</p>
          <p className="text-xs text-gray-400 mt-1">
            ${data.projection.daily_average.toFixed(2)}/day avg
          </p>
        </div>
      </div>

      {/* Cost Breakdown */}
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
        <h4 className="text-sm font-medium text-white mb-3">Cost Breakdown</h4>
        
        {/* Visual bar */}
        <div className="h-8 bg-gray-700 rounded-full overflow-hidden mb-4 flex">
          {data.breakdown.slice(0, 5).map((item, index) => (
            <div
              key={index}
              className={cn(getItemColor(index), "transition-all duration-300")}
              style={{ width: `${item.percentage * 100}%` }}
              title={`${item.name}: ${(item.percentage * 100).toFixed(1)}%`}
            />
          ))}
          {data.breakdown.length > 5 && (
            <div
              className="bg-gray-600"
              style={{ 
                width: `${data.breakdown.slice(5).reduce((sum, item) => sum + item.percentage, 0) * 100}%` 
              }}
              title="Others"
            />
          )}
        </div>
        
        {/* Detailed breakdown */}
        <div className="space-y-3">
          {data.breakdown.slice(0, 5).map((item, index) => (
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
          
          {data.breakdown.length > 5 && (
            <div className="text-xs text-gray-400 italic">
              +{data.breakdown.length - 5} more items...
            </div>
          )}
        </div>
        
        {/* Token efficiency */}
        <div className="mt-4 pt-4 border-t border-gray-700">
          <div className="flex justify-between text-xs">
            <span className="text-gray-400">Cache Token Usage</span>
            <span className="text-white">
              {data.breakdown.reduce((sum, item) => sum + item.tokens.cached, 0).toLocaleString()} / 
              {data.breakdown.reduce((sum, item) => sum + item.tokens.total, 0).toLocaleString()} tokens
            </span>
          </div>
          <div className="mt-2 h-2 bg-gray-700 rounded-full overflow-hidden">
            <div 
              className="h-full bg-success transition-all duration-500"
              style={{ 
                width: `${(
                  data.breakdown.reduce((sum, item) => sum + item.tokens.cached, 0) / 
                  data.breakdown.reduce((sum, item) => sum + item.tokens.total, 0)
                ) * 100}%` 
              }}
            />
          </div>
        </div>
      </div>
    </div>
  );
};