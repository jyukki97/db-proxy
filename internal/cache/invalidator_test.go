package cache

import (
	"testing"
)

func TestInvalidator_HandleMessage_FlushAll(t *testing.T) {
	c := New(Config{MaxEntries: 100, TTL: 60e9})
	c.Set(CacheKey("q1"), []byte("r1"), []string{"users"})
	c.Set(CacheKey("q2"), []byte("r2"), []string{"orders"})

	if c.Len() != 2 {
		t.Fatalf("cache len = %d, want 2", c.Len())
	}

	// Simulate receiving a flush-all message
	inv := &Invalidator{cache: c}
	inv.handleMessage("*")

	if c.Len() != 0 {
		t.Errorf("cache len after flush-all = %d, want 0", c.Len())
	}
}

func TestInvalidator_HandleMessage_Tables(t *testing.T) {
	c := New(Config{MaxEntries: 100, TTL: 60e9})
	c.Set(CacheKey("q1"), []byte("r1"), []string{"users"})
	c.Set(CacheKey("q2"), []byte("r2"), []string{"orders"})

	inv := &Invalidator{cache: c}
	inv.handleMessage("users")

	if c.Len() != 1 {
		t.Errorf("cache len after users invalidation = %d, want 1", c.Len())
	}

	// orders should still be cached
	if c.Get(CacheKey("q2")) == nil {
		t.Error("expected orders cache entry to remain")
	}
}

func TestInvalidator_HandleMessage_MultiTable(t *testing.T) {
	c := New(Config{MaxEntries: 100, TTL: 60e9})
	c.Set(CacheKey("q1"), []byte("r1"), []string{"users"})
	c.Set(CacheKey("q2"), []byte("r2"), []string{"orders"})
	c.Set(CacheKey("q3"), []byte("r3"), []string{"products"})

	inv := &Invalidator{cache: c}
	inv.handleMessage("users,orders")

	if c.Len() != 1 {
		t.Errorf("cache len after multi-table invalidation = %d, want 1", c.Len())
	}
}
