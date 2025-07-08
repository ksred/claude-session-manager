import React from 'react';
import { cn } from '../../utils/classNames';

interface SessionTabsProps {
  activeTab: 'sessions' | 'projects' | 'analytics';
  onTabChange: (tab: 'sessions' | 'projects' | 'analytics') => void;
  className?: string;
}

export const SessionTabs: React.FC<SessionTabsProps> = ({ 
  activeTab, 
  onTabChange, 
  className 
}) => {
  const tabs = [
    { id: 'sessions' as const, label: 'Sessions' },
    { id: 'projects' as const, label: 'Projects' },
    { id: 'analytics' as const, label: 'Analytics' }
  ];

  return (
    <div className={cn(
      "flex bg-gray-850 border-b border-gray-700 px-4",
      className
    )}>
      {tabs.map(tab => (
        <button
          key={tab.id}
          onClick={() => onTabChange(tab.id)}
          className={cn(
            "px-4 py-2 text-xs font-medium transition-all duration-200",
            "border-b-2 border-transparent",
            activeTab === tab.id
              ? "text-primary border-primary"
              : "text-gray-400 hover:text-white hover:bg-white/5"
          )}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
};