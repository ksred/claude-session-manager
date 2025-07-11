package chat

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CLIManager manages Claude CLI processes for chat sessions
type CLIManager struct {
	repository *Repository
	processes  map[string]*CLIProcess
	mutex      sync.RWMutex
	
	// Configuration
	maxProcesses    int
	processTimeout  time.Duration
	inactiveTimeout time.Duration
}

// CLIProcess represents a Claude chat session
type CLIProcess struct {
	ID         string
	SessionID  string
	StartedAt  time.Time
	LastUsed   time.Time
	
	// Communication channels
	InputChan  chan string
	OutputChan chan string
	ErrorChan  chan error
	StopChan   chan struct{}
	
	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	
	// Status
	Status string
	mutex  sync.RWMutex
	
	// Track if this is the first message
	isFirstMessage bool
}

// NewCLIManager creates a new CLI manager
func NewCLIManager(repository *Repository) *CLIManager {
	return &CLIManager{
		repository:      repository,
		processes:       make(map[string]*CLIProcess),
		maxProcesses:    10, // Configurable limit
		processTimeout:  5 * time.Minute,
		inactiveTimeout: 30 * time.Minute,
	}
}

// StartChatSession starts a new Claude CLI process for the given session
func (m *CLIManager) StartChatSession(sessionID string) (*ChatSession, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if we already have an active process for this session
	if existingProcess, exists := m.processes[sessionID]; exists {
		if existingProcess.Status == StatusActive {
			// Update last used time
			existingProcess.mutex.Lock()
			existingProcess.LastUsed = time.Now()
			existingProcess.mutex.Unlock()
			
			// Return existing chat session
			return m.repository.GetChatSessionBySessionID(sessionID)
		}
	}

	// Check process limits
	if len(m.processes) >= m.maxProcesses {
		return nil, fmt.Errorf("maximum number of processes reached: %d", m.maxProcesses)
	}

	// Create new CLI process
	process, err := m.createCLIProcess(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLI process: %w", err)
	}

	// Start the process
	err = m.startProcess(process)
	if err != nil {
		return nil, fmt.Errorf("failed to start CLI process: %w", err)
	}

	// Store the process
	m.processes[sessionID] = process

	// Create chat session in database with PID 0 (since we don't have a real process)
	chatSession, err := m.repository.CreateChatSession(sessionID, 0)
	if err != nil {
		// Cleanup process if database creation fails
		m.stopProcess(process)
		delete(m.processes, sessionID)
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	return chatSession, nil
}

// SendMessage sends a message to the Claude CLI process
func (m *CLIManager) SendMessage(sessionID, message string) error {
	m.mutex.RLock()
	process, exists := m.processes[sessionID]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no active process for session %s", sessionID)
	}

	process.mutex.Lock()
	defer process.mutex.Unlock()

	if process.Status != StatusActive {
		return fmt.Errorf("process is not active for session %s", sessionID)
	}

	// Update last used time
	process.LastUsed = time.Now()

	// Send message to process
	select {
	case process.InputChan <- message:
		return nil
	case <-process.ctx.Done():
		return fmt.Errorf("process context cancelled")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message to process")
	}
}

// StopChatSession stops the Claude CLI process for the given session
func (m *CLIManager) StopChatSession(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	process, exists := m.processes[sessionID]
	if !exists {
		return fmt.Errorf("no process found for session %s", sessionID)
	}

	// Stop the process
	err := m.stopProcess(process)
	delete(m.processes, sessionID)

	// Update database
	chatSession, dbErr := m.repository.GetChatSessionBySessionID(sessionID)
	if dbErr == nil && chatSession != nil {
		m.repository.UpdateChatSessionStatus(chatSession.ID, StatusTerminated)
	}

	return err
}

// GetActiveSessions returns all active chat sessions
func (m *CLIManager) GetActiveSessions() ([]*ChatSession, error) {
	return m.repository.GetActiveChatSessions()
}

// createCLIProcess creates a new CLI process instance
func (m *CLIManager) createCLIProcess(sessionID string) (*CLIProcess, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.processTimeout)

	process := &CLIProcess{
		ID:             fmt.Sprintf("cli-%s-%d", sessionID, time.Now().Unix()),
		SessionID:      sessionID,
		StartedAt:      time.Now(),
		LastUsed:       time.Now(),
		InputChan:      make(chan string, 100),
		OutputChan:     make(chan string, 100),
		ErrorChan:      make(chan error, 50),
		StopChan:       make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		Status:         StatusActive,
		isFirstMessage: true,
	}

	return process, nil
}

// startProcess starts the message handler for this session
func (m *CLIManager) startProcess(process *CLIProcess) error {
	fmt.Printf("[CLI_START] Session %s: Starting Claude chat session\n", process.SessionID)

	// Start message handler goroutine
	go m.handleMessages(process)

	fmt.Printf("[CLI_START] Session %s: Message handler started\n", process.SessionID)

	return nil
}

// handleMessages processes messages for this chat session
func (m *CLIManager) handleMessages(process *CLIProcess) {
	for {
		select {
		case message := <-process.InputChan:
			fmt.Printf("[CLI_MESSAGE] Session %s: Processing message: %s\n", process.SessionID, message)
			
			// Build command
			var cmd *exec.Cmd
			if process.isFirstMessage {
				// First message - start new conversation
				cmd = exec.Command("claude", "--print", message)
				process.isFirstMessage = false
			} else {
				// Continue existing conversation
				cmd = exec.Command("claude", "--print", "-c", message)
			}
			
			// Execute command and get response
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("[CLI_ERROR] Session %s: Command failed: %v\n", process.SessionID, err)
				select {
				case process.ErrorChan <- fmt.Errorf("claude command failed: %w", err):
				default:
				}
				continue
			}
			
			// Send response
			response := strings.TrimSpace(string(output))
			fmt.Printf("[CLI_RESPONSE] Session %s: Got response (%d bytes)\n", process.SessionID, len(response))
			
			select {
			case process.OutputChan <- response:
				fmt.Printf("[CLI_SUCCESS] Session %s: Response sent to channel\n", process.SessionID)
			default:
				fmt.Printf("[CLI_DROPPED] Session %s: Output channel full\n", process.SessionID)
			}
			
		case <-process.StopChan:
			fmt.Printf("[CLI_STOP] Session %s: Stopping message handler\n", process.SessionID)
			return
		case <-process.ctx.Done():
			fmt.Printf("[CLI_TIMEOUT] Session %s: Context cancelled\n", process.SessionID)
			return
		}
	}
}

// stopProcess stops a CLI process and cleans up resources
func (m *CLIManager) stopProcess(process *CLIProcess) error {
	process.mutex.Lock()
	defer process.mutex.Unlock()

	if process.Status == StatusTerminated {
		return nil
	}

	process.Status = StatusTerminated
	process.cancel()

	// Close the stop channel to signal all goroutines
	close(process.StopChan)

	return nil
}

// CleanupInactiveProcesses removes processes that have been inactive
func (m *CLIManager) CleanupInactiveProcesses() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoffTime := time.Now().Add(-m.inactiveTimeout)
	var toDelete []string

	for sessionID, process := range m.processes {
		process.mutex.RLock()
		lastUsed := process.LastUsed
		process.mutex.RUnlock()

		if lastUsed.Before(cutoffTime) {
			toDelete = append(toDelete, sessionID)
		}
	}

	// Stop and remove inactive processes
	for _, sessionID := range toDelete {
		if process, exists := m.processes[sessionID]; exists {
			m.stopProcess(process)
			delete(m.processes, sessionID)
			
			// Update database
			if chatSession, err := m.repository.GetChatSessionBySessionID(sessionID); err == nil && chatSession != nil {
				m.repository.UpdateChatSessionStatus(chatSession.ID, StatusInactive)
			}
		}
	}

	return nil
}

// GetProcessOutput gets output from a specific process
func (m *CLIManager) GetProcessOutput(sessionID string) ([]string, error) {
	m.mutex.RLock()
	process, exists := m.processes[sessionID]
	m.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active process for session %s", sessionID)
	}

	var outputs []string
	
	// Non-blocking read of available outputs
	for {
		select {
		case output := <-process.OutputChan:
			outputs = append(outputs, output)
		default:
			return outputs, nil
		}
	}
}

// GetProcessErrors gets errors from a specific process
func (m *CLIManager) GetProcessErrors(sessionID string) ([]error, error) {
	m.mutex.RLock()
	process, exists := m.processes[sessionID]
	m.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no active process for session %s", sessionID)
	}

	var errors []error
	
	// Non-blocking read of available errors
	for {
		select {
		case err := <-process.ErrorChan:
			errors = append(errors, err)
		default:
			return errors, nil
		}
	}
}