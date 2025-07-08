import React from 'react';
import { ActivityEntry } from '../../types/session';
import { ActivityIcon } from './ActivityIcon';
import { formatTime } from '../../utils/formatters';
import { cn } from '../../utils/classNames';

interface ActivityItemProps {
  activity: ActivityEntry;
  className?: string;
}

// Map API activity types to icon types
const getIconType = (apiType: string): 'working' | 'complete' | 'error' => {
  switch (apiType) {
    case 'message_sent':
      return 'working';
    case 'session_created':
    case 'session_updated':
      return 'complete';
    case 'error':
      return 'error';
    default:
      return 'working';
  }
};

export const ActivityItem: React.FC<ActivityItemProps> = ({ activity, className }) => {
  const iconType = getIconType(activity.type);
  
  return (
    <div className={cn(
      "flex gap-3 pb-3 mb-3 border-b border-gray-700 last:border-b-0 last:pb-0 last:mb-0",
      className
    )}>
      <ActivityIcon type={iconType} className="mt-0.5" />
      
      <div className="flex-1 min-w-0">
        <div className="text-sm text-white mb-1">
          {activity.details}
        </div>
        <div className="text-xs text-gray-400">
          {activity.session_name} • {formatTime(activity.timestamp)} • {activity.type}
        </div>
      </div>
    </div>
  );
};