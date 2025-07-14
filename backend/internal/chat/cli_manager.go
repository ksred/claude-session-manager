package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SessionData represents session data needed by CLI manager
type SessionData struct {
	ID          string `json:"id"`
	ProjectPath string `json:"project_path"`
	ProjectName string `json:"project_name"`
}

// SessionRepository interface for accessing session data
type SessionRepository interface {
	GetSessionByID(sessionID string) (*SessionData, error)
}

// ClaudeResponse represents the JSON response from Claude CLI
type ClaudeResponse struct {
	Type        string  `json:"type"`
	Subtype     string  `json:"subtype"`
	IsError     bool    `json:"is_error"`
	Result      string  `json:"result"`
	SessionID   string  `json:"session_id"`
	Error       string  `json:"error,omitempty"`
	DurationMs  int     `json:"duration_ms,omitempty"`
	NumTurns    int     `json:"num_turns,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
}

// CLIManager manages Claude CLI processes for chat sessions
type CLIManager struct {
	repository        *Repository
	sessionRepository SessionRepository
	processes         map[string]*CLIProcess
	mutex             sync.RWMutex
	
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
	
	// Store the Claude session ID for conversation continuity
	claudeSessionID string
	
	// Store the project directory for setting working directory
	projectPath string
}

// NewCLIManager creates a new CLI manager
func NewCLIManager(repository *Repository, sessionRepository SessionRepository) *CLIManager {
	return &CLIManager{
		repository:        repository,
		sessionRepository: sessionRepository,
		processes:         make(map[string]*CLIProcess),
		maxProcesses:      10, // Configurable limit
		processTimeout:    5 * time.Minute,
		inactiveTimeout:   30 * time.Minute,
	}
}

// StartChatSession starts a new Claude CLI process for the given session
func (m *CLIManager) StartChatSession(sessionID string) (*ChatSession, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fmt.Printf("[CLI_MANAGER] StartChatSession called for session: %s\n", sessionID)
	
	// Get session data to determine project path
	sessionData, err := m.sessionRepository.GetSessionByID(sessionID)
	if err != nil {
		fmt.Printf("[CLI_MANAGER] Failed to get session data: %v\n", err)
		return nil, fmt.Errorf("failed to get session data: %w", err)
	}

	// Check if we already have an active process for this session
	if existingProcess, exists := m.processes[sessionID]; exists {
		if existingProcess.Status == StatusActive {
			fmt.Printf("[CLI_MANAGER] Found existing active process for session: %s\n", sessionID)
			// Update last used time
			existingProcess.mutex.Lock()
			existingProcess.LastUsed = time.Now()
			existingProcess.mutex.Unlock()
			
			// Return existing chat session
			return m.repository.GetChatSessionBySessionID(sessionID)
		}
	}

	// Check if we have an existing chat session with Claude session ID to resume
	existingChatSession, err := m.repository.GetChatSessionBySessionID(sessionID)
	if err == nil && existingChatSession != nil && existingChatSession.ClaudeSessionID != nil && *existingChatSession.ClaudeSessionID != "" {
		fmt.Printf("[CLI_MANAGER] Found existing chat session with Claude ID: %s\n", *existingChatSession.ClaudeSessionID)
		
		// Create a new process but with the existing Claude session ID
		process, err := m.createCLIProcess(sessionID, sessionData.ProjectPath)
		if err != nil {
			fmt.Printf("[CLI_MANAGER] Failed to create CLI process: %v\n", err)
			return nil, fmt.Errorf("failed to create CLI process: %w", err)
		}
		
		// Set the Claude session ID for continuation
		process.claudeSessionID = *existingChatSession.ClaudeSessionID
		process.isFirstMessage = false
		
		// Start the process
		fmt.Printf("[CLI_MANAGER] Starting process for session: %s with existing Claude session: %s\n", sessionID, process.claudeSessionID)
		err = m.startProcess(process)
		if err != nil {
			fmt.Printf("[CLI_MANAGER] Failed to start CLI process: %v\n", err)
			return nil, fmt.Errorf("failed to start CLI process: %w", err)
		}
		
		// Store the process
		m.processes[sessionID] = process
		
		// Update the chat session status to active
		m.repository.UpdateChatSessionStatus(existingChatSession.ID, StatusActive)
		
		return existingChatSession, nil
	}

	// Check process limits
	if len(m.processes) >= m.maxProcesses {
		fmt.Printf("[CLI_MANAGER] Process limit reached: %d/%d\n", len(m.processes), m.maxProcesses)
		return nil, fmt.Errorf("maximum number of processes reached: %d", m.maxProcesses)
	}

	fmt.Printf("[CLI_MANAGER] Creating new CLI process for session: %s\n", sessionID)
	
	// Create new CLI process
	process, err := m.createCLIProcess(sessionID, sessionData.ProjectPath)
	if err != nil {
		fmt.Printf("[CLI_MANAGER] Failed to create CLI process: %v\n", err)
		return nil, fmt.Errorf("failed to create CLI process: %w", err)
	}

	// Start the process
	fmt.Printf("[CLI_MANAGER] Starting process for session: %s\n", sessionID)
	err = m.startProcess(process)
	if err != nil {
		fmt.Printf("[CLI_MANAGER] Failed to start CLI process: %v\n", err)
		return nil, fmt.Errorf("failed to start CLI process: %w", err)
	}

	// Store the process
	m.processes[sessionID] = process
	fmt.Printf("[CLI_MANAGER] Process stored, now have %d processes\n", len(m.processes))

	// Create chat session in database with PID 0 (since we don't have a real process)
	fmt.Printf("[CLI_MANAGER] Creating chat session in database for session: %s\n", sessionID)
	chatSession, err := m.repository.CreateChatSession(sessionID, 0)
	if err != nil {
		fmt.Printf("[CLI_MANAGER] Failed to create chat session in DB: %v\n", err)
		// Cleanup process if database creation fails
		m.stopProcess(process)
		delete(m.processes, sessionID)
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	fmt.Printf("[CLI_MANAGER] Chat session created successfully with ID: %s\n", chatSession.ID)
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
func (m *CLIManager) createCLIProcess(sessionID, projectPath string) (*CLIProcess, error) {
	ctx, cancel := context.WithCancel(context.Background())

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
		projectPath:    projectPath,
	}

	fmt.Printf("[CLI_MANAGER] Created process for session %s with project path: %s\n", sessionID, projectPath)
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
			
			// Process the message in an anonymous function to ensure proper context cleanup
			func() {
				// Build command
				var cmd *exec.Cmd
				claudePath := "claude" // Try to find claude in PATH first
				
				// Check if claude is in the expected location
				if _, err := exec.LookPath("claude"); err != nil {
					// Try common installation paths
					homeDir, _ := os.UserHomeDir()
					possiblePaths := []string{
						filepath.Join(homeDir, ".npm-global", "bin", "claude"),
						filepath.Join(homeDir, ".local", "bin", "claude"),
						"/usr/local/bin/claude",
						"/opt/homebrew/bin/claude",
					}
					
					for _, path := range possiblePaths {
						if _, err := os.Stat(path); err == nil {
							claudePath = path
							break
						}
					}
				}
				
				fmt.Printf("[CLI_COMMAND] Using claude at: %s\n", claudePath)
				
				// Create a timeout context for this specific command (increased timeout for long Claude responses)
				cmdCtx, cmdCancel := context.WithTimeout(process.ctx, 5*time.Minute)
				defer cmdCancel() // This will be called when the anonymous function returns
				
				if process.isFirstMessage {
					// First message - start new conversation with JSON output to get session ID
					cmd = exec.CommandContext(cmdCtx, claudePath, "--print", "--output-format", "json", message)
					fmt.Printf("[CLI_COMMAND] Session %s: Running first message command: %s --print --output-format json \"%s\"\n", process.SessionID, claudePath, message)
					process.isFirstMessage = false
				} else {
					// Continue existing conversation using session ID with JSON output
					if process.claudeSessionID == "" {
						fmt.Printf("[CLI_ERROR] Session %s: No Claude session ID available for continuation\n", process.SessionID)
						select {
						case process.ErrorChan <- fmt.Errorf("no Claude session ID available"):
						default:
						}
						return
					}
					cmd = exec.CommandContext(cmdCtx, claudePath, "--print", "--output-format", "json", "--resume", process.claudeSessionID, message)
					fmt.Printf("[CLI_COMMAND] Session %s: Running continuation command: %s --print --output-format json --resume %s \"%s\"\n", process.SessionID, claudePath, process.claudeSessionID, message)
				}
				
				// Set working directory if project path is available
				if process.projectPath != "" && process.projectPath != "/" {
					cmd.Dir = process.projectPath
					fmt.Printf("[CLI_WORKDIR] Session %s: Set working directory to: %s\n", process.SessionID, process.projectPath)
				}
				
				fmt.Printf("[CLI_EXECUTE] Session %s: About to execute command\n", process.SessionID)
				
				// Check if context is already cancelled
				select {
				case <-cmdCtx.Done():
					fmt.Printf("[CLI_ERROR] Session %s: Context already cancelled before execution: %v\n", process.SessionID, cmdCtx.Err())
					return
				default:
					// Context is still active, proceed
				}
				
				// Get stdout pipe
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					fmt.Printf("[CLI_ERROR] Session %s: Failed to get stdout pipe: %v\n", process.SessionID, err)
					select {
					case process.ErrorChan <- fmt.Errorf("failed to get stdout pipe: %w", err):
					default:
					}
					return
				}

				// Start the command
				fmt.Printf("[CLI_EXECUTE] Session %s: Starting command\n", process.SessionID)
				if err := cmd.Start(); err != nil {
					fmt.Printf("[CLI_ERROR] Session %s: Failed to start command: %v\n", process.SessionID, err)
					select {
					case process.ErrorChan <- fmt.Errorf("failed to start claude: %w", err):
					default:
					}
					return
				}

				// Read output
				var output []byte
				output, err = io.ReadAll(stdout)
				if err != nil {
					fmt.Printf("[CLI_ERROR] Session %s: Failed to read output: %v\n", process.SessionID, err)
					select {
					case process.ErrorChan <- fmt.Errorf("failed to read output: %w", err):
					default:
					}
					return
				}

				// Wait for command to complete
				if err := cmd.Wait(); err != nil {
					fmt.Printf("[CLI_ERROR] Session %s: Command failed: %v\n", process.SessionID, err)
					select {
					case process.ErrorChan <- fmt.Errorf("claude command failed: %w", err):
					default:
					}
					return
				}

				fmt.Printf("[CLI_EXECUTE] Session %s: Command execution completed, output length: %d\n", process.SessionID, len(output))
				
				// Parse JSON response (we always use JSON format now)
				response := strings.TrimSpace(string(output))
				fmt.Printf("[CLI_RESPONSE] Session %s: Got response (%d bytes)\n", process.SessionID, len(response))
				
				var finalResponse string
				var claudeResp ClaudeResponse
				if err := json.Unmarshal([]byte(response), &claudeResp); err != nil {
					fmt.Printf("[CLI_ERROR] Session %s: Failed to parse JSON response: %v\n", process.SessionID, err)
					finalResponse = response // Fall back to raw response
				} else {
					// Always extract and store the Claude session ID (it might change)
					if claudeResp.SessionID != "" {
						process.claudeSessionID = claudeResp.SessionID
						fmt.Printf("[CLI_SESSION] Session %s: Updated Claude session ID: %s\n", process.SessionID, process.claudeSessionID)
						
						// Update the database with the Claude session ID
						if chatSession, err := m.repository.GetChatSessionBySessionID(process.SessionID); err == nil && chatSession != nil {
							m.repository.UpdateChatSessionClaudeID(chatSession.ID, process.claudeSessionID)
						}
					}
					
					// Use the actual response text
					finalResponse = claudeResp.Result
				}
				
				select {
				case process.OutputChan <- finalResponse:
					fmt.Printf("[CLI_SUCCESS] Session %s: Response sent to channel\n", process.SessionID)
				default:
					fmt.Printf("[CLI_DROPPED] Session %s: Output channel full\n", process.SessionID)
				}
			}()
			
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