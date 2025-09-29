package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"egobackend/internal/config"
	"egobackend/internal/database"
	"egobackend/internal/engine"
	"egobackend/internal/models"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

const (
	writeWait         = 10 * time.Second    // Time allowed to write a message to the peer.
	pongWait          = 30 * time.Second    // Time allowed to read the next pong message from the peer.
	pingPeriod        = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait.
	maxMessageSize    = 64 * 1024 * 1024  // Maximum message size allowed from peer.
	sendEventTimeout  = 5 * time.Second     // Timeout for sending regular events (увеличено с 2 до 5).
	finalEventTimeout = 15 * time.Second    // A longer timeout for final events like 'done' or 'error' (увеличено).
)

// Message priority levels for queuing
const (
	PriorityCritical = 0 // done, error, log_created
	PriorityHigh     = 1 // chunk, session_created
	PriorityMedium   = 2 // thought_header
	PriorityLow      = 3 // tool_progress, pong
)

// priorityMessage wraps a message with its priority for queuing
type priorityMessage struct {
	data     []byte
	priority int
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan priorityMessage // Изменено на priorityMessage для приоритетной очереди
	user      *models.User
	processor *engine.Processor
	userDB    *database.DB
	validate  *validator.Validate
	cfg       *config.AppConfig
	connMutex sync.Mutex // Protects concurrent writes to the websocket connection.
}

// NewClient creates a new WebSocket client instance.
func NewClient(hub *Hub, conn *websocket.Conn, user *models.User, db *database.DB, processor *engine.Processor, validate *validator.Validate, cfg *config.AppConfig) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan priorityMessage, 4096), // Увеличен буфер и изменен тип
		user:      user,
		userDB:    db,
		processor: processor,
		validate:  validate,
		cfg:       cfg,
	}
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error for user %s: %v", c.user.Username, err)
			}
			break
		}
		// Process each message in a new goroutine to avoid blocking the read pump.
		go c.handleIncomingMessage(message)
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				c.write(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.write(websocket.TextMessage, message.data); err != nil {
				log.Printf("Error writing message to websocket for user %d: %v", c.user.ID, err)
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, nil); err != nil {
				log.Printf("Error writing ping to websocket for user %d: %v", c.user.ID, err)
				return
			}
		}
	}
}

// write is a helper to safely write messages to the connection.
func (c *Client) write(messageType int, data []byte) error {
	c.connMutex.Lock()
	defer c.connMutex.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(messageType, data)
}

// handleIncomingMessage dispatches incoming messages based on their 'type' field.
func (c *Client) handleIncomingMessage(message []byte) {
	var baseReq struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message, &baseReq); err != nil {
		c.sendEvent("error", map[string]string{"message": "Invalid JSON format"})
		return
	}

	switch baseReq.Type {
	case "stop":
		c.hub.cancelProcess <- uint(c.user.ID)
		c.sendEvent("done", "Stream stopped by user request.")
	case "ping":
		c.sendEvent("pong", nil)
	case "stream_request":
		c.processStreamRequest(message)
	default:
		c.sendEvent("error", map[string]string{"message": "Unknown message type"})
	}
}

// processStreamRequest handles a new request to start a generation stream.
func (c *Client) processStreamRequest(message []byte) {
	var req models.StreamRequest
	if err := json.Unmarshal(message, &req); err != nil {
		c.sendEvent("error", map[string]string{"message": "Invalid request format"})
		return
	}
	if err := c.validate.Struct(req); err != nil {
		c.sendEvent("error", map[string]string{"message": "Validation error: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Query) == "" && len(req.Files) == 0 {
		c.sendEvent("error", map[string]string{"message": "Request is empty: please provide a query or attach a file."})
		return
	}
	
	// Server-side guard for total file size.
	var totalBytes int64
	for _, f := range req.Files {
		totalBytes += int64(float64(len(f.Base64Data)) * 0.75)
		if totalBytes > maxMessageSize {
			c.sendEvent("error", map[string]string{"message": fmt.Sprintf("Total file size exceeds the limit of %dMB.", maxMessageSize/(1024*1024))})
			return
		}
	}
	
	// Debug logging for file transmission
	if len(req.Files) > 0 {
		log.Printf("[DEBUG] Received %d files in WebSocket request", len(req.Files))
		for i, f := range req.Files {
			log.Printf("[DEBUG] File %d: name='%s', mime='%s', size=%d bytes", i+1, f.FileName, f.MimeType, len(f.Base64Data))
		}
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	if req.MemoryEnabled != nil {
		ctx = context.WithValue(ctx, "memory_enabled", *req.MemoryEnabled)
	}

	// Register the cancel function with the hub so it can be called if the user sends a 'stop' message.
	c.hub.registerCancel <- cancelRequest{UserID: uint(c.user.ID), CancelFunc: cancel}
	
	// The callback function is the bridge between the long-running processor and the client.
	// It sends events back to ALL connected clients for this user.
	callback := func(eventType string, data interface{}) {
		c.hub.mu.RLock()
		userClients, ok := c.hub.clients[uint(c.user.ID)]
		c.hub.mu.RUnlock()

		if ok {
			for client := range userClients {
				client.sendEvent(eventType, data)
			}
		}

		// When the process is finished (either by completing or erroring out),
		// we no longer need the cancel function. The client will handle its removal.
		// NOTE: This logic is now inside the engine's finalize step for robustness.
	}

	c.processor.ProcessRequest(ctx, req, c.user, req.TempID, callback)
}

// getMessagePriority returns the priority level for a given event type.
func getMessagePriority(eventType string) int {
	switch eventType {
	case "done", "error", "log_created":
		return PriorityCritical
	case "chunk", "session_created", "session_title_updated":
		return PriorityHigh
	case "thought_header":
		return PriorityMedium
	case "tool_progress", "pong", "usage_update", "tool_call", "tool_output", "tool_error":
		return PriorityLow
	default:
		return PriorityMedium // Default to medium priority
	}
}

// sendEvent marshals and sends an event to the client's send channel.
// It uses a priority-based non-blocking send with a timeout to prevent a slow client
// from blocking the entire system. Critical messages are guaranteed delivery.
func (c *Client) sendEvent(eventType string, data interface{}) {
	eventData := map[string]interface{}{"type": eventType, "data": data}
	
	// Debug logging for thought_header events
	if eventType == "thought_header" {
		log.Printf("[DEBUG] Sending WebSocket event: type=%s, data=%v", eventType, data)
	}
	
	jsonEvent, err := json.Marshal(eventData)
	if err != nil {
		log.Printf("CRITICAL: Failed to marshal event to JSON: %v", err)
		return
	}

	priority := getMessagePriority(eventType)
	msg := priorityMessage{
		data:     jsonEvent,
		priority: priority,
	}

	timeout := sendEventTimeout
	if priority <= PriorityCritical {
		timeout = finalEventTimeout
	}
	
	// For critical messages, block and wait for delivery
	if priority == PriorityCritical {
		select {
		case c.send <- msg:
			return
		case <-time.After(timeout):
			log.Printf("CRITICAL: Failed to send critical event %s for user %d after %v", eventType, c.user.ID, timeout)
			// Try one more time in a goroutine to not block caller
			go func() {
				select {
				case c.send <- msg:
					log.Printf("CRITICAL: Delayed delivery of %s succeeded for user %d", eventType, c.user.ID)
				case <-time.After(timeout):
					log.Printf("CRITICAL: Delayed delivery of %s failed for user %d", eventType, c.user.ID)
				}
			}()
		}
		return
	}
	
	// For non-critical messages, try non-blocking send
	select {
	case c.send <- msg:
		// Successfully sent
	case <-time.After(timeout):
		// Only log warning for high priority messages
		if priority <= PriorityHigh {
			log.Printf("WARNING: WebSocket send channel full for user %d. Dropping %s event (priority %d)", c.user.ID, eventType, priority)
		}
	}
}

// closeConnection safely closes the send channel to terminate the WritePump.
func (c *Client) closeConnection() {
	// Use a non-blocking select to prevent double-closing the channel.
	select {
	case <-c.send:
		// Channel was already closed.
	default:
		close(c.send)
	}
}