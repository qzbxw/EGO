package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"egobackend/internal/models"
)

// EditLog handles the request to update the query of an existing log entry.
// When a log is edited, it also truncates all subsequent logs in the same session
// to maintain a coherent conversation history for future generations.
func (h *SessionHandler) EditLog(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	logID, err := parseIDFromURL(r, "logID")
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid log ID format")
		return
	}

	var req models.UpdateLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Query) == "" {
		RespondWithError(w, http.StatusBadRequest, "Query cannot be empty")
		return
	}

	// Verify that the user owns the log before proceeding.
	logToEdit, err := h.DB.GetRequestLogByID(logID, user.ID)
	if err != nil {
		log.Printf("Database error while checking log ownership for user %d, log %d: %v", user.ID, logID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to verify log ownership")
		return
	}
	if logToEdit == nil {
		RespondWithError(w, http.StatusNotFound, "Log not found or access denied")
		return
	}

	// Update the user's query in the specified log.
	if err := h.DB.UpdateRequestLogQuery(logID, user.ID, req.Query); err != nil {
		log.Printf("Failed to update log %d for user %d: %v", logID, user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update log")
		return
	}

	// After editing, delete all subsequent logs in the session to ensure history consistency.
	// This is a "fire-and-forget" operation from the client's perspective; an error here
	// should not fail the entire request, but it must be logged.
	if err := h.DB.DeleteLogsAfter(logToEdit.SessionUUID, logID, user.ID); err != nil {
		log.Printf("[HANDLERS] CRITICAL: Failed to truncate logs after edit (log %d, session %s): %v", logID, logToEdit.SessionUUID, err)
	}

	w.WriteHeader(http.StatusOK)
}

// GetLog retrieves a single log entry by its ID.
// This is primarily used by the frontend to fetch log details, for example,
// after a WebSocket reconnects, to ensure the UI is in sync with the state.
func (h *SessionHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	logID, err := parseIDFromURL(r, "logID")
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid log ID format")
		return
	}

	logEntry, err := h.DB.GetRequestLogByID(logID, user.ID)
	if err != nil {
		log.Printf("Database error while fetching log %d for user %d: %v", logID, user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve log")
		return
	}
	if logEntry == nil {
		RespondWithError(w, http.StatusNotFound, "Log not found or access denied")
		return
	}

	// It's recommended to move this response structure to the `models` package
	// to maintain a clear API contract, e.g., models.LogDetailResponse.
	type logDetailResponse struct {
		ID            int     `json:"id"`
		FinalResponse *string `json:"final_response"`
		UserQuery     string  `json:"user_query"`
		SessionUUID   string  `json:"session_uuid"`
	}

	resp := logDetailResponse{
		ID:            logEntry.ID,
		FinalResponse: logEntry.FinalResponse,
		UserQuery:     logEntry.UserQuery,
		SessionUUID:   logEntry.SessionUUID,
	}
	RespondWithJSON(w, http.StatusOK, resp)
}
