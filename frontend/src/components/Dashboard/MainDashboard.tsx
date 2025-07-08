import React from 'react';
import { Session, ActivityEntry, ChartDataPoint } from '../../types/session';
import { SessionHeader } from './SessionHeader';
import { MetricsGrid } from './MetricsGrid';
import { SimpleBarChart } from '../Charts/SimpleBarChart';
import { ActivityFeed } from '../ActivityFeed/ActivityFeed';
import { FilesList } from '../Common/FilesList';
import { cn } from '../../utils/classNames';

interface MainDashboardProps {
  selectedSession: Session | null;
  recentActivity: ActivityEntry[];
  chartData: ChartDataPoint[];
  onRefresh?: () => void;
  className?: string;
}

export const MainDashboard: React.FC<MainDashboardProps> = ({
  selectedSession,
  recentActivity,
  chartData,
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
        className="mb-4"
      />
      
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