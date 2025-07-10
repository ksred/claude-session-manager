import React from 'react';
import { LoadingSpinner } from './LoadingSpinner';
import { cn } from '../../utils/classNames';

interface LoadingStateProps {
  message?: string;
  className?: string;
}

export const LoadingState: React.FC<LoadingStateProps> = ({
  message = "Loading...",
  className
}) => {
  return (
    <div className={cn(
      "flex flex-col items-center justify-center p-8 text-center",
      className
    )}>
      <LoadingSpinner size="lg" className="mb-4" />
      <p className="text-gray-400">{message}</p>
    </div>
  );
};