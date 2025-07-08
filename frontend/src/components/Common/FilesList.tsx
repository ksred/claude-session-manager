import React from 'react';
import { cn } from '../../utils/classNames';

interface FilesListProps {
  files: string[] | null;
  title?: string;
  className?: string;
  maxItems?: number;
}

export const FilesList: React.FC<FilesListProps> = ({ 
  files, 
  title = "Modified Files",
  className,
  maxItems = 5
}) => {
  if (!files || files.length === 0) {
    return null;
  }

  const displayFiles = files.slice(0, maxItems);
  const remainingCount = files.length - maxItems;

  return (
    <div className={cn("activity-card", className)}>
      <div className="flex items-center text-white text-sm font-semibold mb-3">
        <span className="mr-2">📁</span>
        {title} ({files.length})
      </div>
      
      <div className="space-y-1">
        {displayFiles.map((file, index) => (
          <div 
            key={index}
            className="text-xs text-gray-300 font-mono truncate p-2 bg-gray-700/50 rounded border-l-2 border-primary/50"
          >
            {file}
          </div>
        ))}
        
        {remainingCount > 0 && (
          <div className="text-xs text-gray-400 italic pt-1">
            +{remainingCount} more files...
          </div>
        )}
      </div>
    </div>
  );
};