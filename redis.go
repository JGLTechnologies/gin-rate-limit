package ratelimit

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"time"
)

type redisStoreType struct {
	rate       int64
	limit      uint
	client     redis.UniversalClient
	ctx        context.Context
	panicOnErr bool
	skip       func(c *gin.Context) bool
}

func (s *redisStoreType) Limit(key string) (bool, time.Duration) {
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
		hits = 0
		p.Set(s.ctx, key+"hits", hits, time.Duration(0))
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

func (s *redisStoreType) Skip(c *gin.Context) bool {
	if s.skip != nil {
		return s.skip(c)
	} else {
		return false
	}
}

type RedisOptions struct {
	// the user can make Limit amount of requests every Rate
	Rate time.Duration
	// the amount of requests that can be made every Rate
	Limit uint
	// takes in a *gin.Context and should return whether the rate limiting should be skipped for this request
	Skip        func(c *gin.Context) bool
	RedisClient redis.UniversalClient
	// should gin-rate-limit panic when there is an error with redis
	PanicOnErr bool
}

func RedisStore(options *RedisOptions) Store {
	return &redisStoreType{
		client:     options.RedisClient,
		rate:       int64(options.Rate.Seconds()),
		limit:      options.Limit,
		ctx:        context.TODO(),
		panicOnErr: options.PanicOnErr,
		skip:       options.Skip,
	}
}
