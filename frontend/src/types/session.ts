// Core session types (updated to match API)
export interface Session {
  id: string;
  project_path: string;
  project_name: string;
  git_branch: string;
  git_worktree: string;
  status: SessionStatus;
  title: string;
  current_task: string;
  created_at: string;
  updated_at: string;
  duration_seconds: number;
  is_active: boolean;
  message_count: number;
  model: string;
  tokens_used: TokenUsage;
  files_modified: string[];
  source?: 'import' | 'ui'; // Source of the session
  
  // Computed fields for UI
  progress?: number;
}

export interface TokenUsage {
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  estimated_cost: number;
  cache_creation_input_tokens: number;
  cache_read_input_tokens: number;
}

export enum SessionStatus {
  ACTIVE = 'active',
  IDLE = 'idle',
  COMPLETED = 'completed',
  ERROR = 'error'
}

export interface ActivityEntry {
  id?: string; // Generated client-side if not provided by API
  session_id: string;
  session_name: string;
  details: string;
  timestamp: string;
  type: 'message_sent' | 'session_created' | 'session_updated' | 'error';
}

// Comprehensive metrics from API
export interface MetricsSummary {
  active_sessions: number;
  total_sessions: number;
  total_messages: number;
  total_tokens_used: number;
  total_estimated_cost: number;
  average_session_duration_minutes: number;
  most_used_model: string;
  model_usage: Record<string, number>;
}

export interface DailySessionCount {
  date: string;
  count: number;
}

export interface PeakHourInfo {
  hour: number;
  average_sessions: number;
}

export interface UsageStats {
  daily_sessions: DailySessionCount[];
  model_usage: Record<string, number>;
  peak_hours: PeakHourInfo[];
}

export interface ChartDataPoint {
  timestamp: Date;
  value: number;
  label: string;
}