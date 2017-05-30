package ratelimiter

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
)

type RateLimiter struct {
	Namespace        string
	Limit            int
	Expires          time.Duration
	Redis            *redis.Pool
	customIdentifier func(*http.Request) string
}

func (m *RateLimiter) defaultIdentifier(r *http.Request) string {

	if m.customIdentifier != nil {
		return m.customIdentifier(r)
	}

	if r.Header.Get("X-Forwarded-For") != "" {
		return r.Header.Get("X-Forwarded-For")
	}

	ipaddr, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ipaddr
}

// Handler für das Limit
func (m *RateLimiter) Handler(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		limiterKey := "rate-limiter-" + m.Namespace + "-" + m.defaultIdentifier(r)

		now := time.Now().UnixNano()
		expires := now - int64(m.Expires*time.Second)

		rpool := m.Redis.Get()
		defer rpool.Close()

		rpool.Send("MULTI")
		rpool.Send("ZREMRANGEBYSCORE", limiterKey, 0, expires)
		rpool.Send("ZADD", limiterKey, now, now)
		rpool.Send("ZCOUNT", limiterKey, "-inf", "+inf")
		rpool.Send("EXPIRE", limiterKey, m.Expires)

		res, err := redis.Values(rpool.Do("EXEC"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var hits int
		redis.Scan(res, nil, nil, &hits)

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(m.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(m.Limit-hits))

		if hits > m.Limit {

			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
