package mcp

import (
	"context"
	"errors"
)

type Dispatcher struct {
	registry *Registry
	name     string
	version  string
}

func NewDispatcher(registry *Registry, name, version string) *Dispatcher {
	return &Dispatcher{registry: registry, name: name, version: version}
}

func (d *Dispatcher) Handle(ctx context.Context, req *Request) *Response {
	if req == nil {
		return Failure(nil, -32600, "invalid request")
	}
	id := req.ID
	switch req.Method {
	case "initialize":
		return Success(id, map[string]any{
			"protocolVersion": "2025-06-18",
			"serverInfo": map[string]any{
				"name":    d.name,
				"version": d.version,
			},
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
		})
	case "tools/list":
		return Success(id, map[string]any{"tools": d.registry.List()})
	case "tools/call":
		call, err := DecodeParams[CallRequest](req.Params)
		if err != nil {
			return Failure(id, -32602, err.Error())
		}
		result, err := d.registry.Call(ctx, call)
		if errors.Is(err, ErrToolNotFound) {
			return Failure(id, -32601, err.Error())
		}
		if err != nil {
			return Success(id, CallResult{
				Content: []Content{{Type: "text", Text: err.Error()}},
				IsError: true,
			})
		}
		return Success(id, result)
	default:
		return Failure(id, -32601, "method not found")
	}
}
