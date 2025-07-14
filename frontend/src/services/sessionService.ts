import api from './api';
import { SessionsResponse, SessionsLimitResponse, ActivityResponse, MetricsSummary, UsageStats, TokenTimelineResponse } from '../types/api';
import { Session } from '../types/session';

export const sessionService = {
  // Get all sessions
  getAllSessions: async (): Promise<SessionsResponse> => {
    const response = await api.get('/sessions');
    return response.data;
  },

  // Get active sessions only
  getActiveSessions: async (): Promise<SessionsResponse> => {
    const response = await api.get('/sessions/active');
    return response.data;
  },

  // Get recent sessions with limit
  getRecentSessions: async (limit: number = 10): Promise<SessionsLimitResponse> => {
    const response = await api.get(`/sessions/recent?limit=${limit}`);
    return response.data;
  },

  // Get session by ID
  getSessionById: async (sessionId: string): Promise<Session> => {
    const response = await api.get(`/sessions/${sessionId}`);
    return response.data;
  },

  // Get metrics summary
  getMetricsSummary: async (): Promise<MetricsSummary> => {
    const response = await api.get('/metrics/summary');
    return response.data;
  },

  // Get usage statistics
  getUsageStats: async (): Promise<UsageStats> => {
    const response = await api.get('/metrics/usage');
    return response.data;
  },

  // Get activity timeline
  getActivity: async (limit: number = 50): Promise<ActivityResponse> => {
    const response = await api.get(`/metrics/activity?limit=${limit}`);
    return response.data;
  },

  // Get session-specific activity
  getSessionActivity: async (sessionId: string, limit: number = 50): Promise<ActivityResponse> => {
    const response = await api.get(`/sessions/${sessionId}/activity?limit=${limit}`);
    return response.data;
  },

  // Get project-specific activity
  getProjectActivity: async (projectName: string, limit: number = 50): Promise<ActivityResponse> => {
    const response = await api.get(`/projects/${encodeURIComponent(projectName)}/activity?limit=${limit}`);
    return response.data;
  },

  // Search sessions
  searchSessions: async (query: string): Promise<SessionsResponse> => {
    const response = await api.get(`/search?q=${encodeURIComponent(query)}`);
    return {
      sessions: response.data.results,
      total: response.data.total
    };
  },

  // Get token timeline for all analytics
  getTokenTimeline: async (hours: number = 24, granularity: 'minute' | 'hour' | 'day' = 'hour'): Promise<TokenTimelineResponse> => {
    const response = await api.get('/analytics/tokens/timeline', {
      params: { hours, granularity }
    });
    return response.data;
  },

  // Get token timeline for a specific session
  getSessionTokenTimeline: async (sessionId: string, hours: number = 24, granularity: 'minute' | 'hour' | 'day' = 'hour'): Promise<TokenTimelineResponse> => {
    const response = await api.get(`/sessions/${sessionId}/tokens/timeline`, {
      params: { hours, granularity }
    });
    return response.data;
  },

  // Get token timeline for a specific project
  getProjectTokenTimeline: async (projectName: string, hours: number = 24, granularity: 'minute' | 'hour' | 'day' = 'hour'): Promise<TokenTimelineResponse> => {
    const response = await api.get(`/projects/${encodeURIComponent(projectName)}/tokens/timeline`, {
      params: { hours, granularity }
    });
    return response.data;
  },

  // Create a new UI session
  createSession: async (data: { project_name: string; project_path: string; model?: string }): Promise<Session> => {
    const response = await api.post('/sessions/create', data);
    return response.data;
  },

  // Get sessions (simplified method)
  getSessions: async (): Promise<Session[]> => {
    const response = await api.get('/sessions');
    return response.data.sessions;
  },

  // Get chat messages for a session
  getChatMessages: async (sessionId: string, limit: number = 100, offset: number = 0): Promise<any> => {
    const response = await api.get(`/chat/sessions/${sessionId}/messages`, {
      params: { limit, offset }
    });
    return response.data;
  }
};