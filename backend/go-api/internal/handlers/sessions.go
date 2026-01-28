package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"egobackend/internal/database"
	"egobackend/internal/models"
	"egobackend/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	defaultSessionTitle      = "New Chat"
	defaultSessionMode       = "default"
	maxUploadSize            = 100 * 1024 * 1024 // 100 MB
	maxMemoryWhenParsingForm = 32 * 1024 * 1024  // 32 MB
)

// SessionHandler handles all HTTP requests related to chat sessions.
type SessionHandler struct {
	DB        *database.DB
	S3Service *storage.S3Service
	LLMClient interface {
		DeleteSessionVectors(ctx context.Context, userID, sessionUUID string) error
		ClearMemory(ctx context.Context, userID string) error
	}
}

// CreateSession creates a new chat session for the authenticated user.
func (h *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	log.Printf("[CreateSession] Request from user ID %d (username: %s)", user.ID, user.Username)

	var req models.CreateSessionRequest
	// It's safe to ignore EOF error for empty body.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	title := defaultSessionTitle
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		title = *req.Title
	}
	mode := defaultSessionMode
	if req.Mode != "" {
		mode = req.Mode
	}

	session, _, err := h.DB.GetOrCreateSession("", title, user.ID, mode)
	if err != nil {
		log.Printf("[CreateSession] DB error: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	if req.CustomInstructions != nil {
		if err := h.DB.UpdateSessionInstructions(session.UUID, user.ID, *req.CustomInstructions); err != nil {
			log.Printf("[CreateSession] Failed to set custom instructions for %s: %v", session.UUID, err)
			// Non-fatal error.
		} else {
			session.CustomInstructions = req.CustomInstructions
		}
	}

	RespondWithJSON(w, http.StatusCreated, models.ToSessionResponse(session))
}

// UpdateSession modifies the title or custom instructions of an existing session.
func (h *SessionHandler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	sessionUUID := chi.URLParam(r, "sessionID")

	var req models.UpdateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Title == nil && req.CustomInstructions == nil {
		RespondWithError(w, http.StatusBadRequest, "At least one field (title or custom_instructions) must be provided for update")
		return
	}

	owns, err := h.DB.CheckSessionOwnership(sessionUUID, user.ID)
	if err != nil || !owns {
		RespondWithError(w, http.StatusNotFound, "Session not found or access denied")
		return
	}

	if req.Title != nil {
		if err := h.DB.UpdateSessionTitle(sessionUUID, user.ID, *req.Title); err != nil {
			log.Printf("Failed to update title for session %s: %v", sessionUUID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to update session title")
			return
		}
	}
	if req.CustomInstructions != nil {
		if err := h.DB.UpdateSessionInstructions(sessionUUID, user.ID, *req.CustomInstructions); err != nil {
			log.Printf("Failed to update instructions for session %s: %v", sessionUUID, err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to update session instructions")
			return
		}
	}

	updatedSession, err := h.DB.GetSessionByUUID(sessionUUID, user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to fetch updated session")
		return
	}

	RespondWithJSON(w, http.StatusOK, models.ToSessionResponse(updatedSession))
}

// GetSessions retrieves a list of all chat sessions for the authenticated user.
func (h *SessionHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	sessions, err := h.DB.GetUserSessions(user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve sessions")
		return
	}
	RespondWithJSON(w, http.StatusOK, models.ToSessionResponseList(sessions))
}

// GetHistory retrieves the message history for a specific session.
func (h *SessionHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	sessionUUID := chi.URLParam(r, "sessionID")
	log.Printf("[GetHistory] User ID %d (username: %s) requesting history for session %s", user.ID, user.Username, sessionUUID)

	owns, err := h.DB.CheckSessionOwnership(sessionUUID, user.ID)
	if err != nil || !owns {
		log.Printf("[GetHistory] Access denied: User ID %d does not own session %s (owns=%v, err=%v)", user.ID, sessionUUID, owns, err)
		RespondWithError(w, http.StatusNotFound, "Session not found or access denied")
		return
	}

	logs, attachmentsMap, err := h.DB.GetSessionHistory(sessionUUID, 50)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve history")
		return
	}

	RespondWithJSON(w, http.StatusOK, models.ToLogResponseList(logs, attachmentsMap))
}

// DeleteSession deletes a specific chat session and all its associated data.
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	sessionUUID := chi.URLParam(r, "sessionID")

	if err := h.DB.DeleteSession(sessionUUID, user.ID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete session")
		return
	}

	// Trigger background deletion of memory vectors for this session
	if h.LLMClient != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := h.LLMClient.DeleteSessionVectors(ctx, fmt.Sprintf("%d", user.ID), sessionUUID); err != nil {
				log.Printf("[DeleteSession] Failed to delete memory vectors for session %s: %v", sessionUUID, err)
			} else {
				log.Printf("[DeleteSession] Successfully deleted memory vectors for session %s", sessionUUID)
			}
		}()
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSession retrieves the details of a single chat session.
func (h *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	sessionUUID := chi.URLParam(r, "sessionID")
	log.Printf("[GetSession] User ID %d (username: %s) requesting session %s", user.ID, user.Username, sessionUUID)

	session, err := h.DB.GetSessionByUUID(sessionUUID, user.ID)
	if err != nil {
		log.Printf("[GetSession] Error fetching session %s for user %d: %v", sessionUUID, user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Server error")
		return
	}
	if session == nil {
		log.Printf("[GetSession] Session %s not found or access denied for user ID %d", sessionUUID, user.ID)
		RespondWithError(w, http.StatusNotFound, "Session not found or access denied")
		return
	}
	RespondWithJSON(w, http.StatusOK, models.ToSessionResponse(session))
}

// PreviewAttachment serves a preview of an attached file, typically a small image.
func (h *SessionHandler) PreviewAttachment(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	logID, err := parseIDFromURL(r, "logID")
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid log ID")
		return
	}
	fileName, err := url.PathUnescape(chi.URLParam(r, "fileName"))
	if err != nil || fileName == "" {
		RespondWithError(w, http.StatusBadRequest, "Invalid file name")
		return
	}

	att, err := h.DB.GetAttachmentOwnedByUser(int(logID), user.ID, fileName)
	if err != nil || att == nil {
		RespondWithError(w, http.StatusNotFound, "Attachment not found or access denied")
		return
	}
	if !strings.HasPrefix(att.MimeType, "image/") {
		RespondWithError(w, http.StatusBadRequest, "Preview is not supported for this file type")
		return
	}

	data, err := h.S3Service.DownloadFile(r.Context(), att.FileURI)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve file from storage")
		return
	}

	w.Header().Set("Content-Type", att.MimeType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Write(data)
}

// ClearAllSessions deletes all sessions for the authenticated user.
func (h *SessionHandler) ClearAllSessions(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)
	if err := h.DB.DeleteAllSessionsForUser(user.ID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete all sessions")
		return
	}

	// Clear the user's profile summary when clearing all sessions
	if err := h.DB.ClearProfileSummary(user.ID); err != nil {
		log.Printf("[ClearAllSessions] Failed to clear user profile summary: %v", err)
		// Non-critical error, continue with memory cleanup
	}

	// Trigger background deletion of ALL memory vectors for this user
	if h.LLMClient != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.LLMClient.ClearMemory(ctx, fmt.Sprintf("%d", user.ID)); err != nil {
				log.Printf("[ClearAllSessions] Failed to clear user memory: %v", err)
			} else {
				log.Printf("[ClearAllSessions] Successfully cleared memory for user %d", user.ID)
			}
		}()
	}

	w.WriteHeader(http.StatusNoContent)
}

// ChatMultipartUpload handles multipart form uploads for chat attachments.
func (h *SessionHandler) ChatMultipartUpload(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	req, files, err := h.parseMultipartRequest(w, r)
	if err != nil {
		// Error response is already sent by the parser.
		return
	}

	sessionUUID, err := h.validateSessionForUpload(req, user.ID)
	if err != nil {
		RespondWithError(w, http.StatusForbidden, err.Error())
		return
	}

	uploadID := uuid.New().String()
	for _, fileHeader := range files {
		if err := h.processAndUploadFile(r.Context(), fileHeader, user, sessionUUID, uploadID); err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"upload_id": uploadID})
}

// --- Private Helpers for ChatMultipartUpload ---

func (h *SessionHandler) parseMultipartRequest(w http.ResponseWriter, r *http.Request) (*models.StreamRequest, []*multipart.FileHeader, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxMemoryWhenParsingForm); err != nil {
		RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid multipart form: %v", err))
		return nil, nil, err
	}

	reqJSON := r.FormValue("request_data")
	var req models.StreamRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil || r.MultipartForm == nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid or missing 'request_data' in multipart form")
		return nil, nil, fmt.Errorf("invalid request_data")
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		RespondWithError(w, http.StatusBadRequest, "No files provided in 'files' field")
		return nil, nil, fmt.Errorf("no files provided")
	}
	return &req, files, nil
}

func (h *SessionHandler) validateSessionForUpload(req *models.StreamRequest, userID int) (string, error) {
	if req.SessionUUID == nil || *req.SessionUUID == "" {
		return "", nil // No session provided, will be created on demand.
	}
	sessionUUID := *req.SessionUUID
	owns, err := h.DB.CheckSessionOwnership(sessionUUID, userID)
	if err != nil {
		return "", fmt.Errorf("server error while checking session ownership")
	}
	if !owns {
		return "", fmt.Errorf("session not found or access denied")
	}
	return sessionUUID, nil
}

func (h *SessionHandler) processAndUploadFile(ctx context.Context, fh *multipart.FileHeader, user *models.User, sessionUUID, uploadID string) error {
	file, err := fh.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file '%s'", fh.Filename)
	}
	defer file.Close()

	// Use a small buffer to sniff the content type without reading the whole file into memory.
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file header for '%s'", fh.Filename)
	}
	// Create a new reader that combines the sniffed buffer and the rest of the file.
	reader := io.MultiReader(bytes.NewReader(buffer[:n]), file)
	mimeType := http.DetectContentType(buffer[:n])

	// Sanitize filename to prevent path traversal and create a unique S3 key.
	safeFilename := filepath.Base(fh.Filename)
	s3Key := fmt.Sprintf("uploads/%d/%s/%s-%s", user.ID, uploadID, uuid.New().String(), safeFilename)

	if err := h.S3Service.UploadStream(ctx, s3Key, mimeType, reader); err != nil {
		return fmt.Errorf("failed to upload file '%s' to storage", fh.Filename)
	}

	if _, err := h.DB.SaveFileAttachment(sessionUUID, user.ID, safeFilename, s3Key, mimeType, "uploaded", &uploadID); err != nil {
		// Best-effort cleanup of S3 object if DB save fails.
		h.S3Service.DeleteFiles(context.Background(), []string{s3Key})
		return fmt.Errorf("failed to save file metadata for '%s'", fh.Filename)
	}
	return nil
}
