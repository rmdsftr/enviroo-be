package middleware

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

type primaryPalsu struct {
	panggilan int
	err       error
	ok        bool
	retry     time.Duration
}

func (p *primaryPalsu) coba(string, rate.Limit, int, time.Duration) (bool, time.Duration, error) {
	p.panggilan++
	if p.err != nil {
		return true, 0, p.err
	}
	return p.ok, p.retry, nil
}

type secondaryPalsu struct {
	panggilan int
	ok        bool
}

func (s *secondaryPalsu) allow(string, rate.Limit, int, time.Duration) (bool, time.Duration) {
	s.panggilan++
	return s.ok, 0
}

func fallbackUji(p storeDenganError, sec Penyimpanan) (*StoreFallback, func(time.Duration)) {
	jam := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	f := NewStoreFallback(p, sec)
	f.now = func() time.Time { return jam }
	return f, func(d time.Duration) { jam = jam.Add(d) }
}

func TestStoreFallback_PrimarySehatSecondaryTidakDipakai(t *testing.T) {
	p := &primaryPalsu{ok: false, retry: 2 * time.Minute}
	sec := &secondaryPalsu{ok: true}
	f, _ := fallbackUji(p, sec)

	ok, ra := f.allow("k", rate.Every(time.Second), 5, 0)

	assert.False(t, ok, "keputusan harus datang dari primary")
	assert.Equal(t, 2*time.Minute, ra)
	assert.Zero(t, sec.panggilan, "secondary tidak boleh disentuh saat primary sehat")
}

func TestStoreFallback_PrimaryErrorJatuhKeSecondary(t *testing.T) {
	p := &primaryPalsu{err: errors.New("redis mati")}
	sec := &secondaryPalsu{ok: false}
	f, _ := fallbackUji(p, sec)

	ok, _ := f.allow("k", rate.Every(time.Second), 5, 0)

	assert.False(t, ok, "keputusan harus datang dari secondary")
	assert.Equal(t, 1, sec.panggilan)
}

func TestStoreFallback_BreakerBerhentiMemanggilPrimary(t *testing.T) {
	p := &primaryPalsu{err: errors.New("redis mati")}
	sec := &secondaryPalsu{ok: true}
	f, maju := fallbackUji(p, sec)

	for i := 0; i < 50; i++ {
		ok, _ := f.allow("k", rate.Every(time.Second), 5, 0)
		assert.True(t, ok, "fail-open: request harus tetap lolos")
	}
	assert.Equal(t, ambangGagalDefault, p.panggilan,
		"setelah breaker terbuka, primary tidak boleh dipanggil lagi")
	assert.Equal(t, 50, sec.panggilan)

	maju(cooldownDefault + time.Second)
	for i := 0; i < 10; i++ {
		f.allow("k", rate.Every(time.Second), 5, 0)
	}
	assert.Equal(t, ambangGagalDefault+1, p.panggilan,
		"lewat cooldown hanya satu request yang dilepas sebagai probe")
}

func TestStoreFallback_ProbeSuksesMenutupBreaker(t *testing.T) {
	p := &primaryPalsu{err: errors.New("redis mati")}
	sec := &secondaryPalsu{ok: true}
	f, maju := fallbackUji(p, sec)

	for i := 0; i < ambangGagalDefault; i++ {
		f.allow("k", rate.Every(time.Second), 5, 0)
	}

	p.err = nil
	p.ok = false
	maju(cooldownDefault + time.Second)

	ok, _ := f.allow("k", rate.Every(time.Second), 5, 0)
	assert.False(t, ok, "probe sukses, keputusan kembali dari primary")

	sebelum := sec.panggilan
	for i := 0; i < 5; i++ {
		f.allow("k", rate.Every(time.Second), 5, 0)
	}
	assert.Equal(t, sebelum, sec.panggilan, "breaker harus sudah tertutup")
}

func TestStoreFallback_LogHanyaSaatTransisi(t *testing.T) {
	var buf bytes.Buffer
	asli := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(asli)

	p := &primaryPalsu{err: errors.New("redis mati")}
	sec := &secondaryPalsu{ok: true}
	f, maju := fallbackUji(p, sec)

	for i := 0; i < 100; i++ {
		f.allow("k", rate.Every(time.Second), 5, 0)
	}
	assert.Equal(t, 1, strings.Count(buf.String(), "fallback ke in-memory"),
		"log per request malah jadi beban sendiri saat Redis mati di bawah beban")

	p.err = nil
	maju(cooldownDefault + time.Second)
	for i := 0; i < 20; i++ {
		f.allow("k", rate.Every(time.Second), 5, 0)
	}
	assert.Equal(t, 1, strings.Count(buf.String(), "Redis pulih"))
}

func TestStoreFallback_SecondaryMemoriNyata(t *testing.T) {
	p := &primaryPalsu{err: errors.New("redis mati")}
	f, _ := fallbackUji(p, NewStoreMemori())

	r, burst := rate.Every(time.Hour), 2
	ok, _ := f.allow("rl:uji:ip:1.1.1.1", r, burst, 0)
	assert.True(t, ok)
	ok, _ = f.allow("rl:uji:ip:1.1.1.1", r, burst, 0)
	assert.True(t, ok)
	ok, _ = f.allow("rl:uji:ip:1.1.1.1", r, burst, 0)
	assert.False(t, ok, "saat fallback, limit masih ditegakkan per instance")
}
