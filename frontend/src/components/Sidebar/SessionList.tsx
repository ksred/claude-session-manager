import React from 'react';
import { Session } from '../../types/session';
import { SessionCard } from './SessionCard';
import { cn } from '../../utils/classNames';

interface SessionListProps {
  sessions: Session[];
  selectedSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
  className?: string;
}

export const SessionList: React.FC<SessionListProps> = ({
  sessions,
  selectedSessionId,
  onSessionSelect,
  className
}) => {
  if (sessions.length === 0) {
    return (
      <div className={cn(
        "flex-1 flex items-center justify-center text-gray-400 text-sm",
        className
      )}>
        No active sessions
      </div>
    );
  }

  return (
    <div className={cn(
      "flex-1 overflow-y-auto p-4 scrollbar-custom",
      className
    )}>
      {sessions.map(session => (
        <SessionCard
          key={session.id}
          session={session}
          isSelected={session.id === selectedSessionId}
          onClick={() => onSessionSelect(session.id)}
        />
      ))}
    </div>
  );
};