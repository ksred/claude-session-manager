# Claude Session Manager - Frontend Implementation Document

## Implementation Status

### Overall Progress: 🟢 **Phase 1 Complete** (25% Complete)

#### Phase 1: Core Infrastructure (Week 1) - ✅ **COMPLETED**
- [x] **Project Setup** - Set up Vite + React + TypeScript project
- [x] **Tailwind Configuration** - Configure CSS framework and build tools
- [x] **Layout Foundation** - Implement terminal-style layout container
- [x] **Basic Components** - Create session list and metric cards with mock data

**✨ Phase 1 Achievements:**
- Fully functional React app with terminal aesthetic
- Complete component architecture implemented
- Mock data integration working
- Beautiful responsive design matching original HTML mockup
- Development server running at http://localhost:5173/

#### Phase 2: Data Integration (Week 2)
- [ ] **API Integration** - Set up service layer and connect to localhost:8080
- [ ] **State Management** - Configure Zustand stores and React Query
- [ ] **Real-time Updates** - Implement WebSocket connection

#### Phase 3: Interactive Features (Week 3)
- [ ] **Charts and Visualizations** - Implement Chart.js components
- [ ] **Enhanced UI** - Add keyboard navigation and search
- [ ] **User Experience** - Loading states and animations

#### Phase 4: Polish and Optimization (Week 4)
- [ ] **Performance** - Code splitting and optimization
- [ ] **Accessibility** - ARIA labels and keyboard support
- [ ] **Testing** - Unit tests and documentation

---

## Overview

This document outlines the complete frontend implementation strategy for the Claude Session Manager web application. Based on the existing HTML mockup and PRD specifications, we'll build a modern React-based dashboard that provides real-time visibility into Claude Code sessions with beautiful visualizations and responsive design.

## Technology Stack

### Core Framework
- **React 18.2+** with TypeScript for type safety
- **Vite** for fast development and optimized builds
- **React Router DOM 6+** for client-side routing

### Styling & UI
- **Tailwind CSS 3.3+** for utility-first styling
- **Headless UI** for accessible component primitives
- **Heroicons** for consistent iconography
- **Custom CSS animations** for smooth transitions

### Data Visualization
- **Chart.js 4.4+** with react-chartjs-2 for interactive charts
- **Recharts** as alternative for React-native charts
- Custom CSS bar charts for simple visualizations

### State Management
- **Zustand** for lightweight global state management
- **React Query (TanStack Query)** for server state management
- **React Context** for theme and user preferences

### Real-time Communication
- **Native WebSocket API** for real-time updates
- **Socket.io-client** as fallback option
- Custom hook for connection management

### API Integration
- **Axios** for HTTP requests
- **OpenAPI/Swagger** integration for type-safe API calls
- **Backend API**: localhost:8080

## Component Architecture

### Core Layout Structure

```
src/
├── components/
│   ├── Layout/
│   │   ├── AppLayout.tsx           # Main application shell
│   │   ├── Terminal.tsx            # Terminal-style container
│   │   ├── TerminalHeader.tsx      # Header with controls
│   │   └── Navigation.tsx          # Tab navigation
│   ├── Sidebar/
│   │   ├── SessionSidebar.tsx      # Left sidebar container
│   │   ├── SessionList.tsx         # Session list component
│   │   ├── SessionCard.tsx         # Individual session item
│   │   ├── SessionTabs.tsx         # Sessions/Projects tabs
│   │   └── StatsSection.tsx        # Daily stats panel
│   ├── Dashboard/
│   │   ├── MainDashboard.tsx       # Primary dashboard view
│   │   ├── SessionHeader.tsx       # Selected session header
│   │   ├── MetricsGrid.tsx         # 4-card metrics layout
│   │   ├── MetricCard.tsx          # Individual metric card
│   │   └── RefreshButton.tsx       # Refresh control
│   ├── Charts/
│   │   ├── TokenUsageChart.tsx     # Bar chart component
│   │   ├── ChartContainer.tsx      # Chart wrapper with controls
│   │   └── TimeRangeSelector.tsx   # Chart time controls
│   ├── ActivityFeed/
│   │   ├── ActivityFeed.tsx        # Recent activity list
│   │   ├── ActivityItem.tsx        # Individual activity
│   │   └── ActivityIcon.tsx        # Status icons
│   ├── Common/
│   │   ├── StatusBadge.tsx         # Session status indicators
│   │   ├── ProgressBar.tsx         # Progress visualization
│   │   ├── LoadingSpinner.tsx      # Loading states
│   │   └── ErrorBoundary.tsx       # Error handling
│   └── UI/
│       ├── Button.tsx              # Reusable button component
│       ├── Card.tsx                # Glass-morphism cards
│       ├── Badge.tsx               # Status badges
│       └── ScrollArea.tsx          # Custom scrollbars
├── hooks/
│   ├── useWebSocket.ts             # WebSocket connection
│   ├── useSessionData.ts           # Session data management
│   ├── useRealTimeUpdates.ts       # Real-time data updates
│   ├── useKeyboardNavigation.ts    # Keyboard shortcuts
│   └── useLocalStorage.ts          # Persistent preferences
├── services/
│   ├── api.ts                      # API client configuration
│   ├── sessionService.ts           # Session data API
│   ├── metricsService.ts           # Metrics and analytics
│   └── websocketService.ts         # WebSocket management
├── store/
│   ├── sessionStore.ts             # Session state management
│   ├── uiStore.ts                  # UI state (selected session, etc.)
│   └── settingsStore.ts            # User preferences
├── types/
│   ├── session.ts                  # Session-related types
│   ├── metrics.ts                  # Metrics and analytics types
│   └── api.ts                      # API response types
├── utils/
│   ├── formatters.ts               # Data formatting utilities
│   ├── constants.ts                # App constants
│   └── classNames.ts               # CSS class utilities
└── styles/
    ├── globals.css                 # Global styles and Tailwind
    ├── components.css              # Component-specific styles
    └── animations.css              # Custom animations
```

## Data Models & TypeScript Types

### Core Session Types

```typescript
// types/session.ts
export interface Session {
  id: string;
  projectPath: string;
  projectName: string;
  gitBranch: string;
  gitWorktree: string;
  status: SessionStatus;
  startTime: Date;
  lastActivity: Date;
  currentTask: string;
  tokensUsed: TokenUsage;
  filesModified: string[];
  messages: Message[];
  progress?: number;
}

export interface TokenUsage {
  inputTokens: number;
  outputTokens: number;
  totalTokens: number;
  estimatedCost: number;
}

export enum SessionStatus {
  WORKING = 'working',
  IDLE = 'idle',
  COMPLETE = 'complete',
  ERROR = 'error'
}

export interface ActivityItem {
  id: string;
  sessionId: string;
  sessionName: string;
  message: string;
  timestamp: Date;
  tokenCount: number;
  type: 'working' | 'complete' | 'error';
}

export interface DailyStats {
  activeSessions: number;
  tokensUsed: number;
  costToday: number;
  projectCount: number;
}
```

### API Integration Types

```typescript
// types/api.ts
export interface SessionResponse {
  sessions: Session[];
  totalCount: number;
  lastUpdated: Date;
}

export interface MetricsResponse {
  dailyStats: DailyStats;
  chartData: ChartDataPoint[];
  recentActivity: ActivityItem[];
}

export interface ChartDataPoint {
  timestamp: Date;
  value: number;
  label: string;
}

export interface WebSocketMessage {
  type: 'sessions' | 'activity' | 'metrics' | 'status';
  data: any;
  timestamp: Date;
}
```

## State Management Strategy

### Zustand Store Structure

```typescript
// store/sessionStore.ts
interface SessionState {
  sessions: Session[];
  selectedSessionId: string | null;
  isLoading: boolean;
  error: string | null;
  lastUpdated: Date | null;
  
  // Actions
  setSessions: (sessions: Session[]) => void;
  selectSession: (id: string) => void;
  updateSession: (id: string, updates: Partial<Session>) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

// store/uiStore.ts
interface UIState {
  sidebarCollapsed: boolean;
  activeTab: 'sessions' | 'projects';
  theme: 'dark' | 'light' | 'auto';
  chartTimeRange: '24h' | '7d' | '30d';
  
  // Actions
  toggleSidebar: () => void;
  setActiveTab: (tab: 'sessions' | 'projects') => void;
  setTheme: (theme: 'dark' | 'light' | 'auto') => void;
  setChartTimeRange: (range: '24h' | '7d' | '30d') => void;
}
```

### React Query Integration

```typescript
// hooks/useSessionData.ts
export const useSessionData = () => {
  return useQuery({
    queryKey: ['sessions'],
    queryFn: () => sessionService.getAllSessions(),
    refetchInterval: 30000, // 30 seconds
    staleTime: 10000, // 10 seconds
  });
};

export const useSessionMetrics = (sessionId: string) => {
  return useQuery({
    queryKey: ['metrics', sessionId],
    queryFn: () => metricsService.getSessionMetrics(sessionId),
    enabled: !!sessionId,
    refetchInterval: 15000, // 15 seconds
  });
};
```

## Real-time Updates Implementation

### WebSocket Hook

```typescript
// hooks/useWebSocket.ts
export const useWebSocket = (url: string) => {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');
  
  useEffect(() => {
    const ws = new WebSocket(url);
    
    ws.onopen = () => {
      setConnectionStatus('connected');
      setSocket(ws);
    };
    
    ws.onclose = () => {
      setConnectionStatus('disconnected');
      setSocket(null);
      
      // Reconnection logic
      setTimeout(() => {
        setConnectionStatus('connecting');
      }, 3000);
    };
    
    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      setConnectionStatus('disconnected');
    };
    
    return () => {
      ws.close();
    };
  }, [url]);
  
  const sendMessage = useCallback((message: any) => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message));
    }
  }, [socket]);
  
  return { socket, connectionStatus, sendMessage };
};
```

### Real-time Data Updates

```typescript
// hooks/useRealTimeUpdates.ts
export const useRealTimeUpdates = () => {
  const { socket } = useWebSocket('ws://localhost:8080/ws');
  const { setSessions, updateSession } = useSessionStore();
  const queryClient = useQueryClient();
  
  useEffect(() => {
    if (!socket) return;
    
    const handleMessage = (event: MessageEvent) => {
      const message: WebSocketMessage = JSON.parse(event.data);
      
      switch (message.type) {
        case 'sessions':
          setSessions(message.data);
          queryClient.invalidateQueries(['sessions']);
          break;
          
        case 'activity':
          queryClient.invalidateQueries(['metrics']);
          break;
          
        case 'status':
          updateSession(message.data.sessionId, { status: message.data.status });
          break;
      }
    };
    
    socket.addEventListener('message', handleMessage);
    
    return () => {
      socket.removeEventListener('message', handleMessage);
    };
  }, [socket, setSessions, updateSession, queryClient]);
};
```

## Component Implementation Examples

### Session Card Component

```typescript
// components/Sidebar/SessionCard.tsx
interface SessionCardProps {
  session: Session;
  isSelected: boolean;
  onClick: () => void;
}

export const SessionCard: React.FC<SessionCardProps> = ({
  session,
  isSelected,
  onClick
}) => {
  const formatTime = (date: Date) => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / (1000 * 60));
    
    if (minutes < 1) return 'just now';
    if (minutes < 60) return `${minutes}m`;
    if (minutes < 1440) return `${Math.floor(minutes / 60)}h`;
    return `${Math.floor(minutes / 1440)}d`;
  };
  
  return (
    <div
      className={cn(
        "session-item",
        "bg-gray-800 border border-gray-700 rounded-lg p-3 mb-2",
        "cursor-pointer transition-all duration-200",
        "hover:border-gray-600 hover:bg-gray-750",
        isSelected && "border-primary bg-primary/5 shadow-lg shadow-primary/20"
      )}
      onClick={onClick}
    >
      <div className="flex justify-between items-center mb-2">
        <h3 className="text-sm font-semibold text-white">
          {session.projectName}
        </h3>
        <StatusBadge status={session.status} />
      </div>
      
      <div className="flex justify-between items-center text-xs text-gray-400 mb-1">
        <span className="text-cyan-400">{session.gitBranch}</span>
        <span>{formatTime(session.lastActivity)}</span>
      </div>
      
      <p className="text-xs text-gray-300 truncate mb-2">
        {session.currentTask}
      </p>
      
      {session.progress && (
        <ProgressBar progress={session.progress} />
      )}
    </div>
  );
};
```

### Metrics Card Component

```typescript
// components/Dashboard/MetricCard.tsx
interface MetricCardProps {
  title: string;
  value: string | number;
  change?: string;
  icon: React.ReactNode;
  trend?: 'up' | 'down' | 'neutral';
}

export const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  change,
  icon,
  trend = 'neutral'
}) => {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
      <div className="flex justify-between items-center mb-3">
        <h3 className="text-xs text-gray-400 font-medium">{title}</h3>
        <div className="text-primary text-sm">{icon}</div>
      </div>
      
      <div className="text-2xl font-bold text-white mb-1">
        {typeof value === 'number' ? value.toLocaleString() : value}
      </div>
      
      {change && (
        <div className={cn(
          "text-xs",
          trend === 'up' && "text-green-400",
          trend === 'down' && "text-red-400",
          trend === 'neutral' && "text-primary"
        )}>
          {change}
        </div>
      )}
    </div>
  );
};
```

### Token Usage Chart

```typescript
// components/Charts/TokenUsageChart.tsx
export const TokenUsageChart: React.FC = () => {
  const { data: chartData, isLoading } = useQuery({
    queryKey: ['chart-data'],
    queryFn: () => metricsService.getChartData(),
  });
  
  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: false },
      tooltip: {
        backgroundColor: '#1f2937',
        titleColor: '#f9fafb',
        bodyColor: '#f9fafb',
        borderColor: '#374151',
        borderWidth: 1,
        callbacks: {
          label: (context: any) => `${context.parsed.y.toLocaleString()} tokens`
        }
      }
    },
    scales: {
      x: {
        grid: { display: false },
        ticks: { color: '#9ca3af', font: { size: 10 } }
      },
      y: {
        beginAtZero: true,
        grid: { color: '#374151' },
        ticks: { color: '#9ca3af', font: { size: 10 } }
      }
    },
    animation: { duration: 750 }
  };
  
  if (isLoading) {
    return <div className="h-48 flex items-center justify-center">
      <LoadingSpinner />
    </div>;
  }
  
  return (
    <div className="chart-container bg-gray-800 border border-gray-700 rounded-lg p-4">
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-sm font-semibold text-white">
          🔥 Token Usage (Last 24h)
        </h3>
        <TimeRangeSelector />
      </div>
      
      <div className="h-48">
        <Bar data={chartData} options={chartOptions} />
      </div>
    </div>
  );
};
```

## Styling Implementation

### Tailwind Configuration

```javascript
// tailwind.config.js
module.exports = {
  content: ['./src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        primary: '#00ff96',
        secondary: '#00d4ff',
        warning: '#ffbd2e',
        error: '#ff5f56',
        success: '#27ca3f',
        
        gray: {
          850: '#1e1e1e',
          900: '#0d0d0d',
          950: '#161616'
        }
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'Monaco', 'Consolas', 'monospace']
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'fade-in': 'fadeIn 0.5s ease-in-out',
        'slide-up': 'slideUp 0.3s ease-out'
      },
      backdropBlur: {
        xs: '2px'
      }
    }
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography')
  ]
};
```

### Custom CSS Classes

```css
/* styles/components.css */
.glass-panel {
  @apply bg-gray-800/80 backdrop-blur-md border border-white/10 shadow-xl rounded-xl;
}

.session-item {
  @apply bg-gray-850 border border-gray-700 rounded-lg p-3 cursor-pointer;
  @apply transition-all duration-200 hover:border-gray-600 hover:bg-gray-800;
}

.session-item.active {
  @apply border-primary bg-primary/5 shadow-lg shadow-primary/20;
}

.status-working {
  @apply bg-primary/20 text-primary px-2 py-1 rounded text-xs font-medium;
  animation: pulse 2s infinite;
}

.status-idle {
  @apply bg-warning/20 text-warning px-2 py-1 rounded text-xs font-medium;
}

.status-complete {
  @apply bg-success/20 text-success px-2 py-1 rounded text-xs font-medium;
}

.status-error {
  @apply bg-error/20 text-error px-2 py-1 rounded text-xs font-medium;
}

.terminal-container {
  @apply bg-gray-900 border-2 border-gray-700 rounded-lg shadow-2xl;
  @apply shadow-primary/10 font-mono;
}

.scrollbar-custom::-webkit-scrollbar {
  @apply w-1.5;
}

.scrollbar-custom::-webkit-scrollbar-track {
  @apply bg-gray-900;
}

.scrollbar-custom::-webkit-scrollbar-thumb {
  @apply bg-gray-700 rounded-full;
}

.scrollbar-custom::-webkit-scrollbar-thumb:hover {
  @apply bg-gray-600;
}
```

## API Integration Strategy

### Service Layer

```typescript
// services/api.ts
import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
});

// Request interceptor for auth (if needed)
api.interceptors.request.use(
  (config) => {
    // Add auth headers if needed
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error);
    return Promise.reject(error);
  }
);

export default api;

// services/sessionService.ts
export const sessionService = {
  getAllSessions: (): Promise<SessionResponse> =>
    api.get('/sessions').then(res => res.data),
    
  getSession: (id: string): Promise<Session> =>
    api.get(`/sessions/${id}`).then(res => res.data),
    
  getSessionMetrics: (id: string): Promise<MetricsResponse> =>
    api.get(`/sessions/${id}/metrics`).then(res => res.data),
    
  getDailyStats: (): Promise<DailyStats> =>
    api.get('/stats/daily').then(res => res.data),
    
  getChartData: (timeRange: string = '24h'): Promise<ChartDataPoint[]> =>
    api.get(`/charts/tokens?range=${timeRange}`).then(res => res.data)
};
```

### Error Handling & Loading States

```typescript
// components/Common/ErrorBoundary.tsx
export class ErrorBoundary extends React.Component {
  constructor(props: any) {
    super(props);
    this.state = { hasError: false, error: null };
  }
  
  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }
  
  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
  }
  
  render() {
    if (this.state.hasError) {
      return (
        <div className="p-8 text-center">
          <div className="text-error text-lg mb-2">Something went wrong</div>
          <button
            className="text-primary hover:underline"
            onClick={() => this.setState({ hasError: false })}
          >
            Try again
          </button>
        </div>
      );
    }
    
    return this.props.children;
  }
}

// components/Common/LoadingSpinner.tsx
export const LoadingSpinner: React.FC = () => (
  <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
);
```

## Implementation Roadmap

### Phase 1: Core Infrastructure (Week 1) - ✅ **COMPLETED**
1. **Project Setup** - ✅ **COMPLETED**
   - [x] Initialize Vite + React + TypeScript project
   - [x] Configure Tailwind CSS and build tools
   - [x] Set up folder structure and basic routing

2. **Layout Foundation** - ✅ **COMPLETED**
   - [x] Implement terminal-style layout container
   - [x] Create responsive sidebar and main content areas
   - [x] Add basic navigation and tab switching

3. **Basic Components** - ✅ **COMPLETED**
   - [x] Session list with mock data
   - [x] Metric cards with static content
   - [x] Activity feed structure

### Phase 2: Data Integration (Week 2) - ⏳ **Next Phase**
1. **API Integration** - ⏳ **Ready to Start**
   - [ ] Set up Axios and service layer
   - [ ] Implement React Query for data fetching
   - [ ] Connect to localhost:8080 backend

2. **State Management** - ⏸️ **Pending**
   - [ ] Configure Zustand stores
   - [ ] Implement session selection logic
   - [ ] Add error handling and loading states

3. **Real-time Updates** - ⏸️ **Pending**
   - [ ] WebSocket connection setup
   - [ ] Real-time session status updates
   - [ ] Live activity feed updates

### Phase 3: Interactive Features (Week 3) - ⏸️ **Pending**
1. **Charts and Visualizations**
   - [ ] Implement Chart.js token usage charts
   - [ ] Add interactive tooltips and animations
   - [ ] Time range selection controls

2. **Enhanced UI**
   - [ ] Keyboard navigation support
   - [ ] Search and filter functionality
   - [ ] Responsive design improvements

3. **User Experience**
   - [ ] Loading states and error boundaries
   - [ ] Smooth transitions and animations
   - [ ] Progress indicators

### Phase 4: Polish and Optimization (Week 4) - ⏸️ **Pending**
1. **Performance**
   - [ ] Code splitting and lazy loading
   - [ ] Optimize re-renders and memory usage
   - [ ] Bundle size optimization

2. **Accessibility**
   - [ ] ARIA labels and keyboard navigation
   - [ ] Screen reader support
   - [ ] High contrast mode

3. **Testing and Documentation**
   - [ ] Unit tests for key components
   - [ ] Integration tests for API calls
   - [ ] Component documentation

## File Structure Setup

```bash
# Initial project setup commands
npm create vite@latest . -- --template react-ts
npm install tailwindcss @tailwindcss/forms @tailwindcss/typography
npm install @types/react @types/react-dom
npm install react-router-dom zustand axios react-query
npm install chart.js react-chartjs-2 @heroicons/react
npm install clsx tailwind-merge
```

### Package.json Dependencies

```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.20.1",
    "zustand": "^4.4.7",
    "axios": "^1.6.2",
    "@tanstack/react-query": "^5.8.4",
    "chart.js": "^4.4.0",
    "react-chartjs-2": "^5.2.0",
    "@heroicons/react": "^2.0.18",
    "clsx": "^2.0.0",
    "tailwind-merge": "^2.0.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.43",
    "@types/react-dom": "^18.2.17",
    "@vitejs/plugin-react": "^4.2.0",
    "typescript": "^5.3.3",
    "vite": "^5.0.8",
    "tailwindcss": "^3.3.6",
    "@tailwindcss/forms": "^0.5.7",
    "@tailwindcss/typography": "^0.5.10",
    "autoprefixer": "^10.4.16",
    "postcss": "^8.4.32"
  }
}
```

## Conclusion

This implementation document provides a complete roadmap for building the Claude Session Manager frontend. The architecture focuses on:

- **Modern React patterns** with TypeScript for type safety
- **Real-time data synchronization** with WebSocket integration  
- **Beautiful, responsive UI** matching the terminal aesthetic
- **Performance optimization** with proper state management
- **Scalable component architecture** for future enhancements

The modular design allows for incremental development while maintaining code quality and user experience standards. The integration with the localhost:8080 backend API ensures seamless data flow and real-time updates for an optimal developer experience.