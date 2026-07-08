# Connect BindKit to real services

An MCP tool is usually a small adapter. The agent asks for something, BindKit
validates and routes the call, and your tool talks to the system that actually
has the data.

```text
MCP client -> BindKit -> your tool -> API, database, queue, or SaaS service
```

Keep that adapter boring. Boring is good here.

## What goes in a tool

A useful production tool usually does five things:

1. Reads typed arguments from the MCP call.
2. Validates them before touching an external service.
3. Calls one service with a timeout.
4. Returns a small answer the agent can use.
5. Avoids returning secrets, raw tokens, stack traces, or huge payloads.

The service credentials should come from environment variables, not from MCP
arguments. Agents should never pass API keys into tool calls.

## Example: call an internal HTTP API

Start with the scaffold:

```bash
make new-tool name=customer_lookup
```

Keep the generated `Register` function. Replace the generated `run` function
and add the imports/types your service call needs:

```go
package customer_lookup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bindfort/bindkit/internal/mcp"
)

type input struct {
	CustomerID string `json:"customer_id"`
}

type customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Plan  string `json:"plan"`
	Usage int    `json:"usage"`
}

func run(ctx context.Context, call mcp.CallRequest) (mcp.CallResult, error) {
	var in input
	raw, _ := json.Marshal(call.Arguments)
	if err := json.Unmarshal(raw, &in); err != nil {
		return mcp.CallResult{}, fmt.Errorf("invalid arguments")
	}
	if in.CustomerID == "" {
		return mcp.CallResult{}, fmt.Errorf("customer_id is required")
	}

	baseURL := os.Getenv("CUSTOMER_API_URL")
	token := os.Getenv("CUSTOMER_API_TOKEN")
	if baseURL == "" || token == "" {
		return mcp.CallResult{}, fmt.Errorf("customer service is not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/customers/"+in.CustomerID, nil)
	if err != nil {
		return mcp.CallResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mcp.CallResult{}, fmt.Errorf("customer service unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return mcp.CallResult{}, fmt.Errorf("customer not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mcp.CallResult{}, fmt.Errorf("customer service returned %d", resp.StatusCode)
	}

	var c customer
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return mcp.CallResult{}, fmt.Errorf("invalid customer response")
	}

	text := fmt.Sprintf("%s is on the %s plan with %d calls used.", c.Name, c.Plan, c.Usage)
	return mcp.CallResult{Content: []mcp.Content{{Type: "text", Text: text}}}, nil
}
```

Then run the server with service credentials:

```bash
CUSTOMER_API_URL=https://api.example.com \
CUSTOMER_API_TOKEN=replace-me \
BINDKIT_TRANSPORT=http \
go run ./cmd/server
```

Call the tool:

```bash
curl -s http://127.0.0.1:8080/mcp \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"customer_lookup.run","arguments":{"customer_id":"cus_123"}}}'
```

## Common service patterns

For a SaaS API, keep the API token in an environment variable and return only
the fields the agent needs.

For a database, put the connection string in an environment variable, use
parameterized queries, and set short query timeouts.

For an internal workflow, make the tool start one clear action: create a ticket,
look up an invoice, check an order, summarize a runbook, or enqueue a job.

For paid access, combine service credentials with BindKit auth and quotas. Your
customer calls BindKit with their API key; BindKit checks the plan; your tool
uses your server-side service token to call the upstream system.

## What not to expose

Do not expose broad tools like `run_sql`, `fetch_url`, or `call_api` without
strict allowlists. They are flexible, but they are also easy to abuse.

Prefer narrow tools:

```text
customer_lookup.run
invoice_status.run
github_issue_create.run
deployment_status.run
```

Small tools are easier to test, meter, price, and secure.
