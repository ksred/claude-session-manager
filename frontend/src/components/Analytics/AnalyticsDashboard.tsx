import React from 'react';
import { ModelPerformance } from './ModelPerformance';
import { TimeSeriesChart } from './TimeSeriesChart';
import { CostAnalytics } from './CostAnalytics';
import { cn } from '../../utils/classNames';

interface AnalyticsDashboardProps {
  className?: string;
}

export const AnalyticsDashboard: React.FC<AnalyticsDashboardProps> = ({ className }) => {
  return (
    <div className={cn("p-6 space-y-6", className)}>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-white">Analytics</h2>
        <p className="text-sm text-gray-400">
          Comprehensive insights into your Claude usage
        </p>
      </div>

      {/* Model Performance Section */}
      <ModelPerformance />

      {/* Time Series Section */}
      <TimeSeriesChart />

      {/* Cost Analytics Section */}
      <CostAnalytics />
    </div>
  );
};