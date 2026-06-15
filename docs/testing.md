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

On the current Windows development machine, the race run is blocked because Go cannot find `gcc` in `PATH`.

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

