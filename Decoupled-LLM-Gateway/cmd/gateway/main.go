package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/config"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/gateway"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/logsink"
	"github.com/raccoonrat/decoupled-llm-gateway/internal/policy"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal(err)
	}

	store := policy.NewMemoryStore()
	if err := store.LoadFromFile(cfg.PolicySeedPath); err != nil {
		log.Fatalf("policy seed: %v", err)
	}

	ctx := context.Background()
	var sink logsink.Sink
	if cfg.RedisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Fatalf("redis ping: %v", err)
		}
		rs := logsink.NewRedisStreamSink(rdb, cfg.RedisStream)
		if cfg.AsyncLogging {
			sink = logsink.Multi{rs, logsink.Stdout()}
		} else {
			sink = rs
		}
		policy.StartRedisRefresher(ctx, rdb, store, cfg.PolicyRedisHashKey, time.Duration(cfg.PolicyRefreshMS)*time.Millisecond)
		log.Printf("redis stream=%q policy_hash=%q refresh_ms=%d", cfg.RedisStream, cfg.PolicyRedisHashKey, cfg.PolicyRefreshMS)
	} else {
		if cfg.AsyncLogging {
			sink = logsink.Stdout()
		} else {
			sink = logsink.Discard{}
		}
	}

	h, err := gateway.New(cfg, store, sink)
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}

	log.Printf("gateway listening %s → upstream %s (obfuscate profile=%q rules_file=%q experiment_mode=%q)",
		cfg.ListenAddr, h.UpstreamBase, cfg.ObfuscateProfile, cfg.ObfuscateRulesFile, cfg.ExperimentMode)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, h))
}
