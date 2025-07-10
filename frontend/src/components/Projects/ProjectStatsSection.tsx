import React from 'react';
import { Project } from '../../types/project';
import { formatTokens, formatCost } from '../../utils/formatters';
import { cn } from '../../utils/classNames';

interface ProjectStatsSectionProps {
  projects: Project[];
  className?: string;
}

export const ProjectStatsSection: React.FC<ProjectStatsSectionProps> = ({ projects, className }) => {
  const totalProjects = projects.length;
  const activeProjects = projects.filter(p => p.activeSessions > 0).length;
  const totalSessions = projects.reduce((acc, p) => acc + p.totalSessions, 0);
  const totalTokens = projects.reduce((acc, p) => acc + p.totalTokens.total_tokens, 0);
  const totalCost = projects.reduce((acc, p) => acc + p.totalCost, 0);
  const totalFiles = new Set(projects.flatMap(p => p.filesModified)).size;

  const statItems = [
    { label: 'Total Projects', value: totalProjects },
    { label: 'Active Projects', value: activeProjects },
    { label: 'Total Sessions', value: totalSessions },
    { label: 'Total Tokens', value: formatTokens(totalTokens) },
    { label: 'Total Cost', value: formatCost(totalCost) },
    { label: 'Unique Files', value: totalFiles }
  ];

  return (
    <div className={cn(
      "border-t border-gray-700 p-4 bg-gray-850",
      className
    )}>
      <div className="flex items-center text-primary text-xs font-semibold mb-3">
        <span className="mr-2">📁</span>
        Project Stats
      </div>
      
      <div className="space-y-2">
        {statItems.map(item => (
          <div key={item.label} className="flex justify-between text-xs">
            <span className="text-gray-400">{item.label}</span>
            <span className="text-white font-medium">{item.value}</span>
          </div>
        ))}
      </div>
    </div>
  );
};