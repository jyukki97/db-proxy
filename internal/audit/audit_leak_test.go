package audit

import (
	"fmt"
	"testing"
	"time"
)

// TestAuditLogger_WebhookDedupCleanup verifies that the lastWebhook map is
// periodically cleaned up, preventing unbounded memory growth.
func TestAuditLogger_WebhookDedupCleanup(t *testing.T) {
	logger := New(Config{
		Enabled:            true,
		SlowQueryThreshold: 1 * time.Millisecond,
		Webhook: WebhookConfig{
			Enabled:       true,
			URL:           "http://localhost:9999",
			DedupInterval: 500 * time.Millisecond,
		},
	})
	defer logger.Close()

	// Simulate unique slow queries to populate lastWebhook map
	for i := 0; i < 50; i++ {
		logger.Log(Event{
			Timestamp:  time.Now(),
			EventType:  "query",
			Query:      fmt.Sprintf("SELECT * FROM table_%d", i),
			DurationMS: 50.0,
		})
	}

	// Wait for events to be processed and sendWebhook goroutines to set map entries
	time.Sleep(300 * time.Millisecond)

	logger.lastWebhookMu.Lock()
	afterInsert := len(logger.lastWebhook)
	logger.lastWebhookMu.Unlock()

	if afterInsert == 0 {
		t.Fatal("expected lastWebhook map to have entries after logging")
	}
	t.Logf("map size after inserts: %d", afterInsert)

	// Wait for entries to expire (500ms dedup interval) + cleanup tick
	time.Sleep(800 * time.Millisecond)

	logger.lastWebhookMu.Lock()
	afterCleanup := len(logger.lastWebhook)
	logger.lastWebhookMu.Unlock()

	t.Logf("map size after cleanup: %d", afterCleanup)

	if afterCleanup >= afterInsert {
		t.Errorf("cleanup did not reduce map: before=%d, after=%d", afterInsert, afterCleanup)
	}
	if afterCleanup != 0 {
		t.Errorf("expected map to be empty after cleanup, got %d entries", afterCleanup)
	}
}

// TestAuditLogger_WebhookDedupBounded verifies that the map stays bounded
// even under continuous unique query load.
func TestAuditLogger_WebhookDedupBounded(t *testing.T) {
	logger := New(Config{
		Enabled:            true,
		SlowQueryThreshold: 1 * time.Millisecond,
		Webhook: WebhookConfig{
			Enabled:       true,
			URL:           "http://localhost:9999",
			DedupInterval: 100 * time.Millisecond,
		},
	})
	defer logger.Close()

	// Send queries in batches with pauses to allow cleanup
	for batch := 0; batch < 5; batch++ {
		for i := 0; i < 50; i++ {
			logger.Log(Event{
				Timestamp:  time.Now(),
				EventType:  "query",
				Query:      fmt.Sprintf("SELECT * FROM batch_%d_table_%d", batch, i),
				DurationMS: 50.0,
			})
		}
		time.Sleep(200 * time.Millisecond) // let cleanup run between batches
	}

	// Final cleanup pass
	time.Sleep(300 * time.Millisecond)

	logger.lastWebhookMu.Lock()
	finalSize := len(logger.lastWebhook)
	logger.lastWebhookMu.Unlock()

	t.Logf("final map size: %d (sent 250 unique queries across 5 batches)", finalSize)

	// Without cleanup this would be ~250; with cleanup it should be much smaller
	if finalSize > 100 {
		t.Errorf("map size %d exceeds expected bound, cleanup may not be working", finalSize)
	}
}
