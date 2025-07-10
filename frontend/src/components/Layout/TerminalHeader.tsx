import React from 'react';
import { ConnectionStatus } from '../Common/ConnectionStatus';
import { cn } from '../../utils/classNames';

interface TerminalHeaderProps {
  title?: string;
  subtitle?: string;
  apiStatus?: 'connected' | 'error' | 'loading';
  wsStatus?: 'connected' | 'disconnected' | 'connecting';
  className?: string;
}

export const TerminalHeader: React.FC<TerminalHeaderProps> = ({ 
  title = "claude-session-manager v1.2.3",
  subtitle = "Connected to localhost:8080",
  apiStatus = 'connected',
  wsStatus = 'connected',
  className 
}) => {
  return (
    <div className={cn(
      "bg-gradient-to-r from-gray-700 to-gray-850",
      "border-b border-gray-700 px-4 py-2",
      "flex justify-between items-center",
      "text-xs",
      className
    )}>
      <div className="text-primary font-semibold">
        {title}
      </div>
      
      <div className="flex items-center gap-4">
        <div className="text-gray-400">
          {subtitle}
        </div>
        <ConnectionStatus 
          apiStatus={apiStatus} 
          wsStatus={wsStatus} 
        />
      </div>
      
      <div className="flex gap-2">
        <div className="w-3 h-3 rounded-full bg-error"></div>
        <div className="w-3 h-3 rounded-full bg-warning"></div>
        <div className="w-3 h-3 rounded-full bg-success"></div>
      </div>
    </div>
  );
};