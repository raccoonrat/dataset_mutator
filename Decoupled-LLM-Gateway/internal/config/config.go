package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is intentionally small: env-only, no config file dependency for MVP.
type Config struct {
	ListenAddr         string
	UpstreamURL        *url.URL
	UpstreamTimeout    time.Duration
	MaxBodyBytes       int64
	AsyncLogging       bool
	PolicySeedPath     string // optional JSON array of PolicyRule; empty = none
	ObfuscateProfile   string // strict|full|minimal|balanced; empty = strict
	ObfuscateRulesFile string // optional JSON array of extra Rule {name,pattern,repl}

	// Milestone 3: Redis stream log + policy hash refresh (empty RedisAddr disables).
	RedisAddr          string
	RedisStream        string
	PolicyRedisHashKey string
	PolicyRefreshMS    int
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
	obProf := getenv("GATEWAY_OBFUSCATE_PROFILE", "strict")
	obRules := os.Getenv("GATEWAY_OBFUSCATE_RULES_FILE")
	redisAddr := strings.TrimSpace(os.Getenv("GATEWAY_REDIS_ADDR"))
	redisStream := getenv("GATEWAY_REDIS_STREAM", "decoupled:gateway:events")
	policyHash := getenv("GATEWAY_POLICY_REDIS_HASH", "decoupled:policy:rules")
	refreshMS, _ := strconv.Atoi(getenv("GATEWAY_POLICY_REFRESH_MS", "2000"))
	if refreshMS <= 0 {
		refreshMS = 2000
	}

	return &Config{
		ListenAddr:         listen,
		UpstreamURL:        u,
		UpstreamTimeout:    time.Duration(ms) * time.Millisecond,
		MaxBodyBytes:       maxBody,
		AsyncLogging:       async,
		PolicySeedPath:     seed,
		ObfuscateProfile:   obProf,
		ObfuscateRulesFile: obRules,
		RedisAddr:          redisAddr,
		RedisStream:        redisStream,
		PolicyRedisHashKey: policyHash,
		PolicyRefreshMS:    refreshMS,
	}, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
