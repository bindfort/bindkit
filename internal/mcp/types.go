package mcp

import (
	"encoding/json"
	"fmt"
)

const JSONRPCVersion = "2.0"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Success(id json.RawMessage, result any) *Response {
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Result: result}
}

func Failure(id json.RawMessage, code int, message string) *Response {
	return &Response{JSONRPC: JSONRPCVersion, ID: id, Error: &Error{Code: code, Message: message}}
}

func DecodeParams[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("invalid params: %w", err)
	}
	return out, nil
}
