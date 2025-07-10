import { Session, SessionStatus, MetricsSummary, ActivityEntry, ChartDataPoint, UsageStats } from '../types/session';

// Mock session data (updated to match API schema)
export const mockSessions: Session[] = [
  {
    id: 'session_123456',
    project_path: '/Users/dev/myproject',
    project_name: 'api-refactor',
    git_branch: 'feature/auth',
    git_worktree: 'main',
    status: SessionStatus.ACTIVE,
    title: 'Implement user authentication',
    current_task: 'Refactoring JWT authentication middleware...',
    created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    updated_at: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
    duration_seconds: 7200, // 2 hours
    is_active: true,
    message_count: 25,
    model: 'claude-3-5-sonnet-20241022',
    tokens_used: {
      input_tokens: 25100,
      output_tokens: 17000,
      total_tokens: 42100,
      estimated_cost: 0.84,
      cache_creation_input_tokens: 1200,
      cache_read_input_tokens: 850
    },
    files_modified: [
      '/src/auth.go',
      '/src/user.go',
      '/tests/auth_test.go'
    ],
    progress: 65
  },
  {
    id: 'session_789012',
    project_path: '/Users/dev/myproject-db',
    project_name: 'db-migration',
    git_branch: 'feature/migration',
    git_worktree: 'migration',
    status: SessionStatus.IDLE,
    title: 'Database schema migration',
    current_task: 'Awaiting user input for migration strategy',
    created_at: new Date(Date.now() - 45 * 60 * 1000).toISOString(),
    updated_at: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
    duration_seconds: 2700, // 45 minutes
    is_active: false,
    message_count: 18,
    model: 'claude-3-opus-20240229',
    tokens_used: {
      input_tokens: 8500,
      output_tokens: 12300,
      total_tokens: 20800,
      estimated_cost: 0.42,
      cache_creation_input_tokens: 500,
      cache_read_input_tokens: 300
    },
    files_modified: [
      '/migrations/001_user_table.sql',
      '/src/models/user.ts'
    ],
    progress: 30
  },
  {
    id: 'session_345678',
    project_path: '/Users/dev/dashboard',
    project_name: 'frontend-bug',
    git_branch: 'bugfix/rendering',
    git_worktree: 'v2',
    status: SessionStatus.COMPLETED,
    title: 'Fix React component rendering',
    current_task: 'Fixed React component rendering issue',
    created_at: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
    updated_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    duration_seconds: 7200, // 2 hours
    is_active: false,
    message_count: 15,
    model: 'claude-3-haiku-20240307',
    tokens_used: {
      input_tokens: 15200,
      output_tokens: 9800,
      total_tokens: 25000,
      estimated_cost: 0.50,
      cache_creation_input_tokens: 200,
      cache_read_input_tokens: 150
    },
    files_modified: [
      '/src/components/Dashboard.tsx',
      '/src/hooks/useData.ts'
    ],
    progress: 100
  },
  {
    id: 'session_901234',
    project_path: '/Users/dev/mobile-app',
    project_name: 'new-feature',
    git_branch: 'feature/new-ui',
    git_worktree: 'feat-x',
    status: SessionStatus.ERROR,
    title: 'Implement new UI feature',
    current_task: 'Build failed: missing dependency',
    created_at: new Date(Date.now() - 20 * 60 * 1000).toISOString(),
    updated_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    duration_seconds: 900, // 15 minutes
    is_active: false,
    message_count: 8,
    model: 'claude-3-5-sonnet-20241022',
    tokens_used: {
      input_tokens: 5200,
      output_tokens: 3800,
      total_tokens: 9000,
      estimated_cost: 0.18,
      cache_creation_input_tokens: 100,
      cache_read_input_tokens: 50
    },
    files_modified: [
      '/package.json',
      '/src/components/NewFeature.tsx'
    ],
    progress: 45
  }
];

export const mockMetricsSummary: MetricsSummary = {
  active_sessions: 4,
  total_sessions: 150,
  total_messages: 2500,
  total_tokens_used: 125000,
  total_estimated_cost: 15.75,
  average_session_duration_minutes: 45.2,
  most_used_model: 'claude-3-5-sonnet-20241022',
  model_usage: {
    'claude-3-5-sonnet-20241022': 85,
    'claude-3-opus-20240229': 45,
    'claude-3-haiku-20240307': 20
  }
};

export const mockRecentActivity: ActivityEntry[] = [
  {
    session_id: 'session_123456',
    session_name: 'api-refactor',
    details: 'User sent a message',
    timestamp: new Date(Date.now() - 30 * 1000).toISOString(),
    type: 'message_sent'
  },
  {
    session_id: 'session_123456',
    session_name: 'api-refactor',
    details: 'Session updated with new task progress',
    timestamp: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
    type: 'session_updated'
  },
  {
    session_id: 'session_345678',
    session_name: 'frontend-bug',
    details: 'Session completed successfully',
    timestamp: new Date(Date.now() - 4 * 60 * 1000).toISOString(),
    type: 'session_updated'
  },
  {
    session_id: 'session_901234',
    session_name: 'new-feature',
    details: 'Build failed: missing dependency',
    timestamp: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    type: 'error'
  },
  {
    session_id: 'session_789012',
    session_name: 'db-migration',
    details: 'New session created',
    timestamp: new Date(Date.now() - 45 * 60 * 1000).toISOString(),
    type: 'session_created'
  }
];

export const mockUsageStats: UsageStats = {
  daily_sessions: [
    { date: '2023-01-01', count: 15 },
    { date: '2023-01-02', count: 12 },
    { date: '2023-01-03', count: 18 },
    { date: '2023-01-04', count: 20 },
    { date: '2023-01-05', count: 16 },
    { date: '2023-01-06', count: 22 },
    { date: '2023-01-07', count: 19 }
  ],
  model_usage: {
    'claude-3-5-sonnet-20241022': 85,
    'claude-3-opus-20240229': 45,
    'claude-3-haiku-20240307': 20
  },
  peak_hours: [
    { hour: 14, average_sessions: 5.2 },
    { hour: 15, average_sessions: 4.8 },
    { hour: 16, average_sessions: 4.5 }
  ]
};

export const mockChartData: ChartDataPoint[] = Array.from({ length: 24 }, (_, i) => {
  const hour = i;
  // Create more realistic usage patterns - higher during work hours, lower at night
  let baseValue;
  if (hour >= 9 && hour <= 17) {
    // Work hours - higher usage
    baseValue = 8000 + Math.random() * 10000;
  } else if (hour >= 6 && hour <= 9 || hour >= 18 && hour <= 22) {
    // Morning/evening - medium usage
    baseValue = 3000 + Math.random() * 5000;
  } else {
    // Night hours - lower usage
    baseValue = 500 + Math.random() * 2000;
  }
  
  return {
    timestamp: new Date(Date.now() - (23 - i) * 60 * 60 * 1000),
    value: Math.floor(baseValue),
    label: `${hour.toString().padStart(2, '0')}:00`
  };
});