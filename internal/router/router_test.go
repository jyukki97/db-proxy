package router

import (
	"testing"
	"time"
)

func TestSession_BasicRouting(t *testing.T) {
	s := NewSession(0)

	if got := s.Route("SELECT * FROM users"); got != RouteReader {
		t.Errorf("SELECT → %d, want RouteReader", got)
	}
	if got := s.Route("INSERT INTO users VALUES (1)"); got != RouteWriter {
		t.Errorf("INSERT → %d, want RouteWriter", got)
	}
}

func TestSession_Transaction(t *testing.T) {
	s := NewSession(0)

	// BEGIN → all queries go to writer
	if got := s.Route("BEGIN"); got != RouteWriter {
		t.Errorf("BEGIN → %d, want RouteWriter", got)
	}
	if got := s.Route("SELECT * FROM users"); got != RouteWriter {
		t.Errorf("SELECT in tx → %d, want RouteWriter", got)
	}
	if got := s.Route("INSERT INTO users VALUES (1)"); got != RouteWriter {
		t.Errorf("INSERT in tx → %d, want RouteWriter", got)
	}

	// COMMIT → back to normal routing
	if got := s.Route("COMMIT"); got != RouteWriter {
		t.Errorf("COMMIT → %d, want RouteWriter", got)
	}
	if got := s.Route("SELECT * FROM users"); got != RouteReader {
		t.Errorf("SELECT after commit → %d, want RouteReader", got)
	}
}

func TestSession_ReadAfterWriteDelay(t *testing.T) {
	s := NewSession(100 * time.Millisecond)

	// Write
	s.Route("INSERT INTO users VALUES (1)")

	// Read immediately after write → writer
	if got := s.Route("SELECT * FROM users"); got != RouteWriter {
		t.Errorf("SELECT after write → %d, want RouteWriter", got)
	}

	// Wait for delay to expire
	time.Sleep(150 * time.Millisecond)

	// Read after delay → reader
	if got := s.Route("SELECT * FROM users"); got != RouteReader {
		t.Errorf("SELECT after delay → %d, want RouteReader", got)
	}
}

func TestSession_Rollback(t *testing.T) {
	s := NewSession(0)

	s.Route("BEGIN")
	if !s.InTransaction() {
		t.Error("expected InTransaction=true after BEGIN")
	}

	s.Route("ROLLBACK")
	if s.InTransaction() {
		t.Error("expected InTransaction=false after ROLLBACK")
	}

	if got := s.Route("SELECT 1"); got != RouteReader {
		t.Errorf("SELECT after rollback → %d, want RouteReader", got)
	}
}
