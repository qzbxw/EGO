// This file contains database methods related to chat sessions.

package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"egobackend/internal/models"
)

const (
	// defaultSessionTitle is the title assigned to newly created sessions.
	defaultSessionTitle = "New Chat"
	// defaultSessionMode is the mode assigned to newly created sessions.
	defaultSessionMode = "default"
)

// GetUserSessions retrieves all chat sessions for a specific user.
// Sessions are sorted by the timestamp of the last message in descending order,
// with sessions without messages appearing last, sorted by their creation date.
func (db *DB) GetUserSessions(userID int) ([]models.ChatSession, error) {
	var sessions []models.ChatSession

	// This query efficiently sorts sessions by their last activity.
	// The LEFT JOIN with a subquery finds the latest timestamp for each session.
	// ORDER BY sorts by that timestamp (DESC), placing the most recently active sessions first.
	// `NULLS LAST` ensures sessions with no logs appear at the end, sorted by creation date.
	query := `
        SELECT
            cs.uuid, cs.user_id, cs.title, cs.mode, cs.custom_instructions, cs.created_at
        FROM chat_sessions cs
        LEFT JOIN (
            SELECT session_uuid, MAX(timestamp) AS last_ts
            FROM request_logs
            GROUP BY session_uuid
        ) rl ON rl.session_uuid = cs.uuid
        WHERE cs.user_id = $1
        ORDER BY rl.last_ts DESC NULLS LAST, cs.created_at DESC`

	err := db.Select(&sessions, query, userID)
	if err != nil {
		log.Printf("DATABASE ERROR in GetUserSessions for user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	return sessions, nil
}

// DeleteSession deletes a specific session and all its related data for a user.
// This operation is transactional and includes a best-effort cleanup of data
// in the 'ego_memory' table, which may be managed by an external Python service.
func (db *DB) DeleteSession(sessionUUID string, userID int) (err error) {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				err = fmt.Errorf("failed to commit transaction: %w", err)
			}
		}
	}()

	// 1. Verify ownership of the session.
	var exists bool
	if err = tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM chat_sessions WHERE uuid=$1 AND user_id=$2)`, sessionUUID, userID); err != nil {
		return fmt.Errorf("failed to verify session ownership: %w", err)
	}
	if !exists {
		return sql.ErrNoRows
	}

	// 2. Delete the session. Foreign key constraints with ON DELETE CASCADE will handle related data
	// in tables like request_logs, file_attachments, etc.
	if _, err = tx.Exec(`DELETE FROM chat_sessions WHERE uuid = $1 AND user_id = $2`, sessionUUID, userID); err != nil {
		return fmt.Errorf("failed to delete session row: %w", err)
	}

	// 3. Best-effort: Purge per-session vector memory from the 'ego_memory' table.
	// We use a query that checks if the table exists before attempting deletion.
	purgeQuery := `
        DELETE FROM ego_memory 
        WHERE user_id = $1 AND session_id = $2
        AND EXISTS (
            SELECT 1 FROM information_schema.tables 
            WHERE table_name = 'ego_memory'
        )`
	if _, err = tx.Exec(purgeQuery, fmt.Sprintf("%d", userID), sessionUUID); err != nil {
		log.Printf("[WARN] Failed to purge ego_memory for user %d session %s (non-fatal): %v", userID, sessionUUID, err)
		err = nil // Reset error to not fail the transaction
	}

	// 4. If this was the user's last session, purge any remaining memory entries for this user.
	var remainingSessions int
	if err = tx.Get(&remainingSessions, `SELECT COUNT(*) FROM chat_sessions WHERE user_id = $1`, userID); err == nil && remainingSessions == 0 {
		purgeAllQuery := `
            DELETE FROM ego_memory 
            WHERE user_id = $1
            AND EXISTS (
                SELECT 1 FROM information_schema.tables 
                WHERE table_name = 'ego_memory'
            )`
		if _, err2 := tx.Exec(purgeAllQuery, fmt.Sprintf("%d", userID)); err2 != nil {
			log.Printf("[WARN] Failed to purge all ego_memory for user %d after last session deletion (non-fatal): %v", userID, err2)
		}
	}
	// Reset potential error from the COUNT query.
	err = nil

	return nil
}

// DeleteAllSessionsForUser deletes all sessions and related data for a user.
// This is a transactional operation, including a best-effort cleanup of 'ego_memory'.
func (db *DB) DeleteAllSessionsForUser(userID int) (err error) {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 1. Delete all sessions for the user. ON DELETE CASCADE handles related data.
	if _, err = tx.Exec(`DELETE FROM chat_sessions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("failed to delete all sessions for user: %w", err)
	}

	// 2. Best-effort: Purge all vector memory for the user. Non-critical.
	purgeAllQuery := `
        DELETE FROM ego_memory 
        WHERE user_id = $1
        AND EXISTS (
            SELECT 1 FROM information_schema.tables 
            WHERE table_name = 'ego_memory'
        )`
	if _, err = tx.Exec(purgeAllQuery, fmt.Sprintf("%d", userID)); err != nil {
		log.Printf("[WARN] Failed to purge ego_memory for user %d during full cleanup (non-fatal): %v", userID, err)
		err = nil // Do not fail the transaction
	}

	return nil
}

// CheckSessionOwnership verifies if a given session belongs to the specified user.
func (db *DB) CheckSessionOwnership(sessionUUID string, userID int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM chat_sessions WHERE uuid = $1 AND user_id = $2)`
	err := db.Get(&exists, query, sessionUUID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check session ownership: %w", err)
	}
	return exists, nil
}

// GetOrCreateSession finds an existing session or creates a new one if it doesn't exist.
// It returns the session and a boolean `wasCreated` which is true if a new session was created.
func (db *DB) GetOrCreateSession(sessionUUID string, title string, userID int, mode string) (*models.ChatSession, bool, error) {
	// If a UUID is provided, attempt to fetch the existing session.
	if sessionUUID != "" {
		var session models.ChatSession
		query := "SELECT uuid, user_id, title, mode, custom_instructions, created_at FROM chat_sessions WHERE uuid = $1 AND user_id = $2"
		err := db.Get(&session, query, sessionUUID, userID)

		if err == nil {
			log.Printf("Found existing session %s for user %d", sessionUUID, userID)
			return &session, false, nil // Found, not created.
		}
		// If the error is anything other than "not found", it's a real problem.
		if err != sql.ErrNoRows {
			return nil, false, fmt.Errorf("error retrieving session: %w", err)
		}
	}

	// If we are here, it means either no UUID was provided, or the provided UUID was not found.
	// We proceed to create a new session.
	log.Printf("Creating new session for user %d with title '%s'", userID, title)
	if mode == "" {
		mode = defaultSessionMode
	}
	if title == "" {
		title = defaultSessionTitle
	}

	session := models.ChatSession{
		UserID:    userID,
		Title:     title,
		Mode:      mode,
		CreatedAt: time.Now().UTC(),
	}

	query := `INSERT INTO chat_sessions (user_id, title, mode, created_at) VALUES ($1, $2, $3, $4) RETURNING uuid`
	err := db.QueryRow(query, session.UserID, session.Title, session.Mode, session.CreatedAt).Scan(&session.UUID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create new session: %w", err)
	}

	return &session, true, nil // Created successfully.
}

// UpdateSessionInstructions updates the custom instructions for a specific session.
func (db *DB) UpdateSessionInstructions(sessionUUID string, userID int, customInstructions string) error {
	query := `UPDATE chat_sessions SET custom_instructions = $1 WHERE uuid = $2 AND user_id = $3`
	_, err := db.Exec(query, customInstructions, sessionUUID, userID)
	if err != nil {
		return fmt.Errorf("failed to update session instructions: %w", err)
	}
	return nil
}

// GetSessionByUUID retrieves a single session by its UUID, verifying user ownership.
// Returns (nil, nil) if the session is not found.
func (db *DB) GetSessionByUUID(sessionUUID string, userID int) (*models.ChatSession, error) {
	var session models.ChatSession
	query := "SELECT uuid, user_id, title, mode, custom_instructions, created_at FROM chat_sessions WHERE uuid = $1 AND user_id = $2"
	err := db.Get(&session, query, sessionUUID, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session by UUID: %w", err)
	}
	return &session, nil
}

// UpdateSessionTitle updates the title of a specific session.
func (db *DB) UpdateSessionTitle(sessionUUID string, userID int, title string) error {
	query := `UPDATE chat_sessions SET title = $1 WHERE uuid = $2 AND user_id = $3`
	_, err := db.Exec(query, title, sessionUUID, userID)
	if err != nil {
		return fmt.Errorf("failed to update session title: %w", err)
	}
	return nil
}

// CreateTempSession creates a new session with a default title and mode for a user.
// It returns the UUID of the newly created session.
func (db *DB) CreateTempSession(userID int) (string, error) {
	query := `INSERT INTO chat_sessions (user_id, title, mode, created_at)
	          VALUES ($1, $2, $3, $4) RETURNING uuid`
	var sessionUUID string
	err := db.QueryRow(query, userID, defaultSessionTitle, defaultSessionMode, time.Now().UTC()).Scan(&sessionUUID)
	if err != nil {
		return "", fmt.Errorf("failed to create temp session: %w", err)
	}
	log.Printf("Created temporary session %s for user %d", sessionUUID, userID)
	return sessionUUID, nil
}
