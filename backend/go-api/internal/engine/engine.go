// Package engine contains the core business logic for processing user requests,
// orchestrating LLM interactions, executing tools, and managing context.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"egobackend/internal/config"
	"egobackend/internal/crypto"
	"egobackend/internal/database"
	"egobackend/internal/models"
	"egobackend/internal/storage"
	"egobackend/internal/telemetry"
)

// EventCallback is a function type for sending streaming events back to the client.
type EventCallback func(eventType string, data interface{})

// Processor is the central orchestrator for handling user requests.
type Processor struct {
	db            *database.DB
	cfg           *config.AppConfig
	s3Service     *storage.S3Service
	llmClient     *llmClient
	fileProcessor *fileProcessor
}

// NewProcessor creates a new, fully initialized Processor.
func NewProcessor(db *database.DB, cfg *config.AppConfig, s3 *storage.S3Service, client *http.Client) *Processor {
	return &Processor{
		db:        db,
		cfg:       cfg,
		s3Service: s3,

		llmClient:     newLLMClient(cfg.PythonBackendURL, client),
		fileProcessor: newFileProcessor(db, s3),
	}
}

// ProcessRequest is the main entry point for handling a streaming request.
func (p *Processor) ProcessRequest(ctx context.Context, req models.StreamRequest, user *models.User, tempID int64, callback EventCallback) {
	// 1. Preparation
	fullUser, err := p.db.GetUserByID(user.ID)
	if err != nil {
		callback("error", map[string]string{"message": "Error loading user data: " + err.Error()})
		return
	}
	user = fullUser

	go p.llmClient.warmUp(context.Background())
	stats := newRequestStats(ctx, req)
	startTime := time.Now()

	// 2. Session and Log Handling
	session, err := p.getOrCreateSession(req, user, callback)
	if err != nil {
		callback("error", map[string]string{"message": err.Error()})
		return
	}

	logID, err := p.prepareRequestLog(req, user, session)
	if err != nil {
		callback("error", map[string]string{"message": err.Error()})
		return
	}
	callback("log_created", map[string]int64{"temp_id": tempID, "log_id": logID})

	// 3. File and Context Building
	newlyAttachedIDs, err := p.handleFileUploads(ctx, req, user, session, stats)
	if err != nil {
		log.Printf("[Processor] Non-fatal error during file uploads: %v", err)
	}

	contextBuilder := newContextBuilder(p.db, p.s3Service)
	llmContext, err := contextBuilder.build(ctx, user, session, logID, req.IsRegeneration, newlyAttachedIDs, req.Query)
	if err != nil {
		callback("error", map[string]string{"message": "Error building context: " + err.Error()})
		return
	}

	// 4. Agent Execution: Thinking and Synthesis
	thinkingStartTime := time.Now()
	thoughtsHistory, err := p.runThinkingCycle(ctx, user, session, req, llmContext, callback, stats)
	if err != nil {
		log.Printf("[Processor] Warning: thinking cycle failed: %v. Proceeding to synthesis.", err)
	}
	thinkingDuration := time.Since(thinkingStartTime)

	synthesisStartTime := time.Now()
	finalResponse, err := p.synthesizeFinalResponse(ctx, user, session, req, llmContext, thoughtsHistory, callback, logID, stats)
	synthesisDuration := time.Since(synthesisStartTime)

	// 5. Finalization
	p.finalize(logID, session, user, req.Query, finalResponse, thoughtsHistory, newlyAttachedIDs, stats, startTime, thinkingDuration, synthesisDuration, err, callback)
}

// --- Private Helper Methods for Orchestration ---

func (p *Processor) getOrCreateSession(req models.StreamRequest, user *models.User, callback EventCallback) (*models.ChatSession, error) {
	if req.SessionUUID != nil {
		session, err := p.db.GetSessionByUUID(*req.SessionUUID, user.ID)
		if err != nil {
			return nil, fmt.Errorf("error finding session: %w", err)
		}
		if session == nil {
			return nil, fmt.Errorf("session not found or access denied")
		}
		// If the session was pre-created via HTTP, it may still carry the default title.
		// Trigger background title generation based on the first user query.
		// This mirrors the behavior we have when the session is created on-demand here.
		if strings.TrimSpace(session.Title) == "New Chat" && strings.TrimSpace(req.Query) != "" {
			go p.updateTitleAndInstructions(session, req, callback)
		}
		return session, nil
	}

	sessionTitle := createInitialSessionTitle(req)
	session, wasCreated, err := p.db.GetOrCreateSession("", sessionTitle, user.ID, req.Mode)
	if err != nil {
		return nil, fmt.Errorf("error getting or creating session: %w", err)
	}

	if wasCreated {
		callback("session_created", session)
		go p.updateTitleAndInstructions(session, req, callback)
	}
	return session, nil
}

func (p *Processor) prepareRequestLog(req models.StreamRequest, user *models.User, session *models.ChatSession) (int64, error) {
	if !req.IsRegeneration {
		return p.db.CreateInitialRequestLog(session.UUID, req.Query)
	}

	if req.RequestLogIDToRegen == 0 {
		return 0, fmt.Errorf("invalid regeneration request: missing request_log_id_to_regen")
	}

	existingLog, err := p.db.GetRequestLogByID(req.RequestLogIDToRegen, user.ID)
	if err != nil || existingLog == nil || existingLog.SessionUUID != session.UUID {
		return 0, fmt.Errorf("log for regeneration not found or access denied")
	}

	if err := p.db.DeleteLogsAfter(session.UUID, req.RequestLogIDToRegen, user.ID); err != nil {
		log.Printf("[Processor] Failed to trim logs after %d for session %s: %v", req.RequestLogIDToRegen, session.UUID, err)
	}

	if strings.TrimSpace(req.Query) != "" && req.Query != existingLog.UserQuery {
		p.db.UpdateRequestLogQuery(req.RequestLogIDToRegen, user.ID, req.Query)
	}

	return req.RequestLogIDToRegen, nil
}

func (p *Processor) handleFileUploads(ctx context.Context, req models.StreamRequest, user *models.User, session *models.ChatSession, stats *requestStats) ([]int64, error) {
	var allAttachedIDs []int64
	if len(req.Files) > 0 {
		ids, err := p.fileProcessor.uploadAndSaveFiles(ctx, req.Files, user, session.UUID)
		if err != nil {
			return nil, err
		}
		allAttachedIDs = append(allAttachedIDs, ids...)

		// Approximate upload bytes from base64 payloads
		for _, f := range req.Files {
			if n := len(f.Base64Data); n > 0 {
				stats.filesBytes += int64(float64(n) * 0.75)
			}
		}
	}

	if req.UploadID != nil && *req.UploadID != "" {
		ids, err := p.db.GetFileIDsByUploadID(*req.UploadID, user.ID, session.UUID)
		if err != nil {
			log.Printf("[Processor] Failed to get files by upload_id %s: %v", *req.UploadID, err)
		} else {
			allAttachedIDs = append(allAttachedIDs, ids...)
		}
	}
	return allAttachedIDs, nil
}

func (p *Processor) runThinkingCycle(ctx context.Context, user *models.User, session *models.ChatSession, req models.StreamRequest, llmCtx *llmContext, callback EventCallback, stats *requestStats) ([]map[string]interface{}, error) {
	thinkingCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var thoughtsHistory []map[string]interface{}
	callback("thought_header", map[string]string{"header": "Thinking..."})

	consecutiveErrors := 0
	maxConsecutiveErrors := 2

	for i := 0; i < p.cfg.MaxThinkingIterations; i++ {
		select {
		case <-thinkingCtx.Done():
			if len(thoughtsHistory) > 0 {
				log.Printf("[Processor] Thinking timeout after %d iterations, proceeding with partial thoughts", i)
				callback("thought_header", map[string]string{"header": "Switching to response..."})
				return thoughtsHistory, nil
			}
			return thoughtsHistory, thinkingCtx.Err()
		default:
		}

		iterationStartTime := time.Now()
		pythonReq := p.buildPythonRequest(user, session, req.Query, llmCtx, thoughtsHistory, false, 0)
		// Send current request inline files (with Base64) so Python receives actual image bytes.
		thoughtData, err := p.llmClient.generateThought(thinkingCtx, pythonReq, req.Files)

		if err != nil {
			consecutiveErrors++
			log.Printf("[Processor] Thought iteration %d error: %v (consecutive: %d)", i+1, err, consecutiveErrors)

			if consecutiveErrors >= maxConsecutiveErrors {
				log.Printf("[Processor] Too many consecutive errors (%d), switching to direct response", consecutiveErrors)
				callback("thought_header", map[string]string{"header": "Switching to direct response..."})
				return thoughtsHistory, nil
			}

			// Retry с exponential backoff
			backoffDelay := time.Duration(consecutiveErrors) * time.Second
			log.Printf("[Processor] Waiting %v before retry...", backoffDelay)
			time.Sleep(backoffDelay)
			continue
		}

		consecutiveErrors = 0
		stats.thoughtDurations = append(stats.thoughtDurations, time.Since(iterationStartTime).Milliseconds())

		// Process the thought first (add to history, send header)
		p.processThoughtOnly(thoughtData, &thoughtsHistory, callback, stats)

		// If tools are needed, pause thinking and execute them
		if len(thoughtData.ToolResults) > 0 || len(thoughtData.Thought.ToolCalls) > 0 {
			p.executeToolsAndAddToHistory(thoughtData, &thoughtsHistory, callback, pythonReq.MemoryEnabled, user.ID)
		}

		if !thoughtData.Thought.NextThoughtNeeded {
			break
		}
	}
	return thoughtsHistory, nil
}

func (p *Processor) synthesizeFinalResponse(ctx context.Context, user *models.User, session *models.ChatSession, req models.StreamRequest, llmCtx *llmContext, thoughtsHistory []map[string]interface{}, callback EventCallback, logID int64, stats *requestStats) (string, error) {
	callback("thought_header", map[string]string{"header": "Synthesizing response..."})
	pythonReq := p.buildPythonRequest(user, session, req.Query, llmCtx, thoughtsHistory, true, logID)

	if req.IsRegeneration {
		pythonReq.RegeneratedLogID = logID
	}

	// Pass current request inline files (with Base64) for synthesis stream as well.
	return p.llmClient.synthesizeStream(ctx, pythonReq, req.Files, callback)
}

func (p *Processor) finalize(logID int64, session *models.ChatSession, user *models.User, query, finalResponse string, thoughtsHistory []map[string]interface{}, newlyAttachedIDs []int64, stats *requestStats, startTime time.Time, thinkingDur, synthDur time.Duration, processErr error, callback EventCallback) {
	if processErr != nil {
		if processErr != context.Canceled {
			callback("error", map[string]string{"message": "Synthesis error: " + processErr.Error()})
		}
	} else {
		p.finalizeLog(logID, thoughtsHistory, finalResponse, newlyAttachedIDs)
		callback("done", "Processing complete.")
		// Отложенная векторизация - выполняется ПОСЛЕ отправки ответа пользователю
		go p.triggerVectorization(logID, query, finalResponse, newlyAttachedIDs)
	}

	metrics := stats.buildMetrics(user, session.UUID, logID, startTime, thinkingDur, synthDur)
	go func() {
		if err := p.db.SaveRequestMetrics(metrics); err != nil {
			log.Printf("!!! ERROR saving request metrics: %v", err)
		}
	}()

	if telemetry.GetBotInstance() != nil {
		msg := stats.composeTelemetryMessage(user, session.UUID, startTime, thinkingDur, synthDur, processErr == nil)
		telemetry.Send(msg)
	}
}

func (p *Processor) finalizeLog(logID int64, thoughtsHistory []map[string]interface{}, finalResponse string, newlyAttachedIDs []int64) {
	thoughtsJSON, _ := json.Marshal(thoughtsHistory)
	filesJSON, _ := json.Marshal(newlyAttachedIDs)

	// Token counts are now collected in the stats object, but we are not passing them here.
	// This can be updated if the DB schema needs it.
	err := p.db.UpdateLogWithFinalResponse(logID, string(thoughtsJSON), finalResponse, string(filesJSON), 0, 0, 0)
	if err != nil {
		log.Printf("!!! CRITICAL ERROR: Failed to update final log %d in DB: %v", logID, err)
	}

	if len(newlyAttachedIDs) > 0 {
		if err := p.db.AssociateFilesWithRequestLog(logID, newlyAttachedIDs); err != nil {
			log.Printf("!!! ERROR: Failed to associate files with log %d: %v", logID, err)
		}
	}
}

func (p *Processor) triggerVectorization(logID int64, query, finalResponse string, fileIDs []int64) {
	// Векторизация файлов и сообщений выполняется в фоне, не блокируя ответ пользователю
	// Увеличен таймаут для больших файлов
	if len(fileIDs) > 0 {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // Увеличен с 5 до 10 минут
			defer cancel()
			log.Printf("[VECTORS] Starting vectorization of %d files for log %d", len(fileIDs), logID)
			if err := p.fileProcessor.vectorizeFiles(ctx, fileIDs); err != nil {
				log.Printf("[VECTORS] Error vectorizing files for log %d: %v", logID, err)
			} else {
				log.Printf("[VECTORS] Successfully vectorized %d files for log %d", len(fileIDs), logID)
			}
		}()
	}

	go func() {
		combinedText := fmt.Sprintf("User: %s\nEGO: %s", query, finalResponse)
		if err := p.fileProcessor.vectorizeAndSaveMessage(logID, combinedText); err != nil {
			log.Printf("[VECTORS] Error vectorizing message for log %d: %v", logID, err)
		} else {
			log.Printf("[VECTORS] Successfully vectorized message for log %d", logID)
		}
	}()
}

// processThoughtOnly handles just the thought processing without tools
func (p *Processor) processThoughtOnly(thoughtData *models.ThoughtResponseWithData, thoughtsHistory *[]map[string]interface{}, callback EventCallback, stats *requestStats) {
	stats.updateFromUsage(thoughtData.Usage)
	if thoughtData.Usage != nil {
		callback("usage_update", thoughtData.Usage)
	}

	thought := thoughtData.Thought
	log.Printf("[DEBUG] processThoughtOnly - ThoughtHeader: '%s'", thought.ThoughtHeader)
	log.Printf("[DEBUG] processThoughtOnly - Thoughts: '%s'", thought.Thoughts[:min(100, len(thought.Thoughts))])

	*thoughtsHistory = append(*thoughtsHistory, map[string]interface{}{
		"type": "thought", "content": thought.Thoughts,
	})

	if thought.ThoughtHeader != "" {
		log.Printf("[DEBUG] Sending thought_header event: '%s'", thought.ThoughtHeader)
		callback("thought_header", map[string]string{"header": thought.ThoughtHeader})
	} else {
		log.Printf("[DEBUG] ThoughtHeader is empty, not sending event")
	}
}

// executeToolsAndAddToHistory handles tool execution separately from thinking
func (p *Processor) executeToolsAndAddToHistory(thoughtData *models.ThoughtResponseWithData, thoughtsHistory *[]map[string]interface{}, callback EventCallback, memEnabled bool, userID int) {
	// Handle tool results from Python backend (tools are now executed in Python)
	if len(thoughtData.ToolResults) > 0 {
		log.Printf("[DEBUG] Processing %d tool results from Python backend", len(thoughtData.ToolResults))

		for _, result := range thoughtData.ToolResults {
			// Handle tool progress updates with dynamic headers
			if result["type"] == "tool_progress" {
				if header, ok := result["header"].(string); ok {
					callback("thought_header", map[string]string{"header": header})
				}
			} else {
				callback(result["type"].(string), result)
			}
		}
		*thoughtsHistory = append(*thoughtsHistory, thoughtData.ToolResults...)
	} else if len(thoughtData.Thought.ToolCalls) > 0 {
		// Fallback: if Python didn't execute tools, execute them locally (for backward compatibility)
		log.Printf("[DEBUG] Python backend didn't execute tools, falling back to local execution")
		callback("thought_header", map[string]string{"header": "Executing tools..."})
		toolExecutor := newToolExecutor(p.llmClient)
		toolResults := toolExecutor.execute(thoughtData.Thought.ToolCalls, callback, memEnabled, userID)
		*thoughtsHistory = append(*thoughtsHistory, toolResults...)
	}
}

// Legacy function for backward compatibility
func (p *Processor) processThoughtData(thoughtData *models.ThoughtResponseWithData, thoughtsHistory *[]map[string]interface{}, callback EventCallback, memEnabled bool, userID int, stats *requestStats) {
	p.processThoughtOnly(thoughtData, thoughtsHistory, callback, stats)
	p.executeToolsAndAddToHistory(thoughtData, thoughtsHistory, callback, memEnabled, userID)
}

func (p *Processor) buildPythonRequest(user *models.User, session *models.ChatSession, query string, llmCtx *llmContext, thoughtsHistory []map[string]interface{}, isSynthesis bool, logID int64) models.PythonRequest {
	req := models.PythonRequest{
		Query:              query,
		Mode:               session.Mode,
		ChatHistory:        llmCtx.chatHistory,
		ThoughtsHistory:    mustMarshal(thoughtsHistory),
		CustomInstructions: session.CustomInstructions,
		RetrievedSnippets:  llmCtx.retrievedSnippets,
		UserID:             fmt.Sprintf("%d", user.ID),
		SessionUUID:        session.UUID,
		LogID:              logID,
		SessionCreatedAt:   session.CreatedAt.Format(time.RFC3339),
		CachedFiles:        llmCtx.cachedFiles,
		MemoryEnabled:      true, // Default
	}

	if user.LLMProvider != nil && user.EncryptedAPIKey != nil {
		if key, err := crypto.Decrypt(*user.EncryptedAPIKey, p.cfg.APIEncryptionKey); err == nil {
			req.LLMSettings = &models.LLMSettings{Provider: *user.LLMProvider, APIKey: key}
			if user.LLMModel != nil {
				req.LLMSettings.Model = *user.LLMModel
			}
		}
	}
	return req
}

func (p *Processor) updateTitleAndInstructions(session *models.ChatSession, req models.StreamRequest, callback EventCallback) {
	if req.CustomInstructions != nil && *req.CustomInstructions != "" {
		if err := p.db.UpdateSessionInstructions(session.UUID, session.UserID, *req.CustomInstructions); err != nil {
			log.Printf("!!! ERROR: Failed to save instructions for new session %s: %v", session.UUID, err)
		} else {
			session.CustomInstructions = req.CustomInstructions
		}
	}

	if newTitle, err := p.llmClient.generateTitle(context.Background(), req.Query); err == nil && newTitle != "" && newTitle != session.Title {
		if err := p.db.UpdateSessionTitle(session.UUID, session.UserID, newTitle); err == nil {
			callback("session_title_updated", map[string]string{"uuid": session.UUID, "title": newTitle})
		}
	}
}

// --- Utility Functions ---

func mustMarshal(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		log.Printf("!!! CRITICAL: Failed to marshal object for history: %v", err)
		return "[]"
	}
	return string(bytes)
}

func createInitialSessionTitle(req models.StreamRequest) string {
	var title string
	if req.Query != "" {
		title = req.Query
	} else if len(req.Files) > 0 {
		var names []string
		for _, f := range req.Files {
			names = append(names, truncateFilename(f.FileName, 15))
		}
		title = strings.Join(names, ", ")
	}

	if title == "" {
		return "New Chat"
	}

	if utf8.RuneCountInString(title) > 50 {
		return string([]rune(title)[:47]) + "..."
	}
	return title
}

func truncateFilename(filename string, maxLen int) string {
	if utf8.RuneCountInString(filename) <= maxLen {
		return filename
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if utf8.RuneCountInString(base) <= 5 {
		return string([]rune(filename)[:maxLen-3]) + "..."
	}
	availableLen := maxLen - utf8.RuneCountInString(ext) - 3
	if availableLen < 1 {
		return string([]rune(filename)[:maxLen-3]) + "..."
	}
	truncatedBase := string([]rune(base)[:availableLen])
	return truncatedBase + "..." + ext
}
