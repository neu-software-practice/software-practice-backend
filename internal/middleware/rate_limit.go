package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	apperrors "github.com/neuhis/software-practice-backend/internal/errors"
)

// RateLimiter implements a simple token-bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     float64 // tokens per second
	capacity float64 // max tokens
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

// NewRateLimiter creates a new rate limiter.
// rate: requests per second, capacity: burst size.
func NewRateLimiter(rate, capacity float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     rate,
		capacity: capacity,
	}
	// Clean up stale buckets periodically
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			if now.Sub(bucket.lastTime) > 5*time.Minute {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.buckets[key]
	if !ok {
		bucket = &tokenBucket{
			tokens:   rl.capacity,
			lastTime: time.Now(),
		}
		rl.buckets[key] = bucket
		return true
	}

	now := time.Now()
	elapsed := now.Sub(bucket.lastTime).Seconds()
	bucket.tokens += elapsed * rl.rate
	if bucket.tokens > rl.capacity {
		bucket.tokens = rl.capacity
	}
	bucket.lastTime = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}
	return false
}

// RateLimitMiddleware creates a rate-limiting middleware.
func RateLimitMiddleware(rate, capacity float64) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, capacity)

	return func(c *gin.Context) {
		// Use IP or token as key
		key := c.ClientIP()
		if patientID := c.GetString("patientId"); patientID != "" {
			key = "patient:" + patientID
		}

		if !limiter.allow(key) {
			apperrors.WriteError(c, apperrors.NewApiError(
				"RATE_LIMITED",
				"too many requests, please try again later",
				http.StatusTooManyRequests,
			))
			return
		}

		c.Next()
	}
}
