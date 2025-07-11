package chat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
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

// CLIProcess represents a running Claude CLI process
type CLIProcess struct {
	ID         string
	SessionID  string
	Cmd        *exec.Cmd
	Stdin      io.WriteCloser
	Stdout     io.ReadCloser
	Stderr     io.ReadCloser
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

	// Create chat session in database
	chatSession, err := m.repository.CreateChatSession(sessionID, process.Cmd.Process.Pid)
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
	
	// Create the Claude CLI command
	// Note: Claude starts in interactive mode by default
	cmd := exec.CommandContext(ctx, "claude")
	
	// Set up pipes for communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		cancel()
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		cancel()
		return nil, err
	}

	process := &CLIProcess{
		ID:         fmt.Sprintf("cli-%s-%d", sessionID, time.Now().Unix()),
		SessionID:  sessionID,
		Cmd:        cmd,
		Stdin:      stdin,
		Stdout:     stdout,
		Stderr:     stderr,
		StartedAt:  time.Now(),
		LastUsed:   time.Now(),
		InputChan:  make(chan string, 10),
		OutputChan: make(chan string, 10),
		ErrorChan:  make(chan error, 10),
		StopChan:   make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
		Status:     StatusActive,
	}

	return process, nil
}

// startProcess starts the CLI process and sets up communication goroutines
func (m *CLIManager) startProcess(process *CLIProcess) error {
	// Start the command
	err := process.Cmd.Start()
	if err != nil {
		process.cancel()
		return err
	}

	// Start input handler goroutine
	go m.handleProcessInput(process)

	// Start output handler goroutine
	go m.handleProcessOutput(process)

	// Start error handler goroutine
	go m.handleProcessError(process)

	// Start process monitor goroutine
	go m.monitorProcess(process)

	return nil
}

// handleProcessInput handles input to the CLI process
func (m *CLIManager) handleProcessInput(process *CLIProcess) {
	defer process.Stdin.Close()

	for {
		select {
		case message := <-process.InputChan:
			_, err := fmt.Fprintf(process.Stdin, "%s\n", message)
			if err != nil {
				select {
				case process.ErrorChan <- fmt.Errorf("failed to write to stdin: %w", err):
				default:
				}
				return
			}
		case <-process.StopChan:
			return
		case <-process.ctx.Done():
			return
		}
	}
}

// handleProcessOutput handles output from the CLI process
func (m *CLIManager) handleProcessOutput(process *CLIProcess) {
	scanner := bufio.NewScanner(process.Stdout)
	
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case process.OutputChan <- line:
		case <-process.StopChan:
			return
		case <-process.ctx.Done():
			return
		default:
			// Drop message if channel is full
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case process.ErrorChan <- fmt.Errorf("error reading stdout: %w", err):
		default:
		}
	}
}

// handleProcessError handles error output from the CLI process
func (m *CLIManager) handleProcessError(process *CLIProcess) {
	scanner := bufio.NewScanner(process.Stderr)
	
	for scanner.Scan() {
		line := scanner.Text()
		select {
		case process.ErrorChan <- fmt.Errorf("CLI error: %s", line):
		case <-process.StopChan:
			return
		case <-process.ctx.Done():
			return
		default:
			// Drop error if channel is full
		}
	}
}

// monitorProcess monitors the process and handles cleanup
func (m *CLIManager) monitorProcess(process *CLIProcess) {
	defer func() {
		process.mutex.Lock()
		process.Status = StatusTerminated
		process.mutex.Unlock()
	}()

	// Wait for process to complete or context to be cancelled
	done := make(chan error)
	go func() {
		done <- process.Cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			select {
			case process.ErrorChan <- fmt.Errorf("process exited with error: %w", err):
			default:
			}
		}
	case <-process.ctx.Done():
		// Context cancelled, kill the process
		if process.Cmd.Process != nil {
			process.Cmd.Process.Signal(syscall.SIGTERM)
			time.Sleep(1 * time.Second)
			process.Cmd.Process.Kill()
		}
	}

	// Signal stop to all goroutines
	close(process.StopChan)
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

	// Give the process a moment to terminate gracefully
	time.Sleep(100 * time.Millisecond)

	// Force kill if still running
	if process.Cmd.Process != nil {
		return process.Cmd.Process.Kill()
	}

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