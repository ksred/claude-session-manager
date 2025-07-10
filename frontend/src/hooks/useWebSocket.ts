import { useState, useEffect, useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { WebSocketMessage } from '../types/api';
import { sessionKeys } from './useSessionData';
import { createDebouncedInvalidator } from '../utils/debounce';

const WS_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/ws`;
const RECONNECT_INTERVAL = 3000;
const MAX_RECONNECT_ATTEMPTS = 5;
const INVALIDATION_DELAY = 5000; // 5 seconds debounce for all invalidations

export const useWebSocket = () => {
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectCountRef = useRef(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const queryClient = useQueryClient();
  const debouncedInvalidator = useRef(createDebouncedInvalidator(INVALIDATION_DELAY));

  const connect = useCallback(() => {
    try {
      setConnectionStatus('connecting');
      setError(null);
      
      const ws = new WebSocket(WS_URL);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('🔗 WebSocket connected');
        setConnectionStatus('connected');
        reconnectCountRef.current = 0;
        setError(null);
        
        // Send subscribe message
        ws.send(JSON.stringify({
          type: 'subscribe',
          timestamp: Date.now()
        }));
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          console.log('📨 WebSocket message:', message);
          
          // Handle different message types
          switch (message.type) {
            case 'session_created':
              // New session created
              if (message.data?.session_id) {
                const sessionId = message.data.session_id;
                
                // Debounce fetching the full session data for the new session
                debouncedInvalidator.current.debounceInvalidation(
                  `fetch-session-${sessionId}`,
                  () => {
                    import('../services/sessionService').then(({ sessionService }) => {
                      sessionService.getSessionById(sessionId).then(newSession => {
                    console.log('🆕 New session created:', newSession.project_name);
                    
                    // Add the new session to all relevant lists
                    queryClient.setQueriesData(
                      { queryKey: sessionKeys.list({}) },
                      (oldData: any) => {
                        if (!oldData?.sessions) return oldData;
                        
                        // Add new session at the beginning (most recent)
                        return {
                          ...oldData,
                          sessions: [newSession, ...oldData.sessions]
                        };
                      }
                    );
                    
                    // Add to active sessions if active
                    if (newSession.is_active) {
                      queryClient.setQueriesData(
                        { queryKey: sessionKeys.list({ active: true }) },
                        (oldData: any) => {
                          if (!oldData?.sessions) return oldData;
                          
                          return {
                            ...oldData,
                            sessions: [newSession, ...oldData.sessions]
                          };
                        }
                      );
                    }
                    
                    // Add to recent sessions
                    queryClient.setQueriesData(
                      { queryKey: sessionKeys.list({ recent: true }) },
                      (oldData: any) => {
                        if (!oldData?.sessions) return oldData;
                        
                        // Keep only the specified limit of recent sessions
                        const sessions = [newSession, ...oldData.sessions];
                        const limit = oldData.sessions.length || 10;
                        
                        return {
                          ...oldData,
                          sessions: sessions.slice(0, limit)
                        };
                      }
                    );
                    
                    // Invalidate metrics to include the new session
                    debouncedInvalidator.current.debounceInvalidation(
                      'metrics',
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() })
                    );
                  }).catch(error => {
                    console.error('❌ Failed to fetch new session data:', error);
                    // Fall back to invalidating queries if fetch fails
                    debouncedInvalidator.current.debounceInvalidation(
                      'sessions-all',
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.all })
                    );
                    debouncedInvalidator.current.debounceInvalidation(
                      'metrics',
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() })
                    );
                    debouncedInvalidator.current.debounceInvalidation(
                      'sessions-active',
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.active() })
                    );
                    debouncedInvalidator.current.debounceInvalidation(
                      'sessions-recent',
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.recent() })
                    );
                      });
                    });
                  }
                );
              } else {
                // Fall back if no session_id provided
                debouncedInvalidator.current.debounceInvalidation(
                  'sessions-all',
                  () => queryClient.invalidateQueries({ queryKey: sessionKeys.all })
                );
                debouncedInvalidator.current.debounceInvalidation(
                  'metrics',
                  () => queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() })
                );
                debouncedInvalidator.current.debounceInvalidation(
                  'sessions-active',
                  () => queryClient.invalidateQueries({ queryKey: sessionKeys.active() })
                );
                debouncedInvalidator.current.debounceInvalidation(
                  'sessions-recent',
                  () => queryClient.invalidateQueries({ queryKey: sessionKeys.recent() })
                );
              }
              break;
              
            case 'session_update':
              // Existing session updated
              if (message.data?.session_id) {
                const sessionId = message.data.session_id;
                
                // Debounce fetching the full session data to get updated tokens, costs, etc.
                debouncedInvalidator.current.debounceInvalidation(
                  `fetch-session-update-${sessionId}`,
                  () => {
                    import('../services/sessionService').then(({ sessionService }) => {
                      sessionService.getSessionById(sessionId).then(updatedSession => {
                    console.log('📊 Fetched updated session data:', updatedSession);
                    
                    // Update the session detail cache
                    queryClient.setQueryData(sessionKeys.detail(sessionId), updatedSession);
                    
                    // Update the session in all lists without invalidating
                    // This prevents reordering and flashing
                    queryClient.setQueriesData(
                      { queryKey: sessionKeys.list({}) },
                      (oldData: any) => {
                        if (!oldData?.sessions) return oldData;
                        
                        const updatedSessions = oldData.sessions.map((session: any) => {
                          if (session.id === sessionId) {
                            // Use the full updated session data
                            return updatedSession;
                          }
                          return session;
                        });
                        
                        return { ...oldData, sessions: updatedSessions };
                      }
                    );
                    
                    // Update active sessions list if the session is active
                    if (updatedSession.is_active) {
                      queryClient.setQueriesData(
                        { queryKey: sessionKeys.list({ active: true }) },
                        (oldData: any) => {
                          if (!oldData?.sessions) return oldData;
                          
                          const updatedSessions = oldData.sessions.map((session: any) => {
                            if (session.id === sessionId) {
                              return updatedSession;
                            }
                            return session;
                          });
                          
                          return { ...oldData, sessions: updatedSessions };
                        }
                      );
                    }
                    
                    // Update recent sessions list
                    queryClient.setQueriesData(
                      { queryKey: sessionKeys.list({ recent: true }) },
                      (oldData: any) => {
                        if (!oldData?.sessions) return oldData;
                        
                        const updatedSessions = oldData.sessions.map((session: any) => {
                          if (session.id === sessionId) {
                            return updatedSession;
                          }
                          return session;
                        });
                        
                        return { ...oldData, sessions: updatedSessions };
                      }
                    );
                    
                    // Debounce metrics invalidation
                    debouncedInvalidator.current.debounceInvalidation(
                      'metrics',
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() })
                    );
                    
                    // Debounce timeline invalidation to prevent excessive API calls
                    debouncedInvalidator.current.debounceInvalidation(
                      `session-timeline-${sessionId}`,
                      () => {
                        queryClient.invalidateQueries({ 
                          queryKey: sessionKeys.sessionTokenTimeline(sessionId) 
                        });
                      }
                    );
                  }).catch(error => {
                    console.error('❌ Failed to fetch updated session data:', error);
                    // Fall back to invalidating queries if fetch fails
                    debouncedInvalidator.current.debounceInvalidation(
                      `session-detail-${sessionId}`,
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.detail(sessionId) })
                    );
                    debouncedInvalidator.current.debounceInvalidation(
                      'sessions-all',
                      () => queryClient.invalidateQueries({ queryKey: sessionKeys.all })
                    );
                      });
                    });
                  }
                );
              }
              break;
              
            case 'session_deleted':
              // Session deleted
              debouncedInvalidator.current.debounceInvalidation(
                'sessions-all',
                () => queryClient.invalidateQueries({ queryKey: sessionKeys.all })
              );
              debouncedInvalidator.current.debounceInvalidation(
                'metrics',
                () => queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() })
              );
              break;
              
            case 'activity_update':
              // Invalidate activity data
              debouncedInvalidator.current.debounceInvalidation(
                'activity',
                () => queryClient.invalidateQueries({ queryKey: sessionKeys.activity() })
              );
              
              // If it's a file modification, also update recent files
              if (message.data?.activity?.type === 'file_modified') {
                debouncedInvalidator.current.debounceInvalidation(
                  'files-recent',
                  () => queryClient.invalidateQueries({ queryKey: ['files', 'recent'] })
                );
              }
              break;
              
            case 'metrics_update':
              // Metrics updated - fetch updated session data if session_id provided
              if (message.data?.session_id) {
                const sessionId = message.data.session_id;
                
                // Debounce fetching updated session data to get new token counts and costs
                debouncedInvalidator.current.debounceInvalidation(
                  `fetch-metrics-update-${sessionId}`,
                  () => {
                    import('../services/sessionService').then(({ sessionService }) => {
                      sessionService.getSessionById(sessionId).then(updatedSession => {
                    console.log('📈 Metrics updated for session:', updatedSession.project_name);
                    
                    // Update the session detail cache
                    queryClient.setQueryData(sessionKeys.detail(sessionId), updatedSession);
                    
                    // Update the session in all lists
                    queryClient.setQueriesData(
                      { queryKey: sessionKeys.list({}) },
                      (oldData: any) => {
                        if (!oldData?.sessions) return oldData;
                        
                        const updatedSessions = oldData.sessions.map((session: any) => {
                          if (session.id === sessionId) {
                            return updatedSession;
                          }
                          return session;
                        });
                        
                        return { ...oldData, sessions: updatedSessions };
                      }
                    );
                    
                    // Debounce timeline invalidation to prevent excessive API calls
                    debouncedInvalidator.current.debounceInvalidation(
                      `session-timeline-${sessionId}`,
                      () => {
                        queryClient.invalidateQueries({ 
                          queryKey: sessionKeys.sessionTokenTimeline(sessionId) 
                        });
                      }
                    );
                      }).catch(error => {
                        console.error('❌ Failed to fetch session after metrics update:', error);
                      });
                    });
                  }
                );
              }
              
              // Always invalidate general metrics
              debouncedInvalidator.current.debounceInvalidation(
                'metrics',
                () => queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() })
              );
              debouncedInvalidator.current.debounceInvalidation(
                'usage',
                () => queryClient.invalidateQueries({ queryKey: sessionKeys.usage() })
              );
              break;
              
            case 'pong':
              // Heartbeat response
              console.log('💓 WebSocket heartbeat received');
              break;
              
            case 'subscribed':
              // Subscription confirmed
              console.log('✅ WebSocket subscription confirmed');
              break;
              
            default:
              console.log('Unknown WebSocket message type:', message.type);
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };

      ws.onclose = (event) => {
        console.log('🔌 WebSocket disconnected:', event.code, event.reason);
        setConnectionStatus('disconnected');
        wsRef.current = null;
        
        // Attempt to reconnect if not manually closed
        if (event.code !== 1000 && reconnectCountRef.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectCountRef.current++;
          console.log(`🔄 Attempting to reconnect (${reconnectCountRef.current}/${MAX_RECONNECT_ATTEMPTS})`);
          
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, RECONNECT_INTERVAL);
        } else if (reconnectCountRef.current >= MAX_RECONNECT_ATTEMPTS) {
          setError('Failed to connect after multiple attempts');
        }
      };

      ws.onerror = () => {
        // Don't log the error event itself as it's not very useful
        // The close event will provide more information
        setError('WebSocket connection error');
      };

    } catch (error) {
      console.error('❌ Failed to create WebSocket connection:', error);
      setError('Failed to create WebSocket connection');
      setConnectionStatus('disconnected');
    }
  }, [queryClient]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    
    if (wsRef.current) {
      wsRef.current.close(1000, 'Manual disconnect');
      wsRef.current = null;
    }
    
    setConnectionStatus('disconnected');
    reconnectCountRef.current = 0;
  }, []);

  const sendMessage = useCallback((message: any) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
      return true;
    }
    console.warn('⚠️ WebSocket not connected, message not sent');
    return false;
  }, []);

  useEffect(() => {
    // Add a small delay to prevent immediate connection on mount
    const connectTimeout = setTimeout(() => {
      connect();
    }, 100);
    
    // Set up periodic ping to keep connection alive
    const pingInterval = setInterval(() => {
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        sendMessage({ type: 'ping', timestamp: Date.now() });
      }
    }, 30000); // Ping every 30 seconds
    
    return () => {
      clearTimeout(connectTimeout);
      clearInterval(pingInterval);
      disconnect();
      // Clear all pending invalidations
      debouncedInvalidator.current.clearAll();
    };
  }, []); // Remove dependencies to prevent reconnect loops

  return {
    connectionStatus,
    error,
    connect,
    disconnect,
    sendMessage,
    isConnected: connectionStatus === 'connected'
  };
};