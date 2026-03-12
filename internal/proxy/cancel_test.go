package proxy

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jyukki97/pgmux/internal/pool"
	"github.com/jyukki97/pgmux/internal/protocol"
)

func TestCancelTarget_SetAndGet(t *testing.T) {
	ct := &cancelTarget{proxyPID: 1, proxySecret: 100}

	// Initially empty
	addr, pid, secret := ct.get()
	if addr != "" || pid != 0 || secret != 0 {
		t.Errorf("expected empty, got addr=%q pid=%d secret=%d", addr, pid, secret)
	}

	// Set from a mock pool.Conn
	mockConn := &pool.Conn{
		Conn:          nil, // not needed for this test
		BackendPID:    42,
		BackendSecret: 99,
	}
	ct.setFromConn("127.0.0.1:5432", mockConn)

	addr, pid, secret = ct.get()
	if addr != "127.0.0.1:5432" || pid != 42 || secret != 99 {
		t.Errorf("after set: addr=%q pid=%d secret=%d", addr, pid, secret)
	}

	// Clear
	ct.clear()
	addr, pid, secret = ct.get()
	if addr != "" || pid != 0 || secret != 0 {
		t.Errorf("after clear: addr=%q pid=%d secret=%d", addr, pid, secret)
	}
}

func TestServer_CancelKeyRegistration(t *testing.T) {
	s := &Server{}

	ct := s.newCancelTarget()
	if ct.proxyPID == 0 {
		t.Error("proxy PID should be non-zero")
	}

	// Should be findable in the map
	key := cancelKeyPair{pid: ct.proxyPID, secret: ct.proxySecret}
	val, ok := s.cancelMap.Load(key)
	if !ok {
		t.Fatal("cancel target not found in map")
	}
	if val.(*cancelTarget) != ct {
		t.Error("cancel target mismatch")
	}

	// Remove
	s.removeCancelTarget(ct)
	_, ok = s.cancelMap.Load(key)
	if ok {
		t.Error("cancel target should be removed")
	}
}

func TestServer_CancelRequestForwarding(t *testing.T) {
	// Start a mock "backend" that accepts cancel requests
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var receivedPID, receivedSecret uint32
	var received atomic.Bool
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Read cancel request: 16 bytes
		var buf [16]byte
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, err = conn.Read(buf[:])
		if err != nil {
			return
		}
		code := binary.BigEndian.Uint32(buf[4:8])
		if code == protocol.CancelRequestCode {
			receivedPID = binary.BigEndian.Uint32(buf[8:12])
			receivedSecret = binary.BigEndian.Uint32(buf[12:16])
			received.Store(true)
		}
	}()

	// Forward a cancel request
	err = forwardCancel(ln.Addr().String(), 123, 456)
	if err != nil {
		t.Fatalf("forwardCancel: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if !received.Load() {
		t.Fatal("backend did not receive cancel request")
	}
	if receivedPID != 123 || receivedSecret != 456 {
		t.Errorf("got pid=%d secret=%d, want 123/456", receivedPID, receivedSecret)
	}
}

func TestServer_HandleCancelRequest_Integration(t *testing.T) {
	// Start a mock backend
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var received atomic.Bool
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var buf [16]byte
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		conn.Read(buf[:])
		received.Store(true)
	}()

	s := &Server{}

	// Register a cancel target with an active backend query
	ct := s.newCancelTarget()
	ct.mu.Lock()
	ct.backendAddr = ln.Addr().String()
	ct.backendPID = 42
	ct.backendSecret = 99
	ct.mu.Unlock()

	// Build cancel request payload (as it would come from ReadStartupMessage)
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], protocol.CancelRequestCode)
	binary.BigEndian.PutUint32(payload[4:8], ct.proxyPID)
	binary.BigEndian.PutUint32(payload[8:12], ct.proxySecret)

	s.handleCancelRequest(payload)

	time.Sleep(200 * time.Millisecond)

	if !received.Load() {
		t.Error("cancel request was not forwarded to backend")
	}

	s.removeCancelTarget(ct)
}
