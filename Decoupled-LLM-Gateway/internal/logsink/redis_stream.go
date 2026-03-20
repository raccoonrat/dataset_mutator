package logsink

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/raccoonrat/decoupled-llm-gateway/internal/contracts"
)

// RedisStreamSink appends JSON events to a Redis Stream (field "payload").
type RedisStreamSink struct {
	rdb    *redis.Client
	stream string
}

func NewRedisStreamSink(rdb *redis.Client, stream string) *RedisStreamSink {
	return &RedisStreamSink{rdb: rdb, stream: stream}
}

func (s *RedisStreamSink) Emit(e contracts.GatewayLogEvent) {
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: s.stream,
		Values: map[string]interface{}{"payload": string(b)},
	}).Err()
	if err != nil {
		log.Printf("logsink redis XADD: %v", err)
	}
}
