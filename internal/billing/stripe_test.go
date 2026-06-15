package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/bindfort/bindkit/internal/metering"
)

func TestStripeReporterBatchesAndReports(t *testing.T) {
	var mu sync.Mutex
	got := map[string]string{} // customer -> value

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/billing/meter_events" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		user, _, _ := r.BasicAuth()
		if user != "sk_test_123" {
			t.Errorf("missing/wrong secret key auth: %q", user)
		}
		_ = r.ParseForm()
		mu.Lock()
		got[r.PostFormValue("payload[stripe_customer_id]")] = r.PostFormValue("payload[value]")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reporter, err := NewStripeReporter(StripeConfig{
		SecretKey:  "sk_test_123",
		MeterEvent: "tool_call",
		BaseURL:    srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	store := NewStripeMeteringStore(metering.NewMemoryStore(), reporter)
	for i := 0; i < 3; i++ {
		_ = store.Increment(context.Background(), "cus_abc")
	}
	_ = store.Increment(context.Background(), "cus_xyz")

	if err := reporter.Flush(context.Background()); err != nil {
		t.Fatalf("flush: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got["cus_abc"] != "3" || got["cus_xyz"] != "1" {
		t.Fatalf("unexpected reported usage: %#v", got)
	}
}

func TestStripeReporterRetriesOnFailure(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	reporter, _ := NewStripeReporter(StripeConfig{SecretKey: "sk", MeterEvent: "m", BaseURL: srv.URL})
	reporter.Add("cus_1", 5)
	if err := reporter.Flush(context.Background()); err == nil {
		t.Fatal("expected error on 500")
	}
	// The failed usage must be retained for retry, not dropped.
	reporter.mu.Lock()
	pending := reporter.pending["cus_1"]
	reporter.mu.Unlock()
	if pending != 5 {
		t.Fatalf("expected usage retained for retry, got %d", pending)
	}
}

func stripeSign(secret string, ts int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func TestVerifyWebhook(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"type":"invoice.payment_failed","data":{"object":{"customer":"cus_1"}}}`)
	now := time.Unix(1_700_000_000, 0)

	evt, err := VerifyWebhook(payload, stripeSign(secret, now.Unix(), payload), secret, 5*time.Minute, now)
	if err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if evt.Type != "invoice.payment_failed" || !evt.RevenueAffecting() {
		t.Fatalf("unexpected event: %+v", evt)
	}

	// Tampered payload must fail.
	if _, err := VerifyWebhook([]byte(`{"type":"x"}`), stripeSign(secret, now.Unix(), payload), secret, 5*time.Minute, now); err == nil {
		t.Fatal("expected signature mismatch on tampered payload")
	}
	// Stale timestamp must fail (replay protection).
	if _, err := VerifyWebhook(payload, stripeSign(secret, now.Add(-time.Hour).Unix(), payload), secret, 5*time.Minute, now); err == nil {
		t.Fatal("expected stale-timestamp rejection")
	}
	// Wrong secret must fail.
	if _, err := VerifyWebhook(payload, stripeSign("whsec_wrong", now.Unix(), payload), secret, 5*time.Minute, now); err == nil {
		t.Fatal("expected wrong-secret rejection")
	}
}
