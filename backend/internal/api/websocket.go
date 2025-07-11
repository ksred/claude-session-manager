package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// TODO: Restrict origins in production
		return true
	},
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID     string
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *WebSocketHub
	Logger *logrus.Logger
}

// WebSocketHub maintains active WebSocket connections
type WebSocketHub struct {
	clients     map[*WebSocketClient]bool
	broadcast   chan []byte
	register    chan *WebSocketClient
	unregister  chan *WebSocketClient
	logger      *logrus.Logger
	ChatHandler ChatMessageHandler
	batcher     *EventBatcher
}

// ChatMessageHandler interface for handling chat messages
type ChatMessageHandler interface {
	HandleMessage(clientID string, msgType string, msg map[string]interface{}, broadcastFn func(string, interface{})) error
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(logger *logrus.Logger) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		logger:     logger,
	}
}

// SetBatcher sets the event batcher for the hub
func (h *WebSocketHub) SetBatcher(batcher *EventBatcher) {
	h.batcher = batcher
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run(ctx context.Context) {
	h.logger.Info("WebSocket hub Run() started")
	for {
		select {
		case <-ctx.Done():
			// Context cancelled, clean up and exit
			h.logger.Info("WebSocket hub received context cancellation")
			// Close all client connections
			h.logger.WithField("client_count", len(h.clients)).Info("Closing all WebSocket client connections")
			for client := range h.clients {
				close(client.Send)
				delete(h.clients, client)
			}
			h.logger.Info("WebSocket hub Run() exiting")
			return

		case client := <-h.register:
			h.clients[client] = true
			h.logger.WithField("client_id", client.ID).Info("WebSocket client connected")

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.logger.WithField("client_id", client.ID).Info("WebSocket client disconnected")
			}

		case message := <-h.broadcast:
			h.logger.WithFields(logrus.Fields{
				"message_size": len(message),
				"clients":      len(h.clients),
			}).Debug("Hub received message to broadcast")

			sentCount := 0
			failedCount := 0
			for client := range h.clients {
				select {
				case client.Send <- message:
					sentCount++
					h.logger.WithFields(logrus.Fields{
						"client_id": client.ID,
					}).Debug("Message queued for client")
				default:
					failedCount++
					h.logger.WithFields(logrus.Fields{
						"client_id": client.ID,
					}).Debug("Failed to send to client (buffer full), closing connection")
					close(client.Send)
					delete(h.clients, client)
				}
			}

			h.logger.WithFields(logrus.Fields{
				"sent_count":    sentCount,
				"failed_count":  failedCount,
				"total_clients": len(h.clients),
			}).Debug("Finished broadcasting message")
		}
	}
}

// BroadcastUpdate sends an update to all connected clients
// Supported update types:
// - "sessions_updated": Full sessions list has been updated
// - "session_new": A new session was created
// - "session_update": An existing session was modified
// - "session_deleted": A session was deleted
func (h *WebSocketHub) BroadcastUpdate(updateType string, data interface{}) {
	// Check if we should batch this event
	shouldBatch := h.shouldBatchEvent(updateType)

	if shouldBatch && h.batcher != nil {
		// Queue the event for batching
		h.logger.WithFields(logrus.Fields{
			"update_type": updateType,
		}).Debug("Queueing event for batch")
		h.batcher.QueueEvent(updateType, data)
		return
	}

	// Send immediately (no batching)
	message := gin.H{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	// Log the update being broadcast
	h.logger.WithFields(logrus.Fields{
		"update_type":  updateType,
		"client_count": len(h.clients),
		"timestamp":    message["timestamp"],
		"batched":      false,
	}).Debug("Broadcasting WebSocket update to frontend")

	// Convert to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	// Log the message size
	h.logger.WithFields(logrus.Fields{
		"update_type":  updateType,
		"message_size": len(jsonData),
		"client_count": len(h.clients),
	}).Debug("Sending WebSocket message to broadcast channel")

	h.broadcast <- jsonData
}

// shouldBatchEvent determines if an event type should be batched
func (h *WebSocketHub) shouldBatchEvent(eventType string) bool {
	// Batch these high-frequency events
	switch eventType {
	case "session_update", "activity_update", "metrics_update":
		return true
	// Don't batch these important events
	case "session_new", "session_deleted", "sessions_updated":
		return false
	// Chat events should not be batched for real-time experience
	case "chat:message:recv", "chat:message:send", "chat:error":
		return false
	default:
		// Default to batching unknown events
		return true
	}
}

// websocketHandler handles WebSocket connections
// @Summary WebSocket endpoint
// @Description Establish a WebSocket connection for real-time session updates
// @Tags WebSocket
// @Accept json
// @Produce json
// @Success 101 {object} WebSocketMessage "WebSocket connection established"
// @Failure 400 {object} ErrorResponse "Failed to upgrade connection"
// @Router /ws [get]
func (s *Server) websocketHandler(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upgrade to WebSocket")
		return
	}

	// Create new client
	client := &WebSocketClient{
		ID:     generateClientID(),
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    s.wsHub, // Assuming wsHub is added to Server struct
		Logger: s.logger,
	}

	// Register client
	client.Hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from the WebSocket client
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Logger.WithError(err).Error("WebSocket error")
			}
			break
		}

		// Handle incoming message
		c.Logger.WithField("message", string(message)).Debug("Received WebSocket message")

		// Process incoming messages
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			c.Logger.WithError(err).Error("Failed to parse WebSocket message")
			continue
		}

		// Handle different message types
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "ping":
				// Respond with pong
				pong := gin.H{"type": "pong", "timestamp": time.Now().Unix()}
				if pongData, err := json.Marshal(pong); err == nil {
					c.Send <- pongData
				}
			case "subscribe":
				// Handle subscription requests
				c.Logger.WithFields(logrus.Fields{
					"client_id":    c.ID,
					"subscription": msg,
				}).Info("Client subscribed to WebSocket updates")
				// Send acknowledgment
				ack := gin.H{"type": "subscribed", "timestamp": time.Now().Unix()}
				if ackData, err := json.Marshal(ack); err == nil {
					c.Logger.WithField("client_id", c.ID).Debug("Sending subscription acknowledgment")
					c.Send <- ackData
				}
			case "chat:session:start", "chat:session:end", "chat:message:send", "chat:typing:start", "chat:typing:stop":
				// Handle chat messages through the chat handler
				if c.Hub.ChatHandler != nil {
					err := c.Hub.ChatHandler.HandleMessage(c.ID, msgType, msg, c.Hub.BroadcastUpdate)
					if err != nil {
						c.Logger.WithError(err).WithFields(logrus.Fields{
							"client_id":    c.ID,
							"message_type": msgType,
						}).Error("Failed to handle chat message")
					}
				} else {
					c.Logger.WithField("type", msgType).Warn("Received chat message but no chat handler configured")
				}
			default:
				c.Logger.WithField("type", msgType).Debug("Received unknown message type")
			}
		}
	}
}

// writePump handles sending messages to the WebSocket client
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Logger.WithField("client_id", c.ID).Debug("Send channel closed, sending close message")
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.Logger.WithFields(logrus.Fields{
				"client_id":    c.ID,
				"message_size": len(message),
			}).Debug("Writing message to WebSocket client")

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.Logger.WithError(err).WithField("client_id", c.ID).Debug("Failed to get writer")
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			if n > 0 {
				c.Logger.WithFields(logrus.Fields{
					"client_id":        c.ID,
					"batched_messages": n,
				}).Debug("Batching additional messages")
			}
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				c.Logger.WithError(err).WithField("client_id", c.ID).Debug("Failed to close writer")
				return
			}

			c.Logger.WithField("client_id", c.ID).Debug("Successfully sent message to client")

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}
