package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/bindfort/bindkit/internal/metering"
)

func TestQuotaCheckerAllowsUnlimitedPlans(t *testing.T) {
	store := metering.NewMemoryStore()
	checker := NewQuotaChecker(store, map[string]int{"free": 0})
	if err := checker.Check(context.Background(), "free", "key"); err != nil {
		t.Fatalf("expected unlimited plan to pass, got %v", err)
	}
}

func TestQuotaCheckerBlocksAtLimit(t *testing.T) {
	ctx := context.Background()
	store := metering.NewMemoryStore()
	checker := NewQuotaChecker(store, map[string]int{"free": 1})
	if err := checker.Check(ctx, "free", "key"); err != nil {
		t.Fatalf("first check should pass: %v", err)
	}
	if err := store.Increment(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if err := checker.Check(ctx, "free", "key"); !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("expected quota exceeded, got %v", err)
	}
}
