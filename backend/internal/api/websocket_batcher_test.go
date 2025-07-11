package api

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestEventBatcher(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a mock hub
	hub := &WebSocketHub{
		broadcast: make(chan []byte, 100),
		logger:    logger,
	}

	// Create batcher with short interval for testing
	batcher := NewEventBatcher(hub, logger, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the batcher
	go batcher.Start(ctx)

	// Test 1: Queue multiple events
	t.Run("BatchMultipleEvents", func(t *testing.T) {
		// Queue several events
		for i := 0; i < 5; i++ {
			batcher.QueueEvent("session_update", gin.H{
				"session_id": "test-session-1",
				"update_num": i,
			})
		}

		// Queue different session updates
		for i := 0; i < 3; i++ {
			batcher.QueueEvent("session_update", gin.H{
				"session_id": "test-session-2",
				"update_num": i,
			})
		}

		// Verify events are queued
		batcher.eventsMutex.RLock()
		eventCount := len(batcher.events)
		batcher.eventsMutex.RUnlock()

		// Should have 2 unique events (one per session)
		assert.Equal(t, 2, eventCount, "Should have 2 unique events after deduplication")
	})

	// Test 2: Deduplication
	t.Run("DeduplicationWithinWindow", func(t *testing.T) {
		// Clear events
		batcher.eventsMutex.Lock()
		batcher.events = make(map[string]*BatchedEvent)
		batcher.eventsMutex.Unlock()

		// Queue same event multiple times rapidly
		sessionData := gin.H{"session_id": "dedup-test"}
		for i := 0; i < 10; i++ {
			batcher.QueueEvent("session_update", sessionData)
			time.Sleep(100 * time.Millisecond)
		}

		// Check that we have only one event
		batcher.eventsMutex.RLock()
		eventCount := len(batcher.events)
		if eventCount == 1 {
			for _, event := range batcher.events {
				assert.Equal(t, 10, event.Count, "Event should have count of 10")
			}
		}
		batcher.eventsMutex.RUnlock()

		assert.Equal(t, 1, eventCount, "Should have 1 unique event after rapid queueing")
	})

	// Test 3: Manual flush
	t.Run("ManualFlush", func(t *testing.T) {
		// Clear events and broadcast channel
		batcher.eventsMutex.Lock()
		batcher.events = make(map[string]*BatchedEvent)
		batcher.eventsMutex.Unlock()

		// Drain the broadcast channel
		for len(hub.broadcast) > 0 {
			<-hub.broadcast
		}

		// Queue some events
		batcher.QueueEvent("metrics_update", gin.H{
			"session_id": "flush-test",
			"tokens":     100,
		})

		// Manually flush
		batcher.flushBatch()

		// Check that message was sent to broadcast
		select {
		case msg := <-hub.broadcast:
			assert.NotNil(t, msg, "Should have received a broadcast message")
			t.Logf("Received broadcast message: %s", string(msg))
		case <-time.After(1 * time.Second):
			t.Fatal("No message received on broadcast channel")
		}

		// Verify events are cleared
		batcher.eventsMutex.RLock()
		eventCount := len(batcher.events)
		batcher.eventsMutex.RUnlock()
		assert.Equal(t, 0, eventCount, "Events should be cleared after flush")
	})

	// Test 4: Event key generation
	t.Run("EventKeyGeneration", func(t *testing.T) {
		// Test session update key
		key1 := batcher.generateEventKey("session_update", gin.H{
			"session_id": "test-123",
		})
		assert.Equal(t, "session_update:test-123", key1)

		// Test activity update key
		key2 := batcher.generateEventKey("activity_update", gin.H{
			"activity": map[string]interface{}{
				"activity_type": "token_usage",
				"session_id":    "test-456",
			},
		})
		assert.Equal(t, "activity_update:token_usage:test-456", key2)

		// Test metrics update key
		key3 := batcher.generateEventKey("metrics_update", gin.H{
			"session_id": "test-789",
		})
		assert.Equal(t, "metrics_update:test-789", key3)
	})
}
