// Package ratelimit limits request rates with fixed one-minute windows kept in Redis.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter counts hits per key and window.
type Limiter struct {
	rdb    *redis.Client
	prefix string
	limit  int64
	window time.Duration
	now    func() time.Time
}

// New returns a Limiter allowing limit hits per key per window.
func New(rdb *redis.Client, limit int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{rdb: rdb, prefix: "ratelimit", limit: int64(limit), window: window, now: now}
}

// WithPrefix returns a copy of the limiter that keeps its counters under a
// separate key prefix, so that several limiters can share one Redis.
func (l *Limiter) WithPrefix(prefix string) *Limiter {
	c := *l
	c.prefix = prefix
	return &c
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	slot := l.now().UnixNano() / int64(l.window)
	redisKey := fmt.Sprintf("%s:%s:%d", l.prefix, key, slot)

	pipe := l.rdb.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	return incr.Val() <= l.limit, nil
}
