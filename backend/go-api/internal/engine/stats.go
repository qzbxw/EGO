package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"egobackend/internal/models"
)

// requestStats aggregates per-request metrics for telemetry and database logging.
type requestStats struct {
	started          time.Time
	thoughtDurations []int64 // Milliseconds per thinking iteration.
	filesCount       int
	filesBytes       int64
	memoryEnabled    bool
	isRegeneration   bool
	promptTokens     int
	completionTokens int
	totalTokens      int
	usedModel        *string
}

// newRequestStats creates and initializes a new requestStats object.
func newRequestStats(ctx context.Context, req models.StreamRequest) *requestStats {
	stats := &requestStats{
		started:        time.Now(),
		isRegeneration: req.IsRegeneration,
		filesCount:     len(req.Files),
		memoryEnabled:  true, // Default to true
	}
	if req.MemoryEnabled != nil {
		stats.memoryEnabled = *req.MemoryEnabled
	}
	return stats
}

// updateFromUsage updates the token and model stats from a usage block received from Python.
func (st *requestStats) updateFromUsage(usage map[string]interface{}) {
	if usage == nil {
		return
	}

	// Safely extract token counts.
	if v, ok := usage["prompt_tokens"]; ok {
		if f, ok := v.(float64); ok {
			st.promptTokens = int(f)
		}
	}
	if v, ok := usage["completion_tokens"]; ok {
		if f, ok := v.(float64); ok {
			st.completionTokens = int(f)
		}
	}
	if v, ok := usage["total_tokens"]; ok {
		if f, ok := v.(float64); ok {
			st.totalTokens = int(f)
		}
	}
	// Safely extract metadata about the model used.
	if metaRaw, ok := usage["ego_meta"]; ok {
		if meta, ok := metaRaw.(map[string]interface{}); ok {
			if used, ok := meta["used"].(string); ok && used != "" {
				st.usedModel = &used
			}
		}
	}
}

// buildMetrics constructs the RequestMetrics DB model from the collected stats.
func (st *requestStats) buildMetrics(user *models.User, sessionUUID string, logID int64, startTime time.Time, thinkingDur, synthDur time.Duration) *models.RequestMetrics {
	provider := "ego"
	if user.LLMProvider != nil {
		provider = *user.LLMProvider
	}
	model := user.LLMModel
	if st.usedModel != nil {
		model = st.usedModel
	}

	return &models.RequestMetrics{
		UserID:             user.ID,
		SessionUUID:        sessionUUID,
		RequestLogID:       logID,
		ResponseTimeMs:     int(time.Since(startTime).Milliseconds()),
		ThinkingTimeMs:     int(thinkingDur.Milliseconds()),
		SynthesisTimeMs:    int(synthDur.Milliseconds()),
		ThinkingIterations: len(st.thoughtDurations),
		IsRegeneration:     st.isRegeneration,
		MemoryEnabled:      st.memoryEnabled,
		FilesCount:         st.filesCount,
		FilesSizeBytes:     st.filesBytes,
		LLMProvider:        provider,
		LLMModel:           model,
		PromptTokens:       st.promptTokens,
		CompletionTokens:   st.completionTokens,
		TotalTokens:        st.totalTokens,
		CreatedAt:          startTime,
	}
}

// composeTelemetryMessage creates a formatted Markdown string for sending to Telegram.
func (st *requestStats) composeTelemetryMessage(user *models.User, sessionUUID string, startTime time.Time, tThought, tSynth time.Duration, ok bool) string {
	ts := time.Now().In(time.FixedZone("UTC+3", 3*3600)).Format("2006-01-02 15:04:05")
	statusEmoji := "🟢"
	if !ok {
		statusEmoji = "🔴"
	}
	totalDur := time.Since(startTime)

	var b strings.Builder
	fmt.Fprintf(&b, "%s *EGO Request Stats* `%s`\n\n", statusEmoji, ts)
	fmt.Fprintf(&b, "👤 *User:* `#%d` | 🏷️ *Session:* `%s`\n", user.ID, sessionUUID[:8]+"...")
	fmt.Fprintf(&b, "🧠 *Memory:* %s | 🔄 *Regen:* %s\n",
		formatBool(st.memoryEnabled),
		formatBool(st.isRegeneration),
	)

	if st.usedModel != nil {
		fmt.Fprintf(&b, "🔌 *Model:* `%s`\n", *st.usedModel)
	}
	if st.totalTokens > 0 {
		fmt.Fprintf(&b, "🔢 Tokens: in `%d` • out `%d` • total `%d`\n", st.promptTokens, st.completionTokens, st.totalTokens)
	}

	if st.filesCount > 0 {
		fmt.Fprintf(&b, "📎 Files: `%d` • Size: `%s`\n", st.filesCount, formatBytes(st.filesBytes))
	}

	fmt.Fprintf(&b, "\n⏳ *Timing Breakdown*\n")
	if len(st.thoughtDurations) > 0 {
		var sum int64
		for _, v := range st.thoughtDurations {
			sum += v
		}
		avg := sum / int64(len(st.thoughtDurations))

		var parts []string
		for i := 0; i < len(st.thoughtDurations) && i < 8; i++ {
			parts = append(parts, fmt.Sprintf("%d", st.thoughtDurations[i]))
		}
		if len(st.thoughtDurations) > 8 {
			parts = append(parts, "…")
		}
		fmt.Fprintf(&b, "• 🧩 Thinking: `%dms` (%d iters, avg %dms)\n", tThought.Milliseconds(), len(st.thoughtDurations), avg)
	} else {
		fmt.Fprintf(&b, "• 🧩 Thinking: `%dms`\n", tThought.Milliseconds())
	}
	fmt.Fprintf(&b, "• 📝 Synthesis: `%dms`\n", tSynth.Milliseconds())
	fmt.Fprintf(&b, "• ⏱️ Total: `%dms`\n", totalDur.Milliseconds())

	totalMs := totalDur.Milliseconds()
	var speed string
	switch {
	case totalMs < 2000:
		speed = "🚀 Lightning"
	case totalMs < 6000:
		speed = "⚡ Fast"
	case totalMs < 20000:
		speed = "🐎 Normal"
	case totalMs < 45000:
		speed = "🐢 Slow"
	default:
		speed = "🐌 Very Slow"
	}
	fmt.Fprintf(&b, "• 💨 Speed: *%s*\n", speed)

	return b.String()
}

// --- Helper functions for formatting ---

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatBool(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}
