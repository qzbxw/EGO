package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"egobackend/internal/database"
	"egobackend/internal/models"
	"egobackend/internal/storage"
)

const (
	// Defines the maximum number of historical messages to fetch for context.
	historyLimit = 10
	// Defines the number of relevant snippets to retrieve for RAG.
	retrievalTopK = 5
	// A high similarity score threshold to fetch only the most relevant cross-session snippets.
	highRelevanceMinScore = 0.8
)

// llmContext holds all the contextual information required by the LLM to generate a response.
type llmContext struct {
	chatHistory       string
	retrievedSnippets []string
	allFiles          []models.FilePayload
	cachedFiles       []models.CachedFile
	effectiveHistory  []models.RequestLog // The raw logs used to build the context.
	currentPlan       *models.SessionPlan // The active plan for the session, if any.
	userProfile       string              // The persistent user profile summary.
}

// contextBuilder is responsible for gathering and assembling the context for the LLM.
type contextBuilder struct {
	db        *database.DB
	s3Service *storage.S3Service
}

// newContextBuilder creates a new context builder.
func newContextBuilder(db *database.DB, s3 *storage.S3Service) *contextBuilder {
	return &contextBuilder{db: db, s3Service: s3}
}

// build orchestrates the entire context gathering process.
func (cb *contextBuilder) build(ctx context.Context, user *models.User, session *models.ChatSession, logID int64, isRegen bool, newlyAttachedIDs []int64, query string) (*llmContext, error) {
	// 1. Fetch raw historical logs from the database.
	historyLogs, attachmentsMap, err := cb.db.GetSessionHistory(session.UUID, historyLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to load history: %w", err)
	}

	// 2. Prepare the "effective" history, handling the regeneration case.
	effectiveHistory := cb.prepareEffectiveHistory(historyLogs, logID, isRegen)

	// 3. Format the chat history into a string for the LLM.
	chatHistoryString := cb.formatChatHistory(effectiveHistory, attachmentsMap)
	log.Printf("[DEBUG] Final chat history length: %d characters", len(chatHistoryString))

	// 4. Retrieve relevant snippets using vector search (RAG).
	retrievedSnippets, err := cb.retrieveSnippets(ctx, user, session.UUID, query)
	if err != nil {
		// RAG failure is non-critical; log the error and continue without snippets.
		log.Printf("[ContextBuilder] Snippet retrieval failed, proceeding without them: %v", err)
	}

	// 5. Gather information about all files (new and historical) for the request.
	allFiles, cachedFiles, err := cb.gatherAllFileInfo(effectiveHistory, attachmentsMap, newlyAttachedIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to gather file payloads: %w", err)
	}

	// 6. Fetch the active plan for this session.
	activePlan, err := cb.db.GetActivePlan(session.UUID)
	if err != nil {
		log.Printf("[ContextBuilder] Failed to fetch active plan (non-critical): %v", err)
	}

	return &llmContext{
		chatHistory:       chatHistoryString,
		retrievedSnippets: retrievedSnippets,
		allFiles:          allFiles,
		cachedFiles:       cachedFiles,
		effectiveHistory:  effectiveHistory,
		currentPlan:       activePlan,
		userProfile:       user.ProfileSummary,
	}, nil
}

// prepareEffectiveHistory adjusts the list of historical logs based on whether it's a new message or a regeneration.
func (cb *contextBuilder) prepareEffectiveHistory(historyLogs []models.RequestLog, logID int64, isRegen bool) []models.RequestLog {
	log.Printf("[DEBUG] prepareEffectiveHistory: logID=%d, isRegen=%v, historyLogs count=%d", logID, isRegen, len(historyLogs))

	if !isRegen {
		// For a new message, filter out the (currently empty) log entry we just created.
		filtered := make([]models.RequestLog, 0, len(historyLogs))
		for _, logEntry := range historyLogs {
			log.Printf("[DEBUG] Checking log ID=%d vs current logID=%d", logEntry.ID, logID)
			if int64(logEntry.ID) != logID {
				filtered = append(filtered, logEntry)
				log.Printf("[DEBUG] Including log ID=%d in history", logEntry.ID)
			} else {
				log.Printf("[DEBUG] Filtering out current log ID=%d", logEntry.ID)
			}
		}
		log.Printf("[DEBUG] Filtered history: %d logs", len(filtered))
		return filtered
	}

	// For regeneration, completely exclude the target log from the history.
	// The user query from this log will be passed as the new query.
	// We want the LLM to treat it as if it's answering this query for the first time
	// given the *previous* context.
	filtered := make([]models.RequestLog, 0, len(historyLogs))
	for _, logEntry := range historyLogs {
		if int64(logEntry.ID) != logID {
			filtered = append(filtered, logEntry)
		}
	}
	return filtered
}

// formatChatHistory converts a slice of log entries into a single string for the LLM prompt.
func (cb *contextBuilder) formatChatHistory(logs []models.RequestLog, attachmentsMap map[int][]models.FileAttachment) string {
	var builder strings.Builder
	for _, logEntry := range logs {
		finalResponse := ""
		if logEntry.FinalResponse != nil {
			finalResponse = *logEntry.FinalResponse
		}

		attachmentsText := ""
		if attachments, ok := attachmentsMap[logEntry.ID]; ok && len(attachments) > 0 {
			var names []string
			for _, a := range attachments {
				names = append(names, a.FileName)
			}
			attachmentsText = fmt.Sprintf(" [Attached: %s]", strings.Join(names, ", "))
		}

		// Include thoughts history with tool results for context continuity
		thoughtsContext := ""
		if logEntry.EgoThoughtsJSON != "" && logEntry.EgoThoughtsJSON != "[]" && logEntry.EgoThoughtsJSON != "null" {
			// Parse thoughts to extract tool outputs for context
			var thoughts []map[string]interface{}
			if err := json.Unmarshal([]byte(logEntry.EgoThoughtsJSON), &thoughts); err == nil {
				var toolOutputs []string
				for _, thought := range thoughts {
					if thoughtType, ok := thought["type"].(string); ok && thoughtType == "tool_output" {
						if toolName, ok := thought["tool_name"].(string); ok {
							if output, ok := thought["output"].(string); ok && output != "" {
								toolOutputs = append(toolOutputs, fmt.Sprintf("[%s: %s]", toolName, truncateString(output, 150)))
							}
						}
					}
				}
				if len(toolOutputs) > 0 {
					thoughtsContext = fmt.Sprintf(" [Tools used: %s]", strings.Join(toolOutputs, "; "))
				}
			}
		}

		// Using E'\n' to be explicit about newlines for robustness.
		builder.WriteString(fmt.Sprintf("User: %s%s\nEGO: %s%s\n\n", logEntry.UserQuery, attachmentsText, finalResponse, thoughtsContext))
	}
	return builder.String()
}

// truncateString returns first n characters of s, or all of s if shorter
func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// retrieveSnippets performs vector similarity search to find relevant context (RAG).
func (cb *contextBuilder) retrieveSnippets(ctx context.Context, user *models.User, sessionUUID string, query string) ([]string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	// The retrieval logic lives in the engine package now.
	retriever := newRetriever(cb.db)

	// First, embed the user's query into a vector.
	qVec, err := localEmbedTextFallback(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query for retrieval: %w", err)
	}

	// Then, search for highly relevant snippets using the vector.
	highlyRelevant, err := retriever.SearchTopSnippetsByUserWithScore(user.ID, qVec, retrievalTopK, highRelevanceMinScore)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve snippets: %w", err)
	}

	return highlyRelevant, nil
}

// gatherAllFileInfo collects details about newly uploaded files and historical attachments.
// It prepares two lists: one for inline transport (allFiles) and one for reference (cachedFiles).
func (cb *contextBuilder) gatherAllFileInfo(historyLogs []models.RequestLog, attachmentsMap map[int][]models.FileAttachment, newlyAttachedIDs []int64) ([]models.FilePayload, []models.CachedFile, error) {
	log.Printf("[DEBUG] gatherAllFileInfo called with %d history logs, %d attachment groups, %d newly attached IDs", len(historyLogs), len(attachmentsMap), len(newlyAttachedIDs))
	var allFiles []models.FilePayload
	var cachedFiles []models.CachedFile
	processedURIs := make(map[string]bool)

	// Add historical files from previous messages.
	for logID, attachments := range attachmentsMap {
		log.Printf("[DEBUG] Processing attachments for log %d: %d files", logID, len(attachments))
		for _, att := range attachments {
			if processedURIs[att.FileURI] {
				continue
			}
			log.Printf("[DEBUG] Adding historical file to cachedFiles: name='%s', uri='%s', mime='%s'", att.FileName, att.FileURI, att.MimeType)
			cachedFile := models.CachedFile{
				URI:      att.FileURI,
				FileName: att.FileName,
				MimeType: att.MimeType,
			}
			cachedFiles = append(cachedFiles, cachedFile)
			processedURIs[att.FileURI] = true
		}
	}

	// Add newly uploaded files from the current request.
	if len(newlyAttachedIDs) > 0 {
		log.Printf("[DEBUG] Processing %d newly attached files", len(newlyAttachedIDs))
		attachments, err := cb.db.GetAttachmentsByIDs(newlyAttachedIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get newly attached files: %w", err)
		}
		for _, att := range attachments {
			if processedURIs[att.FileURI] {
				continue
			}
			log.Printf("[DEBUG] Adding newly attached file: name='%s', uri='%s', mime='%s'", att.FileName, att.FileURI, att.MimeType)
			// Add to both lists: allFiles for potential inline data, cachedFiles for reference.
			payload := models.FilePayload{
				FileName:   att.FileName,
				MimeType:   att.MimeType,
				Base64Data: "", // To be populated by the file_processor if needed.
			}
			cachedFile := models.CachedFile{
				URI:      fmt.Sprintf("s3://%s/%s", cb.s3Service.BucketName(), att.FileURI),
				FileName: att.FileName,
				MimeType: att.MimeType,
			}
			allFiles = append(allFiles, payload)
			cachedFiles = append(cachedFiles, cachedFile)
			processedURIs[att.FileURI] = true
		}
	}

	return allFiles, cachedFiles, nil
}
