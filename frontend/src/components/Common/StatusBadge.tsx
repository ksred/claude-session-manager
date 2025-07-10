import React from 'react';
import { SessionStatus } from '../../types/session';
import { cn } from '../../utils/classNames';

interface StatusBadgeProps {
  status: SessionStatus;
  className?: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, className }) => {
  const getStatusClasses = () => {
    switch (status) {
      case SessionStatus.ACTIVE:
        return 'status-working';
      case SessionStatus.IDLE:
        return 'status-idle';
      case SessionStatus.COMPLETED:
        return 'status-complete';
      case SessionStatus.ERROR:
        return 'status-error';
      default:
        return 'status-idle';
    }
  };

  const getStatusText = () => {
    switch (status) {
      case SessionStatus.ACTIVE:
        return 'Active';
      case SessionStatus.IDLE:
        return 'Idle';
      case SessionStatus.COMPLETED:
        return 'Complete';
      case SessionStatus.ERROR:
        return 'Error';
      default:
        return 'Unknown';
    }
  };

  return (
    <span className={cn(getStatusClasses(), className)}>
      {getStatusText()}
    </span>
  );
};