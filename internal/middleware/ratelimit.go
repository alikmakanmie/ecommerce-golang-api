package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
)

// clientIP returns the first IP from X-Forwarded-For when present (the
// tunnel / reverse proxy in front of us sets it), otherwise the remote
// address without its port.
func clientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

type bucket struct {
	tokens   float64
	last     time.Time
	lastSeen time.Time
}

// RateLimiter is a per-key token-bucket limiter kept in memory.
type RateLimiter struct {
	rate    float64 // tokens added per second
	burst   int
	mu      sync.Mutex
	buckets map[string]*bucket
}

func NewRateLimiter(ratePerSecond float64, burst int) *RateLimiter {
	l := &RateLimiter{rate: ratePerSecond, burst: burst, buckets: make(map[string]*bucket)}
	go l.cleanup()
	return l
}

// cleanup drops buckets idle for more than 5 minutes, once a minute.
func (l *RateLimiter) cleanup() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		cutoff := time.Now().Add(-5 * time.Minute)
		l.mu.Lock()
		for k, b := range l.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(l.buckets, k)
			}
		}
		l.mu.Unlock()
	}
}

func (l *RateLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(l.burst), last: now, lastSeen: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.last = now
	b.lastSeen = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Limit returns a Gin middleware enforcing this limiter per client IP.
func (l *RateLimiter) Limit(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(clientIP(c)) {
			utils.Error(c, http.StatusTooManyRequests, message)
			c.Abort()
			return
		}
		c.Next()
	}
}
