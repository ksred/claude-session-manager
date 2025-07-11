import React, { useState } from 'react';
import { Session, ActivityEntry, ChartDataPoint } from '../../types/session';
import { SessionHeader } from './SessionHeader';
import { MetricsGrid } from './MetricsGrid';
import { SimpleBarChart } from '../Charts/SimpleBarChart';
import { SimpleLineChart, LineChartDataPoint } from '../Charts/SimpleLineChart';
import { ActivityFeed } from '../ActivityFeed/ActivityFeed';
import { FilesList } from '../Common/FilesList';
import { TerminalChat } from '../Terminal/TerminalChat';
import { cn } from '../../utils/classNames';
import '../Terminal/TerminalChat.css';

interface SessionDetailViewProps {
  selectedSession: Session | null;
  recentActivity: ActivityEntry[];
  chartData: ChartDataPoint[];
  lineChartData?: LineChartDataPoint[];
  isChartLoading?: boolean;
  timeRange: number;
  timeGranularity: 'minute' | 'hour' | 'day';
  onTimeRangeChange: (hours: number) => void;
  onTimeGranularityChange: (granularity: 'minute' | 'hour' | 'day') => void;
  onRefresh?: () => void;
  showTerminal?: boolean;
  className?: string;
}

export const SessionDetailView: React.FC<SessionDetailViewProps> = ({
  selectedSession,
  recentActivity,
  chartData,
  lineChartData,
  isChartLoading = false,
  timeRange,
  timeGranularity,
  onTimeRangeChange,
  onTimeGranularityChange,
  onRefresh,
  showTerminal = false,
  className
}) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'terminal'>('overview');

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Tab Navigation */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex space-x-8 px-5 pt-2" aria-label="Tabs">
          <button
            onClick={() => setActiveTab('overview')}
            className={cn(
              "py-2 px-1 border-b-2 font-medium text-sm transition-colors",
              activeTab === 'overview'
                ? "border-blue-500 text-blue-600 dark:text-blue-400"
                : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300"
            )}
          >
            Overview
          </button>
          {showTerminal && (
            <button
              onClick={() => setActiveTab('terminal')}
              className={cn(
                "py-2 px-1 border-b-2 font-medium text-sm transition-colors",
                activeTab === 'terminal'
                  ? "border-blue-500 text-blue-600 dark:text-blue-400"
                  : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300"
              )}
            >
              Terminal Chat
            </button>
          )}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'overview' ? (
          <div className="h-full p-5 overflow-y-auto scrollbar-custom">
            <SessionHeader 
              session={selectedSession} 
              onRefresh={onRefresh}
            />
            
            <MetricsGrid session={selectedSession} />
            
            <SimpleBarChart 
              data={chartData}
              title="Token Usage"
              isLoading={isChartLoading}
              timeRange={timeRange}
              timeGranularity={timeGranularity}
              onTimeRangeChange={onTimeRangeChange}
              onTimeGranularityChange={onTimeGranularityChange}
              showControls={true}
              className="mb-4"
            />
            
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
                className="mb-4"
              />
            )}
            
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <ActivityFeed activities={recentActivity} />
              
              {selectedSession && (
                <FilesList
                  files={selectedSession.files_modified}
                  title="Modified Files"
                />
              )}
            </div>
          </div>
        ) : (
          <div className="h-full">
            {selectedSession && (
              <TerminalChat
                sessionId={selectedSession.id}
                className="h-full"
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
};