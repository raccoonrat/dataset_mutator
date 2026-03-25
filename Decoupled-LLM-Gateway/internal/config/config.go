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
	ListenAddr  string
	UpstreamURL *url.URL
	// UpstreamAPIKey sent as Authorization: Bearer (OpenAI-compatible: DeepSeek, OpenAI, etc.).
	// Resolved from GATEWAY_UPSTREAM_API_KEY, else DEEPSEEK_API_KEY, else OPENAI_API_KEY.
	UpstreamAPIKey     string
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

	// Paper evaluation: default gateway behavior variant (per-request header can override).
	// Values: default | no_obfuscate | no_decoy | intent_only | structured_wrap (see docs/DEFENSE_INVENTORY_AND_LITERATURE.md).
	ExperimentMode string

	// LogAfterResponse: emit audit/log events after the response body is written (avoids Redis blocking the client).
	LogAfterResponse bool
	// LogMaxPromptRunes / LogMaxLLMRunes: 0 = no truncation (UTF-8 rune counts).
	LogMaxPromptRunes int
	LogMaxLLMRunes    int

	// Optional post-upstream HTTP guard (judge_service refusal_binary). See RUN_GUIDE.md.
	OutputGuardURL             string
	OutputGuardBearer          string
	OutputGuardTimeout         time.Duration
	OutputGuardFailOpen        bool
	OutputGuardTemplate        string
	OutputGuardRequireHeader   bool // if true (default when URL set), client must send X-Gateway-Output-Guard: 1
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
	expMode := strings.TrimSpace(strings.ToLower(os.Getenv("GATEWAY_EXPERIMENT_MODE")))
	if expMode == "" {
		expMode = "default"
	}
	upKey := strings.TrimSpace(os.Getenv("GATEWAY_UPSTREAM_API_KEY"))
	if upKey == "" {
		upKey = strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	}
	if upKey == "" {
		upKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	}

	logAfterResp := getenv("GATEWAY_LOG_AFTER_RESPONSE", "1") != "0"
	maxPromptLog, _ := strconv.Atoi(getenv("GATEWAY_LOG_MAX_PROMPT_RUNES", "0"))
	maxLLMLog, _ := strconv.Atoi(getenv("GATEWAY_LOG_MAX_LLM_RUNES", "0"))
	if maxPromptLog < 0 {
		maxPromptLog = 0
	}
	if maxLLMLog < 0 {
		maxLLMLog = 0
	}

	guardURL := strings.TrimSpace(os.Getenv("GATEWAY_OUTPUT_GUARD_URL"))
	guardBearer := strings.TrimSpace(os.Getenv("GATEWAY_OUTPUT_GUARD_BEARER"))
	guardMS, _ := strconv.Atoi(getenv("GATEWAY_OUTPUT_GUARD_TIMEOUT_MS", "10000"))
	if guardMS <= 0 {
		guardMS = 10000
	}
	guardFailOpen := getenv("GATEWAY_OUTPUT_GUARD_FAIL_OPEN", "1") != "0"
	guardTpl := getenv(
		"GATEWAY_OUTPUT_GUARD_TEMPLATE",
		"I'm sorry, I can't help with that.",
	)
	// When URL is set, default to requiring per-request opt-in header (safe for mixed traffic).
	guardRequireHdr := true
	if guardURL != "" {
		if v := strings.TrimSpace(os.Getenv("GATEWAY_OUTPUT_GUARD_REQUIRE_HEADER")); v != "" {
			guardRequireHdr = v != "0" && strings.ToLower(v) != "false"
		}
	} else {
		guardRequireHdr = false
	}

	return &Config{
		ListenAddr:         listen,
		UpstreamURL:        u,
		UpstreamAPIKey:     upKey,
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
		ExperimentMode:     expMode,
		LogAfterResponse:   logAfterResp,
		LogMaxPromptRunes:  maxPromptLog,
		LogMaxLLMRunes:     maxLLMLog,

		OutputGuardURL:           guardURL,
		OutputGuardBearer:        guardBearer,
		OutputGuardTimeout:       time.Duration(guardMS) * time.Millisecond,
		OutputGuardFailOpen:      guardFailOpen,
		OutputGuardTemplate:      guardTpl,
		OutputGuardRequireHeader: guardRequireHdr,
	}, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
