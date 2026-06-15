# Bindkit Task Board

## Now

- [x] Create Go module skeleton: `cmd/server`, `internal/mcp`, `internal/server`, `internal/config`, `tools/example_weather`.
- [x] Define MCP request/response types and `Handler` interface.
- [x] Implement registry and dispatcher for `initialize`, `tools/list`, `tools/call`.
- [x] Add stdio transport.
- [x] Add middleware chain shape around dispatcher.

## Next

- [x] Add HTTP transport with graceful shutdown.
- [x] Add auth middleware: bearer key to principal/plan.
- [x] Add rate-limit middleware: token bucket per key.
- [x] Add metering package: memory counter first and store interface for later Redis adapter.
- [x] Add billing middleware: quota check and adapter boundary for later Stripe lookup.
- [x] Add config package with typed env and aggregated startup validation.
- [x] Add structured logging with redaction.

## Weekend Completion

- [x] Add `make test`.
- [x] Add `make new-tool name=x`.
- [x] Add `scripts/new_tool.sh`.
- [x] Add Dockerfile.
- [x] Add docker-compose for the memory-backed HTTP starter service.
- [x] Add Fly deploy config.
- [x] Add GitHub Actions CI.
- [x] Add docs: clients, pricing, deploy, launch.

## Tests

- [x] Discovery works without gates.
- [x] Tool call succeeds with gates disabled.
- [x] Missing key is denied when auth enabled.
- [x] Rate limit denies before dispatch.
- [x] Quota denies before dispatch.
- [x] Metering increments successful calls.
- [x] Config reports all missing/invalid env values.
- [x] Logs redact secrets.
- [x] Package tests cover registry, dispatcher, auth, billing, config, metering, rate limit, HTTP, stdio, and example tool behavior.
- [ ] Race detector run on a machine with CGO and `gcc` installed.

## Launch

- [ ] Replace weather stub with first real B2B tool.
- [ ] Add Redis metering adapter if multi-instance hosting is needed.
- [ ] Replace static API-key map with Stripe/customer subscription lookup.
- [ ] Write README with 5-minute local quickstart.
- [ ] Publish `bindfort/bindkit`.
- [ ] Post to MCP registries.
- [ ] Prepare Show HN / r/mcp / dev.to launch copy.
