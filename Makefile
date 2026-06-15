.PHONY: test run-http run-stdio new-tool

test:
	go test ./...

run-http:
	BINDKIT_TRANSPORT=http go run ./cmd/server

run-stdio:
	BINDKIT_TRANSPORT=stdio go run ./cmd/server

new-tool:
	@if [ -z "$(name)" ]; then echo "usage: make new-tool name=my_tool"; exit 1; fi
	./scripts/new_tool.sh "$(name)"

