package GinRateLimit

import (
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

type user struct {
	ts     int64
	tokens int
}

func clearInBackground(data map[string]*user, rate int64, mutex *sync.Mutex) {
	for {
		mutex.Lock()
		for k, v := range data {
			if v.ts+rate <= time.Now().Unix() {
				delete(data, k)
			}
		}
		mutex.Unlock()
		time.Sleep(time.Minute)
	}
}

type InMemoryStoreType struct {
	rate  int64
	limit int
	data  map[string]*user
	mutex *sync.Mutex
}

func (s *InMemoryStoreType) Limit(key string) (bool, time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.data[key]
	if !ok {
		s.data[key] = &user{time.Now().Unix(), s.limit}
	}
	u := s.data[key]
	if u.ts+s.rate <= time.Now().Unix() {
		u.tokens = s.limit
	}
	remaining := time.Duration((s.rate - (time.Now().Unix() - u.ts)) * time.Second.Nanoseconds())
	if u.tokens <= 0 {
		return true, remaining
	}
	u.tokens--
	u.ts = time.Now().Unix()
	s.data[key] = u
	return false, time.Duration(0)
}

type store interface {
	Limit(key string) (bool, time.Duration)
}

func InMemoryStore(rate time.Duration, limit int) *InMemoryStoreType {
	mutex := &sync.Mutex{}
	data := map[string]*user{}
	store := InMemoryStoreType{int64(rate.Seconds()), limit, data, mutex}
	go clearInBackground(data, store.rate, mutex)
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
