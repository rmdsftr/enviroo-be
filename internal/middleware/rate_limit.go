package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter membatasi jumlah request per alamat IP menggunakan algoritma
// token bucket (in-memory, tanpa dependency eksternal seperti Redis).
//
// Setiap IP punya bucket sendiri. Bucket yang lama tidak aktif dibersihkan
// oleh goroutine cleanup agar map tidak tumbuh tanpa batas.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientBucket
	rate    rate.Limit
	burst   int
	ttl     time.Duration
}

type clientBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter membuat limiter dengan laju isi ulang `r` dan kapasitas burst `b`.
// Contoh: NewIPRateLimiter(rate.Every(6*time.Second), 5) ≈ 10 request/menit, burst 5.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	l := &IPRateLimiter{
		clients: make(map[string]*clientBucket),
		rate:    r,
		burst:   b,
		ttl:     10 * time.Minute,
	}
	go l.cleanupLoop()
	return l
}

// getLimiter mengembalikan bucket untuk sebuah IP, membuatnya jika belum ada.
func (l *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	if c, ok := l.clients[ip]; ok {
		c.lastSeen = time.Now()
		return c.limiter
	}

	limiter := rate.NewLimiter(l.rate, l.burst)
	l.clients[ip] = &clientBucket{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

// cleanupLoop menghapus bucket IP yang tidak aktif lebih dari ttl.
func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.ttl)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, c := range l.clients {
			if time.Since(c.lastSeen) > l.ttl {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

// Middleware menolak request dengan 429 jika IP melebihi kuota.
func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.getLimiter(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Terlalu banyak permintaan. Silakan coba lagi beberapa saat.",
			})
			return
		}
		c.Next()
	}
}
