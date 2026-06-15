package mcp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

var ErrToolNotFound = errors.New("tool not found")

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type CallRequest struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CallResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type ToolHandler func(context.Context, CallRequest) (CallResult, error)

type Registry struct {
	mu       sync.RWMutex
	tools    map[string]Tool
	handlers map[string]ToolHandler
}

func NewRegistry() *Registry {
	return &Registry{
		tools:    map[string]Tool{},
		handlers: map[string]ToolHandler{},
	}
}

func (r *Registry) Register(tool Tool, handler ToolHandler) error {
	if tool.Name == "" {
		return errors.New("tool name is required")
	}
	if handler == nil {
		return errors.New("tool handler is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %q already registered", tool.Name)
	}
	r.tools[tool.Name] = tool
	r.handlers[tool.Name] = handler
	return nil
}

func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, r.tools[name])
	}
	return tools
}

func (r *Registry) Call(ctx context.Context, call CallRequest) (CallResult, error) {
	r.mu.RLock()
	handler := r.handlers[call.Name]
	r.mu.RUnlock()
	if handler == nil {
		return CallResult{}, ErrToolNotFound
	}
	return handler(ctx, call)
}
