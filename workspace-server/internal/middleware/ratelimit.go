// Package middleware provides HTTP middleware for the platform API.
package middleware

import (
	"context"
	"net/http"
	"strconv"
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
		// Tier-1b dev-mode hatch — same gate as AdminAuth / WorkspaceAuth /
		// discovery. On a local single-user Docker setup the 600-req/min
		// bucket fills fast: a 15-workspace canvas + activity polling +
		// approvals polling + A2A overlay + initial hydration all share
		// one IP bucket, so a minute of active use can trip 429 and blank
		// the page. Gated by MOLECULE_ENV=development + empty ADMIN_TOKEN
		// so SaaS production keeps the bucket.
		if isDevModeFailOpen() {
			c.Header("X-RateLimit-Limit", "unlimited")
			c.Next()
			return
		}

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

		// Issue #105 — advertise the current bucket state so clients and
		// monitoring tools can back off proactively. Headers are set on every
		// response (both allowed and throttled) so they're observable against
		// any endpoint — /health, /metrics, and every /workspaces/* route.
		//
		// The `reset` value is seconds until the current bucket refills,
		// matching the RFC 6585 Retry-After spec for 429 responses and the
		// de-facto X-RateLimit-Reset convention (GitHub, Stripe, etc.).
		remaining := b.tokens - 1
		if remaining < 0 {
			remaining = 0
		}
		resetSeconds := int(time.Until(b.lastReset.Add(rl.interval)).Seconds())
		if resetSeconds < 0 {
			resetSeconds = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.Itoa(resetSeconds))

		if b.tokens <= 0 {
			rl.mu.Unlock()
			// Retry-After is the canonical 429 signal per RFC 6585.
			c.Header("Retry-After", strconv.Itoa(resetSeconds))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": resetSeconds,
			})
			c.Abort()
			return
		}

		b.tokens--
		rl.mu.Unlock()

		c.Next()
	}
}
