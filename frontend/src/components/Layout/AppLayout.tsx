import React, { useState } from 'react';
import { Terminal } from './Terminal';
import { TerminalHeader } from './TerminalHeader';
import { cn } from '../../utils/classNames';

interface AppLayoutProps {
  sidebar: React.ReactNode;
  main: React.ReactNode;
  apiStatus?: 'connected' | 'error' | 'loading';
  wsStatus?: 'connected' | 'disconnected' | 'connecting';
  className?: string;
}

export const AppLayout: React.FC<AppLayoutProps> = ({ 
  sidebar, 
  main,
  apiStatus = 'connected',
  wsStatus = 'connected',
  className 
}) => {
  const [sidebarCollapsed] = useState(false);

  return (
    <Terminal className={className}>
      <TerminalHeader 
        apiStatus={apiStatus}
        wsStatus={wsStatus}
      />
      
      <div className="flex flex-1 min-h-0">
        {/* Sidebar */}
        <div className={cn(
          "border-r border-gray-700 bg-gray-950 flex flex-col",
          sidebarCollapsed ? "w-12" : "w-80",
          "transition-all duration-300"
        )}>
          {sidebar}
        </div>

        {/* Main Content */}
        <div className="flex-1 flex flex-col min-w-0 min-h-0 overflow-hidden">
          {main}
        </div>
      </div>
    </Terminal>
  );
};