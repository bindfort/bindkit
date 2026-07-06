// Package urlcheck provides a self-contained BindKit tool: it probes an HTTP(S)
// endpoint and reports status, latency, and security-header posture. It
// demonstrates production tool patterns: typed input, validation, timeouts,
// SSRF guarding, structured output, and clear errors.
package urlcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bindfort/bindkit/internal/mcp"
)

// securityHeaders are the response headers a hardened endpoint should set.
var securityHeaders = []string{
	"Strict-Transport-Security",
	"Content-Security-Policy",
	"X-Content-Type-Options",
	"X-Frame-Options",
	"Referrer-Policy",
}

// Tool holds the HTTP client and SSRF policy for url.check.
type Tool struct {
	client       *http.Client
	allowPrivate bool
}

// New builds the tool. allowPrivate permits requests to loopback/private
// addresses; keep it false in production to prevent SSRF into internal networks.
func New(allowPrivate bool) *Tool {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		// Guard at dial time: this covers the original host, every redirect
		// target, and DNS rebinding, because the IP is resolved and pinned here
		// rather than re-resolved after a separate pre-flight check.
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("cannot resolve host %q", host)
			}
			var lastErr error
			for _, ipa := range ips {
				if !allowPrivate && isBlockedIP(ipa.IP) {
					lastErr = fmt.Errorf("refusing to connect to non-public address %s", ipa.IP)
					continue
				}
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ipa.IP.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("no address found for %q", host)
			}
			return nil, lastErr
		},
	}
	return &Tool{
		client: &http.Client{
			Timeout:   8 * time.Second,
			Transport: transport,
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("stopped after %d redirects", len(via))
				}
				return nil
			},
		},
		allowPrivate: allowPrivate,
	}
}

// Register wires url.check into the registry, reading its SSRF policy from env.
func Register(registry *mcp.Registry) error {
	allowPrivate := os.Getenv("BINDKIT_URLCHECK_ALLOW_PRIVATE") == "1"
	tool := New(allowPrivate)
	return registry.Register(mcp.Tool{
		Name:        "url.check",
		Description: "Fetch an HTTP(S) URL and report status code, latency, and which security headers are present or missing.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "Fully-qualified http(s) URL to check",
				},
			},
			"required": []string{"url"},
		},
	}, tool.Handle)
}

// Handle runs one check and returns a human-readable report.
func (t *Tool) Handle(ctx context.Context, call mcp.CallRequest) (mcp.CallResult, error) {
	raw, _ := call.Arguments["url"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return mcp.CallResult{}, fmt.Errorf("url is required")
	}

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return mcp.CallResult{}, fmt.Errorf("invalid url: provide a fully-qualified http(s) URL")
	}
	// SSRF protection is enforced at dial time (see New) so it also covers
	// redirects and DNS rebinding.

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return mcp.CallResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "bindkit-url-check/1.0")

	start := time.Now()
	resp, err := t.client.Do(req)
	if err != nil {
		return mcp.CallResult{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	elapsed := time.Since(start)

	return mcp.CallResult{Content: []mcp.Content{{Type: "text", Text: report(u.String(), resp, elapsed)}}}, nil
}

func report(target string, resp *http.Response, elapsed time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "GET %s -> %d %s (%dms)\n", target, resp.StatusCode, http.StatusText(resp.StatusCode), elapsed.Milliseconds())
	if server := resp.Header.Get("Server"); server != "" {
		fmt.Fprintf(&b, "server: %s\n", server)
	}
	b.WriteString("security headers:\n")
	present, missing := 0, []string{}
	for _, h := range securityHeaders {
		if resp.Header.Get(h) != "" {
			fmt.Fprintf(&b, "  [+] %s\n", h)
			present++
		} else {
			missing = append(missing, h)
		}
	}
	sort.Strings(missing)
	for _, h := range missing {
		fmt.Fprintf(&b, "  [-] %s (missing)\n", h)
	}
	fmt.Fprintf(&b, "score: %d/%d security headers present", present, len(securityHeaders))
	return b.String()
}

// isBlockedIP reports whether an address must not be contacted from a public
// tool: loopback, private (RFC1918 + IPv6 ULA), link-local (covers the cloud
// metadata IP 169.254.169.254), or unspecified.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
