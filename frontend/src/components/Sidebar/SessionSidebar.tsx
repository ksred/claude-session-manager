import React, { useState, useMemo } from 'react';
import { Session, MetricsSummary } from '../../types/session';
import { SessionTabs } from './SessionTabs';
import { SessionList } from './SessionList';
import { StatsSection } from './StatsSection';
import { ProjectsList } from '../Projects/ProjectsList';
import { ProjectStatsSection } from '../Projects/ProjectStatsSection';
import { groupSessionsByProject } from '../../utils/projectHelpers';
import { cn } from '../../utils/classNames';
import { PlusIcon } from '@heroicons/react/24/outline';
import { NewSessionModal } from '../NewSessionModal';

interface SessionSidebarProps {
  sessions: Session[];
  selectedSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
  selectedProjectId: string | null;
  onProjectSelect: (projectId: string) => void;
  metricsSummary: MetricsSummary;
  onAnalyticsSelect: () => void;
  onSessionCreated?: (session: Session) => void;
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
  onSessionCreated,
  className
}) => {
  const [activeTab, setActiveTab] = useState<'sessions' | 'projects' | 'analytics'>('sessions');
  const [showNewSessionModal, setShowNewSessionModal] = useState(false);

  const projects = useMemo(() => groupSessionsByProject(sessions), [sessions]);

  const handleSessionCreated = (session: Session) => {
    setShowNewSessionModal(false);
    if (onSessionCreated) {
      onSessionCreated(session);
    }
    // Select the newly created session
    onSessionSelect(session.id);
  };

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
          <div className="p-4 border-b border-gray-200 dark:border-gray-700">
            <button
              onClick={() => setShowNewSessionModal(true)}
              className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md transition-colors"
            >
              <PlusIcon className="w-5 h-5" />
              <span>New Session</span>
            </button>
          </div>
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
      
      <NewSessionModal
        isOpen={showNewSessionModal}
        onClose={() => setShowNewSessionModal(false)}
        onSessionCreated={handleSessionCreated}
      />
    </div>
  );
};