package policy

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
)

// StartRedisRefresher periodically loads PolicyRule JSON objects from a Redis hash (field = rule_id, value = JSON) into store.
func StartRedisRefresher(ctx context.Context, rdb *redis.Client, store Store, hashKey string, every time.Duration) {
	if rdb == nil || store == nil || hashKey == "" || every <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		refreshFromRedis(ctx, rdb, store, hashKey)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refreshFromRedis(ctx, rdb, store, hashKey)
			}
		}
	}()
}

func refreshFromRedis(ctx context.Context, rdb *redis.Client, store Store, hashKey string) {
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	m, err := rdb.HGetAll(c, hashKey).Result()
	if err != nil {
		log.Printf("policy redis HGETALL %q: %v", hashKey, err)
		return
	}
	for _, v := range m {
		var rule contracts.PolicyRule
		if err := json.Unmarshal([]byte(v), &rule); err != nil {
			log.Printf("policy redis skip invalid JSON: %v", err)
			continue
		}
		if rule.RuleID == "" || rule.TriggerPattern == "" {
			continue
		}
		if err := store.Upsert(rule); err != nil {
			log.Printf("policy redis upsert %q: %v", rule.RuleID, err)
		}
	}
}
