package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
	"encoding/base64"

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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second) // Generous timeout for a quick task.
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

// generateThought calls the /generate_thought endpoint. It's a non-streaming endpoint.
func (c *llmClient) generateThought(ctx context.Context, requestData models.PythonRequest, files []models.FilePayload) (*models.ThoughtResponseWithData, error) {
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

		// Read the entire body since this is a non-streaming endpoint.
		responseBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("failed to read thought response body: %w", readErr)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var thoughtResponse models.ThoughtResponseWithData
			if err := json.Unmarshal(responseBody, &thoughtResponse); err != nil {
				return nil, fmt.Errorf("failed to unmarshal thought response: %w", err)
			}
			return &thoughtResponse, nil
		}

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