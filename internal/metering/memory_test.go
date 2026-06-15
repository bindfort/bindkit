package metering

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryStoreCountsStartAtZero(t *testing.T) {
	store := NewMemoryStore()
	count, err := store.Count(context.Background(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected zero count, got %d", count)
	}
}

func TestMemoryStoreIncrement(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	if err := store.Increment(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if err := store.Increment(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	count, err := store.Count(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestMemoryStoreConcurrentIncrement(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := store.Increment(ctx, "key"); err != nil {
				t.Errorf("increment failed: %v", err)
			}
		}()
	}
	wg.Wait()
	count, err := store.Count(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	if count != 50 {
		t.Fatalf("expected count 50, got %d", count)
	}
}
