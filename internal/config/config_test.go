package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Transport != "stdio" || cfg.HTTPAddr != ":8080" || cfg.RatePerMin != 60 {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}

func TestLoadReportsMissingKeysWhenAuthEnabled(t *testing.T) {
	t.Setenv("BINDKIT_AUTH_ENABLED", "true")
	t.Setenv("BINDKIT_API_KEYS", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadReportsMultipleInvalidValues(t *testing.T) {
	t.Setenv("BINDKIT_TRANSPORT", "bad")
	t.Setenv("BINDKIT_METERING", "redis")
	t.Setenv("BINDKIT_RATE_PER_MIN", "nope")
	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, part := range []string{"BINDKIT_TRANSPORT", "BINDKIT_METERING", "BINDKIT_RATE_PER_MIN"} {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("expected %s in error, got %q", part, err.Error())
		}
	}
}

func TestLoadParsesKeysAndQuotas(t *testing.T) {
	t.Setenv("BINDKIT_AUTH_ENABLED", "true")
	t.Setenv("BINDKIT_API_KEYS", "abc:pro,def:free")
	t.Setenv("BINDKIT_PLAN_QUOTAS", "free:3,pro:9")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKeys["abc"] != "pro" || cfg.PlanQuotas["pro"] != 9 {
		t.Fatalf("bad parsed config: %#v", cfg)
	}
}
