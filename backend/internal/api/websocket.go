package api

import (
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
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	logger     *logrus.Logger
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

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
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
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
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
	message := gin.H{
		"type":      updateType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	// Convert to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	h.broadcast <- jsonData
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
				c.Logger.WithField("subscription", msg).Info("Client subscribed to updates")
				// Send acknowledgment
				ack := gin.H{"type": "subscribed", "timestamp": time.Now().Unix()}
				if ackData, err := json.Marshal(ack); err == nil {
					c.Send <- ackData
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
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

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