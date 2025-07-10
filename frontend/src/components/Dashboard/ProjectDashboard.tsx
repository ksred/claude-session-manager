import React from 'react';
import { Project } from '../../types/project';
import { ChartDataPoint } from '../../types/session';
import { formatTokens, formatCost, formatDuration, formatModel } from '../../utils/formatters';
import { MetricCard } from './MetricCard';
import { FilesList } from '../Common/FilesList';
import { SimpleBarChart } from '../Charts/SimpleBarChart';
import { SimpleLineChart, LineChartDataPoint } from '../Charts/SimpleLineChart';
import { cn } from '../../utils/classNames';

interface ProjectDashboardProps {
  selectedProject: Project | null;
  chartData: ChartDataPoint[];
  lineChartData?: LineChartDataPoint[];
  timeRange: number;
  timeGranularity: 'hour' | 'day';
  onTimeRangeChange: (hours: number) => void;
  onTimeGranularityChange: (granularity: 'hour' | 'day') => void;
  onSessionSelect: (sessionId: string) => void;
  onRefresh?: () => void;
  className?: string;
}

export const ProjectDashboard: React.FC<ProjectDashboardProps> = ({
  selectedProject,
  chartData,
  lineChartData,
  timeRange,
  timeGranularity,
  onTimeRangeChange,
  onTimeGranularityChange,
  onSessionSelect,
  onRefresh,
  className
}) => {
  if (!selectedProject) {
    return (
      <div className={cn(
        "flex-1 p-5 flex items-center justify-center text-gray-400",
        className
      )}>
        <div className="text-center">
          <div className="text-6xl mb-4">📁</div>
          <h2 className="text-xl font-semibold mb-2">No Project Selected</h2>
          <p>Select a project from the sidebar to view details</p>
        </div>
      </div>
    );
  }

  const mostUsedModel = Object.entries(selectedProject.models)
    .sort(([, a], [, b]) => b - a)[0] || ['Unknown', 0];

  const cacheTokens = selectedProject.totalTokens.cache_creation_input_tokens + 
                     selectedProject.totalTokens.cache_read_input_tokens;
  const cachePercentage = selectedProject.totalTokens.total_tokens > 0 
    ? ((cacheTokens / selectedProject.totalTokens.total_tokens) * 100).toFixed(1)
    : '0';

  const metrics = [
    {
      title: 'Total Sessions',
      value: selectedProject.totalSessions,
      change: `${selectedProject.activeSessions} active`,
      icon: '🔄',
      trend: 'neutral' as const
    },
    {
      title: 'Total Tokens',
      value: formatTokens(selectedProject.totalTokens.total_tokens),
      change: 'All sessions',
      icon: '🔢',
      trend: 'neutral' as const
    },
    {
      title: 'Total Cost',
      value: formatCost(selectedProject.totalCost),
      change: 'Project total',
      icon: '💰',
      trend: 'up' as const
    },
    {
      title: 'Cache Efficiency',
      value: `${cachePercentage}%`,
      change: `${formatTokens(cacheTokens)} cached`,
      icon: '⚡',
      trend: 'up' as const
    },
    {
      title: 'Total Duration',
      value: formatDuration(selectedProject.totalDuration),
      change: 'Combined time',
      icon: '⏱️',
      trend: 'neutral' as const
    },
    {
      title: 'Most Used Model',
      value: formatModel(mostUsedModel[0]),
      change: `${mostUsedModel[1]} sessions`,
      icon: '🤖',
      trend: 'neutral' as const
    }
  ];

  return (
    <div className={cn(
      "flex-1 p-5 overflow-y-auto scrollbar-custom",
      className
    )}>
      {/* Project Header */}
      <div className="flex justify-between items-start mb-5">
        <div>
          <h1 className="text-lg font-semibold text-white mb-1">
            📁 {selectedProject.name}
          </h1>
          <p className="text-sm text-gray-400 mb-1">
            {selectedProject.path}
          </p>
          <div className="flex gap-4 text-xs text-gray-500">
            <span>{selectedProject.branches.length} branches</span>
            <span>{selectedProject.filesModified.length} files modified</span>
            <span>Last activity: {new Date(selectedProject.lastActivity).toLocaleDateString()}</span>
          </div>
        </div>
        
        {onRefresh && (
          <button
            onClick={onRefresh}
            className="btn btn-primary text-xs"
          >
            Refresh
          </button>
        )}
      </div>
      
      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-4 mb-6">
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

      {/* Token Usage Chart */}
      <SimpleBarChart 
        data={chartData}
        title="Project Token Usage"
        timeRange={timeRange}
        timeGranularity={timeGranularity}
        onTimeRangeChange={onTimeRangeChange}
        onTimeGranularityChange={onTimeGranularityChange}
        showControls={true}
        className="mb-6"
      />

      {/* Cost and Message Count Chart */}
      {lineChartData && lineChartData.length > 0 && (
        <SimpleLineChart
          data={lineChartData}
          series={[
            { key: 'cost', label: 'Cost ($)', color: '#10b981', yAxisID: 'y' },
            { key: 'messages', label: 'Messages', color: '#8b5cf6', yAxisID: 'y1' }
          ]}
          title="Cost & Activity Trends"
          showLegend={true}
          formatValue={(value, key) => {
            if (key === 'cost') return `$${value.toFixed(2)}`;
            return value.toLocaleString();
          }}
          className="mb-6"
        />
      )}

      {/* Project Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Project Sessions */}
        <div className="activity-card">
          <div className="flex items-center text-white text-sm font-semibold mb-4">
            <span className="mr-2">🔄</span>
            Project Sessions ({selectedProject.sessions.length})
          </div>
          <div className="max-h-80 overflow-y-auto scrollbar-custom">
            {selectedProject.sessions.map(session => (
              <div
                key={session.id}
                className="session-item mb-2 cursor-pointer"
                onClick={() => onSessionSelect(session.id)}
              >
                <div className="flex justify-between items-center mb-1">
                  <span className="text-sm font-medium text-white">{session.title}</span>
                  <span className={`status-${session.status === 'active' ? 'working' : session.status} text-xs`}>
                    {session.status}
                  </span>
                </div>
                <div className="text-xs text-gray-400">
                  {session.git_branch} • {formatTokens(session.tokens_used.total_tokens)} tokens
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Modified Files */}
        <FilesList
          files={selectedProject.filesModified}
          title="All Modified Files"
          maxItems={10}
        />
      </div>
    </div>
  );
};