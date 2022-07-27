package ratelimit

import (
	"github.com/gin-gonic/gin"
	"time"
)

type Store interface {
	Limit(key string) (bool, time.Duration)
	Skip(c *gin.Context) bool
}

func RateLimiter(keyFunc func(c *gin.Context) string, errorHandler func(c *gin.Context, remaining time.Duration), s Store) func(ctx *gin.Context) {
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
