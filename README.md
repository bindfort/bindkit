# Bindkit

**Ship a monetizable MCP server this weekend.** Bindkit is a production-shaped Go
starter kit for paid [MCP](https://modelcontextprotocol.io) servers — the auth,
billing, metering, rate limiting, transport, logging, Docker, and CI are already
wired. You write the tools.

```bash
go test ./...                          # everything is tested
BINDKIT_TRANSPORT=http go run ./cmd/server
```

## What's in the box

| Concern | Implementation |
|---|---|
| **MCP core** | JSON-RPC dispatch, tool registry, `initialize` / `tools/list` / `tools/call` |
| **Transports** | stdio (desktop clients) + HTTP `/mcp`, with **streamable HTTP (SSE)** on `Accept: text/event-stream` |
| **Auth** | static API keys **or** OAuth 2.1 bearer (JWT validated against your provider's JWKS) |
| **Billing** | Stripe usage-based metering (batched meter events) + webhook signature verification |
| **Rate limiting** | per-key token bucket with idle-bucket eviction |
| **Metering** | per-key counters (in-memory; swap the `Store` for Redis) |
| **Config** | typed env with loud, fail-fast validation |
| **Logging** | `slog` JSON with a secret-redaction helper |
| **Packaging** | distroless Docker (~8 MB), compose, Fly config, GitHub Actions CI |
| **Tools** | `url.check` (a real, SSRF-guarded endpoint auditor) + `weather.current` demo |

## Quickstart

Run over HTTP and list tools:

```bash
BINDKIT_TRANSPORT=http go run ./cmd/server
curl -s http://127.0.0.1:8080/mcp -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Call the real tool:

```bash
curl -s http://127.0.0.1:8080/mcp -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"url.check","arguments":{"url":"https://example.com"}}}'
```

Turn on auth, rate limits, and quota:

```bash
BINDKIT_TRANSPORT=http BINDKIT_AUTH_ENABLED=true \
BINDKIT_API_KEYS=dev-key:pro BINDKIT_BILLING_ENABLED=true \
BINDKIT_PLAN_QUOTAS=free:100,pro:100000 \
go run ./cmd/server
```

## Add your own tool

```bash
make new-tool name=invoice_lookup     # or scripts/new_tool.ps1 -Name invoice_lookup
```

Scaffolds a typed handler, JSON schema, test file, and registry entry. Register
it in `cmd/server/main.go`.

## Going to production

The full env reference (OAuth provider, Stripe meter + webhook, deployment) and
the launch + fulfillment runbook are in **[docs/go-live.md](docs/go-live.md)**.
Key environment variables:

```bash
BINDKIT_AUTH_MODE=oauth                 # or "static"
BINDKIT_OAUTH_ISSUER=...  BINDKIT_OAUTH_JWKS_URL=...  BINDKIT_OAUTH_AUDIENCE=...
STRIPE_SECRET_KEY=sk_...  STRIPE_METER_EVENT=tool_call  STRIPE_WEBHOOK_SECRET=whsec_...
```

Build the image:

```bash
docker build -t bindkit .               # distroless, ~8 MB
```

## Layout

```
cmd/server/        entrypoint + graceful shutdown
internal/
  auth/            static keys + OAuth 2.1 (JWKS) authenticators
  billing/         quota gate, Stripe meter reporter, webhook verification
  config/          typed env + validation
  logging/         slog redaction helper
  mcp/             JSON-RPC, registry, dispatcher
  metering/        counter Store (memory; Redis-ready)
  ratelimit/       token bucket + eviction
  server/          stdio + HTTP transports, middleware chain
tools/             url_check (real) + example_weather (demo)
docs/              clients, pricing, deploy, testing, go-live
```

## License

Commercial — see [LICENSE.md](LICENSE.md). You keep 100% of what you build and
sell with it; you may not resell the kit itself.
