import { useQuery, useQueryClient } from '@tanstack/react-query';
import { sessionService } from '../services/sessionService';

// Query keys for cache management
export const sessionKeys = {
  all: ['sessions'] as const,
  lists: () => [...sessionKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...sessionKeys.lists(), { filters }] as const,
  details: () => [...sessionKeys.all, 'detail'] as const,
  detail: (id: string) => [...sessionKeys.details(), id] as const,
  active: () => [...sessionKeys.lists(), { active: true }] as const,
  recent: () => [...sessionKeys.lists(), { recent: true }] as const,
  metrics: () => ['metrics'] as const,
  activity: () => ['activity'] as const,
  usage: () => ['usage'] as const,
  tokenTimeline: () => ['tokenTimeline'] as const,
  sessionTokenTimeline: (id: string) => ['tokenTimeline', 'session', id] as const,
  projectTokenTimeline: (name: string) => ['tokenTimeline', 'project', name] as const,
};

// Get all sessions
export const useAllSessions = () => {
  const queryClient = useQueryClient();
  
  return useQuery({
    queryKey: sessionKeys.list({}),
    queryFn: sessionService.getAllSessions,
    staleTime: 30000, // 30 seconds
    refetchInterval: 60000, // 1 minute
    retry: 3,
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
    // Preserve existing order when updating to prevent flashing
    select: (data) => {
      const previousData = queryClient.getQueryData(sessionKeys.list({})) as typeof data;
      if (!previousData || !data) return data;
      
      // Create a map of session IDs to their current index
      const orderMap = new Map<string, number>();
      previousData.sessions?.forEach((session, index) => {
        orderMap.set(session.id, index);
      });
      
      // Sort new sessions to maintain previous order where possible
      const sortedSessions = [...(data.sessions || [])].sort((a, b) => {
        const aIndex = orderMap.get(a.id) ?? Infinity;
        const bIndex = orderMap.get(b.id) ?? Infinity;
        
        // If both sessions existed before, maintain their order
        if (aIndex !== Infinity && bIndex !== Infinity) {
          return aIndex - bIndex;
        }
        
        // If one is new, sort by last_activity
        return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
      });
      
      return {
        ...data,
        sessions: sortedSessions
      };
    }
  });
};

// Get active sessions only
export const useActiveSessions = () => {
  return useQuery({
    queryKey: sessionKeys.list({ active: true }),
    queryFn: sessionService.getActiveSessions,
    staleTime: 10000, // 10 seconds (more frequent for active sessions)
    refetchInterval: 30000, // 30 seconds
    retry: 3,
  });
};

// Get recent sessions
export const useRecentSessions = (limit: number = 10) => {
  return useQuery({
    queryKey: sessionKeys.list({ recent: true, limit }),
    queryFn: () => sessionService.getRecentSessions(limit),
    staleTime: 30000,
    refetchInterval: 60000,
    retry: 3,
  });
};

// Get session by ID
export const useSession = (sessionId: string | null) => {
  return useQuery({
    queryKey: sessionKeys.detail(sessionId || ''),
    queryFn: () => sessionService.getSessionById(sessionId!),
    enabled: !!sessionId,
    staleTime: 30000,
    retry: 3,
  });
};

// Get metrics summary
export const useMetricsSummary = () => {
  return useQuery({
    queryKey: sessionKeys.metrics(),
    queryFn: sessionService.getMetricsSummary,
    staleTime: 60000, // 1 minute
    refetchInterval: 120000, // 2 minutes
    retry: 3,
  });
};

// Get usage statistics
export const useUsageStats = () => {
  return useQuery({
    queryKey: sessionKeys.usage(),
    queryFn: sessionService.getUsageStats,
    staleTime: 300000, // 5 minutes
    refetchInterval: 600000, // 10 minutes
    retry: 3,
  });
};

// Get activity timeline
export const useActivity = (limit: number = 50) => {
  return useQuery({
    queryKey: sessionKeys.activity(),
    queryFn: () => sessionService.getActivity(limit),
    staleTime: 15000, // 15 seconds
    refetchInterval: 30000, // 30 seconds
    retry: 3,
  });
};

// Get session-specific activity
export const useSessionActivity = (sessionId: string | undefined, limit: number = 50) => {
  return useQuery({
    queryKey: ['activity', 'session', sessionId, limit],
    queryFn: () => sessionService.getSessionActivity(sessionId!, limit),
    enabled: !!sessionId,
    staleTime: 30000, // 30 seconds
    refetchInterval: 60000, // 1 minute
  });
};

// Get project-specific activity
export const useProjectActivity = (projectName: string | undefined, limit: number = 50) => {
  return useQuery({
    queryKey: ['activity', 'project', projectName, limit],
    queryFn: () => sessionService.getProjectActivity(projectName!, limit),
    enabled: !!projectName,
    staleTime: 30000, // 30 seconds
    refetchInterval: 60000, // 1 minute
  });
};

// Search sessions
export const useSearchSessions = (query: string) => {
  return useQuery({
    queryKey: sessionKeys.list({ search: query }),
    queryFn: () => sessionService.searchSessions(query),
    enabled: query.length > 0,
    staleTime: 60000,
    retry: 2,
  });
};

// Get token timeline for all analytics
export const useTokenTimeline = (hours: number = 24, granularity: 'hour' | 'day' = 'hour') => {
  return useQuery({
    queryKey: [...sessionKeys.tokenTimeline(), { hours, granularity }],
    queryFn: () => sessionService.getTokenTimeline(hours, granularity),
    staleTime: 60000, // 1 minute
    refetchInterval: 120000, // 2 minutes
    retry: 3,
  });
};

// Get token timeline for a specific session
export const useSessionTokenTimeline = (sessionId: string | null, hours: number = 24, granularity: 'hour' | 'day' = 'hour') => {
  return useQuery({
    queryKey: [...sessionKeys.sessionTokenTimeline(sessionId || ''), { hours, granularity }],
    queryFn: () => sessionService.getSessionTokenTimeline(sessionId!, hours, granularity),
    enabled: !!sessionId,
    staleTime: 60000,
    refetchInterval: 120000,
    retry: 3,
  });
};

// Get token timeline for a specific project
export const useProjectTokenTimeline = (projectName: string | null, hours: number = 24, granularity: 'hour' | 'day' = 'hour') => {
  return useQuery({
    queryKey: [...sessionKeys.projectTokenTimeline(projectName || ''), { hours, granularity }],
    queryFn: () => sessionService.getProjectTokenTimeline(projectName!, hours, granularity),
    enabled: !!projectName,
    staleTime: 60000,
    refetchInterval: 120000,
    retry: 3,
  });
};

// Hook for manual refresh of all data
export const useRefreshData = () => {
  const queryClient = useQueryClient();
  
  return () => {
    queryClient.invalidateQueries({ queryKey: sessionKeys.all });
    queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() });
    queryClient.invalidateQueries({ queryKey: sessionKeys.activity() });
    queryClient.invalidateQueries({ queryKey: sessionKeys.usage() });
    queryClient.invalidateQueries({ queryKey: sessionKeys.tokenTimeline() });
  };
};