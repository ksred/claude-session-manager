import React, { useState } from 'react';
import { ModelPerformance } from './ModelPerformance';
import { TimeSeriesChart } from './TimeSeriesChart';
import { CostAnalytics } from './CostAnalytics';
import { SimpleBarChart } from '../Charts/SimpleBarChart';
import { SimpleLineChart } from '../Charts/SimpleLineChart';
import { useTokenTimeline } from '../../hooks/useSessionData';
import { transformTokenTimelineToChartData, transformTokenTimelineToLineChartData } from '../../utils/formatters';
import { cn } from '../../utils/classNames';

interface AnalyticsDashboardProps {
  className?: string;
}

export const AnalyticsDashboard: React.FC<AnalyticsDashboardProps> = ({ className }) => {
  const [hours, setHours] = useState(168);
  const [granularity, setGranularity] = useState<'hour' | 'day'>('hour');
  
  const { data: tokenData } = useTokenTimeline(hours, granularity);
  const chartData = tokenData?.timeline ? transformTokenTimelineToChartData(tokenData.timeline) : [];
  const lineChartData = tokenData?.timeline ? transformTokenTimelineToLineChartData(tokenData.timeline) : [];

  return (
    <div className={cn("flex-1 p-6 space-y-6 overflow-y-auto scrollbar-custom", className)}>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white">Analytics</h2>
        <p className="text-sm text-gray-400">
          Comprehensive insights into your Claude usage
        </p>
      </div>

      {/* Token Usage Chart */}
      <SimpleBarChart 
        data={chartData} 
        title="Token Usage Overview"
        timeRange={hours}
        timeGranularity={granularity}
        onTimeRangeChange={setHours}
        onTimeGranularityChange={setGranularity}
        showControls={true}
      />

      {/* Cost and Message Count Chart */}
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
      />

      {/* Model Performance Section */}
      <ModelPerformance />

      {/* Time Series Section */}
      <TimeSeriesChart />

      {/* Cost Analytics Section */}
      <CostAnalytics />
    </div>
  );
};