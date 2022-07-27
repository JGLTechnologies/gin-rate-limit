package ratelimit

import (
	"github.com/gin-gonic/gin"
	"time"
)

type Store interface {
	// Limit takes in a key and should return whether that key is allowed to make another request
	Limit(key string) (bool, time.Duration)
	// Skip takes in a *gin.Context and should return whether the rate limiting should be skipped for this request
	Skip(c *gin.Context) bool
}

// RateLimiter is a function to get gin.HandlerFunc
// 	keyFunc: takes in *gin.Context and return a string
// 	errorHandler: takes in *gin.Context and time.Duration
// 	store: Store
func RateLimiter(keyFunc func(c *gin.Context) string, errorHandler func(c *gin.Context, remaining time.Duration), s Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.Skip(c) {
			c.Next()
			return
		}
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
