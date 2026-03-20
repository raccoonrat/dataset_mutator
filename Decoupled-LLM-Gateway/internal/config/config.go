package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config is intentionally small: env-only, no config file dependency for MVP.
type Config struct {
	ListenAddr      string
	UpstreamURL     *url.URL
	UpstreamTimeout time.Duration
	MaxBodyBytes    int64
	AsyncLogging    bool
	PolicySeedPath  string // optional JSON array of PolicyRule; empty = none
}

func FromEnv() (*Config, error) {
	listen := getenv("GATEWAY_LISTEN", ":8080")
	up := getenv("GATEWAY_UPSTREAM", "http://127.0.0.1:9090")
	u, err := url.Parse(up)
	if err != nil {
		return nil, fmt.Errorf("GATEWAY_UPSTREAM: %w", err)
	}
	ms, _ := strconv.Atoi(getenv("GATEWAY_UPSTREAM_TIMEOUT_MS", "30000"))
	if ms <= 0 {
		ms = 30000
	}
	maxBody, _ := strconv.ParseInt(getenv("GATEWAY_MAX_BODY_BYTES", "4194304"), 10, 64)
	if maxBody <= 0 {
		maxBody = 4 << 20
	}
	async := getenv("GATEWAY_ASYNC_LOG", "1") != "0"
	seed := os.Getenv("GATEWAY_POLICY_SEED_FILE")

	return &Config{
		ListenAddr:      listen,
		UpstreamURL:     u,
		UpstreamTimeout: time.Duration(ms) * time.Millisecond,
		MaxBodyBytes:    maxBody,
		AsyncLogging:    async,
		PolicySeedPath:  seed,
	}, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
