package ratelimit

import (
	"context"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gin-gonic/gin"
)

type memcachedStoreType struct {
	rate       int64
	limit      uint
	client     memcache.Client
	ctx        context.Context
	panicOnErr bool
	skip       func(c *gin.Context) bool
}

func (s *memcachedStoreType) Limit(key string, c *gin.Context) Info {
	client := s.client
	var ts, hits int64

	tsVal, err := client.Get(key + "ts")
	if err != nil {
		ts = time.Now().Unix()
	} else {
		ts = int64(binary.LittleEndian.Uint64(tsVal.Value))
	}

	hitsVal, err := client.Get(key + "hits")
	if err != nil {
		hits = 0
	} else {
		hits = int64(binary.LittleEndian.Uint64(hitsVal.Value))
	}

	if ts+s.rate <= time.Now().Unix() {
		hits = 0
		hitsByte := []byte(strconv.FormatInt(hits, 10))
		client.Set(&memcache.Item{Key: key + "hits", Value: hitsByte, Expiration: int32(time.Duration(0))})
	}
	if s.skip != nil && s.skip(c) {
		return Info{
			RateLimited:   false,
			ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
			RemainingHits: s.limit - uint(hits),
		}
	}
	if hits >= int64(s.limit) {
		return Info{
			RateLimited:   true,
			ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
			RemainingHits: 0,
		}
	}
	ts = time.Now().Unix()
	hits++
	client.Increment(key+"hits", uint64(hits))
	client.Set(
		&memcache.Item{
			Key:        key + "ts",
			Value:      []byte(strconv.FormatInt(time.Now().Unix(), 10)),
			Expiration: int32(time.Duration(0)),
		},
	)
	client.Set(
		&memcache.Item{
			Key:        key + "hits",
			Value:      []byte(strconv.FormatInt(time.Now().Unix(), 10)),
			Expiration: int32(time.Duration(int64(time.Second) * s.rate * 2)),
		},
	)
	client.Set(
		&memcache.Item{
			Key:        key + "ts",
			Value:      []byte(strconv.FormatInt(time.Now().Unix(), 10)),
			Expiration: int32(time.Duration(int64(time.Second) * s.rate * 2)),
		},
	)

	return Info{
		RateLimited:   false,
		ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - ts)) * time.Second.Nanoseconds())),
		RemainingHits: s.limit - uint(hits),
	}
}

type MemcachedOptions struct {
	// the user can make Limit amount of requests every Rate
	Rate time.Duration
	// the amount of requests that can be made every Rate
	Limit           uint
	MemcachedClient memcache.Client
	// should gin-rate-limit panic when there is an error with redis
	PanicOnErr bool
	// a function that returns true if the request should not count toward the rate limit
	Skip func(*gin.Context) bool
}

func MemcachedStore(options *MemcachedOptions) Store {
	return &memcachedStoreType{
		client:     options.MemcachedClient,
		rate:       int64(options.Rate.Seconds()),
		limit:      options.Limit,
		ctx:        context.TODO(),
		panicOnErr: options.PanicOnErr,
		skip:       options.Skip,
	}
}
