import React from 'react';
import { cn } from '../../utils/classNames';

interface ConnectionStatusProps {
  apiStatus: 'connected' | 'error' | 'loading';
  wsStatus: 'connected' | 'disconnected' | 'connecting';
  className?: string;
}

export const ConnectionStatus: React.FC<ConnectionStatusProps> = ({
  apiStatus,
  wsStatus,
  className
}) => {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
        return 'text-success';
      case 'error':
      case 'disconnected':
        return 'text-error';
      case 'loading':
      case 'connecting':
        return 'text-warning';
      default:
        return 'text-gray-400';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'connected':
        return '🟢';
      case 'error':
      case 'disconnected':
        return '🔴';
      case 'loading':
      case 'connecting':
        return '🟡';
      default:
        return '⚪';
    }
  };

  return (
    <div className={cn("flex items-center gap-3 text-xs", className)}>
      <div className="flex items-center gap-1">
        <span>{getStatusIcon(apiStatus)}</span>
        <span className={getStatusColor(apiStatus)}>
          API: {apiStatus}
        </span>
      </div>
      <div className="flex items-center gap-1">
        <span>{getStatusIcon(wsStatus)}</span>
        <span className={getStatusColor(wsStatus)}>
          Live: {wsStatus}
        </span>
      </div>
    </div>
  );
};