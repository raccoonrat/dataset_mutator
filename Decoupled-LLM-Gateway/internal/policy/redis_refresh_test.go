package policy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
)

func TestStartRedisRefresher_LoadsRule(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	store := NewMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	StartRedisRefresher(ctx, rdb, store, "rules", 40*time.Millisecond)

	rule := contracts.PolicyRule{
		RuleID:           "r-redis",
		Action:           ActionDegradeToTemplate,
		TriggerPattern:   `evil-token`,
		TemplateResponse: "blocked-by-redis",
	}
	b, err := json.Marshal(rule)
	if err != nil {
		t.Fatal(err)
	}
	if err := rdb.HSet(context.Background(), "rules", "r-redis", string(b)).Err(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var m *contracts.PolicyRule
	for time.Now().Before(deadline) {
		m = store.MatchPrompt("prefix evil-token suffix")
		if m != nil {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}
	if m == nil || m.TemplateResponse != "blocked-by-redis" {
		t.Fatalf("got %+v", m)
	}
}
