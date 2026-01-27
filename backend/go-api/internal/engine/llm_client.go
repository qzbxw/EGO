package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"egobackend/internal/database"
	"egobackend/internal/models"
)

// llmClient is a dedicated client for communicating with the Python backend service.
// It handles request creation, streaming responses, and retry logic.
type llmClient struct {
	baseURL    string
	httpClient *http.Client
}

// newLLMClient creates a new client for the Python LLM service.
func newLLMClient(baseURL string, client *http.Client) *llmClient {
	return &llmClient{
		baseURL:    baseURL,
		httpClient: client,
	}
}

// warmUp sends a simple health check to the Python service to mitigate cold starts.
func (c *llmClient) warmUp(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/healthcheck", nil)
	if err != nil {
		log.Printf("[LLM Client] Warm-up request creation failed: %v", err)
		return
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[LLM Client] Warm-up request failed: %v", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Ensure the body is read and closed.
}

// generateTitle asynchronously asks the Python service to generate a concise title for a session.
func (c *llmClient) generateTitle(ctx context.Context, queryText string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // Generous timeout for a quick task.
	defer cancel()

	payload, _ := json.Marshal(map[string]string{"text": queryText})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/generate_title", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create title generation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("title generation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("title generation service returned status %d", resp.StatusCode)
	}

	var out struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode title generation response: %w", err)
	}

	return strings.TrimSpace(out.Title), nil
}

// GenerateProfileSummary asks the Python service to update the user profile based on recent history.
func (c *llmClient) GenerateProfileSummary(ctx context.Context, currentProfile string, recentHistory string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // Summarization might take a bit
	defer cancel()

	payload, _ := json.Marshal(map[string]string{
		"current_profile": currentProfile,
		"recent_history":  recentHistory,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/generate_profile_summary", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create profile summary request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("profile summary request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("profile summary service returned status %d", resp.StatusCode)
	}

	var out struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode profile summary response: %w", err)
	}

	return strings.TrimSpace(out.Summary), nil
}

// DeleteSessionVectors notifies the Python service to delete all memory vectors for a session.
func (c *llmClient) DeleteSessionVectors(ctx context.Context, userID, sessionUUID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	payload, _ := json.Marshal(map[string]string{
		"user_id":      userID,
		"session_uuid": sessionUUID,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/delete_session_vectors", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create delete vectors request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete vectors request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete vectors service returned status %d", resp.StatusCode)
	}

	return nil
}

// ClearMemory notifies the Python service to delete all memory vectors for a user.
func (c *llmClient) ClearMemory(ctx context.Context, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	payload, _ := json.Marshal(map[string]string{
		"user_id": userID,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/clear_memory", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create clear memory request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("clear memory request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clear memory service returned status %d", resp.StatusCode)
	}

	return nil
}

// GetEmbedding calls the Python backend to generate a vector embedding for the given text.
func (c *llmClient) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	payload, _ := json.Marshal(map[string]string{"text": text})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned status %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		Embedding []float64 `json:"embedding"`
		Error     string    `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if out.Error != "" {
		return nil, fmt.Errorf("embedding service error: %s", out.Error)
	}

	return out.Embedding, nil
}

// generateThought calls the /generate_thought endpoint and processes the SSE stream.
func (c *llmClient) generateThought(ctx context.Context, requestData models.PythonRequest, files []models.FilePayload, callback EventCallback, db *database.DB, sessionUUID string) (*models.ThoughtResponseWithData, error) {
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		endpoint := "/generate_thought"
		resp, err := c.doMultipartRequest(ctx, endpoint, requestData, files)
		if err != nil {
			lastErr = err
			if ctx.Err() == context.Canceled {
				return nil, ctx.Err()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			// Success, process the stream and return accumulated data
			return c.readThoughtStream(ctx, resp.Body, callback, db, sessionUUID)
		}

		// Read error body for logging
		responseBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Handle retryable errors
		if isRetryableStatusCode(resp.StatusCode) {
			lastErr = fmt.Errorf("thought generation service is temporarily unavailable (status %d)", resp.StatusCode)
			time.Sleep(time.Duration(250*attempt) * time.Millisecond) // Backoff
			continue
		}

		// Non-retryable error
		lastErr = fmt.Errorf("thought generation service returned a non-retryable error (status %d): %s", resp.StatusCode, string(responseBody))
		break
	}
	return nil, lastErr
}

// readThoughtStream processes an SSE stream from the generate_thought endpoint.
func (c *llmClient) readThoughtStream(ctx context.Context, body io.ReadCloser, callback EventCallback, db *database.DB, sessionUUID string) (*models.ThoughtResponseWithData, error) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
			return i + 2, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	var result models.ThoughtResponseWithData
	result.ToolResults = make([]map[string]interface{}, 0)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			eventBlock := scanner.Bytes()
			if !bytes.HasPrefix(eventBlock, []byte("data: ")) {
				continue
			}
			jsonPayload := bytes.TrimPrefix(eventBlock, []byte("data: "))
			if len(jsonPayload) == 0 {
				continue
			}

			var rawEvent models.StreamEvent
			if err := json.Unmarshal(jsonPayload, &rawEvent); err != nil {
				log.Printf("!!! SSE JSON parsing error in thought stream: %v", err)
				continue
			}

			// Intercept plan tool output during streaming
			if rawEvent.Type == "tool_output" {
				if dataMap, ok := rawEvent.Data.(map[string]interface{}); ok {
					if output, ok := dataMap["output"].(string); ok && strings.HasPrefix(output, "LOCAL_TOOL_SIGNAL:manage_plan:") {
						log.Printf("[llmClient] Intercepted plan signal during stream: %s", output)
						jsonQuery := strings.TrimPrefix(output, "LOCAL_TOOL_SIGNAL:manage_plan:")

						te := newToolExecutor(c, db)
						localResult, err := te.executePlanManager(jsonQuery, sessionUUID, callback)
						if err != nil {
							rawEvent.Type = "tool_error"
							dataMap["error"] = err.Error()
							delete(dataMap, "output")
						} else {
							dataMap["output"] = localResult
						}
					}
				}
			}

			// Forward all events to the client-facing SSE stream
			callback(rawEvent.Type, rawEvent.Data)

			// Accumulate data for the final object returned to Processor
			switch rawEvent.Type {
			case "thought":
				if dataMap, ok := rawEvent.Data.(map[string]interface{}); ok {
					jsonData, _ := json.Marshal(dataMap)
					json.Unmarshal(jsonData, &result.Thought)
				}
			case "tool_progress":
				// We don't save progress events to history, they are UI only
			case "tool_output", "tool_error":
				if dataMap, ok := rawEvent.Data.(map[string]interface{}); ok {
					result.ToolResults = append(result.ToolResults, dataMap)
				}
			case "usage_update":
				if dataMap, ok := rawEvent.Data.(map[string]interface{}); ok {
					result.Usage = dataMap
				}
			case "error":
				if dataMap, ok := rawEvent.Data.(map[string]interface{}); ok {
					if msg, ok := dataMap["message"].(string); ok {
						return &result, fmt.Errorf("error from Python service: %s", msg)
					}
				}
				return &result, fmt.Errorf("received error from Python service")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return &result, fmt.Errorf("error reading thought stream: %w", err)
	}
	return &result, nil
}

// synthesizeStream calls the /synthesize_stream endpoint and processes the SSE stream.
func (c *llmClient) synthesizeStream(ctx context.Context, requestData models.PythonRequest, files []models.FilePayload, callback EventCallback) (string, error) {
	const maxAttempts = 3
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		endpoint := "/synthesize_stream"
		resp, err := c.doMultipartRequest(ctx, endpoint, requestData, files)
		if err != nil {
			lastErr = err
			if ctx.Err() == context.Canceled {
				return "", ctx.Err()
			}
			continue
		}

		if resp.StatusCode == http.StatusOK {
			// Success, process the stream and return.
			return c.readStreamResponse(ctx, resp.Body, callback)
		}

		// Handle retryable errors
		resp.Body.Close()
		if isRetryableStatusCode(resp.StatusCode) {
			lastErr = fmt.Errorf("synthesis service is temporarily unavailable (status %d)", resp.StatusCode)
			time.Sleep(time.Duration(300*attempt) * time.Millisecond) // Backoff
			continue
		}

		// Non-retryable error
		lastErr = fmt.Errorf("synthesis service returned a non-retryable error (status %d)", resp.StatusCode)
		break
	}
	return "", lastErr
}

// doMultipartRequest handles the creation and execution of a multipart/form-data request.
func (c *llmClient) doMultipartRequest(ctx context.Context, endpoint string, requestData models.PythonRequest, files []models.FilePayload) (*http.Response, error) {
	// Count how many inline files actually contain Base64 data
	inlineCount := 0
	for _, f := range files {
		if strings.TrimSpace(f.Base64Data) != "" {
			inlineCount++
		}
	}

	bodyReader, contentType, err := c.createMultipartBody(requestData, files)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart body: %w", err)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		// If the context was canceled, the pipe might be closed.
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to create multipart request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-Accel-Buffering", "no")
	req.Header.Set("Accept-Encoding", "identity") // Disable compression for SSE

	// Debug logging for verification of file forwarding
	log.Printf("[LLM Client] POST %s inline_files=%d cached_files=%d", endpoint, inlineCount, len(requestData.CachedFiles))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to contact generation service: %w", err)
	}

	return resp, nil
}

// createMultipartBody creates an io.Reader for a multipart request body.
func (c *llmClient) createMultipartBody(requestData models.PythonRequest, files []models.FilePayload) (io.Reader, string, error) {
	pipeReader, pipeWriter := io.Pipe()
	mpWriter := multipart.NewWriter(pipeWriter)

	go func() {
		defer pipeWriter.Close()
		defer mpWriter.Close()

		// Write the JSON metadata part.
		jsonPart, err := json.Marshal(requestData)
		if err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to marshal request_data: %w", err))
			return
		}
		if err := mpWriter.WriteField("request_data", string(jsonPart)); err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to write request_data field: %w", err))
			return
		}

		// Sequentially write each file part.
		log.Printf("[DEBUG] createMultipartBody - processing %d files", len(files))
		for i, file := range files {
			log.Printf("[DEBUG] createMultipartBody - file %d: name='%s', mime='%s', base64_length=%d", i+1, file.FileName, file.MimeType, len(file.Base64Data))
			// Skip files that are passed by reference (empty Base64Data).
			if strings.TrimSpace(file.Base64Data) == "" {
				log.Printf("[DEBUG] createMultipartBody - skipping file %s (empty base64)", file.FileName)
				continue
			}

			header := make(textproto.MIMEHeader)
			header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename="%s"`, file.FileName))
			header.Set("Content-Type", file.MimeType)

			partWriter, err := mpWriter.CreatePart(header)
			if err != nil {
				pipeWriter.CloseWithError(fmt.Errorf("failed to create form part for %s: %w", file.FileName, err))
				return
			}

			// This part is now handled in file_processor.go before calling the client.
			// The client assumes Base64Data is clean.
			decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(file.Base64Data))
			if _, err := io.Copy(partWriter, decoder); err != nil {
				pipeWriter.CloseWithError(fmt.Errorf("failed to write file data for %s: %w", file.FileName, err))
				return
			}
		}
	}()

	return pipeReader, mpWriter.FormDataContentType(), nil
}

// readStreamResponse processes an SSE stream from the response body.
func (c *llmClient) readStreamResponse(ctx context.Context, body io.ReadCloser, callback EventCallback) (string, error) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // 64KB initial, 8MB max token size

	// Define a custom split function for SSE.
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
			return i + 2, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	var fullResponseBuilder strings.Builder
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			eventBlock := scanner.Bytes()
			if !bytes.HasPrefix(eventBlock, []byte("data: ")) {
				continue
			}
			jsonPayload := bytes.TrimPrefix(eventBlock, []byte("data: "))
			if len(jsonPayload) == 0 {
				continue
			}

			var rawEvent models.StreamEvent
			if err := json.Unmarshal(jsonPayload, &rawEvent); err != nil {
				log.Printf("!!! SSE JSON parsing error: %v. Payload: %s", err, string(jsonPayload))
				continue
			}

			callback(rawEvent.Type, rawEvent.Data)

			if rawEvent.Type == "chunk" {
				if dataMap, ok := rawEvent.Data.(map[string]interface{}); ok {
					if text, ok := dataMap["text"].(string); ok {
						fullResponseBuilder.WriteString(text)
					}
				}
			} else if rawEvent.Type == "error" {
				if dataMap, ok := rawEvent.Data.(map[string]interface{}); ok {
					if msg, ok := dataMap["message"].(string); ok {
						return fullResponseBuilder.String(), fmt.Errorf("error from Python service: %s", msg)
					}
				}
				return fullResponseBuilder.String(), fmt.Errorf("received an unspecified error from Python service")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream from Python service: %w", err)
	}
	return fullResponseBuilder.String(), nil
}

// isRetryableStatusCode checks if an HTTP status code indicates a transient error.
func isRetryableStatusCode(code int) bool {
	switch code {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}
