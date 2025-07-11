import React, { useState, useEffect, useRef, useCallback } from 'react';
import { cn } from '../../utils/classNames';
import { useWebSocket } from '../../hooks/useWebSocket';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface TerminalChatProps {
  sessionId: string;
  className?: string;
}

interface ChatMessage {
  id: string;
  type: 'user' | 'claude' | 'system';
  content: string;
  timestamp: Date;
  metadata?: Record<string, any>;
}

interface ChatState {
  messages: ChatMessage[];
  isConnected: boolean;
  isTyping: boolean;
  chatSessionId?: string;
  status: 'idle' | 'starting' | 'active' | 'error';
  error?: string;
}

export const TerminalChat: React.FC<TerminalChatProps> = ({ sessionId, className }) => {
  const [state, setState] = useState<ChatState>({
    messages: [],
    isConnected: false,
    isTyping: false,
    status: 'idle',
  });
  const sessionStartedRef = useRef(false);
  const [input, setInput] = useState('');
  const [isComposing, setIsComposing] = useState(false);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const { sendMessage, isConnected } = useWebSocket();

  // Auto-scroll to bottom when new messages arrive
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [state.messages]);

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Start a new chat session
  const startChatSession = useCallback(() => {
    setState(prev => ({ ...prev, status: 'starting' }));
    sendMessage({
      type: 'chat:session:start',
      session_id: sessionId,
    });
  }, [sessionId, sendMessage]);

  // Update connection status
  useEffect(() => {
    setState(prev => ({ ...prev, isConnected }));
  }, [isConnected]);

  // Start chat session when component mounts
  useEffect(() => {
    if (isConnected && state.status === 'idle' && !sessionStartedRef.current) {
      sessionStartedRef.current = true;
      startChatSession();
    }
  }, [isConnected, state.status, startChatSession]);

  // End chat session on unmount
  useEffect(() => {
    return () => {
      if (state.chatSessionId) {
        sendMessage({
          type: 'chat:session:end',
          session_id: sessionId,
        });
      }
    };
  }, [state.chatSessionId, sessionId, sendMessage]);

  // Handle WebSocket messages
  useEffect(() => {
    const handleWebSocketMessage = (event: MessageEvent) => {
      try {
        // The WebSocket hook forwards the raw event.data, so we need to parse it
        const message = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
        console.log('TerminalChat received message:', message.type, message.data?.session_id);
        
        switch (message.type) {
          case 'chat:session:start':
            if (message.data?.session_id === sessionId) {
              setState(prev => ({
                ...prev,
                status: 'active',
                chatSessionId: message.data.metadata?.chat_session_id,
                messages: [
                  ...prev.messages,
                  {
                    id: `system-start-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
                    type: 'system',
                    content: 'Chat session started. You can now talk to Claude.',
                    timestamp: new Date(),
                  },
                ],
              }));
            }
            break;

          case 'chat:session:end':
            if (message.data?.session_id === sessionId) {
              setState(prev => ({
                ...prev,
                status: 'idle',
                chatSessionId: undefined,
                messages: [
                  ...prev.messages,
                  {
                    id: `system-end-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
                    type: 'system',
                    content: 'Chat session ended.',
                    timestamp: new Date(),
                  },
                ],
              }));
            }
            break;

          case 'chat:message:send':
            if (message.data?.session_id === sessionId && message.data?.metadata?.echo) {
              setState(prev => ({
                ...prev,
                messages: [
                  ...prev.messages,
                  {
                    id: message.data.metadata.message_id || `user-${Date.now()}`,
                    type: 'user',
                    content: message.data.content,
                    timestamp: new Date(message.data.timestamp),
                    metadata: message.data.metadata,
                  },
                ],
              }));
            }
            break;

          case 'chat:message:receive':
            if (message.data?.session_id === sessionId) {
              setState(prev => ({
                ...prev,
                isTyping: false,
                messages: [
                  ...prev.messages,
                  {
                    id: message.data.metadata?.message_id || `claude-${Date.now()}`,
                    type: 'claude',
                    content: message.data.content,
                    timestamp: new Date(message.data.timestamp),
                    metadata: message.data.metadata,
                  },
                ],
              }));
            }
            break;

          case 'chat:typing:start':
            if (message.data?.session_id === sessionId) {
              setState(prev => ({ ...prev, isTyping: true }));
            }
            break;

          case 'chat:typing:stop':
            if (message.data?.session_id === sessionId) {
              setState(prev => ({ ...prev, isTyping: false }));
            }
            break;

          case 'chat:error':
            if (message.data?.session_id === sessionId) {
              setState(prev => ({
                ...prev,
                status: 'error',
                error: message.data.content,
                messages: [
                  ...prev.messages,
                  {
                    id: `error-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
                    type: 'system',
                    content: `Error: ${message.data.content}`,
                    timestamp: new Date(),
                  },
                ],
              }));
            }
            break;
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    // Listen for WebSocket messages
    window.addEventListener('message', handleWebSocketMessage);
    
    return () => {
      window.removeEventListener('message', handleWebSocketMessage);
    };
  }, [sessionId, sendMessage]);

  // Handle sending messages
  const handleSendMessage = useCallback(() => {
    if (!input.trim() || state.status !== 'active') return;

    const message = input.trim();
    setInput('');

    // Send message via WebSocket
    sendMessage({
      type: 'chat:message:send',
      session_id: sessionId,
      content: message,
    });

    // Show typing indicator
    setState(prev => ({ ...prev, isTyping: true }));
  }, [input, state.status, sessionId, sendMessage]);

  // Handle key press
  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !e.shiftKey && !isComposing) {
      e.preventDefault();
      handleSendMessage();
    }
  };

  // Render message with markdown support
  const renderMessage = (message: ChatMessage) => {
    if (message.type === 'system') {
      return (
        <div className="text-gray-500 text-sm italic">{message.content}</div>
      );
    }

    const isUser = message.type === 'user';
    
    return (
      <div className={cn(
        "message",
        isUser ? "message-user" : "message-claude"
      )}>
        <div className="message-header text-xs text-gray-500 mb-1">
          {isUser ? 'You' : 'Claude'} • {new Date(message.timestamp).toLocaleTimeString()}
        </div>
        <div className={cn(
          "message-content prose prose-sm max-w-none",
          isUser ? "prose-invert" : ""
        )}>
          <ReactMarkdown
            components={{
              code({ className, children, ...props }: any) {
                const match = /language-(\w+)/.exec(className || '');
                const inline = !match;
                return !inline ? (
                  <SyntaxHighlighter
                    style={oneDark as any}
                    language={match[1]}
                    PreTag="div"
                    {...props}
                  >
                    {String(children).replace(/\n$/, '')}
                  </SyntaxHighlighter>
                ) : (
                  <code className={className} {...props}>
                    {children}
                  </code>
                );
              },
            }}
          >
            {message.content}
          </ReactMarkdown>
        </div>
      </div>
    );
  };

  return (
    <div className={cn(
      "terminal-chat flex flex-col h-full bg-gray-900 text-gray-100",
      className
    )}>
      {/* Status bar */}
      <div className="terminal-status-bar flex items-center justify-between px-4 py-2 bg-gray-800 border-b border-gray-700">
        <div className="flex items-center space-x-2">
          <div className={cn(
            "w-2 h-2 rounded-full",
            state.isConnected ? "bg-green-500" : "bg-red-500"
          )} />
          <span className="text-sm">
            {state.status === 'active' ? 'Connected to Claude' : 
             state.status === 'starting' ? 'Starting session...' :
             state.status === 'error' ? 'Error' : 'Disconnected'}
          </span>
        </div>
        {state.error && (
          <span className="text-sm text-red-400">{state.error}</span>
        )}
      </div>

      {/* Messages area */}
      <div className="terminal-messages flex-1 overflow-y-auto p-4 space-y-4">
        {state.messages.map((message) => (
          <div key={message.id}>
            {renderMessage(message)}
          </div>
        ))}
        
        {state.isTyping && (
          <div className="message message-claude">
            <div className="message-header text-xs text-gray-500 mb-1">
              Claude is typing...
            </div>
            <div className="typing-indicator flex space-x-1">
              <div className="typing-dot w-2 h-2 bg-gray-400 rounded-full animate-bounce" />
              <div className="typing-dot w-2 h-2 bg-gray-400 rounded-full animate-bounce delay-100" />
              <div className="typing-dot w-2 h-2 bg-gray-400 rounded-full animate-bounce delay-200" />
            </div>
          </div>
        )}
        
        <div ref={messagesEndRef} />
      </div>

      {/* Input area */}
      <div className="terminal-input border-t border-gray-700 p-4">
        <div className="flex items-center space-x-2">
          <span className="text-green-400">❯</span>
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyPress}
            onCompositionStart={() => setIsComposing(true)}
            onCompositionEnd={() => setIsComposing(false)}
            placeholder={state.status === 'active' ? "Type a message..." : "Connecting..."}
            disabled={state.status !== 'active'}
            className={cn(
              "flex-1 bg-transparent outline-none",
              "placeholder-gray-600",
              state.status !== 'active' && "cursor-not-allowed opacity-50"
            )}
          />
        </div>
      </div>
    </div>
  );
};