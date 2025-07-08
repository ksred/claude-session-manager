import React, { useState, useMemo } from 'react';
import { AppLayout } from './components/Layout/AppLayout';
import { SessionSidebar } from './components/Sidebar/SessionSidebar';
import { MainDashboard } from './components/Dashboard/MainDashboard';
import { ProjectDashboard } from './components/Dashboard/ProjectDashboard';
import { AnalyticsDashboard } from './components/Analytics/AnalyticsDashboard';
import { LoadingState } from './components/Common/LoadingState';
import { ErrorMessage } from './components/Common/ErrorMessage';
import { 
  useAllSessions, 
  useMetricsSummary, 
  useActivity, 
  useUsageStats,
  useRefreshData 
} from './hooks/useSessionData';
import { useWebSocket } from './hooks/useWebSocket';
import { groupSessionsByProject } from './utils/projectHelpers';
import { mockChartData } from './utils/mockData'; // Keep chart data as mock for now

export const AppContent: React.FC = () => {
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const [currentView, setCurrentView] = useState<'session' | 'project' | 'analytics'>('session');

  // API data hooks
  const { data: sessionsData, isLoading: sessionsLoading, error: sessionsError } = useAllSessions();
  const { data: metricsData, isLoading: metricsLoading, error: metricsError } = useMetricsSummary();
  const { data: activityData } = useActivity();
  const { data: _usageData } = useUsageStats();
  
  // WebSocket for real-time updates
  const { connectionStatus } = useWebSocket();
  
  // Refresh function
  const refreshData = useRefreshData();

  // Extract data with fallbacks
  const sessions = sessionsData?.sessions || [];
  const metricsSummary = metricsData || {
    active_sessions: 0,
    total_sessions: 0,
    total_messages: 0,
    total_tokens_used: 0,
    total_estimated_cost: 0,
    average_session_duration_minutes: 0,
    most_used_model: 'unknown',
    model_usage: {}
  };
  const recentActivity = activityData?.activity || [];

  // Set default selected session if none selected and sessions are available
  React.useEffect(() => {
    if (!selectedSessionId && sessions.length > 0) {
      setSelectedSessionId(sessions[0].id);
    }
  }, [sessions, selectedSessionId]);

  // Memoized computed data
  const selectedSession = selectedSessionId 
    ? sessions.find(s => s.id === selectedSessionId) || null
    : null;

  const projects = useMemo(() => groupSessionsByProject(sessions), [sessions]);
  const selectedProject = selectedProjectId 
    ? projects.find(p => p.id === selectedProjectId) || null
    : null;

  // Event handlers
  const handleSessionSelect = (sessionId: string) => {
    setSelectedSessionId(sessionId);
    setCurrentView('session');
  };

  const handleProjectSelect = (projectId: string) => {
    setSelectedProjectId(projectId);
    setCurrentView('project');
  };

  const handleAnalyticsSelect = () => {
    setCurrentView('analytics');
  };

  const handleRefresh = () => {
    console.log('🔄 Refreshing all data...');
    refreshData();
  };

  // Loading state
  if (sessionsLoading || metricsLoading) {
    return <LoadingState message="Loading Claude sessions..." />;
  }

  // Error state
  if (sessionsError || metricsError) {
    const errorMessage = sessionsError?.message || metricsError?.message || 'An error occurred';
    return (
      <ErrorMessage
        title="Failed to Load Data"
        message={errorMessage}
        onRetry={handleRefresh}
      />
    );
  }

  return (
    <AppLayout
      apiStatus={sessionsError || metricsError ? 'error' : 'connected'}
      wsStatus={connectionStatus}
      sidebar={
        <SessionSidebar
          sessions={sessions}
          selectedSessionId={selectedSessionId}
          onSessionSelect={handleSessionSelect}
          selectedProjectId={selectedProjectId}
          onProjectSelect={handleProjectSelect}
          metricsSummary={metricsSummary}
          onAnalyticsSelect={handleAnalyticsSelect}
        />
      }
      main={
        currentView === 'session' ? (
          <MainDashboard
            selectedSession={selectedSession}
            recentActivity={recentActivity}
            chartData={mockChartData} // TODO: Replace with real chart data from usage stats
            onRefresh={handleRefresh}
          />
        ) : currentView === 'project' ? (
          <ProjectDashboard
            selectedProject={selectedProject}
            onSessionSelect={handleSessionSelect}
            onRefresh={handleRefresh}
          />
        ) : (
          <AnalyticsDashboard />
        )
      }
    />
  );
};