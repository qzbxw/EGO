// This file contains database methods for saving, calculating, and retrieving user and global statistics.
package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"egobackend/internal/models"
)

// SaveRequestMetrics saves the metrics for a single request and triggers an efficient,
// incremental update of the user's aggregated statistics.
func (db *DB) SaveRequestMetrics(metrics *models.RequestMetrics) error {
	query := `
		INSERT INTO request_metrics (
			user_id, session_uuid, request_log_id, response_time_ms, thinking_time_ms,
			synthesis_time_ms, thinking_iterations, is_regeneration, memory_enabled,
			files_count, files_size_bytes, llm_provider, llm_model, prompt_tokens,
			completion_tokens, total_tokens, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (request_log_id) DO UPDATE SET
			response_time_ms = EXCLUDED.response_time_ms,
			thinking_time_ms = EXCLUDED.thinking_time_ms,
			synthesis_time_ms = EXCLUDED.synthesis_time_ms,
			thinking_iterations = EXCLUDED.thinking_iterations,
			prompt_tokens = EXCLUDED.prompt_tokens,
			completion_tokens = EXCLUDED.completion_tokens,
			total_tokens = EXCLUDED.total_tokens`

	_, err := db.Exec(query,
		metrics.UserID, metrics.SessionUUID, metrics.RequestLogID, metrics.ResponseTimeMs,
		metrics.ThinkingTimeMs, metrics.SynthesisTimeMs, metrics.ThinkingIterations,
		metrics.IsRegeneration, metrics.MemoryEnabled, metrics.FilesCount,
		metrics.FilesSizeBytes, metrics.LLMProvider, metrics.LLMModel,
		metrics.PromptTokens, metrics.CompletionTokens, metrics.TotalTokens, metrics.CreatedAt,
	)
	if err != nil {
		log.Printf("ERROR saving request metrics: %v", err)
		return fmt.Errorf("failed to save request metrics: %w", err)
	}

	return db.updateUserStatisticsIncrementally(metrics)
}

func (db *DB) updateUserStatisticsIncrementally(m *models.RequestMetrics) error {
	providerKey := m.LLMProvider
	// Use sql.NullString so the driver binds a concrete text type even when NULL
	var modelKey sql.NullString
	if m.LLMModel != nil {
		modelKey = sql.NullString{String: *m.LLMModel, Valid: true}
	} else {
		modelKey = sql.NullString{Valid: false}
	}

	query := `
		INSERT INTO user_statistics (
			user_id, total_requests, total_sessions, total_files_uploaded, total_files_size_bytes,
			total_prompt_tokens, total_completion_tokens, total_tokens, total_response_time_ms,
			min_response_time_ms, max_response_time_ms,
			total_thinking_iterations, total_regenerations, memory_enabled_requests,
			provider_usage, model_usage, first_request_at, last_request_at, updated_at
		) VALUES (
			$1, 1, 1, $2, $3, $4, $5, $6, $7,
			$8, $8,
			$9, 
			CASE WHEN $10 THEN 1 ELSE 0 END,
			CASE WHEN $11 THEN 1 ELSE 0 END,
			jsonb_build_object(($12)::text, 1),
			CASE WHEN ($13)::text IS NOT NULL THEN jsonb_build_object(($13)::text, 1) ELSE '{}'::jsonb END,
			$14, $14, NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			total_requests = user_statistics.total_requests + 1,
			total_files_uploaded = user_statistics.total_files_uploaded + EXCLUDED.total_files_uploaded,
			total_files_size_bytes = user_statistics.total_files_size_bytes + EXCLUDED.total_files_size_bytes,
			total_prompt_tokens = user_statistics.total_prompt_tokens + EXCLUDED.total_prompt_tokens,
			total_completion_tokens = user_statistics.total_completion_tokens + EXCLUDED.total_completion_tokens,
			total_tokens = user_statistics.total_tokens + EXCLUDED.total_tokens,
			total_response_time_ms = user_statistics.total_response_time_ms + EXCLUDED.total_response_time_ms,
			min_response_time_ms = COALESCE(
				LEAST(NULLIF(user_statistics.min_response_time_ms, 0), EXCLUDED.min_response_time_ms),
				EXCLUDED.min_response_time_ms
			),
			max_response_time_ms = GREATEST(user_statistics.max_response_time_ms, EXCLUDED.max_response_time_ms),
			total_thinking_iterations = user_statistics.total_thinking_iterations + EXCLUDED.total_thinking_iterations,
			total_regenerations = user_statistics.total_regenerations + EXCLUDED.total_regenerations,
			memory_enabled_requests = user_statistics.memory_enabled_requests + EXCLUDED.memory_enabled_requests,
			provider_usage = user_statistics.provider_usage || jsonb_build_object(
				($12)::text,
				(COALESCE(user_statistics.provider_usage ->> ($12)::text, '0')::int + 1)
			),
			model_usage = CASE
				WHEN ($13)::text IS NOT NULL THEN user_statistics.model_usage || jsonb_build_object(
					($13)::text,
					(COALESCE(user_statistics.model_usage ->> ($13)::text, '0')::int + 1)
				)
				ELSE user_statistics.model_usage
			END,
			last_request_at = EXCLUDED.last_request_at,
			updated_at = NOW()
	`

	_, err := db.Exec(query,
		m.UserID,             // $1
		m.FilesCount,         // $2
		m.FilesSizeBytes,     // $3
		m.PromptTokens,       // $4
		m.CompletionTokens,   // $5
		m.TotalTokens,        // $6
		m.ResponseTimeMs,     // $7 total_response_time_ms increment
		m.ResponseTimeMs,     // $8 min_response_time_ms (on insert)
		m.ThinkingIterations, // $9 total_thinking_iterations increment
		m.IsRegeneration,     // $10 total_regenerations increment (boolean to CASE)
		m.MemoryEnabled,      // $11 memory_enabled_requests increment (boolean to CASE)
		providerKey,          // $12 provider key
		modelKey,             // $13 model key
		m.CreatedAt,          // $14 first/last request at
	)

	if err != nil {
		log.Printf("ERROR incrementally updating user statistics for user %d: %v", m.UserID, err)
		return fmt.Errorf("failed to update user statistics: %w", err)
	}
	return nil
}

func (db *DB) GetUserStatistics(userID int) (*models.UserStatsResponse, error) {
	var stats models.UserStatistics
	query := `
		SELECT
			user_id, total_requests, total_sessions, total_files_uploaded, total_files_size_bytes,
			total_prompt_tokens, total_completion_tokens, total_tokens,
			CASE WHEN total_requests > 0 THEN total_response_time_ms / total_requests ELSE 0 END AS avg_response_time_ms,
			min_response_time_ms, max_response_time_ms,
			total_response_time_ms, total_thinking_iterations,
			CASE WHEN total_requests > 0 THEN total_thinking_iterations::float / total_requests ELSE 0 END AS avg_thinking_iterations,
			total_regenerations, memory_enabled_requests, provider_usage, model_usage,
			first_request_at, last_request_at
		FROM user_statistics WHERE user_id = $1`

	err := db.Get(&stats, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no stats exist yet, try to build them for the first time.
			// This covers users who existed before the stats table was added.
			return &models.UserStatsResponse{UserID: userID, ProviderUsage: map[string]int{}, ModelUsage: map[string]int{}}, nil
		}
		return nil, fmt.Errorf("failed to get user statistics: %w", err)
	}

	var username string
	if err := db.Get(&username, `SELECT username FROM users WHERE id = $1`, userID); err != nil {
		return nil, fmt.Errorf("failed to get username for stats: %w", err)
	}

	recentActivity, err := db.GetUserRecentActivity(userID, 30)
	if err != nil {
		log.Printf("Warning: failed to get recent activity for user %d: %v", userID, err)
	}

	var daysActive int
	var requestsPerDay float64
	if stats.FirstRequestAt != nil && stats.LastRequestAt != nil {
		days := stats.LastRequestAt.Sub(*stats.FirstRequestAt).Hours() / 24
		if days < 1 {
			daysActive = 1
		} else {
			daysActive = int(days) + 1
		}
		if daysActive > 0 {
			requestsPerDay = float64(stats.TotalRequests) / float64(daysActive)
		}
	}

	var memoryUsagePercent float64
	if stats.TotalRequests > 0 {
		memoryUsagePercent = (float64(stats.MemoryEnabledRequests) / float64(stats.TotalRequests)) * 100
	}

	providerUsage := make(map[string]int)
	json.Unmarshal(stats.ProviderUsage, &providerUsage)
	modelUsage := make(map[string]int)
	json.Unmarshal(stats.ModelUsage, &modelUsage)

	return &models.UserStatsResponse{
		UserID:                  stats.UserID,
		Username:                username,
		TotalRequests:           stats.TotalRequests,
		TotalSessions:           stats.TotalSessions,
		TotalFilesUploaded:      stats.TotalFilesUploaded,
		TotalFilesSizeMB:        float64(stats.TotalFilesSizeBytes) / (1024 * 1024),
		TotalPromptTokens:       stats.TotalPromptTokens,
		TotalCompletionTokens:   stats.TotalCompletionTokens,
		TotalTokens:             stats.TotalTokens,
		AvgResponseTimeMs:       stats.AvgResponseTimeMs,
		MinResponseTimeMs:       stats.MinResponseTimeMs,
		MaxResponseTimeMs:       stats.MaxResponseTimeMs,
		TotalThinkingIterations: stats.TotalThinkingIterations,
		AvgThinkingIterations:   stats.AvgThinkingIterations,
		TotalRegenerations:      stats.TotalRegenerations,
		MemoryEnabledRequests:   stats.MemoryEnabledRequests,
		MemoryUsagePercent:      memoryUsagePercent,
		ProviderUsage:           providerUsage,
		ModelUsage:              modelUsage,
		FirstRequestAt:          stats.FirstRequestAt,
		LastRequestAt:           stats.LastRequestAt,
		DaysActive:              daysActive,
		RequestsPerDay:          requestsPerDay,
		RecentActivity:          recentActivity,
	}, nil
}

func (db *DB) GetUserRecentActivity(userID int, days int) ([]models.DailyActivitySummary, error) {
	query := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as requests,
			COALESCE(SUM(total_tokens), 0) as tokens
		FROM request_metrics
		WHERE user_id = $1 AND created_at >= NOW() - ($2 * INTERVAL '1 day')
		GROUP BY DATE(created_at)
		ORDER BY date DESC`

	var rows []struct {
		Date     time.Time `db:"date"`
		Requests int       `db:"requests"`
		Tokens   int64     `db:"tokens"`
	}

	err := db.Select(&rows, query, userID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent user activity: %w", err)
	}

	activity := make([]models.DailyActivitySummary, len(rows))
	for i, row := range rows {
		activity[i] = models.DailyActivitySummary{
			Date:     row.Date.Format("2006-01-02"),
			Requests: row.Requests,
			Tokens:   row.Tokens,
		}
	}
	return activity, nil
}

func (db *DB) GetGlobalStatistics() (*models.GlobalStatsResponse, error) {
	var stats struct {
		TotalUsers           int   `db:"total_users"`
		TotalRequests        int64 `db:"total_requests"`
		TotalSessions        int64 `db:"total_sessions"`
		TotalTokens          int64 `db:"total_tokens"`
		AvgResponseTimeMs    int   `db:"avg_response_time_ms"`
		TotalFilesUploaded   int64 `db:"total_files_uploaded"`
		TotalFilesSizeBytes  int64 `db:"total_files_size_bytes"`
		ActiveUsersToday     int   `db:"active_users_today"`
		ActiveUsersThisWeek  int   `db:"active_users_this_week"`
		ActiveUsersThisMonth int   `db:"active_users_this_month"`
	}

	query := `
		SELECT
			(SELECT COUNT(*) FROM users) as total_users,
			(SELECT COALESCE(SUM(total_requests), 0) FROM user_statistics) as total_requests,
			(SELECT COALESCE(SUM(total_sessions), 0) FROM user_statistics) as total_sessions,
			(SELECT COALESCE(SUM(total_tokens), 0) FROM user_statistics) as total_tokens,
			(SELECT COALESCE(SUM(total_response_time_ms), 0) / NULLIF(SUM(total_requests), 0) FROM user_statistics)::INTEGER as avg_response_time_ms,
			(SELECT COALESCE(SUM(total_files_uploaded), 0) FROM user_statistics) as total_files_uploaded,
			(SELECT COALESCE(SUM(total_files_size_bytes), 0) FROM user_statistics) as total_files_size_bytes,
			(SELECT COUNT(DISTINCT user_id) FROM request_metrics WHERE created_at >= CURRENT_DATE) as active_users_today,
			(SELECT COUNT(DISTINCT user_id) FROM request_metrics WHERE created_at >= CURRENT_DATE - INTERVAL '7 days') as active_users_this_week,
			(SELECT COUNT(DISTINCT user_id) FROM request_metrics WHERE created_at >= CURRENT_DATE - INTERVAL '30 days') as active_users_this_month`

	if err := db.Get(&stats, query); err != nil {
		return nil, fmt.Errorf("failed to get global stats aggregates: %w", err)
	}

	var providerRows []struct {
		Provider string `db:"llm_provider"`
		Count    int    `db:"count"`
	}
	err := db.Select(&providerRows, `SELECT llm_provider, COUNT(*) as count FROM request_metrics GROUP BY llm_provider ORDER BY count DESC LIMIT 10`)
	if err != nil {
		return nil, err
	}
	topProviders := make(map[string]int)
	for _, row := range providerRows {
		topProviders[row.Provider] = row.Count
	}

	var modelRows []struct {
		Model *string `db:"llm_model"`
		Count int     `db:"count"`
	}
	err = db.Select(&modelRows, `SELECT llm_model, COUNT(*) as count FROM request_metrics WHERE llm_model IS NOT NULL GROUP BY llm_model ORDER BY count DESC LIMIT 10`)
	if err != nil {
		return nil, err
	}
	topModels := make(map[string]int)
	for _, row := range modelRows {
		if row.Model != nil {
			topModels[*row.Model] = row.Count
		}
	}

	recentActivity, err := db.GetGlobalRecentActivity(30)
	if err != nil {
		log.Printf("Warning: failed to get global recent activity: %v", err)
	}

	return &models.GlobalStatsResponse{
		TotalUsers:           stats.TotalUsers,
		TotalRequests:        stats.TotalRequests,
		TotalSessions:        stats.TotalSessions,
		TotalTokens:          stats.TotalTokens,
		AvgResponseTimeMs:    stats.AvgResponseTimeMs,
		TotalFilesUploaded:   stats.TotalFilesUploaded,
		TotalFilesSizeMB:     float64(stats.TotalFilesSizeBytes) / (1024 * 1024),
		TopProviders:         topProviders,
		TopModels:            topModels,
		ActiveUsersToday:     stats.ActiveUsersToday,
		ActiveUsersThisWeek:  stats.ActiveUsersThisWeek,
		ActiveUsersThisMonth: stats.ActiveUsersThisMonth,
		RecentActivity:       recentActivity,
	}, nil
}

func (db *DB) GetAllUsersStatistics() ([]models.UserStatsResponse, error) {
	var users []struct {
		ID       int    `db:"id"`
		Username string `db:"username"`
	}
	if err := db.Select(&users, `SELECT id, username FROM users ORDER BY id`); err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	var results []models.UserStatsResponse
	for _, user := range users {
		stats, err := db.GetUserStatistics(user.ID)
		if err != nil {
			log.Printf("Warning: failed to get stats for user %s (%d): %v", user.Username, user.ID, err)
			continue
		}
		if stats != nil {
			results = append(results, *stats)
		}
	}
	return results, nil
}

func (db *DB) GetUserProviderTokenUsage(userID int) ([]models.ProviderTokenUsageRow, error) {
	rows := []models.ProviderTokenUsageRow{}
	query := `
		SELECT llm_provider,
		       COALESCE(SUM(prompt_tokens), 0)     AS prompt_tokens,
		       COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
		       COALESCE(SUM(total_tokens), 0)      AS total_tokens
		FROM request_metrics
		WHERE user_id = $1
		GROUP BY llm_provider
		ORDER BY total_tokens DESC`
	if err := db.Select(&rows, query, userID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (db *DB) GetGlobalProviderTokenUsage() ([]models.ProviderTokenUsageRow, error) {
	rows := []models.ProviderTokenUsageRow{}
	query := `
		SELECT llm_provider,
		       COALESCE(SUM(prompt_tokens), 0)     AS prompt_tokens,
		       COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
		       COALESCE(SUM(total_tokens), 0)      AS total_tokens
		FROM request_metrics
		GROUP BY llm_provider
		ORDER BY total_tokens DESC`
	if err := db.Select(&rows, query); err != nil {
		return nil, err
	}
	return rows, nil
}

// GetGlobalRecentActivity returns a summary of system-wide activity for the last N days.
func (db *DB) GetGlobalRecentActivity(days int) ([]models.DailyActivitySummary, error) {
	query := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as requests,
			COALESCE(SUM(total_tokens), 0) as tokens
		FROM request_metrics
		WHERE created_at >= NOW() - ($1 * INTERVAL '1 day')
		GROUP BY DATE(created_at)
		ORDER BY date DESC`

	var rows []struct {
		Date     time.Time `db:"date"`
		Requests int       `db:"requests"`
		Tokens   int64     `db:"tokens"`
	}

	err := db.Select(&rows, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get global recent activity: %w", err)
	}

	activity := make([]models.DailyActivitySummary, len(rows))
	for i, row := range rows {
		activity[i] = models.DailyActivitySummary{
			Date:     row.Date.Format("2006-01-02"),
			Requests: row.Requests,
			Tokens:   row.Tokens,
		}
	}
	return activity, nil
}
