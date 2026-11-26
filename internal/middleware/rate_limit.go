package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

const maksIntipBody = 64 << 10

type Penyimpanan interface {
	allow(key string, r rate.Limit, burst int, block time.Duration) (bool, time.Duration)
}

type IPRateLimiter struct {
	scope         string
	store         Penyimpanan
	rate          rate.Limit
	burst         int
	blockDuration time.Duration
}

type Dimensi struct {
	nama    string
	resolve func(*gin.Context) string
}

func DimIP() Dimensi {
	return Dimensi{nama: "ip", resolve: func(c *gin.Context) string { return c.ClientIP() }}
}

func DimUser() Dimensi {
	return Dimensi{nama: "usr", resolve: userIDUntukLimit}
}

func DimBody() Dimensi {
	return Dimensi{nama: "idt", resolve: identitasDariBody}
}

func userIDUntukLimit(c *gin.Context) string {
	if claims, ok := GetClaims(c); ok && claims != nil {
		return claims.UserID
	}
	tokenStr := utils.ExtractTokenFromRequest(c)
	if tokenStr == "" {
		return ""
	}
	claims, err := utils.ParseJWT(tokenStr)
	if err != nil || claims == nil {
		return ""
	}
	return claims.UserID
}

func identitasDariBody(c *gin.Context) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}
	if !strings.HasPrefix(c.ContentType(), "application/json") {
		return ""
	}

	buf, err := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(buf))
	if err != nil || len(buf) == 0 || len(buf) > maksIntipBody {
		return ""
	}

	var isi struct {
		Email  string `json:"email"`
		UserID string `json:"user_id"`
	}
	if json.Unmarshal(buf, &isi) != nil {
		return ""
	}
	if email := strings.ToLower(strings.TrimSpace(isi.Email)); email != "" {
		return email
	}
	return strings.TrimSpace(isi.UserID)
}

func NewIPRateLimiter(scope string, r rate.Limit, b int) *IPRateLimiter {
	return NewIPRateLimiterWithBlock(scope, r, b, 0)
}

func NewIPRateLimiterWithBlock(scope string, r rate.Limit, b int, blockDuration time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		scope:         scope,
		store:         NewStoreMemori(),
		rate:          r,
		burst:         b,
		blockDuration: blockDuration,
	}
}

func (l *IPRateLimiter) WithStore(s Penyimpanan) *IPRateLimiter {
	if s != nil {
		l.store = s
	}
	return l
}

func (l *IPRateLimiter) kunci(dim, nilai string) string {
	return "rl:" + l.scope + ":" + dim + ":" + nilai
}

func (l *IPRateLimiter) Middleware(dims ...Dimensi) gin.HandlerFunc {
	if len(dims) == 0 {
		dims = []Dimensi{DimIP()}
	}

	return func(c *gin.Context) {
		ditolak := false
		var retryAfter time.Duration

		for _, d := range dims {
			nilai := d.resolve(c)
			if nilai == "" {
				continue
			}
			allowed, ra := l.store.allow(l.kunci(d.nama, nilai), l.rate, l.burst, l.blockDuration)
			if !allowed {
				ditolak = true
				if ra > retryAfter {
					retryAfter = ra
				}
			}
		}

		if !ditolak {
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

const ttlEntriMemori = 10 * time.Minute

type StoreMemori struct {
	mu       sync.Mutex
	clients  map[string]*clientBucket
	stopCh   chan struct{}
	stopOnce sync.Once
}

type clientBucket struct {
	limiter      *rate.Limiter
	lastSeen     time.Time
	blockedUntil time.Time
}

func NewStoreMemori() *StoreMemori {
	s := &StoreMemori{
		clients: make(map[string]*clientBucket),
		stopCh:  make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

func (s *StoreMemori) Stop() {
	s.stopOnce.Do(func() { close(s.stopCh) })
}

func (s *StoreMemori) allow(key string, r rate.Limit, burst int, block time.Duration) (bool, time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	c, ok := s.clients[key]
	if !ok {
		c = &clientBucket{limiter: rate.NewLimiter(r, burst)}
		s.clients[key] = c
	}
	c.lastSeen = now

	if now.Before(c.blockedUntil) {
		return false, c.blockedUntil.Sub(now)
	}

	if c.limiter.AllowN(now, 1) {
		return true, 0
	}

	if block > 0 {
		c.blockedUntil = now.Add(block)
		return false, block
	}
	return false, 0
}

func (s *StoreMemori) cleanupLoop() {
	ticker := time.NewTicker(ttlEntriMemori)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, c := range s.clients {
				if now.Sub(c.lastSeen) > ttlEntriMemori && now.After(c.blockedUntil) {
					delete(s.clients, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
