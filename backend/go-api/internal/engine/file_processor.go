// Package engine contains the core business logic of the application.
// This file specifically handles all logic related to file processing:
// uploading, content extraction, chunking, and vectorization.
package engine

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"path/filepath"
	"regexp"
	"strings"

	"egobackend/internal/database"
	"egobackend/internal/models"
	"egobackend/internal/storage"

	"github.com/google/uuid"
)

// --- Constants for File Processing ---

const (
	perFileLimitBytes     = 50 * 1024 * 1024
	totalUploadLimitBytes = 100 * 1024 * 1024
	maxTextVectorizeBytes = 2 * 1024 * 1024
	maxPDFVectorizeBytes  = 25 * 1024 * 1024
	chunkSize             = 1200
	chunkOverlap          = 200
)

// --- Errors for Size Limits ---

var (
	ErrPerFileTooLarge = errors.New("decoded file exceeds per-file size limit")
	ErrTotalTooLarge   = errors.New("combined decoded files exceed total size limit")
)

// --- sizeCappedReader for Safe Streaming ---

// sizeCappedReader is a wrapper for io.Reader that enforces per-file and total byte limits.
type sizeCappedReader struct {
	r            io.Reader
	perCount     *int64
	totalCount   *int64
	perFileLimit int64
	totalLimit   int64
}

func (s *sizeCappedReader) Read(p []byte) (int, error) {
	perRemain := s.perFileLimit - *s.perCount
	totRemain := s.totalLimit - *s.totalCount
	if perRemain <= 0 {
		return 0, ErrPerFileTooLarge
	}
	if totRemain <= 0 {
		return 0, ErrTotalTooLarge
	}
	maxN := len(p)
	if int64(maxN) > perRemain {
		maxN = int(perRemain)
	}
	if int64(maxN) > totRemain {
		maxN = int(totRemain)
	}
	if maxN <= 0 {
		return 0, ErrTotalTooLarge
	}

	buf := p
	if len(p) > maxN {
		buf = p[:maxN]
	}

	n, err := s.r.Read(buf)
	if n > 0 {
		*s.perCount += int64(n)
		*s.totalCount += int64(n)
	}
	return n, err
}

// --- fileProcessor ---

// fileProcessor handles all logic related to file processing.
type fileProcessor struct {
	db        *database.DB
	s3Service *storage.S3Service
	llmClient *llmClient
}

func newFileProcessor(db *database.DB, s3 *storage.S3Service, llm *llmClient) *fileProcessor {
	return &fileProcessor{db: db, s3Service: s3, llmClient: llm}
}

// uploadAndSaveFiles processes files from a request, uploads them to S3, and saves metadata.
func (fp *fileProcessor) uploadAndSaveFiles(ctx context.Context, files []models.FilePayload, user *models.User, sessionUUID string) ([]int64, error) {
	if fp.s3Service == nil {
		return nil, errors.New("S3 service is not initialized")
	}

	log.Printf("[DEBUG] uploadAndSaveFiles called with %d files for user %d, session %s", len(files), user.ID, sessionUUID)

	var attachedFileIDs []int64
	var totalDecodedSize int64

	for i, fileData := range files {
		log.Printf("[DEBUG] Processing file %d: name='%s', mime='%s', base64_length=%d", i+1, fileData.FileName, fileData.MimeType, len(fileData.Base64Data))
		cleanedBase64 := cleanBase64Data(fileData.Base64Data)
		if cleanedBase64 == "" {
			continue
		}

		s3Key := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(fileData.FileName))
		decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(cleanedBase64))

		var perFileCount int64
		cappedReader := &sizeCappedReader{
			r:            decoder,
			perCount:     &perFileCount,
			totalCount:   &totalDecodedSize,
			perFileLimit: perFileLimitBytes,
			totalLimit:   totalUploadLimitBytes,
		}

		err := fp.s3Service.UploadStream(ctx, s3Key, fileData.MimeType, cappedReader)
		if err != nil {
			log.Printf("!!! ERROR: Failed to upload %s to S3: %v. Deleting partial object.", fileData.FileName, err)
			fp.s3Service.DeleteFiles(context.Background(), []string{s3Key})
			continue
		}

		fileID, err := fp.db.SaveFileAttachment(sessionUUID, user.ID, fileData.FileName, s3Key, fileData.MimeType, "uploaded", nil)
		if err != nil {
			log.Printf("!!! ERROR: Failed to save file metadata for %s: %v. Deleting object from S3.", fileData.FileName, err)
			fp.s3Service.DeleteFiles(context.Background(), []string{s3Key})
			continue
		}
		attachedFileIDs = append(attachedFileIDs, fileID)
	}
	return attachedFileIDs, nil
}

// vectorizeFiles fetches files, extracts text, chunks it, and stores embeddings.
func (fp *fileProcessor) vectorizeFiles(ctx context.Context, fileIDs []int64) error {
	if len(fileIDs) == 0 {
		return nil
	}

	attachments, err := fp.db.GetAttachmentsByIDs(fileIDs)
	if err != nil {
		return fmt.Errorf("could not get attachments for vectorization: %w", err)
	}

	for _, att := range attachments {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			textContent, err := fp.extractText(ctx, &att)
			if err != nil {
				log.Printf("[VECTORS] Could not extract text from %s: %v", att.FileName, err)
				continue
			}
			if textContent == "" {
				continue
			}

			chunks := chunkPlainText(textContent, chunkSize, chunkOverlap)
			for idx, chunkText := range chunks {
				if strings.TrimSpace(chunkText) == "" {
					continue
				}
				chunkID, err := fp.db.UpsertFileChunk(att.ID, idx, chunkText)
				if err != nil {
					log.Printf("[VECTORS] Failed to save chunk for %s: %v", att.FileName, err)
					continue
				}

				// Use proper embedding from Python/Gemini
				vec, err := fp.llmClient.GetEmbedding(ctx, chunkText)
				if err != nil {
					log.Printf("[VECTORS] Failed to get embedding for chunk of %s: %v. Falling back to local hash.", att.FileName, err)
					vec, _ = localEmbedTextFallback(chunkText) // Keep fallback just in case
				}

				if err := fp.db.SaveFileChunkEmbedding(chunkID, vec); err != nil {
					log.Printf("[VECTORS] Failed to save chunk embedding for %s: %v", att.FileName, err)
				}
			}
		}
	}
	return nil
}

// vectorizeAndSaveMessage generates and saves an embedding for a message.
func (fp *fileProcessor) vectorizeAndSaveMessage(logID int64, text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	// Use proper embedding from Python/Gemini
	vec, err := fp.llmClient.GetEmbedding(context.Background(), text)
	if err != nil {
		log.Printf("[VECTORS] Failed to get embedding for message log %d: %v. Falling back to local hash.", logID, err)
		vec, err = localEmbedTextFallback(text)
		if err != nil {
			return fmt.Errorf("failed to embed message text: %w", err)
		}
	}

	if err := fp.db.SaveMessageEmbedding(logID, vec); err != nil {
		return fmt.Errorf("failed to save message embedding: %w", err)
	}
	return nil
}

// extractText downloads a file and extracts its text content.
func (fp *fileProcessor) extractText(ctx context.Context, att *models.FileAttachment) (string, error) {
	rc, err := fp.s3Service.DownloadStream(ctx, att.FileURI)
	if err != nil {
		return "", fmt.Errorf("failed to open S3 stream for %s: %w", att.FileURI, err)
	}
	defer rc.Close()

	mime := strings.ToLower(att.MimeType)
	switch {
	case strings.HasPrefix(mime, "text/") || mime == "application/json":
		var buf bytes.Buffer
		_, err := io.Copy(&buf, io.LimitReader(rc, maxTextVectorizeBytes))
		return buf.String(), err
	case mime == "application/pdf":
		data, err := io.ReadAll(io.LimitReader(rc, maxPDFVectorizeBytes))
		if err != nil {
			return "", err
		}
		return safeExtractPDFText(data), nil
	default:
		return "", nil
	}
}

// --- Helper Functions (Copied from original processor.go) ---

func cleanBase64Data(data string) string {
	clean := strings.TrimSpace(data)
	if strings.HasPrefix(clean, "data:") {
		if idx := strings.Index(clean, ","); idx >= 0 {
			clean = clean[idx+1:]
		}
	}
	clean = strings.ReplaceAll(clean, "\n", "")
	clean = strings.ReplaceAll(clean, "\r", "")
	clean = strings.ReplaceAll(clean, "\t", "")
	clean = strings.ReplaceAll(clean, " ", "")
	return clean
}

func chunkPlainText(s string, chunkSize int, overlap int) []string {
	if chunkSize <= 0 {
		return nil
	}
	if overlap < 0 {
		overlap = 0
	}
	runes := []rune(s)
	var chunks []string
	for start := 0; start < len(runes); start += (chunkSize - overlap) {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func localEmbedTextFallback(text string) ([]float64, error) {
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty text")
	}
	const buckets = 256
	vec := make([]float64, buckets)
	for _, token := range tokenize(text) {
		h := fnv32(token) % buckets
		vec[h] += 1.0
	}
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	if sum == 0 {
		return vec, nil
	}
	norm := 1.0 / (math.Sqrt(sum))
	for i := range vec {
		vec[i] *= norm
	}
	return vec, nil
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var tokens []string
	cur := strings.Builder{}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'а' && r <= 'я') {
			cur.WriteRune(r)
		} else {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func fnv32(s string) int {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	var h uint32 = offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	if h == 0 {
		return 0
	}
	return int(h)
}

// --- PDF Parsing Logic (Copied from original, translated and documented) ---

var (
	pdfTextPattern       = regexp.MustCompile(`\(([^)]*?)\)`)
	pdfStreamPattern     = regexp.MustCompile(`stream\s*([\s\S]*?)\s*endstream`)
	pdfBTBlockPattern    = regexp.MustCompile(`BT\s*([\s\S]*?)\s*ET`)
	pdfTjOperatorPattern = regexp.MustCompile(`\(([^)]*?)\)\s*Tj`)
	pdfTJOperatorPattern = regexp.MustCompile(`\[([^\]]*?)\]\s*TJ`)
)

func safeExtractPDFText(data []byte) string {
	if len(data) < 100 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return ""
	}

	var text strings.Builder
	dataStr := string(data)

	// Pattern 1: Text in parentheses
	matches := pdfTextPattern.FindAllStringSubmatch(dataStr, -1)
	for _, match := range matches {
		if len(match) > 1 {
			text.WriteString(cleanPDFText(match[1]) + " ")
		}
	}

	// Pattern 2: Text streams
	streamMatches := pdfStreamPattern.FindAllStringSubmatch(dataStr, -1)
	for _, match := range streamMatches {
		if len(match) > 1 {
			text.WriteString(extractTextFromStream(match[1]) + "\n")
		}
	}

	// Pattern 3: BT/ET blocks
	btMatches := pdfBTBlockPattern.FindAllStringSubmatch(dataStr, -1)
	for _, match := range btMatches {
		if len(match) > 1 {
			text.WriteString(extractTextFromBTBlock(match[1]) + "\n")
		}
	}

	result := strings.Join(strings.Fields(text.String()), " ")
	return strings.TrimSpace(result)
}

func extractTextFromStream(streamContent string) string {
	var text strings.Builder
	lines := strings.Split(streamContent, "\n")
	for _, line := range lines {
		if match := pdfTjOperatorPattern.FindStringSubmatch(line); len(match) > 1 {
			text.WriteString(cleanPDFText(match[1]) + " ")
		}
		if match := pdfTJOperatorPattern.FindStringSubmatch(line); len(match) > 1 {
			text.WriteString(extractTextFromTJArray(match[1]))
		}
	}
	return text.String()
}

func extractTextFromBTBlock(block string) string {
	return extractTextFromStream(block) // The logic is identical
}

func extractTextFromTJArray(arrayStr string) string {
	var text strings.Builder
	parts := strings.Split(arrayStr, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "(") && strings.HasSuffix(part, ")") {
			textStr := strings.Trim(part, "()")
			text.WriteString(cleanPDFText(textStr))
		}
	}
	return text.String()
}

func cleanPDFText(text string) string {
	text = strings.ReplaceAll(text, "\\(", "(")
	text = strings.ReplaceAll(text, "\\)", ")")
	text = strings.ReplaceAll(text, "\\\\", "\\")
	return text
}
