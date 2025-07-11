package api

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// EventBatcher collects and batches WebSocket events before sending
type EventBatcher struct {
	hub            *WebSocketHub
	logger         *logrus.Logger
	events         map[string]*BatchedEvent // Key is generated from event type and data
	eventsMutex    sync.RWMutex
	batchInterval  time.Duration
	maxBatchSize   int
	dedupKeys      map[string]time.Time // Track when we last sent each unique event
	dedupKeysMutex sync.RWMutex
	dedupWindow    time.Duration // How long to remember sent events for deduplication
}

// BatchedEvent represents an event that can be batched
type BatchedEvent struct {
	Type      string
	Data      interface{}
	Count     int       // Number of times this event was triggered
	FirstSeen time.Time // When we first saw this event
	LastSeen  time.Time // When we last saw this event
}

// BatchedMessage represents the batched message sent to clients
type BatchedMessage struct {
	Type      string                 `json:"type"`
	Events    []BatchedEventPayload  `json:"events"`
	Timestamp int64                  `json:"timestamp"`
	BatchInfo map[string]interface{} `json:"batch_info"`
}

// BatchedEventPayload represents individual events in a batch
type BatchedEventPayload struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Count     int         `json:"count"`
	FirstSeen int64       `json:"first_seen"`
	LastSeen  int64       `json:"last_seen"`
}

// NewEventBatcher creates a new event batcher
func NewEventBatcher(hub *WebSocketHub, logger *logrus.Logger, batchInterval time.Duration) *EventBatcher {
	if batchInterval < 10*time.Second {
		batchInterval = 10 * time.Second
	}
	if batchInterval > 30*time.Second {
		batchInterval = 30 * time.Second
	}

	return &EventBatcher{
		hub:           hub,
		logger:        logger,
		events:        make(map[string]*BatchedEvent),
		batchInterval: batchInterval,
		maxBatchSize:  100, // Maximum events in a single batch
		dedupKeys:     make(map[string]time.Time),
		dedupWindow:   5 * time.Minute, // Remember events for 5 minutes
	}
}

// Start begins the batching process
func (b *EventBatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(b.batchInterval)
	defer ticker.Stop()

	// Cleanup old dedup keys periodically
	cleanupTicker := time.NewTicker(1 * time.Minute)
	defer cleanupTicker.Stop()

	b.logger.WithField("batch_interval", b.batchInterval).Info("Event batcher started")

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("Event batcher stopped")
			return
		case <-ticker.C:
			b.flushBatch()
		case <-cleanupTicker.C:
			b.cleanupDedupKeys()
		}
	}
}

// QueueEvent adds an event to the batch
func (b *EventBatcher) QueueEvent(eventType string, data interface{}) {
	// Generate a unique key for this event
	key := b.generateEventKey(eventType, data)

	// Check if we've sent this event recently (deduplication)
	b.dedupKeysMutex.RLock()
	lastSent, exists := b.dedupKeys[key]
	b.dedupKeysMutex.RUnlock()

	now := time.Now()

	// Skip if we sent this exact event in the last 5 seconds
	if exists && now.Sub(lastSent) < 5*time.Second {
		b.logger.WithFields(logrus.Fields{
			"event_type":    eventType,
			"key":           key,
			"last_sent_ago": now.Sub(lastSent),
		}).Debug("Skipping duplicate event")
		return
	}

	// Add or update the event in the batch
	b.eventsMutex.Lock()
	defer b.eventsMutex.Unlock()

	if existingEvent, ok := b.events[key]; ok {
		// Update existing event
		existingEvent.Count++
		existingEvent.LastSeen = now
		b.logger.WithFields(logrus.Fields{
			"event_type": eventType,
			"count":      existingEvent.Count,
		}).Debug("Updated batched event count")
	} else {
		// Add new event
		b.events[key] = &BatchedEvent{
			Type:      eventType,
			Data:      data,
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
		b.logger.WithFields(logrus.Fields{
			"event_type": eventType,
			"batch_size": len(b.events),
		}).Debug("Added new event to batch")
	}

	// Check if we should flush early due to batch size
	if len(b.events) >= b.maxBatchSize {
		b.logger.WithField("batch_size", len(b.events)).Info("Flushing batch early due to size limit")
		go b.flushBatch()
	}
}

// flushBatch sends all batched events
func (b *EventBatcher) flushBatch() {
	b.eventsMutex.Lock()

	if len(b.events) == 0 {
		b.eventsMutex.Unlock()
		return
	}

	// Copy events to send
	eventsToSend := make(map[string]*BatchedEvent)
	for k, v := range b.events {
		eventsToSend[k] = v
	}

	// Clear the events map
	b.events = make(map[string]*BatchedEvent)
	b.eventsMutex.Unlock()

	// Convert to payload format
	eventPayloads := make([]BatchedEventPayload, 0, len(eventsToSend))
	totalCount := 0

	for _, event := range eventsToSend {
		eventPayloads = append(eventPayloads, BatchedEventPayload{
			Type:      event.Type,
			Data:      event.Data,
			Count:     event.Count,
			FirstSeen: event.FirstSeen.Unix(),
			LastSeen:  event.LastSeen.Unix(),
		})
		totalCount += event.Count
	}

	// Create batched message
	batchedMsg := BatchedMessage{
		Type:      "batched_updates",
		Events:    eventPayloads,
		Timestamp: time.Now().Unix(),
		BatchInfo: gin.H{
			"event_count":       len(eventPayloads),
			"total_occurrences": totalCount,
			"batch_interval":    b.batchInterval.Seconds(),
		},
	}

	b.logger.WithFields(logrus.Fields{
		"event_count":       len(eventPayloads),
		"total_occurrences": totalCount,
	}).Info("Flushing event batch")

	// Send the batched message
	if b.hub != nil {
		// Convert to JSON
		jsonData, err := json.Marshal(batchedMsg)
		if err != nil {
			b.logger.WithError(err).Error("Failed to marshal batched message")
			return
		}

		// Send directly to broadcast channel
		b.hub.broadcast <- jsonData
	}

	// Update dedup keys
	b.dedupKeysMutex.Lock()
	now := time.Now()
	for key := range eventsToSend {
		b.dedupKeys[key] = now
	}
	b.dedupKeysMutex.Unlock()
}

// generateEventKey creates a unique key for an event based on its type and data
func (b *EventBatcher) generateEventKey(eventType string, data interface{}) string {
	// For different event types, generate appropriate keys
	switch eventType {
	case "session_update", "session_new":
		// For session events, key by session ID
		if m, ok := data.(gin.H); ok {
			if sessionID, ok := m["session_id"].(string); ok {
				return eventType + ":" + sessionID
			}
		}
	case "activity_update":
		// For activity events, key by activity type and session
		if m, ok := data.(gin.H); ok {
			if activity, ok := m["activity"].(map[string]interface{}); ok {
				activityType := ""
				sessionID := ""

				if at, ok := activity["activity_type"].(string); ok {
					activityType = at
				}
				if sid, ok := activity["session_id"].(string); ok {
					sessionID = sid
				}

				return eventType + ":" + activityType + ":" + sessionID
			}
		}
	case "metrics_update":
		// For metrics events, key by session ID
		if m, ok := data.(gin.H); ok {
			if sessionID, ok := m["session_id"].(string); ok {
				return eventType + ":" + sessionID
			}
		}
	}

	// Default: use event type and current timestamp (no deduplication)
	return eventType + ":" + time.Now().Format("20060102150405.000")
}

// cleanupDedupKeys removes old deduplication keys
func (b *EventBatcher) cleanupDedupKeys() {
	b.dedupKeysMutex.Lock()
	defer b.dedupKeysMutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-b.dedupWindow)

	oldCount := len(b.dedupKeys)
	for key, timestamp := range b.dedupKeys {
		if timestamp.Before(cutoff) {
			delete(b.dedupKeys, key)
		}
	}

	if removed := oldCount - len(b.dedupKeys); removed > 0 {
		b.logger.WithFields(logrus.Fields{
			"removed":   removed,
			"remaining": len(b.dedupKeys),
		}).Debug("Cleaned up old deduplication keys")
	}
}

// DirectBroadcast sends an event immediately without batching
// Use this for critical events that need immediate delivery
func (b *EventBatcher) DirectBroadcast(eventType string, data interface{}) {
	if b.hub != nil {
		b.hub.BroadcastUpdate(eventType, data)
	}
}
