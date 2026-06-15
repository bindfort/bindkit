package ratelimit

import (
	"context"
	"errors"
	"testing"
)

func TestLimiterAllowsFirstCallAndBlocksSecondAtOnePerMinute(t *testing.T) {
	limiter := New(1)
	if err := limiter.Allow(context.Background(), "key"); err != nil {
		t.Fatalf("first call should pass: %v", err)
	}
	if err := limiter.Allow(context.Background(), "key"); !errors.Is(err, ErrLimited) {
		t.Fatalf("second call should be limited, got %v", err)
	}
}

func TestLimiterSeparatesKeys(t *testing.T) {
	limiter := New(1)
	if err := limiter.Allow(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	if err := limiter.Allow(context.Background(), "b"); err != nil {
		t.Fatalf("different key should have its own bucket: %v", err)
	}
}
