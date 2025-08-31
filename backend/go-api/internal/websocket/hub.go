// Package websocket implements the WebSocket communication layer, including
// the central hub for managing client connections.
package websocket

import (
	"context"
	"log"
	"sync"
)

// Hub manages the lifecycle of all WebSocket clients. It handles registration,
// unregistration, and broadcasting messages, as well as managing cancellation
// signals for long-running processes initiated by clients.
type Hub struct {
	// A map of active clients, keyed by user ID. This allows for targeting
	// messages or actions to all connections from a specific user.
	clients map[uint]map[*Client]bool

	// A map of cancellation functions for active generation processes, keyed by user ID.
	// This allows the hub to cancel a running task (e.g., a long LLM generation)
	// when a user requests it or disconnects.
	cancelFuncs map[uint]context.CancelFunc

	// A single mutex to protect both clients and cancelFuncs maps.
	// Since all modifications happen within the single-threaded Run loop,
	// one mutex is sufficient and safer than multiple.
	mu sync.RWMutex

	// --- Channels for concurrent operations ---

	// register is a channel for new clients waiting to be added to the hub.
	register chan *Client
	// unregister is a channel for clients that have disconnected.
	unregister chan *Client
	// registerCancel is a channel for registering a new cancellation function for a user's process.
	registerCancel chan cancelRequest
	// cancelProcess is a channel for triggering the cancellation of a user's running process.
	cancelProcess chan uint
}

// cancelRequest is a struct used to safely register a new cancellation function.
type cancelRequest struct {
	UserID     uint
	CancelFunc context.CancelFunc
}

// NewHub creates and initializes a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[uint]map[*Client]bool),
		cancelFuncs:    make(map[uint]context.CancelFunc),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		registerCancel: make(chan cancelRequest),
		cancelProcess:  make(chan uint),
	}
}

// Register sends a client to the register channel for safe registration.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Run starts the central event loop for the Hub. It listens on its channels
// and processes client registrations, unregistrations, and cancellations.
// This method should be run as a goroutine.
func (h *Hub) Run() {
	log.Println("[WebSocket Hub] Hub is running.")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			userID := uint(client.user.ID)
			if _, ok := h.clients[userID]; !ok {
				h.clients[userID] = make(map[*Client]bool)
			}
			h.clients[userID][client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Client registered for user ID %d", userID)

		case client := <-h.unregister:
			h.mu.Lock()
			userID := uint(client.user.ID)
			if userClients, ok := h.clients[userID]; ok {
				if _, clientExists := userClients[client]; clientExists {
					delete(userClients, client)
					client.closeConnection() // Ensure the underlying connection is closed.
					if len(userClients) == 0 {
						// If this was the last client for the user, remove the user's entry.
						delete(h.clients, userID)
						// Also remove any lingering cancellation functions for this user.
						delete(h.cancelFuncs, userID)
						log.Printf("[WebSocket Hub] Last client for user ID %d disconnected.", userID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Client unregistered for user ID %d", userID)

		case req := <-h.registerCancel:
			h.mu.Lock()
			// Registering a new cancel function for a user. If one already exists,
			// it's overwritten. This is the desired behavior, as only the most recent
			// process should be cancellable.
			h.cancelFuncs[req.UserID] = req.CancelFunc
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Cancellation function registered for user ID %d.", req.UserID)

		case userID := <-h.cancelProcess:
			h.mu.RLock()
			cancelFunc, ok := h.cancelFuncs[userID]
			h.mu.RUnlock()
			
			if ok {
				log.Printf("[WebSocket Hub] Invoking cancellation for user ID %d.", userID)
				cancelFunc()
				// The function is removed when the process finishes or the user disconnects.
			}
		}
	}
}