package ratelimit

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type user struct {
	ts     int64
	tokens uint
}

func clearInBackground(data *sync.Map, resetTime int64) {
	for {
		data.Range(func(k, v interface{}) bool {
			if v.(user).ts+resetTime <= time.Now().Unix() {
				data.Delete(k)
			}
			return true
		})
		time.Sleep(time.Minute)
	}
}

type inMemoryStoreType struct {
	rate      int64
	limit     uint
	resetTime int64
	data      *sync.Map
	skip      func(ctx *gin.Context) bool
}

func (s *inMemoryStoreType) Limit(key string, c *gin.Context) Info {
	var u user
	m, ok := s.data.Load(key)
	if !ok {
		u = user{time.Now().Unix(), s.limit}
	} else {
		u = m.(user)
	}
	if u.ts+s.resetTime <= time.Now().Unix() {
		u.tokens = s.limit
	}
	if s.skip != nil && s.skip(c) {
		return Info{
			Limit:         s.limit,
			RateLimited:   false,
			ResetTime:     time.Now().Add(time.Duration((s.resetTime - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())),
			RemainingHits: u.tokens,
		}
	}
	if u.tokens <= 0 {
		return Info{
			Limit:         s.limit,
			RateLimited:   true,
			ResetTime:     time.Now().Add(time.Duration((s.resetTime - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())),
			RemainingHits: 0,
		}
	}
	u.tokens--
	u.ts = time.Now().Unix()
	s.data.Store(key, u)
	return Info{
		Limit:         s.limit,
		RateLimited:   false,
		ResetTime:     time.Now().Add(time.Duration((s.resetTime - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())),
		RemainingHits: u.tokens,
	}
}

type InMemoryOptions struct {
	// the user can make Limit amount of requests every Rate
	Rate time.Duration
	// the amount of requests that can be made every Rate
	Limit uint
	// the user will be unblocked after the ResetTime
	ResetTime time.Duration
	// a function that returns true if the request should not count toward the rate limit
	Skip func(*gin.Context) bool
}

func InMemoryStore(options *InMemoryOptions) Store {
	data := &sync.Map{}
	store := inMemoryStoreType{int64(options.Rate.Seconds()), options.Limit, int64(options.ResetTime.Seconds()), data, options.Skip}
	go clearInBackground(data, store.resetTime)
	return &store
}
