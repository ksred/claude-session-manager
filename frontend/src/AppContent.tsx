import React, { useState, useMemo } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { AppLayout } from './components/Layout/AppLayout';
import { SessionSidebar } from './components/Sidebar/SessionSidebar';
import { SessionDetailView } from './components/Dashboard/SessionDetailView';
import { ProjectDashboard } from './components/Dashboard/ProjectDashboard';
import { AnalyticsDashboard } from './components/Analytics/AnalyticsDashboard';
import { LoadingState } from './components/Common/LoadingState';
import { ErrorMessage } from './components/Common/ErrorMessage';
import { 
  useAllSessions, 
  useMetricsSummary, 
  useActivity,
  useSessionActivity,
  useProjectActivity, 
  useUsageStats,
  useTokenTimeline,
  useSessionTokenTimeline,
  useProjectTokenTimeline,
  useRefreshData,
  sessionKeys
} from './hooks/useSessionData';
import { useWebSocket } from './hooks/useWebSocket';
import { groupSessionsByProject } from './utils/projectHelpers';
import { transformTokenTimelineToChartData, transformTokenTimelineToLineChartData } from './utils/formatters';

// Feature flags
// To enable terminal chat: Set TERMINAL_CHAT to true
// Terminal chat will only show for UI-initiated sessions (not imported ones)
const FEATURES = {
  TERMINAL_CHAT: false // Set to true to enable terminal chat feature
};

export const AppContent: React.FC = () => {
  const queryClient = useQueryClient();
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const [currentView, setCurrentView] = useState<'session' | 'project' | 'analytics'>('session');
  const [timeRange, setTimeRange] = useState(168);
  const [timeGranularity, setTimeGranularity] = useState<'minute' | 'hour' | 'day'>('minute');

  // API data hooks
  const { data: sessionsData, isLoading: sessionsLoading, error: sessionsError } = useAllSessions();
  const { data: metricsData, isLoading: metricsLoading, error: metricsError } = useMetricsSummary();
  const { data: activityData } = useActivity();
  const { data: sessionActivityData } = useSessionActivity(selectedSessionId || undefined);
  const { data: projectActivityData } = useProjectActivity(selectedProjectId || undefined);
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
  
  // Use appropriate activity based on view and selection
  const recentActivity = useMemo(() => {
    // When a session is selected (regardless of view), show session-specific activity
    if (selectedSessionId && sessionActivityData) {
      return sessionActivityData.activity || [];
    } 
    // When in project view with a selected project, use project-specific activity
    else if (currentView === 'project' && selectedProjectId && projectActivityData) {
      return projectActivityData.activity || [];
    }
    // Default to overall activity
    return activityData?.activity || [];
  }, [selectedSessionId, currentView, selectedProjectId, activityData, sessionActivityData, projectActivityData]);

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

  // Show terminal only for UI-initiated sessions AND when feature is enabled
  const showTerminal = FEATURES.TERMINAL_CHAT && selectedSession?.source === 'ui';

  const projects = useMemo(() => groupSessionsByProject(sessions), [sessions]);
  const selectedProject = selectedProjectId 
    ? projects.find(p => p.id === selectedProjectId) || null
    : null;

  // Token timeline hooks based on current view
  const { data: generalTokenData, isLoading: generalTokenLoading } = useTokenTimeline(timeRange, timeGranularity);
  const { data: sessionTokenData, isLoading: sessionTokenLoading } = useSessionTokenTimeline(
    currentView === 'session' ? selectedSessionId : null,
    timeRange,
    timeGranularity
  );
  const { data: projectTokenData, isLoading: projectTokenLoading } = useProjectTokenTimeline(
    currentView === 'project' && selectedProject ? selectedProject.name : null,
    timeRange,
    timeGranularity
  );

  // Compute chart data based on current view and available data
  const chartData = useMemo(() => {
    let timelineData = null;
    
    if (currentView === 'session' && sessionTokenData?.timeline) {
      timelineData = sessionTokenData.timeline;
    } else if (currentView === 'project' && projectTokenData?.timeline) {
      timelineData = projectTokenData.timeline;
    } else if (generalTokenData?.timeline) {
      timelineData = generalTokenData.timeline;
    }
    
    // Transform the timeline data to chart format, or return empty array
    return timelineData ? transformTokenTimelineToChartData(timelineData) : [];
  }, [currentView, sessionTokenData, projectTokenData, generalTokenData]);

  // Compute line chart data based on current view and available data
  const lineChartData = useMemo(() => {
    let timelineData = null;
    
    if (currentView === 'session' && sessionTokenData?.timeline) {
      timelineData = sessionTokenData.timeline;
    } else if (currentView === 'project' && projectTokenData?.timeline) {
      timelineData = projectTokenData.timeline;
    } else if (generalTokenData?.timeline) {
      timelineData = generalTokenData.timeline;
    }
    
    // Transform the timeline data to line chart format, or return empty array
    return timelineData ? transformTokenTimelineToLineChartData(timelineData) : [];
  }, [currentView, sessionTokenData, projectTokenData, generalTokenData]);
  
  // Determine if chart data is loading based on current view
  const isChartLoading = useMemo(() => {
    if (currentView === 'session') {
      return sessionTokenLoading;
    } else if (currentView === 'project') {
      return projectTokenLoading;
    } else {
      return generalTokenLoading;
    }
  }, [currentView, sessionTokenLoading, projectTokenLoading, generalTokenLoading]);

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
          onSessionCreated={() => {
            // Invalidate sessions query to refresh the list
            queryClient.invalidateQueries({ queryKey: sessionKeys.list({}) });
          }}
        />
      }
      main={
        currentView === 'session' ? (
          <SessionDetailView
            selectedSession={selectedSession}
            recentActivity={recentActivity}
            chartData={chartData}
            lineChartData={lineChartData}
            isChartLoading={isChartLoading}
            timeRange={timeRange}
            timeGranularity={timeGranularity}
            onTimeRangeChange={setTimeRange}
            onTimeGranularityChange={setTimeGranularity}
            onRefresh={handleRefresh}
            showTerminal={showTerminal}
          />
        ) : currentView === 'project' ? (
          <ProjectDashboard
            selectedProject={selectedProject}
            recentActivity={recentActivity}
            chartData={chartData}
            lineChartData={lineChartData}
            isChartLoading={isChartLoading}
            timeRange={timeRange}
            timeGranularity={timeGranularity}
            onTimeRangeChange={setTimeRange}
            onTimeGranularityChange={setTimeGranularity}
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