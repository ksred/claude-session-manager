import React from 'react';
import { Session, ActivityEntry, ChartDataPoint } from '../../types/session';
import { SessionHeader } from './SessionHeader';
import { MetricsGrid } from './MetricsGrid';
import { SimpleBarChart } from '../Charts/SimpleBarChart';
import { SimpleLineChart, LineChartDataPoint } from '../Charts/SimpleLineChart';
import { ActivityFeed } from '../ActivityFeed/ActivityFeed';
import { FilesList } from '../Common/FilesList';
import { cn } from '../../utils/classNames';

interface MainDashboardProps {
  selectedSession: Session | null;
  recentActivity: ActivityEntry[];
  chartData: ChartDataPoint[];
  lineChartData?: LineChartDataPoint[];
  isChartLoading?: boolean;
  timeRange: number;
  timeGranularity: 'hour' | 'day';
  onTimeRangeChange: (hours: number) => void;
  onTimeGranularityChange: (granularity: 'hour' | 'day') => void;
  onRefresh?: () => void;
  className?: string;
}

export const MainDashboard: React.FC<MainDashboardProps> = ({
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
  className
}) => {
  return (
    <div className={cn(
      "flex-1 p-5 overflow-y-auto scrollbar-custom",
      className
    )}>
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
  );
};