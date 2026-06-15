package billing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bindfort/bindkit/internal/metering"
)

// StripeConfig configures usage reporting to Stripe Billing meter events.
type StripeConfig struct {
	SecretKey  string        // Stripe secret key (sk_...)
	MeterEvent string        // meter event_name configured in Stripe
	ValueKey   string        // meter payload value key (Stripe default: "value")
	ReportFreq time.Duration // how often pending usage is flushed
	BaseURL    string        // override for tests; defaults to https://api.stripe.com
	HTTPClient *http.Client
}

// StripeReporter batches per-customer tool-call usage and flushes it to Stripe's
// Billing Meter Events API on an interval, matching usage-based ("metered")
// pricing. Batching keeps the hot path off the network.
type StripeReporter struct {
	cfg     StripeConfig
	client  *http.Client
	mu      sync.Mutex
	pending map[string]int
}

func NewStripeReporter(cfg StripeConfig) (*StripeReporter, error) {
	if cfg.SecretKey == "" || cfg.MeterEvent == "" {
		return nil, errors.New("stripe billing requires STRIPE_SECRET_KEY and STRIPE_METER_EVENT")
	}
	if cfg.ValueKey == "" {
		cfg.ValueKey = "value"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.stripe.com"
	}
	if cfg.ReportFreq <= 0 {
		cfg.ReportFreq = 60 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &StripeReporter{cfg: cfg, client: client, pending: map[string]int{}}, nil
}

// Add records n calls for a Stripe customer reference. Non-blocking.
func (s *StripeReporter) Add(customer string, n int) {
	if customer == "" || n <= 0 {
		return
	}
	s.mu.Lock()
	s.pending[customer] += n
	s.mu.Unlock()
}

// Flush reports all pending usage to Stripe. On failure the unsent counts are
// restored so they are retried on the next flush (no silent revenue loss).
func (s *StripeReporter) Flush(ctx context.Context) error {
	s.mu.Lock()
	batch := s.pending
	s.pending = map[string]int{}
	s.mu.Unlock()
	if len(batch) == 0 {
		return nil
	}

	var failed []string
	for customer, count := range batch {
		if err := s.report(ctx, customer, count); err != nil {
			s.mu.Lock()
			s.pending[customer] += count
			s.mu.Unlock()
			failed = append(failed, fmt.Sprintf("%s: %v", customer, err))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("stripe usage report failed for %d customer(s): %s", len(failed), strings.Join(failed, "; "))
	}
	return nil
}

// Run flushes on cfg.ReportFreq until ctx is cancelled, then flushes once more.
func (s *StripeReporter) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.ReportFreq)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = s.Flush(flushCtx)
			cancel()
			return
		case <-ticker.C:
			_ = s.Flush(ctx)
		}
	}
}

func (s *StripeReporter) report(ctx context.Context, customer string, count int) error {
	form := url.Values{}
	form.Set("event_name", s.cfg.MeterEvent)
	form.Set("payload[stripe_customer_id]", customer)
	form.Set(fmt.Sprintf("payload[%s]", s.cfg.ValueKey), strconv.Itoa(count))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/v1/billing/meter_events", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.cfg.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return fmt.Errorf("stripe status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// StripeMeteringStore decorates a metering.Store so that every successful tool
// call is both counted locally (for quota) and queued for Stripe usage billing.
// The metering key is treated as the Stripe customer reference.
type StripeMeteringStore struct {
	base     metering.Store
	reporter *StripeReporter
}

func NewStripeMeteringStore(base metering.Store, reporter *StripeReporter) *StripeMeteringStore {
	return &StripeMeteringStore{base: base, reporter: reporter}
}

func (s *StripeMeteringStore) Increment(ctx context.Context, key string) error {
	if err := s.base.Increment(ctx, key); err != nil {
		return err
	}
	s.reporter.Add(key, 1)
	return nil
}

func (s *StripeMeteringStore) Count(ctx context.Context, key string) (int, error) {
	return s.base.Count(ctx, key)
}
