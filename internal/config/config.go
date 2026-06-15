package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Transport            string
	HTTPAddr             string
	AuthEnabled          bool
	AuthMode             string // "static" | "oauth"
	Metering             string
	BillingEnabled       bool
	RatePerMin           int
	APIKeys              map[string]string
	PlanQuotas           map[string]int

	// OAuth 2.1 resource server (used when AuthMode == "oauth").
	OAuthIssuer    string
	OAuthAudience  string
	OAuthJWKSURL   string
	OAuthPlanClaim string

	// Stripe usage-based billing (optional; engaged when StripeSecretKey is set).
	StripeSecretKey     string
	StripeMeterEvent    string
	StripeMeterValueKey string
	StripeWebhookSecret string
	StripeReportEvery   int // seconds
}

func Load() (Config, error) {
	cfg := Config{
		Transport:            env("BINDKIT_TRANSPORT", "stdio"),
		HTTPAddr:             env("BINDKIT_HTTP_ADDR", ":8080"),
		AuthEnabled:          boolEnv("BINDKIT_AUTH_ENABLED", false),
		AuthMode:             env("BINDKIT_AUTH_MODE", "static"),
		Metering:             env("BINDKIT_METERING", "memory"),
		BillingEnabled:       boolEnv("BINDKIT_BILLING_ENABLED", false),
		RatePerMin:           intEnv("BINDKIT_RATE_PER_MIN", 60),
		APIKeys:              parseAPIKeys(os.Getenv("BINDKIT_API_KEYS")),
		PlanQuotas:           parseQuotas(os.Getenv("BINDKIT_PLAN_QUOTAS")),
		OAuthIssuer:          env("BINDKIT_OAUTH_ISSUER", ""),
		OAuthAudience:        env("BINDKIT_OAUTH_AUDIENCE", ""),
		OAuthJWKSURL:         env("BINDKIT_OAUTH_JWKS_URL", ""),
		OAuthPlanClaim:       env("BINDKIT_OAUTH_PLAN_CLAIM", "plan"),
		StripeSecretKey:      env("STRIPE_SECRET_KEY", ""),
		StripeMeterEvent:     env("STRIPE_METER_EVENT", ""),
		StripeMeterValueKey:  env("STRIPE_METER_VALUE_KEY", "value"),
		StripeWebhookSecret:  env("STRIPE_WEBHOOK_SECRET", ""),
		StripeReportEvery:    intEnv("BINDKIT_STRIPE_REPORT_EVERY", 60),
	}
	var problems []string
	if cfg.Transport != "stdio" && cfg.Transport != "http" {
		problems = append(problems, "BINDKIT_TRANSPORT must be stdio or http")
	}
	if cfg.Metering != "memory" {
		problems = append(problems, "BINDKIT_METERING currently supports memory only")
	}
	if cfg.RatePerMin <= 0 {
		problems = append(problems, "BINDKIT_RATE_PER_MIN must be a positive integer")
	}
	if cfg.AuthMode != "static" && cfg.AuthMode != "oauth" {
		problems = append(problems, "BINDKIT_AUTH_MODE must be static or oauth")
	}
	if cfg.AuthEnabled && cfg.AuthMode == "static" && len(cfg.APIKeys) == 0 {
		problems = append(problems, "BINDKIT_API_KEYS is required when auth is enabled in static mode")
	}
	if cfg.AuthEnabled && cfg.AuthMode == "oauth" && (cfg.OAuthIssuer == "" || cfg.OAuthJWKSURL == "" || cfg.OAuthAudience == "") {
		problems = append(problems, "BINDKIT_OAUTH_ISSUER, BINDKIT_OAUTH_JWKS_URL, and BINDKIT_OAUTH_AUDIENCE are required when BINDKIT_AUTH_MODE=oauth")
	}
	if cfg.BillingEnabled && !cfg.AuthEnabled {
		problems = append(problems, "BINDKIT_BILLING_ENABLED requires BINDKIT_AUTH_ENABLED (quotas are keyed to an authenticated principal)")
	}
	if cfg.StripeSecretKey != "" && cfg.StripeMeterEvent == "" {
		problems = append(problems, "STRIPE_METER_EVENT is required when STRIPE_SECRET_KEY is set")
	}
	if len(problems) > 0 {
		return cfg, errors.New(strings.Join(problems, "; "))
	}
	return cfg, nil
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}

func parseAPIKeys(raw string) map[string]string {
	out := map[string]string{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		key := strings.TrimSpace(parts[0])
		plan := "free"
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			plan = strings.TrimSpace(parts[1])
		}
		out[key] = plan
	}
	return out
}

func parseQuotas(raw string) map[string]int {
	out := map[string]int{"free": 100, "pro": 10000}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}
		quota, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}
		out[strings.TrimSpace(parts[0])] = quota
	}
	return out
}

func (c Config) String() string {
	stripe := c.StripeSecretKey != ""
	return fmt.Sprintf("transport=%s http=%s auth=%t(%s) metering=%s billing=%t stripe=%t rate=%d/min",
		c.Transport, c.HTTPAddr, c.AuthEnabled, c.AuthMode, c.Metering, c.BillingEnabled, stripe, c.RatePerMin)
}
