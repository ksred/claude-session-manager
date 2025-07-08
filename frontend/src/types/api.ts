import { Session, ActivityEntry } from './session';
export type { MetricsSummary, UsageStats } from './session';

// API response types (matching Swagger spec)
export interface SessionsResponse {
  sessions: Session[];
  total: number;
}

export interface SessionsLimitResponse {
  sessions: Session[];
  limit: number;
}

export interface ActivityResponse {
  activity: ActivityEntry[];
  total: number;
}

export interface SearchResponse {
  query: string;
  results: Session[];
  total: number;
}

export interface HealthResponse {
  status: string;
  timestamp: string;
  uptime: string;
  version: string;
}

export interface WebSocketMessage {
  type: string;
  session_id?: string;
  data: any;
  timestamp: string;
}

// API error types
export interface ErrorResponse {
  error: string;
  code: number;
  details: string;
}