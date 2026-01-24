// This file contains database methods related to request logs and their attachments.

package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"egobackend/internal/models"

	"github.com/jmoiron/sqlx"
)

// CreateInitialRequestLog creates a placeholder log entry immediately after a request is received.
// It returns the ID of the newly created log entry.
func (db *DB) CreateInitialRequestLog(sessionUUID string, userQuery string) (int64, error) {
	query := `
        INSERT INTO request_logs (session_uuid, user_query, timestamp, ego_thoughts_json, final_response, attached_file_ids)
        VALUES ($1, $2, $3, '[]', '', '[]')
        RETURNING id`
	var logID int64
	err := db.QueryRow(query, sessionUUID, userQuery, time.Now().UTC()).Scan(&logID)
	if err != nil {
		return 0, fmt.Errorf("failed to create initial request log: %w", err)
	}
	return logID, nil
}

// UpdateLogWithFinalResponse updates an existing log entry with the complete data
// after the response from the LLM has been generated.
func (db *DB) UpdateLogWithFinalResponse(logID int64, thoughtsJSON string, finalResponse string, attachedFileIDsJSON string, promptTokens, completionTokens, totalTokens int) error {
	query := `
        UPDATE request_logs
        SET
            ego_thoughts_json = $1,
            final_response = $2,
            attached_file_ids = $3,
            prompt_tokens = $4,
            completion_tokens = $5,
            total_tokens = $6
        WHERE id = $7`
	_, err := db.Exec(query, thoughtsJSON, finalResponse, attachedFileIDsJSON, promptTokens, completionTokens, totalTokens, logID)
	if err != nil {
		return fmt.Errorf("failed to update log with final response: %w", err)
	}
	return nil
}

// GetSessionHistory retrieves a chronological list of request logs for a given session.
// It also fetches all associated file attachments and maps them to their respective logs.
func (db *DB) GetSessionHistory(sessionUUID string, limit int) ([]models.RequestLog, map[int][]models.FileAttachment, error) {
	var logs []models.RequestLog
	// Fetch logs in chronological order directly from the database.
	query := `
        SELECT id, session_uuid, user_query, ego_thoughts_json, final_response,
               prompt_tokens, completion_tokens, total_tokens, attached_file_ids, timestamp
        FROM request_logs
        WHERE session_uuid = $1
        ORDER BY timestamp ASC
        LIMIT $2`
	err := db.Select(&logs, query, sessionUUID, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session history: %w", err)
	}

	attachmentsMap, err := db.GetAttachmentsForLogs(logs)
	if err != nil {
		return logs, nil, fmt.Errorf("failed to get attachments for logs: %w", err)
	}

	return logs, attachmentsMap, nil
}

// GetSessionHistoryBefore retrieves a chronological list of request logs for a session
// that occurred before a specified time, used for pagination.
func (db *DB) GetSessionHistoryBefore(sessionUUID string, beforeTime time.Time, limit int) ([]models.RequestLog, map[int][]models.FileAttachment, error) {
	var logs []models.RequestLog
	// Fetch logs in chronological order directly.
	query := `
        SELECT id, session_uuid, user_query, ego_thoughts_json, final_response, attached_file_ids, timestamp
        FROM request_logs
        WHERE session_uuid = $1 AND timestamp < $2
        ORDER BY timestamp ASC
        LIMIT $3`
	err := db.Select(&logs, query, sessionUUID, beforeTime, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session history before time: %w", err)
	}

	attachmentsMap, err := db.GetAttachmentsForLogs(logs)
	if err != nil {
		return logs, nil, fmt.Errorf("failed to get attachments for paginated logs: %w", err)
	}

	return logs, attachmentsMap, nil
}

// GetAttachmentsForLogs efficiently fetches all file attachments for a given slice of logs.
// It performs one database query to get all attachments and then maps them in memory.
func (db *DB) GetAttachmentsForLogs(logs []models.RequestLog) (map[int][]models.FileAttachment, error) {
	if len(logs) == 0 {
		return make(map[int][]models.FileAttachment), nil
	}

	var allFileIDs []int64
	for _, log := range logs {
		if !log.AttachedFileIDs.Valid || log.AttachedFileIDs.String == "" || log.AttachedFileIDs.String == "[]" {
			continue
		}
		var fileIDs []int64
		if err := json.Unmarshal([]byte(log.AttachedFileIDs.String), &fileIDs); err == nil {
			allFileIDs = append(allFileIDs, fileIDs...)
		}
	}

	// Prepare the map to be returned. Initialize slices for all logs.
	attachmentsMap := make(map[int][]models.FileAttachment, len(logs))
	for _, log := range logs {
		attachmentsMap[log.ID] = []models.FileAttachment{}
	}

	if len(allFileIDs) > 0 {
		var attachments []models.FileAttachment
		query, args, err := sqlx.In(`
            SELECT id, session_uuid, user_id, file_name, file_uri, mime_type, status, created_at, request_log_id
            FROM file_attachments WHERE id IN (?)`, allFileIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to build query for attachments: %w", err)
		}

		query = db.Rebind(query)
		err = db.Select(&attachments, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute query for attachments: %w", err)
		}

		// Efficiently map attachments to their corresponding logs.
		for _, att := range attachments {
			// The request_log_id from the DB tells us which log this attachment belongs to.
			if att.RequestLogID.Valid {
				logID := int(att.RequestLogID.Int64)
				attachmentsMap[logID] = append(attachmentsMap[logID], att)
			}
		}
	}

	return attachmentsMap, nil
}

// GetRequestLogByID retrieves a single request log by its ID, ensuring it belongs to the specified user.
// Returns (nil, nil) if the log is not found or the user does not have permission.
func (db *DB) GetRequestLogByID(logID int64, userID int) (*models.RequestLog, error) {
	var log models.RequestLog
	query := `
        SELECT rl.* FROM request_logs rl
        JOIN chat_sessions cs ON rl.session_uuid = cs.uuid
        WHERE rl.id = $1 AND cs.user_id = $2`
	err := db.Get(&log, query, logID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // Not found is not a critical error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get request log by ID: %w", err)
	}
	return &log, nil
}

// UpdateRequestLogQuery updates the user_query text of a specific log entry.
// It verifies ownership by checking against the user ID associated with the session.
func (db *DB) UpdateRequestLogQuery(logID int64, userID int, newQuery string) error {
	query := `
        UPDATE request_logs SET user_query = $1, timestamp = $2
        WHERE id = $3 AND session_uuid IN (SELECT uuid FROM chat_sessions WHERE user_id = $4)`
	_, err := db.Exec(query, newQuery, time.Now().UTC(), logID, userID)
	if err != nil {
		return fmt.Errorf("failed to update request log query: %w", err)
	}
	return err
}

// GetAttachmentsByLogID retrieves all file attachments linked to a specific request log ID.
func (db *DB) GetAttachmentsByLogID(logID int64) ([]models.FileAttachment, error) {
	var attachments []models.FileAttachment
	query := `
        SELECT id, session_uuid, user_id, file_name, file_uri, mime_type, status, created_at, request_log_id
        FROM file_attachments WHERE request_log_id = $1`
	err := db.Select(&attachments, query, logID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get attachments by log ID: %w", err)
	}
	// Return an empty slice if no rows are found, which is expected.
	return attachments, nil
}

// DeleteLogsAfter removes all request logs in a given session that have an ID greater than the cutoffLogID.
// Ownership is validated by ensuring the session belongs to the provided userID and that the cutoff
// log is part of the same session. This operation is performed within a transaction.
// Dependent rows in other tables are handled by database constraints (ON DELETE CASCADE/SET NULL).
func (db *DB) DeleteLogsAfter(sessionUUID string, cutoffLogID int64, userID int) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Defer a function to handle rollback on error.
	defer func() {
		if p := recover(); p != nil {
			// A panic occurred, rollback.
			tx.Rollback()
			panic(p) // Re-throw panic after Rollback
		} else if err != nil {
			// An error occurred, rollback.
			tx.Rollback()
		}
	}()

	// 1. Validate that the user owns the session.
	var ownsSession bool
	err = tx.Get(&ownsSession, `SELECT EXISTS(SELECT 1 FROM chat_sessions WHERE uuid = $1 AND user_id = $2)`, sessionUUID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate session ownership: %w", err)
	}
	if !ownsSession {
		return sql.ErrNoRows // Using ErrNoRows to signify "not found" or "no permission"
	}

	// 2. Ensure the cutoff log ID belongs to that same session.
	var logExistsInSession bool
	err = tx.Get(&logExistsInSession, `SELECT EXISTS(SELECT 1 FROM request_logs WHERE id = $1 AND session_uuid = $2)`, cutoffLogID, sessionUUID)
	if err != nil {
		return fmt.Errorf("failed to validate log existence in session: %w", err)
	}
	if !logExistsInSession {
		return sql.ErrNoRows // The specified log is not in this session.
	}

	// 3. Delete all logs after the cutoff within the same session.
	_, err = tx.Exec(`DELETE FROM request_logs WHERE session_uuid = $1 AND id > $2`, sessionUUID, cutoffLogID)
	if err != nil {
		return fmt.Errorf("failed to delete logs: %w", err)
	}

	// 4. If all went well, commit the transaction.
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
