# Monetization and quota hooks

BindKit is open source. The included billing code is an adapter layer for teams
that want to build paid MCP servers or quota-limited internal services.

## Built-in controls

- `BINDKIT_AUTH_ENABLED=true` turns auth checks on for `tools/call`.
- `BINDKIT_AUTH_MODE=static` uses `BINDKIT_API_KEYS=key1:free,key2:pro`.
- `BINDKIT_AUTH_MODE=oauth` validates bearer tokens against a JWKS endpoint.
- `BINDKIT_BILLING_ENABLED=true` turns quota checks on.
- `BINDKIT_PLAN_QUOTAS=free:100,pro:10000` sets plan limits.
- `STRIPE_SECRET_KEY`, `STRIPE_METER_EVENT`, and `STRIPE_WEBHOOK_SECRET` enable
  Stripe usage reporting and webhook verification.

## What you still own

BindKit does not include a customer database, checkout page, pricing page, or
subscription-management UI. Those should stay in your product.

Typical integration:

1. Authenticate the caller with an API key or OAuth token.
2. Resolve the caller to a plan and customer id.
3. Let BindKit enforce per-plan quotas and rate limits.
4. Report usage through the Stripe meter adapter or replace it with your own
   billing reporter.

The internal contract is intentionally small: authenticated calls carry an
`auth.Principal{Key, Plan}` that rate limiting, quota checks, and metering can
use consistently.
