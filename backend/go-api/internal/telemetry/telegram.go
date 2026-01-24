// Package telemetry provides an administrative Telegram bot for monitoring
// and managing the application.
package telemetry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"egobackend/internal/database"
)

const (
	telegramAPIURL = "https://api.telegram.org/bot%s/%s"
	pollingTimeout = 30 * time.Second
	requestTimeout = 10 * time.Second
)

var botInstance *TelegramBot

// TelegramBot handles communication with the Telegram Bot API.
type TelegramBot struct {
	token  string
	chatID string
	db     *database.DB
	client *http.Client
}

// TelegramUpdate represents a single update received from the Telegram API.
type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

// --- Public Interface ---

// InitializeBot creates and starts the global Telegram bot instance.
func InitializeBot(db *database.DB) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" || chatID == "" {
		log.Println("[Telegram] Admin bot is disabled: TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is not set.")
		return
	}

	botInstance = &TelegramBot{
		token:  token,
		chatID: chatID,
		db:     db,
		client: &http.Client{Timeout: pollingTimeout + 5*time.Second},
	}
	botInstance.startPolling()
	log.Println("[Telegram] Admin bot initialized and started polling for commands.")
}

// Send dispatches a message to the configured Telegram chat.
func Send(text string) {
	if botInstance == nil || text == "" {
		return
	}
	botInstance.sendMessage(text)
}

// GetBotInstance returns the global bot instance. It returns nil if the bot is not initialized.
// This is used to check if telemetry is enabled before sending messages.
func GetBotInstance() *TelegramBot {
	return botInstance
}

func LogAuthSuccess(username string, userID int, ipAddress string) {
	if botInstance == nil {
		return
	}
	message := fmt.Sprintf("🟢 *AUTH SUCCESS*\nUser: %s\nID Hash: `%s`\nIP: %s\nTime: %s",
		maskUsername(username), hashUserID(strconv.Itoa(userID)), ipAddress, getCurrentTime())
	botInstance.sendMessage(message)
}

func LogAuthFailure(username, reason, ipAddress string) {
	if botInstance == nil {
		return
	}
	message := fmt.Sprintf("🔴 *AUTH FAILED*\nUser: %s\nReason: %s\nIP: %s\nTime: %s",
		maskUsername(username), reason, ipAddress, getCurrentTime())
	botInstance.sendMessage(message)
}

func LogTokenRefresh(userID int, success bool, ipAddress string) {
	if botInstance == nil {
		return
	}
	var emoji, status string
	if success {
		emoji, status = "🔄", "SUCCESS"
	} else {
		emoji, status = "❌", "FAILED"
	}
	message := fmt.Sprintf("%s *TOKEN REFRESH %s*\nUser ID Hash: `%s`\nIP: %s\nTime: %s",
		emoji, status, hashUserID(strconv.Itoa(userID)), ipAddress, getCurrentTime())
	botInstance.sendMessage(message)
}

func LogGoogleAuth(username string, userID int, success bool, ipAddress string) {
	if botInstance == nil {
		return
	}
	var emoji, status string
	if success {
		emoji, status = "🟢", "GOOGLE AUTH SUCCESS"
	} else {
		emoji, status = "🔴", "GOOGLE AUTH FAILED"
	}
	message := fmt.Sprintf("%s *%s*\nUser: %s\nID Hash: `%s`\nIP: %s\nTime: %s",
		emoji, status, maskUsername(username), hashUserID(strconv.Itoa(userID)), ipAddress, getCurrentTime())
	botInstance.sendMessage(message)
}

// --- Internal Bot Logic ---

func (b *TelegramBot) sendMessage(text string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Telegram] Recovered from panic in sendMessage: %v", r)
			}
		}()

		payload, _ := json.Marshal(map[string]string{
			"chat_id":    b.chatID,
			"text":       text,
			"parse_mode": "Markdown",
		})

		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		url := fmt.Sprintf(telegramAPIURL, b.token, "sendMessage")
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
		if err != nil {
			log.Printf("[Telegram] Error creating request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := b.client.Do(req)
		if err != nil {
			log.Printf("[Telegram] Error sending message: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}

func (b *TelegramBot) startPolling() {
	go func() {
		offset := 0
		for {
			updates, err := b.getUpdates(offset)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				log.Printf("[Telegram] Error getting updates: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, update := range updates {
				b.handleUpdate(update)
				if update.UpdateID >= offset {
					offset = update.UpdateID + 1
				}
			}
		}
	}()
}

func (b *TelegramBot) getUpdates(offset int) ([]TelegramUpdate, error) {
	url := fmt.Sprintf(telegramAPIURL, b.token, fmt.Sprintf("getUpdates?offset=%d&timeout=%.0f", offset, pollingTimeout.Seconds()))

	ctx, cancel := context.WithTimeout(context.Background(), pollingTimeout+5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Result []TelegramUpdate `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Telegram response: %w", err)
	}
	return response.Result, nil
}

func (b *TelegramBot) handleUpdate(update TelegramUpdate) {
	if update.Message == nil || update.Message.Chat.ID != mustParseInt64(b.chatID) {
		return
	}

	command, args := parseCommand(update.Message.Text)

	switch command {
	case "/stats":
		if args == "" {
			b.handleGlobalStats()
		} else {
			b.handleUserStats(args)
		}
	case "/down":
		b.handleMaintenanceDown(args, false)
	case "/downego":
		b.handleMaintenanceDown(args, true)
	case "/up", "/upego":
		b.handleMaintenanceUp()
	case "/status":
		b.handleMaintenanceStatus()
	}
}

// --- Command Handlers ---

func (b *TelegramBot) handleGlobalStats() {
	stats, err := b.db.GetGlobalStatistics()
	if err != nil {
		b.sendMessage("Error getting global statistics:\n\n`" + err.Error() + "`")
		return
	}

	loc := time.FixedZone("UTC+3", 3*3600)
	var sb strings.Builder
	fmt.Fprintf(&sb, "🌐 *EGO Global Statistics*\n")
	fmt.Fprintf(&sb, "`%s`\n\n", time.Now().In(loc).Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "👥 Users: `%s`\n", formatNumber(int64(stats.TotalUsers)))
	fmt.Fprintf(&sb, "🧾 Requests: `%s`\n", formatNumber(stats.TotalRequests))
	fmt.Fprintf(&sb, "🗂️ Sessions: `%s`\n", formatNumber(stats.TotalSessions))
	fmt.Fprintf(&sb, "🔡 Tokens: `%s`\n", formatNumber(stats.TotalTokens))
	fmt.Fprintf(&sb, "📎 Files: `%s` (%.1f MB)\n", formatNumber(stats.TotalFilesUploaded), stats.TotalFilesSizeMB)
	fmt.Fprintf(&sb, "\n*Performance:*\n")
	fmt.Fprintf(&sb, "• Avg Response: `%dms`\n", stats.AvgResponseTimeMs)
	fmt.Fprintf(&sb, "\n*Active Users:*\n")
	fmt.Fprintf(&sb, "• Today: `%d`\n", stats.ActiveUsersToday)
	fmt.Fprintf(&sb, "• This Week: `%d`\n", stats.ActiveUsersThisWeek)
	fmt.Fprintf(&sb, "• This Month: `%d`\n", stats.ActiveUsersThisMonth)

	if len(stats.TopProviders) > 0 {
		fmt.Fprintf(&sb, "\n*Top Providers (by requests):*\n")
		for provider, count := range stats.TopProviders {
			fmt.Fprintf(&sb, "• %s: `%s`\n", provider, formatNumber(int64(count)))
		}
	}

	if rows, err := b.db.GetGlobalProviderTokenUsage(); err == nil && len(rows) > 0 {
		fmt.Fprintf(&sb, "\n*Tokens by provider:*\n")
		for _, r := range rows {
			fmt.Fprintf(&sb, "• %s: total `%s` (in %s | out %s)\n", r.Provider, formatNumber(r.TotalTokens), formatNumber(r.PromptTokens), formatNumber(r.CompletionTokens))
		}
	}
	b.sendMessage(sb.String())
}

func (b *TelegramBot) handleUserStats(userIDStr string) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		b.sendMessage("❌ *Error:* Invalid User ID.\n\nUsage: `/stats [UserID]`")
		return
	}

	stats, err := b.db.GetUserStatistics(userID)
	if err != nil {
		b.sendMessage(fmt.Sprintf("❌ *Error getting stats for user #%d*\n\n`%s`", userID, err.Error()))
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "👤 *User Statistics for %s (#%d)*\n\n", stats.Username, userID)
	fmt.Fprintf(&sb, "*Usage Overview:*\n")
	fmt.Fprintf(&sb, "• Requests: `%s` (`%.1f/day`)\n", formatNumber(int64(stats.TotalRequests)), stats.RequestsPerDay)
	fmt.Fprintf(&sb, "• Sessions: `%s`\n", formatNumber(int64(stats.TotalSessions)))
	fmt.Fprintf(&sb, "• Tokens: `%s` (in: %s | out: %s)\n", formatNumber(stats.TotalTokens), formatNumber(stats.TotalPromptTokens), formatNumber(stats.TotalCompletionTokens))

	if stats.FirstRequestAt != nil {
		fmt.Fprintf(&sb, "\n*Timeline:*\n")
		fmt.Fprintf(&sb, "• Days Active: `%d`\n", stats.DaysActive)
		fmt.Fprintf(&sb, "• First Request: `%s`\n", stats.FirstRequestAt.Format("2006-01-02"))
		if stats.LastRequestAt != nil {
			fmt.Fprintf(&sb, "• Last Request: `%s`\n", stats.LastRequestAt.Format("2006-01-02 15:04"))
		}
	}
	b.sendMessage(sb.String())
}

func (b *TelegramBot) handleMaintenanceDown(args string, chatOnly bool) {
	// Fixed message for security and consistency
	fixedMessage := "Service is temporarily unavailable for maintenance."
	if chatOnly {
		fixedMessage = "AI Chat is temporarily disabled for maintenance. Other functions are available."
	}

	bypassToken, err := b.db.EnableMaintenanceMode(fixedMessage, chatOnly)
	if err != nil {
		b.sendMessage("❌ Error enabling maintenance mode:\n\n`" + err.Error() + "`")
		return
	}
	var sb strings.Builder
	if chatOnly {
		fmt.Fprintf(&sb, "🤖 *Chat Maintenance: ENABLED*\n\n")
	} else {
		fmt.Fprintf(&sb, "🔒 *Full Maintenance Mode: ENABLED*\n\n")
	}
	fmt.Fprintf(&sb, "Display message: *%s*\n\n", fixedMessage)
	fmt.Fprintf(&sb, "🔑 *Bypass Token:*\n`%s`\n\n", bypassToken)
	fmt.Fprintf(&sb, "To disable, use the `/up` (or `/upego`) command.")
	b.sendMessage(sb.String())
}

func (b *TelegramBot) handleMaintenanceUp() {
	if err := b.db.DisableMaintenanceMode(); err != nil {
		b.sendMessage("❌ Error disabling maintenance mode:\n\n`" + err.Error() + "`")
		return
	}
	b.sendMessage("✅ *Maintenance Mode: DISABLED*\n\nThe service is now available to all users.")
}

func (b *TelegramBot) handleMaintenanceStatus() {
	status, err := b.db.GetMaintenanceMode()
	if err != nil {
		b.sendMessage("❌ Error getting maintenance status:\n\n`" + err.Error() + "`")
		return
	}
	var sb strings.Builder
	if status.IsEnabled {
		if status.IsChatOnly {
			fmt.Fprintf(&sb, "🤖 *Chat Maintenance: ENABLED*\n\n")
		} else {
			fmt.Fprintf(&sb, "🔒 *Full Maintenance Mode: ENABLED*\n\n")
		}
		if status.Message != nil {
			fmt.Fprintf(&sb, "Message: *%s*\n", *status.Message)
		}
		if status.EnabledAt != nil {
			fmt.Fprintf(&sb, "Enabled at: `%s`\n", status.EnabledAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Fprintf(&sb, "✅ *Maintenance Mode: DISABLED*\n\n")
		if status.DisabledAt != nil {
			fmt.Fprintf(&sb, "Disabled at: `%s`\n", status.DisabledAt.Format("2006-01-02 15:04:05"))
		}
	}
	b.sendMessage(sb.String())
}

// --- Helper Functions ---

func parseCommand(text string) (string, string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func maskUsername(username string) string {
	if len(username) <= 4 {
		return strings.Repeat("*", len(username))
	}
	return username[:2] + strings.Repeat("*", len(username)-4) + username[len(username)-2:]
}

func hashUserID(userID string) string {
	hash := sha256.Sum256([]byte(userID))
	return hex.EncodeToString(hash[:])[:8]
}

func getCurrentTime() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
}

func mustParseInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func formatNumber(num int64) string {
	if num < 1000 {
		return fmt.Sprintf("%d", num)
	}
	if num < 1000000 {
		return fmt.Sprintf("%.1fK", float64(num)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(num)/1000000)
}
