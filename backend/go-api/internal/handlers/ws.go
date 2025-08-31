package handlers

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"egobackend/internal/config"
	"egobackend/internal/database"
	"egobackend/internal/engine"
	"egobackend/internal/models"
	appwebsocket "egobackend/internal/websocket"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

// WSHandler handles the WebSocket connection lifecycle.
type WSHandler struct {
	Hub       *appwebsocket.Hub
	UserDB    *database.DB
	Processor *engine.Processor
	Validate  *validator.Validate
	Cfg       *config.AppConfig
	upgrader  websocket.Upgrader
}

// NewWSHandler creates a new WSHandler and configures the WebSocket upgrader.
func NewWSHandler(hub *appwebsocket.Hub, db *database.DB, processor *engine.Processor, validate *validator.Validate, cfg *config.AppConfig) *WSHandler {
	// allowedOrigins should be a comma-separated string in the .env file.
	origins := strings.Split(cfg.CORSAllowedOrigins, ",")
	
	upgrader := websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		// CheckOrigin validates the origin of the WebSocket request to prevent
		// Cross-Site WebSocket Hijacking (CSWH). It should only allow origins
		// from your frontend application.
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// Allow requests with no origin (e.g., from native clients or tools like Postman).
				return true
			}
			originURL, err := url.Parse(origin)
			if err != nil {
				return false
			}
			for _, allowed := range origins {
				if strings.EqualFold(allowed, originURL.String()) || strings.EqualFold(allowed, originURL.Hostname()) {
					return true
				}
			}
			log.Printf("WebSocket connection from disallowed origin rejected: %s", origin)
			return false
		},
	}
	
	return &WSHandler{
		Hub:       hub,
		UserDB:    db,
		Processor: processor,
		Validate:  validate,
		Cfg:       cfg,
		upgrader:  upgrader,
	}
}

// ServeWs handles the initial HTTP request and upgrades it to a WebSocket connection.
// It then creates a new client instance and registers it with the Hub.
func (h *WSHandler) ServeWs(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		// The auth middleware should have already caught this, but as a safeguard:
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// The upgrader writes a response to the client on error, so we just log it.
		log.Printf("WebSocket upgrade failed for user %d: %v", user.ID, err)
		return
	}

	// Create a new client to handle this specific connection.
	client := appwebsocket.NewClient(h.Hub, conn, user, h.UserDB, h.Processor, h.Validate, h.Cfg)
	// Register the client with the hub, which will manage its lifecycle.
	h.Hub.Register(client)

	// Start the client's read and write pumps in separate goroutines.
	// These will handle all further communication over the WebSocket.
	go client.WritePump()
	go client.ReadPump()

	log.Printf("WebSocket client connected for user %d (%s)", user.ID, user.Username)
}