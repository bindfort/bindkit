package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrLimited = errors.New("rate limit exceeded")

// sweepEvery controls how often (in Allow calls) the limiter evicts idle buckets
// so the map cannot grow without bound under a large number of distinct keys.
const sweepEvery = 1024

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu        sync.Mutex
	rate      float64
	capacity  float64
	fullAfter time.Duration
	now       func() time.Time
	buckets   map[string]*bucket
	ops       int
}

func New(ratePerMinute int) *Limiter {
	if ratePerMinute <= 0 {
		ratePerMinute = 60
	}
	rate := float64(ratePerMinute) / 60
	return &Limiter{
		rate:     rate,
		capacity: float64(ratePerMinute),
		// An empty bucket fully refills after capacity/rate seconds; once full it
		// is indistinguishable from a brand-new bucket, so it is safe to evict.
		fullAfter: time.Duration(float64(ratePerMinute) / rate * float64(time.Second)),
		now:       time.Now,
		buckets:   map[string]*bucket{},
	}
}

func (l *Limiter) Allow(_ context.Context, key string) error {
	if key == "" {
		key = "anonymous"
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if l.ops++; l.ops >= sweepEvery {
		l.ops = 0
		l.sweep(now)
	}

	b := l.buckets[key]
	if b == nil {
		l.buckets[key] = &bucket{tokens: l.capacity - 1, last: now}
		return nil
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.last = now
	if b.tokens < 1 {
		return ErrLimited
	}
	b.tokens--
	return nil
}

// sweep drops buckets that have been idle long enough to have fully refilled.
// The caller must hold l.mu.
func (l *Limiter) sweep(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.last) >= l.fullAfter {
			delete(l.buckets, key)
		}
	}
}
