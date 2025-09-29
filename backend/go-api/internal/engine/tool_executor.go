package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"io"

	"egobackend/internal/models"
)

// toolExecutor is responsible for managing the concurrent execution of tool calls.
type toolExecutor struct {
	llmClient *llmClient
}

// newToolExecutor creates a new tool executor.
func newToolExecutor(client *llmClient) *toolExecutor {
	return &toolExecutor{llmClient: client}
}

// execute concurrently runs all tool calls requested by the LLM.
// It uses goroutines for parallelism and channels to collect results.
func (te *toolExecutor) execute(toolCalls []models.ToolCall, callback EventCallback, memEnabled bool, userID int) []map[string]interface{} {
	if len(toolCalls) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	resultsChan := make(chan map[string]interface{}, len(toolCalls))
	toolCounts := make(map[string]int)

	// First, count the calls for each tool to provide better feedback.
	for _, tc := range toolCalls {
		toolCounts[tc.ToolName]++
	}
	// Send a header message for each tool type.
	for name, count := range toolCounts {
		header := fmt.Sprintf("Using %s", name)
		if count > 1 {
			header = fmt.Sprintf("Using %s (%d times)", name, count)
		}
		callback("thought_header", header)
	}

	// Launch a goroutine for each tool call.
	for _, toolCall := range toolCalls {
		wg.Add(1)
		go func(tc models.ToolCall) {
			defer wg.Done()
			callback("tool_call", tc)
			toolResult, err := te.callPythonTool(tc.ToolName, tc.ToolQuery, memEnabled, userID)
			if err != nil {
				log.Printf("!!! Tool call error for '%s': %v", tc.ToolName, err)
				resultsChan <- map[string]interface{}{
					"type":      "tool_error",
					"tool_name": tc.ToolName,
					"error":     err.Error(),
				}
			} else {
				resultsChan <- map[string]interface{}{
					"type":      "tool_output",
					"tool_name": tc.ToolName,
					"output":    toolResult,
				}
			}
		}(toolCall)
	}

	// A separate goroutine waits for all tool calls to complete, then closes the channel.
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results and provide progress updates.
	progress := make(map[string]int)
	var finalResults []map[string]interface{}
	for result := range resultsChan {
		finalResults = append(finalResults, result)
		callback(result["type"].(string), result)

		if toolName, ok := result["tool_name"].(string); ok {
			progress[toolName]++
			done := progress[toolName]
			total := toolCounts[toolName]
			status := "completed"
			if result["type"] == "tool_error" {
				status = "failed"
			}
			callback("thought_header", fmt.Sprintf("%s %s (%d/%d)", toolName, status, done, total))
		}
	}

	return finalResults
}

// callPythonTool executes a single tool by making a POST request to the Python service.
// It dynamically sets the timeout based on the tool being called.
func (te *toolExecutor) callPythonTool(toolName, toolQuery string, memEnabled bool, userID int) (string, error) {
	reqBody := map[string]interface{}{
		"query":          toolQuery,
		"user_id":        fmt.Sprintf("%d", userID),
		"memory_enabled": memEnabled,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tool request body: %w", err)
	}

	url := te.llmClient.baseURL + fmt.Sprintf("/execute_tool/%s", toolName)
	timeout := te.getTimeoutForTool(toolName)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create tool request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[ToolExecutor] Executing %s with timeout %v", toolName, timeout)
	resp, err := te.llmClient.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("tool '%s' timed out after %v", toolName, timeout)
		}
		return "", fmt.Errorf("request for tool '%s' failed: %w", toolName, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read tool response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tool service for '%s' returned error (status %d): %s", toolName, resp.StatusCode, string(respBytes))
	}

	var toolResult map[string]string
	if err := json.Unmarshal(respBytes, &toolResult); err != nil {
		return "", fmt.Errorf("failed to parse tool result JSON: %w. Response: %s", err, string(respBytes))
	}

	if result, ok := toolResult["result"]; ok {
		return result, nil
	}

	return "", fmt.Errorf("key 'result' not found in tool response")
}

// getTimeoutForTool returns a specific timeout duration for a given tool name.
func (te *toolExecutor) getTimeoutForTool(toolName string) time.Duration {
	// TODO: Move these values to the configuration file.
	switch toolName {
	case "EgoTube":
		return 10 * time.Minute // Longer timeout for video analysis.
	case "EgoSearch":
		return 2 * time.Minute
	case "EgoCodeExec":
		return 5 * time.Minute
	default:
		return 1 * time.Minute // Default timeout for other tools.
	}
}