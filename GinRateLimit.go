package GinRateLimit

import (
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

type user struct {
	time     int
	requests int
}

func clearInBackground(data map[string]user, rate int, mutex *sync.Mutex) {
	for {
		mutex.Lock()
		for k, v := range data {
			if v.time+rate <= int(time.Now().Unix()) {
				delete(data, k)
			}
		}
		mutex.Unlock()
		time.Sleep(time.Minute)
	}
}

type InMemoryStoreType struct {
	rate  int
	limit int
	data  map[string]user
	mutex *sync.Mutex
}

func (s *InMemoryStoreType) Limit(key string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.data[key]
	if !ok {
		s.data[key] = user{int(time.Now().Unix()), s.limit}
	}
	u := s.data[key]
	if u.time+s.rate <= int(time.Now().Unix()) {
		u.requests = s.limit
	}
	if u.requests <= 0 {
		return true
	}
	u.requests--
	u.time = int(time.Now().Unix())
	s.data[key] = u
	return false

}

type store interface {
	Limit(key string) bool
}

func InMemoryStore(rate int, limit int) *InMemoryStoreType {
	mutex := &sync.Mutex{}
	data := map[string]user{}
	store := InMemoryStoreType{rate, limit, data, mutex}
	go clearInBackground(data, rate, mutex)
	return &store
}

func RateLimiter(keyFunc func(c *gin.Context) string, errorHandler func(c *gin.Context), s store) func(ctx *gin.Context) {
	return func(c *gin.Context) {
		key := keyFunc(c)
		limited := s.Limit(key)
		if limited {
			errorHandler(c)
			c.Abort()
		} else {
			c.Next()
		}
	}
}
