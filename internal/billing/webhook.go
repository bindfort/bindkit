package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidSignature = errors.New("invalid stripe webhook signature")

// WebhookEvent is the minimal shape of a Stripe event the kit reacts to.
type WebhookEvent struct {
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// RevenueAffecting reports whether the event should trigger access revocation.
func (e WebhookEvent) RevenueAffecting() bool {
	switch e.Type {
	case "invoice.payment_failed", "customer.subscription.deleted", "customer.subscription.paused":
		return true
	default:
		return false
	}
}

// VerifyWebhook authenticates a Stripe webhook payload against the signing
// secret using the documented t=/v1= HMAC-SHA256 scheme and a timestamp
// tolerance to stop replay. now is injectable for testing.
func VerifyWebhook(payload []byte, sigHeader, secret string, tolerance time.Duration, now time.Time) (WebhookEvent, error) {
	var evt WebhookEvent
	ts, sigs := parseSignatureHeader(sigHeader)
	if ts == 0 || len(sigs) == 0 {
		return evt, ErrInvalidSignature
	}
	if tolerance > 0 {
		if delta := now.Sub(time.Unix(ts, 0)); delta > tolerance || delta < -tolerance {
			return evt, fmt.Errorf("%w: timestamp outside tolerance", ErrInvalidSignature)
		}
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := mac.Sum(nil)

	matched := false
	for _, sig := range sigs {
		raw, err := hex.DecodeString(sig)
		if err != nil {
			continue
		}
		if hmac.Equal(raw, expected) {
			matched = true
			break
		}
	}
	if !matched {
		return evt, ErrInvalidSignature
	}
	if err := json.Unmarshal(payload, &evt); err != nil {
		return evt, fmt.Errorf("decode event: %w", err)
	}
	return evt, nil
}

func parseSignatureHeader(header string) (int64, []string) {
	var ts int64
	var v1 []string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts, _ = strconv.ParseInt(kv[1], 10, 64)
		case "v1":
			v1 = append(v1, kv[1])
		}
	}
	return ts, v1
}

// WebhookHandler returns an http.Handler that verifies the Stripe signature and
// invokes onEvent for revenue-affecting events so the caller can revoke the
// customer's access. It answers 400 on a bad signature, 200 otherwise.
func WebhookHandler(secret string, onEvent func(WebhookEvent)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		evt, err := VerifyWebhook(body, r.Header.Get("Stripe-Signature"), secret, 5*time.Minute, time.Now())
		if err != nil {
			http.Error(w, "invalid signature", http.StatusBadRequest)
			return
		}
		if onEvent != nil && evt.RevenueAffecting() {
			onEvent(evt)
		}
		w.WriteHeader(http.StatusOK)
	})
}
