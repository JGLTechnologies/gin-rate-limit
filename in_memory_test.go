package ratelimit

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestInMemoryStoreConcurrentDoesNotExceedLimit fires many concurrent requests
// for a single key and checks that the store never admits more than Limit of
// them. The window (Rate) is long enough that no token refill happens during the
// test, so the only thing under test is the read-modify-write on the token count.
// Before the store took a lock around Load/decrement/Store, concurrent requests
// could each read the same count and store it back, losing decrements and
// admitting well over the limit.
func TestInMemoryStoreConcurrentDoesNotExceedLimit(t *testing.T) {
	const limit = 50

	store := InMemoryStore(&InMemoryOptions{
		Rate:  time.Minute,
		Limit: limit,
	})

	const goroutines = 500
	var allowed int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Skip is nil, so the *gin.Context is never dereferenced here.
			info := store.Limit("user-1", &gin.Context{})
			if !info.RateLimited {
				atomic.AddInt64(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	if allowed > limit {
		t.Fatalf("in-memory store admitted %d concurrent requests, over the limit of %d", allowed, limit)
	}
}
