package router

import (
	"strings"
	"sync"
	"time"
)

type Route int

const (
	RouteWriter Route = iota
	RouteReader
)

type Session struct {
	mu                  sync.Mutex
	inTransaction       bool
	lastWriteTime       time.Time
	readAfterWriteDelay time.Duration
}

func NewSession(readAfterWriteDelay time.Duration) *Session {
	return &Session{
		readAfterWriteDelay: readAfterWriteDelay,
	}
}

// Route determines where to send the query based on session state and query type.
func (s *Session) Route(query string) Route {
	s.mu.Lock()
	defer s.mu.Unlock()

	upper := strings.ToUpper(strings.TrimSpace(query))

	// Track transaction state
	if strings.HasPrefix(upper, "BEGIN") || strings.HasPrefix(upper, "START TRANSACTION") {
		s.inTransaction = true
		return RouteWriter
	}
	if strings.HasPrefix(upper, "COMMIT") || strings.HasPrefix(upper, "ROLLBACK") {
		s.inTransaction = false
		return RouteWriter
	}

	// All queries in a transaction go to writer
	if s.inTransaction {
		return RouteWriter
	}

	qtype := Classify(query)

	// Write query
	if qtype == QueryWrite {
		s.lastWriteTime = time.Now()
		return RouteWriter
	}

	// Read-after-write: send to writer within delay window
	if s.readAfterWriteDelay > 0 && !s.lastWriteTime.IsZero() &&
		time.Since(s.lastWriteTime) < s.readAfterWriteDelay {
		return RouteWriter
	}

	return RouteReader
}

// InTransaction returns whether the session is currently in a transaction.
func (s *Session) InTransaction() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inTransaction
}
