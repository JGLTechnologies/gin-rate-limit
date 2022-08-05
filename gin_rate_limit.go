package ratelimit

import (
	"github.com/gin-gonic/gin"
	"time"
)

type Store interface {
	// Limit takes in a key and should return whether that key is allowed to make another request
	Limit(key string) (bool, time.Duration)
}

type Options struct {
	ErrorHandler func(*gin.Context, time.Duration)
	KeyFunc      func(*gin.Context) string
	// a function that returns true if the request should not count toward the rate limit
	Skip func(*gin.Context) bool
}

// RateLimiter is a function to get gin.HandlerFunc
func RateLimiter(s Store, options *Options) gin.HandlerFunc {
	if options.ErrorHandler == nil {
		options.ErrorHandler = func(c *gin.Context, remaining time.Duration) {
			c.Header("X-Rate-Limit-Reset", remaining.String())
			c.String(429, "Too many requests")
		}
	}
	if options.KeyFunc == nil {
		options.KeyFunc = func(c *gin.Context) string {
			return c.ClientIP() + c.FullPath()
		}
	}
	return func(c *gin.Context) {
		if options.Skip != nil && options.Skip(c) {
			c.Next()
			return
		}
		key := options.KeyFunc(c)
		limited, remaining := s.Limit(key)
		if limited {
			options.ErrorHandler(c, remaining)
			c.Abort()
		} else {
			c.Next()
		}
	}
}
