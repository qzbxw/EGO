package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"egobackend/internal/models"

	"github.com/google/uuid"
)

// UploadHandler handles file uploads separately from the chat stream.
type UploadHandler struct {
	*ChatHandler // Inherit common dependencies
}

// NewUploadHandler creates a new UploadHandler.
func NewUploadHandler(ch *ChatHandler) *UploadHandler {
	return &UploadHandler{ChatHandler: ch}
}

// HandleUpload handles POST /api/upload
func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 1. Authentication Check
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Parse Multipart Form
	// Limit upload size (e.g., 50MB per file, max 100MB total request)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "File too large or invalid format", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	var uploadedFiles []map[string]string

	// 3. Process each file
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Read first 512 bytes for MimeType detection
		head := make([]byte, 512)
		n, _ := io.ReadFull(file, head)
		mimeType := http.DetectContentType(head[:n])

		// If detection fails or is generic, trust the header extension or header type
		if mimeType == "application/octet-stream" {
			if fileHeader.Header.Get("Content-Type") != "" {
				mimeType = fileHeader.Header.Get("Content-Type")
			}
		}

		// Prepare full stream (head + rest)
		fullStream := io.MultiReader(bytes.NewReader(head[:n]), file)

		sessionUUID := r.FormValue("session_uuid")
		if sessionUUID == "" {
			sessionUUID = "temp_" + uuid.New().String()
		}

		// Generate a unique upload ID for this file
		uploadID := uuid.New().String()

		fileURI, err := h.Processor.UploadFileRaw(r.Context(), fullStream, fileHeader.Filename, mimeType, user, sessionUUID, uploadID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to upload file %s: %v", fileHeader.Filename, err), http.StatusInternalServerError)
			return
		}

		uploadedFiles = append(uploadedFiles, map[string]string{
			"upload_id": uploadID,
			"file_name": fileHeader.Filename,
			"file_uri":  fileURI,
			"mime_type": mimeType,
		})
	}

	// 4. Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": uploadedFiles,
	})
}
