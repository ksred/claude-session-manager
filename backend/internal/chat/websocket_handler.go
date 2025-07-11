package chat

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// WebSocketChatHandler handles chat-specific WebSocket messages
type WebSocketChatHandler struct {
	cliManager *CLIManager
	repository *Repository
	logger     *logrus.Logger
}

// NewWebSocketChatHandler creates a new chat WebSocket handler
func NewWebSocketChatHandler(cliManager *CLIManager, repository *Repository, logger *logrus.Logger) *WebSocketChatHandler {
	return &WebSocketChatHandler{
		cliManager: cliManager,
		repository: repository,
		logger:     logger,
	}
}

// HandleMessage processes incoming chat WebSocket messages
func (h *WebSocketChatHandler) HandleMessage(clientID string, msgType string, msg map[string]interface{}, broadcastFn func(string, interface{})) error {
	switch msgType {
	case WSMsgChatSessionStart:
		return h.handleSessionStart(clientID, msg, broadcastFn)
	case WSMsgChatSessionEnd:
		return h.handleSessionEnd(clientID, msg, broadcastFn)
	case WSMsgChatMessageSend:
		return h.handleMessageSend(clientID, msg, broadcastFn)
	case WSMsgChatTypingStart:
		return h.handleTypingStart(clientID, msg, broadcastFn)
	case WSMsgChatTypingStop:
		return h.handleTypingStop(clientID, msg, broadcastFn)
	default:
		return fmt.Errorf("unknown chat message type: %s", msgType)
	}
}

// handleSessionStart handles starting a new chat session
func (h *WebSocketChatHandler) handleSessionStart(clientID string, msg map[string]interface{}, broadcastFn func(string, interface{})) error {
	sessionID, ok := msg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id in chat session start message")
	}

	h.logger.WithFields(logrus.Fields{
		"client_id":  clientID,
		"session_id": sessionID,
	}).Info("Starting chat session")

	// Start the CLI process for this session
	chatSession, err := h.cliManager.StartChatSession(sessionID)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"client_id":  clientID,
			"session_id": sessionID,
		}).Error("Failed to start chat session")

		// Send error to client
		errorMsg := WebSocketMessage{
			Type:      WSMsgChatError,
			SessionID: sessionID,
			Content:   fmt.Sprintf("Failed to start chat session: %v", err),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error":     true,
				"client_id": clientID,
			},
		}
		broadcastFn(WSMsgChatError, errorMsg)
		return err
	}

	// Start monitoring the CLI process for output
	go h.monitorCLIOutput(sessionID, broadcastFn)
	
	h.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"process_id": chatSession.ProcessID,
	}).Info("CLI monitoring started")

	// Send session started confirmation
	startMsg := WebSocketMessage{
		Type:      WSMsgChatSessionStart,
		SessionID: sessionID,
		Content:   "Chat session started successfully",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"chat_session_id": chatSession.ID,
			"process_id":      chatSession.ProcessID,
			"client_id":       clientID,
		},
	}

	broadcastFn(WSMsgChatSessionStart, startMsg)
	return nil
}

// handleSessionEnd handles ending a chat session
func (h *WebSocketChatHandler) handleSessionEnd(clientID string, msg map[string]interface{}, broadcastFn func(string, interface{})) error {
	sessionID, ok := msg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id in chat session end message")
	}

	h.logger.WithFields(logrus.Fields{
		"client_id":  clientID,
		"session_id": sessionID,
	}).Info("Ending chat session")

	// Stop the CLI process
	err := h.cliManager.StopChatSession(sessionID)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"client_id":  clientID,
			"session_id": sessionID,
		}).Error("Failed to stop chat session")

		// Send error to client
		errorMsg := WebSocketMessage{
			Type:      WSMsgChatError,
			SessionID: sessionID,
			Content:   fmt.Sprintf("Failed to stop chat session: %v", err),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error":     true,
				"client_id": clientID,
			},
		}
		broadcastFn(WSMsgChatError, errorMsg)
		return err
	}

	// Send session ended confirmation
	endMsg := WebSocketMessage{
		Type:      WSMsgChatSessionEnd,
		SessionID: sessionID,
		Content:   "Chat session ended successfully",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"client_id": clientID,
		},
	}

	broadcastFn(WSMsgChatSessionEnd, endMsg)
	return nil
}

// handleMessageSend handles sending a message to Claude CLI
func (h *WebSocketChatHandler) handleMessageSend(clientID string, msg map[string]interface{}, broadcastFn func(string, interface{})) error {
	sessionID, ok := msg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id in message send")
	}

	content, ok := msg["content"].(string)
	if !ok {
		return fmt.Errorf("missing content in message send")
	}

	h.logger.WithFields(logrus.Fields{
		"client_id":  clientID,
		"session_id": sessionID,
		"content":    content[:min(len(content), 100)], // Log first 100 chars
		"full_len":   len(content),
	}).Info("Sending message to Claude CLI")

	// Get the chat session
	chatSession, err := h.repository.GetChatSessionBySessionID(sessionID)
	if err != nil || chatSession == nil {
		h.logger.WithError(err).Error("No active chat session found")
		return fmt.Errorf("no active chat session found for session %s", sessionID)
	}

	h.logger.WithFields(logrus.Fields{
		"chat_session_id": chatSession.ID,
		"process_id":      chatSession.ProcessID,
		"status":          chatSession.Status,
	}).Debug("Found chat session")

	// Store the user message in database
	userMessage, err := h.repository.CreateChatMessage(chatSession.ID, MessageTypeUser, content, map[string]interface{}{
		"client_id": clientID,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to store user message")
		// Continue processing even if storage fails
	}

	h.logger.Info("About to send message to CLI process via CLIManager")
	
	// Send message to CLI process
	err = h.cliManager.SendMessage(sessionID, content)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"client_id":  clientID,
			"session_id": sessionID,
		}).Error("Failed to send message to CLI")

		// Send error to client
		errorMsg := WebSocketMessage{
			Type:      WSMsgChatError,
			SessionID: sessionID,
			Content:   fmt.Sprintf("Failed to send message: %v", err),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error":     true,
				"client_id": clientID,
			},
		}
		broadcastFn(WSMsgChatError, errorMsg)
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"content":    content[:min(len(content), 50)],
	}).Info("Successfully sent message to CLI process")

	// Update session activity
	h.repository.UpdateChatSessionActivity(chatSession.ID)

	// Echo the user message back to confirm it was sent
	echoMsg := WebSocketMessage{
		Type:      WSMsgChatMessageSend,
		SessionID: sessionID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"message_id":      userMessage.ID,
			"message_type":    MessageTypeUser,
			"client_id":       clientID,
			"echo":            true,
		},
	}

	broadcastFn(WSMsgChatMessageSend, echoMsg)
	return nil
}

// handleTypingStart handles typing indicators
func (h *WebSocketChatHandler) handleTypingStart(clientID string, msg map[string]interface{}, broadcastFn func(string, interface{})) error {
	sessionID, ok := msg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id in typing start")
	}

	typingMsg := WebSocketMessage{
		Type:      WSMsgChatTypingStart,
		SessionID: sessionID,
		Content:   "",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"client_id": clientID,
		},
	}

	broadcastFn(WSMsgChatTypingStart, typingMsg)
	return nil
}

// handleTypingStop handles stopping typing indicators
func (h *WebSocketChatHandler) handleTypingStop(clientID string, msg map[string]interface{}, broadcastFn func(string, interface{})) error {
	sessionID, ok := msg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id in typing stop")
	}

	typingMsg := WebSocketMessage{
		Type:      WSMsgChatTypingStop,
		SessionID: sessionID,
		Content:   "",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"client_id": clientID,
		},
	}

	broadcastFn(WSMsgChatTypingStop, typingMsg)
	return nil
}

// monitorCLIOutput monitors CLI output for a session and broadcasts responses
func (h *WebSocketChatHandler) monitorCLIOutput(sessionID string, broadcastFn func(string, interface{})) {
	h.logger.WithField("session_id", sessionID).Info("Starting CLI output monitoring")

	ticker := time.NewTicker(100 * time.Millisecond) // Check for output every 100ms
	defer ticker.Stop()

	checkCount := 0
	for {
		select {
		case <-ticker.C:
			checkCount++
			if checkCount%50 == 0 { // Log every 5 seconds
				h.logger.WithFields(logrus.Fields{
					"session_id": sessionID,
					"checks":     checkCount,
				}).Debug("Still monitoring CLI output")
			}
			
			// Check for output from CLI process
			outputs, err := h.cliManager.GetProcessOutput(sessionID)
			if err != nil {
				h.logger.WithError(err).WithField("session_id", sessionID).Debug("Error getting process output")
				continue
			}

			// Process each output line
			if len(outputs) > 0 {
				h.logger.WithFields(logrus.Fields{
					"session_id": sessionID,
					"count":      len(outputs),
				}).Info("Got outputs from CLI process")
			}
			
			for _, output := range outputs {
				if output == "" {
					continue
				}

				h.logger.WithFields(logrus.Fields{
					"session_id": sessionID,
					"output":     output[:min(len(output), 100)],
					"full_len":   len(output),
				}).Info("Received CLI output")

				// Get chat session for storing message
				chatSession, err := h.repository.GetChatSessionBySessionID(sessionID)
				if err != nil {
					h.logger.WithError(err).Error("Failed to get chat session for output storage")
					continue
				}

				// Store Claude's response in database
				claudeMessage, err := h.repository.CreateChatMessage(chatSession.ID, MessageTypeClaude, output, map[string]interface{}{
					"source": "cli_output",
				})
				if err != nil {
					h.logger.WithError(err).Error("Failed to store Claude message")
					// Continue processing even if storage fails
				}

				// Broadcast the response
				responseMsg := WebSocketMessage{
					Type:      WSMsgChatMessageRecv,
					SessionID: sessionID,
					Content:   output,
					Timestamp: time.Now(),
					Metadata: map[string]interface{}{
						"message_id":   claudeMessage.ID,
						"message_type": MessageTypeClaude,
						"source":       "cli_output",
					},
				}

				broadcastFn(WSMsgChatMessageRecv, responseMsg)
			}

			// Check for errors from CLI process
			errors, err := h.cliManager.GetProcessErrors(sessionID)
			if err != nil {
				h.logger.WithError(err).WithField("session_id", sessionID).Debug("Error getting process errors")
				continue
			}

			// Process each error
			for _, processErr := range errors {
				h.logger.WithFields(logrus.Fields{
					"session_id": sessionID,
					"error":      processErr.Error(),
				}).Warn("CLI process error")

				// Broadcast the error
				errorMsg := WebSocketMessage{
					Type:      WSMsgChatError,
					SessionID: sessionID,
					Content:   processErr.Error(),
					Timestamp: time.Now(),
					Metadata: map[string]interface{}{
						"error":  true,
						"source": "cli_process",
					},
				}

				broadcastFn(WSMsgChatError, errorMsg)
			}

			// Check if session is still active
			chatSession, err := h.repository.GetChatSessionBySessionID(sessionID)
			if err != nil || chatSession == nil || chatSession.Status != StatusActive {
				h.logger.WithField("session_id", sessionID).Info("Chat session no longer active, stopping monitoring")
				return
			}
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}