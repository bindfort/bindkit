package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bindfort/bindkit/internal/auth"
	"github.com/bindfort/bindkit/internal/billing"
	"github.com/bindfort/bindkit/internal/config"
	"github.com/bindfort/bindkit/internal/mcp"
	"github.com/bindfort/bindkit/internal/metering"
	"github.com/bindfort/bindkit/internal/ratelimit"
	"github.com/bindfort/bindkit/internal/server"
	example_weather "github.com/bindfort/bindkit/tools/example_weather"
	urlcheck "github.com/bindfort/bindkit/tools/url_check"
)

const version = "0.1.0"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("config invalid", "error", err)
		os.Exit(1)
	}

	registry := mcp.NewRegistry()
	for _, register := range []func(*mcp.Registry) error{
		urlcheck.Register,        // real B2B tool
		example_weather.Register, // demo tool
	} {
		if err := register(registry); err != nil {
			logger.Error("register tools", "error", err)
			os.Exit(1)
		}
	}

	authenticator, err := buildAuthenticator(cfg)
	if err != nil {
		logger.Error("authenticator init", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var meteringStore metering.Store = metering.NewMemoryStore()
	var webhook http.Handler
	if cfg.StripeSecretKey != "" {
		reporter, err := billing.NewStripeReporter(billing.StripeConfig{
			SecretKey:  cfg.StripeSecretKey,
			MeterEvent: cfg.StripeMeterEvent,
			ValueKey:   cfg.StripeMeterValueKey,
			ReportFreq: time.Duration(cfg.StripeReportEvery) * time.Second,
		})
		if err != nil {
			logger.Error("stripe init", "error", err)
			os.Exit(1)
		}
		meteringStore = billing.NewStripeMeteringStore(meteringStore, reporter)
		go reporter.Run(ctx)
		if cfg.StripeWebhookSecret != "" {
			webhook = billing.WebhookHandler(cfg.StripeWebhookSecret, func(evt billing.WebhookEvent) {
				// Revenue-affecting event: wire your revocable key/customer store here.
				logger.Warn("stripe revenue event", "type", evt.Type)
			})
		}
	}

	dispatcher := mcp.NewDispatcher(registry, "bindkit", version)
	handler := server.BuildHandler(dispatcher, server.Options{
		AuthEnabled:    cfg.AuthEnabled,
		BillingEnabled: cfg.BillingEnabled,
		Authenticator:  authenticator,
		Limiter:        ratelimit.New(cfg.RatePerMin),
		Metering:       meteringStore,
		Quota:          billing.NewQuotaChecker(meteringStore, cfg.PlanQuotas),
	})

	logger.Info("bindkit starting", "version", version, "config", cfg.String())

	switch cfg.Transport {
	case "stdio":
		if err := server.ServeStdio(ctx, handler, os.Stdin, os.Stdout); err != nil && ctx.Err() == nil {
			logger.Error("stdio failed", "error", err)
			os.Exit(1)
		}
	case "http":
		httpServer := &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           server.HTTPHandler(handler, webhook),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = httpServer.Shutdown(shutdownCtx)
		}()
		if err := httpServer.ListenAndServe(); err != nil && !server.IsClosedServer(err) {
			logger.Error("http failed", "error", err)
			os.Exit(1)
		}
	}
}

func buildAuthenticator(cfg config.Config) (auth.Authenticator, error) {
	if cfg.AuthMode == "oauth" {
		return auth.NewOAuthAuthenticator(auth.OAuthConfig{
			Issuer:    cfg.OAuthIssuer,
			Audience:  cfg.OAuthAudience,
			JWKSURL:   cfg.OAuthJWKSURL,
			PlanClaim: cfg.OAuthPlanClaim,
		})
	}
	return auth.NewStaticAuthenticator(cfg.APIKeys), nil
}
