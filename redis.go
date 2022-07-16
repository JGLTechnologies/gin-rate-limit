package GinRateLimit

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisStoreType struct {
	rate       int64
	limit      int
	client     redis.UniversalClient
	ctx        context.Context
	panicOnErr bool
}

func (s *RedisStoreType) Limit(key string) (bool, time.Duration) {
	p := s.client.Pipeline()
	defer p.Close()
	cmds, _ := s.client.Pipelined(s.ctx, func(pipeliner redis.Pipeliner) error {
		pipeliner.Get(s.ctx, key+"ts")
		pipeliner.Get(s.ctx, key+"hits")
		return nil
	})
	ts, err := cmds[0].(*redis.StringCmd).Int64()
	if err != nil {
		ts = time.Now().Unix()
	}
	hits, err := cmds[1].(*redis.StringCmd).Int64()
	if err != nil {
		hits = 0
	}
	if ts+s.rate <= time.Now().Unix() {
		p.Set(s.ctx, key+"hits", 0, time.Duration(0))
	}
	remaining := time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())
	if hits >= int64(s.limit) {
		_, err = p.Exec(s.ctx)
		if err != nil {
			if s.panicOnErr {
				panic(err)
			} else {
				return false, time.Duration(0)
			}
		}
		return true, remaining
	}
	p.Incr(s.ctx, key+"hits")
	p.Set(s.ctx, key+"ts", time.Now().Unix(), time.Duration(0))
	p.Expire(s.ctx, key+"hits", time.Duration(int64(time.Second)*s.rate*2))
	p.Expire(s.ctx, key+"ts", time.Duration(int64(time.Second)*s.rate*2))
	_, err = p.Exec(s.ctx)
	if err != nil {
		if s.panicOnErr {
			panic(err)
		} else {
			return false, time.Duration(0)
		}
	}
	return false, time.Duration(0)
}

func RedisStore(rate time.Duration, limit int, redisClient redis.UniversalClient, panicOnErr bool) *RedisStoreType {
	return &RedisStoreType{client: redisClient, rate: int64(rate.Seconds()), limit: limit, ctx: context.TODO(), panicOnErr: panicOnErr}
}
