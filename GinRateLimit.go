package GinRateLimit

import (
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

type expiringDict struct {
	mutex      *sync.Mutex
	data       map[string]int
	expiryData map[string]int
}

func (e *expiringDict) incr(key string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	v, _ := e.expiryData[key]
	if v <= int(time.Now().Unix()) {
		delete(e.data, key)
		delete(e.expiryData, key)
	}
	e.data[key]++
}

func (e *expiringDict) get(key string) int {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	v, _ := e.expiryData[key]
	if v <= int(time.Now().Unix()) {
		delete(e.data, key)
		delete(e.expiryData, key)
	}
	return e.data[key]
}

func (e *expiringDict) expire(key string, seconds int) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	_, ok := e.expiryData[key]
	if ok {
		return
	} else {
		e.expiryData[key] = int(time.Now().Unix()) + seconds
	}
}

func (e *expiringDict) clearInBackground() {
	for {
		e.mutex.Lock()
		for k, v := range e.expiryData {
			if v <= int(time.Now().Unix()) {
				delete(e.data, k)
				delete(e.expiryData, k)
			}
		}
		e.mutex.Unlock()
		time.Sleep(time.Minute)
	}
}

type inMemoryStore struct {
	rate  int
	limit int
	data  expiringDict
}

type store interface {
	Limit(key string) bool
}

func InMemoryStore(rate int, limit int) inMemoryStore {
	data := expiringDict{&sync.Mutex{}, map[string]int{}, map[string]int{}}
	store := inMemoryStore{rate, limit, data}
	go data.clearInBackground()
	return store
}

func (s *inMemoryStore) Limit(key string) bool {
	s.data.incr(key)
	s.data.expire(key, s.rate)
	if s.data.get(key) > s.limit {
		return true
	}
	return false
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
