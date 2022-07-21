package ratelimit

import (
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

type user struct {
	ts     int64
	tokens int
}

func clearInBackground(data *sync.Map, rate int64) {
	for {
		data.Range(func(k, v any) bool {
			if v.(user).ts+rate <= time.Now().Unix() {
				data.Delete(k)
			}
			return true
		})
		time.Sleep(time.Minute)
	}
}

type InMemoryStoreType struct {
	rate  int64
	limit int
	data  *sync.Map
}

func (s *InMemoryStoreType) Limit(key string) (bool, time.Duration) {
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

type store interface {
	Limit(key string) (bool, time.Duration)
}

func InMemoryStore(rate time.Duration, limit int) *InMemoryStoreType {
	data := &sync.Map{}
	store := InMemoryStoreType{int64(rate.Seconds()), limit, data}
	go clearInBackground(data, store.rate)
	return &store
}

func RateLimiter(keyFunc func(c *gin.Context) string, errorHandler func(c *gin.Context, remaining time.Duration), s store) func(ctx *gin.Context) {
	return func(c *gin.Context) {
		key := keyFunc(c)
		limited, remaining := s.Limit(key)
		if limited {
			errorHandler(c, remaining)
			c.Abort()
		} else {
			c.Next()
		}
	}
}
