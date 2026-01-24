package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Job represents a background processing task.
type Job struct {
	ID          string
	UserID      int
	Ctx         context.Context
	Cancel      context.CancelFunc
	Subscribers map[chan []byte]struct{}
	mu          sync.RWMutex
	History     [][]byte // Buffer of all events for this job
	CreatedAt   time.Time
}

// StreamManager manages active background jobs.
type StreamManager struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewStreamManager creates a new StreamManager.
func NewStreamManager() *StreamManager {
	return &StreamManager{
		jobs: make(map[string]*Job),
	}
}

// CreateJob starts a new tracking job for a session.
// If a job already exists for this session, it cancels it.
func (sm *StreamManager) CreateJob(sessionID string, userID int) *Job {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if existing, ok := sm.jobs[sessionID]; ok {
		existing.Cancel() // Cancel the old context
		existing.CloseSubscribers()
		delete(sm.jobs, sessionID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	job := &Job{
		ID:          sessionID,
		UserID:      userID,
		Ctx:         ctx,
		Cancel:      cancel,
		Subscribers: make(map[chan []byte]struct{}),
		History:     make([][]byte, 0),
		CreatedAt:   time.Now(),
	}

	sm.jobs[sessionID] = job
	return job
}

// GetJob returns an existing job if available.
func (sm *StreamManager) GetJob(sessionID string) (*Job, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	job, ok := sm.jobs[sessionID]
	return job, ok
}

// RemoveJob removes the job from the manager.
func (sm *StreamManager) RemoveJob(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if job, ok := sm.jobs[sessionID]; ok {
		job.Cancel()
		job.CloseSubscribers()
		delete(sm.jobs, sessionID)
	}
}

// Subscribe adds a channel to receive events from the job and optionally replays history.
// It returns the channel and the current history snapshot.
func (j *Job) Subscribe() (chan []byte, [][]byte) {
	j.mu.Lock()
	defer j.mu.Unlock()

	ch := make(chan []byte, 100) // Buffer to prevent blocking
	j.Subscribers[ch] = struct{}{}

	// Return a copy of the current history
	historyCopy := make([][]byte, len(j.History))
	copy(historyCopy, j.History)

	return ch, historyCopy
}

// Unsubscribe removes a channel.
func (j *Job) Unsubscribe(ch chan []byte) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, ok := j.Subscribers[ch]; ok {
		delete(j.Subscribers, ch)
		close(ch)
	}
}

// Broadcast sends data to all subscribers and appends to history.
func (j *Job) Broadcast(eventType string, data interface{}) {
	j.mu.Lock() // Use Lock instead of RLock because we modify History
	defer j.mu.Unlock()

	eventData := map[string]interface{}{
		"type": eventType,
		"data": data,
	}
	jsonData, _ := json.Marshal(eventData)

	// Append to history
	j.History = append(j.History, jsonData)

	for ch := range j.Subscribers {
		select {
		case ch <- jsonData:
		default:
			// Drop message if client is too slow, but it's in history now.
		}
	}
}

// CloseSubscribers closes all subscriber channels.
func (j *Job) CloseSubscribers() {
	j.mu.Lock()
	defer j.mu.Unlock()
	for ch := range j.Subscribers {
		close(ch)
	}
	j.Subscribers = make(map[chan []byte]struct{})
}
