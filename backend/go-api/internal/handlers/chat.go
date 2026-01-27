package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"egobackend/internal/config"
	"egobackend/internal/database"
	"egobackend/internal/engine"
	"egobackend/internal/models"

	"github.com/go-playground/validator/v10"
)

// Global manager for streaming jobs
var streamManager = NewStreamManager()

// ChatHandler handles the HTTP streaming chat requests.
type ChatHandler struct {
	UserDB    *database.DB
	Processor *engine.Processor
	Validate  *validator.Validate
	Cfg       *config.AppConfig
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(db *database.DB, processor *engine.Processor, validate *validator.Validate, cfg *config.AppConfig) *ChatHandler {
	return &ChatHandler{
		UserDB:    db,
		Processor: processor,
		Validate:  validate,
		Cfg:       cfg,
	}
}

type contextKey string

const memoryEnabledKey contextKey = "memory_enabled"

// HandleChatStream processes the streaming chat request.
func (h *ChatHandler) HandleChatStream(w http.ResponseWriter, r *http.Request) {
	// 1. Authentication Check
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Parse and Validate Request Body
	var req models.StreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.Validate.Struct(req); err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// 1. Check for maintenance mode
	maintenance, err := h.UserDB.GetMaintenanceMode()
	if err == nil && maintenance.IsEnabled {
		// If full maintenance, middleware should have caught it, but we check here too.
		// If chat-only maintenance, we block the stream specifically.

		// Note: We check if there's a bypass token in headers
		bypassToken := r.Header.Get("X-Bypass-Token")
		hasBypass := false
		if bypassToken != "" {
			valid, _ := h.UserDB.ValidateBypassToken(bypassToken)
			hasBypass = valid
		}

		if !hasBypass {
			http.Error(w, "AI Chat is temporarily disabled for maintenance.", http.StatusServiceUnavailable)
			return
		}
	}

	// 2. Set Up Streaming Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 4. Create Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	sessionUUID := ""
	if req.SessionUUID != nil && *req.SessionUUID != "new" {
		sessionUUID = *req.SessionUUID
	}

	// 5. Determine Mode: New Job vs. Reconnect
	var job *Job
	isReconnection := false

	// If no query and no files, treat as an attempt to reconnect to an existing stream
	if req.Query == "" && len(req.Files) == 0 && sessionUUID != "" {
		existingJob, exists := streamManager.GetJob(sessionUUID)
		if exists {
			// Security Check: Ensure user owns this job
			if existingJob.UserID != user.ID {
				log.Printf("[ChatStream] Security Warning: User %d attempted to connect to session %s owned by user %d", user.ID, sessionUUID, existingJob.UserID)
				// Respond with 403 Forbidden via HTTP (SSE connection fails)
				http.Error(w, "Forbidden: You do not own this session stream", http.StatusForbidden)
				return
			}
			job = existingJob
			isReconnection = true
			log.Printf("[ChatStream] Reconnecting to existing stream for user %d (Session: %s)", user.ID, sessionUUID)
		} else {
			// No active job found.
			// Send a "done" event so the client knows not to wait.
			fmt.Fprintf(w, "data: {\"type\": \"done\", \"data\": \"No active stream found\"}\n\n")
			flusher.Flush()
			return
		}
	} else {
		// Start a new job
		tempSessionUUID := sessionUUID
		if tempSessionUUID == "" {
			// Generate a temporary ID to track this job until the Processor creates the real session
			tempSessionUUID = fmt.Sprintf("new-%d", time.Now().UnixNano())
		}

		log.Printf("[ChatStream] Starting new stream for user %d (Session: %s)", user.ID, tempSessionUUID)
		job = streamManager.CreateJob(tempSessionUUID, user.ID)
		// We use the actual sessionUUID if it becomes available, but for the manager,
		// we must use the one we used to create the job.
		sessionUUID = tempSessionUUID
	}

	// 6. Subscribe to the job (getting channel + history)
	eventChan, history := job.Subscribe()
	defer job.Unsubscribe(eventChan)

	// 7. Replay History (Crucial for Reconnection/Late Joiners)
	// We send this immediately before entering the loop
	for _, historyEvent := range history {
		fmt.Fprintf(w, "data: %s\n\n", historyEvent)
	}
	flusher.Flush()

	// 8. If NEW job, kick off the processor in a goroutine
	if !isReconnection {
		// Capture the exact ID used to create the job for safe removal later
		jobID := sessionUUID
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in processor: %v", r)
				}
				streamManager.RemoveJob(jobID)
			}()

			// Define callback that broadcasts to the job
			callback := func(eventType string, data interface{}) {
				job.Broadcast(eventType, data)
			}

			// Use the job's context, NOT the request context
			// Pass memory_enabled if needed
			ctx := job.Ctx
			if req.MemoryEnabled != nil {
				ctx = context.WithValue(ctx, memoryEnabledKey, *req.MemoryEnabled)
			}

			// Run the processor
			h.Processor.ProcessRequest(ctx, req, user, req.TempID, callback)
		}()
	}

	// 9. Stream Loop
	// Listen for events from the channel (pushed by the background goroutine)
	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			// Client disconnected
			log.Printf("[ChatStream] Client disconnected for user %d (Session: %s)", user.ID, sessionUUID)
			return

		case msgBytes, ok := <-eventChan:
			if !ok {
				// Channel closed, job finished
				return
			}
			// Forward the exact bytes (already marshaled in Broadcast)
			fmt.Fprintf(w, "data: %s\n\n", msgBytes)
			flusher.Flush()
		}
	}
}
