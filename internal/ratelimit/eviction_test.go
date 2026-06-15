package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestLimiterEvictsIdleBuckets(t *testing.T) {
	current := time.Unix(0, 0)
	limiter := New(60)
	limiter.now = func() time.Time { return current }

	// Fill the map with distinct keys, stopping just before a sweep is triggered.
	for i := 0; i < sweepEvery-1; i++ {
		_ = limiter.Allow(context.Background(), fmt.Sprintf("key-%d", i))
	}
	if got := len(limiter.buckets); got != sweepEvery-1 {
		t.Fatalf("expected %d buckets before sweep, got %d", sweepEvery-1, got)
	}

	// Advance past the full-refill window, then make the call that hits the sweep
	// threshold. Every earlier bucket is now idle and must be evicted.
	current = current.Add(2 * time.Minute)
	if err := limiter.Allow(context.Background(), "fresh"); err != nil {
		t.Fatal(err)
	}
	if got := len(limiter.buckets); got != 1 {
		t.Fatalf("expected only the fresh bucket after sweep, got %d", got)
	}
}

func TestLimiterRefillsOverTime(t *testing.T) {
	current := time.Unix(0, 0)
	limiter := New(60) // 1 token/sec, capacity 60
	limiter.now = func() time.Time { return current }

	// Drain the bucket.
	for i := 0; i < 60; i++ {
		if err := limiter.Allow(context.Background(), "k"); err != nil {
			t.Fatalf("call %d should pass: %v", i, err)
		}
	}
	if err := limiter.Allow(context.Background(), "k"); err == nil {
		t.Fatal("expected limit after draining the bucket")
	}

	// One second later, exactly one token has refilled.
	current = current.Add(time.Second)
	if err := limiter.Allow(context.Background(), "k"); err != nil {
		t.Fatalf("expected one refilled token: %v", err)
	}
	if err := limiter.Allow(context.Background(), "k"); err == nil {
		t.Fatal("expected limit again after spending the refilled token")
	}
}
