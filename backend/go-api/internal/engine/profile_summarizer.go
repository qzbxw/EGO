package engine

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"egobackend/internal/database"
)

// ProfileSummarizer is a background worker that updates user profiles.
type ProfileSummarizer struct {
	db        *database.DB
	llmClient *llmClient
}

// NewProfileSummarizer creates a new summarizer.
func NewProfileSummarizer(db *database.DB, llm *llmClient) *ProfileSummarizer {
	return &ProfileSummarizer{
		db:        db,
		llmClient: llm,
	}
}

// Start begins the background loop.
func (ps *ProfileSummarizer) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("[ProfileSummarizer] Started background worker")

	for {
		select {
		case <-ctx.Done():
			log.Println("[ProfileSummarizer] Stopping...")
			return
		case <-ticker.C:
			if err := ps.runBatch(ctx); err != nil {
				log.Printf("[ProfileSummarizer] Batch error: %v", err)
			}
		}
	}
}

func (ps *ProfileSummarizer) runBatch(ctx context.Context) error {
	// 1. Get users who need an update
	userIDs, err := ps.db.GetUsersNeedingSummary(10) // Process 10 users at a time
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}
	if len(userIDs) == 0 {
		return nil
	}

	log.Printf("[ProfileSummarizer] Processing %d users", len(userIDs))

	for _, userID := range userIDs {
		// Create a separate timeout for each user to prevent one blocking the batch
		func() {
			userCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
			defer cancel()
			if err := ps.processUser(userCtx, userID); err != nil {
				log.Printf("[ProfileSummarizer] Failed to process user %d: %v", userID, err)
			}
		}()
	}
	return nil
}

func (ps *ProfileSummarizer) processUser(ctx context.Context, userID int) error {
	// 1. Fetch current profile
	currentProfile, err := ps.db.GetProfileSummary(userID)
	if err != nil {
		return err
	}

	// 2. Fetch "recent" logs. We need to know when the last summary was.
	// Since GetRecentLogsForUser needs a timestamp, let's fetch the user object again or trust the DB method.
	// Actually GetRecentLogsForUser is what we need, but we need the 'LastSummaryAt' timestamp.
	user, err := ps.db.GetUserByID(userID)
	if err != nil {
		return err
	}

	var lastSummaryTime time.Time
	if user.LastSummaryAt != nil {
		lastSummaryTime = *user.LastSummaryAt
	}
	// If never summarized, fetch all (or last 50 to avoid overflow)
	// For now, let's just fetch logs since that time.

	logs, err := ps.db.GetRecentLogsForUser(userID, lastSummaryTime)
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	if len(logs) == 0 {
		// No new logs, just update the timestamp so we don't check again immediately?
		// Actually GetUsersNeedingSummary checks for logs > last_summary_at, so this shouldn't happen
		// unless there's a race or I misunderstood the query.
		// If it happens, just update timestamp to now.
		return ps.db.UpdateProfileSummary(userID, currentProfile)
	}

	// 3. Format history
	var sb strings.Builder
	for _, l := range logs {
		resp := ""
		if l.FinalResponse != nil {
			resp = *l.FinalResponse
		}
		// Limit length of each message to save tokens
		sb.WriteString(fmt.Sprintf("User: %s\nEGO: %s\n\n", truncateString(l.UserQuery, 500), truncateString(resp, 500)))
	}
	history := sb.String()

	// 4. Generate new profile via LLM
	newProfile, err := ps.llmClient.GenerateProfileSummary(ctx, currentProfile, history)
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// 5. Save
	if err := ps.db.UpdateProfileSummary(userID, newProfile); err != nil {
		return fmt.Errorf("db update failed: %w", err)
	}

	log.Printf("[ProfileSummarizer] Updated profile for user %d. Length: %d", userID, len(newProfile))
	return nil
}
