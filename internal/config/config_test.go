package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	content := `
proxy:
  listen: "0.0.0.0:5432"
writer:
  host: "primary.db.internal"
  port: 5432
readers:
  - host: "replica-1.db.internal"
    port: 5432
  - host: "replica-2.db.internal"
    port: 5432
pool:
  min_connections: 5
  max_connections: 50
  idle_timeout: 10m
  max_lifetime: 1h
  connection_timeout: 5s
routing:
  read_after_write_delay: 500ms
cache:
  enabled: true
  cache_ttl: 10s
  max_cache_entries: 10000
  max_result_size: "1MB"
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Proxy
	if cfg.Proxy.Listen != "0.0.0.0:5432" {
		t.Errorf("Proxy.Listen = %q, want %q", cfg.Proxy.Listen, "0.0.0.0:5432")
	}

	// Writer
	if cfg.Writer.Host != "primary.db.internal" {
		t.Errorf("Writer.Host = %q, want %q", cfg.Writer.Host, "primary.db.internal")
	}
	if cfg.Writer.Port != 5432 {
		t.Errorf("Writer.Port = %d, want %d", cfg.Writer.Port, 5432)
	}

	// Readers
	if len(cfg.Readers) != 2 {
		t.Fatalf("len(Readers) = %d, want 2", len(cfg.Readers))
	}
	if cfg.Readers[0].Host != "replica-1.db.internal" {
		t.Errorf("Readers[0].Host = %q, want %q", cfg.Readers[0].Host, "replica-1.db.internal")
	}

	// Pool
	if cfg.Pool.MinConnections != 5 {
		t.Errorf("Pool.MinConnections = %d, want 5", cfg.Pool.MinConnections)
	}
	if cfg.Pool.MaxConnections != 50 {
		t.Errorf("Pool.MaxConnections = %d, want 50", cfg.Pool.MaxConnections)
	}
	if cfg.Pool.IdleTimeout != 10*time.Minute {
		t.Errorf("Pool.IdleTimeout = %v, want 10m", cfg.Pool.IdleTimeout)
	}
	if cfg.Pool.MaxLifetime != time.Hour {
		t.Errorf("Pool.MaxLifetime = %v, want 1h", cfg.Pool.MaxLifetime)
	}
	if cfg.Pool.ConnectionTimeout != 5*time.Second {
		t.Errorf("Pool.ConnectionTimeout = %v, want 5s", cfg.Pool.ConnectionTimeout)
	}

	// Routing
	if cfg.Routing.ReadAfterWriteDelay != 500*time.Millisecond {
		t.Errorf("Routing.ReadAfterWriteDelay = %v, want 500ms", cfg.Routing.ReadAfterWriteDelay)
	}

	// Cache
	if !cfg.Cache.Enabled {
		t.Error("Cache.Enabled = false, want true")
	}
	if cfg.Cache.CacheTTL != 10*time.Second {
		t.Errorf("Cache.CacheTTL = %v, want 10s", cfg.Cache.CacheTTL)
	}
	if cfg.Cache.MaxCacheEntries != 10000 {
		t.Errorf("Cache.MaxCacheEntries = %d, want 10000", cfg.Cache.MaxCacheEntries)
	}
	if cfg.Cache.MaxResultSize != "1MB" {
		t.Errorf("Cache.MaxResultSize = %q, want %q", cfg.Cache.MaxResultSize, "1MB")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("Load() expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("invalid: yaml: [broken"); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}
