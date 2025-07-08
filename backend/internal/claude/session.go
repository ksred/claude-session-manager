package claude

import (
	"time"
)

// SessionStatus represents the current state of a Claude session
type SessionStatus int

const (
	StatusWorking SessionStatus = iota
	StatusIdle
	StatusComplete
	StatusError
)

func (s SessionStatus) String() string {
	switch s {
	case StatusWorking:
		return "Working"
	case StatusIdle:
		return "Idle"
	case StatusComplete:
		return "Complete"
	case StatusError:
		return "Error"
	default:
		return "Unknown"
	}
}

// ModelPricing represents pricing for a specific model (per 1K tokens)
type ModelPricing struct {
	InputTokenPrice              float64
	OutputTokenPrice             float64
	CacheCreationInputTokenPrice float64
	CacheReadInputTokenPrice     float64
}

// Official Claude pricing as of July 2025 (converted from per million to per 1K tokens)
// Using 5m cache tier pricing (most common)
var ModelPricingTable = map[string]ModelPricing{
	// Claude 4 models  
	"claude-opus-4": {
		InputTokenPrice: 0.015, OutputTokenPrice: 0.075, // $15/$75 per million
		CacheCreationInputTokenPrice: 0.01875, CacheReadInputTokenPrice: 0.0015, // $18.75/$1.50 per million
	},
	"claude-opus-4-20250514": {
		InputTokenPrice: 0.015, OutputTokenPrice: 0.075,
		CacheCreationInputTokenPrice: 0.01875, CacheReadInputTokenPrice: 0.0015,
	},
	"claude-sonnet-4": {
		InputTokenPrice: 0.003, OutputTokenPrice: 0.015, // $3/$15 per million
		CacheCreationInputTokenPrice: 0.00375, CacheReadInputTokenPrice: 0.0003, // $3.75/$0.30 per million
	},
	"claude-sonnet-4-20250514": {
		InputTokenPrice: 0.003, OutputTokenPrice: 0.015,
		CacheCreationInputTokenPrice: 0.00375, CacheReadInputTokenPrice: 0.0003,
	},
	
	// Claude 3.x models
	"claude-3-opus": {
		InputTokenPrice: 0.015, OutputTokenPrice: 0.075,
		CacheCreationInputTokenPrice: 0.01875, CacheReadInputTokenPrice: 0.0015,
	},
	"claude-3-sonnet": {
		InputTokenPrice: 0.003, OutputTokenPrice: 0.015,
		CacheCreationInputTokenPrice: 0.00375, CacheReadInputTokenPrice: 0.0003,
	},
	"claude-3.5-sonnet": {
		InputTokenPrice: 0.003, OutputTokenPrice: 0.015,
		CacheCreationInputTokenPrice: 0.00375, CacheReadInputTokenPrice: 0.0003,
	},
	"claude-3.7-sonnet": {
		InputTokenPrice: 0.003, OutputTokenPrice: 0.015,
		CacheCreationInputTokenPrice: 0.00375, CacheReadInputTokenPrice: 0.0003,
	},
	"claude-3-haiku": {
		InputTokenPrice: 0.00025, OutputTokenPrice: 0.00125, // $0.25/$1.25 per million
		CacheCreationInputTokenPrice: 0.0003, CacheReadInputTokenPrice: 0.00003, // $0.30/$0.03 per million
	},
	"claude-3.5-haiku": {
		InputTokenPrice: 0.0008, OutputTokenPrice: 0.004, // $0.80/$4 per million
		CacheCreationInputTokenPrice: 0.001, CacheReadInputTokenPrice: 0.00008, // $1.00/$0.08 per million
	},
}

// Default pricing for unknown models (Sonnet pricing)
var DefaultModelPricing = ModelPricing{
	InputTokenPrice:              0.003,   // $3.00 per million = $0.003 per 1K
	OutputTokenPrice:             0.015,   // $15.00 per million = $0.015 per 1K
	CacheCreationInputTokenPrice: 0.00375, // $3.75 per million = $0.00375 per 1K
	CacheReadInputTokenPrice:     0.0003,  // $0.30 per million = $0.0003 per 1K
}

// Token pricing constants (per 1K tokens) - DEPRECATED
// Deprecated: Use GetModelPricing instead
const (
	InputTokenPricePerK  = 0.003  // Default to Sonnet pricing
	OutputTokenPricePerK = 0.015
)

// GetModelPricing returns pricing for a specific model, with fallback to default
func GetModelPricing(modelName string) ModelPricing {
	if pricing, exists := ModelPricingTable[modelName]; exists {
		return pricing
	}
	return DefaultModelPricing
}

// TokenUsage tracks token consumption and costs for a session
type TokenUsage struct {
	InputTokens              int     `json:"input_tokens"`
	OutputTokens             int     `json:"output_tokens"`
	CacheCreationInputTokens int     `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int     `json:"cache_read_input_tokens"`
	TotalTokens              int     `json:"total_tokens"`
	EstimatedCost            float64 `json:"estimated_cost"`
}

// CalculateCost computes the estimated cost based on token usage
// Deprecated: Use CalculateCostWithPricing instead
func (tu *TokenUsage) CalculateCost() float64 {
	inputCost := float64(tu.InputTokens) / 1000.0 * InputTokenPricePerK
	outputCost := float64(tu.OutputTokens) / 1000.0 * OutputTokenPricePerK
	return inputCost + outputCost
}

// CalculateCostWithPricing computes the estimated cost based on token usage with custom pricing
func (tu *TokenUsage) CalculateCostWithPricing(inputPricePerK, outputPricePerK float64) float64 {
	inputCost := float64(tu.InputTokens) / 1000.0 * inputPricePerK
	outputCost := float64(tu.OutputTokens) / 1000.0 * outputPricePerK
	return inputCost + outputCost
}

// CalculateCostWithModel computes the estimated cost using model-specific pricing including cache costs
func (tu *TokenUsage) CalculateCostWithModel(modelName string) float64 {
	pricing := GetModelPricing(modelName)
	
	inputCost := float64(tu.InputTokens) / 1000.0 * pricing.InputTokenPrice
	outputCost := float64(tu.OutputTokens) / 1000.0 * pricing.OutputTokenPrice
	cacheCreationCost := float64(tu.CacheCreationInputTokens) / 1000.0 * pricing.CacheCreationInputTokenPrice
	cacheReadCost := float64(tu.CacheReadInputTokens) / 1000.0 * pricing.CacheReadInputTokenPrice
	
	return inputCost + outputCost + cacheCreationCost + cacheReadCost
}

// UpdateTotals recalculates total tokens and estimated cost
// Deprecated: Use UpdateTotalsWithPricing instead
func (tu *TokenUsage) UpdateTotals() {
	tu.TotalTokens = tu.InputTokens + tu.OutputTokens + tu.CacheCreationInputTokens + tu.CacheReadInputTokens
	tu.EstimatedCost = tu.CalculateCost()
}

// UpdateTotalsWithPricing recalculates total tokens and estimated cost with custom pricing
func (tu *TokenUsage) UpdateTotalsWithPricing(inputPricePerK, outputPricePerK float64) {
	tu.TotalTokens = tu.InputTokens + tu.OutputTokens + tu.CacheCreationInputTokens + tu.CacheReadInputTokens
	tu.EstimatedCost = tu.CalculateCostWithPricing(inputPricePerK, outputPricePerK)
}

// Message represents a single message in the Claude session
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Usage     TokenUsage             `json:"usage,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

// Session represents a complete Claude Code session with all metadata
type Session struct {
	ID            string        `json:"id"`
	ProjectPath   string        `json:"project_path"`
	ProjectName   string        `json:"project_name"`
	GitBranch     string        `json:"git_branch"`
	GitWorktree   string        `json:"git_worktree"`
	Status        SessionStatus `json:"status"`
	StartTime     time.Time     `json:"start_time"`
	LastActivity  time.Time     `json:"last_activity"`
	CurrentTask   string        `json:"current_task"`
	TokensUsed    TokenUsage    `json:"tokens_used"`
	FilesModified []string      `json:"files_modified"`
	Messages      []Message     `json:"messages"`
	FilePath      string        `json:"file_path"` // Path to the .jsonl file
}

// Duration returns the total duration of the session
func (s *Session) Duration() time.Duration {
	if s.StartTime.IsZero() {
		return 0
	}
	endTime := s.LastActivity
	if endTime.IsZero() {
		endTime = time.Now()
	}
	return endTime.Sub(s.StartTime)
}

// TimeSinceLastActivity returns how long since the last activity
func (s *Session) TimeSinceLastActivity() time.Duration {
	if s.LastActivity.IsZero() {
		return 0
	}
	return time.Since(s.LastActivity)
}

// IsActive returns true if the session has had recent activity
func (s *Session) IsActive() bool {
	if s.LastActivity.IsZero() {
		return false
	}
	return s.TimeSinceLastActivity() < 30*time.Minute
}

// UpdateStatus determines the current status based on activity and message content
func (s *Session) UpdateStatus() {
	timeSinceActivity := s.TimeSinceLastActivity()
	
	// Check if there are any messages
	if len(s.Messages) == 0 {
		s.Status = StatusIdle
		return
	}
	
	// Check the last message for error indicators
	lastMessage := s.Messages[len(s.Messages)-1]
	if lastMessage.Type == "error" {
		s.Status = StatusError
		return
	}
	
	// Determine status based on recent activity
	switch {
	case timeSinceActivity < 2*time.Minute:
		s.Status = StatusWorking
	case timeSinceActivity < 15*time.Minute:
		s.Status = StatusIdle
	default:
		s.Status = StatusComplete
	}
}

// GetMessageCount returns the total number of messages in the session
func (s *Session) GetMessageCount() int {
	return len(s.Messages)
}

// GetUserMessageCount returns the number of user messages
func (s *Session) GetUserMessageCount() int {
	count := 0
	for _, msg := range s.Messages {
		if msg.Role == "user" {
			count++
		}
	}
	return count
}

// GetAssistantMessageCount returns the number of assistant messages
func (s *Session) GetAssistantMessageCount() int {
	count := 0
	for _, msg := range s.Messages {
		if msg.Role == "assistant" {
			count++
		}
	}
	return count
}

// GetLastUserMessage returns the most recent user message content
func (s *Session) GetLastUserMessage() string {
	for i := len(s.Messages) - 1; i >= 0; i-- {
		if s.Messages[i].Role == "user" {
			return s.Messages[i].Content
		}
	}
	return ""
}

// HasErrors returns true if the session contains any error messages
func (s *Session) HasErrors() bool {
	for _, msg := range s.Messages {
		if msg.Type == "error" {
			return true
		}
	}
	return false
}

// GetErrorMessages returns all error messages in the session
func (s *Session) GetErrorMessages() []Message {
	var errors []Message
	for _, msg := range s.Messages {
		if msg.Type == "error" {
			errors = append(errors, msg)
		}
	}
	return errors
}