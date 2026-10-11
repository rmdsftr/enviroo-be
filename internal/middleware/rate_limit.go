package middleware

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	mu            sync.Mutex
	clients       map[string]*clientBucket
	rate          rate.Limit
	burst         int
	ttl           time.Duration
	blockDuration time.Duration
}

type clientBucket struct {
	limiter      *rate.Limiter
	lastSeen     time.Time
	blockedUntil time.Time
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return NewIPRateLimiterWithBlock(r, b, 0)
}

func NewIPRateLimiterWithBlock(r rate.Limit, b int, blockDuration time.Duration) *IPRateLimiter {
	l := &IPRateLimiter{
		clients:       make(map[string]*clientBucket),
		rate:          r,
		burst:         b,
		ttl:           10 * time.Minute,
		blockDuration: blockDuration,
	}
	go l.cleanupLoop()
	return l
}

func (l *IPRateLimiter) allow(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	c, ok := l.clients[ip]
	if !ok {
		c = &clientBucket{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.clients[ip] = c
	}
	c.lastSeen = now

	if now.Before(c.blockedUntil) {
		return false, c.blockedUntil.Sub(now)
	}

	if c.limiter.AllowN(now, 1) {
		return true, 0
	}

	if l.blockDuration > 0 {
		c.blockedUntil = now.Add(l.blockDuration)
		return false, l.blockDuration
	}
	return false, 0
}

func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for ip, c := range l.clients {
			if now.Sub(c.lastSeen) > l.ttl && now.After(c.blockedUntil) {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := l.allow(c.ClientIP())
		if allowed {
			c.Next()
			return
		}

		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("Terlalu banyak percobaan. Silakan coba lagi dalam %d menit.", int(math.Ceil(retryAfter.Minutes()))),
			})
			return
		}

		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
			"error": "Terlalu banyak permintaan. Silakan coba lagi beberapa saat.",
		})
	}
}
