// This file contains database methods related to file attachments.

package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"egobackend/internal/models"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// SaveFileAttachment creates a record for a new file attachment in the database.
// If the sessionUUID is empty, it automatically creates a temporary session for the file.
// It returns the ID of the newly created file attachment record.
func (db *DB) SaveFileAttachment(sessionUUID string, userID int, fileName, fileURI, mimeType, status string, uploadID *string) (int64, error) {
	// If a file is uploaded to a new chat that doesn't have a session UUID yet,
	// create a temporary one to associate the file with.
	if strings.TrimSpace(sessionUUID) == "" {
		tempSessionUUID, err := db.CreateTempSession(userID)
		if err != nil {
			log.Printf("Failed to create temporary session for file upload: %v", err)
			return 0, fmt.Errorf("could not create temporary session for file: %w", err)
		}
		sessionUUID = tempSessionUUID
	}

	query := `
        INSERT INTO file_attachments (session_uuid, user_id, file_name, file_uri, mime_type, status, created_at, upload_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id`

	// Use sql.NullString for nullable UUID columns for better type safety and clarity.
	var nullableUploadID sql.NullString
	if uploadID != nil {
		nullableUploadID = sql.NullString{String: *uploadID, Valid: true}
	}

	var fileID int64
	err := db.QueryRow(query, sessionUUID, userID, fileName, fileURI, mimeType, status, time.Now().UTC(), nullableUploadID).Scan(&fileID)
	if err != nil {
		return 0, fmt.Errorf("failed to save file attachment: %w", err)
	}
	return fileID, nil
}

// GetFileIDsByUploadID retrieves and optionally updates file attachment records for a given upload batch ID.
// If a sessionUUID is provided, it updates the session for all matching files in a single atomic query.
func (db *DB) GetFileIDsByUploadID(uploadID string, userID int, sessionUUID string) ([]int64, error) {
	if uploadID == "" {
		return []int64{}, nil
	}

	var ids []int64
	var err error

	// If a sessionUUID is provided, we perform an UPDATE and return the IDs of the updated rows.
	// This is more efficient than a separate SELECT and UPDATE.
	if sessionUUID != "" {
		query := `
            UPDATE file_attachments
            SET session_uuid = $1
            WHERE upload_id = $2 AND user_id = $3 AND request_log_id IS NULL
            RETURNING id`
		err = db.Select(&ids, query, sessionUUID, uploadID, userID)
		if err == nil {
			log.Printf("Updated %d files from upload_id %s to session %s", len(ids), uploadID, sessionUUID)
		}
	} else {
		// If no sessionUUID is provided, just select the IDs.
		query := `SELECT id FROM file_attachments WHERE upload_id = $1 AND user_id = $2 AND request_log_id IS NULL`
		err = db.Select(&ids, query, uploadID, userID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to process files for upload ID %s: %w", uploadID, err)
	}

	return ids, nil
}

// AssociateFilesWithRequestLog links multiple file attachments to a single request log entry.
// This is done in a single, efficient UPDATE query.
func (db *DB) AssociateFilesWithRequestLog(logID int64, fileIDs []int64) error {
	if len(fileIDs) == 0 {
		return nil
	}

	query := `UPDATE file_attachments SET request_log_id = $1 WHERE id = ANY($2)`
	_, err := db.Exec(query, logID, pq.Array(fileIDs))
	if err != nil {
		return fmt.Errorf("failed to associate files with request log: %w", err)
	}
	return nil
}

// GetAttachmentsByIDs retrieves multiple file attachment records based on a slice of IDs.
func (db *DB) GetAttachmentsByIDs(ids []int64) ([]models.FileAttachment, error) {
	if len(ids) == 0 {
		return []models.FileAttachment{}, nil
	}

	query, args, err := sqlx.In("SELECT * FROM file_attachments WHERE id IN (?)", ids)
	if err != nil {
		return nil, fmt.Errorf("failed to build query for attachments by IDs: %w", err)
	}

	query = db.Rebind(query)
	var attachments []models.FileAttachment
	err = db.Select(&attachments, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select attachments by IDs: %w", err)
	}

	return attachments, nil
}

// DeleteOldFileAttachments deletes file attachment records older than a specified duration.
// It returns a slice of the file URIs that were deleted, which can be used to delete the actual files from storage.
func (db *DB) DeleteOldFileAttachments(maxAge time.Duration) ([]string, error) {
	cutoffTime := time.Now().UTC().Add(-maxAge)
	query := `DELETE FROM file_attachments WHERE created_at < $1 RETURNING file_uri`

	var deletedURIs []string
	err := db.Select(&deletedURIs, query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to delete old file attachments: %w", err)
	}
	return deletedURIs, nil
}

// DeleteOrphanedFileAttachments deletes file attachments that were never associated with a request log
// and are older than a specified duration. This cleans up files from aborted or failed uploads.
// It returns the URIs of the deleted files.
func (db *DB) DeleteOrphanedFileAttachments(maxAge time.Duration) ([]string, error) {
	cutoffTime := time.Now().UTC().Add(-maxAge)
	query := `DELETE FROM file_attachments WHERE request_log_id IS NULL AND created_at < $1 RETURNING file_uri`

	var deletedURIs []string
	err := db.Select(&deletedURIs, query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to delete orphaned file attachments: %w", err)
	}
	return deletedURIs, nil
}

// GetAttachmentOwnedByUser retrieves a specific file attachment, ensuring it belongs to the
// specified user and is associated with the correct log entry.
func (db *DB) GetAttachmentOwnedByUser(logID int, userID int, fileName string) (*models.FileAttachment, error) {
	var att models.FileAttachment
	query := `
        SELECT fa.id, fa.session_uuid, fa.user_id, fa.file_name, fa.file_uri, fa.mime_type, fa.status, fa.created_at, fa.request_log_id
        FROM file_attachments fa
        WHERE fa.request_log_id = $1 AND fa.user_id = $2 AND fa.file_name = $3
        LIMIT 1`
	err := db.Get(&att, query, logID, userID, fileName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil for not found, as it's not a system error.
		}
		return nil, fmt.Errorf("failed to get owned attachment: %w", err)
	}
	return &att, nil
}
