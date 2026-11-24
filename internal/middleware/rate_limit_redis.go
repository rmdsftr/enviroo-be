package middleware

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

//go:embed gcra.lua
var gcraSource string

var gcraScript = redis.NewScript(gcraSource)

const (
	StateSuffix = ":s"
	BlockSuffix = ":b"
)

type StoreRedis struct {
	rdb     *redis.Client
	timeout time.Duration
}

func NewClientRedis(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   -1,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
	})
}

func NewStoreRedis(rdb *redis.Client, timeout time.Duration) *StoreRedis {
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	return &StoreRedis{rdb: rdb, timeout: timeout}
}

func (s *StoreRedis) allow(key string, r rate.Limit, burst int, block time.Duration) (bool, time.Duration) {
	allowed, retryAfter, err := s.coba(key, r, burst, block)
	if err != nil {
		return true, 0
	}
	return allowed, retryAfter
}

func (s *StoreRedis) coba(key string, r rate.Limit, burst int, block time.Duration) (bool, time.Duration, error) {
	if r <= 0 || burst <= 0 {
		return true, 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	keys := []string{key + StateSuffix, key + BlockSuffix}
	args := []any{
		float64(time.Second/time.Millisecond) / float64(r),
		burst,
		block.Milliseconds(),
	}

	hasil, err := gcraScript.Run(ctx, s.rdb, keys, args...).Int64Slice()
	if err != nil && redis.HasErrorPrefix(err, "NOSCRIPT") {
		if errLoad := gcraScript.Load(ctx, s.rdb).Err(); errLoad != nil {
			return true, 0, errLoad
		}
		hasil, err = gcraScript.Run(ctx, s.rdb, keys, args...).Int64Slice()
	}
	if err != nil {
		return true, 0, err
	}
	if len(hasil) != 2 {
		return true, 0, fmt.Errorf("gcra: balikan tidak terduga %v", hasil)
	}

	return hasil[0] == 1, time.Duration(hasil[1]) * time.Millisecond, nil
}

func (s *StoreRedis) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.rdb.Ping(ctx).Err()
}
