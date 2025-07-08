package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchEvent represents a file system event for a session
type WatchEvent struct {
	Type      string    `json:"type"`      // created, modified, deleted
	SessionID string    `json:"session_id"`
	Session   *Session  `json:"session,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionWatcher watches for changes in Claude session files
type SessionWatcher struct {
	watcher      *fsnotify.Watcher
	callback     func([]Session)
	eventCallback func(WatchEvent)
	stopChan     chan struct{}
}

// NewSessionWatcher creates a new file system watcher for Claude sessions
func NewSessionWatcher(callback func([]Session)) (*SessionWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	
	sw := &SessionWatcher{
		watcher:  watcher,
		callback: callback,
		stopChan: make(chan struct{}),
	}
	
	return sw, nil
}

// SetEventCallback sets a callback for individual file events
func (sw *SessionWatcher) SetEventCallback(callback func(WatchEvent)) {
	sw.eventCallback = callback
}

// Start begins watching for session file changes
func (sw *SessionWatcher) Start() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	claudeDir := filepath.Join(homeDir, ".claude", "projects")
	
	// Add all directories in the projects folder to the watcher
	err = filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip paths we can't access
		}
		
		if info.IsDir() {
			return sw.watcher.Add(path)
		}
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to setup directory watching: %w", err)
	}
	
	// Start the event loop
	go sw.eventLoop()
	
	return nil
}

// Stop stops the file watcher
func (sw *SessionWatcher) Stop() error {
	close(sw.stopChan)
	return sw.watcher.Close()
}

// eventLoop processes file system events
func (sw *SessionWatcher) eventLoop() {
	// Debounce timer to avoid excessive updates
	var debounceTimer *time.Timer
	debounceDelay := 500 * time.Millisecond
	
	for {
		select {
		case <-sw.stopChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
			
		case event, ok := <-sw.watcher.Events:
			if !ok {
				return
			}
			
			// Watch new directories
			if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					sw.watcher.Add(event.Name)
				}
			}
			
			// Only care about .jsonl files that aren't summaries
			if strings.HasSuffix(event.Name, ".jsonl") && !strings.Contains(event.Name, "summary") {
				// Extract session ID from filename
				sessionID := strings.TrimSuffix(filepath.Base(event.Name), ".jsonl")
				
				// Determine event type
				eventType := ""
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					eventType = "created"
				case event.Op&fsnotify.Write == fsnotify.Write:
					eventType = "modified"
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					eventType = "deleted"
				}
				
				if eventType != "" {
					// Send individual event if callback is set
					if sw.eventCallback != nil {
						watchEvent := WatchEvent{
							Type:      eventType,
							SessionID: sessionID,
							Timestamp: time.Now(),
						}
						
						// Parse session for created/modified events
						if eventType != "deleted" {
							if session, err := ParseSessionFile(event.Name); err == nil {
								watchEvent.Session = &session
							}
						}
						
						sw.eventCallback(watchEvent)
					}
					
					// Debounce full session discovery
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(debounceDelay, func() {
						sessions, err := DiscoverSessions()
						if err == nil && sw.callback != nil {
							sw.callback(sessions)
						}
					})
				}
			}
			
		case err, ok := <-sw.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			fmt.Printf("File watcher error: %v\n", err)
		}
	}
}