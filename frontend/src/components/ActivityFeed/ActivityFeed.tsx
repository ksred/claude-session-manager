import React from 'react';
import { ActivityEntry } from '../../types/session';
import { ActivityItem } from './ActivityItem';
import { cn } from '../../utils/classNames';

interface ActivityFeedProps {
  activities: ActivityEntry[];
  className?: string;
}

export const ActivityFeed: React.FC<ActivityFeedProps> = ({ activities, className }) => {
  return (
    <div className={cn("activity-card max-h-80", className)}>
      <div className="flex items-center text-white text-sm font-semibold mb-4">
        <span className="mr-2">🕒</span>
        Recent Activity
      </div>
      
      <div className="overflow-y-auto scrollbar-custom max-h-64">
        {activities.length === 0 ? (
          <div className="text-gray-400 text-sm text-center py-8">
            No recent activity
          </div>
        ) : (
          activities.map((activity, index) => (
            <ActivityItem
              key={activity.id || `activity-${index}`}
              activity={activity}
            />
          ))
        )}
      </div>
    </div>
  );
};