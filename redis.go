package GinRateLimit

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisStoreType struct {
	rate   int64
	limit  int
	client *redis.Client
	ctx    context.Context
}

func (s *RedisStoreType) Limit(key string) (bool, time.Duration) {
	p := s.client.Pipeline()
	defer func(s *RedisStoreType, p redis.Pipeliner) {
		p.Exec(s.ctx)
		p.Close()
	}(s, p)
	ts, err := s.client.Get(s.ctx, key+"ts").Int64()
	if err != nil {
		ts = time.Now().Unix()
	}
	hits, err := s.client.Get(s.ctx, key+"hits").Int64()
	if err != nil {
		hits = 0
	}
	p.Expire(s.ctx, key+"hits", time.Duration(int64(time.Second)*s.rate*2))
	p.Expire(s.ctx, key+"ts", time.Duration(int64(time.Second)*s.rate*2))
	if ts+s.rate <= time.Now().Unix() {
		p.Set(s.ctx, key+"hits", 0, time.Duration(0))
	}
	remaining := time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())
	if hits >= int64(s.limit) {
		return true, remaining
	}
	fmt.Println(hits)
	p.Incr(s.ctx, key+"hits")
	p.Set(s.ctx, key+"ts", time.Now().Unix(), time.Duration(0))
	return false, time.Duration(0)
}

func RedisStore(rate time.Duration, limit int, redisClient *redis.Client) *RedisStoreType {
	return &RedisStoreType{client: redisClient, rate: int64(rate.Seconds()), limit: limit, ctx: context.TODO()}
}
