# Pricing Hooks

Bindkit keeps pricing intentionally simple for v1.

- `BINDKIT_AUTH_ENABLED=true` turns API-key checks on.
- `BINDKIT_API_KEYS=key1:free,key2:pro` maps keys to plans.
- `BINDKIT_BILLING_ENABLED=true` turns quota checks on.
- `BINDKIT_PLAN_QUOTAS=free:100,pro:10000` sets monthly-style usage limits.

Stripe is deliberately left as an adapter seam. Replace the static key map with
your customer subscription lookup, then keep the same `Principal{Key, Plan}`
contract for rate limiting, billing, and metering.

