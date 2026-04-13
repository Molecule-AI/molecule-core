// Package middleware provides HTTP middleware for the platform API.
package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a simple token bucket rate limiter per IP.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    int           // tokens per interval
	interval time.Duration
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a rate limiter with the given rate per interval.
// Pass a context to stop the cleanup goroutine on shutdown.
func NewRateLimiter(rate int, interval time.Duration, ctx context.Context) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
	}
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rl.mu.Lock()
				cutoff := time.Now().Add(-10 * time.Minute)
				for ip, b := range rl.buckets {
					if b.lastReset.Before(cutoff) {
						delete(rl.buckets, ip)
					}
				}
				rl.mu.Unlock()
			}
		}
	}()
	return rl
}

// Middleware returns a Gin middleware that rate limits by client IP.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		b, exists := rl.buckets[ip]
		if !exists {
			b = &bucket{tokens: rl.rate, lastReset: time.Now()}
			rl.buckets[ip] = b
		}

		// Reset tokens if interval has passed
		if time.Since(b.lastReset) >= rl.interval {
			b.tokens = rl.rate
			b.lastReset = time.Now()
		}

		if b.tokens <= 0 {
			rl.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": rl.interval.Seconds(),
			})
			c.Abort()
			return
		}

		b.tokens--
		rl.mu.Unlock()

		c.Next()
	}
}
