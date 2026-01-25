package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"egobackend/internal/database"
	"egobackend/internal/models"
)

// toolExecutor is responsible for managing the concurrent execution of tool calls.
type toolExecutor struct {
	llmClient *llmClient
	db        *database.DB
}

// newToolExecutor creates a new tool executor.
func newToolExecutor(client *llmClient, db *database.DB) *toolExecutor {
	return &toolExecutor{llmClient: client, db: db}
}

// execute concurrently runs all tool calls requested by the LLM.
// It uses goroutines for parallelism and channels to collect results.
func (te *toolExecutor) execute(toolCalls []models.ToolCall, callback EventCallback, memEnabled bool, userID int, sessionUUID string) []map[string]interface{} {
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
	
	// Launch a goroutine for each tool call.
	for i, toolCall := range toolCalls {
		wg.Add(1)
		// Generate a unique ID for this specific call instance
		callID := fmt.Sprintf("%s_%d_%d", toolCall.ToolName, time.Now().UnixNano(), i)
		
		go func(tc models.ToolCall, cid string) {
			defer wg.Done()
			
			// Send tool_call with explicit call_id
			callback("tool_call", map[string]interface{}{
				"tool_name":  tc.ToolName,
				"tool_query": tc.ToolQuery,
				"call_id":    cid,
			})

			var toolResult string
			var err error

			if tc.ToolName == "manage_plan" {
				toolResult, err = te.executePlanManager(tc.ToolQuery, sessionUUID, callback)
			} else {
				toolResult, err = te.callPythonTool(tc.ToolName, tc.ToolQuery, memEnabled, userID)
			}
			
			if err != nil {
				log.Printf("!!! Tool call error for '%s': %v", tc.ToolName, err)
				resultsChan <- map[string]interface{}{
					"type":      "tool_error",
					"tool_name": tc.ToolName,
					"call_id":   cid,
					"error":     err.Error(),
				}
			} else {
				resultsChan <- map[string]interface{}{
					"type":      "tool_output",
					"tool_name": tc.ToolName,
					"call_id":   cid,
					"output":    toolResult,
				}
			}
		}(toolCall, callID)
	}

	// A separate goroutine waits for all tool calls to complete, then closes the channel.
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results and provide progress updates.
	var finalResults []map[string]interface{}
	for result := range resultsChan {
		finalResults = append(finalResults, result)
		callback(result["type"].(string), result)
	}

	return finalResults
}

// executePlanManager handles the local execution of the plan management tool.
func (te *toolExecutor) executePlanManager(query string, sessionUUID string, callback EventCallback) (string, error) {
	var args struct {
		Action      string   `json:"action"` // create, update_step, complete
		Title       string   `json:"title,omitempty"`
		Steps       []string `json:"steps,omitempty"`
		StepOrder   int      `json:"step_order,omitempty"`
		Status      string   `json:"status,omitempty"`
		Description string   `json:"description,omitempty"`
	}

	if err := json.Unmarshal([]byte(query), &args); err != nil {
		return "", fmt.Errorf("invalid JSON for manage_plan: %w", err)
	}

	// Validate action before processing
	validActions := []string{"create", "update_step", "complete"}
	isValidAction := false
	for _, va := range validActions {
		if args.Action == va {
			isValidAction = true
			break
		}
	}
	if !isValidAction {
		return "", fmt.Errorf("invalid action '%s'. Valid actions are: create, update_step, complete", args.Action)
	}

	switch args.Action {
	case "create":
		if len(args.Steps) == 0 {
			return "", fmt.Errorf("cannot create plan without steps")
		}
		plan, err := te.db.CreatePlan(sessionUUID, args.Title, args.Steps)
		if err != nil {
			return "", err
		}
		callback("plan_updated", plan)

		// Return the full plan JSON so the frontend can render it nicely
		planJSON, err := json.Marshal(plan)
		if err != nil {
			return fmt.Sprintf("Plan created with ID %d, but failed to marshal result.", plan.ID), nil
		}
		return string(planJSON), nil

	case "update_step":
		plan, err := te.db.GetActivePlan(sessionUUID)
		if err != nil || plan == nil {
			return "", fmt.Errorf("no active plan found for this session")
		}

		// Validate status if provided
		if args.Status != "" {
			validStatuses := []string{"pending", "in_progress", "completed", "failed", "skipped"}
			isValidStatus := false
			for _, vs := range validStatuses {
				if args.Status == vs {
					isValidStatus = true
					break
				}
			}
			if !isValidStatus {
				return "", fmt.Errorf("invalid status '%s'. Valid statuses: pending, in_progress, completed, failed, skipped", args.Status)
			}

			if err := te.db.UpdateStepStatus(plan.ID, args.StepOrder, args.Status); err != nil {
				return "", err
			}
		}
		if args.Description != "" {
			if err := te.db.UpdateStepDescription(plan.ID, args.StepOrder, args.Description); err != nil {
				return "", err
			}
		}
		// Fetch updated plan to send to frontend and return as output
		updatedPlan, err := te.db.GetActivePlan(sessionUUID)
		if err != nil {
			return "Step updated, but failed to fetch updated plan.", nil
		}

		if updatedPlan != nil {
			callback("plan_updated", updatedPlan)
			planJSON, err := json.Marshal(updatedPlan)
			if err == nil {
				return string(planJSON), nil
			}
		}
		return fmt.Sprintf("Step %d updated.", args.StepOrder), nil

	case "complete":
		plan, err := te.db.GetActivePlan(sessionUUID)
		if err != nil || plan == nil {
			return "No active plan to complete.", nil
		}
		if err := te.db.MarkPlanComplete(plan.ID); err != nil {
			return "", err
		}
		callback("plan_updated", nil) // Inform frontend that plan is gone/inactive
		return "Plan marked as complete.", nil

	default:
		return "", fmt.Errorf("unknown action: %s", args.Action)
	}
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
