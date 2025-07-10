import { useState, useEffect, useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { WebSocketMessage } from '../types/api';
import { sessionKeys } from './useSessionData';

const WS_URL = 'ws://localhost:8080/api/v1/ws';
const RECONNECT_INTERVAL = 3000;
const MAX_RECONNECT_ATTEMPTS = 5;

export const useWebSocket = () => {
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectCountRef = useRef(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const queryClient = useQueryClient();

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
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          console.log('📨 WebSocket message:', message);
          
          // Handle different message types
          switch (message.type) {
            case 'session_update':
            case 'session_created':
              // Invalidate sessions data to trigger refresh
              queryClient.invalidateQueries({ queryKey: sessionKeys.all });
              queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() });
              break;
              
            case 'activity_update':
              // Invalidate activity data
              queryClient.invalidateQueries({ queryKey: sessionKeys.activity() });
              break;
              
            case 'metrics_update':
              // Invalidate metrics data
              queryClient.invalidateQueries({ queryKey: sessionKeys.metrics() });
              queryClient.invalidateQueries({ queryKey: sessionKeys.usage() });
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
    
    return () => {
      clearTimeout(connectTimeout);
      disconnect();
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