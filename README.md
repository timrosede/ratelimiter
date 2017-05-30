# ratelimiter

This is an implementation of a rolling window rate limiter in golang.

## Usage

```Go
package main

import (
	"net/http"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/iamtimde/ratelimiter"
)

func main() {

	redispool := &redis.Pool{
		MaxIdle:     2,
		IdleTimeout: 600 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ":6379")
			if err != nil {
				return nil, err
			}

			return c, nil
		},
	}

	rateLimiter := ratelimiter.RateLimiter{
		Namespace: "default",
		Limit:     100,
		Expires:   60 * time.Second,
		Redis:     redispool,
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	}

	http.Handle("/", rateLimiter.Handler(http.HandlerFunc(handler)))

	http.ListenAndServe(":3000", nil)
}
```



