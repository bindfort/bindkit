# Clients

## stdio

Use stdio when a desktop client launches the server as a child process.

```bash
BINDKIT_TRANSPORT=stdio go run ./cmd/server
```

Send one JSON-RPC request per line.

## HTTP

Use HTTP when hosting the MCP server.

```bash
BINDKIT_TRANSPORT=http BINDKIT_HTTP_ADDR=:8080 go run ./cmd/server
```

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

