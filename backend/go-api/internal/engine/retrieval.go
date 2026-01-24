// в internal/engine/retrieval.go

package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"egobackend/internal/database"
	"egobackend/internal/models"
)

// retriever handles the logic for searching and retrieving relevant information (RAG).
type retriever struct {
	db *database.DB
}

// newRetriever creates a new retriever instance.
func newRetriever(db *database.DB) *retriever {
	return &retriever{db: db}
}

// unmarshalEmbedding is a helper to safely unmarshal a raw JSONB embedding into a float slice.
func (r *retriever) unmarshalEmbedding(rawEmbedding []byte, target *[]float64) {
	if len(rawEmbedding) > 0 && len(*target) == 0 {
		if err := json.Unmarshal(rawEmbedding, target); err != nil {
			log.Printf("Warning: failed to unmarshal embedding: %v", err)
		}
	}
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
	var b strings.Builder
	b.Grow(2 + len(vec)*8)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		// Use fmt to ensure proper float formatting
		b.WriteString(fmt.Sprintf("%g", v))
	}
	b.WriteByte(']')
	return b.String()
}

// GetRecentFileChunkEmbeddingsBySession retrieves the most recent file chunk embeddings for a specific session.
func (r *retriever) GetRecentFileChunkEmbeddingsBySession(sessionUUID string, limit int) ([]models.FileChunkWithEmbedding, error) {
	if limit <= 0 {
		limit = 50
	}

	var results []models.FileChunkWithEmbedding
	query := `
        SELECT fc.id, fc.chunk_text, fce.embedding AS embedding_raw
        FROM file_chunks fc
        JOIN file_attachments fa ON fc.file_attachment_id = fa.id
        JOIN file_chunk_embeddings fce ON fce.file_chunk_id = fc.id
        WHERE fa.session_uuid = $1
        ORDER BY fc.created_at DESC
        LIMIT $2`
	if err := r.db.Select(&results, query, sessionUUID, limit); err != nil {
		return nil, fmt.Errorf("failed to get recent file chunk embeddings by session: %w", err)
	}

	for i := range results {
		r.unmarshalEmbedding(results[i].EmbeddingRaw, &results[i].Embedding)
	}
	return results, nil
}

// GetRecentMessageEmbeddingsByUser retrieves the most recent message embeddings across all sessions for a user.
func (r *retriever) GetRecentMessageEmbeddingsByUser(userID int, limit int) ([]models.MessageEmbeddingWithText, error) {
	if limit <= 0 {
		limit = 100
	}

	var results []models.MessageEmbeddingWithText
	query := `
        SELECT me.log_id,
               ('User: ' || rl.user_query || E'\n' || 'EGO: ' || COALESCE(rl.final_response, '')) AS text,
               me.embedding AS embedding_raw
        FROM message_embeddings me
        JOIN request_logs rl ON rl.id = me.log_id
        JOIN chat_sessions s ON s.uuid = rl.session_uuid
        WHERE s.user_id = $1
        ORDER BY me.created_at DESC
        LIMIT $2`
	if err := r.db.Select(&results, query, userID, limit); err != nil {
		return nil, fmt.Errorf("failed to get recent message embeddings by user: %w", err)
	}

	for i := range results {
		r.unmarshalEmbedding(results[i].EmbeddingRaw, &results[i].Embedding)
	}
	return results, nil
}

// SearchTopSnippetsByUserWithScore performs a vector similarity search with a minimum score threshold.
func (r *retriever) SearchTopSnippetsByUserWithScore(userID int, qVec []float64, topK int, minScore float64) ([]string, error) {
	if topK <= 0 {
		topK = 3
	}

	query := `
        WITH m AS (
            SELECT ('User: ' || rl.user_query || E'\n' || 'EGO: ' || COALESCE(rl.final_response, '')) AS text, me.embedding_vec AS v
            FROM message_embeddings me
            JOIN request_logs rl ON rl.id = me.log_id
            JOIN chat_sessions s ON s.uuid = rl.session_uuid
            WHERE s.user_id = $1 AND me.embedding_vec IS NOT NULL
        ), c AS (
            SELECT fc.chunk_text AS text, fce.embedding_vec AS v
            FROM file_chunk_embeddings fce
            JOIN file_chunks fc ON fc.id = fce.file_chunk_id
            JOIN file_attachments fa ON fa.id = fc.file_attachment_id
            JOIN chat_sessions s ON s.uuid = fa.session_uuid
            WHERE s.user_id = $1 AND fce.embedding_vec IS NOT NULL
        ), all_sources AS (
            SELECT text, v FROM m UNION ALL SELECT text, v FROM c
        ), scored AS (
            SELECT text, 1 - (v <=> $2::vector) AS score
            FROM all_sources
            WHERE text IS NOT NULL AND length(text) > 0
        )
        SELECT text FROM scored
        WHERE score >= $4
        ORDER BY score DESC
        LIMIT $3`

	var results []string
	// Ensure correct dimension (matches schema vector(256)) and pgvector literal formatting
	qv := normalizeVec(qVec, 256)
	qvLiteral := toVectorLiteral(qv)
	err := r.db.Select(&results, query, userID, qvLiteral, topK, minScore)
	if err != nil {
		return nil, fmt.Errorf("failed to search for snippets with score: %w", err)
	}
	return results, nil
}
