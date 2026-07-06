# Contributing to BindKit

Thanks for improving BindKit. The project aims to stay small, auditable, and
useful as a production-shaped MCP server starter kit.

## Development

```bash
go test ./...
BINDKIT_TRANSPORT=http go run ./cmd/server
```

Before opening a pull request:

- run `go test ./...`;
- add or update tests for behavior changes;
- keep new dependencies justified and minimal;
- update docs when changing configuration, auth, billing, or transport behavior;
- avoid committing generated binaries, logs, secrets, or local packaging output.

## Contribution license

By submitting a contribution, you agree that it is licensed under the Apache
License, Version 2.0, unless explicitly stated otherwise in writing.

## Code style

- Prefer standard-library Go where practical.
- Keep MCP tool handlers small and typed.
- Fail loudly on configuration errors.
- Do not log secrets, tokens, API keys, webhook signatures, or raw credentials.
- Keep examples safe by default. Network-capable examples should document their
  boundaries and SSRF protections.

## Security changes

For suspected vulnerabilities, follow `SECURITY.md` instead of opening a public
issue first.
