<a href="https://jgltechnologies.com/discord">
<img src="https://discord.com/api/guilds/844418702430175272/embed.png">
</a>

# gin-rate-limit

gin-rate-limit is a rate limiter for the <a href="https://github.com/gin-gonic/gin">gin framework</a>. By default, it
can only store rate limit info in memory and with redis. If you want to store it somewhere else you can make your own
store or use third party stores. The library is new so there are no third party stores yet, so I would appreciate if
someone could make one.

Install

 ```shell
 go get github.com/JGLTechnologies/gin-rate-limit
```

<br>

Redis Example

```go
package main

import (
	"github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"time"
)

func keyFunc(c *gin.Context) string {
	return c.ClientIP()
}

func errorHandler(c *gin.Context, remaining time.Duration) {
	c.String(429, "Too many requests. Try again in "+remaining.String())
}

func main() {
	server := gin.Default()
	// This makes it so each ip can only make 5 requests per second
	store := ratelimit.RedisStore(&ratelimit.RedisOptions{
		RedisClient: redis.NewClient(&redis.Options{
			Addr: "localhost:7680",
		}),
		Rate:  time.Second,
		Limit: 5,
	})
	mw := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHanlder: errorHandler,
		KeyFunc: keyfunc,
    })
	server.GET("/", mw, func(c *gin.Context) {
		c.String(200, "Hello World")
	})
	server.Run(":8080")
}
```

<br>

Basic Setup

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/JGLTechnologies/gin-rate-limit"
	"time"
)

func keyFunc(c *gin.Context) string {
	return c.ClientIP()
}

func errorHandler(c *gin.Context, remaining time.Duration) {
	c.String(429, "Too many requests. Try again in "+remaining.String())
}

func main() {
	server := gin.Default()
	// This makes it so each ip can only make 5 requests per second
	store := ratelimit.InMemoryStore(&ratelimit.InMemoryOptions{
		Rate:  time.Second,
		Limit: 5,
	})
	mw := ratelimit.RateLimiter(store, &ratelimit.Options{
		ErrorHanlder: errorHandler,
		KeyFunc: keyfunc,
	})
	server.GET("/", mw, func(c *gin.Context) {
		c.String(200, "Hello World")
	})
	server.Run(":8080")
}
```

<br>


Custom Store Example

```go
package main

import (
	"github.com/gin-gonic/gin"
	"time"
)

type CustomStore struct {
}

// Your store must have a method called Limit that takes a key and returns a bool, time.Duration
func (s *CustomStore) Limit(key string) (bool, time.Duration) {
	// Do your rate limit logic, and return true if the user went over the rate limit, otherwise return false
	// Return the amount of time the client needs to wait to make a new request
	if UserWentOverLimit {
		return true, remaining
	}
	return false, remaining
}
```