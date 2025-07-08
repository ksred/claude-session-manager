import { Session, SessionStatus } from '../types/session';
import { Project } from '../types/project';

export const groupSessionsByProject = (sessions: Session[]): Project[] => {
  const projectMap = new Map<string, Project>();

  sessions.forEach(session => {
    const projectKey = session.project_name;
    
    if (!projectMap.has(projectKey)) {
      projectMap.set(projectKey, {
        id: projectKey,
        name: session.project_name,
        path: session.project_path,
        sessions: [],
        totalSessions: 0,
        activeSessions: 0,
        totalTokens: {
          input_tokens: 0,
          output_tokens: 0,
          total_tokens: 0,
          estimated_cost: 0,
          cache_creation_input_tokens: 0,
          cache_read_input_tokens: 0
        },
        totalCost: 0,
        totalDuration: 0,
        lastActivity: session.updated_at,
        branches: [],
        filesModified: [],
        models: {}
      });
    }

    const project = projectMap.get(projectKey)!;
    
    // Add session to project
    project.sessions.push(session);
    project.totalSessions++;
    
    // Count active sessions
    if (session.status === SessionStatus.ACTIVE) {
      project.activeSessions++;
    }
    
    // Aggregate token usage
    project.totalTokens.input_tokens += session.tokens_used.input_tokens;
    project.totalTokens.output_tokens += session.tokens_used.output_tokens;
    project.totalTokens.total_tokens += session.tokens_used.total_tokens;
    project.totalTokens.estimated_cost += session.tokens_used.estimated_cost;
    project.totalTokens.cache_creation_input_tokens += session.tokens_used.cache_creation_input_tokens;
    project.totalTokens.cache_read_input_tokens += session.tokens_used.cache_read_input_tokens;
    
    // Aggregate cost
    project.totalCost += session.tokens_used.estimated_cost;
    
    // Aggregate duration
    project.totalDuration += session.duration_seconds;
    
    // Track latest activity
    if (new Date(session.updated_at) > new Date(project.lastActivity)) {
      project.lastActivity = session.updated_at;
    }
    
    // Collect unique branches
    if (!project.branches.includes(session.git_branch)) {
      project.branches.push(session.git_branch);
    }
    
    // Collect unique files
    session.files_modified?.forEach(file => {
      if (!project.filesModified.includes(file)) {
        project.filesModified.push(file);
      }
    });
    
    // Track model usage
    if (project.models[session.model]) {
      project.models[session.model]++;
    } else {
      project.models[session.model] = 1;
    }
  });

  return Array.from(projectMap.values()).sort((a, b) => 
    new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime()
  );
};