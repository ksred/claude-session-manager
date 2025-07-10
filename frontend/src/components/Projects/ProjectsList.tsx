import React from 'react';
import { Project } from '../../types/project';
import { ProjectCard } from './ProjectCard';
import { cn } from '../../utils/classNames';

interface ProjectsListProps {
  projects: Project[];
  selectedProjectId: string | null;
  onProjectSelect: (projectId: string) => void;
  className?: string;
}

export const ProjectsList: React.FC<ProjectsListProps> = ({
  projects,
  selectedProjectId,
  onProjectSelect,
  className
}) => {
  if (projects.length === 0) {
    return (
      <div className={cn(
        "flex-1 flex items-center justify-center text-gray-400 text-sm",
        className
      )}>
        No projects found
      </div>
    );
  }

  return (
    <div className={cn(
      "flex-1 overflow-y-auto p-4 scrollbar-custom",
      className
    )}>
      {projects.map(project => (
        <ProjectCard
          key={project.id}
          project={project}
          isSelected={project.id === selectedProjectId}
          onClick={() => onProjectSelect(project.id)}
        />
      ))}
    </div>
  );
};