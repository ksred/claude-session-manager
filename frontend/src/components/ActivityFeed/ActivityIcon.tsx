import React from 'react';
import { cn } from '../../utils/classNames';

interface ActivityIconProps {
  type: 'working' | 'complete' | 'error';
  className?: string;
}

export const ActivityIcon: React.FC<ActivityIconProps> = ({ type, className }) => {
  const getIconClasses = () => {
    switch (type) {
      case 'working':
        return 'bg-primary/20 text-primary';
      case 'complete':
        return 'bg-success/20 text-success';
      case 'error':
        return 'bg-error/20 text-error';
      default:
        return 'bg-gray-600 text-gray-300';
    }
  };

  const getIcon = () => {
    switch (type) {
      case 'working':
        return '⚡';
      case 'complete':
        return '✓';
      case 'error':
        return '✗';
      default:
        return '•';
    }
  };

  return (
    <div className={cn(
      "w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold",
      getIconClasses(),
      className
    )}>
      {getIcon()}
    </div>
  );
};