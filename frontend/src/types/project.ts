import { Session, TokenUsage } from './session';

export interface Project {
  id: string;
  name: string;
  path: string;
  sessions: Session[];
  totalSessions: number;
  activeSessions: number;
  totalTokens: TokenUsage;
  totalCost: number;
  totalDuration: number;
  lastActivity: string;
  branches: string[];
  filesModified: string[];
  models: Record<string, number>;
}