# Testing

Run the standard suite:

```bash
go test ./...
```

Run coverage:

```bash
go test -cover ./...
```

Run the race detector when the local machine has CGO and a C compiler installed:

```bash
CGO_ENABLED=1 go test -race ./...
```

On Windows, install a C toolchain and make sure `gcc` is in `PATH` before using
the race detector.

## End-to-end smoke checks

After `go test ./...`, run the server over HTTP:

```bash
BINDKIT_TRANSPORT=http BINDKIT_HTTP_ADDR=:8080 go run ./cmd/server
```

Then verify:

- `GET /healthz` returns `ok`.
- `GET /mcp` returns `405 Method Not Allowed`.
- `initialize` returns server info.
- `tools/list` includes `url.check` and `weather.current`.
- `weather.current` returns a deterministic demo result.
- `url.check` can check `https://example.com`.
- `url.check` blocks loopback/private targets unless explicitly allowed.
- `Accept: text/event-stream` returns an SSE response.

For auth and quotas, run:

```bash
BINDKIT_TRANSPORT=http \
BINDKIT_AUTH_ENABLED=true \
BINDKIT_API_KEYS=dev-key:free \
BINDKIT_BILLING_ENABLED=true \
BINDKIT_PLAN_QUOTAS=free:1 \
go run ./cmd/server
```

Expected behavior:

- `tools/list` stays public.
- unauthenticated `tools/call` fails.
- `Authorization: Bearer dev-key` allows the first call.
- the second authenticated call returns `quota exceeded`.

## Covered Areas

- MCP registry validation, duplicate detection, sorting, and missing-tool errors.
- MCP dispatcher initialize, unknown-method, invalid-params, and tool-error behavior.
- Static API-key auth and principal context handling.
- Billing quota allow/block paths.
- Typed env config defaults, parsing, and aggregated validation errors.
- Secret redaction.
- Memory metering, including concurrent increments.
- Rate limiting by key.
- Middleware ordering for auth, rate, quota, and metering.
- stdio parse-error recovery.
- HTTP health, method rejection, parse errors, and authenticated tool calls.
- Example weather tool behavior.
- `url.check` SSRF protections.
