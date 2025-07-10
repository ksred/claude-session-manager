import React from 'react';
import { Session } from '../../types/session';
import { StatusBadge } from '../Common/StatusBadge';
import { ProgressBar } from '../Common/ProgressBar';
import { formatTime, formatModel, formatDuration } from '../../utils/formatters';
import { cn } from '../../utils/classNames';

interface SessionCardProps {
  session: Session;
  isSelected: boolean;
  onClick: () => void;
  className?: string;
}

export const SessionCard: React.FC<SessionCardProps> = ({
  session,
  isSelected,
  onClick,
  className
}) => {
  return (
    <div
      className={cn(
        "session-item mb-2",
        isSelected && "active",
        className
      )}
      onClick={onClick}
    >
      {/* Header */}
      <div className="flex justify-between items-center mb-2">
        <h3 className="text-sm font-semibold text-white truncate">
          {session.project_name}
        </h3>
        <StatusBadge status={session.status} />
      </div>

      {/* Project Info and Model */}
      <div className="flex justify-between items-center text-xs text-gray-400 mb-1">
        <span className="text-secondary truncate">
          {session.git_branch}
        </span>
        <span>{formatTime(session.updated_at)}</span>
      </div>

      {/* Model and Duration */}
      <div className="flex justify-between items-center text-xs text-gray-500 mb-2">
        <span className="text-warning">
          {formatModel(session.model)}
        </span>
        <span>{formatDuration(session.duration_seconds)}</span>
      </div>

      {/* Current Task */}
      <p className="text-xs text-gray-300 truncate mb-2">
        {session.current_task}
      </p>

      {/* Message Count */}
      <div className="flex justify-between items-center text-xs text-gray-500 mb-2">
        <span>{session.message_count} messages</span>
        <span className="text-primary">
          {session.tokens_used.total_tokens.toLocaleString()} tokens
        </span>
      </div>

      {/* Progress Bar */}
      {session.progress !== undefined && (
        <ProgressBar progress={session.progress} />
      )}
    </div>
  );
};