package database

import (
	"encoding/json"
	"strings"
	"time"
)

// ToolCall represents a parsed tool invocation from a message
type ToolCall struct {
	ToolName   string
	FilePath   string
	Parameters map[string]interface{}
	Timestamp  time.Time
}

// ExtractToolCallsFromMessage parses message content to extract tool invocations
func ExtractToolCallsFromMessage(content string, timestamp time.Time) []ToolCall {
	var toolCalls []ToolCall

	// Try to parse as JSON first (for structured tool results)
	var jsonContent interface{}
	if err := json.Unmarshal([]byte(content), &jsonContent); err == nil {
		// Handle array of tool results
		if arr, ok := jsonContent.([]interface{}); ok {
			for _, item := range arr {
				if toolCall := parseJSONToolCall(item, timestamp); toolCall != nil {
					toolCalls = append(toolCalls, *toolCall)
				}
			}
			return toolCalls
		} else if toolCall := parseJSONToolCall(jsonContent, timestamp); toolCall != nil {
			toolCalls = append(toolCalls, *toolCall)
			return toolCalls
		}
	}

	// Look for common tool patterns in the content
	// Pattern: <invoke name="ToolName">
	if strings.Contains(content, "<invoke name=") {
		toolCalls = append(toolCalls, extractFromInvokeTags(content, timestamp)...)
	}

	// Don't double count Edit tools that were already found in invoke tags
	// Pattern: Edit tool with old_string/new_string (only if not already found)
	if len(toolCalls) == 0 && strings.Contains(content, "old_string") && strings.Contains(content, "new_string") {
		if toolCall := extractEditTool(content, timestamp); toolCall != nil {
			toolCalls = append(toolCalls, *toolCall)
		}
	}

	return toolCalls
}

// parseJSONToolCall parses a JSON object that might be a tool call
func parseJSONToolCall(data interface{}, timestamp time.Time) *ToolCall {
	obj, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	// Check for different tool call formats
	var toolName string
	var filePath string
	var params map[string]interface{}
	
	// Format 1: {"type": "tool_use", "name": "Edit", "input": {"file_path": "..."}}
	if typ, ok := obj["type"].(string); ok && typ == "tool_use" {
		if name, ok := obj["name"].(string); ok {
			toolName = name
		}
		if input, ok := obj["input"].(map[string]interface{}); ok {
			params = input
			if fp, ok := input["file_path"].(string); ok {
				filePath = fp
			}
		}
	} else if name, ok := obj["tool_name"].(string); ok {
		// Format 2: Legacy format with tool_name directly
		toolName = name
		if fp, ok := obj["file_path"].(string); ok {
			filePath = fp
		}
		// Extract parameters
		params = make(map[string]interface{})
		for k, v := range obj {
			if k != "tool_name" && k != "file_path" && k != "type" {
				params[k] = v
			}
		}
	} else if typ, ok := obj["type"].(string); ok && typ == "tool_result" {
		// Format 3: Tool result format
		if content, ok := obj["content"].([]interface{}); ok && len(content) > 0 {
			if contentObj, ok := content[0].(map[string]interface{}); ok {
				if name, ok := contentObj["tool_name"].(string); ok {
					toolName = name
				}
				if fp, ok := contentObj["file_path"].(string); ok {
					filePath = fp
				}
			}
		}
	}

	if toolName == "" {
		return nil
	}

	if params == nil {
		params = make(map[string]interface{})
	}

	return &ToolCall{
		ToolName:   toolName,
		FilePath:   filePath,
		Parameters: params,
		Timestamp:  timestamp,
	}
}

// extractFromInvokeTags extracts tool calls from <invoke> tags
func extractFromInvokeTags(content string, timestamp time.Time) []ToolCall {
	var toolCalls []ToolCall

	// Find all invoke tags
	parts := strings.Split(content, "<invoke name=\"")
	for i := 1; i < len(parts); i++ {
		endIdx := strings.Index(parts[i], "\"")
		if endIdx == -1 {
			continue
		}
		
		toolName := parts[i][:endIdx]
		
		// Extract parameters
		invokeEnd := strings.Index(parts[i], "</invoke>")
		if invokeEnd == -1 {
			continue
		}
		
		paramSection := parts[i][endIdx+2 : invokeEnd]
		params := extractParameters(paramSection)
		
		// Get file path from parameters
		filePath := ""
		if fp, ok := params["file_path"].(string); ok {
			filePath = fp
		}
		
		toolCalls = append(toolCalls, ToolCall{
			ToolName:   toolName,
			FilePath:   filePath,
			Parameters: params,
			Timestamp:  timestamp,
		})
	}

	return toolCalls
}

// extractParameters extracts parameters from parameter tags
func extractParameters(content string) map[string]interface{} {
	params := make(map[string]interface{})
	
	// Find parameter tags
	parts := strings.Split(content, "<parameter name=\"")
	for i := 1; i < len(parts); i++ {
		nameEnd := strings.Index(parts[i], "\"")
		if nameEnd == -1 {
			continue
		}
		
		paramName := parts[i][:nameEnd]
		
		// Find the parameter value
		valueStart := strings.Index(parts[i], ">")
		valueEnd := strings.Index(parts[i], "</")
		
		if valueStart != -1 && valueEnd != -1 && valueEnd > valueStart {
			paramValue := parts[i][valueStart+1 : valueEnd]
			params[paramName] = strings.TrimSpace(paramValue)
		}
	}
	
	return params
}

// extractEditTool looks for Edit tool patterns in content
func extractEditTool(content string, timestamp time.Time) *ToolCall {
	// Look for common Edit tool patterns
	if !strings.Contains(content, "file_path") {
		return nil
	}

	params := make(map[string]interface{})
	
	// Try to extract file path
	if idx := strings.Index(content, "file_path"); idx != -1 {
		// Look for the file path value after file_path
		remaining := content[idx+9:] // Skip "file_path"
		
		// Skip to the actual value (after : or ")
		valueStart := strings.IndexAny(remaining, ":\"")
		if valueStart != -1 {
			remaining = remaining[valueStart+1:]
			// Trim quotes and whitespace
			remaining = strings.TrimSpace(remaining)
			remaining = strings.Trim(remaining, "\"'")
			
			// Find the end of the path
			endIdx := strings.IndexAny(remaining, "\",\n")
			if endIdx != -1 {
				filePath := remaining[:endIdx]
				return &ToolCall{
					ToolName:   "Edit",
					FilePath:   filePath,
					Parameters: params,
					Timestamp:  timestamp,
				}
			}
		}
	}

	return nil
}

// isFileModifyingTool checks if a tool name is one that modifies files
func isFileModifyingTool(toolName string) bool {
	fileTools := []string{"Edit", "Write", "MultiEdit", "NotebookEdit", "NotebookWrite"}
	toolNameLower := strings.ToLower(toolName)
	
	for _, tool := range fileTools {
		if strings.ToLower(tool) == toolNameLower {
			return true
		}
	}
	
	return false
}