# Public launch checklist

Use this when preparing a public open-source announcement.

## Repository readiness

- [ ] README explains the project in the first screen.
- [ ] License badge and Apache-2.0 license are visible.
- [ ] `go test ./...` passes locally and in CI.
- [ ] No compiled binaries are tracked.
- [ ] No logs, secrets, private notes, or customer references are tracked.
- [ ] Issues and discussions are enabled if maintainers want community input.
- [ ] Security reporting path is visible.

## Suggested positioning

Use a concrete claim:

> Open-source Go starter kit for production-shaped MCP servers.

Avoid vague platform language. BindKit is strongest when framed as the missing
server plumbing for teams that already know what MCP tool they want to build.

## Suggested social copy

```text
We open-sourced BindKit.

It is a Go starter kit for production-shaped MCP servers:
- stdio + HTTP transports
- streamable HTTP
- static API key or OAuth/JWKS auth
- metering, quotas, rate limits
- Stripe usage hooks
- Docker + CI
- SSRF-guarded url.check example tool

Apache-2.0.
```

## First issues worth opening

- Redis-backed metering store.
- Postgres-backed API key and plan lookup.
- More MCP client configuration examples.
- Helm chart.
- Additional safe example tools.
- OpenTelemetry tracing.
