# BindKit open-source release checklist

Use this checklist before publishing a public repository or announcing a new
release.

## 1. Verify the code

```bash
go test ./...
docker build -t bindkit .
```

Check that:

- auth tests pass for static API keys and OAuth bearer tokens;
- Stripe webhook signature verification tests pass;
- `url.check` SSRF guard tests pass;
- HTTP and stdio transport tests pass;
- docs still match the current environment variables.

## 2. Verify the open-source package

- [x] Apache-2.0 `LICENSE` file exists.
- [x] `NOTICE` file exists.
- [x] `THIRD_PARTY_NOTICES.md` records direct dependency licenses.
- [x] `CONTRIBUTING.md` explains contribution licensing.
- [x] `SECURITY.md` explains coordinated disclosure.
- [x] `CODE_OF_CONDUCT.md` sets the contributor standard.
- [ ] No built binaries, logs, secrets, or local packaging output are tracked.
- [ ] `git status --short` is reviewed before publishing.

## 3. Tag a release

Recommended first public tags:

```bash
git tag -a v0.1.0 -m "BindKit v0.1.0"
git push origin main --tags
```

Use `v0.x` until the public API, config variables, and tool scaffolding are
stable enough for a `v1.0.0` compatibility promise.

## 4. Release notes

Suggested first release headline:

> BindKit is an Apache-2.0 Go starter kit for production-shaped MCP servers:
> stdio and HTTP transports, streamable HTTP, static or OAuth auth, metering,
> rate limits, quotas, Stripe usage hooks, Docker, and CI.

Call out:

- bundled `url.check` tool with SSRF protections;
- OAuth/JWKS support;
- Stripe meter-event and webhook adapters;
- per-key rate limiting and metering;
- typed tool scaffolding scripts;
- Apache-2.0 license.

## 5. Announcement copy

Short version:

> We open-sourced BindKit: a Go starter kit for building production-shaped MCP
> servers. It includes stdio/HTTP transports, auth, metering, quotas, rate
> limits, Stripe usage hooks, Docker, CI, and an SSRF-guarded example tool.
> Apache-2.0.

Long version:

> MCP makes tool wiring fast, but production teams still need auth, metering,
> quotas, rate limits, billing hooks, deployment shape, and tests. BindKit is a
> compact Go starter kit for that layer. Bring your tool logic; the server
> plumbing is already wired.

Good launch targets:

- GitHub release;
- Hacker News "Show HN";
- r/golang;
- r/mcp;
- MCP community Discord/Slack spaces where self-promotion is allowed;
- Dev.to or personal technical blog post;
- LinkedIn/X from the maintainer account.

## 6. Maintainer follow-up

After launch:

- triage installation friction first;
- label good first issues;
- keep the starter small;
- require tests for auth, billing, transport, or SSRF-sensitive changes;
- document any breaking config changes in release notes.
