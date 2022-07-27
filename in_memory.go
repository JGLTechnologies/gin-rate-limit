package ratelimit

import (
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

type user struct {
	ts     int64
	tokens uint
}

func clearInBackground(data *sync.Map, rate int64) {
	for {
		data.Range(func(k, v interface{}) bool {
			if v.(user).ts+rate <= time.Now().Unix() {
				data.Delete(k)
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
	skip  func(c *gin.Context) bool
}

func (s *inMemoryStoreType) Limit(key string) (bool, time.Duration) {
	var u user
	m, ok := s.data.Load(key)
	if !ok {
		u = user{time.Now().Unix(), s.limit}
	} else {
		u = m.(user)
	}
	if u.ts+s.rate <= time.Now().Unix() {
		u.tokens = s.limit
	}
	remaining := time.Duration((s.rate - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())
	if u.tokens <= 0 {
		return true, remaining
	}
	u.tokens--
	u.ts = time.Now().Unix()
	s.data.Store(key, u)
	return false, time.Duration(0)
}

func (s *inMemoryStoreType) Skip(c *gin.Context) bool {
	if s.skip != nil {
		return s.skip(c)
	} else {
		return false
	}
}

type InMemoryOptions struct {
	// the user can make Limit amount of requests every Rate
	Rate time.Duration
	// the amount of requests that can be made every Rate
	Limit uint
	// takes in a *gin.Context and should return whether the rate limiting should be skipped for this request
	Skip func(c *gin.Context) bool
}

func InMemoryStore(options *InMemoryOptions) Store {
	data := &sync.Map{}
	store := inMemoryStoreType{int64(options.Rate.Seconds()), options.Limit, data, options.Skip}
	go clearInBackground(data, store.rate)
	return &store
}
