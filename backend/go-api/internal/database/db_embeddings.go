// This file contains database methods for handling embeddings, both for messages and file chunks.
// It includes logic to work with or without the pgvector extension.

package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

// SaveMessageEmbedding stores an embedding vector for a specific request log.
// It attempts to use a dedicated 'vector' column if it exists, falling back to JSONB.
func (db *DB) SaveMessageEmbedding(logID int64, vector []float64) error {
	if len(vector) == 0 {
		return errors.New("embedding vector cannot be empty")
	}
	vecJSON, err := json.Marshal(vector)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding vector to JSON: %w", err)
	}

	// If a pgvector column exists, use it for efficient similarity search.
	if db.columnExists("message_embeddings", "embedding_vec") {
		// Ensure correct dimension (vector(256)) and format as pgvector literal
		v := normalizeVec(vector, 256)
		vLit := toVectorLiteral(v)
		_, err = db.Exec(`
            INSERT INTO message_embeddings (log_id, embedding, embedding_vec)
            VALUES ($1, $2, $3::vector)
            ON CONFLICT (log_id) DO UPDATE SET
                embedding = EXCLUDED.embedding,
                embedding_vec = EXCLUDED.embedding_vec
        `, logID, vecJSON, vLit)
		// If the insert is successful, we are done.
		if err == nil {
			return nil
		}
		// If the vector upsert failed for any reason (e.g., wrong vector dimensions),
		// log the error and fall back to a JSONB-only insert.
		log.Printf("[WARN] Failed to save to vector column, falling back to JSONB for log_id %d: %v", logID, err)
	}

	// Fallback for databases without pgvector or if the vector insert failed.
	_, err = db.Exec(`
        INSERT INTO message_embeddings (log_id, embedding)
        VALUES ($1, $2)
        ON CONFLICT (log_id) DO UPDATE SET embedding = EXCLUDED.embedding`, logID, vecJSON)

	if err != nil {
		return fmt.Errorf("failed to save message embedding: %w", err)
	}
	return nil
}

// UpsertFileChunk inserts or updates a text chunk for a file attachment.
// It uses an "ON CONFLICT" clause to ensure atomicity and performance.
func (db *DB) UpsertFileChunk(fileAttachmentID int64, chunkIndex int, chunkText string) (int64, error) {
	// Sanitize null bytes, which PostgreSQL cannot store in text fields.
	sanitizedChunkText := strings.ReplaceAll(chunkText, "\x00", "")

	var id int64
	// The ON CONFLICT clause handles the upsert logic atomically within the database.
	// This is more efficient than performing a SELECT and then an INSERT/UPDATE.
	query := `
        INSERT INTO file_chunks (file_attachment_id, chunk_index, chunk_text)
        VALUES ($1, $2, $3)
        ON CONFLICT (file_attachment_id, chunk_index) DO UPDATE SET chunk_text = EXCLUDED.chunk_text
        RETURNING id`
	err := db.QueryRow(query, fileAttachmentID, chunkIndex, sanitizedChunkText).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert file chunk: %w", err)
	}
	return id, nil
}

// SaveFileChunkEmbedding stores an embedding vector for a specific file chunk.
// It mirrors the logic of SaveMessageEmbedding, using 'vector' or JSONB columns.
func (db *DB) SaveFileChunkEmbedding(fileChunkID int64, vector []float64) error {
	if len(vector) == 0 {
		return errors.New("embedding vector cannot be empty")
	}
	vecJSON, err := json.Marshal(vector)
	if err != nil {
		return fmt.Errorf("failed to marshal embedding vector to JSON: %w", err)
	}

	if db.columnExists("file_chunk_embeddings", "embedding_vec") {
		v := normalizeVec(vector, 256)
		vLit := toVectorLiteral(v)
		_, err = db.Exec(`
            INSERT INTO file_chunk_embeddings (file_chunk_id, embedding, embedding_vec)
            VALUES ($1, $2, $3::vector)
            ON CONFLICT (file_chunk_id) DO UPDATE SET
                embedding = EXCLUDED.embedding,
                embedding_vec = EXCLUDED.embedding_vec
        `, fileChunkID, vecJSON, vLit)
		if err == nil {
			return nil
		}
		log.Printf("[WARN] Failed to save to vector column, falling back to JSONB for file_chunk_id %d: %v", fileChunkID, err)
	}

	_, err = db.Exec(`
        INSERT INTO file_chunk_embeddings (file_chunk_id, embedding)
        VALUES ($1, $2)
        ON CONFLICT (file_chunk_id) DO UPDATE SET embedding = EXCLUDED.embedding`, fileChunkID, vecJSON)
	if err != nil {
		return fmt.Errorf("failed to save file chunk embedding: %w", err)
	}
	return nil
}

// normalizeVec ensures the vector has exactly dim elements by truncating or padding with zeros.
func normalizeVec(vec []float64, dim int) []float64 {
	if len(vec) == dim {
		return vec
	}
	out := make([]float64, dim)
	n := len(vec)
	if n > dim {
		copy(out, vec[:dim])
	} else {
		copy(out, vec)
		// remaining are zeros by default
	}
	return out
}

// toVectorLiteral converts a slice of float64 into a pgvector literal string: "[v1,v2,...]".
func toVectorLiteral(vec []float64) string {
	// Avoid importing fmt for every element; fmt is already imported above
	var b strings.Builder
	b.Grow(2 + len(vec)*8)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%g", v))
	}
	b.WriteByte(']')
	return b.String()
}

// columnExists checks if a column exists in a table. Results are cached in memory
// to avoid repeated queries to the database's information_schema.
func (db *DB) columnExists(tableName, columnName string) bool {
	cacheKey := tableName + "." + columnName
	db.columnCacheMutex.RLock()
	exists, found := db.columnCache[cacheKey]
	db.columnCacheMutex.RUnlock()
	if found {
		return exists
	}

	query := `
        SELECT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
        )`
	err := db.QueryRow(query, tableName, columnName).Scan(&exists)
	if err != nil {
		log.Printf("Could not check for column existence %s on %s: %v", columnName, tableName, err)
		exists = false
	}

	db.columnCacheMutex.Lock()
	if db.columnCache == nil {
		db.columnCache = make(map[string]bool)
	}
	db.columnCache[cacheKey] = exists
	db.columnCacheMutex.Unlock()

	return exists
}
