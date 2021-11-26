<a href="https://jgltechnologies.com/discord">
<img src="https://discord.com/api/guilds/844418702430175272/embed.png">
</a>

# GinRateLimit

GinRateLimit is a rate limiter for the <a href="https://github.com/gin-gonic/gin">gin framework</a>. By default, it can
only store rate limit info in memory. If you want to store it somewhere else like redis you can make your own store or
use third party stores, similar to how <a href="https://github.com/nfriedly/express-rate-limit">express-rate-limit</a> does it. The library is new so there are no third party stores yet, so I would appreciate if someone
could make one.

Install

 ```shell
 go get github.com/Nebulizer1213/GinRateLimit
```

<br>

Basic Setup

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/Nebulizer1213/GinRateLimit"
)

func keyFunc(c *gin.Context) string {
	return c.ClientIP()
}

func errorHandler(c *gin.Context) {
	c.String(429, "Too many requests")
}

func main() {
	server := gin.Default()
	// This makes it so each ip can only make 5 requests per second
	store := GinRateLimit.InMemoryStore(1, 5)
	mw := GinRateLimit.RateLimiter(keyFunc, errorHandler, store)
	server.GET("/", mw, func(c *gin.Context) {
		c.String(200, "Hello World")
	})
}
```

<br>

Custom Store Example

```go
package main

type CustomStore struct {
}

// Your store must have a method called Limit that takes a key and returns a bool
func (s *CustomStore) Limit(key string) bool {
	// Do your rate limit logic, and return true if the user went over the rate limit, otherwise return false
	if UserWentOverLimit {
		return true
	}
	return false
}
```