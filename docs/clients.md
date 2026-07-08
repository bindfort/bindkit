# Clients

BindKit supports two MCP transport shapes:

- `stdio` for desktop clients that launch the server as a child process;
- HTTP `/mcp` for hosted or locally reachable servers.

## stdio

Use stdio when an MCP client starts BindKit directly.

```bash
BINDKIT_TRANSPORT=stdio go run ./cmd/server
```

PowerShell:

```powershell
$env:BINDKIT_TRANSPORT = "stdio"
go run ./cmd/server
```

The server reads one JSON-RPC request per line from stdin and writes one
JSON-RPC response per line to stdout.

Manual smoke test:

```bash
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"weather.current","arguments":{"city":"Krakow"}}}' \
  | BINDKIT_TRANSPORT=stdio go run ./cmd/server
```

Expected result:

- `initialize` returns server info for `bindkit`;
- `tools/list` includes `url.check` and `weather.current`;
- the weather tool returns a deterministic demo response.

## HTTP

Use HTTP when hosting the MCP server.

```bash
BINDKIT_TRANSPORT=http BINDKIT_HTTP_ADDR=:8080 go run ./cmd/server
```

PowerShell:

```powershell
$env:BINDKIT_TRANSPORT = "http"
$env:BINDKIT_HTTP_ADDR = "127.0.0.1:8080"
go run ./cmd/server
```

Health check:

```bash
curl -s http://127.0.0.1:8080/healthz
```

List tools:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Call a tool:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"weather.current","arguments":{"city":"Warsaw"}}}'
```

## Streamable HTTP

If the client sends `Accept: text/event-stream`, BindKit returns the MCP response
as a Server-Sent Event.

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "accept: text/event-stream" \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Expected result starts with:

```text
event: message
data: {"jsonrpc":"2.0", ...}
```

## Authenticated tool calls

`initialize` and `tools/list` remain public. `tools/call` is gated when auth is
enabled.

```bash
BINDKIT_TRANSPORT=http \
BINDKIT_AUTH_ENABLED=true \
BINDKIT_API_KEYS=dev-key:free \
go run ./cmd/server
```

Unauthenticated call:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"weather.current","arguments":{"city":"Warsaw"}}}'
```

Authenticated call:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "authorization: Bearer dev-key" \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"weather.current","arguments":{"city":"Warsaw"}}}'
```

## Quota check

To verify quotas, set a tiny plan limit:

```bash
BINDKIT_TRANSPORT=http \
BINDKIT_AUTH_ENABLED=true \
BINDKIT_API_KEYS=dev-key:free \
BINDKIT_BILLING_ENABLED=true \
BINDKIT_PLAN_QUOTAS=free:1 \
go run ./cmd/server
```

The first authenticated `tools/call` succeeds. The second call with the same key
returns `quota exceeded`.
