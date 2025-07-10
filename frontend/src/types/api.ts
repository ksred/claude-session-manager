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

// Token timeline types
export interface TokenTimelinePoint {
  timestamp: string;
  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  total_tokens: number;
  estimated_cost: number;
  message_count: number;
  label?: string;
}

export interface TokenTimelineResponse {
  timeline: TokenTimelinePoint[];
  total: number;
  hours: number;
  granularity: string;
}

// API error types
export interface ErrorResponse {
  error: string;
  code: number;
  details: string;
}