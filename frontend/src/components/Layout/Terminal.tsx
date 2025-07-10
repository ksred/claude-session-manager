import React from 'react';
import { cn } from '../../utils/classNames';

interface TerminalProps {
  children: React.ReactNode;
  className?: string;
}

export const Terminal: React.FC<TerminalProps> = ({ children, className }) => {
  return (
    <div className={cn(
      "terminal-container",
      "h-screen m-2 flex flex-col",
      "shadow-lg shadow-primary/10",
      className
    )}>
      {children}
    </div>
  );
};