package fsm

import (
	"sync"
	"time"
)

const (
	sessionTTL      = 30 * time.Minute
	cleanupInterval = 5 * time.Minute
)

// Session holds the FSM state for a single user.
type Session struct {
	State     State
	Data      map[string]any
	UpdatedAt time.Time
}

// Store is a thread-safe in-memory store for user FSM sessions.
type Store struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
	stop     chan struct{}
}

// NewStore creates a Store and starts the background cleanup goroutine.
func NewStore() *Store {
	s := &Store{
		sessions: make(map[int64]*Session),
		stop:     make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Get returns the session for userID and true, or nil and false if not found.
func (s *Store) Get(userID int64) (*Session, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[userID]
	s.mu.RUnlock()
	return sess, ok
}

// Set stores or replaces the session for userID.
func (s *Store) Set(userID int64, sess *Session) {
	sess.UpdatedAt = time.Now()
	s.mu.Lock()
	s.sessions[userID] = sess
	s.mu.Unlock()
}

// Clear removes the session for userID.
func (s *Store) Clear(userID int64) {
	s.mu.Lock()
	delete(s.sessions, userID)
	s.mu.Unlock()
}

// Stop shuts down the background cleanup goroutine.
func (s *Store) Stop() {
	close(s.stop)
}

func (s *Store) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.evictExpired()
		case <-s.stop:
			return
		}
	}
}

func (s *Store) evictExpired() {
	now := time.Now()
	s.mu.Lock()
	for userID, sess := range s.sessions {
		if now.Sub(sess.UpdatedAt) > sessionTTL {
			delete(s.sessions, userID)
		}
	}
	s.mu.Unlock()
}
