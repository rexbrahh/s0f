package ipc

import "encoding/json"

// Request models RPC requests.
type Request struct {
	ID     string          `json:"id,omitempty"`
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response models RPC responses.
type Response struct {
	ID      string          `json:"id,omitempty"`
	OK      bool            `json:"ok"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	TraceID string          `json:"traceId,omitempty"`
}

// Error follows the API contract for structured failures.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
