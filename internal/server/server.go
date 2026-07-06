package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bindfort/bindkit/internal/auth"
	"github.com/bindfort/bindkit/internal/billing"
	"github.com/bindfort/bindkit/internal/mcp"
	"github.com/bindfort/bindkit/internal/metering"
	"github.com/bindfort/bindkit/internal/ratelimit"
)

type Handler func(context.Context, *mcp.Request) *mcp.Response
type Middleware func(Handler) Handler

type Options struct {
	AuthEnabled    bool
	BillingEnabled bool
	Authenticator  auth.Authenticator
	Limiter        *ratelimit.Limiter
	Metering       metering.Store
	Quota          *billing.QuotaChecker
}

type apiKeyContext struct{}

func WithAPIKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, apiKeyContext{}, key)
}

func APIKeyFromContext(ctx context.Context) string {
	value, _ := ctx.Value(apiKeyContext{}).(string)
	return value
}

func Chain(base Handler, middleware ...Middleware) Handler {
	out := base
	for i := len(middleware) - 1; i >= 0; i-- {
		out = middleware[i](out)
	}
	return out
}

func BuildHandler(dispatcher *mcp.Dispatcher, opts Options) Handler {
	middleware := []Middleware{
		AuthMiddleware(opts.AuthEnabled, opts.Authenticator),
		RateLimitMiddleware(opts.Limiter),
		BillingMiddleware(opts.BillingEnabled, opts.Quota),
		MeteringMiddleware(opts.Metering),
	}
	return Chain(dispatcher.Handle, middleware...)
}

func gated(req *mcp.Request) bool {
	return req != nil && req.Method == "tools/call"
}

func AuthMiddleware(enabled bool, authenticator auth.Authenticator) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *mcp.Request) *mcp.Response {
			if !enabled || !gated(req) {
				return next(ctx, req)
			}
			if authenticator == nil {
				return mcp.Failure(req.ID, -32001, auth.ErrUnauthorized.Error())
			}
			key := APIKeyFromContext(ctx)
			if key == "" {
				key = apiKeyFromParams(req.Params)
			}
			principal, err := authenticator.Authenticate(ctx, key)
			if err != nil {
				return mcp.Failure(req.ID, -32001, err.Error())
			}
			return next(auth.WithPrincipal(ctx, principal), req)
		}
	}
}

func RateLimitMiddleware(limiter *ratelimit.Limiter) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *mcp.Request) *mcp.Response {
			if limiter == nil || !gated(req) {
				return next(ctx, req)
			}
			key := "anonymous"
			if principal, ok := auth.PrincipalFromContext(ctx); ok {
				key = principal.Key
			}
			if err := limiter.Allow(ctx, key); err != nil {
				return mcp.Failure(req.ID, -32002, err.Error())
			}
			return next(ctx, req)
		}
	}
}

func BillingMiddleware(enabled bool, quota *billing.QuotaChecker) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *mcp.Request) *mcp.Response {
			if !enabled || quota == nil || !gated(req) {
				return next(ctx, req)
			}
			principal, ok := auth.PrincipalFromContext(ctx)
			if !ok {
				return mcp.Failure(req.ID, -32003, "billing requires an authenticated principal")
			}
			if err := quota.Check(ctx, principal.Plan, principal.Key); err != nil {
				return mcp.Failure(req.ID, -32003, err.Error())
			}
			return next(ctx, req)
		}
	}
}

func MeteringMiddleware(store metering.Store) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *mcp.Request) *mcp.Response {
			response := next(ctx, req)
			if store == nil || !gated(req) || response == nil || response.Error != nil {
				return response
			}
			key := "anonymous"
			if principal, ok := auth.PrincipalFromContext(ctx); ok {
				key = principal.Key
			}
			_ = store.Increment(ctx, key)
			return response
		}
	}
}

func ServeStdio(ctx context.Context, handler Handler, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Allow request lines up to 1 MiB (matching the HTTP body limit). The default
	// 64 KiB Scanner cap would otherwise silently drop larger requests.
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req mcp.Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			if err := encoder.Encode(mcp.Failure(nil, -32700, "parse error")); err != nil {
				return err
			}
			continue
		}
		if err := encoder.Encode(handler(ctx, &req)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func HTTPHandler(handler Handler, webhook ...http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>BindKit local server</title>
  <style>
    body{margin:0;min-height:100vh;display:grid;place-items:center;background:#0b1217;color:#e6edf3;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
    main{width:min(720px,calc(100% - 32px));border:1px solid #1e2a33;border-radius:14px;background:#121a21;padding:28px}
    h1{margin:0 0 8px;font-size:28px}p{color:#96a4b4;line-height:1.6}code{font-family:ui-monospace,Menlo,Consolas,monospace;color:#5ac8e8}.row{display:flex;flex-wrap:wrap;gap:10px;margin-top:20px}
    a{display:inline-flex;padding:10px 14px;border:1px solid #253746;border-radius:8px;color:#e6edf3;text-decoration:none;font-weight:700}.primary{background:#e8a33d;border-color:#e8a33d;color:#151008}
  </style>
</head>
<body>
  <main>
    <h1>BindKit local MCP server</h1>
    <p>This is the local API server. Use <code>POST /mcp</code> for MCP JSON-RPC calls and <code>GET /healthz</code> for health checks.</p>
    <p>Project docs live in the repository README. Use this page only to confirm the local server is running.</p>
    <div class="row">
      <a class="primary" href="/healthz">Health check</a>
      <a href="https://modelcontextprotocol.io" rel="noreferrer">MCP docs</a>
    </div>
  </main>
</body>
</html>`))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var req mcp.Request
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeJSON(w, mcp.Failure(nil, -32700, "parse error"))
			return
		}
		ctx := r.Context()
		if key := bearerToken(r.Header.Get("Authorization")); key != "" {
			ctx = WithAPIKey(ctx, key)
		}
		resp := handler(ctx, &req)
		// Streamable HTTP: when the client accepts an event stream, return the
		// response as Server-Sent Events; otherwise return a single JSON body.
		if acceptsEventStream(r.Header.Get("Accept")) {
			writeEventStream(w, resp)
			return
		}
		writeJSON(w, resp)
	})

	// Mount only the first webhook handler; registering the same pattern twice
	// would panic.
	if len(webhook) > 0 && webhook[0] != nil {
		mux.Handle("/stripe/webhook", webhook[0])
	}
	return mux
}

func acceptsEventStream(accept string) bool {
	return strings.Contains(accept, "text/event-stream")
}

func writeEventStream(w http.ResponseWriter, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func bearerToken(header string) string {
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func apiKeyFromParams(raw json.RawMessage) string {
	var params struct {
		APIKey string `json:"apiKey"`
		Meta   struct {
			APIKey string `json:"apiKey"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return ""
	}
	if params.APIKey != "" {
		return params.APIKey
	}
	return params.Meta.APIKey
}

func IsClosedServer(err error) bool {
	return errors.Is(err, http.ErrServerClosed)
}
