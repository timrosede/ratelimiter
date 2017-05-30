package ratelimiter

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"github.com/stretchr/testify/assert"
)

type redigomockPool interface {
	Get() redis.Conn
}

func getRedisMockPool(zcount int) *redis.Pool {

	mockRedisConn := redigomock.NewConn()

	mockRedisConn.Command("MULTI")
	mockRedisConn.Command("ZREMRANGEBYSCORE")
	mockRedisConn.Command("ZADD")
	mockRedisConn.Command("ZCOUNT", "rate-limiter-testing-", "-inf", "+inf")
	mockRedisConn.Command("EXPIRE")

	values := []interface{}{}
	values = append(values, interface{}([]byte("")))
	values = append(values, interface{}([]byte("")))
	values = append(values, interface{}([]byte(strconv.Itoa(zcount))))

	mockRedisConn.Command("EXEC").Expect(values)

	redisPool := &redis.Pool{Dial: func() (redis.Conn, error) { return mockRedisConn, nil }}
	return redisPool

}

func getRateLimiter(namespace string, limit int, count int) RateLimiter {

	redisPool := getRedisMockPool(count)

	return RateLimiter{
		Namespace: namespace,
		Limit:     limit,
		Expires:   60 * time.Second,
		Redis:     redisPool,
	}

}

func TestRateLimiterTooManyRequests(t *testing.T) {
	namespace := "testing"
	limit := 100
	count := 101

	rateLimiter := getRateLimiter(namespace, limit, count)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	rateLimiter.Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(rr, r)

	assert.Equal(t, strconv.Itoa(limit), rr.Header().Get("X-Ratelimit-Limit"), "handler returned wrong header value")
	assert.Equal(t, strconv.Itoa(limit-count), rr.Header().Get("X-Ratelimit-Remaining"), "handler returned wrong header value")
	assert.Equal(t, http.StatusTooManyRequests, rr.Code, "handler returned wrong status code")
}
func TestRateLimiterOk(t *testing.T) {

	namespace := "testing"
	limit := 100
	count := 10

	rateLimiter := getRateLimiter(namespace, limit, count)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	rateLimiter.Handler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(rr, r)

	assert.Equal(t, strconv.Itoa(limit), rr.Header().Get("X-Ratelimit-Limit"), "handler returned wrong header value")
	assert.Equal(t, strconv.Itoa(limit-count), rr.Header().Get("X-Ratelimit-Remaining"), "handler returned wrong header value")
	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")
}
