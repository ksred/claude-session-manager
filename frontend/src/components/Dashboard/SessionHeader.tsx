import React from 'react';
import { Session } from '../../types/session';
import { cn } from '../../utils/classNames';

interface SessionHeaderProps {
  session: Session | null;
  onRefresh?: () => void;
  className?: string;
}

export const SessionHeader: React.FC<SessionHeaderProps> = ({ 
  session, 
  onRefresh,
  className 
}) => {
  return (
    <div className={cn(
      "flex justify-between items-start mb-5",
      className
    )}>
      <div>
        <h1 className="text-lg font-semibold text-white mb-1">
          {session ? session.title : 'No Session Selected'}
        </h1>
        {session && (
          <p className="text-sm text-gray-400">
            {session.project_name} • {session.git_branch}
          </p>
        )}
      </div>
      
      {onRefresh && (
        <button
          onClick={onRefresh}
          className="btn btn-primary text-xs"
        >
          Refresh
        </button>
      )}
    </div>
  );
};