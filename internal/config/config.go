package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxy   ProxyConfig   `yaml:"proxy"`
	Writer  DBConfig      `yaml:"writer"`
	Readers []DBConfig    `yaml:"readers"`
	Pool    PoolConfig    `yaml:"pool"`
	Routing RoutingConfig `yaml:"routing"`
	Cache   CacheConfig   `yaml:"cache"`
}

type ProxyConfig struct {
	Listen string `yaml:"listen"`
}

type DBConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type PoolConfig struct {
	MinConnections    int           `yaml:"min_connections"`
	MaxConnections    int           `yaml:"max_connections"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	MaxLifetime       time.Duration `yaml:"max_lifetime"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
}

type RoutingConfig struct {
	ReadAfterWriteDelay time.Duration `yaml:"read_after_write_delay"`
}

type CacheConfig struct {
	Enabled        bool          `yaml:"enabled"`
	CacheTTL       time.Duration `yaml:"cache_ttl"`
	MaxCacheEntries int          `yaml:"max_cache_entries"`
	MaxResultSize   string       `yaml:"max_result_size"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
