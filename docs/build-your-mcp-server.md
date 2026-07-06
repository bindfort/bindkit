# Build your MCP server in 5 steps

This path gets you from a fresh clone to a running MCP server with your own
tool. The examples use HTTP because it is easy to test with `curl`; the same
tool registry also works over stdio.

## 1. Run the server

```bash
git clone https://github.com/bindfort/bindkit.git
cd bindkit
go test ./...
BINDKIT_TRANSPORT=http go run ./cmd/server
```

On Windows PowerShell:

```powershell
.\run-local.ps1
```

## 2. Check that MCP works

List the bundled tools:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

Call the SSRF-guarded URL checker:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"url.check","arguments":{"url":"https://example.com"}}}'
```

## 3. Scaffold your first tool

```bash
make new-tool name=invoice_lookup
```

On Windows PowerShell:

```powershell
pwsh scripts/new_tool.ps1 -Name invoice_lookup
```

This creates:

```text
tools/invoice_lookup/invoice_lookup.go
tools/invoice_lookup/invoice_lookup_test.go
```

The scaffold includes:

- an MCP tool name, `invoice_lookup.run`;
- a typed handler function;
- a JSON input schema placeholder;
- a focused test.

## 4. Register the tool

Open `cmd/server/main.go`, add the import:

```go
invoice_lookup "github.com/bindfort/bindkit/tools/invoice_lookup"
```

Then register it:

```go
for _, register := range []func(*mcp.Registry) error{
	urlcheck.Register,
	example_weather.Register,
	invoice_lookup.Register,
} {
	// existing error handling
}
```

Run tests:

```bash
go test ./...
```

Start the server and call your tool:

```bash
BINDKIT_TRANSPORT=http go run ./cmd/server

curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"invoice_lookup.run","arguments":{}}}'
```

## 5. Add production controls

Enable API-key auth:

```bash
BINDKIT_TRANSPORT=http \
BINDKIT_AUTH_ENABLED=true \
BINDKIT_API_KEYS=dev-key:free \
go run ./cmd/server
```

Authenticated call:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "authorization: Bearer dev-key" \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"invoice_lookup.run","arguments":{}}}'
```

Enable quotas:

```bash
BINDKIT_TRANSPORT=http \
BINDKIT_AUTH_ENABLED=true \
BINDKIT_API_KEYS=dev-key:free \
BINDKIT_BILLING_ENABLED=true \
BINDKIT_PLAN_QUOTAS=free:100,pro:100000 \
go run ./cmd/server
```

From there, replace the in-memory metering store with Redis or another backend,
wire OAuth/JWKS if you need delegated identity, and connect Stripe meter events
if you charge by usage.
