package middleware

import (
	"regexp"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func storeRedisUji(t *testing.T) (*StoreRedis, *miniredis.Miniredis, func(time.Duration)) {
	t.Helper()
	m := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	jam := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	m.SetTime(jam)

	maju := func(d time.Duration) {
		jam = jam.Add(d)
		m.SetTime(jam)
		m.FastForward(d)
	}

	return NewStoreRedis(rdb, time.Second), m, maju
}

func TestStoreRedis_BurstTepatLalimTolak(t *testing.T) {
	s, _, _ := storeRedisUji(t)
	r, burst := rate.Every(6*time.Second), 5

	for i := 0; i < burst; i++ {
		ok, ra, err := s.coba("rl:uji:ip:1.1.1.1", r, burst, 0)
		require.NoError(t, err)
		assert.True(t, ok, "request ke-%d harus lolos", i+1)
		assert.Zero(t, ra)
	}

	ok, ra, err := s.coba("rl:uji:ip:1.1.1.1", r, burst, 0)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Zero(t, ra, "tanpa block, retryAfter harus 0 supaya pesan 429 tetap yang generik")
}

func TestStoreRedis_SatuEmissionMengisiSatuToken(t *testing.T) {
	s, _, maju := storeRedisUji(t)
	r, burst := rate.Every(6*time.Second), 5

	for i := 0; i < burst; i++ {
		_, _, err := s.coba("k", r, burst, 0)
		require.NoError(t, err)
	}
	ok, _, _ := s.coba("k", r, burst, 0)
	assert.False(t, ok)

	maju(6 * time.Second)
	ok, _, err := s.coba("k", r, burst, 0)
	require.NoError(t, err)
	assert.True(t, ok, "satu token harus sudah terisi")

	ok, _, _ = s.coba("k", r, burst, 0)
	assert.False(t, ok, "hanya satu token yang terisi")
}

func TestStoreRedis_TTLTidakPernahMinusSatu(t *testing.T) {
	s, m, _ := storeRedisUji(t)
	r, burst := rate.Every(6*time.Second), 5
	key := "rl:uji:ip:1.1.1.1"

	for i := 0; i < burst; i++ {
		ok, _, err := s.coba(key, r, burst, 0)
		require.NoError(t, err)
		require.True(t, ok)

		ttl := m.TTL(key + StateSuffix)
		assert.Greater(t, ttl, time.Duration(0),
			"kunci state tanpa TTL berarti counter tidak pernah reset")
		assert.LessOrEqual(t, ttl, 30*time.Second)
	}
}

func TestStoreRedis_NilaiTersimpanIntegerPolos(t *testing.T) {
	s, m, _ := storeRedisUji(t)
	key := "rl:uji:ip:1.1.1.1"

	ok, _, err := s.coba(key, rate.Every(6*time.Second), 5, 0)
	require.NoError(t, err)
	require.True(t, ok)

	nilai, err := m.Get(key + StateSuffix)
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^\d{13,}$`), nilai,
		"tanpa string.format('%.0f') epoch-ms bisa keluar sebagai 1.7914e+12")
}

func TestStoreRedis_PenaltyBox(t *testing.T) {
	s, m, maju := storeRedisUji(t)
	r, burst, block := rate.Every(6*time.Second), 5, 10*time.Minute
	key := "rl:uji:idt:korban@enviroo.test"

	for i := 0; i < burst; i++ {
		_, _, err := s.coba(key, r, burst, block)
		require.NoError(t, err)
	}

	ok, ra, err := s.coba(key, r, burst, block)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, block, ra)

	ttl := m.TTL(key + BlockSuffix)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, block)

	maju(4 * time.Minute)
	ok, ra, err = s.coba(key, r, burst, block)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, 6*time.Minute, ra, "sisa blokir harus mengecil, bukan direset")

	maju(6 * time.Minute)
	for i := 0; i < burst; i++ {
		ok, _, err := s.coba(key, r, burst, block)
		require.NoError(t, err)
		assert.True(t, ok, "burst penuh harus kembali setelah blokir habis")
	}
}

func TestStoreRedis_FloodingTidakMemperpanjangBlokir(t *testing.T) {
	s, _, maju := storeRedisUji(t)
	r, burst, block := rate.Every(6*time.Second), 5, 10*time.Minute
	key := "k"

	for i := 0; i < burst; i++ {
		_, _, err := s.coba(key, r, burst, block)
		require.NoError(t, err)
	}
	_, _, err := s.coba(key, r, burst, block)
	require.NoError(t, err)

	for i := 0; i < 500; i++ {
		maju(time.Second)
		ok, _, err := s.coba(key, r, burst, block)
		require.NoError(t, err)
		require.False(t, ok)
	}

	_, ra, err := s.coba(key, r, burst, block)
	require.NoError(t, err)
	assert.Equal(t, 100*time.Second, ra,
		"request yang ditolak tidak boleh memperpanjang hukumannya sendiri")
}

func TestStoreRedis_ScopeMemisahkanBucket(t *testing.T) {
	s, _, _ := storeRedisUji(t)
	r, burst := rate.Every(time.Hour), 1

	globalIP := NewIPRateLimiter("glob_ip", r, burst).WithStore(s)
	loginIP := NewIPRateLimiter("login_ip", r, burst).WithStore(s)

	ok, _ := s.allow(globalIP.kunci("ip", "1.1.1.1"), r, burst, 0)
	assert.True(t, ok)
	ok, _ = s.allow(globalIP.kunci("ip", "1.1.1.1"), r, burst, 0)
	assert.False(t, ok, "kuota glob_ip habis")

	ok, _ = s.allow(loginIP.kunci("ip", "1.1.1.1"), r, burst, 0)
	assert.True(t, ok, "login_ip harus punya bucket sendiri walau IP sama dan store sama")
}

func TestStoreRedis_DimensiTidakBerbagiBucket(t *testing.T) {
	s, _, _ := storeRedisUji(t)
	r, burst := rate.Every(time.Hour), 1
	l := NewIPRateLimiter("login_ip", r, burst).WithStore(s)

	ok, _ := s.allow(l.kunci("ip", "sama"), r, burst, 0)
	assert.True(t, ok)
	ok, _ = s.allow(l.kunci("ip", "sama"), r, burst, 0)
	assert.False(t, ok)

	ok, _ = s.allow(l.kunci("idt", "sama"), r, burst, 0)
	assert.True(t, ok, "dimensi berbeda harus bucket berbeda")
}

func TestStoreRedis_RuleTidakTerkonfigurasiDilewatkan(t *testing.T) {
	s, _, _ := storeRedisUji(t)

	ok, _, err := s.coba("k", 0, 5, 0)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, _, err = s.coba("k", rate.Every(time.Second), 0, 0)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestStoreRedis_PulihSetelahScriptFlush(t *testing.T) {
	s, _, _ := storeRedisUji(t)
	r, burst := rate.Every(time.Second), 5

	ok, _, err := s.coba("k", r, burst, 0)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, s.rdb.ScriptFlush(t.Context()).Err())

	ok, _, err = s.coba("k", r, burst, 0)
	require.NoError(t, err, "harus memuat ulang script setelah NOSCRIPT")
	assert.True(t, ok)
}

func TestStoreRedis_RedisMatiFailOpen(t *testing.T) {
	s, m, _ := storeRedisUji(t)
	m.Close()

	ok, ra, err := s.coba("k", rate.Every(time.Second), 1, 0)
	assert.Error(t, err, "coba harus melaporkan error supaya fallback bisa bereaksi")
	assert.True(t, ok)

	ok, ra = s.allow("k", rate.Every(time.Second), 1, 0)
	assert.True(t, ok, "allow harus fail-open")
	assert.Zero(t, ra)
}
