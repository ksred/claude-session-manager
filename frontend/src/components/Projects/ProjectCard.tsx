import React from 'react';
import { Project } from '../../types/project';
import { formatTime, formatTokens, formatCost, formatDuration } from '../../utils/formatters';
import { cn } from '../../utils/classNames';

interface ProjectCardProps {
  project: Project;
  isSelected: boolean;
  onClick: () => void;
  className?: string;
}

export const ProjectCard: React.FC<ProjectCardProps> = ({
  project,
  isSelected,
  onClick,
  className
}) => {
  const mostUsedModel = Object.entries(project.models)
    .sort(([, a], [, b]) => b - a)[0]?.[0] || 'Unknown';

  return (
    <div
      className={cn(
        "session-item mb-2 cursor-pointer",
        isSelected && "active",
        className
      )}
      onClick={onClick}
    >
      {/* Header */}
      <div className="flex justify-between items-center mb-2">
        <h3 className="text-sm font-semibold text-white truncate">
          {project.name}
        </h3>
        <div className="flex items-center gap-1">
          {project.activeSessions > 0 && (
            <span className="status-working text-xs px-1">
              {project.activeSessions} active
            </span>
          )}
        </div>
      </div>

      {/* Project Stats */}
      <div className="grid grid-cols-2 gap-2 text-xs mb-2">
        <div className="text-gray-400">
          Sessions: <span className="text-white">{project.totalSessions}</span>
        </div>
        <div className="text-gray-400">
          Branches: <span className="text-white">{project.branches.length}</span>
        </div>
        <div className="text-gray-400">
          Tokens: <span className="text-primary">{formatTokens(project.totalTokens.total_tokens)}</span>
        </div>
        <div className="text-gray-400">
          Cost: <span className="text-warning">{formatCost(project.totalCost)}</span>
        </div>
      </div>

      {/* Time and Model */}
      <div className="flex justify-between items-center text-xs text-gray-500 mb-2">
        <span>Duration: {formatDuration(project.totalDuration)}</span>
        <span>{formatTime(project.lastActivity)}</span>
      </div>

      {/* Files and Model */}
      <div className="flex justify-between items-center text-xs text-gray-500">
        <span>{project.filesModified.length} files modified</span>
        <span className="text-secondary truncate">
          {mostUsedModel.replace('claude-3-', '').replace('-20240229', '').replace('-20240307', '').replace('-20241022', '')}
        </span>
      </div>
    </div>
  );
};