package billing

import (
	"context"
	"errors"

	"github.com/bindfort/bindkit/internal/metering"
)

var ErrQuotaExceeded = errors.New("quota exceeded")

type QuotaChecker struct {
	store  metering.Store
	limits map[string]int
}

func NewQuotaChecker(store metering.Store, limits map[string]int) *QuotaChecker {
	return &QuotaChecker{store: store, limits: limits}
}

func (q *QuotaChecker) Check(ctx context.Context, plan, key string) error {
	limit := q.limits[plan]
	if limit <= 0 {
		return nil
	}
	count, err := q.store.Count(ctx, key)
	if err != nil {
		return err
	}
	if count >= limit {
		return ErrQuotaExceeded
	}
	return nil
}
