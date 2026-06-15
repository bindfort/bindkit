package metering

import (
	"context"
	"sync"
)

type Store interface {
	Increment(ctx context.Context, key string) error
	Count(ctx context.Context, key string) (int, error)
}

type MemoryStore struct {
	mu     sync.RWMutex
	counts map[string]int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{counts: map[string]int{}}
}

func (s *MemoryStore) Increment(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[key]++
	return nil
}

func (s *MemoryStore) Count(_ context.Context, key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.counts[key], nil
}
