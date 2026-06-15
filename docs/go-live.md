# Bindkit go-live runbook

Everything you need to take Bindkit from "code on disk" to "selling at bindkit.dev."

## What is now real in the code

These were the claims-vs-reality gaps; they are now implemented and tested
(`go test ./...` is green):

| Claim | Where it lives |
|---|---|
| OAuth 2.1 bearer auth | `internal/auth/oauth.go` — JWT validation against the provider JWKS (issuer, audience, expiry, signature). RS256/384/512. |
| Stripe metered billing | `internal/billing/stripe.go` — batched usage reported to Stripe Billing Meter Events. |
| Stripe webhook reconciliation | `internal/billing/webhook.go` — HMAC-SHA256 signature verification + revenue-event hook to revoke access. |
| Streamable HTTP | `internal/server/server.go` — `/mcp` returns SSE when the client sends `Accept: text/event-stream`, JSON otherwise. |
| Real B2B tool | `tools/url_check/` — `url.check` (status, latency, security headers, SSRF-guarded). |
| <20 MB distroless image | `Dockerfile` — stripped static binary (~6 MB) on `gcr.io/distroless/static` (~8 MB total). |
| Per-key/per-tool rate limits | `internal/ratelimit/` — token bucket with idle-bucket eviction. |
| Loud config validation | `internal/config/` — fails at startup on misconfig. |

## 1. Configure the server (env)

```bash
BINDKIT_TRANSPORT=http
BINDKIT_HTTP_ADDR=:8080
BINDKIT_AUTH_ENABLED=true

# --- Auth: static keys OR oauth ---
BINDKIT_AUTH_MODE=oauth            # or "static"
BINDKIT_OAUTH_ISSUER=https://YOUR_TENANT.auth0.com/
BINDKIT_OAUTH_JWKS_URL=https://YOUR_TENANT.auth0.com/.well-known/jwks.json
BINDKIT_OAUTH_AUDIENCE=bindkit-api
BINDKIT_OAUTH_PLAN_CLAIM=plan      # claim that carries the plan name
# (static mode instead:) BINDKIT_API_KEYS=key1:pro,key2:free

# --- Billing (Stripe usage-based) ---
BINDKIT_BILLING_ENABLED=true
STRIPE_SECRET_KEY=sk_live_xxx
STRIPE_METER_EVENT=tool_call       # the meter's event_name in Stripe
STRIPE_WEBHOOK_SECRET=whsec_xxx
BINDKIT_STRIPE_REPORT_EVERY=60     # seconds between usage flushes

BINDKIT_RATE_PER_MIN=120
BINDKIT_PLAN_QUOTAS=free:100,pro:100000
```

Map the OAuth `sub` (or static key) to the **Stripe customer id** so usage is
billed to the right customer. Point the Stripe webhook at `POST /stripe/webhook`.

## 2. Set up Lemon Squeezy (payments + VAT + fulfillment)

Lemon Squeezy is the Merchant of Record — it handles EU VAT, invoices, license
keys, and the download email, so you don't.

1. Create a store, then two **products/variants**: **Solo $129**, **Team $349**.
2. Turn on **license keys** (1 activation for Solo, 5 for Team) if you want to
   enforce seats; enable the **file download** of the kit zip (see step 4) as the
   fulfillment asset.
3. Copy each variant's **checkout URL** (`https://YOURSTORE.lemonsqueezy.com/buy/<id>`).
4. Wire them into the landing page via env at build time:
   ```
   NEXT_PUBLIC_BINDKIT_SOLO_CHECKOUT_URL=https://YOURSTORE.lemonsqueezy.com/buy/<solo-id>
   NEXT_PUBLIC_BINDKIT_TEAM_CHECKOUT_URL=https://YOURSTORE.lemonsqueezy.com/buy/<team-id>
   ```
   (Or replace the placeholders at the top of `src/app/bindkit/page.tsx`.)

## 3. Package the deliverable

```bash
# from the bindkit/ repo root
pwsh scripts/package.ps1      # Windows  -> dist/bindkit-<version>.zip
./scripts/package.sh          # mac/linux/git-bash
```

The script runs the tests, then zips the source **excluding** internal files
(`handoff.json`, `agent.md`, `tasks.md`, logs, build output, the landing page).
Upload `dist/bindkit-<version>.zip` as the Lemon Squeezy download asset.

## 4. Publish the landing page (bindkit.dev)

The page is the `/bindkit` route in the Next app. To serve it at **bindkit.dev**:

- Point `bindkit.dev` DNS at your host (the metadata/canonical already use it).
- Build & deploy with the checkout env vars set:
  ```
  NEXT_PUBLIC_BINDKIT_SOLO_CHECKOUT_URL=... NEXT_PUBLIC_BINDKIT_TEAM_CHECKOUT_URL=... npm run build
  ```
- (Optional) Replace the SVG OG image with a 1200×630 PNG for better social cards.

## 5. Pre-launch checklist

- [x] Claims match the code (OAuth, Stripe, streamable HTTP, distroless, real tool)
- [x] LICENSE.md present (commercial)
- [x] Real B2B tool (`url.check`) shipping alongside the demo
- [ ] Lemon Squeezy products created + checkout URLs wired
- [ ] Stripe meter + webhook configured with live keys
- [ ] OAuth provider (Auth0/Okta/Clerk/...) tenant + audience created
- [ ] `dist/bindkit-<version>.zip` uploaded as the download asset
- [ ] bindkit.dev DNS + deploy
- [ ] Have a lawyer glance at LICENSE.md

## 6. Then post it

Once the boxes above are checked: **Show HN**, r/golang, r/mcp + the MCP Discord,
a launch post on dev.to/X, Product Hunt, and the MCP server registries. The pitch
that converts: "ship a *paid* MCP server this weekend — auth, Stripe billing,
metering, rate limits, Docker, CI already done."
