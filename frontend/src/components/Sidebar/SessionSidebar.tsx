import React, { useState, useMemo } from 'react';
import { Session, MetricsSummary } from '../../types/session';
import { SessionTabs } from './SessionTabs';
import { SessionList } from './SessionList';
import { StatsSection } from './StatsSection';
import { ProjectsList } from '../Projects/ProjectsList';
import { ProjectStatsSection } from '../Projects/ProjectStatsSection';
import { groupSessionsByProject } from '../../utils/projectHelpers';
import { cn } from '../../utils/classNames';

interface SessionSidebarProps {
  sessions: Session[];
  selectedSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
  selectedProjectId: string | null;
  onProjectSelect: (projectId: string) => void;
  metricsSummary: MetricsSummary;
  onAnalyticsSelect: () => void;
  className?: string;
}

export const SessionSidebar: React.FC<SessionSidebarProps> = ({
  sessions,
  selectedSessionId,
  onSessionSelect,
  selectedProjectId,
  onProjectSelect,
  metricsSummary,
  onAnalyticsSelect,
  className
}) => {
  const [activeTab, setActiveTab] = useState<'sessions' | 'projects' | 'analytics'>('sessions');

  const projects = useMemo(() => groupSessionsByProject(sessions), [sessions]);

  return (
    <div className={cn("flex flex-col h-full", className)}>
      <SessionTabs 
        activeTab={activeTab} 
        onTabChange={(tab) => {
          setActiveTab(tab);
          if (tab === 'analytics') {
            onAnalyticsSelect();
          }
        }} 
      />
      
      {activeTab === 'sessions' ? (
        <>
          <SessionList
            sessions={sessions}
            selectedSessionId={selectedSessionId}
            onSessionSelect={onSessionSelect}
          />
          <StatsSection stats={metricsSummary} />
        </>
      ) : activeTab === 'projects' ? (
        <>
          <ProjectsList
            projects={projects}
            selectedProjectId={selectedProjectId}
            onProjectSelect={onProjectSelect}
          />
          <ProjectStatsSection projects={projects} />
        </>
      ) : (
        <div className="flex-1 flex items-center justify-center p-4 text-center">
          <div>
            <div className="text-4xl mb-2">📊</div>
            <p className="text-sm text-gray-400">Analytics View</p>
            <p className="text-xs text-gray-500 mt-1">View comprehensive insights</p>
          </div>
        </div>
      )}
    </div>
  );
};