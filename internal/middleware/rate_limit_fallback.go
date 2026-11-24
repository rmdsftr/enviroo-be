package middleware

import (
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	ambangGagalDefault = 3
	cooldownDefault    = 5 * time.Second
)

type storeDenganError interface {
	coba(key string, r rate.Limit, burst int, block time.Duration) (bool, time.Duration, error)
}

type StoreFallback struct {
	primary   storeDenganError
	secondary Penyimpanan
	ambang    int
	cooldown  time.Duration
	now       func() time.Time

	mu         sync.Mutex
	gagal      int
	bukaSampai time.Time
}

func NewStoreFallback(primary storeDenganError, secondary Penyimpanan) *StoreFallback {
	return &StoreFallback{
		primary:   primary,
		secondary: secondary,
		ambang:    ambangGagalDefault,
		cooldown:  cooldownDefault,
		now:       time.Now,
	}
}

func (s *StoreFallback) allow(key string, r rate.Limit, burst int, block time.Duration) (bool, time.Duration) {
	if !s.bolehCoba() {
		return s.secondary.allow(key, r, burst, block)
	}

	ok, retryAfter, err := s.primary.coba(key, r, burst, block)
	if err != nil {
		s.catatGagal(err)
		return s.secondary.allow(key, r, burst, block)
	}

	s.catatSukses()
	return ok, retryAfter
}

func (s *StoreFallback) bolehCoba() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.gagal < s.ambang {
		return true
	}
	if s.now().Before(s.bukaSampai) {
		return false
	}
	s.bukaSampai = s.now().Add(s.cooldown)
	return true
}

func (s *StoreFallback) catatGagal(err error) {
	s.mu.Lock()
	s.gagal++
	baruTerbuka := s.gagal == s.ambang
	if s.gagal >= s.ambang {
		s.bukaSampai = s.now().Add(s.cooldown)
	}
	s.mu.Unlock()

	if baruTerbuka {
		log.Printf("Warning: rate limit Redis gagal %d kali berturut-turut, fallback ke in-memory (coba lagi tiap %s): %v",
			s.ambang, s.cooldown, err)
	}
}

func (s *StoreFallback) catatSukses() {
	s.mu.Lock()
	pulih := s.gagal >= s.ambang
	s.gagal = 0
	s.bukaSampai = time.Time{}
	s.mu.Unlock()

	if pulih {
		log.Println("rate limit: Redis pulih, kembali memakai backend Redis")
	}
}
