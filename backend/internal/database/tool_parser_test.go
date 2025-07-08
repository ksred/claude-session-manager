package database

import (
	"testing"
	"time"
)

func TestExtractToolCallsFromMessage(t *testing.T) {
	testTime := time.Now()

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "Extract Edit tool from invoke tags",
			content: `<function_calls>
<invoke name="Edit">
<parameter name="file_path">/src/app.ts</parameter>
<parameter name="old_string">const x = 1</parameter>
<parameter name="new_string">const x = 2</parameter>
</invoke>
</function_calls>`,
			expected: 1,
		},
		{
			name: "Extract Write tool from invoke tags",
			content: `<function_calls>
<invoke name="Write">
<parameter name="file_path">/src/config.json</parameter>
<parameter name="content">{"key": "value"}</parameter>
</invoke>
</function_calls>`,
			expected: 1,
		},
		{
			name: "Extract multiple tools",
			content: `<function_calls>
<invoke name="Edit">
<parameter name="file_path">/src/app.ts</parameter>
<parameter name="old_string">old</parameter>
<parameter name="new_string">new</parameter>
</invoke>
<invoke name="Write">
<parameter name="file_path">/src/test.ts</parameter>
<parameter name="content">test content</parameter>
</invoke>
</function_calls>`,
			expected: 2,
		},
		{
			name: "JSON tool result format",
			content: `[{"type":"tool_result","tool_use_id":"abc123","content":[{"tool_name":"Edit","file_path":"/src/app.ts"}]}]`,
			expected: 1,
		},
		{
			name: "No tools in message",
			content: `Just a regular message without any tool calls`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCalls := ExtractToolCallsFromMessage(tt.content, testTime)
			if len(toolCalls) != tt.expected {
				t.Errorf("ExtractToolCallsFromMessage() returned %d tool calls, expected %d", len(toolCalls), tt.expected)
			}

			// Verify file paths are extracted correctly
			if len(toolCalls) > 0 && tt.expected > 0 {
				if toolCalls[0].FilePath == "" {
					t.Error("Expected file path to be extracted")
				}
			}
		})
	}
}

func TestIsFileModifyingTool(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		{"Edit", true},
		{"Write", true},
		{"MultiEdit", true},
		{"NotebookEdit", true},
		{"NotebookWrite", true},
		{"Read", false},
		{"Bash", false},
		{"Search", false},
		{"edit", true}, // Case insensitive
		{"WRITE", true}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := isFileModifyingTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("isFileModifyingTool(%s) = %v, expected %v", tt.toolName, result, tt.expected)
			}
		})
	}
}