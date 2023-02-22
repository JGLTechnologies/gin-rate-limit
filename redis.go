package ratelimit

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type redisStoreType struct {
	rate       int64
	limit      uint
	client     *redis.Client
	ctx        context.Context
	panicOnErr bool
	skip       func(c *gin.Context) bool
}

func (s *redisStoreType) Limit(key string, c *gin.Context) Info {
	p := s.client.Pipeline()
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
	if s.skip != nil && s.skip(c) {
		return Info{
			Limit:         s.limit,
			RateLimited:   false,
			ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
			RemainingHits: s.limit - uint(hits),
		}
	}
	if hits >= int64(s.limit) {
		_, err = p.Exec(s.ctx)
		if err != nil {
			if s.panicOnErr {
				panic(err)
			} else {
				return Info{
					Limit:         s.limit,
					RateLimited:   false,
					ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
					RemainingHits: 0,
				}
			}
		}
		return Info{
			Limit:         s.limit,
			RateLimited:   true,
			ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
			RemainingHits: 0,
		}
	}
	ts = time.Now().Unix()
	hits++
	p.Incr(s.ctx, key+"hits")
	p.Set(s.ctx, key+"ts", time.Now().Unix(), time.Duration(0))
	p.Expire(s.ctx, key+"hits", time.Duration(int64(time.Second)*s.rate*2))
	p.Expire(s.ctx, key+"ts", time.Duration(int64(time.Second)*s.rate*2))
	_, err = p.Exec(s.ctx)
	if err != nil {
		if s.panicOnErr {
			panic(err)
		} else {
			return Info{
				Limit:         s.limit,
				RateLimited:   false,
				ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
				RemainingHits: s.limit - uint(hits),
			}
		}
	}
	return Info{
		Limit:         s.limit,
		RateLimited:   false,
		ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
		RemainingHits: s.limit - uint(hits),
	}
}

type RedisOptions struct {
	// the user can make Limit amount of requests every Rate
	Rate time.Duration
	// the amount of requests that can be made every Rate
	Limit       uint
	RedisClient *redis.Client
	// should gin-rate-limit panic when there is an error with redis
	PanicOnErr bool
	// a function that returns true if the request should not count toward the rate limit
	Skip func(*gin.Context) bool
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
