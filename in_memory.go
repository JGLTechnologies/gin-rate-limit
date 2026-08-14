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

func clearInBackground(data *sync.Map, rate int64) {
	for {
		now := time.Now().Unix()
		data.Range(func(k, v interface{}) bool {
			u := v.(user)
			if u.ts+rate <= now {
				data.CompareAndDelete(k, u)
			}

			return true
		})
		time.Sleep(time.Minute)
	}
}

type inMemoryStoreType struct {
	rate  int64
	limit uint
	data  *sync.Map
	skip  func(ctx *gin.Context) bool
}

func (s *inMemoryStoreType) Limit(key string, c *gin.Context) Info {
	shouldSkip := s.skip != nil && s.skip(c)

	for {
		now := time.Now().Unix()

		m, ok := s.data.Load(key)
		if !ok {
			u := user{
				ts:     now,
				tokens: s.limit,
			}

			if shouldSkip {
				return Info{
					Limit:         s.limit,
					RateLimited:   false,
					ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())),
					RemainingHits: u.tokens,
				}
			}

			actual, _ := s.data.LoadOrStore(key, u)
			m = actual
		}

		old := m.(user)
		u := old

		if u.ts+s.rate <= now {
			u.tokens = s.limit
		}

		if shouldSkip {
			return Info{
				Limit:         s.limit,
				RateLimited:   false,
				ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())),
				RemainingHits: u.tokens,
			}
		}

		if u.tokens <= 0 {
			return Info{
				Limit:         s.limit,
				RateLimited:   true,
				ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())),
				RemainingHits: 0,
			}
		}

		u.tokens--
		u.ts = now

		if !s.data.CompareAndSwap(key, old, u) {
			continue
		}

		return Info{
			Limit:         s.limit,
			RateLimited:   false,
			ResetTime:     time.Now().Add(time.Duration((s.rate - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())),
			RemainingHits: u.tokens,
		}
	}
}

type InMemoryOptions struct {
	// the user can make Limit number of requests every Rate
	Rate time.Duration
	// the number of requests that can be made every Rate
	Limit uint
	// a function that returns true if the request should not count toward the rate limit
	Skip func(*gin.Context) bool
}

func InMemoryStore(options *InMemoryOptions) Store {
	data := &sync.Map{}
	store := inMemoryStoreType{
		rate:  int64(options.Rate.Seconds()),
		limit: options.Limit,
		data:  data,
		skip:  options.Skip,
	}
	go clearInBackground(data, store.rate)
	return &store
}
